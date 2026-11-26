package sso

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// httpErrorDeleteStorage injects revocation failures after successful credential lookup. httpErrorDeleteStorage 在凭证查询成功后注入撤销故障。
type httpErrorDeleteStorage struct {
	*MemoryStorage
	failure error
	readErr error
}

// Get injects credential-read faults without changing client-registration lookup. Get 注入凭证读取故障，不影响客户端登记查询。
func (s *httpErrorDeleteStorage) Get(ctx context.Context, key string) (any, error) {
	if s.readErr != nil {
		for _, prefix := range []string{TicketKeySuffix, OAuth2CodeKeySuffix, SharedTokenKeySuffix, RemoteSessionKeySuffix} {
			if strings.Contains(key, prefix) {
				return nil, s.readErr
			}
		}
	}
	return s.MemoryStorage.Get(ctx, key)
}

// Delete retains the credential on an injected failure. Delete 在注入故障时保留凭证。
func (s *httpErrorDeleteStorage) Delete(ctx context.Context, keys ...string) error {
	if s.failure != nil {
		return s.failure
	}
	return s.MemoryStorage.Delete(ctx, keys...)
}

// TestHTTPErrorCredentialLifecycle covers issuance, inspection and revocation failures in every credential mode. TestHTTPErrorCredentialLifecycle 覆盖所有凭证模式的签发、检查与撤销故障。
func TestHTTPErrorCredentialLifecycle(t *testing.T) {
	for _, mode := range []Mode{ModeTicket, ModeOAuth2, ModeSharedToken, ModeRemoteSession} {
		t.Run(string(mode), func(t *testing.T) {
			store := &httpErrorDeleteStorage{MemoryStorage: NewMemoryStorage()}
			s := NewServer(WithStorage(store))
			defer s.Close()
			client := newTestClient()
			client.Modes = []Mode{mode}
			if err := s.RegisterClient(client); err != nil {
				t.Fatal(err)
			}
			failure := errors.New("private-business-backend")
			var hookErr error = failure
			observations := 0
			var last error
			h := NewHTTPServer(s, HTTPOptions{
				LoginIDResolver:       func(*http.Request) (string, bool) { return "alice", true },
				AuthorizationExtra:    func(*http.Request, string, string) (map[string]any, error) { return nil, hookErr },
				CheckCredentialActive: func(*http.Request, CredentialInfo) (bool, error) { return true, hookErr },
				OnHTTPError: func(_ *http.Request, status int, err error) {
					observations++
					last = err
					if status != 500 {
						t.Fatalf("unexpected observed status %d", status)
					}
				},
			})
			issue := url.Values{"client": {client.ClientID}, "redirect": {client.RedirectURIs[0]}, "mode": {string(mode)}}
			assertHTTPFailure(t, replayTestRequest(h.Handler(), http.MethodGet, "/sso/authorize", issue), 500)
			if observations != 1 || !errors.Is(last, failure) {
				t.Fatal("issuance diagnostic lost")
			}
			hookErr = nil
			issued := replayTestRequest(h.Handler(), http.MethodGet, "/sso/authorize", issue)
			location, err := url.Parse(issued.Header().Get("Location"))
			if err != nil || issued.Code != 302 || observations != 1 {
				t.Fatal("issuance recovery failed")
			}
			values := location.Query()
			values.Set("client", client.ClientID)
			values.Set("clientSecret", client.ClientSecret)
			values.Set("mode", string(mode))
			for _, path := range []string{"/sso/introspect", "/sso/userinfo"} {
				hookErr = failure
				before := observations
				w := replayTestRequest(h.Handler(), http.MethodPost, path, values)
				assertHTTPFailure(t, w, 500)
				if observations != before+1 || !errors.Is(last, failure) || strings.Contains(w.Body.String(), `"active"`) || strings.Contains(w.Body.String(), "alice") {
					t.Fatal("inspection failure exposed identity or lost diagnostics")
				}
				hookErr = nil
				if w := replayTestRequest(h.Handler(), http.MethodPost, path, values); w.Code != 200 || observations != before+1 {
					t.Fatal("inspection failure consumed the credential")
				}
				store.readErr = failure
				assertHTTPFailure(t, replayTestRequest(h.Handler(), http.MethodPost, path, values), 500)
				if observations != before+2 || !errors.Is(last, ErrStorageUnavailable) {
					t.Fatal("credential storage outage was treated as inactivity")
				}
				store.readErr = nil
			}
			store.failure = failure
			before := observations
			assertHTTPFailure(t, replayTestRequest(h.Handler(), http.MethodPost, "/sso/revoke", values), 500)
			if observations != before+1 || !errors.Is(last, ErrStorageUnavailable) || !strings.Contains(last.Error(), failure.Error()) {
				t.Fatal("revocation storage diagnostic lost")
			}
			store.failure = nil
			if w := replayTestRequest(h.Handler(), http.MethodPost, "/sso/userinfo", values); w.Code != 200 {
				t.Fatal("failed deletion removed the credential")
			}
			if w := replayTestRequest(h.Handler(), http.MethodPost, "/sso/revoke", values); w.Code != 200 || observations != before+1 {
				t.Fatal("revocation recovery failed")
			}
		})
	}
}

