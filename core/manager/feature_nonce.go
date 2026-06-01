// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"context"
	"time"

	"github.com/Zany2/dtoken-go/core/derror"
)

// GenerateNonce generates nonce with default timeout. GenerateNonce 使用默认有效期生成 nonce。
func (m *Manager) GenerateNonce(ctx context.Context) (string, error) {
	if m.nonceManager == nil {
		return "", derror.ErrModuleNotEnabled
	}
	return m.nonceManager.Generate(ctx)
}

// GenerateNonceWithTimeout generates nonce with custom timeout. GenerateNonceWithTimeout 使用指定有效期生成 nonce。
func (m *Manager) GenerateNonceWithTimeout(ctx context.Context, timeout time.Duration) (string, error) {
	if m.nonceManager == nil {
		return "", derror.ErrModuleNotEnabled
	}
	return m.nonceManager.GenerateWithTimeout(ctx, timeout)
}

// VerifyNonce verifies and consumes nonce. VerifyNonce 校验并消费一次性 nonce。
func (m *Manager) VerifyNonce(ctx context.Context, nonce string) bool {
	if m.nonceManager == nil {
		return false
	}
	return m.nonceManager.Verify(ctx, nonce)
}

// VerifyAndConsumeNonce verifies and consumes nonce with error detail. VerifyAndConsumeNonce 校验并消费 nonce，失败时返回错误。
func (m *Manager) VerifyAndConsumeNonce(ctx context.Context, nonce string) error {
	if m.nonceManager == nil {
		return derror.ErrModuleNotEnabled
	}
	return m.nonceManager.VerifyAndConsume(ctx, nonce)
}

// IsNonceValid checks nonce validity without consuming it. IsNonceValid 检查 nonce 是否有效且不消费。
func (m *Manager) IsNonceValid(ctx context.Context, nonce string) bool {
	if m.nonceManager == nil {
		return false
	}
	return m.nonceManager.IsValid(ctx, nonce)
}

// GetNonceTTL gets nonce ttl in seconds. GetNonceTTL 获取 nonce 剩余有效秒数。
func (m *Manager) GetNonceTTL(ctx context.Context, nonce string) (int64, error) {
	if m.nonceManager == nil {
		return 0, derror.ErrModuleNotEnabled
	}
	return m.nonceManager.GetTTL(ctx, nonce)
}
