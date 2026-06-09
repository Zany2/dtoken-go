package sso

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestEndpointsPrefixHelpers(t *testing.T) {
	api := DefaultEndpoints().ReplacePrefix("/auth-center")
	if api.Authorize != "/auth-center/authorize" || api.Token != "/auth-center/token" {
		t.Fatalf("ReplacePrefix() = %+v, want auth-center paths", api)
	}

	api = DefaultEndpoints().AddPrefix("/gateway")
	if api.Authorize != "/gateway/sso/authorize" || api.ClientCallback != "/gateway/sso/callback" {
		t.Fatalf("AddPrefix() = %+v, want gateway paths", api)
	}
}

func TestClientAppExchangesTicketOverHTTP(t *testing.T) {
	var gotValues url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sso/token" {
			t.Fatalf("request path = %q, want /sso/token", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm() error = %v", err)
		}
		gotValues = r.Form
		_ = json.NewEncoder(w).Encode(OKResponse(TicketExchangeResult{
			LoginID: "user-1001",
		}))
	}))
	defer server.Close()

	app := NewClientApp(ClientConfig{
		Mode:         ModeTicket,
		ClientID:     "app-a",
		ClientSecret: "secret-a",
		ServerURL:    server.URL,
		CheckSign:    false,
		Endpoints:    DefaultEndpoints(),
		Params:       DefaultParamNames(),
	})
	result, err := app.ExchangeTicket(context.Background(), "ticket-value", "https://app.example.com/sso/callback")
	if err != nil {
		t.Fatalf("ExchangeTicket() error = %v", err)
	}
	if result.LoginID != "user-1001" {
		t.Fatalf("ExchangeTicket() loginID = %q, want user-1001", result.LoginID)
	}
	if gotValues.Get("ticket") != "ticket-value" || gotValues.Get("client") != "app-a" || gotValues.Get("redirect") == "" {
		t.Fatalf("ExchangeTicket() form = %v, want ticket client and redirect", gotValues)
	}
}

func TestClientAppIntrospectAndRevokeOverHTTP(t *testing.T) {
	requests := make([]string, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Path)
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm() error = %v", err)
		}
		if r.Form.Get("mode") != string(ModeSharedToken) || r.Form.Get("tokenValue") != "token-value" {
			t.Fatalf("request form = %v, want shared token params", r.Form)
		}
		if r.URL.Path == "/sso/introspect" {
			_ = json.NewEncoder(w).Encode(OKResponse(CredentialInfo{
				Active:  true,
				Mode:    ModeSharedToken,
				LoginID: "user-1001",
			}))
			return
		}
		_ = json.NewEncoder(w).Encode(OKResponse(map[string]string{"result": ResultOK}))
	}))
	defer server.Close()

	app := NewClientApp(ClientConfig{
		Mode:         ModeTicket,
		ClientID:     "app-a",
		ClientSecret: "secret-a",
		ServerURL:    server.URL,
		CheckSign:    false,
		Endpoints:    DefaultEndpoints(),
		Params:       DefaultParamNames(),
	})
	info, err := app.Introspect(context.Background(), CredentialRequest{
		Mode:       ModeSharedToken,
		TokenValue: "token-value",
	})
	if err != nil {
		t.Fatalf("Introspect() error = %v", err)
	}
	if !info.Active || info.LoginID != "user-1001" {
		t.Fatalf("Introspect() = %+v, want active user-1001", info)
	}
	if err = app.Revoke(context.Background(), CredentialRequest{
		Mode:       ModeSharedToken,
		TokenValue: "token-value",
	}); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
	if strings.Join(requests, ",") != "/sso/introspect,/sso/revoke" {
		t.Fatalf("request paths = %v, want introspect then revoke", requests)
	}
}

func TestSignerIgnoresParameterOrder(t *testing.T) {
	signer := NewSigner("secret")
	a := url.Values{"b": {"2"}, "a": {"1"}}
	b := url.Values{"a": {"1"}, "b": {"2"}}
	if signer.Sign(a) != signer.Sign(b) {
		t.Fatal("Sign() should ignore parameter order")
	}

	signed := signer.AttachSign(a)
	if !signer.Verify(signed) {
		t.Fatal("Verify() = false, want true")
	}
	signed.Set("a", "changed")
	if signer.Verify(signed) {
		t.Fatal("Verify() = true after parameter change, want false")
	}
}

func TestSignerWithZeroParamsUsesDefaults(t *testing.T) {
	values := NewSignerWithParams("secret", ParamNames{}).AttachSign(url.Values{
		"client": {"app-a"},
	})
	if values.Get(DefaultParamNames().Sign) == "" {
		t.Fatalf("AttachSign() values = %v, want default sign parameter", values)
	}
	if !NewSigner("secret").Verify(values) {
		t.Fatal("Verify() with default signer = false, want true")
	}
}