// TestHTTPErrorExchangeStages preserves credentials before policy approval and keeps post-consumption failures consumed. TestHTTPErrorExchangeStages 在策略批准前保留凭证，消费后的失败保持已消费状态。
func TestHTTPErrorExchangeStages(t *testing.T) {
	for _, mode := range []Mode{ModeTicket, ModeOAuth2} {
		t.Run(string(mode), func(t *testing.T) {
			s := NewServer()
			defer s.Close()
			client := newTestClient()
			client.Modes = []Mode{mode}
			if err := s.RegisterClient(client); err != nil {
				t.Fatal(err)
			}
			failure := errors.New("private-exchange-details")
			stage := "issue"
			observed, enriched := 0, 0
			var last error
			var lastStatus int
			h := NewHTTPServer(s, HTTPOptions{
				LoginIDResolver: func(*http.Request) (string, bool) { return "alice", true },
				AuthorizationPolicy: func(context.Context, AuthorizationRequest) (AuthorizationDecision, error) {
					if stage == "policy" {
						return AuthorizationDecision{}, failure
					}
					return AuthorizationDecision{Allowed: true}, nil
				},
				EnrichExchangeResult: func(*http.Request, string, *TicketExchangeResult) error { enriched++; return failure },
				OnHTTPError:          func(_ *http.Request, status int, err error) { observed++; last, lastStatus = err, status },
			})
			issued := replayTestRequest(h.Handler(), http.MethodGet, "/sso/authorize", url.Values{"client": {client.ClientID}, "redirect": {client.RedirectURIs[0]}, "mode": {string(mode)}})
			location, err := url.Parse(issued.Header().Get("Location"))
			if err != nil || issued.Code != 302 {
				t.Fatal("could not issue credential")
			}
			values := location.Query()
			values.Set("client", client.ClientID)
			values.Set("clientSecret", client.ClientSecret)
			values.Set("redirect", client.RedirectURIs[0])
			values.Set("mode", string(mode))
			stage = "policy"
			assertHTTPFailure(t, replayTestRequest(h.Handler(), http.MethodPost, "/sso/token", values), 503)
			if observed != 1 || enriched != 0 || lastStatus != 503 || !errors.Is(last, failure) || !errors.Is(last, ErrAuthorizationPolicyUnavailable) {
				t.Fatal("policy failure lifecycle or diagnostics changed")
			}
			stage = "enrichment"
			assertHTTPFailure(t, replayTestRequest(h.Handler(), http.MethodPost, "/sso/token", values), 500)
			if observed != 2 || enriched != 1 || lastStatus != 500 || !errors.Is(last, failure) {
				t.Fatal("credential was consumed before policy approval or hook error was lost")
			}
			assertHTTPFailure(t, replayTestRequest(h.Handler(), http.MethodPost, "/sso/token", values), 400)
			if observed != 3 || enriched != 1 {
				t.Fatal("failed enrichment restored consumed credential")
			}
		})
	}
}

// TestHTTPErrorValidationBoundaries observes rejections once without invoking business hooks. TestHTTPErrorValidationBoundaries 对校验拒绝观察一次且不调用业务钩子。
func TestHTTPErrorValidationBoundaries(t *testing.T) {
	s := NewServer()
	defer s.Close()
	calls, effects := 0, 0
	h := NewHTTPServer(s, HTTPOptions{
		RequestLimits:      RequestLimits{MaxQueryBytes: 64, MaxBodyBytes: 64},
		AuthorizationExtra: func(*http.Request, string, string) (map[string]any, error) { effects++; return nil, nil },
		OnHTTPError:        func(*http.Request, int, error) { calls++ },
	})
	for _, path := range []string{"/sso/authorize", "/sso/token", "/sso/introspect", "/sso/userinfo", "/sso/revoke", "/sso/logout"} {
		for _, tc := range []struct {
			method, query string
			status        int
		}{
			{http.MethodDelete, "", 405},
			{http.MethodPost, "?private=%GG", 400},
			{http.MethodPost, "?private=" + strings.Repeat("x", 65), 413},
		} {
			method := tc.method
			if path == "/sso/authorize" && method == http.MethodPost {
				method = http.MethodGet
			}
			before := calls
			w := httptest.NewRecorder()
			h.Handler().ServeHTTP(w, httptest.NewRequest(method, path+tc.query, nil))
			assertHTTPFailure(t, w, tc.status)
			if calls != before+1 || effects != 0 {
				t.Fatal("validation observer or business boundary changed")
			}
		}
	}
}
