// @Author daixk 2026/06/05
package fiber

import (
	"time"

	gofiber "github.com/gofiber/fiber/v2"
)

// GenerateNonceWithTimeoutByContext generates nonce with timeout GenerateNonceWithTimeoutByContext 使用指定有效期生成 Nonce
func GenerateNonceWithTimeoutByContext(c *gofiber.Ctx, timeout time.Duration) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Nonce().GenerateWithTimeout(requestContext(c), timeout)
}

// IsValidNonceByContext checks nonce state IsValidNonceByContext 检查 Nonce 状态
func IsValidNonceByContext(c *gofiber.Ctx, nonce string) bool {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return false
	}
	return dCtx.Nonce().IsValid(requestContext(c), nonce)
}

// GetNonceTTLByContext gets nonce TTL GetNonceTTLByContext 获取 Nonce 剩余有效期
func GetNonceTTLByContext(c *gofiber.Ctx, nonce string) (int64, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return 0, err
	}
	return dCtx.Nonce().GetTTL(requestContext(c), nonce)
}
