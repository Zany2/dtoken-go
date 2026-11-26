package sso

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newRemoteAuthTestApp creates an isolated client using a controlled transport. newRemoteAuthTestApp 使用受控传输创建隔离客户端。
func newRemoteAuthTestApp(transport http.RoundTripper) *ClientApp {
	return NewClientApp(ClientConfig{Mode: ModeRemoteSession, ClientID: "app", ServerURL: "https://center.example", HTTPClient: &http.Client{Transport: transport}})
}

// remoteAuthTestRequest provides a bearer request without exposing its credential in the URL. remoteAuthTestRequest 提供 Bearer 请求且不将凭证写入 URL。
func remoteAuthTestRequest() *http.Request {
	r := httptest.NewRequest("GET", "https://app.example/orders", nil)
	r.Header.Set("Authorization", "Bearer opaque-session")
	return r
}

// TestRemoteBearerCredential rejects ambiguous headers and ignores all other credential sources. TestRemoteBearerCredential 拒绝歧义请求头并忽略其他凭证来源。
func TestRemoteBearerCredential(t *testing.T) {
	for _, value := range []string{"Bearer abc-._~+/==", "bearer opaque", "BEARER   opaque"} {
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("Authorization", value)
		if got, err := RemoteBearerCredential(r); err != nil || got == "" {
			t.Fatalf("valid bearer rejected: %v", err)
		}
	}
	for _, values := range [][]string{nil, {""}, {"Basic abc"}, {"Bearer"}, {"Bearer "}, {"Bearer abc "}, {"Bearer abc, Bearer def"}, {"Bearer abc", "Bearer abc"}, {"Bearer\tabc"}, {"Bearer ab\tcd"}, {"Bearer ="}, {"Bearer a=b"}, {"Bearer 中文"}, {"Bearer abc\r\n"}} {
		r := httptest.NewRequest("GET", "/?sessionId=ignored&tokenValue=ignored", nil)
		r.AddCookie(&http.Cookie{Name: "session", Value: "ignored"})
		if values != nil {
			r.Header["Authorization"] = values
		}
		if got, err := RemoteBearerCredential(r); got != "" || !errors.Is(err, ErrRemoteCredential) {
			t.Fatal("invalid bearer or alternate credential source accepted")
		}
	}
	if _, err := RemoteBearerCredential(nil); !errors.Is(err, ErrRemoteCredential) {
		t.Fatal("nil request accepted")
	}
}

// TestRemoteAuthConfiguration validates mode, signing, endpoint, scopes, and owned option copies before serving. TestRemoteAuthConfiguration 在接入前校验模式、签名、端点、范围及选项副本。
func TestRemoteAuthConfiguration(t *testing.T) {
	base := ClientConfig{ClientID: "app", Mode: ModeRemoteSession, ServerURL: "https://center.example"}
	for _, mutate := range []func(*ClientConfig){
		func(c *ClientConfig) { c.ClientID = "" },
		func(c *ClientConfig) { c.Mode = ModeTicket },
		func(c *ClientConfig) { c.Mode = ModeOAuth2 },
		func(c *ClientConfig) { c.ServerURL = "relative" },
		func(c *ClientConfig) { c.Endpoints.Introspect = "https://other.example/introspect" },
		func(c *ClientConfig) { c.CheckSign = true },
		func(c *ClientConfig) { c.Params.Client = "mode" },
		func(c *ClientConfig) { c.ReplayProtection = newReplayTestProtection(t, nil) },
	} {
		cfg := base
		mutate(&cfg)
		if _, err := NewRemoteAuth(NewClientApp(cfg), RemoteAuthOptions{}); err == nil {
			t.Fatal("invalid configuration accepted")
		}
	}
	for _, opts := range []RemoteAuthOptions{{Timeout: -time.Second}, {RequiredScopes: []string{""}}, {RequiredScopes: []string{"read write"}}, {RequiredScopes: []string{"read\n"}}} {
		if _, err := NewRemoteAuth(NewClientApp(base), opts); !errors.Is(err, ErrRemoteAuthConfig) {
			t.Fatalf("invalid options: %v", err)
		}
	}
	if _, err := NewRemoteAuth(nil, RemoteAuthOptions{}); !errors.Is(err, ErrRemoteAuthConfig) {
		t.Fatal("nil client accepted")
	}
	scopes := []string{"read"}
	a, err := NewRemoteAuth(NewClientApp(base), RemoteAuthOptions{RequiredScopes: scopes})
	if err != nil {
		t.Fatal(err)
	}
	scopes[0] = "changed"
	if a.options.RequiredScopes[0] != "read" || a.options.Timeout != 3*time.Second {
		t.Fatal("options were not copied or defaulted")
	}
	if _, err := a.Authenticate(nil); !errors.Is(err, ErrRemoteAuthConfig) {
		t.Fatal("nil request accepted")
	}
	if _, ok := RemoteIdentityFromContext(context.Background()); ok {
		t.Fatal("identity found without authentication")
	}
}

