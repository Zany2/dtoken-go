package engine

import (
	"context"
	"fmt"
	"time"

	adapter "github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
)

// retireDisabledTerminals invalidates session credentials while the account lock is held. retireDisabledTerminals 在持有账号锁时使 Session 凭证失效。
func (m *Manager) retireDisabledTerminals(ctx context.Context, sess *Session) error {
	tokens := make([]string, 0, len(sess.TerminalInfos))
	for _, terminal := range sess.TerminalInfos {
		if terminal.Token == "" {
			continue
		}

		record, allowed, err := m.loadTerminalCleanupRecord(ctx, sess.LoginID, terminal)
		if err != nil {
			return err
		}
		if !allowed {
			continue
		}

		// Preserve inspectable credential details while revoking only the checked lifecycle. 保留可查询的凭据详情，只废弃已核对的生命周期。
		if record != nil {
			ttl, ttlErr := m.storage.TTL(ctx, m.getTokenKey(terminal.Token))
			if ttlErr != nil {
				return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, ttlErr)
			}
			if ttl == adapter.TTLNoExpire || ttl > 0 {
				record.Revoked = true
				if ttl == adapter.TTLNoExpire {
					ttl = 0
				}
				if err := m.saveTokenRecord(ctx, terminal.Token, record, ttl); err != nil {
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
	// DisableInfo preserves common public fields. DisableInfo 保留公共封禁字段。
	DisableInfo `msgpack:",inline"`
	// LoginID identifies the owning account. LoginID 标识所属账号。
	LoginID string `json:"loginId,omitempty"`
	// Kind separates account, service, and device records. Kind 区分账号、服务和设备记录。
	Kind string `json:"kind,omitempty"`
	// Service identifies a service restriction. Service 标识服务限制。
	Service string `json:"service,omitempty"`
	// Level stores the service restriction level. Level 存储服务封禁等级。
	Level int `json:"level,omitempty"`
	// Device identifies the device type. Device 标识设备类型。
	Device string `json:"device,omitempty"`
	// DeviceID identifies a concrete device. DeviceID 标识具体设备。
	DeviceID string `json:"deviceId,omitempty"`
}

// storedDisableRecord associates a validated marker with its actual storage key. storedDisableRecord 关联已核对的封禁记录和实际存储键。
type storedDisableRecord struct {
	key    string
	marker disableMarker
}

// matchesDisableIdentity requires explicit ownership and rejects records from another scope. matchesDisableIdentity 要求明确归属，并拒绝其他作用域的记录。
func matchesDisableIdentity(actual, expected disableMarker) (bool, error) {
	if actual.LoginID == "" || actual.Kind == "" {
		return false, fmt.Errorf("%w: disable marker requires explicit loginId and kind; migrate legacy data offline", derror.ErrInvalidParam)
	}
	return actual.LoginID == expected.LoginID && actual.Kind == expected.Kind &&
		actual.Service == expected.Service && actual.Device == expected.Device && actual.DeviceID == expected.DeviceID, nil
}

// saveDisableMarker validates the canonical key before replacing its restriction. saveDisableMarker 替换封禁前校验规范键中记录的归属。
func (m *Manager) saveDisableMarker(ctx context.Context, key string, marker disableMarker, expiration time.Duration) error {
	var existing disableMarker
	exists, err := m.loadDisableMarker(ctx, key, &existing)
	if err != nil {
		return err
	}
	if exists {
		matches, err := matchesDisableIdentity(existing, marker)
		if err != nil {
			return err
		}
		if !matches {
			return fmt.Errorf("%w: disable key is occupied by a different identity", derror.ErrInvalidParam)
		}
	}
	return m.saveToStorage(ctx, key, marker, expiration)
}

// loadDisableRecord reads one canonical key without probing legacy candidates. loadDisableRecord 只读取一个规范键，不探测旧键候选。
func (m *Manager) loadDisableRecord(ctx context.Context, key string, expected disableMarker) (*storedDisableRecord, error) {
	var marker disableMarker
	exists, err := m.loadDisableMarker(ctx, key, &marker)
	if err != nil || !exists {
		return nil, err
	}
	matches, err := matchesDisableIdentity(marker, expected)
	if err != nil || !matches {
		return nil, err
	}
	return &storedDisableRecord{key: key, marker: marker}, nil
}

// deleteDisableRecord reuses validated ownership and avoids a second read on ordinary storage. deleteDisableRecord 复用已核对的归属，避免普通存储再次读取。
func (m *Manager) deleteDisableRecord(ctx context.Context, record *storedDisableRecord) (bool, error) {
	if record == nil {
		return false, nil
	}
	if storage, ok := m.storage.(adapter.AtomicStorage); ok {
		value, err := storage.GetAndDelete(ctx, record.key)
		if err != nil {
			return false, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
		return value != nil, nil
	}
	if err := m.storage.Delete(ctx, record.key); err != nil {
		return false, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	return true, nil
}

// loadAccountDisableRecord loads the canonical account restriction. loadAccountDisableRecord 加载规范键中的账号封禁。
func (m *Manager) loadAccountDisableRecord(ctx context.Context, loginID string) (*storedDisableRecord, error) {
	return m.loadDisableRecord(ctx, m.getDisableKey(loginID), disableMarker{LoginID: loginID, Kind: "account"})
}

// loadServiceDisableRecord loads the canonical service restriction. loadServiceDisableRecord 加载规范键中的服务封禁。
func (m *Manager) loadServiceDisableRecord(ctx context.Context, loginID, service string) (*storedDisableRecord, error) {
	return m.loadDisableRecord(ctx, m.getDisableServiceKey(loginID, service), disableMarker{LoginID: loginID, Kind: "service", Service: service})
}

// loadDeviceDisableRecord loads an exact device scope without applying parent rules. loadDeviceDisableRecord 加载精确设备作用域，不合并设备类型规则。
func (m *Manager) loadDeviceDisableRecord(ctx context.Context, loginID, device, deviceID string) (*storedDisableRecord, error) {
	var key string
	if deviceID == "" {
		key = m.getDisableDeviceKey(loginID, device)
	} else {
		key = m.getDisableDeviceAndDeviceIDKey(loginID, device, deviceID)
	}
	return m.loadDisableRecord(ctx, key, disableMarker{LoginID: loginID, Kind: "device", Device: device, DeviceID: deviceID})
}
