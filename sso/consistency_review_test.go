package sso

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// assertSSONoStore checks protocol responses, including responses from callback goroutines. assertSSONoStore 检查协议响应，也适用于回调协程中的响应。
func assertSSONoStore(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	if response.Header().Get("Cache-Control") != "no-store" || response.Header().Get("Pragma") != "no-cache" {
		t.Errorf("cacheable SSO response: status=%d headers=%v", response.Code, response.Header())
	}
}

// TestSSOSignedProtocolLifecycle verifies all modes through real client/server handlers and both storage capability paths. TestSSOSignedProtocolLifecycle 通过真实客户端与服务端处理器验证全部模式及两种存储能力路径。
func TestSSOSignedProtocolLifecycle(t *testing.T) {
	for _, storageKind := range []string{"atomic", "basic"} {
		for _, mode := range []Mode{ModeTicket, ModeOAuth2, ModeSharedToken, ModeRemoteSession} {
			t.Run(storageKind+"/"+string(mode), func(t *testing.T) {
				ctx := context.Background()
				storageOption := WithStorage(NewMemoryStorage())
				if storageKind == "basic" {
					storageOption = WithStorage(&basicSSOStorage{inner: NewMemoryStorage()})
				}
				server := NewServer(storageOption)
				defer server.Close()
				registered := newTestClient()
				registered.Modes = []Mode{mode}
				if err := server.RegisterClient(registered); err != nil {
					t.Fatal(err)
				}
				other := *registered
				other.ClientID, other.ClientSecret = "app-b", "secret-b"
				if err := server.RegisterClient(&other); err != nil {
					t.Fatal(err)
				}

				// Exercise normalized custom parameters and gateway-prefixed routes together. 同时覆盖自定义参数补全及网关前缀路由。
				params := normalizeParamNames(ParamNames{Client: "application", ClientSecret: "application_secret", Back: "state", Sign: "signature", Timestamp: "at", Ticket: "ticket_id", Code: "code_id"})
				endpoints := DefaultEndpoints().AddPrefix("/gateway")
				var center http.Handler
				app := NewClientApp(ClientConfig{
					Mode: mode, ClientID: registered.ClientID, ClientSecret: registered.ClientSecret,
					ServerURL: "https://center.example.com", CheckSign: true, SecretKey: "protocol-secret",
					RegisterCallback: true, LogoutCallbackURL: "https://app.example.com/sso/logout-callback",
					Params: params, Endpoints: endpoints,
					HTTPClient: &http.Client{Transport: logoutReviewTransport(func(r *http.Request) (*http.Response, error) {
						recorder := httptest.NewRecorder()
						center.ServeHTTP(recorder, r)
						assertSSONoStore(t, recorder)
						return recorder.Result(), nil
					})},
				})
				var callbacks atomic.Int32
				callback := app.LogoutCallbackHandler(func(_ *http.Request, received LogoutCallback) error {
					callbacks.Add(1)
					if received.LoginID != "alice" || received.ClientID != registered.ClientID {
						t.Errorf("wrong callback identity: %+v", received)
					}
					return nil
				})
				center = NewHTTPServer(server, HTTPOptions{
					ServerOptions: ServerOptions{
						EnableSLO: true, CheckSign: true, SecretKey: "protocol-secret", Params: params, Endpoints: endpoints,
						LogoutHTTPClient: &http.Client{Transport: logoutReviewTransport(func(r *http.Request) (*http.Response, error) {
							if r.URL.String() != app.Config().LogoutCallbackURL {
								t.Errorf("unexpected callback target: %s", r.URL)
							}
							recorder := httptest.NewRecorder()
							callback.ServeHTTP(recorder, r)
							assertSSONoStore(t, recorder)
							return recorder.Result(), nil
						})},
					},
					LoginIDResolver: func(*http.Request) (string, bool) { return "alice", true },
				}).Handler()
				state := "browser state + & = 测试"
				authURL, err := app.AuthURL(registered.RedirectURIs[0], url.Values{params.Back: {state}, params.Scope: {"profile"}})
				if err != nil {
					t.Fatal(err)
				}
				authorized := httptest.NewRecorder()
				center.ServeHTTP(authorized, httptest.NewRequest(http.MethodGet, authURL, nil))
				assertSSONoStore(t, authorized)
				if authorized.Code != http.StatusFound || authorized.Header().Get("Referrer-Policy") != "no-referrer" {
					t.Fatalf("authorization: status=%d body=%s", authorized.Code, authorized.Body.String())
				}
				location, err := url.Parse(authorized.Header().Get("Location"))
				if err != nil || location.Query().Get(params.Back) != state {
					t.Fatalf("callback state mismatch: %v, %v", location, err)
				}
				values := location.Query()
				credential := CredentialRequest{Mode: mode, RedirectURI: registered.RedirectURIs[0], Ticket: values.Get(params.Ticket), Code: values.Get(params.Code), TokenValue: values.Get(params.TokenValue), SessionID: values.Get(params.SessionID)}
				info, err := app.Introspect(ctx, credential)
				if err != nil || !info.Active || info.LoginID != "alice" || info.ClientID != registered.ClientID || info.Mode != mode || info.ExpiresIn <= 0 {
					t.Fatalf("initial introspection: %+v, %v", info, err)
				}
				info, err = app.UserInfo(ctx, credential)
				if err != nil || !info.Active || info.LoginID != "alice" || len(info.Scopes) != 1 || info.Scopes[0] != "profile" {
					t.Fatalf("user info: %+v, %v", info, err)
				}

				// A different registered client must not read or revoke this credential. 其他已注册客户端不得读取或撤销当前凭证。
				otherConfig := app.Config()
				otherConfig.ClientID, otherConfig.ClientSecret = other.ClientID, other.ClientSecret
				otherApp := NewClientApp(otherConfig)
				if info, err := otherApp.Introspect(ctx, credential); err != nil || info.Active || info.LoginID != "" {
					t.Fatalf("cross-client introspection: %+v, %v", info, err)
				}
				if err := otherApp.Revoke(ctx, credential); err == nil {
					t.Fatal("cross-client revocation succeeded")
				}
				if mode == ModeTicket || mode == ModeOAuth2 {
					if _, err := otherApp.ExchangeCredential(ctx, credential); err == nil {
						t.Fatal("cross-client exchange succeeded")
					}
					wrongRedirect := credential
					wrongRedirect.RedirectURI = "https://app.example.com/wrong"
					if _, err := app.ExchangeCredential(ctx, wrongRedirect); err == nil {
						t.Fatal("exchange accepted a different redirect URI")
					}
					result, err := app.ExchangeCredential(ctx, credential)
					if err != nil || result.LoginID != "alice" {
						t.Fatalf("valid exchange after rejection: %+v, %v", result, err)
					}
					if _, err := app.ExchangeCredential(ctx, credential); err == nil {
						t.Fatal("one-time credential was replayed")
					}
				} else if info, err := app.Introspect(ctx, credential); err != nil || !info.Active {
					t.Fatalf("rejected revocation changed credential: %+v, %v", info, err)
				}
				for range 2 {
					if err := app.Revoke(ctx, credential); err != nil {
						t.Fatal(err)
					}
				}
				if info, err := app.Introspect(ctx, credential); err != nil || info.Active || info.LoginID != "" {
					t.Fatalf("removed credential: %+v, %v", info, err)
				}
				if _, err := app.UserInfo(ctx, credential); err == nil {
					t.Fatal("removed credential still exposes user info")
				}

				// Complete signed single logout through the real client callback handler. 通过真实客户端回调处理器完成带签名的单点注销。
				logoutURL, err := app.SignoutURL("alice", nil)
				if err != nil {
					t.Fatal(err)
				}
				loggedOut := httptest.NewRecorder()
				center.ServeHTTP(loggedOut, httptest.NewRequest(http.MethodGet, logoutURL, nil))
				assertSSONoStore(t, loggedOut)
				if loggedOut.Code != http.StatusOK || callbacks.Load() != 1 {
					t.Fatalf("logout: status=%d calls=%d body=%s", loggedOut.Code, callbacks.Load(), loggedOut.Body.String())
				}
				sessions, err := server.GetClientSessions(ctx, "alice")
				if err != nil || len(sessions) != 0 {
					t.Fatalf("logout retained client registrations: %+v, %v", sessions, err)
				}
				cookies := loggedOut.Result().Cookies()
				if len(cookies) != 1 || cookies[0].MaxAge != -1 {
					t.Fatalf("logout did not expire the center Cookie: %v", cookies)
				}
			})
		}
	}
}

