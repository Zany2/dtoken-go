package engine

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	adapter "github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
	"github.com/Zany2/dtoken-go/internal/utils"
)

const tokenRecordPrefix = "dtk:2:" // tokenRecordPrefix identifies the compact credential format. tokenRecordPrefix 标识紧凑凭据格式。

// encodeTokenRecord encodes fixed credential fields without serializing extension data. encodeTokenRecord 编码固定凭据字段，不序列化扩展数据。
func encodeTokenRecord(record *tokenRecord) string {
	var out strings.Builder
	out.Grow(96 + len(record.AuthType) + len(record.LoginID) + len(record.Device) + len(record.DeviceID) + len(record.AccessID))
	out.WriteString(tokenRecordPrefix)
	for _, value := range [...]string{record.AuthType, record.LoginID, record.Device, record.DeviceID, record.AccessID} {
		out.WriteString(strconv.Itoa(len(value)))
		out.WriteByte(':')
		out.WriteString(value)
	}
	flags := int64(0)
	if record.Revoked {
		flags |= 1
	}
	if record.HasExtra {
		flags |= 2
	}
	for _, value := range [...]int64{record.CreateTime, record.Timeout, record.ActiveTimeout, record.TerminalIndex, flags} {
		out.WriteString(strconv.FormatInt(value, 10))
		out.WriteByte(':')
	}
	return out.String()
}

// decodeTokenRecord parses bounded fields and rejects truncated or unknown credential data. decodeTokenRecord 解析有边界的字段，拒绝截断或未知凭据数据。
func decodeTokenRecord(data string) (*tokenRecord, error) {
	if !strings.HasPrefix(data, tokenRecordPrefix) {
		return nil, derror.ErrInvalidToken
	}
	data = data[len(tokenRecordPrefix):]
	record := &tokenRecord{}
	for _, field := range [...]*string{&record.AuthType, &record.LoginID, &record.Device, &record.DeviceID, &record.AccessID} {
		length, rest, found := strings.Cut(data, ":")
		if !found {
			return nil, derror.ErrInvalidToken
		}
		size, err := strconv.ParseUint(length, 10, 63)
		if err != nil || size > uint64(len(rest)) {
			return nil, derror.ErrInvalidToken
		}
		*field, data = rest[:int(size)], rest[int(size):]
	}
	var flags int64
	for _, field := range [...]*int64{&record.CreateTime, &record.Timeout, &record.ActiveTimeout, &record.TerminalIndex, &flags} {
		value, rest, found := strings.Cut(data, ":")
		if !found {
			return nil, derror.ErrInvalidToken
		}
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, derror.ErrInvalidToken
		}
		*field, data = parsed, rest
	}
	if data != "" || flags < 0 || flags > 3 || record.LoginID == "" || record.CreateTime < 0 || record.TerminalIndex < 0 || record.Timeout < -1 || record.ActiveTimeout < -1 {
		return nil, derror.ErrInvalidToken
	}
	record.Revoked, record.HasExtra = flags&1 != 0, flags&2 != 0
	if record.HasExtra && record.AccessID == "" {
		return nil, derror.ErrInvalidToken
	}
	return record, nil
}

// loadTokenExtra loads extensions only for operations returning or copying token details. loadTokenExtra 仅为返回或复制 Token 详情的操作加载扩展数据。
func (m *Manager) loadTokenExtra(ctx context.Context, record *tokenRecord) error {
	if !record.HasExtra || record.Extra != nil {
		return nil
	}
	data, err := m.storage.Get(ctx, m.getTokenExtraKey(record.AccessID))
	if err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	if data == nil {
		return fmt.Errorf("%w: token extension data is missing", derror.ErrKeyNotFound)
	}
	encoded, err := utils.ToBytes(data)
	if err != nil {
		return fmt.Errorf("%w: %v", derror.ErrTypeConvert, err)
	}
	if err = m.serializer.Decode(encoded, &record.Extra); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}
	if record.Extra == nil {
		return fmt.Errorf("%w: token extension data is null", derror.ErrSerializeFailed)
	}
	return nil
}