func TestClientAppBuildsSignedURLs(t *testing.T) {
	app := NewClientApp(ClientConfig{
		Mode:         ModeTicket,
		ClientID:     "app-a",
		ClientSecret: "secret-a",
		ServerURL:    "https://sso.example.com",
		SecretKey:    "sign-secret",
		CheckSign:    true,
		Endpoints:    DefaultEndpoints(),
		Params:       DefaultParamNames(),
	})

	authURL, err := app.AuthURL("https://app.example.com/sso/callback", nil)
	if err != nil {
		t.Fatalf("AuthURL() error = %v", err)
	}
	if !strings.HasPrefix(authURL, "https://sso.example.com/sso/authorize?") {
		t.Fatalf("AuthURL() = %q, want sso auth URL", authURL)
	}
	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	values := parsed.Query()
	if values.Get("client") != "app-a" || values.Get("redirect") == "" || values.Get("sign") == "" {
		t.Fatalf("AuthURL() query = %v, want client redirect and sign", values)
	}
	if !NewSigner("sign-secret").Verify(values) {
		t.Fatal("AuthURL() signature verification failed")
	}

	exchangeURL, err := app.ExchangeTicketURLWithRedirect("ticket-value", "https://app.example.com/sso/callback", nil)
	if err != nil {
		t.Fatalf("ExchangeTicketURLWithRedirect() error = %v", err)
	}
	parsed, err = url.Parse(exchangeURL)
	if err != nil {
		t.Fatalf("url.Parse(exchangeURL) error = %v", err)
	}
	values = parsed.Query()
	if values.Get("ticket") != "ticket-value" || values.Get("redirect") == "" || values.Get("clientSecret") != "secret-a" {
		t.Fatalf("ExchangeTicketURLWithRedirect() query = %v, want ticket redirect and clientSecret", values)
	}
}

func TestClientAppVerifiesLogoutCallback(t *testing.T) {
	app := NewClientApp(ClientConfig{
		ClientID:  "app-a",
		SecretKey: "sign-secret",
		CheckSign: true,
		Params:    DefaultParamNames(),
	})
	values := url.Values{}
	values.Set("loginId", "user-1001")
	values.Set("client", "app-a")
	values.Set("timestamp", time.Now().Format(time.RFC3339))
	values = NewSigner("sign-secret").AttachSign(values)

	req := httptest.NewRequest(http.MethodPost, "/sso/logout-callback", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	callback, err := app.VerifyLogoutCallback(req)
	if err != nil {
		t.Fatalf("VerifyLogoutCallback() error = %v", err)
	}
	if callback.LoginID != "user-1001" || callback.ClientID != "app-a" || callback.Timestamp == "" {
		t.Fatalf("VerifyLogoutCallback() = %+v, want parsed callback", callback)
	}

	values.Set("loginId", "user-2002")
	badReq := httptest.NewRequest(http.MethodPost, "/sso/logout-callback", strings.NewReader(values.Encode()))
	badReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if _, err = app.VerifyLogoutCallback(badReq); err == nil {
		t.Fatal("VerifyLogoutCallback() tampered request error = nil, want error")
	}
}

func TestClientAppRejectsExpiredLogoutCallback(t *testing.T) {
	app := NewClientApp(ClientConfig{
		ClientID:             "app-a",
		CheckSign:            false,
		LogoutCallbackMaxAge: time.Minute,
		Params:               DefaultParamNames(),
	})
	values := url.Values{}
	values.Set("loginId", "user-1001")
	values.Set("client", "app-a")
	values.Set("timestamp", time.Now().Add(-2*time.Minute).Format(time.RFC3339))

	req := httptest.NewRequest(http.MethodPost, "/sso/logout-callback", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if _, err := app.VerifyLogoutCallback(req); !errors.Is(err, ErrCallbackExpired) {
		t.Fatalf("VerifyLogoutCallback() expired error = %v, want ErrCallbackExpired", err)
	}
}

func TestClientAppLogoutCallbackHandler(t *testing.T) {
	app := NewClientApp(ClientConfig{
		ClientID:  "app-a",
		CheckSign: false,
		Params:    DefaultParamNames(),
	})
	calledLoginID := ""
	handler := app.LogoutCallbackHandler(func(_ *http.Request, callback LogoutCallback) error {
		calledLoginID = callback.LoginID
		return nil
	})

	form := url.Values{}
	form.Set("loginId", "user-1001")
	form.Set("client", "app-a")
	req := httptest.NewRequest(http.MethodPost, "/sso/logout-callback", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("LogoutCallbackHandler() status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if calledLoginID != "user-1001" {
		t.Fatalf("LogoutCallbackHandler() loginID = %q, want user-1001", calledLoginID)
	}
}
