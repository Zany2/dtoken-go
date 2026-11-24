package sso

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"testing/iotest"
)

// TestRequestLimitsRepeatedFailure prevents an exhausted body from becoming an accepted query-only form. TestRequestLimitsRepeatedFailure 防止已耗尽的请求体变成可接受的纯查询串表单。
func TestRequestLimitsRepeatedFailure(t *testing.T) {
	for _, tc := range []struct {
		name string
		body func() io.Reader
		want error
	}{
		{"malformed", func() io.Reader { return strings.NewReader("private=%GG") }, ErrInvalidRequest},
		{"oversized", func() io.Reader { return strings.NewReader("extension=" + strings.Repeat("a", 64)) }, ErrRequestTooLarge},
		{"reader failure", func() io.Reader { return iotest.ErrReader(errors.New("private reader diagnostic")) }, ErrInvalidRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/?client=app", tc.body())
			r.ContentLength = -1
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			for attempt := 0; attempt < 3; attempt++ {
				if err := parseRequestForm(r, RequestLimits{MaxBodyBytes: 32}); !errors.Is(err, tc.want) {
					t.Fatalf("attempt %d: %v", attempt, err)
				}
				if r.Form != nil || r.PostForm != nil {
					t.Fatal("failed parse exposed partial fields")
				}
			}
			// Context-preserving middleware copies must keep the failure classification. 保留 Context 的中间件请求副本必须保持失败分类。
			copy := r.Clone(r.Context())
			if err := parseRequestForm(copy, DefaultRequestLimits()); !errors.Is(err, tc.want) {
				t.Fatalf("copied request lost failure: %v", err)
			}
		})
	}
	valid := httptest.NewRequest(http.MethodPost, "/?tag=query", strings.NewReader("tag=body"))
	valid.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for attempt := 0; attempt < 2; attempt++ {
		if err := parseRequestForm(valid, DefaultRequestLimits()); err != nil {
			t.Fatalf("successful parsing became a failure: %v", err)
		}
		if items := valid.Form["tag"]; len(items) != 2 || items[0] != "body" || items[1] != "query" {
			t.Fatal("successful repeat changed merged form values")
		}
	}
}

// TestRequestLimitsRepeatedCenterRejection preserves credentials and nonces across repeated handler calls. TestRequestLimitsRepeatedCenterRejection 在重复调用处理器时保留凭证及随机值。
func TestRequestLimitsRepeatedCenterRejection(t *testing.T) {
	s := NewServer()
	defer s.Close()
	client := newTestClient()
	client.SecretKey = "signing-key"
	if err := s.RegisterClient(client); err != nil {
		t.Fatal(err)
	}
	ticket, err := s.GenerateTicket(context.Background(), client.ClientID, "alice", client.RedirectURIs[0], nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	p := DefaultParamNames()
	protection := newReplayTestProtection(t, nil)
	h := NewHTTPServer(s, HTTPOptions{ServerOptions: ServerOptions{CheckSign: true}, ReplayProtection: protection})
	for _, endpoint := range []struct{ action, path string }{
		{ActionToken, "/sso/token"}, {ActionIntrospect, "/sso/introspect"}, {ActionUserInfo, "/sso/userinfo"}, {ActionRevoke, "/sso/revoke"}, {ActionLogout, "/sso/logout"},
	} {
		values := url.Values{p.Client: {client.ClientID}, p.ClientSecret: {client.ClientSecret}, p.Ticket: {ticket.Ticket}, p.Redirect: {client.RedirectURIs[0]}}
		values = protection.stamp(values, p, endpoint.action)
		values = NewSigner(client.SecretKey).AttachSign(values)
		r := httptest.NewRequest(http.MethodPost, endpoint.path+"?"+values.Encode(), strings.NewReader("private=%GG"))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		for attempt := 0; attempt < 2; attempt++ {
			w := httptest.NewRecorder()
			h.Handler().ServeHTTP(w, r)
			if w.Code != http.StatusBadRequest || strings.Contains(w.Body.String(), "%GG") {
				t.Fatalf("%s attempt %d: status %d", endpoint.action, attempt, w.Code)
			}
		}
		if _, err := s.ValidateTicket(context.Background(), ticket.Ticket); err != nil {
			t.Fatalf("credential changed: %v", err)
		}
		if err := h.verifyRequest(context.Background(), values, endpoint.action, true); err != nil {
			t.Fatalf("nonce claimed on failure: %v", err)
		}
	}
	// A fresh request still exchanges the original credential normally. 新请求仍可正常兑换原凭证。
	values := protection.stamp(url.Values{p.Client: {client.ClientID}, p.ClientSecret: {client.ClientSecret}, p.Ticket: {ticket.Ticket}, p.Redirect: {client.RedirectURIs[0]}}, p, ActionToken)
	values = NewSigner(client.SecretKey).AttachSign(values)
	if w := requestValidationPost(h.Handler(), "/sso/token", values, ""); w.Code != http.StatusOK {
		t.Fatalf("fresh exchange: %d", w.Code)
	}
}

// TestRequestLimitsRepeatedCallbackRejection never runs logout after a failed verification is retried. TestRequestLimitsRepeatedCallbackRejection 验签前解析失败后重试不得执行注销。
func TestRequestLimitsRepeatedCallbackRejection(t *testing.T) {
	protection := newReplayTestProtection(t, nil)
	p := DefaultParamNames()
	app := NewClientApp(ClientConfig{ClientID: "app", CheckSign: true, SecretKey: "key", ReplayProtection: protection})
	values := protection.stamp(url.Values{p.Client: {"app"}, p.LoginID: {"alice"}}, p, ActionLogoutCallback)
	values = NewSigner("key").AttachSign(values)
	r := httptest.NewRequest(http.MethodPost, "/callback?"+values.Encode(), strings.NewReader("private=%GG"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if _, err := app.VerifyLogoutCallback(r); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("first verification: %v", err)
	}
	calls := 0
	handler := app.LogoutCallbackHandler(func(*http.Request, LogoutCallback) error { calls++; return nil })
	w := httptest.NewRecorder()
	handler(w, r)
	if w.Code != http.StatusBadRequest || calls != 0 {
		t.Fatal("repeated verification accepted exhausted body")
	}
	if w := requestValidationPost(handler, "/callback", values, ""); w.Code != http.StatusOK || calls != 1 {
		t.Fatal("fresh callback lost its nonce")
	}
}
