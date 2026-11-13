// @Author daixk 2026/06/05
package context

import (
	"context"
	"time"

	derror "github.com/Zany2/dtoken-go/errors"
)

// Generate generates nonce Generate 生成 Nonce
func (c *NonceContext) Generate(ctx context.Context) (string, error) {
	service := c.d.nonceService()
	if service == nil {
		return "", derror.ErrModuleNotEnabled
	}
	return service.GenerateNonce(ctx)
}

// GenerateWithTimeout generates nonce with timeout GenerateWithTimeout 使用指定有效期生成 Nonce
func (c *NonceContext) GenerateWithTimeout(ctx context.Context, timeout time.Duration) (string, error) {
	service := c.d.nonceService()
	if service == nil {
		return "", derror.ErrModuleNotEnabled
	}
	return service.GenerateNonceWithTimeout(ctx, timeout)
}

// Verify verifies and consumes nonce Verify 校验并消费 Nonce
func (c *NonceContext) Verify(ctx context.Context, nonce string) bool {
	service := c.d.nonceService()
	if service == nil {
		return false
	}
	return service.VerifyNonce(ctx, nonce)
}

// VerifyAndConsume verifies nonce with error VerifyAndConsume 校验并消费 Nonce，失败时返回错误
func (c *NonceContext) VerifyAndConsume(ctx context.Context, nonce string) error {
	service := c.d.nonceService()
	if service == nil {
		return derror.ErrModuleNotEnabled
	}
	return service.VerifyAndConsumeNonce(ctx, nonce)
}

// IsValid checks nonce validity IsValid 检查 Nonce 是否有效
func (c *NonceContext) IsValid(ctx context.Context, nonce string) bool {
	service := c.d.nonceService()
	if service == nil {
		return false
	}
	return service.IsNonceValid(ctx, nonce)
}

// GetTTL gets nonce TTL GetTTL 获取 Nonce 剩余有效期
func (c *NonceContext) GetTTL(ctx context.Context, nonce string) (int64, error) {
	service := c.d.nonceService()
	if service == nil {
		return 0, derror.ErrModuleNotEnabled
	}
	return service.GetNonceTTL(ctx, nonce)
}
