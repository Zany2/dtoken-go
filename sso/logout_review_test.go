package sso

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// logoutReviewTransport handles callbacks without opening network connections. logoutReviewTransport 无需网络连接即可处理回调。
type logoutReviewTransport func(*http.Request) (*http.Response, error)

// RoundTrip delegates to the test callback. RoundTrip 转发到测试回调。
func (f logoutReviewTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// TestLogoutClearsSessionsAccordingToCallbackResult verifies strict and best-effort completion. TestLogoutClearsSessionsAccordingToCallbackResult 验证严格模式和尽力模式的完成行为。
func TestLogoutClearsSessionsAccordingToCallbackResult(t *testing.T) {
	for _, tc := range []struct {
		name       string
		status     int
		bestEffort bool
		cleared    bool
	}{
		{"strict success", http.StatusOK, false, true},
		{"strict failure", http.StatusServiceUnavailable, false, false},
		{"best effort failure", http.StatusServiceUnavailable, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			s := NewServer()
			defer s.Close()
			client := newTestClient()
			if err := s.RegisterClient(client); err != nil {
				t.Fatal(err)
			}
			if _, err := s.RegisterClientSession(ctx, "user", client.ClientID, client.RedirectURIs[0]); err != nil {
				t.Fatal(err)
			}
			var calls atomic.Int32
			app := NewClientApp(ClientConfig{ClientID: client.ClientID, CheckSign: true, SecretKey: "sign-secret"})
			transport := logoutReviewTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if _, ok := r.Context().Deadline(); !ok {
					t.Error("callback lacks a timeout")
				}
				if callback, err := app.VerifyLogoutCallback(r); err != nil || callback.LoginID != "user" {
					t.Errorf("callback verification = %+v, %v", callback, err)
				}
				return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(""))}, nil
			})
			h := NewHTTPServer(s, HTTPOptions{
				ServerOptions:   ServerOptions{EnableSLO: true, CheckSign: true, SecretKey: "sign-secret", LogoutCallbackBestEffort: tc.bestEffort, LogoutHTTPClient: &http.Client{Transport: transport}},
				LoginIDResolver: func(*http.Request) (string, bool) { return "user", true },
			})
			response := protocolReviewPost(h.HandleLogout, NewSigner("sign-secret").AttachSign(nil))
			wantStatus := http.StatusInternalServerError
			if tc.cleared {
				wantStatus = http.StatusOK
			}
			if response.Code != wantStatus || calls.Load() != 1 {
				t.Fatalf("logout status = %d, callback calls = %d", response.Code, calls.Load())
			}
			sessions, err := s.GetClientSessions(ctx, "user")
			if err != nil || (len(sessions) == 0) != tc.cleared {
				t.Fatalf("sessions = %+v, %v; cleared = %v", sessions, err, tc.cleared)
			}
			cookies := response.Result().Cookies()
			if tc.cleared && (len(cookies) != 1 || cookies[0].MaxAge != -1) {
				t.Fatal("successful logout did not clear the cookie")
			}
			if !tc.cleared && len(cookies) != 0 {
				t.Fatal("failed strict logout cleared the cookie")
			}
		})
	}
}

// TestLogoutRechecksCallbackRegistration verifies stale targets are never contacted. TestLogoutRechecksCallbackRegistration 验证不会访问已失效的回调地址。
func TestLogoutRechecksCallbackRegistration(t *testing.T) {
	for _, removed := range []bool{false, true} {
		s := NewServer()
		client := newTestClient()
		client.AllowOrigins = []string{"https://callback.example.com"}
		if err := s.RegisterClient(client); err != nil {
			t.Fatal(err)
		}
		session, err := s.RegisterClientSession(context.Background(), "user", client.ClientID, "https://callback.example.com/logout")
		if err != nil {
			t.Fatal(err)
		}
		if removed {
			err = s.UnregisterClient(client.ClientID)
		} else {
			client.AllowOrigins = nil
			err = s.RegisterClient(client)
		}
		if err != nil {
			t.Fatal(err)
		}
		var calls atomic.Int32
		h := NewHTTPServer(s, HTTPOptions{ServerOptions: ServerOptions{LogoutHTTPClient: &http.Client{Transport: logoutReviewTransport(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return nil, errors.New("unexpected callback request")
		})}}})
		err = h.postLogoutCallback(httptest.NewRequest(http.MethodPost, "/logout", nil), *session)
		if removed && err != nil || !removed && !errors.Is(err, ErrInvalidCallbackURL) || calls.Load() != 0 {
			t.Fatalf("removed = %v, callback error = %v, calls = %d", removed, err, calls.Load())
		}
		_ = s.Close()
	}
}

