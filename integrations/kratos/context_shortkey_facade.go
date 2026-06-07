// @Author daixk 2026/06/05
package kratos

import (
	"context"
	"time"

	"github.com/Zany2/dtoken-go/core/shortkey"
)

// CreateShortKeyByCtx creates short key CreateShortKeyByCtx ?Key
func CreateShortKeyByCtx(ctx context.Context, opts shortkey.CreateOptions) (*shortkey.ShortKey, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().Create(ctx, opts)
}

// CreateShortKeyWithTimeoutByCtx creates short key with timeout CreateShortKeyWithTimeoutByCtx ?Key
func CreateShortKeyWithTimeoutByCtx(ctx context.Context, opts shortkey.CreateOptions, timeout time.Duration) (*shortkey.ShortKey, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().CreateWithTimeout(ctx, opts, timeout)
}

// ConfirmShortKeyByCtx confirms short key ConfirmShortKeyByCtx ?Key
func ConfirmShortKeyByCtx(ctx context.Context, key string, opts shortkey.ConfirmOptions) (*shortkey.ShortKey, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().Confirm(ctx, key, opts)
}

// ConfirmShortKeyForCurrentLoginByCtx confirms short key for current user ConfirmShortKeyForCurrentLoginByCtx ?Key
func ConfirmShortKeyForCurrentLoginByCtx(ctx context.Context, key string, opts shortkey.ConfirmOptions) (*shortkey.ShortKey, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().ConfirmForCurrentLogin(ctx, key, opts)
}

// ValidateShortKeyByCtx validates short key ValidateShortKeyByCtx ?Key
func ValidateShortKeyByCtx(ctx context.Context, key string, opts ...shortkey.ValidateOptions) (*shortkey.ShortKey, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().Validate(ctx, key, opts...)
}

// ConsumeShortKeyByCtx consumes short key ConsumeShortKeyByCtx ?Key
func ConsumeShortKeyByCtx(ctx context.Context, key string, opts ...shortkey.ValidateOptions) (*shortkey.ConsumeResult, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().Consume(ctx, key, opts...)
}

// RevokeShortKeyByCtx revokes short key RevokeShortKeyByCtx ?Key
func RevokeShortKeyByCtx(ctx context.Context, key string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.ShortKey().Revoke(ctx, key)
}

// GetShortKeyStatusByCtx gets short key status GetShortKeyStatusByCtx ?Key ?
func GetShortKeyStatusByCtx(ctx context.Context, key string) (shortkey.Status, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.ShortKey().GetStatus(ctx, key)
}

// GetShortKeyTTLByCtx gets short key TTL GetShortKeyTTLByCtx ?Key ?
func GetShortKeyTTLByCtx(ctx context.Context, key string) (int64, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.ShortKey().GetTTL(ctx, key)
}
