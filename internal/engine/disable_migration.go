package engine

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strings"

	adapter "github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
	"github.com/Zany2/dtoken-go/internal/utils"
)

// DisableMigrationValue describes a replacement key and payload while retaining the source deadline. DisableMigrationValue 描述替代键与载荷，并保留源记录截止时间。
type DisableMigrationValue struct {
	// Key is the canonical namespace-relative destination key. Key 是规范化的命名空间内目标键。
	Key string
	// Value contains the marker with explicit ownership and kind. Value 包含明确归属及类型的封禁标记。
	Value any
}

// UpgradeDisableSnapshot resolves legacy restrictions without storage access and rejects ambiguous ownership or duplicate destinations. UpgradeDisableSnapshot 不访问存储地解析旧封禁，拒绝归属歧义及重复目标键。
func UpgradeDisableSnapshot(ctx context.Context, codec adapter.Codec, values map[string]any) (map[string]DisableMigrationValue, error) {
	if utils.IsNil(codec) {
		return nil, derror.ErrInvalidParam
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	keys := make([]string, 0)
	for key := range values {
		if strings.HasPrefix(key, DisableKeyPrefix) {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)
	changes := make(map[string]DisableMigrationValue)
	destinations := make(map[string]int, len(keys))
	for i, key := range keys {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		var original *disableMarker
		if err := decodeMigrationValue(codec, values[key], &original); err != nil {
			return nil, fmt.Errorf("disable preflight record %d: %w", i+1, err)
		}
		if original == nil {
			return nil, fmt.Errorf("%w: null disable record %d", derror.ErrSerializeFailed, i+1)
		}
		marker, target, err := normalizeDisableMigration(key, *original)
		if err != nil {
			return nil, fmt.Errorf("disable preflight record %d: %w", i+1, err)
		}

		// Never merge old/current duplicates: their independent TTLs and precedence can represent different restrictions. 不合并新旧重复记录：独立 TTL 与读取优先级可能代表不同封禁。
		if previous, exists := destinations[target]; exists {
			return nil, fmt.Errorf("%w: disable records %d and %d resolve to the same destination", derror.ErrInvalidParam, previous, i+1)
		}
		destinations[target] = i + 1
		if target == key && marker == *original {
			continue
		}
		encoded, err := codec.Encode(&marker)
		if err != nil {
			return nil, fmt.Errorf("disable conversion record %d: %w", i+1, err)
		}
		changes[key] = DisableMigrationValue{Key: target, Value: bytes.Clone(encoded)}
	}
	return changes, nil
}

// normalizeDisableMigration derives only identities supported by both payload fields and an exact source key. normalizeDisableMigration 仅恢复载荷字段与完整源键共同支持的身份。
func normalizeDisableMigration(key string, original disableMarker) (disableMarker, string, error) {
	marker := original
	if marker.DisableTime < 0 || marker.Level < 0 ||
		marker.Service != strings.TrimSpace(marker.Service) ||
		marker.Device != strings.TrimSpace(marker.Device) || marker.DeviceID != strings.TrimSpace(marker.DeviceID) {
		return marker, "", derror.ErrInvalidParam
	}

	// Payload fields distinguish restriction kinds even when an account ID starts with service: or device:. 即使账号 ID 以 service: 或 device: 开头，也通过载荷字段区分封禁类型。
	kind, prefix, rawSuffix, escapedSuffix := "account", DisableKeyPrefix, "", ""
	switch {
	case marker.Service != "":
		if marker.Device != "" || marker.DeviceID != "" {
			return marker, "", derror.ErrInvalidParam
		}
		kind, prefix = "service", DisableServiceKeyPrefix
		rawSuffix, escapedSuffix = ":"+marker.Service, ":"+escapeStorageKeyComponent(marker.Service)
	case marker.Device != "":
		if marker.Level != 0 {
			return marker, "", derror.ErrInvalidParam
		}
		kind, prefix = "device", DisableDeviceKeyPrefix
		rawSuffix, escapedSuffix = ":"+marker.Device, ":"+escapeStorageKeyComponent(marker.Device)
		if marker.DeviceID != "" {
			prefix = DisableDeviceIDKeyPrefix
			rawSuffix += ":" + marker.DeviceID
			escapedSuffix += ":" + escapeStorageKeyComponent(marker.DeviceID)
		}
	default:
		if marker.DeviceID != "" || marker.Level != 0 {
			return marker, "", derror.ErrInvalidParam
		}
	}
	if marker.Kind != "" && marker.Kind != kind {
		return marker, "", fmt.Errorf("%w: disable kind disagrees with fields", derror.ErrInvalidParam)
	}
	if marker.LoginID == "" {
		// A backslash can be literal legacy data or escaping; never guess which owner it denotes. 反斜杠可能是旧字面数据或转义字符，不猜测其所属账号。
		if marker.Kind != "" || strings.Contains(key, "\\") {
			return marker, "", fmt.Errorf("%w: ambiguous disable ownership; explicit loginId and kind are required", derror.ErrInvalidParam)
		}
		if !strings.HasPrefix(key, prefix) || !strings.HasSuffix(key, rawSuffix) {
			return marker, "", derror.ErrInvalidParam
		}
		marker.LoginID = strings.TrimSuffix(strings.TrimPrefix(key, prefix), rawSuffix)
		if marker.LoginID == "" {
			return marker, "", derror.ErrInvalidParam
		}
	}
	marker.Kind = kind
	target := prefix + escapeStorageKeyComponent(marker.LoginID) + escapedSuffix
	legacy := prefix + marker.LoginID + rawSuffix
	if key != target && key != legacy {
		return marker, "", fmt.Errorf("%w: disable key disagrees with payload ownership", derror.ErrInvalidParam)
	}
	return marker, target, nil
}
