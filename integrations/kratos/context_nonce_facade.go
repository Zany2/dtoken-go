// @Author daixk 2026/06/05
package kratos

import (
	"context"
	"time"
)

// GenerateNonceWithTimeoutByCtx generates nonce with timeout GenerateNonceWithTimeoutByCtx 使用指定有效期生成 Nonce
func GenerateNonceWithTimeoutByCtx(ctx context.Context, timeout time.Duration) (string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Nonce().GenerateWithTimeout(ctx, timeout)
}

// IsValidNonceByCtx checks nonce state IsValidNonceByCtx 检查 Nonce 状态
func IsValidNonceByCtx(ctx context.Context, nonce string) bool {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return false
	}
	return dCtx.Nonce().IsValid(ctx, nonce)
}

// GetNonceTTLByCtx gets nonce TTL GetNonceTTLByCtx 获取 Nonce 剩余有效期
func GetNonceTTLByCtx(ctx context.Context, nonce string) (int64, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Nonce().GetTTL(ctx, nonce)
}
