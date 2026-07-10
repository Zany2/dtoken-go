// @Author daixk 2026/06/05
package chi

import (
	"context"
	"time"

	"github.com/Zany2/dtoken-go/core/shortkey"
)

// CreateShortKeyByCtx creates short key CreateShortKeyByCtx 创建 ShortKey
func CreateShortKeyByCtx(ctx context.Context, opts shortkey.CreateOptions) (*shortkey.ShortKey, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().Create(ctx, opts)
}

// CreateShortKeyWithTimeoutByCtx creates short key with timeout CreateShortKeyWithTimeoutByCtx 使用指定有效期创建 ShortKey
func CreateShortKeyWithTimeoutByCtx(ctx context.Context, opts shortkey.CreateOptions, timeout time.Duration) (*shortkey.ShortKey, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().CreateWithTimeout(ctx, opts, timeout)
}

// ConfirmShortKeyByCtx confirms short key ConfirmShortKeyByCtx 确认短 Key
func ConfirmShortKeyByCtx(ctx context.Context, key string, opts shortkey.ConfirmOptions) (*shortkey.ShortKey, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().Confirm(ctx, key, opts)
}

// ConfirmShortKeyForCurrentLoginByCtx confirms short key for current user ConfirmShortKeyForCurrentLoginByCtx 使用当前用户确认 ShortKey
func ConfirmShortKeyForCurrentLoginByCtx(ctx context.Context, key string, opts shortkey.ConfirmOptions) (*shortkey.ShortKey, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().ConfirmForCurrentLogin(ctx, key, opts)
}

// ValidateShortKeyByCtx validates short key ValidateShortKeyByCtx 校验 ShortKey
func ValidateShortKeyByCtx(ctx context.Context, key string, opts ...shortkey.ValidateOptions) (*shortkey.ShortKey, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().Validate(ctx, key, opts...)
}

// ConsumeShortKeyByCtx consumes short key ConsumeShortKeyByCtx 消费短 Key
func ConsumeShortKeyByCtx(ctx context.Context, key string, opts ...shortkey.ValidateOptions) (*shortkey.ConsumeResult, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().Consume(ctx, key, opts...)
}

// RevokeShortKeyByCtx revokes short key RevokeShortKeyByCtx 撤销短 Key
func RevokeShortKeyByCtx(ctx context.Context, key string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.ShortKey().Revoke(ctx, key)
}

// GetShortKeyStatusByCtx gets short key status GetShortKeyStatusByCtx 获取 ShortKey 状态
func GetShortKeyStatusByCtx(ctx context.Context, key string) (shortkey.Status, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.ShortKey().GetStatus(ctx, key)
}

// GetShortKeyTTLByCtx gets short key TTL GetShortKeyTTLByCtx 获取 ShortKey 剩余有效期
func GetShortKeyTTLByCtx(ctx context.Context, key string) (int64, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.ShortKey().GetTTL(ctx, key)
}
