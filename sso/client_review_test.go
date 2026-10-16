package sso

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// TestClientResponsePreservesNumbers verifies typed and extension integers survive decoding. TestClientResponsePreservesNumbers 验证类型字段和扩展字段的整数解码不丢失精度。
func TestClientResponsePreservesNumbers(t *testing.T) {
	app := NewClientApp(ClientConfig{
		ServerURL: "https://sso.example.com",
		HTTPClient: &http.Client{Transport: logoutReviewTransport(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
				`{"code":0,"data":{"loginId":"user","remainTokenTimeout":9007199254740993,"remainSessionTimeout":9223372036854775807,"extra":{"id":9007199254740993,"nested":{"id":9223372036854775807}}}}`,
			))}, nil
		})},
	})
	result, err := app.ExchangeTicket(nil, "ticket", "")
	if err != nil {
		t.Fatal(err)
	}
	if result.RemainTokenTimeout != 9007199254740993 || result.RemainSessionTimeout != 9223372036854775807 {
		t.Fatalf("integer fields lost precision: %+v", result)
	}
	if result.Extra["id"] != json.Number("9007199254740993") {
		t.Fatalf("extension integer lost precision: %v", result.Extra)
	}
	nested, ok := result.Extra["nested"].(map[string]any)
	if !ok || nested["id"] != json.Number("9223372036854775807") {
		t.Fatalf("nested integer lost precision: %v", result.Extra)
	}
}

// TestClientRejectsMalformedResponses verifies invalid successes cannot yield an empty identity. TestClientRejectsMalformedResponses 验证无效成功响应不能返回空身份。
func TestClientRejectsMalformedResponses(t *testing.T) {
	for _, payload := range []string{
		`null`, `{}`, `{"data":{"loginId":"user"}}`, `{"code":null,"data":{"loginId":"user"}}`,
		`{"code":0}`, `{"code":0,"data":null}`, `{"code":0,"data":{}}`,
		`{"code":0,"data":[]}`, `{"code":0,"data":{"loginId":123}}`,
		`{"code":1}`, `{"code":"0","data":{"loginId":"user"}}`,
		`{"code":0,"data":{"loginId":"user"}} {}`, `not-json`,
	} {
		t.Run(payload, func(t *testing.T) {
			app := NewClientApp(ClientConfig{
				ServerURL: "https://sso.example.com",
				HTTPClient: &http.Client{Transport: logoutReviewTransport(func(r *http.Request) (*http.Response, error) {
					return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(payload))}, nil
				})},
			})
			if result, err := app.ExchangeTicket(nil, "ticket", ""); err == nil || result != nil {
				t.Fatalf("ExchangeTicket() = %+v, %v; want rejection", result, err)
			}
			if result, err := app.UserInfo(nil, CredentialRequest{}); err == nil || result != nil {
				t.Fatalf("UserInfo() = %+v, %v; want rejection", result, err)
			}
		})
	}

	// Revocation needs a valid envelope but no identity payload. 撤销需要有效响应封装，但不要求身份数据。
	payload := `{"code":0}`
	app := NewClientApp(ClientConfig{
		ServerURL: "https://sso.example.com",
		HTTPClient: &http.Client{Transport: logoutReviewTransport(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(payload))}, nil
		})},
	})
	if err := app.Revoke(nil, CredentialRequest{}); err != nil {
		t.Fatal(err)
	}
	payload = `{}`
	if err := app.Revoke(nil, CredentialRequest{}); err == nil {
		t.Fatal("Revoke accepted an envelope without code")
	}
	payload = `{"code":0,"data":{"active":false}}`
	if info, err := app.Introspect(nil, CredentialRequest{}); err != nil || info.Active {
		t.Fatalf("inactive introspection = %+v, %v", info, err)
	}
}

