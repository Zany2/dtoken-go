package sso

import (
	"context"
	"errors"
	"net/http"
)

// HTTPErrorFunc observes a center HTTP rejection synchronously; errors and requests may contain secrets, so never log them verbatim or mutate the request. HTTPErrorFunc 同步观察中心 HTTP 拒绝，错误及请求可能含秘密值，不得原样记录或修改请求。
type HTTPErrorFunc func(r *http.Request, status int, err error)

// protocolErrors keeps public classifications separate from wrapped private causes; infrastructure failures take precedence over their causes. protocolErrors 将公开分类与包装的私有原因分离，基础设施故障优先于底层原因。
var protocolErrors = []struct {
	kind   error
	status int
}{
	{ErrServerNotInitialized, http.StatusInternalServerError},
	{ErrSSOConfig, http.StatusInternalServerError},
	{ErrReplayProtectionConfig, http.StatusInternalServerError},
	{ErrAuthorizationPolicyUnavailable, http.StatusServiceUnavailable},
	{ErrStorageUnavailable, http.StatusInternalServerError},
	{ErrSerializeFailed, http.StatusInternalServerError},
	{context.Canceled, http.StatusInternalServerError},
	{context.DeadlineExceeded, http.StatusInternalServerError},
	{ErrRequestTooLarge, http.StatusRequestEntityTooLarge},
	{ErrAuthorizationDecision, http.StatusInternalServerError},
	{ErrAuthorizationDenied, http.StatusForbidden},
	{ErrPKCERequired, http.StatusBadRequest},
	{ErrInvalidPKCE, http.StatusBadRequest},
	{ErrClientOrClientIDEmpty, http.StatusBadRequest},
	{ErrInvalidRequest, http.StatusBadRequest},
	{ErrInvalidRedirectURI, http.StatusBadRequest},
	{ErrRedirectURIMismatch, http.StatusBadRequest},
	{ErrInvalidScope, http.StatusBadRequest},
	{ErrUserIDEmpty, http.StatusBadRequest},
	{ErrClientMismatch, http.StatusBadRequest},
	{ErrInvalidTicket, http.StatusBadRequest},
	{ErrTicketUsed, http.StatusBadRequest},
	{ErrTicketExpired, http.StatusBadRequest},
	{ErrInvalidSharedToken, http.StatusBadRequest},
	{ErrSharedTokenExpired, http.StatusBadRequest},
	{ErrInvalidRemoteSession, http.StatusBadRequest},
	{ErrRemoteSessionExpired, http.StatusBadRequest},
	{ErrInvalidOAuth2Code, http.StatusBadRequest},
	{ErrOAuth2CodeUsed, http.StatusBadRequest},
	{ErrOAuth2CodeExpired, http.StatusBadRequest},
	{ErrModeUnsupported, http.StatusBadRequest},
	{ErrStorageCapabilityUnsupported, http.StatusBadRequest},
	{ErrInvalidCallbackURL, http.StatusBadRequest},
	{ErrCallbackExpired, http.StatusBadRequest},
	{ErrClientSessionLimit, http.StatusBadRequest},
	{ErrInvalidClientCredentials, http.StatusUnauthorized},
	{ErrRequestExpired, http.StatusUnauthorized},
	{ErrRequestReplayed, http.StatusUnauthorized},
	{ErrInvalidSign, http.StatusUnauthorized},
	{ErrMethodNotAllowed, http.StatusMethodNotAllowed},
	{ErrClientNotFound, http.StatusNotFound},
}

// writeHTTPResponse observes encoding failures before committing a fixed public fallback. writeHTTPResponse 在提交固定公开回退响应前观察编码失败。
func (h *HTTPServer) writeHTTPResponse(w http.ResponseWriter, r *http.Request, status int, response Response) {
	status, payload, err := prepareJSON(status, response)
	if err != nil {
		h.observeHTTPError(r, status, err)
	}
	writeJSONBytes(w, status, payload)
}

// protocolError returns a known public message without calling Error on the supplied diagnostic. protocolError 返回已知公开消息，不调用传入诊断错误的 Error 方法。
func protocolError(err error) (int, string) {
	for _, entry := range protocolErrors {
		if errors.Is(err, entry.kind) {
			return entry.status, entry.kind.Error()
		}
	}
	return http.StatusInternalServerError, "SSO operation failed"
}

// statusFromError maps protocol errors using the same precedence as public messages. statusFromError 按与公开消息相同的优先级映射协议错误。
func statusFromError(err error) int {
	status, _ := protocolError(err)
	return status
}

// observeHTTPError retains private causes for application diagnostics without giving the observer response ownership. observeHTTPError 为应用诊断保留私有原因，不向观察方交出响应控制权。
func (h *HTTPServer) observeHTTPError(r *http.Request, status int, err error) {
	if h != nil && h.options.OnHTTPError != nil {
		h.options.OnHTTPError(r, status, err)
	}
}

// writeHTTPError preserves explicit endpoint status overrides while hiding internal diagnostics. writeHTTPError 保留端点的显式状态覆盖，同时隐藏内部诊断。
func (h *HTTPServer) writeHTTPError(w http.ResponseWriter, r *http.Request, status int, err error) {
	classified, message := protocolError(err)
	if classified != status {
		message = http.StatusText(status)
	}
	h.observeHTTPError(r, status, err)
	writeJSON(w, status, ErrorResponse(status, message))
}
