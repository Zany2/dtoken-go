// @Author daixk 2026/06/05
package gf

import (
	"context"
	"time"

	"github.com/Zany2/dtoken-go/core/shortkey"
)

// CreateShortKeyByCtx creates short key CreateShortKeyByCtx 鍒涘缓鐭?Key
func CreateShortKeyByCtx(ctx context.Context, opts shortkey.CreateOptions) (*shortkey.ShortKey, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().Create(ctx, opts)
}

// CreateShortKeyWithTimeoutByCtx creates short key with timeout CreateShortKeyWithTimeoutByCtx 浣跨敤鎸囧畾鏈夋晥鏈熷垱寤虹煭 Key
func CreateShortKeyWithTimeoutByCtx(ctx context.Context, opts shortkey.CreateOptions, timeout time.Duration) (*shortkey.ShortKey, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().CreateWithTimeout(ctx, opts, timeout)
}

// ConfirmShortKeyByCtx confirms short key ConfirmShortKeyByCtx 纭鐭?Key
func ConfirmShortKeyByCtx(ctx context.Context, key string, opts shortkey.ConfirmOptions) (*shortkey.ShortKey, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().Confirm(ctx, key, opts)
}

// ConfirmShortKeyForCurrentLoginByCtx confirms short key for current user ConfirmShortKeyForCurrentLoginByCtx 浣跨敤褰撳墠鐢ㄦ埛纭鐭?Key
func ConfirmShortKeyForCurrentLoginByCtx(ctx context.Context, key string, opts shortkey.ConfirmOptions) (*shortkey.ShortKey, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().ConfirmForCurrentLogin(ctx, key, opts)
}

// ValidateShortKeyByCtx validates short key ValidateShortKeyByCtx 鏍￠獙鐭?Key
func ValidateShortKeyByCtx(ctx context.Context, key string, opts ...shortkey.ValidateOptions) (*shortkey.ShortKey, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().Validate(ctx, key, opts...)
}

// ConsumeShortKeyByCtx consumes short key ConsumeShortKeyByCtx 娑堣垂鐭?Key
func ConsumeShortKeyByCtx(ctx context.Context, key string, opts ...shortkey.ValidateOptions) (*shortkey.ConsumeResult, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().Consume(ctx, key, opts...)
}

// RevokeShortKeyByCtx revokes short key RevokeShortKeyByCtx 鎾ら攢鐭?Key
func RevokeShortKeyByCtx(ctx context.Context, key string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.ShortKey().Revoke(ctx, key)
}

// GetShortKeyStatusByCtx gets short key status GetShortKeyStatusByCtx 鑾峰彇鐭?Key 鐘舵€?
func GetShortKeyStatusByCtx(ctx context.Context, key string) (shortkey.Status, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.ShortKey().GetStatus(ctx, key)
}

// GetShortKeyTTLByCtx gets short key TTL GetShortKeyTTLByCtx 鑾峰彇鐭?Key 鍓╀綑鏈夋晥鏈?
func GetShortKeyTTLByCtx(ctx context.Context, key string) (int64, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.ShortKey().GetTTL(ctx, key)
}
