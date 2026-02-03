// @Author daixk 2026/2/3 16:14:00
package manager

import (
	"context"
)

// ============================================================================
// Nonce Management - Nonce 管理
// ============================================================================

// GenerateNonce generates a new nonce.
// GenerateNonce 生成新的 nonce。
func (m *Manager) GenerateNonce(ctx context.Context) (string, error) {
	return m.nonceManager.Generate(ctx)
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