// TestRemoteAuthResponseValidation fails closed on inconsistent identities and keeps scopes literal. TestRemoteAuthResponseValidation 对身份不一致结果拒绝放行，并按字面值检查权限范围。
func TestRemoteAuthResponseValidation(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*CredentialInfo)
		want   error
	}{
		{"valid", func(*CredentialInfo) {}, nil},
		{"unlimited", func(i *CredentialInfo) { i.ExpiresIn = -1 }, nil},
		{"large timeout", func(i *CredentialInfo) { i.ExpiresIn = 1<<63 - 1 }, nil},
		{"inactive", func(i *CredentialInfo) { i.Active = false }, ErrRemoteCredential},
		{"expired", func(i *CredentialInfo) { i.ExpiresIn = 0 }, ErrRemoteCredential},
		{"invalid ttl", func(i *CredentialInfo) { i.ExpiresIn = -2 }, ErrRemoteUnavailable},
		{"empty subject", func(i *CredentialInfo) { i.LoginID = "" }, ErrRemoteUnavailable},
		{"other client", func(i *CredentialInfo) { i.ClientID = "other" }, ErrRemoteUnavailable},
		{"other mode", func(i *CredentialInfo) { i.Mode = ModeTicket }, ErrRemoteUnavailable},
		{"missing scope", func(i *CredentialInfo) { i.Scopes = []string{"read"} }, ErrRemoteScope},
		{"case sensitive", func(i *CredentialInfo) { i.Scopes = []string{"Read", "write"} }, ErrRemoteScope},
		{"no wildcard", func(i *CredentialInfo) { i.Scopes = []string{"*"} }, ErrRemoteScope},
	} {
		t.Run(test.name, func(t *testing.T) {
			info := CredentialInfo{Active: true, Mode: ModeRemoteSession, ClientID: "app", LoginID: "alice", ExpiresIn: 60, Scopes: []string{"read", "write"}}
			test.mutate(&info)
			app := newRemoteAuthTestApp(logoutReviewTransport(func(r *http.Request) (*http.Response, error) {
				if err := r.ParseForm(); err != nil {
					t.Fatal(err)
				}
				if r.Method != "POST" || r.URL.Path != "/sso/introspect" || r.URL.RawQuery != "" || r.PostForm.Get("sessionId") != "opaque-session" || r.PostForm.Get("mode") != string(ModeRemoteSession) || r.Header.Get("Authorization") != "" {
					t.Fatal("unexpected protocol request or forwarded application authorization")
				}
				out := httptest.NewRecorder()
				writeJSON(out, http.StatusOK, OKResponse(info))
				return out.Result(), nil
			}))
			a, err := NewRemoteAuth(app, RemoteAuthOptions{RequiredScopes: []string{"read", "write"}})
			if err != nil {
				t.Fatal(err)
			}
			got, err := a.Authenticate(remoteAuthTestRequest())
			if !errors.Is(err, test.want) || (test.want != nil && got != nil) {
				t.Fatalf("Authenticate error = %v, want %v", err, test.want)
			}
			if test.want == nil && (got == nil || got.LoginID != "alice") {
				t.Fatal("successful validation lost identity")
			}
		})
	}
}

