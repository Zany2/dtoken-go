// @Author daixk 2026/2/3 16:14:00
package manager

import (
	"context"
	"time"
)

// ============================================================================
// Nonce Management - Nonce 管理
// ============================================================================

// GenerateNonce generates a new nonce.
// GenerateNonce 生成新的 nonce（使用默认有效期）。
func (m *Manager) GenerateNonce(ctx context.Context) (string, error) {
	return m.nonceManager.Generate(ctx)
}

// GenerateNonceWithTimeout generates a new nonce with a custom timeout duration.
// GenerateNonceWithTimeout 生成新的 nonce，使用指定的有效期。
func (m *Manager) GenerateNonceWithTimeout(ctx context.Context, timeout time.Duration) (string, error) {
	return m.nonceManager.GenerateWithTimeout(ctx, timeout)
}

// VerifyNonce verifies and consumes a nonce (one-time use).
// VerifyNonce 验证并消费 nonce（一次性使用）。
func (m *Manager) VerifyNonce(ctx context.Context, nonce string) bool {
	return m.nonceManager.Verify(ctx, nonce)
}

// VerifyAndConsumeNonce verifies and consumes a nonce, returns error if invalid.
// VerifyAndConsumeNonce 验证并消费 nonce，无效时返回错误。
func (m *Manager) VerifyAndConsumeNonce(ctx context.Context, nonce string) error {
	return m.nonceManager.VerifyAndConsume(ctx, nonce)
}

// IsNonceValid checks if a nonce is valid without consuming it.
// IsNonceValid 检查 nonce 是否有效（不消费）。
func (m *Manager) IsNonceValid(ctx context.Context, nonce string) bool {
	return m.nonceManager.IsValid(ctx, nonce)
}

// GetNonceTTL returns the remaining TTL of a nonce in seconds.
// GetNonceTTL 获取 nonce 的剩余有效时间（秒）。
func (m *Manager) GetNonceTTL(ctx context.Context, nonce string) (int64, error) {
	return m.nonceManager.GetTTL(ctx, nonce)
}
