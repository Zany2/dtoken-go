package sso

import (
	"context"
	"errors"
)

var (
	// ErrAuthorizationDenied indicates an explicit application policy denial. ErrAuthorizationDenied 表示应用策略明确拒绝授权。
	ErrAuthorizationDenied = errors.New("SSO authorization denied")
	// ErrAuthorizationPolicyUnavailable indicates a failed policy evaluation. ErrAuthorizationPolicyUnavailable 表示授权策略求值失败。
	ErrAuthorizationPolicyUnavailable = errors.New("SSO authorization policy unavailable")
	// ErrAuthorizationDecision indicates an invalid or scope-expanding policy result. ErrAuthorizationDecision 表示策略结果无效或扩大了权限范围。
	ErrAuthorizationDecision = errors.New("invalid SSO authorization decision")
)

// AuthorizationPhase identifies when the HTTP center evaluates application access. AuthorizationPhase 标识 HTTP 中心评估应用访问权限的阶段。
type AuthorizationPhase string

const (
	// AuthorizationIssue checks validated browser authorization before issuing any credential. AuthorizationIssue 在签发任何凭证前检查已校验的浏览器授权。
	AuthorizationIssue AuthorizationPhase = "issue"
	// AuthorizationExchange checks a validated one-time credential before consumption. AuthorizationExchange 在消费前检查已校验的一次性凭证。
	AuthorizationExchange AuthorizationPhase = "exchange"
	// AuthorizationInspect checks an active credential during introspection or userinfo. AuthorizationInspect 在 introspect 或 userinfo 中检查有效凭证。
	AuthorizationInspect AuthorizationPhase = "inspect"
)

// AuthorizationRequest contains validated identity and the maximum grant for this evaluation, without credential secrets. AuthorizationRequest 包含已校验身份及本次求值的最大授权范围，不包含凭证秘密。
type AuthorizationRequest struct {
	// Phase identifies issuance, exchange, or inspection. Phase 标识签发、兑换或检查阶段。
	Phase AuthorizationPhase
	// ClientID identifies the validated target application. ClientID 标识已校验的目标应用。
	ClientID string
	// LoginID comes from the center session or stored credential, never request parameters. LoginID 来自中心会话或存储凭证，不取自请求参数。
	LoginID string
	// Mode is the validated credential mode. Mode 是已校验的凭证模式。
	Mode Mode
	// Scopes contains a private copy of requested or originally granted scopes. Scopes 包含请求范围或原始授予范围的独立副本。
	Scopes []string
}

// AuthorizationDecision allows access and optionally reduces scopes; the zero value denies access. AuthorizationDecision 允许访问并可缩减范围，零值拒绝访问。
type AuthorizationDecision struct {
	// Allowed must be true to continue; false does not revoke the stored credential. Allowed 必须为 true 才继续，false 不撤销存储凭证。
	Allowed bool
	// Scopes nil preserves input scopes; a non-nil empty slice removes all scopes; additions are rejected. Scopes 为 nil 时保留输入范围，非 nil 空切片移除全部范围，新增范围会被拒绝。
	Scopes []string
}

// AuthorizationPolicyFunc evaluates current application access using trusted identity; honor context and return errors for lookup failures. AuthorizationPolicyFunc 按可信身份评估当前应用访问权限，应遵守上下文，查询故障须返回错误。
type AuthorizationPolicyFunc func(context.Context, AuthorizationRequest) (AuthorizationDecision, error)

// authorizationPolicyError preserves diagnostic causes without exposing them through HTTP error messages. authorizationPolicyError 保留诊断原因但不通过 HTTP 错误消息公开。
type authorizationPolicyError struct {
	kind  error
	cause error
}

// Error returns only the bounded policy classification. Error 仅返回固定的策略错误分类。
func (e *authorizationPolicyError) Error() string { return e.kind.Error() }

// Unwrap preserves both classification and the private diagnostic cause. Unwrap 保留错误分类及私有诊断原因。
func (e *authorizationPolicyError) Unwrap() []error { return []error{e.kind, e.cause} }

// evaluateAuthorization copies policy inputs and accepts only a subset of the original scopes. evaluateAuthorization 复制策略输入且仅接受原始范围的子集。
func (h *HTTPServer) evaluateAuthorization(ctx context.Context, request AuthorizationRequest) ([]string, error) {
	return h.exchangeCore(nil).evaluateAuthorization(ctx, request)
}