// TestRemoteAuthErrorsAndContext checks bounded errors, resolver failures, handler isolation, and business context lifetime. TestRemoteAuthErrorsAndContext 检查固定错误、解析故障、处理器隔离与业务上下文寿命。
func TestRemoteAuthErrorsAndContext(t *testing.T) {
	private := "private-upstream-credential"
	for _, test := range []struct {
		name     string
		resolver RemoteCredentialResolver
		body     string
		status   int
		want     int
	}{
		{"missing", func(*http.Request) (string, error) { return "", nil }, "", 200, 401},
		{"invalid", func(*http.Request) (string, error) { return "", fmt.Errorf("%w: %s", ErrRemoteCredential, private) }, "", 200, 401},
		{"store outage", func(*http.Request) (string, error) { return "", errors.New(private) }, "", 200, 503},
		{"upstream error", nil, private, 500, 503},
		{"center client auth", nil, private, 401, 503},
		{"malformed response", nil, "broken " + private, 200, 503},
		{"inactive", nil, `{"code":0,"data":{"active":false}}`, 200, 401},
		{"scopes", nil, `{"code":0,"data":{"active":true,"loginId":"alice","clientId":"app","mode":"remote_session","expiresIn":60}}`, 200, 403},
	} {
		t.Run(test.name, func(t *testing.T) {
			requests, calls := 0, 0
			app := newRemoteAuthTestApp(logoutReviewTransport(func(*http.Request) (*http.Response, error) {
				requests++
				out := httptest.NewRecorder()
				out.WriteHeader(test.status)
				_, _ = out.WriteString(test.body)
				return out.Result(), nil
			}))
			a, err := NewRemoteAuth(app, RemoteAuthOptions{ResolveCredential: test.resolver, RequiredScopes: []string{"read"}})
			if err != nil {
				t.Fatal(err)
			}
			out := httptest.NewRecorder()
			a.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ })).ServeHTTP(out, remoteAuthTestRequest())
			if out.Code != test.want || calls != 0 || strings.Contains(out.Body.String(), private) || out.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("rejection: status %d, calls %d, body %s", out.Code, calls, out.Body.String())
			}
			if test.resolver != nil && requests != 0 {
				t.Fatal("resolver failure contacted center")
			}
		})
	}
	app := newRemoteAuthTestApp(logoutReviewTransport(func(*http.Request) (*http.Response, error) {
		out := httptest.NewRecorder()
		writeJSON(out, 200, OKResponse(CredentialInfo{Active: true, ClientID: "app", Mode: ModeRemoteSession, LoginID: "alice", ExpiresIn: 60}))
		return out.Result(), nil
	}))
	a, err := NewRemoteAuth(app, RemoteAuthOptions{})
	if err != nil {
		t.Fatal(err)
	}
	called := false
	a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		identity, ok := RemoteIdentityFromContext(r.Context())
		if !ok || identity.LoginID != "alice" || r.Context().Err() != nil {
			t.Fatal("validated identity missing or business context canceled")
		}
		if _, exists := r.Context().Deadline(); exists {
			t.Fatal("validation timeout leaked into business request")
		}
	})).ServeHTTP(httptest.NewRecorder(), remoteAuthTestRequest())
	if !called {
		t.Fatal("valid identity did not reach business handler")
	}
	observed := false
	a.options.OnError = func(w http.ResponseWriter, r *http.Request, err error) {
		observed = errors.Is(err, ErrRemoteCredential)
		w.WriteHeader(http.StatusTeapot)
	}
	out := httptest.NewRecorder()
	a.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("custom error allowed access") })).ServeHTTP(out, httptest.NewRequest("GET", "/", nil))
	if !observed || out.Code != http.StatusTeapot {
		t.Fatal("custom rejection handler not called")
	}
}

// TestRemoteAuthStoredSession resolves private credentials without an Authorization header and keeps the caller's context. TestRemoteAuthStoredSession 在没有 Authorization 头时解析私有凭证，并保留调用方上下文。
func TestRemoteAuthStoredSession(t *testing.T) {
	var resolutionContext context.Context
	app := newRemoteAuthTestApp(logoutReviewTransport(func(r *http.Request) (*http.Response, error) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.PostForm.Get("sessionId") != "stored-center-session" || len(r.Cookies()) != 0 {
			t.Fatal("local session cookie was forwarded or resolved credential was lost")
		}
		out := httptest.NewRecorder()
		writeJSON(out, 200, OKResponse(CredentialInfo{Active: true, ClientID: "app", Mode: ModeRemoteSession, LoginID: "alice", ExpiresIn: 60}))
		return out.Result(), nil
	}))
	a, err := NewRemoteAuth(app, RemoteAuthOptions{ResolveCredential: func(r *http.Request) (string, error) {
		resolutionContext = r.Context()
		if _, ok := resolutionContext.Deadline(); !ok {
			t.Fatal("resolver has no validation deadline")
		}
		cookie, err := r.Cookie("application-session")
		if err != nil || cookie.Value != "local-id" {
			return "", ErrRemoteCredential
		}
		return "stored-center-session", nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	r := httptest.NewRequest("GET", "/orders", nil).WithContext(ctx)
	r.AddCookie(&http.Cookie{Name: "application-session", Value: "local-id"})
	identity, err := a.Authenticate(r)
	if err != nil || identity.LoginID != "alice" || ctx.Err() != nil || resolutionContext.Err() == nil {
		t.Fatalf("stored credential resolution failed: %v", err)
	}
	if _, ok := RemoteIdentityFromContext(r.Context()); ok {
		t.Fatal("direct authentication unexpectedly mutated caller context")
	}
}