// TestClientBlocksImplicitRedirects verifies POST credentials stay at the configured endpoint. TestClientBlocksImplicitRedirects 验证 POST 凭证不会隐式转发到重定向地址。
func TestClientBlocksImplicitRedirects(t *testing.T) {
	for _, status := range []int{301, 302, 303, 307, 308} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			requests := 0
			client := &http.Client{Transport: logoutReviewTransport(func(r *http.Request) (*http.Response, error) {
				requests++
				if requests > 1 {
					t.Fatal("followed a redirect carrying a credential request")
				}
				if err := r.ParseForm(); err != nil {
					t.Fatal(err)
				}
				if r.PostForm.Get("clientSecret") != "secret" || r.PostForm.Get("ticket") != "ticket" {
					t.Fatalf("missing POST credentials: %v", r.PostForm)
				}
				return &http.Response{
					StatusCode: status,
					Header:     http.Header{"Location": {"https://other.example.com/token"}},
					Body:       io.NopCloser(strings.NewReader("redirect")),
				}, nil
			})}
			app := NewClientApp(ClientConfig{ServerURL: "https://sso.example.com", ClientSecret: "secret", HTTPClient: client})
			if _, err := app.ExchangeTicket(nil, "ticket", ""); err == nil {
				t.Fatal("redirect response was accepted")
			}
			if requests != 1 || client.CheckRedirect != nil || client.Timeout != 0 {
				t.Fatal("request count or caller's HTTP client was changed")
			}
		})
	}
}

// TestClientHonorsExplicitHTTPPolicy verifies timeout, cancellation, and redirect overrides. TestClientHonorsExplicitHTTPPolicy 验证自定义超时、取消和重定向策略。
func TestClientHonorsExplicitHTTPPolicy(t *testing.T) {
	policyErr := errors.New("redirect policy rejected request")
	policyCalls := 0
	client := &http.Client{
		Timeout: 2 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			policyCalls++
			return policyErr
		},
		Transport: logoutReviewTransport(func(r *http.Request) (*http.Response, error) {
			if err := r.Context().Err(); err != nil {
				return nil, err
			}
			deadline, ok := r.Context().Deadline()
			if !ok || time.Until(deadline) > 2*time.Second {
				t.Fatal("custom timeout was not applied")
			}
			return &http.Response{
				StatusCode: http.StatusTemporaryRedirect,
				Header:     http.Header{"Location": {"https://other.example.com/token"}},
				Body:       io.NopCloser(strings.NewReader("redirect")),
			}, nil
		}),
	}
	app := NewClientApp(ClientConfig{ServerURL: "https://sso.example.com", HTTPClient: client})
	if _, err := app.ExchangeTicket(nil, "ticket", ""); !errors.Is(err, policyErr) || policyCalls != 1 {
		t.Fatalf("explicit redirect policy: calls=%d, error=%v", policyCalls, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := app.ExchangeTicket(ctx, "ticket", ""); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled request error = %v", err)
	}
}

