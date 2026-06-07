// @Author daixk 2026/06/05
package gf

import (
	"context"
	"time"
)

// GenerateNonceWithTimeoutByCtx generates nonce with timeout GenerateNonceWithTimeoutByCtx 浣跨敤鎸囧畾鏈夋晥鏈熺敓鎴?Nonce
func GenerateNonceWithTimeoutByCtx(ctx context.Context, timeout time.Duration) (string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Nonce().GenerateWithTimeout(ctx, timeout)
}

// IsValidNonceByCtx checks nonce state IsValidNonceByCtx 妫€鏌?Nonce 鐘舵€?
func IsValidNonceByCtx(ctx context.Context, nonce string) bool {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return false
	}
	return dCtx.Nonce().IsValid(ctx, nonce)
}

// GetNonceTTLByCtx gets nonce TTL GetNonceTTLByCtx 鑾峰彇 Nonce 鍓╀綑鏈夋晥鏈?
func GetNonceTTLByCtx(ctx context.Context, nonce string) (int64, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Nonce().GetTTL(ctx, nonce)
}
