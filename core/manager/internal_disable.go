package manager

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/derror"
)

// retireDisabledTerminals invalidates session credentials while the account lock is held. retireDisabledTerminals 在持有账号锁时使 Session 凭证失效。
func (m *Manager) retireDisabledTerminals(ctx context.Context, sess *Session) error {
	tokens := make([]string, 0, len(sess.TerminalInfos))
	for _, terminal := range sess.TerminalInfos {
		if terminal.Token == "" {
			continue
		}

		record, err := m.getTokenRecord(ctx, terminal.Token)
		if err != nil {
			if !isTokenInactiveError(err) {
				return err
			}

			// A missing access token can still own a live refresh token; foreign records must be preserved. 已消失的访问令牌仍可能关联有效刷新令牌，但须保留其他认证体系记录。
			if errors.Is(err, derror.ErrInvalidToken) {
				data, readErr := m.storage.Get(ctx, m.getTokenKey(terminal.Token))
				if readErr != nil {
					return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, readErr)
				}
				if data != nil {
					continue
				}
			}
		} else {
			// Stale session entries must not retire a replacement lifecycle. 陈旧 Session 条目不能废弃替换后的生命周期。
			if !terminalMatchesTokenRecord(sess.LoginID, terminal, record) {
				continue
			}

			ttl, ttlErr := m.storage.TTL(ctx, m.getTokenKey(terminal.Token))
			if ttlErr != nil {
				return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, ttlErr)
			}
			if ttl == adapter.TTLNoExpire || ttl > 0 {
				record.Revoked = true
				if ttl == adapter.TTLNoExpire {
					ttl = 0
				}
				if err := m.saveToStorage(ctx, m.getTokenKey(terminal.Token), *record, ttl); err != nil {
					return err
				}
			}
		}
		tokens = append(tokens, terminal.Token)
	}

	// Synchronous cleanup avoids pool lock reentry and refresh revival after a quick untie. 同步清理避免协程池重入锁及快速解封后刷新令牌复活。
	return m.cleanTokenMetadata(ctx, tokens)
}

// disableMarker binds flat public disable fields to their stored owner and kind. disableMarker 为平铺的公开封禁字段绑定存储账号和类型。
type disableMarker struct {
	DisableInfo `msgpack:",inline"` // DisableInfo preserves common public fields. DisableInfo 保留公共封禁字段。
	LoginID     string              `json:"loginId,omitempty"`  // LoginID identifies the owning account. LoginID 标识所属账号。
	Kind        string              `json:"kind,omitempty"`     // Kind separates account, service, and device records. Kind 区分账号、服务和设备记录。
	Service     string              `json:"service,omitempty"`  // Service identifies a service restriction. Service 标识服务限制。
	Level       int                 `json:"level,omitempty"`    // Level stores the service restriction level. Level 存储服务封禁等级。
	Device      string              `json:"device,omitempty"`   // Device identifies the device type. Device 标识设备类型。
	DeviceID    string              `json:"deviceId,omitempty"` // DeviceID identifies a concrete device. DeviceID 标识具体设备。
}

// storedDisableRecord associates a validated marker with its actual storage key. storedDisableRecord 关联已核对的封禁记录和实际存储键。
type storedDisableRecord struct {
	key    string
	marker disableMarker
}

// matchesDisableIdentity rejects foreign owners and ambiguous ownerless legacy keys. matchesDisableIdentity 拒绝其他账号记录及无法确定归属的旧键。
func (m *Manager) matchesDisableIdentity(key string, actual, expected disableMarker) (bool, error) {
	if actual.Kind != "" && actual.Kind != expected.Kind ||
		actual.Service != expected.Service || actual.Device != expected.Device || actual.DeviceID != expected.DeviceID {
		return false, nil
	}
	if actual.LoginID != "" {
		return actual.LoginID == expected.LoginID, nil
	}

	// Escaped and raw legacy keys overlap; field checks alone cannot recover the owner. 转义键与原始旧键重叠，仅检查业务字段无法恢复账号归属。
	if actual.Kind != "" || strings.Contains(strings.TrimPrefix(key, m.storageNamespace()), "\\") {
		return false, fmt.Errorf("%w: disable marker has ambiguous legacy ownership; migrate it with an explicit loginId and kind", derror.ErrInvalidParam)
	}
	return true, nil
}

// saveDisableMarker refuses to overwrite records belonging to another identity. saveDisableMarker 拒绝覆盖其他身份的封禁记录。
func (m *Manager) saveDisableMarker(ctx context.Context, key, legacyKey string, marker disableMarker, expiration time.Duration) error {
	var existing disableMarker
	exists, err := m.loadDisableMarker(ctx, key, &existing)
	if err != nil {
		return err
	}
	if exists {
		matches, err := m.matchesDisableIdentity(key, existing, marker)
		if err != nil {
			return err
		}
		if !matches {
			return fmt.Errorf("%w: disable key is occupied by a different identity", derror.ErrInvalidParam)
		}
	}

	// Resolve legacy ownership before replacing a restriction, so expiry cannot reveal an older ban. 重设封禁前核对旧记录归属，避免新记录到期后旧封禁重新生效。
	var legacyRecords []storedDisableRecord
	if legacyKey != key {
		legacyRecords, err = m.loadDisableRecords(ctx, marker, legacyKey, legacyKey, true)
		if err != nil {
			return err
		}
	}
	if err := m.saveToStorage(ctx, key, marker, expiration); err != nil {
		return err
	}
	for _, record := range legacyRecords {
		if err := m.storage.Delete(ctx, record.key); err != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
	}
	return nil
}

// loadDisableRecords keeps current-key precedence and validates every legacy candidate. loadDisableRecords 保持当前键优先，并核对每个旧键候选记录。
func (m *Manager) loadDisableRecords(ctx context.Context, expected disableMarker, key, legacyKey string, all bool) ([]storedDisableRecord, error) {
	keys := uniqueStorageKeys(key, legacyKey)
	records := make([]storedDisableRecord, 0, len(keys))
	for _, candidate := range keys {
		var marker disableMarker
		exists, err := m.loadDisableMarker(ctx, candidate, &marker)
		if err != nil {
			return nil, err
		}
		if !exists {
			continue
		}
		matches, err := m.matchesDisableIdentity(candidate, marker, expected)
		if err != nil {
			return nil, err
		}
		if !matches {
			continue
		}
		records = append(records, storedDisableRecord{key: candidate, marker: marker})
		if !all {
			break
		}
	}
	return records, nil
}

// loadAccountDisableRecords validates current and legacy account markers. loadAccountDisableRecords 核对新旧账号封禁记录。
func (m *Manager) loadAccountDisableRecords(ctx context.Context, loginID string, all bool) ([]storedDisableRecord, error) {
	return m.loadDisableRecords(ctx, disableMarker{LoginID: loginID, Kind: "account"}, m.getDisableKey(loginID), m.getLegacyDisableKey(loginID), all)
}
