// @Author daixk 2026/06/05
package gin

import (
	"time"

	"github.com/gin-gonic/gin"
)

// GenerateNonceWithTimeoutByContext generates nonce with timeout GenerateNonceWithTimeoutByContext 浣跨敤鎸囧畾鏈夋晥鏈熺敓鎴?Nonce
func GenerateNonceWithTimeoutByContext(c *gin.Context, timeout time.Duration) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Nonce().GenerateWithTimeout(requestContext(c), timeout)
}

// IsValidNonceByContext checks nonce state IsValidNonceByContext 妫€鏌?Nonce 鐘舵€?
func IsValidNonceByContext(c *gin.Context, nonce string) bool {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return false
	}
	return dCtx.Nonce().IsValid(requestContext(c), nonce)
}

// GetNonceTTLByContext gets nonce TTL GetNonceTTLByContext 鑾峰彇 Nonce 鍓╀綑鏈夋晥鏈?
func GetNonceTTLByContext(c *gin.Context, nonce string) (int64, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return 0, err
	}
	return dCtx.Nonce().GetTTL(requestContext(c), nonce)
}
