package sso

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
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
