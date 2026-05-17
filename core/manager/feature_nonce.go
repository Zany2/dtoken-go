// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"context"
	"time"
)

// GenerateNonce generates nonce with default timeout. GenerateNonce 使用默认有效期生成 nonce。
func (m *Manager) GenerateNonce(ctx context.Context) (string, error) {
	// Delegate nonce generation 委托生成 nonce。
	return m.nonceManager.Generate(ctx)
}

// GenerateNonceWithTimeout generates nonce with custom timeout. GenerateNonceWithTimeout 使用指定有效期生成 nonce。
func (m *Manager) GenerateNonceWithTimeout(ctx context.Context, timeout time.Duration) (string, error) {
	// Delegate nonce generation with timeout 委托按有效期生成 nonce。
	return m.nonceManager.GenerateWithTimeout(ctx, timeout)
}

// VerifyNonce verifies and consumes nonce. VerifyNonce 验证并消费一次性 nonce。
func (m *Manager) VerifyNonce(ctx context.Context, nonce string) bool {
	// Delegate nonce verification 委托验证 nonce。
	return m.nonceManager.Verify(ctx, nonce)
}

// VerifyAndConsumeNonce verifies and consumes nonce with error detail. VerifyAndConsumeNonce 验证并消费 nonce，失败时返回错误。
func (m *Manager) VerifyAndConsumeNonce(ctx context.Context, nonce string) error {
	// Delegate nonce verification with error 委托验证 nonce 并返回错误。
	return m.nonceManager.VerifyAndConsume(ctx, nonce)
}

// IsNonceValid checks nonce validity without consuming it. IsNonceValid 检查 nonce 是否有效且不消费。
func (m *Manager) IsNonceValid(ctx context.Context, nonce string) bool {
	// Delegate nonce validity check 委托检查 nonce 有效性。
	return m.nonceManager.IsValid(ctx, nonce)
}

// GetNonceTTL gets nonce ttl in seconds. GetNonceTTL 获取 nonce 剩余有效秒数。
func (m *Manager) GetNonceTTL(ctx context.Context, nonce string) (int64, error) {
	// Delegate nonce TTL query 委托查询 nonce TTL。
	return m.nonceManager.GetTTL(ctx, nonce)
}
