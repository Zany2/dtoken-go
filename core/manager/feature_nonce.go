// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"context"
	"time"

	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/listener"
)

// GenerateNonce generates nonce with default timeout. GenerateNonce 使用默认有效期生成 nonce。
func (m *Manager) GenerateNonce(ctx context.Context) (string, error) {
	if m.nonceManager == nil {
		return "", derror.ErrModuleNotEnabled
	}
	value, err := m.nonceManager.Generate(ctx)
	if err != nil {
		return "", err
	}
	m.triggerEvent(listener.EventNonceGenerate, "", "", "", value, map[string]any{
		listener.ExtraKeyAction: listener.ActionCreate,
	})
	return value, nil
}

// GenerateNonceWithTimeout generates nonce with custom timeout. GenerateNonceWithTimeout 使用指定有效期生成 nonce。
func (m *Manager) GenerateNonceWithTimeout(ctx context.Context, timeout time.Duration) (string, error) {
	if m.nonceManager == nil {
		return "", derror.ErrModuleNotEnabled
	}
	value, err := m.nonceManager.GenerateWithTimeout(ctx, timeout)
	if err != nil {
		return "", err
	}
	extra := map[string]any{
		listener.ExtraKeyAction: listener.ActionCreate,
	}
	if timeout > 0 {
		extra[listener.ExtraKeyTTL] = int64(timeout.Seconds())
	}
	m.triggerEvent(listener.EventNonceGenerate, "", "", "", value, extra)
	return value, nil
}

// VerifyNonce verifies and consumes nonce. VerifyNonce 校验并消费一次 nonce。
func (m *Manager) VerifyNonce(ctx context.Context, nonce string) bool {
	if m.nonceManager == nil {
		return false
	}
	ok := m.nonceManager.Verify(ctx, nonce)
	m.triggerEvent(listener.EventNonceVerify, "", "", "", nonce, map[string]any{
		listener.ExtraKeyAction: listener.ActionConsume,
		listener.ExtraKeyResult: ok,
	})
	return ok
}

// VerifyAndConsumeNonce verifies and consumes nonce with error detail. VerifyAndConsumeNonce 校验并消费 nonce，失败时返回错误。
func (m *Manager) VerifyAndConsumeNonce(ctx context.Context, nonce string) error {
	if m.nonceManager == nil {
		return derror.ErrModuleNotEnabled
	}
	err := m.nonceManager.VerifyAndConsume(ctx, nonce)
	m.triggerEvent(listener.EventNonceVerify, "", "", "", nonce, map[string]any{
		listener.ExtraKeyAction: listener.ActionConsume,
		listener.ExtraKeyResult: err == nil,
	})
	return err
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
