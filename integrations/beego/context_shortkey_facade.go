// @Author daixk 2026/06/06
package beego

import (
	"time"

	"github.com/Zany2/dtoken-go/core/shortkey"
	beegocontext "github.com/beego/beego/v2/server/web/context"
)

// CreateShortKeyByContext creates short key CreateShortKeyByContext 创建短 Key
func CreateShortKeyByContext(c *beegocontext.Context, opts shortkey.CreateOptions) (*shortkey.ShortKey, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().Create(requestContext(c), opts)
}

// CreateShortKeyWithTimeoutByContext creates short key with timeout CreateShortKeyWithTimeoutByContext 使用指定有效期创建短 Key
func CreateShortKeyWithTimeoutByContext(c *beegocontext.Context, opts shortkey.CreateOptions, timeout time.Duration) (*shortkey.ShortKey, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().CreateWithTimeout(requestContext(c), opts, timeout)
}

// ConfirmShortKeyByContext confirms short key ConfirmShortKeyByContext 确认短 Key
func ConfirmShortKeyByContext(c *beegocontext.Context, key string, opts shortkey.ConfirmOptions) (*shortkey.ShortKey, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().Confirm(requestContext(c), key, opts)
}

// ConfirmShortKeyForCurrentLoginByContext confirms short key for current user ConfirmShortKeyForCurrentLoginByContext 为当前登录用户确认短 Key
func ConfirmShortKeyForCurrentLoginByContext(c *beegocontext.Context, key string, opts shortkey.ConfirmOptions) (*shortkey.ShortKey, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().ConfirmForCurrentLogin(requestContext(c), key, opts)
}

// ValidateShortKeyByContext validates short key ValidateShortKeyByContext 校验短 Key
func ValidateShortKeyByContext(c *beegocontext.Context, key string, opts ...shortkey.ValidateOptions) (*shortkey.ShortKey, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().Validate(requestContext(c), key, opts...)
}

// ConsumeShortKeyByContext consumes short key ConsumeShortKeyByContext 消费短 Key
func ConsumeShortKeyByContext(c *beegocontext.Context, key string, opts ...shortkey.ValidateOptions) (*shortkey.ConsumeResult, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().Consume(requestContext(c), key, opts...)
}

// RevokeShortKeyByContext revokes short key RevokeShortKeyByContext 撤销短 Key
func RevokeShortKeyByContext(c *beegocontext.Context, key string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.ShortKey().Revoke(requestContext(c), key)
}

// GetShortKeyStatusByContext gets short key status GetShortKeyStatusByContext 获取短 Key 状态
func GetShortKeyStatusByContext(c *beegocontext.Context, key string) (shortkey.Status, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.ShortKey().GetStatus(requestContext(c), key)
}

// GetShortKeyTTLByContext gets short key TTL GetShortKeyTTLByContext 获取短 Key 剩余有效期
func GetShortKeyTTLByContext(c *beegocontext.Context, key string) (int64, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return 0, err
	}
	return dCtx.ShortKey().GetTTL(requestContext(c), key)
}
