package sso

import "net/url"

// validateScalarParameters rejects ambiguous protocol fields without restricting application extension values. validateScalarParameters 拒绝歧义协议字段，不限制应用扩展参数的多值。
func validateScalarParameters(values url.Values, names ParamNames) error {
	for _, name := range protocolParamNames(names) {
		if items, exists := values[name]; exists && len(items) != 1 {
			return ErrInvalidRequest
		}
	}
	return nil
}

// validateCredentialParameters rejects foreign credential fields while preserving endpoint-specific missing-value behavior. validateCredentialParameters 拒绝其他模式凭证字段，保留各端点对缺失值的原有处理。
func validateCredentialParameters(values url.Values, names ParamNames, action string) error {
	if action != ActionToken && action != ActionIntrospect && action != ActionUserInfo && action != ActionRevoke {
		return nil
	}
	mode := Mode(values.Get(names.Mode))
	if mode == "" {
		mode = ModeTicket
		// Only exchange historically infers code mode; other endpoints default to Ticket. 仅兑换沿用授权码模式推断，其他端点默认 Ticket。
		if action == ActionToken && values.Get(names.Code) != "" {
			mode = ModeOAuth2
		}
	}
	var expected string
	switch mode {
	case ModeTicket:
		expected = names.Ticket
	case ModeOAuth2:
		expected = names.Code
	case ModeSharedToken:
		expected = names.TokenValue
	case ModeRemoteSession:
		expected = names.SessionID
	default:
		return ErrModeUnsupported
	}
	if action == ActionToken && mode != ModeTicket && mode != ModeOAuth2 {
		return ErrModeUnsupported
	}
	for _, key := range []string{names.Ticket, names.Code, names.TokenValue, names.SessionID} {
		if _, exists := values[key]; exists && key != expected {
			return ErrInvalidRequest
		}
	}
	return nil
}
