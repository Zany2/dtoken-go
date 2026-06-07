// @Author daixk 2026/06/06
package beego

import (
	"time"

	beegocontext "github.com/beego/beego/v2/server/web/context"
)

// GenerateNonceByContext generates nonce with current manager GenerateNonceByContext 使用当前管理器生成 nonce
func GenerateNonceByContext(c *beegocontext.Context) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Nonce().Generate(requestContext(c))
}

// GenerateNonceWithTimeoutByContext generates nonce with timeout GenerateNonceWithTimeoutByContext 使用指定有效期生成 Nonce
func GenerateNonceWithTimeoutByContext(c *beegocontext.Context, timeout time.Duration) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Nonce().GenerateWithTimeout(requestContext(c), timeout)
}

// VerifyNonceByContext verifies nonce with current manager VerifyNonceByContext 使用当前管理器验证 nonce
func VerifyNonceByContext(c *beegocontext.Context, nonce string) bool {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return false
	}
	return dCtx.Nonce().Verify(requestContext(c), nonce)
}

// VerifyAndConsumeNonceByContext verifies and consumes nonce VerifyAndConsumeNonceByContext 验证并消费 nonce
func VerifyAndConsumeNonceByContext(c *beegocontext.Context, nonce string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Nonce().VerifyAndConsume(requestContext(c), nonce)
}

// IsValidNonceByContext checks nonce state IsValidNonceByContext 检查 Nonce 状态
func IsValidNonceByContext(c *beegocontext.Context, nonce string) bool {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return false
	}
	return dCtx.Nonce().IsValid(requestContext(c), nonce)
}

// GetNonceTTLByContext gets nonce TTL GetNonceTTLByContext 获取 Nonce 剩余有效期
func GetNonceTTLByContext(c *beegocontext.Context, nonce string) (int64, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return 0, err
	}
	return dCtx.Nonce().GetTTL(requestContext(c), nonce)
}
