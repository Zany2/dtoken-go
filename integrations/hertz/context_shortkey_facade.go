// @Author daixk 2026/06/05
package hertz

import (
	"time"

	"github.com/Zany2/dtoken-go/core/shortkey"
	hertzapp "github.com/cloudwego/hertz/pkg/app"
)

// CreateShortKeyByContext creates short key CreateShortKeyByContext ?Key
func CreateShortKeyByContext(ctx *hertzapp.RequestContext, opts shortkey.CreateOptions) (*shortkey.ShortKey, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().Create(requestContext(ctx), opts)
}

// CreateShortKeyWithTimeoutByContext creates short key with timeout CreateShortKeyWithTimeoutByContext ?Key
func CreateShortKeyWithTimeoutByContext(ctx *hertzapp.RequestContext, opts shortkey.CreateOptions, timeout time.Duration) (*shortkey.ShortKey, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().CreateWithTimeout(requestContext(ctx), opts, timeout)
}

// ConfirmShortKeyByContext confirms short key ConfirmShortKeyByContext ?Key
func ConfirmShortKeyByContext(ctx *hertzapp.RequestContext, key string, opts shortkey.ConfirmOptions) (*shortkey.ShortKey, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().Confirm(requestContext(ctx), key, opts)
}

// ConfirmShortKeyForCurrentLoginByContext confirms short key for current user ConfirmShortKeyForCurrentLoginByContext ?Key
func ConfirmShortKeyForCurrentLoginByContext(ctx *hertzapp.RequestContext, key string, opts shortkey.ConfirmOptions) (*shortkey.ShortKey, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().ConfirmForCurrentLogin(requestContext(ctx), key, opts)
}

// ValidateShortKeyByContext validates short key ValidateShortKeyByContext ?Key
func ValidateShortKeyByContext(ctx *hertzapp.RequestContext, key string, opts ...shortkey.ValidateOptions) (*shortkey.ShortKey, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().Validate(requestContext(ctx), key, opts...)
}

// ConsumeShortKeyByContext consumes short key ConsumeShortKeyByContext ?Key
func ConsumeShortKeyByContext(ctx *hertzapp.RequestContext, key string, opts ...shortkey.ValidateOptions) (*shortkey.ConsumeResult, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().Consume(requestContext(ctx), key, opts...)
}

// RevokeShortKeyByContext revokes short key RevokeShortKeyByContext ?Key
func RevokeShortKeyByContext(ctx *hertzapp.RequestContext, key string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.ShortKey().Revoke(requestContext(ctx), key)
}

// GetShortKeyStatusByContext gets short key status GetShortKeyStatusByContext ?Key ?
func GetShortKeyStatusByContext(ctx *hertzapp.RequestContext, key string) (shortkey.Status, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.ShortKey().GetStatus(requestContext(ctx), key)
}

// GetShortKeyTTLByContext gets short key TTL GetShortKeyTTLByContext ?Key ?
func GetShortKeyTTLByContext(ctx *hertzapp.RequestContext, key string) (int64, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.ShortKey().GetTTL(requestContext(ctx), key)
}
