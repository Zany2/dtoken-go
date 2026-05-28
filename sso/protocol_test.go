package sso

import (
	"net/url"
	"strings"
	"testing"
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
