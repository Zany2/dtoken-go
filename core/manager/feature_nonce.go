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
	m.triggerNonceGenerateEvent(value, m.nonceManager.TTL())
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
	m.triggerNonceGenerateEvent(value, m.resolveNonceEventTTL(timeout))
	return value, nil
}

// VerifyNonce verifies and consumes nonce. VerifyNonce 验证并消费一次 nonce。
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

// VerifyAndConsumeNonce verifies and consumes nonce with error detail. VerifyAndConsumeNonce 验证并消费 nonce，失败时返回错误。
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

// triggerNonceGenerateEvent triggers nonce creation event. triggerNonceGenerateEvent 触发 Nonce 创建事件。
func (m *Manager) triggerNonceGenerateEvent(value string, ttl time.Duration) {
	extra := map[string]any{
		listener.ExtraKeyAction: listener.ActionCreate,
	}
	if ttl > 0 {
		extra[listener.ExtraKeyTTL] = durationSecondsCeil(ttl)
	}
	m.triggerEvent(listener.EventNonceGenerate, "", "", "", value, extra)
}

// resolveNonceEventTTL resolves the effective nonce ttl for event data. resolveNonceEventTTL 解析事件数据中的实际 Nonce 有效期。
func (m *Manager) resolveNonceEventTTL(timeout time.Duration) time.Duration {
	if timeout > 0 {
		return timeout
	}
	if m.nonceManager == nil {
		return 0
	}
	return m.nonceManager.TTL()
}

// durationSecondsCeil converts duration to seconds and rounds up. durationSecondsCeil 将时长转换为秒并向上取整。
func durationSecondsCeil(duration time.Duration) int64 {
	if duration <= 0 {
		return 0
	}
	seconds := int64(duration / time.Second)
	if duration%time.Second != 0 {
		seconds++
	}
	if seconds <= 0 {
		return 1
	}
	return seconds
}
