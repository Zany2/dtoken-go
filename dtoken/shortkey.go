// @Author daixk 2026/06/01
package dtoken

import (
	"context"

	"github.com/Zany2/dtoken-go/core/shortkey"
)

// CreateShortKey creates a pending short key. CreateShortKey 创建待确认短 Key。
func CreateShortKey(ctx context.Context, authType ...string) (*shortkey.ShortKey, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.CreateShortKey(ctx, shortkey.CreateOptions{})
}

// CreateShortKeyWithOptions creates a short key with options. CreateShortKeyWithOptions 使用选项创建短 Key。
func CreateShortKeyWithOptions(ctx context.Context, opts shortkey.CreateOptions, authType ...string) (*shortkey.ShortKey, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.CreateShortKey(ctx, opts)
}

// ConfirmShortKey confirms a pending short key. ConfirmShortKey 确认待处理短 Key。
func ConfirmShortKey(ctx context.Context, key string, loginID string, authType ...string) (*shortkey.ShortKey, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.ConfirmShortKey(ctx, key, shortkey.ConfirmOptions{LoginID: loginID})
}

// ConfirmShortKeyWithOptions confirms a pending short key with options. ConfirmShortKeyWithOptions 使用选项确认短 Key。
func ConfirmShortKeyWithOptions(ctx context.Context, key string, opts shortkey.ConfirmOptions, authType ...string) (*shortkey.ShortKey, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.ConfirmShortKey(ctx, key, opts)
}

// ValidateShortKey validates a short key without consuming it. ValidateShortKey 校验短 Key 但不消费。
func ValidateShortKey(ctx context.Context, key string, authType ...string) (*shortkey.ShortKey, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.ValidateShortKey(ctx, key)
}

// ValidateShortKeyWithOptions validates a short key with constraints. ValidateShortKeyWithOptions 使用约束校验短 Key。
func ValidateShortKeyWithOptions(ctx context.Context, key string, opts shortkey.ValidateOptions, authType ...string) (*shortkey.ShortKey, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.ValidateShortKey(ctx, key, opts)
}

// ConsumeShortKey validates and consumes a short key. ConsumeShortKey 校验并消费短 Key。
func ConsumeShortKey(ctx context.Context, key string, authType ...string) (*shortkey.ConsumeResult, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.ConsumeShortKey(ctx, key)
}

// ConsumeShortKeyWithOptions validates and consumes a short key with constraints. ConsumeShortKeyWithOptions 使用约束校验并消费短 Key。
func ConsumeShortKeyWithOptions(ctx context.Context, key string, opts shortkey.ValidateOptions, authType ...string) (*shortkey.ConsumeResult, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.ConsumeShortKey(ctx, key, opts)
}

// RevokeShortKey revokes a short key. RevokeShortKey 撤销短 Key。
func RevokeShortKey(ctx context.Context, key string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.RevokeShortKey(ctx, key)
}

// GetShortKeyStatus returns short key lifecycle status. GetShortKeyStatus 返回短 Key 生命周期状态。
func GetShortKeyStatus(ctx context.Context, key string, authType ...string) (shortkey.Status, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return shortkey.StatusInvalid, err
	}
	return mgr.GetShortKeyStatus(ctx, key)
}

// GetShortKeyTTL returns short key ttl in seconds. GetShortKeyTTL 获取短 Key 剩余有效秒数。
func GetShortKeyTTL(ctx context.Context, key string, authType ...string) (int64, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return 0, err
	}
	return mgr.GetShortKeyTTL(ctx, key)
}
