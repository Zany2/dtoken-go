package engine

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"slices"
	"strings"

	"github.com/Zany2/dtoken-go/config"
	adapter "github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
	"github.com/Zany2/dtoken-go/internal/utils"
)

// TokenMigrationValue describes a changed value and the snapshot key supplying its deadline. TokenMigrationValue 描述变更值及提供其截止时间的快照键。
type TokenMigrationValue struct {
	// Value contains the converted storage payload. Value 包含转换后的存储载荷。
	Value any
	// TTLSource identifies a namespace-relative source key. TTLSource 标识命名空间内的相对源键。
	TTLSource string
}

// UpgradeTokenSnapshot converts credential payloads without storage access; keys are namespace-relative and authType must be normalized. UpgradeTokenSnapshot 不访问存储地转换凭据载荷；键须为命名空间内相对键，authType 须已规范化。
func UpgradeTokenSnapshot(ctx context.Context, codec adapter.Codec, authType string, values map[string]any) (map[string]TokenMigrationValue, error) {
	if utils.IsNil(codec) || authType == "" {
		return nil, derror.ErrInvalidParam
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	tokens := make(map[string]*tokenRecord)
	refreshes := make(map[string]*refreshTokenRecord)
	usedIDs := make(map[string]bool)

	// Decode all identities before assigning IDs, retaining original binding semantics. 分配标识前先解码所有身份，保留原有绑定判定语义。
	for i, key := range keys {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		var err error
		switch {
		case strings.HasPrefix(key, config.TokenKeyPrefix):
			if key == config.TokenKeyPrefix {
				err = derror.ErrInvalidToken
				break
			}
			var raw []byte
			raw, err = utils.ToBytes(values[key])
			if err != nil {
				break
			}
			if tokenStateError(TokenState(raw)) != nil {
				continue
			}
			var record *tokenRecord
			if strings.HasPrefix(string(raw), "dtk:") {
				record, err = decodeTokenRecord(string(raw))
			} else {
				record = &tokenRecord{}
				err = codec.Decode(raw, record)
			}
			if err != nil {
				break
			}
			if record.AuthType != "" && record.AuthType != authType || record.HasExtra && record.Extra != nil {
				err = derror.ErrInvalidToken
				break
			}

			// Reuse the runtime parser's field validation for legacy payloads. 复用运行时解析器的字段校验检查旧载荷。
			if _, err = decodeTokenRecord(encodeTokenRecord(record)); err != nil {
				break
			}
			if record.AccessID != "" && usedIDs[record.AccessID] {
				err = fmt.Errorf("%w: duplicate access lifecycle", derror.ErrInvalidToken)
				break
			}
			tokens[key] = record
			if record.AccessID != "" {
				usedIDs[record.AccessID] = true
			}
		case strings.HasPrefix(key, TokenRefreshKeyPrefix):
			// Reverse mappings are checked after forward records have been decoded. 正向记录解码完成后再核对反向映射。
			continue
		case strings.HasPrefix(key, RefreshTokenKeyPrefix):
			info := &refreshTokenRecord{}
			err = decodeMigrationValue(codec, values[key], info)
			if err != nil {
				break
			}
			if key == RefreshTokenKeyPrefix || info.LoginID == "" || info.AccessToken == "" || info.AuthType != "" && info.AuthType != authType || info.CreateTime < 0 || info.AccessTTL < -1 || info.ActiveTTL < -1 || info.ExpiresIn < -1 {
				err = derror.ErrInvalidRefreshToken
				break
			}
			refreshes[key] = info
		case strings.HasPrefix(key, SessionKeyPrefix):
			var sess *Session
			err = decodeMigrationValue(codec, values[key], &sess)
			if err != nil {
				break
			}
			loginID := strings.TrimPrefix(key, SessionKeyPrefix)
			if sess == nil || loginID == "" || sess.AuthType != "" && sess.AuthType != authType || sess.LoginID != "" && sess.LoginID != loginID || sess.CreateTime < 0 || sess.HistoryTerminalCount < 0 {
				err = derror.ErrInvalidToken
			}
		}
		if err != nil {
			return nil, fmt.Errorf("credential preflight record %d: %w", i+1, err)
		}
	}
	for _, info := range refreshes {
		if info.AccessID != "" {
			usedIDs[info.AccessID] = true
		}
	}
	newID := func() string {
		for {
			id := rand.Text()
			if !usedIDs[id] && values["token-extra:"+id] == nil {
				usedIDs[id] = true
				return id
			}
		}
	}
	changes := make(map[string]TokenMigrationValue)
	upgraded := make(map[string]*tokenRecord)

	// Preserve existing lifecycle IDs and detach inline Extra with the access deadline. 保留已有生命周期标识，分离内联 Extra 并沿用访问凭据截止时间。
	for _, key := range keys {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		original := tokens[key]
		if original == nil {
			continue
		}
		record := *original
		record.AuthType = authType
		if record.AccessID == "" {
			record.AccessID = newID()
		}
		extraKey := "token-extra:" + record.AccessID
		if record.HasExtra {
			var extra map[string]any
			if err := decodeMigrationValue(codec, values[extraKey], &extra); err != nil {
				return nil, fmt.Errorf("credential extension: %w", err)
			}
			if extra == nil {
				return nil, fmt.Errorf("%w: missing credential extension", derror.ErrInvalidToken)
			}
		}
		if record.Extra != nil {
			if _, exists := values[extraKey]; exists {
				return nil, fmt.Errorf("%w: extension key is occupied", derror.ErrInvalidToken)
			}
			encoded, err := codec.Encode(record.Extra)
			if err != nil {
				return nil, err
			}
			changes[extraKey] = TokenMigrationValue{Value: bytes.Clone(encoded), TTLSource: key}
			record.Extra, record.HasExtra = nil, true
		}
		encoded := encodeTokenRecord(&record)
		raw, _ := utils.ToBytes(values[key])
		if string(raw) != encoded {
			changes[key] = TokenMigrationValue{Value: encoded, TTLSource: key}
		}
		upgraded[key] = &record
	}

	// Only matching original lifecycles share the new ID; expired or replaced access credentials stay detached. 仅原生命周期匹配时共用新标识；已过期或被替换的访问凭据保持分离。
	for i, key := range keys {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if strings.HasPrefix(key, TokenRefreshKeyPrefix) {
			value, err := utils.ToBytes(values[key])
			if err != nil || len(value) == 0 || key == TokenRefreshKeyPrefix {
				return nil, derror.ErrInvalidRefreshToken
			}
			if info := refreshes[RefreshTokenKeyPrefix+string(value)]; info != nil && info.AccessToken != strings.TrimPrefix(key, TokenRefreshKeyPrefix) {
				return nil, fmt.Errorf("%w: reverse binding disagrees with refresh record", derror.ErrInvalidRefreshToken)
			}
			continue
		}
		original := refreshes[key]
		if original == nil {
			continue
		}
		info := *original
		info.AuthType = authType
		if info.AccessID == "" {
			accessKey := config.TokenKeyPrefix + info.AccessToken
			if refreshMatchesAccessRecord(original, tokens[accessKey]) {
				info.AccessID = upgraded[accessKey].AccessID
			} else {
				info.AccessID = newID()
			}
		}
		if info.AccessID != original.AccessID || info.AuthType != original.AuthType {
			encoded, err := codec.Encode(&info)
			if err != nil {
				return nil, fmt.Errorf("refresh migration record %d: %w", i+1, err)
			}
			changes[key] = TokenMigrationValue{Value: bytes.Clone(encoded), TTLSource: key}
		}
	}
	return changes, nil
}

// decodeMigrationValue decodes a required snapshot value through the source Codec. decodeMigrationValue 通过源 Codec 解码必需的快照值。
func decodeMigrationValue(codec adapter.Codec, value any, target any) error {
	if value == nil {
		return derror.ErrKeyNotFound
	}
	raw, err := utils.ToBytes(value)
	if err != nil {
		return err
	}
	return codec.Decode(raw, target)
}
