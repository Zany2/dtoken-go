// @Author daixk 2026/06/05
package fiber

import (
	"time"

	"github.com/Zany2/dtoken-go/core/shortkey"
	gofiber "github.com/gofiber/fiber/v2"
)

// CreateShortKeyByContext creates short key CreateShortKeyByContext 创建 ShortKey
func CreateShortKeyByContext(c *gofiber.Ctx, opts shortkey.CreateOptions) (*shortkey.ShortKey, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().Create(requestContext(c), opts)
}

// CreateShortKeyWithTimeoutByContext creates short key with timeout CreateShortKeyWithTimeoutByContext 使用指定有效期创建 ShortKey
func CreateShortKeyWithTimeoutByContext(c *gofiber.Ctx, opts shortkey.CreateOptions, timeout time.Duration) (*shortkey.ShortKey, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().CreateWithTimeout(requestContext(c), opts, timeout)
}

// ConfirmShortKeyByContext confirms short key ConfirmShortKeyByContext 确认短 Key
func ConfirmShortKeyByContext(c *gofiber.Ctx, key string, opts shortkey.ConfirmOptions) (*shortkey.ShortKey, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().Confirm(requestContext(c), key, opts)
}

// ConfirmShortKeyForCurrentLoginByContext confirms short key for current user ConfirmShortKeyForCurrentLoginByContext 使用当前用户确认 ShortKey
func ConfirmShortKeyForCurrentLoginByContext(c *gofiber.Ctx, key string, opts shortkey.ConfirmOptions) (*shortkey.ShortKey, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().ConfirmForCurrentLogin(requestContext(c), key, opts)
}

// ValidateShortKeyByContext validates short key ValidateShortKeyByContext 校验 ShortKey
func ValidateShortKeyByContext(c *gofiber.Ctx, key string, opts ...shortkey.ValidateOptions) (*shortkey.ShortKey, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().Validate(requestContext(c), key, opts...)
}

// ConsumeShortKeyByContext consumes short key ConsumeShortKeyByContext 消费短 Key
func ConsumeShortKeyByContext(c *gofiber.Ctx, key string, opts ...shortkey.ValidateOptions) (*shortkey.ConsumeResult, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().Consume(requestContext(c), key, opts...)
}

// RevokeShortKeyByContext revokes short key RevokeShortKeyByContext 撤销短 Key
func RevokeShortKeyByContext(c *gofiber.Ctx, key string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.ShortKey().Revoke(requestContext(c), key)
}

// GetShortKeyStatusByContext gets short key status GetShortKeyStatusByContext 获取 ShortKey 状态
func GetShortKeyStatusByContext(c *gofiber.Ctx, key string) (shortkey.Status, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.ShortKey().GetStatus(requestContext(c), key)
}

// GetShortKeyTTLByContext gets short key TTL GetShortKeyTTLByContext 获取 ShortKey 剩余有效期
func GetShortKeyTTLByContext(c *gofiber.Ctx, key string) (int64, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return 0, err
	}
	return dCtx.ShortKey().GetTTL(requestContext(c), key)
}
