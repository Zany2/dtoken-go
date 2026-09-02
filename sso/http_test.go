package sso

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// TestHTTPServerAuthorizeRedirectsWithTicket verifies the HTTP Server Authorize Redirects With Ticket scenario. TestHTTPServerAuthorizeRedirectsWithTicket 验证对应的 HTTP SSO 场景。
func TestHTTPServerAuthorizeRedirectsWithTicket(t *testing.T) {
	server := NewServer()
	registerTestClient(t, server)

	handler := NewHTTPServer(server, HTTPOptions{
		ServerOptions: ServerOptions{
			CheckSign: false,
			Endpoints: DefaultEndpoints(),
			Params:    DefaultParamNames(),
		},
		LoginIDResolver: func(*http.Request) (string, bool) {
			return "user-1001", true
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/sso/authorize?client=app-a&redirect="+url.QueryEscape("https://app.example.com/sso/callback"), nil)
	rec := httptest.NewRecorder()
	handler.HandleAuthorize(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("HandleAuthorize() status = %d, want 302", rec.Code)
	}
	location := rec.Header().Get("Location")
	parsed, err := url.Parse(location)
	if err != nil {
		t.Fatalf("url.Parse(Location) error = %v", err)
	}
	if parsed.Scheme != "https" || parsed.Host != "app.example.com" || parsed.Query().Get("ticket") == "" {
		t.Fatalf("Location = %q, want callback URL with ticket", location)
	}
}

// TestHTTPServerAuthorizeIssuesRequestedOAuth2Code verifies authorize dispatches the requested supported mode. TestHTTPServerAuthorizeIssuesRequestedOAuth2Code 验证授权端点会分发请求的受支持模式。
func TestHTTPServerAuthorizeIssuesRequestedOAuth2Code(t *testing.T) {
	server := NewServer()
	client := newTestClient()
	client.Modes = []Mode{ModeOAuth2}
	if err := server.RegisterClient(client); err != nil {
		t.Fatalf("RegisterClient() error = %v", err)
	}
	handler := NewHTTPServer(server, HTTPOptions{
		ServerOptions: ServerOptions{CheckSign: false, Endpoints: DefaultEndpoints(), Params: DefaultParamNames()},
		LoginIDResolver: func(*http.Request) (string, bool) {
			return "user-1001", true
		},
	})

	authorizeURL := "/sso/authorize?mode=oauth2&client=app-a&redirect=" + url.QueryEscape(client.RedirectURIs[0])
	recorder := httptest.NewRecorder()
	handler.HandleAuthorize(recorder, httptest.NewRequest(http.MethodGet, authorizeURL, nil))
	if recorder.Code != http.StatusFound {
		t.Fatalf("HandleAuthorize(oauth2) status = %d, want 302, body=%s", recorder.Code, recorder.Body.String())
	}
	location, err := url.Parse(recorder.Header().Get("Location"))
	if err != nil {
		t.Fatalf("url.Parse(Location) error = %v", err)
	}
	code := location.Query().Get("code")
	if code == "" || location.Query().Get("ticket") != "" {
		t.Fatalf("HandleAuthorize(oauth2) Location = %q, want code only", location.String())
	}
	if consumed, err := server.ConsumeOAuth2Code(context.Background(), code, client.ClientID, client.ClientSecret, client.RedirectURIs[0]); err != nil || consumed == nil || consumed.LoginID != "user-1001" {
		t.Fatalf("ConsumeOAuth2Code() = %+v, %v", consumed, err)
	}
}

// TestHTTPServerAuthorizeIssuesSharedAndRemoteCredentials verifies authorize dispatches reusable credential modes. TestHTTPServerAuthorizeIssuesSharedAndRemoteCredentials 验证授权端点会分发可复用凭证模式。
func TestHTTPServerAuthorizeIssuesSharedAndRemoteCredentials(t *testing.T) {
	for _, test := range []struct {
		mode  Mode
		param string
	}{
		{mode: ModeSharedToken, param: "tokenValue"},
		{mode: ModeRemoteSession, param: "sessionId"},
	} {
		t.Run(string(test.mode), func(t *testing.T) {
			server := NewServer()
			client := newTestClient()
			client.Modes = []Mode{test.mode}
			if err := server.RegisterClient(client); err != nil {
				t.Fatalf("RegisterClient() error = %v", err)
			}
			handler := NewHTTPServer(server, HTTPOptions{
				ServerOptions: ServerOptions{CheckSign: false},
				LoginIDResolver: func(*http.Request) (string, bool) {
					return "user-1001", true
				},
			})

			authorizeURL := "/sso/authorize?mode=" + url.QueryEscape(string(test.mode)) +
				"&client=" + url.QueryEscape(client.ClientID) +
				"&redirect=" + url.QueryEscape(client.RedirectURIs[0])
			recorder := httptest.NewRecorder()
			handler.HandleAuthorize(recorder, httptest.NewRequest(http.MethodGet, authorizeURL, nil))
			if recorder.Code != http.StatusFound {
				t.Fatalf("HandleAuthorize(%s) status = %d, want 302, body=%s", test.mode, recorder.Code, recorder.Body.String())
			}
			location, err := url.Parse(recorder.Header().Get("Location"))
			if err != nil {
				t.Fatalf("url.Parse(Location) error = %v", err)
			}
			if location.Query().Get(test.param) == "" || location.Query().Get("ticket") != "" || location.Query().Get("code") != "" {
				t.Fatalf("HandleAuthorize(%s) Location = %q, want %s only", test.mode, location.String(), test.param)
			}
		})
	}
}

// TestHTTPServerSensitiveEndpointsRequirePost verifies credential-bearing endpoints reject GET. TestHTTPServerSensitiveEndpointsRequirePost 验证携带凭证的端点拒绝 GET。
func TestHTTPServerSensitiveEndpointsRequirePost(t *testing.T) {
	handler := NewHTTPServer(NewServer(), HTTPOptions{ServerOptions: ServerOptions{CheckSign: false}}).Handler()
	for _, path := range []string{"/sso/token", "/sso/introspect", "/sso/userinfo", "/sso/revoke"} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusMethodNotAllowed {
			t.Fatalf("GET %s status = %d, want 405", path, recorder.Code)
		}
	}
}

// TestHTTPServerClientSessionLimit verifies the configured per-account callback limit while allowing updates. TestHTTPServerClientSessionLimit 验证账号回调注册上限并允许更新已有客户端。
func TestHTTPServerClientSessionLimit(t *testing.T) {
	server := NewServer()
	for _, client := range []*Client{
		{ClientID: "app-a", RedirectURIs: []string{"https://a.example.com/callback"}, Modes: []Mode{ModeTicket}},
		{ClientID: "app-b", RedirectURIs: []string{"https://b.example.com/callback"}, Modes: []Mode{ModeTicket}},
	} {
		if err := server.RegisterClient(client); err != nil {
			t.Fatalf("RegisterClient(%s) error = %v", client.ClientID, err)
		}
	}
	handler := NewHTTPServer(server, HTTPOptions{ServerOptions: ServerOptions{MaxRegisteredClient: 1, CheckSign: false}})
	ctx := context.Background()
	if _, err := handler.registerClientSession(ctx, "user-1", "app-a", "https://a.example.com/logout"); err != nil {
		t.Fatalf("registerClientSession(app-a) error = %v", err)
	}
	if _, err := handler.registerClientSession(ctx, "user-1", "app-a", "https://a.example.com/logout-2"); err != nil {
		t.Fatalf("registerClientSession(app-a update) error = %v", err)
	}
	if _, err := handler.registerClientSession(ctx, "user-1", "app-b", "https://b.example.com/logout"); !errors.Is(err, ErrClientSessionLimit) {
		t.Fatalf("registerClientSession(app-b) error = %v, want ErrClientSessionLimit", err)
	}
}

// TestHTTPServerAuthorizeRejectsMissingClientByDefault verifies the HTTP Server Authorize Rejects Missing Client By Default scenario. TestHTTPServerAuthorizeRejectsMissingClientByDefault 验证对应的 HTTP SSO 场景。
func TestHTTPServerAuthorizeRejectsMissingClientByDefault(t *testing.T) {
	server := NewServer()
	handler := NewHTTPServer(server, HTTPOptions{
		ServerOptions: ServerOptions{
			CheckSign: false,
			Endpoints: DefaultEndpoints(),
			Params:    DefaultParamNames(),
		},
		LoginIDResolver: func(*http.Request) (string, bool) {
			return "user-1001", true
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/sso/authorize?redirect="+url.QueryEscape("https://public.example.com/sso/callback"), nil)
	rec := httptest.NewRecorder()
	handler.HandleAuthorize(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("HandleAuthorize() status = %d, want 400", rec.Code)
	}
}

// TestHTTPServerAuthorizeAllowsAnonymousClient verifies the HTTP Server Authorize Allows Anonymous Client scenario. TestHTTPServerAuthorizeAllowsAnonymousClient 验证对应的 HTTP SSO 场景。
func TestHTTPServerAuthorizeAllowsAnonymousClient(t *testing.T) {
	server := NewServer()
	handler := NewHTTPServer(server, HTTPOptions{
		ServerOptions: ServerOptions{
			AllowAnonymousClient: true,
			AllowURLs:            []string{"https://public.example.com/sso/callback"},
			CheckSign:            false,
			Endpoints:            DefaultEndpoints(),
			Params:               DefaultParamNames(),
		},
		LoginIDResolver: func(*http.Request) (string, bool) {
			return "user-1001", true
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/sso/authorize?redirect="+url.QueryEscape("https://public.example.com/sso/callback"), nil)
	rec := httptest.NewRecorder()
	handler.HandleAuthorize(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("HandleAuthorize() status = %d, want 302, body=%s", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	parsed, err := url.Parse(location)
	if err != nil {
		t.Fatalf("url.Parse(Location) error = %v", err)
	}
	if parsed.Host != "public.example.com" || parsed.Query().Get("ticket") == "" {
		t.Fatalf("Location = %q, want public callback with ticket", location)
	}
}

// TestHTTPServerAuthorizeKeepsRegisteredAnonymousClient verifies the HTTP Server Authorize Keeps Registered Anonymous Client scenario. TestHTTPServerAuthorizeKeepsRegisteredAnonymousClient 验证对应的 HTTP SSO 场景。
func TestHTTPServerAuthorizeKeepsRegisteredAnonymousClient(t *testing.T) {
	server := NewServer()
	if err := server.RegisterClient(&Client{
		ClientID:     ClientAnonymous,
		RedirectURIs: []string{"https://custom.example.com/sso/callback"},
		Modes:        []Mode{ModeTicket},
	}); err != nil {
		t.Fatalf("RegisterClient() error = %v", err)
	}
	handler := NewHTTPServer(server, HTTPOptions{
		ServerOptions: ServerOptions{
			AllowAnonymousClient: true,
			AllowURLs:            []string{"https://default.example.com/sso/callback"},
			CheckSign:            false,
			Endpoints:            DefaultEndpoints(),
			Params:               DefaultParamNames(),
		},
		LoginIDResolver: func(*http.Request) (string, bool) {
			return "user-1001", true
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/sso/authorize?redirect="+url.QueryEscape("https://custom.example.com/sso/callback"), nil)
	rec := httptest.NewRecorder()
	handler.HandleAuthorize(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("HandleAuthorize() status = %d, want 302, body=%s", rec.Code, rec.Body.String())
	}
}

// TestHTTPServerTokenConsumesTicket verifies the HTTP Server Token Consumes Ticket scenario. TestHTTPServerTokenConsumesTicket 验证对应的 HTTP SSO 场景。
func TestHTTPServerTokenConsumesTicket(t *testing.T) {
	server := NewServer()
	registerTestClient(t, server)

	ticket, err := server.GenerateTicket(context.Background(), "app-a", "user-1001", "https://app.example.com/sso/callback", nil, nil)
	if err != nil {
		t.Fatalf("GenerateTicket() error = %v", err)
	}
	handler := NewHTTPServer(server, HTTPOptions{
		ServerOptions: ServerOptions{
			CheckSign: false,
			Endpoints: DefaultEndpoints(),
			Params:    DefaultParamNames(),
		},
	})

	form := url.Values{}
	form.Set("ticket", ticket.Ticket)
	form.Set("client", "app-a")
	form.Set("clientSecret", "secret-a")
	form.Set("redirect", "https://app.example.com/sso/callback")
	req := httptest.NewRequest(http.MethodPost, "/sso/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	handler.HandleToken(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("HandleToken() status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var response Response
	if err = json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	data, ok := response.Data.(map[string]any)
	if !ok || data["loginId"] != "user-1001" {
		t.Fatalf("HandleToken() data = %#v, want loginId user-1001", response.Data)
	}
}

// TestHTTPServerTokenConsumesOAuth2Code verifies OAuth2 Code exchange through the HTTP token endpoint. TestHTTPServerTokenConsumesOAuth2Code 验证通过 HTTP Token 端点交换 OAuth2 授权码。
func TestHTTPServerTokenConsumesOAuth2Code(t *testing.T) {
	server := NewServer()
	client := newTestClient()
	client.Modes = []Mode{ModeOAuth2}
	if err := server.RegisterClient(client); err != nil {
		t.Fatalf("RegisterClient() error = %v", err)
	}

	code, err := server.GenerateOAuth2Code(context.Background(), "app-a", "user-1001", "https://app.example.com/sso/callback", nil, nil)
	if err != nil {
		t.Fatalf("GenerateOAuth2Code() error = %v", err)
	}
	handler := NewHTTPServer(server, HTTPOptions{
		ServerOptions: ServerOptions{
			CheckSign: false,
			Endpoints: DefaultEndpoints(),
			Params:    DefaultParamNames(),
		},
	})

	form := url.Values{}
	form.Set("code", code.Code)
	form.Set("client", "app-a")
	form.Set("clientSecret", "secret-a")
	form.Set("redirect", "https://app.example.com/sso/callback")
	req := httptest.NewRequest(http.MethodPost, "/sso/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	handler.HandleToken(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("HandleToken() status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var response Response
	if err = json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	data, ok := response.Data.(map[string]any)
	if !ok || data["loginId"] != "user-1001" {
		t.Fatalf("HandleToken() data = %#v, want loginId user-1001", response.Data)
	}
}

// TestHTTPServerIntrospectSharedToken verifies the HTTP Server Introspect Shared Token scenario. TestHTTPServerIntrospectSharedToken 验证对应的 HTTP SSO 场景。
func TestHTTPServerIntrospectSharedToken(t *testing.T) {
	server := NewServer()
	client := newTestClient()
	client.Modes = []Mode{ModeSharedToken}
	if err := server.RegisterClient(client); err != nil {
		t.Fatalf("RegisterClient() error = %v", err)
	}
	token, err := server.GenerateSharedToken(context.Background(), "app-a", "user-1001", nil, nil)
	if err != nil {
		t.Fatalf("GenerateSharedToken() error = %v", err)
	}
	handler := NewHTTPServer(server, HTTPOptions{
		ServerOptions: ServerOptions{
			CheckSign: false,
			Endpoints: DefaultEndpoints(),
			Params:    DefaultParamNames(),
		},
	})

	form := url.Values{}
	form.Set("mode", string(ModeSharedToken))
	form.Set("client", "app-a")
	form.Set("clientSecret", "secret-a")
	form.Set("tokenValue", token.Token)
	req := httptest.NewRequest(http.MethodPost, "/sso/introspect", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	handler.HandleIntrospect(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("HandleIntrospect() status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var response Response
	if err = json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	data, ok := response.Data.(map[string]any)
	if !ok || data["active"] != true || data["loginId"] != "user-1001" {
		t.Fatalf("HandleIntrospect() data = %#v, want active user-1001", response.Data)
	}
}

// TestHTTPServerRevokeRemoteSession verifies the HTTP Server Revoke Remote Session scenario. TestHTTPServerRevokeRemoteSession 验证对应的 HTTP SSO 场景。
func TestHTTPServerRevokeRemoteSession(t *testing.T) {
	server := NewServer()
	client := newTestClient()
	client.Modes = []Mode{ModeRemoteSession}
	if err := server.RegisterClient(client); err != nil {
		t.Fatalf("RegisterClient() error = %v", err)
	}
	session, err := server.CreateRemoteSession(context.Background(), "app-a", "user-1001", nil, nil)
	if err != nil {
		t.Fatalf("CreateRemoteSession() error = %v", err)
	}
	handler := NewHTTPServer(server, HTTPOptions{
		ServerOptions: ServerOptions{
			CheckSign: false,
			Endpoints: DefaultEndpoints(),
			Params:    DefaultParamNames(),
		},
	})

	form := url.Values{}
	form.Set("mode", "remote_session")
	form.Set("client", "app-a")
	form.Set("clientSecret", "secret-a")
	form.Set("sessionId", session.SessionID)
	req := httptest.NewRequest(http.MethodPost, "/sso/revoke", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	handler.HandleRevoke(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("HandleRevoke() status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if _, err = server.ValidateRemoteSession(context.Background(), session.SessionID, "app-a"); !errors.Is(err, ErrInvalidRemoteSession) {
		t.Fatalf("ValidateRemoteSession() after revoke error = %v, want ErrInvalidRemoteSession", err)
	}
}

// TestHTTPServerRevokeIsIdempotentAndClientScoped verifies revoke ownership checks and repeated deletion behavior. TestHTTPServerRevokeIsIdempotentAndClientScoped 验证撤销归属校验与重复删除幂等性。
func TestHTTPServerRevokeIsIdempotentAndClientScoped(t *testing.T) {
	server := NewServer()
	owner := newTestClient()
	other := &Client{ClientID: "app-b", ClientSecret: "secret-b", RedirectURIs: []string{"https://other.example.com/callback"}, Modes: []Mode{ModeTicket}}
	if err := server.RegisterClient(owner); err != nil {
		t.Fatalf("RegisterClient(owner) error = %v", err)
	}
	if err := server.RegisterClient(other); err != nil {
		t.Fatalf("RegisterClient(other) error = %v", err)
	}
	ticket, err := server.GenerateTicket(context.Background(), owner.ClientID, "user-1", owner.RedirectURIs[0], nil, nil)
	if err != nil {
		t.Fatalf("GenerateTicket() error = %v", err)
	}
	handler := NewHTTPServer(server, HTTPOptions{ServerOptions: ServerOptions{CheckSign: false}})

	requestRevoke := func(clientID, clientSecret, ticketValue string) *httptest.ResponseRecorder {
		form := url.Values{
			"mode":         {string(ModeTicket)},
			"client":       {clientID},
			"clientSecret": {clientSecret},
			"ticket":       {ticketValue},
		}
		request := httptest.NewRequest(http.MethodPost, "/sso/revoke", strings.NewReader(form.Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		recorder := httptest.NewRecorder()
		handler.HandleRevoke(recorder, request)
		return recorder
	}

	if recorder := requestRevoke(other.ClientID, other.ClientSecret, ticket.Ticket); recorder.Code != http.StatusBadRequest {
		t.Fatalf("revoke by other client status = %d, want 400", recorder.Code)
	}
	if recorder := requestRevoke(owner.ClientID, owner.ClientSecret, ticket.Ticket); recorder.Code != http.StatusOK {
		t.Fatalf("revoke by owner status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder := requestRevoke(owner.ClientID, owner.ClientSecret, ticket.Ticket); recorder.Code != http.StatusOK {
		t.Fatalf("repeat revoke status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
}

// TestHTTPServerLogoutRejectsNilServer verifies the HTTP Server Logout Rejects Nil Server scenario. TestHTTPServerLogoutRejectsNilServer 验证对应的 HTTP SSO 场景。
func TestHTTPServerLogoutRejectsNilServer(t *testing.T) {
	handler := NewHTTPServer(nil, HTTPOptions{
		ServerOptions: ServerOptions{
			EnableSLO: true,
			CheckSign: false,
			Endpoints: DefaultEndpoints(),
			Params:    DefaultParamNames(),
		},
	})

	form := url.Values{}
	form.Set("loginId", "user-1001")
	req := httptest.NewRequest(http.MethodPost, "/sso/logout", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	handler.HandleLogout(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("HandleLogout() status = %d, want 500", rec.Code)
	}
}

// TestHTTPServerLogoutRequiresResolvedIdentity verifies request parameters cannot select another logout subject. TestHTTPServerLogoutRequiresResolvedIdentity 验证请求参数不能选择其他注销主体。
func TestHTTPServerLogoutRequiresResolvedIdentity(t *testing.T) {
	server := NewServer()
	handler := NewHTTPServer(server, HTTPOptions{
		ServerOptions: ServerOptions{EnableSLO: true, CheckSign: false},
		LoginIDResolver: func(*http.Request) (string, bool) {
			return "trusted-user", true
		},
	})

	form := url.Values{"loginId": {"other-user"}}
	request := httptest.NewRequest(http.MethodPost, "/sso/logout", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	handler.HandleLogout(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("HandleLogout(mismatch) status = %d, want 401", recorder.Code)
	}

	unauthenticated := NewHTTPServer(server, HTTPOptions{
		ServerOptions: ServerOptions{EnableSLO: true, CheckSign: false},
		LoginIDResolver: func(*http.Request) (string, bool) {
			return "", false
		},
	})
	recorder = httptest.NewRecorder()
	unauthenticated.HandleLogout(recorder, httptest.NewRequest(http.MethodPost, "/sso/logout", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("HandleLogout(anonymous) status = %d, want 401", recorder.Code)
	}
}

// TestHTTPServerSingleLogoutCallback verifies the HTTP Server Single Logout Callback scenario. TestHTTPServerSingleLogoutCallback 验证对应的 HTTP SSO 场景。
func TestHTTPServerSingleLogoutCallback(t *testing.T) {
	server := NewServer()

	var callbackCalled atomic.Bool
	callbackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("logout callback method = %s, want POST", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("logout callback ParseForm() error = %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.FormValue("loginId") != "user-1001" || r.FormValue("client") != "app-a" {
			t.Errorf("logout callback form = %v, want user-1001 app-a", r.Form)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		callbackCalled.Store(true)
		w.WriteHeader(http.StatusOK)
	}))
	defer callbackServer.Close()

	client := newTestClient()
	client.AllowOrigins = []string{callbackServer.URL}
	if err := server.RegisterClient(client); err != nil {
		t.Fatalf("RegisterClient() error = %v", err)
	}

	handler := NewHTTPServer(server, HTTPOptions{
		ServerOptions: ServerOptions{
			EnableSLO: true,
			CheckSign: false,
			Endpoints: DefaultEndpoints(),
			Params:    DefaultParamNames(),
		},
		LoginIDResolver: func(*http.Request) (string, bool) {
			return "user-1001", true
		},
	})

	authorizeURL := "/sso/authorize?client=app-a&redirect=" +
		url.QueryEscape("https://app.example.com/sso/callback") +
		"&callback=" + url.QueryEscape(callbackServer.URL)
	authorizeReq := httptest.NewRequest(http.MethodGet, authorizeURL, nil)
	authorizeRec := httptest.NewRecorder()
	handler.HandleAuthorize(authorizeRec, authorizeReq)
	if authorizeRec.Code != http.StatusFound {
		t.Fatalf("HandleAuthorize() status = %d, want 302, body=%s", authorizeRec.Code, authorizeRec.Body.String())
	}
	sessions, err := server.GetClientSessions(context.Background(), "user-1001")
	if err != nil {
		t.Fatalf("GetClientSessions() error = %v", err)
	}
	if len(sessions) != 1 || sessions[0].LogoutCallbackURL != callbackServer.URL {
		t.Fatalf("GetClientSessions() = %+v, want registered callback", sessions)
	}

	form := url.Values{}
	form.Set("loginId", "user-1001")
	logoutReq := httptest.NewRequest(http.MethodPost, "/sso/logout", strings.NewReader(form.Encode()))
	logoutReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	logoutRec := httptest.NewRecorder()
	handler.HandleLogout(logoutRec, logoutReq)
	if logoutRec.Code != http.StatusOK {
		t.Fatalf("HandleLogout() status = %d, want 200, body=%s", logoutRec.Code, logoutRec.Body.String())
	}
	if !callbackCalled.Load() {
		t.Fatal("HandleLogout() did not call registered client logout callback")
	}
	sessions, err = server.GetClientSessions(context.Background(), "user-1001")
	if err != nil {
		t.Fatalf("GetClientSessions() after logout error = %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("GetClientSessions() after logout = %+v, want empty", sessions)
	}
}

// TestHTTPServerSingleLogoutBestEffortClearsSessions verifies the HTTP Server Single Logout Best Effort Clears Sessions scenario. TestHTTPServerSingleLogoutBestEffortClearsSessions 验证对应的 HTTP SSO 场景。
func TestHTTPServerSingleLogoutBestEffortClearsSessions(t *testing.T) {
	server := NewServer()
	client := newTestClient()
	client.AllowOrigins = []string{"http://127.0.0.1:1"}
	if err := server.RegisterClient(client); err != nil {
		t.Fatalf("RegisterClient() error = %v", err)
	}
	if _, err := server.RegisterClientSession(context.Background(), "user-1001", "app-a", "http://127.0.0.1:1/logout"); err != nil {
		t.Fatalf("RegisterClientSession() error = %v", err)
	}

	handler := NewHTTPServer(server, HTTPOptions{
		ServerOptions: ServerOptions{
			EnableSLO:                true,
			LogoutCallbackBestEffort: true,
			LogoutCallbackTimeout:    10 * time.Millisecond,
			CheckSign:                false,
			Endpoints:                DefaultEndpoints(),
			Params:                   DefaultParamNames(),
		},
		LoginIDResolver: func(*http.Request) (string, bool) {
			return "user-1001", true
		},
	})

	form := url.Values{}
	form.Set("loginId", "user-1001")
	req := httptest.NewRequest(http.MethodPost, "/sso/logout", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	handler.HandleLogout(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("HandleLogout() best-effort status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	sessions, err := server.GetClientSessions(context.Background(), "user-1001")
	if err != nil {
		t.Fatalf("GetClientSessions() error = %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("GetClientSessions() = %+v, want empty after best-effort logout", sessions)
	}
}

// TestHTTPServerLogoutDoesNotFollowCallbackRedirects verifies default logout callbacks do not follow redirects to another host. TestHTTPServerLogoutDoesNotFollowCallbackRedirects 验证默认注销回调不会跟随重定向访问其他主机。
func TestHTTPServerLogoutDoesNotFollowCallbackRedirects(t *testing.T) {
	server := NewServer()
	var redirected atomic.Bool
	second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirected.Store(true)
		w.WriteHeader(http.StatusOK)
	}))
	defer second.Close()
	callback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, second.URL, http.StatusFound)
	}))
	defer callback.Close()

	client := newTestClient()
	client.AllowOrigins = []string{callback.URL}
	if err := server.RegisterClient(client); err != nil {
		t.Fatalf("RegisterClient() error = %v", err)
	}
	if _, err := server.RegisterClientSession(context.Background(), "user-1", client.ClientID, callback.URL); err != nil {
		t.Fatalf("RegisterClientSession() error = %v", err)
	}

	handler := NewHTTPServer(server, HTTPOptions{
		ServerOptions: ServerOptions{
			EnableSLO: true,
			CheckSign: false,
		},
		LoginIDResolver: func(*http.Request) (string, bool) {
			return "user-1", true
		},
	})
	request := httptest.NewRequest(http.MethodPost, "/sso/logout", nil)
	recorder := httptest.NewRecorder()
	handler.HandleLogout(recorder, request)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("HandleLogout() status = %d, want 500 for rejected callback redirect", recorder.Code)
	}
	if redirected.Load() {
		t.Fatal("logout callback followed redirect to secondary target")
	}
}

// TestSharedCookieHelpers verifies the Shared Cookie Helpers scenario. TestSharedCookieHelpers 验证对应的 HTTP SSO 场景。
func TestSharedCookieHelpers(t *testing.T) {
	options := CookieOptions{Name: "sso_login", Domain: ".example.com", SecretKey: "cookie-secret"}
	rec := httptest.NewRecorder()
	SetLoginIDCookie(rec, options, "user-1001")

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "sso_login" || cookies[0].Value == "" || cookies[0].Value == "user-1001" {
		t.Fatalf("SetLoginIDCookie() cookies = %+v, want signed sso_login cookie", cookies)
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(cookies[0])
	loginID, ok := LoginIDFromCookie(options)(req)
	if !ok || loginID != "user-1001" {
		t.Fatalf("LoginIDFromCookie() = %q, %v; want user-1001, true", loginID, ok)
	}
	tamperedRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	tamperedRequest.AddCookie(&http.Cookie{Name: options.Name, Value: cookies[0].Value + "tampered"})
	if loginID, ok = LoginIDFromCookie(options)(tamperedRequest); ok || loginID != "" {
		t.Fatalf("LoginIDFromCookie(tampered) = %q, %v, want empty false", loginID, ok)
	}
	unsignedRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	unsignedRequest.AddCookie(&http.Cookie{Name: options.Name, Value: "user-1001"})
	if loginID, ok = LoginIDFromCookie(options)(unsignedRequest); ok || loginID != "" {
		t.Fatalf("LoginIDFromCookie(unsigned) = %q, %v, want empty false", loginID, ok)
	}
	missingSecretRecorder := httptest.NewRecorder()
	SetLoginIDCookie(missingSecretRecorder, CookieOptions{Name: "unsigned"}, "user-1001")
	missingSecretCookies := missingSecretRecorder.Result().Cookies()
	if len(missingSecretCookies) != 1 || missingSecretCookies[0].MaxAge != -1 {
		t.Fatalf("SetLoginIDCookie(missing secret) cookies = %+v, want cleared cookie", missingSecretCookies)
	}
}
