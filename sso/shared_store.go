package sso

import (
	"context"
	"net/http"
	"net/url"
	"strings"
)

// exchangeRequest shares authentication, request limits, policy, and consumption between HTTP and shared-store clients. exchangeRequest 为 HTTP 和共享存储客户端复用认证、请求限制、策略及消费流程。
func (h *HTTPServer) exchangeRequest(r *http.Request) (*TicketExchangeResult, error) {
	if err := h.initializationError(); err != nil {
		return nil, err
	}
	if r == nil {
		return nil, ErrInvalidRequest
	}
	if r.Method != http.MethodPost {
		return nil, ErrMethodNotAllowed
	}
	if err := r.Context().Err(); err != nil {
		return nil, err
	}
	if err := parseRequestForm(r, h.options.RequestLimits); err != nil {
		return nil, err
	}
	// Keep exchange proofs out of URLs, including POST query strings. 兑换证明不得出现在 URL 中，包括 POST 查询串。
	if _, exists := r.URL.Query()[h.options.Params.CodeVerifier]; exists {
		return nil, ErrInvalidPKCE
	}
	if err := h.verifyRequest(r.Context(), r.Form, ActionToken, true); err != nil {
		return nil, err
	}
	return h.exchangeCredential(r, r.Form)
}

// exchangeSharedStore runs the existing exchange pipeline locally without sending an HTTP request. exchangeSharedStore 在本地执行既有兑换流程，不发送 HTTP 请求。
func (c *ClientApp) exchangeSharedStore(ctx context.Context, values url.Values) (*TicketExchangeResult, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	values, err := c.signRequest(values, ActionToken)
	if err != nil {
		return nil, err
	}
	target, err := joinURL(c.config.ServerURL, c.config.Endpoints.Token)
	if err != nil {
		return nil, err
	}

	// Keep proof in the body and retain the caller's context for policy and enrichment hooks. 证明仅放在请求体中，并为策略及补充钩子保留调用方上下文。
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(), strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	defer request.Body.Close()
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return c.config.SharedStoreServer.exchangeRequest(request)
}
