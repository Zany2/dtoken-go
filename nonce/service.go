// @Author daixk 2025/12/22 15:56:00
package nonce

import (
	"context"
	"time"

	capability "github.com/Zany2/dtoken-go/contract/nonce"
	derror "github.com/Zany2/dtoken-go/errors"
	listener "github.com/Zany2/dtoken-go/event"
	"github.com/Zany2/dtoken-go/internal/utils"
)

// GenerateNonce generates nonce with default timeout. GenerateNonce 使用默认有效期生成 nonce。
func (m *Service) GenerateNonce(ctx context.Context) (string, error) {
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
func (m *Service) GenerateNonceWithTimeout(ctx context.Context, timeout time.Duration) (string, error) {
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

// VerifyNonce verifies and consumes  VerifyNonce 验证并消费一次 nonce。
func (m *Service) VerifyNonce(ctx context.Context, nonce string) bool {
	if m.nonceManager == nil {
		return false
	}
	ok := m.nonceManager.Verify(ctx, nonce)
	listener.Emit(m.events, listener.EventNonceVerify, "", "", "", nonce, map[string]any{
		listener.ExtraKeyAction: listener.ActionConsume,
		listener.ExtraKeyResult: ok,
	})
	return ok
}

// VerifyAndConsumeNonce verifies and consumes nonce with error detail. VerifyAndConsumeNonce 验证并消费 nonce，失败时返回错误。
func (m *Service) VerifyAndConsumeNonce(ctx context.Context, nonce string) error {
	if m.nonceManager == nil {
		return derror.ErrModuleNotEnabled
	}
	err := m.nonceManager.VerifyAndConsume(ctx, nonce)
	listener.Emit(m.events, listener.EventNonceVerify, "", "", "", nonce, map[string]any{
		listener.ExtraKeyAction: listener.ActionConsume,
		listener.ExtraKeyResult: err == nil,
	})
	return err
}

// IsNonceValid checks nonce validity without consuming it. IsNonceValid 检查 nonce 是否有效且不消费。
func (m *Service) IsNonceValid(ctx context.Context, nonce string) bool {
	if m.nonceManager == nil {
		return false
	}
	return m.nonceManager.IsValid(ctx, nonce)
}

// GetNonceTTL gets nonce ttl in seconds. GetNonceTTL 获取 nonce 剩余有效秒数。
func (m *Service) GetNonceTTL(ctx context.Context, nonce string) (int64, error) {
	if m.nonceManager == nil {
		return 0, derror.ErrModuleNotEnabled
	}
	return m.nonceManager.GetTTL(ctx, nonce)
}

// triggerNonceGenerateEvent triggers nonce creation event. triggerNonceGenerateEvent 触发 Nonce 创建事件。
func (m *Service) triggerNonceGenerateEvent(value string, ttl time.Duration) {
	extra := map[string]any{
		listener.ExtraKeyAction: listener.ActionCreate,
	}
	if ttl > 0 {
		extra[listener.ExtraKeyTTL] = utils.DurationSecondsCeil(ttl)
	}
	listener.Emit(m.events, listener.EventNonceGenerate, "", "", "", value, extra)
}

// resolveNonceEventTTL resolves the effective nonce ttl for event data. resolveNonceEventTTL 解析事件数据中的实际 Nonce 有效期。
func (m *Service) resolveNonceEventTTL(timeout time.Duration) time.Duration {
	if timeout > 0 {
		return timeout
	}
	if m.nonceManager == nil {
		return 0
	}
	return m.nonceManager.TTL()
}

// Service coordinates capability operations and lifecycle events. Service 协调能力操作与生命周期事件。
type Service struct {
	nonceManager *NonceManager
	events       listener.Emitter
}

// NewService creates a capability service with an optional event emitter and borrows components without transferring ownership; callers must keep them available throughout its lifetime. NewService 创建可选接入事件发布器的能力服务，仅借用组件，不转移所有权；调用方须保证组件在服务使用期间可用。
func NewService(manager *NonceManager, events listener.Emitter) *Service {
	return &Service{nonceManager: manager, events: events}
}

// The bundled service satisfies the public contract. 内置服务实现公开契约。
var _ capability.Service = (*Service)(nil)
