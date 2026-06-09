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

	req := httptest.NewRequest(http.MethodGet, "/sso/introspect?mode=shared_token&client=app-a&tokenValue="+token.Token, nil)
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

func TestSharedCookieHelpers(t *testing.T) {
	options := CookieOptions{Name: "sso_login", Domain: ".example.com"}
	rec := httptest.NewRecorder()
	SetLoginIDCookie(rec, options, "user-1001")

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "sso_login" || cookies[0].Value != "user-1001" {
		t.Fatalf("SetLoginIDCookie() cookies = %+v, want sso_login user-1001", cookies)
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(cookies[0])
	loginID, ok := LoginIDFromCookie(options)(req)
	if !ok || loginID != "user-1001" {
		t.Fatalf("LoginIDFromCookie() = %q, %v; want user-1001, true", loginID, ok)
	}
}