// prepareTokenRecord saves supplied extensions before publishing a compact credential. prepareTokenRecord 发布紧凑凭据前保存调用方提供的扩展数据。
func (m *Manager) prepareTokenRecord(ctx context.Context, record *tokenRecord, expiration time.Duration) error {
	// Login or offline migration assigns identity before any runtime write. 登录或离线迁移在运行时写入前分配身份标识。
	if record == nil || record.AccessID == "" || record.AuthType != m.config.AuthType {
		return derror.ErrInvalidToken
	}

	if record.Extra == nil {
		return nil
	}
	if err := m.saveToStorage(ctx, m.getTokenExtraKey(record.AccessID), record.Extra, expiration); err != nil {
		return err
	}
	record.HasExtra = true
	return nil
}

// createTokenRecord publishes a fresh credential without overwriting an existing token. createTokenRecord 发布新凭据，不覆盖已存在的 Token。
func (m *Manager) createTokenRecord(ctx context.Context, token string, record *tokenRecord, expiration time.Duration) (bool, error) {
	if err := m.prepareTokenRecord(ctx, record, expiration); err != nil {
		return false, err
	}
	data := encodeTokenRecord(record)
	var saved bool
	var err error
	if storage, ok := m.storage.(adapter.AtomicStorage); ok {
		saved, err = storage.SetIfAbsent(ctx, m.getTokenKey(token), data, expiration)
	} else if !m.storage.Exists(ctx, m.getTokenKey(token)) {
		err = m.storage.Set(ctx, m.getTokenKey(token), data, expiration)
		saved = err == nil
	}
	if !saved || err != nil {
		if record.HasExtra {
			err = errors.Join(err, m.storage.Delete(ctx, m.getTokenExtraKey(record.AccessID)))
		}
		if err != nil {
			return false, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
		return false, nil
	}
	return true, nil
}

// saveTokenRecord updates fixed metadata while preserving separately stored extensions. saveTokenRecord 更新固定元数据并保留独立存储的扩展数据。
func (m *Manager) saveTokenRecord(ctx context.Context, token string, record *tokenRecord, expiration time.Duration) error {
	if err := m.prepareTokenRecord(ctx, record, expiration); err != nil {
		return err
	}
	if err := m.storage.Set(ctx, m.getTokenKey(token), encodeTokenRecord(record), expiration); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	if record.Revoked || record.Extra != nil {
		return nil
	}
	return m.renewTokenExtra(ctx, record, expiration)
}

// renewTokenExtra updates extension lifetime without decoding or re-encoding its value. renewTokenExtra 调整扩展数据有效期，不解码或重新编码其内容。
func (m *Manager) renewTokenExtra(ctx context.Context, record *tokenRecord, expiration time.Duration) error {
	if !record.HasExtra {
		return nil
	}
	key := m.getTokenExtraKey(record.AccessID)
	if expiration > 0 {
		if err := m.storage.Expire(ctx, key, expiration); err != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
		return nil
	}

	return m.persistStoredKey(ctx, key)
}

// deleteTokenRecords removes credentials and their lifecycle-bound extensions in one batch. deleteTokenRecords 批量删除凭据及绑定其生命周期的扩展数据。
func (m *Manager) deleteTokenRecords(ctx context.Context, tokens ...string) error {
	if len(tokens) == 0 {
		return nil
	}
	keys := make([]string, 0, len(tokens)*2)
	for _, token := range tokens {
		record, err := m.getTokenRecord(ctx, token)
		if err != nil && !isTokenInactiveError(err) && !errors.Is(err, derror.ErrTypeConvert) {
			return err
		}
		keys = append(keys, m.getTokenKey(token))
		if err == nil && record.HasExtra {
			keys = append(keys, m.getTokenExtraKey(record.AccessID))
		}
	}
	if err := m.storage.Delete(ctx, keys...); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	return nil
}