// TestLogoutCallbackRequiresHTTPURL verifies exact registration cannot bypass URL constraints. TestLogoutCallbackRequiresHTTPURL 验证精确注册也不能绕过 URL 约束。
func TestLogoutCallbackRequiresHTTPURL(t *testing.T) {
	s := NewServer()
	defer s.Close()
	for _, target := range []string{"/callback", "ftp://app.example.com/callback", "https://user:password@app.example.com/callback", "https://app.example.com/callback#fragment", "http://:80/callback"} {
		client := &Client{RedirectURIs: []string{target}}
		if s.isValidLogoutCallbackURL(client, target) {
			t.Fatalf("invalid callback accepted: %s", target)
		}
	}
}

// TestSharedCookieEnforcesSignedDeadline verifies server-side expiry and legacy rejection. TestSharedCookieEnforcesSignedDeadline 验证服务端检查签名有效期并拒绝旧格式。
func TestSharedCookieEnforcesSignedDeadline(t *testing.T) {
	options := CookieOptions{Name: "login", SecretKey: "cookie-secret"}
	resolve := LoginIDFromCookie(options)
	legacyPayload := base64.RawURLEncoding.EncodeToString([]byte("user"))
	legacy := legacyPayload + "." + NewSigner(options.SecretKey).Sign(url.Values{"value": {legacyPayload}})
	for _, tc := range []struct {
		name, value string
		valid       bool
	}{
		{"valid", encodeLoginIDCookie("user", options.SecretKey, time.Now().Add(time.Hour)), true},
		{"expired", encodeLoginIDCookie("user", options.SecretKey, time.Now().Add(-time.Hour)), false},
		{"legacy", legacy, false},
		{"wrong key", encodeLoginIDCookie("user", "different-key", time.Now().Add(time.Hour)), false},
	} {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.AddCookie(&http.Cookie{Name: options.Name, Value: tc.value})
		id, ok := resolve(request)
		if ok != tc.valid || ok && id != "user" || !ok && id != "" {
			t.Fatalf("%s resolved = %q, %v", tc.name, id, ok)
		}
	}
}

// TestCookieLifetimeAndDeletionAttributes verifies bounded MaxAge and matching deletion scope. TestCookieLifetimeAndDeletionAttributes 验证 MaxAge 边界和一致的删除作用域。
func TestCookieLifetimeAndDeletionAttributes(t *testing.T) {
	for _, lifetime := range []time.Duration{500 * time.Millisecond, 1500 * time.Millisecond, time.Minute, time.Duration(1<<63 - 1)} {
		options := CookieOptions{Name: "login", Domain: "example.com", Path: "/auth", MaxAge: lifetime, Secure: true, HTTPOnly: true, SameSite: http.SameSiteStrictMode, SecretKey: "cookie-secret"}
		recorder := httptest.NewRecorder()
		SetLoginIDCookie(recorder, options, "user")
		cookies := recorder.Result().Cookies()
		if len(cookies) != 1 || cookies[0].MaxAge <= 0 || cookies[0].Expires.IsZero() {
			t.Fatalf("lifetime %v: cookies = %+v", lifetime, cookies)
		}
		if lifetime == 500*time.Millisecond && cookies[0].MaxAge != 1 || lifetime == 1500*time.Millisecond && cookies[0].MaxAge != 2 {
			t.Fatalf("lifetime %v: MaxAge = %d", lifetime, cookies[0].MaxAge)
		}
		cleared := httptest.NewRecorder()
		ClearLoginIDCookie(cleared, options)
		deleted := cleared.Result().Cookies()
		if len(deleted) != 1 {
			t.Fatal("missing deletion cookie")
		}
		got, original := deleted[0], cookies[0]
		if got.Name != original.Name || got.Path != original.Path || got.Domain != original.Domain || got.Secure != original.Secure || got.HttpOnly != original.HttpOnly || got.SameSite != original.SameSite || got.MaxAge != -1 || !got.Expires.Before(time.Now()) {
			t.Fatalf("incorrect deletion attributes: %+v", got)
		}
	}
}
