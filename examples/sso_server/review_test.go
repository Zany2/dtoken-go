package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Zany2/dtoken-go/sso"
)

// TestDemoLoginSubmissionBoundary verifies browser origin checks and form-only identity input. TestDemoLoginSubmissionBoundary 验证浏览器来源检查及仅从表单读取身份。
func TestDemoLoginSubmissionBoundary(t *testing.T) {
	server := sso.NewServer()
	defer server.Close()
	handler, err := newDemoHandler(server)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, origin, site, body, wantID string
		status                           int
	}{
		{"query ignored", "", "", "", "user-1001", http.StatusFound},
		{"same origin", "http://localhost:9000", "same-origin", "loginId=alice&back=%2F", "alice", http.StatusFound},
		{"foreign origin", "https://evil.example", "", "loginId=mallory", "", http.StatusForbidden},
		{"foreign site", "", "cross-site", "loginId=mallory", "", http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "http://localhost:9000/login?loginId=query-user&back=%2Fquery-path", strings.NewReader(tc.body))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			if tc.origin != "" {
				request.Header.Set("Origin", tc.origin)
			}
			if tc.site != "" {
				request.Header.Set("Sec-Fetch-Site", tc.site)
			}
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code != tc.status {
				t.Fatalf("status = %d, want %d", recorder.Code, tc.status)
			}
			cookies := recorder.Result().Cookies()
			if tc.wantID == "" {
				if len(cookies) != 0 {
					t.Fatal("rejected submission issued a cookie")
				}
				return
			}
			if len(cookies) != 1 || recorder.Header().Get("Location") != "/" {
				t.Fatalf("unexpected login headers: %v", recorder.Header())
			}
			identityRequest := httptest.NewRequest(http.MethodGet, "/", nil)
			identityRequest.AddCookie(cookies[0])
			if id, ok := sso.LoginIDFromCookie(cookie)(identityRequest); !ok || id != tc.wantID {
				t.Fatalf("cookie identity = %q, %v; want %q", id, ok, tc.wantID)
			}
		})
	}
}

// demoCallbackTransport exercises logout delivery without opening network connections. demoCallbackTransport 无需网络连接即可验证注销投递。
type demoCallbackTransport func(*http.Request) (*http.Response, error)

// RoundTrip delegates the callback request to the test. RoundTrip 将回调请求交给测试处理。
func (f demoCallbackTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// TestDemoLogoutRetriesFailedCallback verifies strict SLO keeps state on failure and clears it on success. TestDemoLogoutRetriesFailedCallback 验证严格 SLO 失败时保留状态，成功时清理状态。
func TestDemoLogoutRetriesFailedCallback(t *testing.T) {
	server := sso.NewServer()
	defer server.Close()
	handler, err := newDemoHandler(server)
	if err != nil {
		t.Fatal(err)
	}
	callback := "http://localhost:9001/sso/logout-callback"
	if _, err := server.RegisterClientSession(context.Background(), "alice", clientID, callback); err != nil {
		t.Fatal(err)
	}
	original := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = original })
	fail := true
	calls := 0
	http.DefaultTransport = demoCallbackTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != http.MethodPost || r.URL.String() != callback {
			t.Errorf("unexpected callback request: %s %s", r.Method, r.URL)
		}
		if err := r.ParseForm(); err != nil || r.PostForm.Get("loginId") != "alice" || r.PostForm.Get("client") != clientID {
			t.Errorf("unexpected callback form: %v, %v", r.PostForm, err)
		}
		if fail {
			return nil, errors.New("demo callback offline")
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"code":0}`))}, nil
	})
	cookieRecorder := httptest.NewRecorder()
	sso.SetLoginIDCookie(cookieRecorder, cookie, "alice")
	loginCookie := cookieRecorder.Result().Cookies()[0]
	for attempt := 0; attempt < 2; attempt++ {
		request := httptest.NewRequest(http.MethodGet, "/sso/logout", nil)
		request.AddCookie(loginCookie)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		sessions, err := server.GetClientSessions(context.Background(), "alice")
		if err != nil {
			t.Fatal(err)
		}
		if fail {
			if recorder.Code < 400 || len(recorder.Result().Cookies()) != 0 || len(sessions) != 1 {
				t.Fatalf("failed callback lost retry state: status=%d sessions=%v", recorder.Code, sessions)
			}
			fail = false
			continue
		}
		cookies := recorder.Result().Cookies()
		if recorder.Code != http.StatusOK || len(sessions) != 0 || len(cookies) != 1 || cookies[0].Name != cookie.Name || cookies[0].Path != cookie.Path || cookies[0].MaxAge != -1 {
			t.Fatalf("successful logout: status=%d sessions=%v cookies=%v", recorder.Code, sessions, cookies)
		}
	}
	if calls != 2 {
		t.Fatalf("callback calls = %d, want 2", calls)
	}
}

// TestDemoRejectsPublicSigningKey verifies the old demo key cannot create a trusted center identity. TestDemoRejectsPublicSigningKey 验证旧示例公开密钥不能创建可信中心身份。
func TestDemoRejectsPublicSigningKey(t *testing.T) {
	publicCookie := cookie
	publicCookie.SecretKey = "demo-cookie-signing-secret"
	recorder := httptest.NewRecorder()
	sso.SetLoginIDCookie(recorder, publicCookie, "forged-user")
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(recorder.Result().Cookies()[0])
	if id, ok := sso.LoginIDFromCookie(cookie)(request); ok || id != "" {
		t.Fatalf("public signing key was trusted: %q, %v", id, ok)
	}
}