// TestHTTPResponseProtectionCoversEarlyReturns verifies login redirects and rejected protocol requests cannot be cached. TestHTTPResponseProtectionCoversEarlyReturns 验证登录重定向及被拒绝的协议请求不会被缓存。
func TestHTTPResponseProtectionCoversEarlyReturns(t *testing.T) {
	server := NewServer()
	defer server.Close()
	center := NewHTTPServer(server, HTTPOptions{LoginPageURL: "https://center.example.com/login"})
	response := httptest.NewRecorder()
	center.HandleAuthorize(response, httptest.NewRequest(http.MethodGet, "/sso/authorize?back=browser-state", nil))
	assertSSONoStore(t, response)
	if response.Code != http.StatusFound || response.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Fatalf("login redirect: status=%d headers=%v", response.Code, response.Header())
	}
	for _, handler := range []http.HandlerFunc{center.HandleAuthorize, center.HandleToken, center.HandleIntrospect, center.HandleUserInfo, center.HandleRevoke, center.HandleLogout} {
		response = httptest.NewRecorder()
		response.Header().Set("Cache-Control", "public, max-age=3600")
		handler(response, httptest.NewRequest(http.MethodPut, "/", nil))
		assertSSONoStore(t, response)
		if response.Code != http.StatusMethodNotAllowed {
			t.Fatalf("wrong method status = %d", response.Code)
		}
	}
	response = httptest.NewRecorder()
	NewHTTPServer(nil, HTTPOptions{}).HandleAuthorize(response, httptest.NewRequest(http.MethodGet, "/", nil))
	assertSSONoStore(t, response)
	if response.Code != http.StatusInternalServerError || response.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Fatalf("startup error: status=%d headers=%v", response.Code, response.Header())
	}
}

