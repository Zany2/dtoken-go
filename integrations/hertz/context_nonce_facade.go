// @Author daixk 2026/06/05
package hertz

import (
	"time"

	hertzapp "github.com/cloudwego/hertz/pkg/app"
)

// GenerateNonceWithTimeoutByContext generates nonce with timeout GenerateNonceWithTimeoutByContext ?Nonce
func GenerateNonceWithTimeoutByContext(ctx *hertzapp.RequestContext, timeout time.Duration) (string, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Nonce().GenerateWithTimeout(requestContext(ctx), timeout)
}

// IsValidNonceByContext checks nonce state IsValidNonceByContext ?Nonce ?
func IsValidNonceByContext(ctx *hertzapp.RequestContext, nonce string) bool {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return false
	}
	return dCtx.Nonce().IsValid(requestContext(ctx), nonce)
}

// GetNonceTTLByContext gets nonce TTL GetNonceTTLByContext ?Nonce ?
func GetNonceTTLByContext(ctx *hertzapp.RequestContext, nonce string) (int64, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Nonce().GetTTL(requestContext(ctx), nonce)
}