// TestClientDefaultTimeout verifies default requests have a finite deadline without waiting. TestClientDefaultTimeout 验证默认请求具有有限截止时间，无需实际等待。
func TestClientDefaultTimeout(t *testing.T) {
	original := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = original })
	http.DefaultTransport = logoutReviewTransport(func(r *http.Request) (*http.Response, error) {
		deadline, ok := r.Context().Deadline()
		if !ok || time.Until(deadline) <= 0 || time.Until(deadline) > 10*time.Second {
			t.Fatalf("unexpected default request deadline: %v, %v", deadline, ok)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"code":0}`))}, nil
	})
	app := NewClientApp(ClientConfig{ServerURL: "https://sso.example.com"})
	if err := app.Revoke(nil, CredentialRequest{}); err != nil {
		t.Fatal(err)
	}
}

// TestClientURLValidation verifies shared URL rules, signatures, and escaped gateway prefixes. TestClientURLValidation 验证统一 URL 规则、签名及转义网关前缀。
func TestClientURLValidation(t *testing.T) {
	app := NewClientApp(ClientConfig{
		ServerURL: "https://sso.example.com/gateway%2F/", ClientID: "app", ClientSecret: "secret",
		CheckSign: true, SecretKey: "sign-secret",
		HTTPClient: &http.Client{Transport: logoutReviewTransport(func(r *http.Request) (*http.Response, error) {
			if r.URL.EscapedPath() != "/gateway%2F/sso/token" || r.URL.RawQuery != "" {
				t.Fatalf("POST URL = %s", r.URL)
			}
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			if !NewSigner("sign-secret").Verify(r.PostForm) || r.PostForm.Get("clientSecret") != "secret" {
				t.Fatal("invalid POST credentials or signature")
			}
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"code":0,"data":{"loginId":"user"}}`))}, nil
		})},
	})
	extra := url.Values{"tag": {"first", "second"}}
	address, err := app.AuthURL("https://app.example.com/callback", extra)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(address)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.EscapedPath() != "/gateway%2F/sso/authorize" || !NewSigner("sign-secret").Verify(parsed.Query()) || parsed.Query().Get("clientSecret") != "" {
		t.Fatalf("authorization URL = %s", address)
	}
	if len(extra) != 1 || extra.Get("tag") != "first" {
		t.Fatal("extra parameters were mutated")
	}
	if _, err = app.ExchangeTicket(nil, "ticket", ""); err != nil {
		t.Fatal(err)
	}

	for _, base := range []string{"", "/gateway", "//sso.example.com", "ftp://sso.example.com", "https://", "https://user:pass@sso.example.com", "https://sso.example.com?tenant=a", "https://sso.example.com?", "https://sso.example.com#fragment", "https://sso.example.com#"} {
		invalid := NewClientApp(ClientConfig{ServerURL: base})
		if _, err := invalid.AuthURL("", nil); err == nil {
			t.Fatalf("accepted invalid base URL %q", base)
		}
		if _, err := invalid.ExchangeTicket(nil, "ticket", ""); err == nil {
			t.Fatalf("POST accepted invalid base URL %q", base)
		}
	}
	for _, path := range []string{"https://other.example.com/token", "//other.example.com/token", "/token?client=other", "/token#fragment"} {
		if _, err := joinURL("https://sso.example.com/gateway", path); err == nil {
			t.Fatalf("accepted invalid endpoint %q", path)
		}
	}
}

// TestClientUserInfoMatchesServer verifies the real handler returns the credential state. TestClientUserInfoMatchesServer 验证真实处理器返回一致的凭证状态。
func TestClientUserInfoMatchesServer(t *testing.T) {
	s := NewServer()
	defer s.Close()
	client := newTestClient()
	client.Modes = []Mode{ModeSharedToken}
	if err := s.RegisterClient(client); err != nil {
		t.Fatal(err)
	}
	token, err := s.GenerateSharedToken(context.Background(), client.ClientID, "user", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	options := DefaultHTTPOptions()
	options.ServerOptions.SecretKey = "sign-secret"
	handler := NewHTTPServer(s, options).Handler()
	app := NewClientApp(ClientConfig{
		Mode: ModeSharedToken, ServerURL: "https://sso.example.com", ClientID: client.ClientID,
		ClientSecret: client.ClientSecret, CheckSign: true, SecretKey: "sign-secret",
		HTTPClient: &http.Client{Transport: logoutReviewTransport(func(r *http.Request) (*http.Response, error) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, r)
			return recorder.Result(), nil
		})},
	})
	info, err := app.UserInfo(nil, CredentialRequest{TokenValue: token.Token})
	if err != nil {
		t.Fatal(err)
	}
	if !info.Active || info.Mode != ModeSharedToken || info.LoginID != "user" || info.ClientID != client.ClientID || info.ExpiresIn <= 0 {
		t.Fatalf("UserInfo() lost credential fields: %+v", info)
	}
}