// TestMalformedLogoutCallbackIsClientError verifies malformed forms never invoke the local logout action. TestMalformedLogoutCallbackIsClientError 验证损坏表单返回客户端错误且不触发本地注销。
func TestMalformedLogoutCallbackIsClientError(t *testing.T) {
	app := NewClientApp(ClientConfig{ClientID: "app-a"})
	called := false
	handler := app.LogoutCallbackHandler(func(*http.Request, LogoutCallback) error {
		called = true
		return nil
	})
	form := url.Values{"client": {"app-a"}, "loginId": {"alice"}, "timestamp": {time.Now().Format(time.RFC3339)}}
	for _, suffix := range []string{"&broken=%ZZ", "&broken=one;two"} {
		newRequest := func() *http.Request {
			request := httptest.NewRequest(http.MethodPost, "/sso/logout-callback", strings.NewReader(form.Encode()+suffix))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			return request
		}
		if _, err := app.VerifyLogoutCallback(newRequest()); !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("malformed form error = %v, want ErrInvalidRequest", err)
		}
		response := httptest.NewRecorder()
		handler(response, newRequest())
		assertSSONoStore(t, response)
		if response.Code != http.StatusBadRequest || called {
			t.Fatalf("malformed callback: status=%d actionCalled=%v", response.Code, called)
		}
	}
}
