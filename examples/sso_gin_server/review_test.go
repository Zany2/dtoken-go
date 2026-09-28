package main

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Zany2/dtoken-go/sso"
	"github.com/gin-gonic/gin"
)

// TestGinLoginInputBoundary verifies malformed and cross-origin forms never issue identity Cookies. TestGinLoginInputBoundary 验证损坏表单和跨源提交不能签发身份 Cookie。
func TestGinLoginInputBoundary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := sso.NewServer()
	defer server.Close()
	router, err := newDemoRouter(server)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, body, contentType, origin, site, wantID string
		status                                        int
	}{
		{"bad escape", "loginId=%zz&back=%2F", "application/x-www-form-urlencoded", "", "", "", http.StatusBadRequest},
		{"partial form", "loginId=alice&back=%zz", "application/x-www-form-urlencoded", "", "", "", http.StatusBadRequest},
		{"broken multipart", "broken", "multipart/form-data; boundary=demo", "", "", "", http.StatusBadRequest},
		{"missing boundary", "broken", "multipart/form-data", "", "", "", http.StatusBadRequest},
		{"foreign origin", "loginId=mallory", "application/x-www-form-urlencoded", "https://evil.example", "", "", http.StatusForbidden},
		{"foreign site", "loginId=mallory", "application/x-www-form-urlencoded", "", "cross-site", "", http.StatusForbidden},
		{"same origin", "loginId=alice&back=%2F", "application/x-www-form-urlencoded", "http://localhost:9100", "same-origin", "alice", http.StatusFound},
		{"query ignored", "", "application/x-www-form-urlencoded", "", "", "user-1001", http.StatusFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "http://localhost:9100/login?loginId=query-user&back=%2Fquery-path", strings.NewReader(tc.body))
			request.Header.Set("Content-Type", tc.contentType)
			if tc.origin != "" {
				request.Header.Set("Origin", tc.origin)
			}
			if tc.site != "" {
				request.Header.Set("Sec-Fetch-Site", tc.site)
			}
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			if recorder.Code != tc.status {
				t.Fatalf("status = %d, want %d", recorder.Code, tc.status)
			}
			cookies := recorder.Result().Cookies()
			if tc.wantID == "" {
				if len(cookies) != 0 {
					t.Fatal("rejected form issued a Cookie")
				}
				return
			}
			if len(cookies) != 1 || recorder.Header().Get("Location") != "/" {
				t.Fatalf("login headers = %v", recorder.Header())
			}
			check := httptest.NewRequest(http.MethodGet, "/", nil)
			check.AddCookie(cookies[0])
			if id, ok := sso.LoginIDFromCookie(cookie)(check); !ok || id != tc.wantID {
				t.Fatalf("identity = %q, %v; want %q", id, ok, tc.wantID)
			}
		})
	}

	// Preserve successful multipart submissions while checking their parsing errors. 检查解析错误的同时保留合法 multipart 提交。
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("loginId", "multipart-user"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/login", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusFound || len(recorder.Result().Cookies()) != 1 {
		t.Fatalf("multipart login status = %d", recorder.Code)
	}
	check := httptest.NewRequest(http.MethodGet, "/", nil)
	check.AddCookie(recorder.Result().Cookies()[0])
	if id, ok := sso.LoginIDFromCookie(cookie)(check); !ok || id != "multipart-user" {
		t.Fatalf("multipart identity = %q, %v", id, ok)
	}
	page := performGinRequest(router, http.MethodGet, "/login", "")
	if page.Header().Get("Content-Type") != "text/html; charset=utf-8" || page.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("login page headers = %v", page.Header())
	}
}

// ginReviewTransport tests callbacks without opening network connections. ginReviewTransport 无需网络连接即可测试回调。
type ginReviewTransport func(*http.Request) (*http.Response, error)

// RoundTrip delegates callback delivery to the test. RoundTrip 将回调投递交给测试。
func (f ginReviewTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// TestGinBestEffortLogout verifies trusted identity checks and cleanup despite callback failure. TestGinBestEffortLogout 验证可信身份检查及回调失败后的清理策略。
func TestGinBestEffortLogout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := sso.NewServer()
	defer server.Close()
	router, err := newDemoRouter(server)
	if err != nil {
		t.Fatal(err)
	}
	callback := "http://localhost:9101/sso/logout-callback"
	if _, err := server.RegisterClientSession(context.Background(), "alice", clientID, callback); err != nil {
		t.Fatal(err)
	}
	original := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = original })
	calls := 0
	http.DefaultTransport = ginReviewTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != http.MethodPost || r.URL.String() != callback {
			t.Errorf("unexpected callback: %s %s", r.Method, r.URL)
		}
		if err := r.ParseForm(); err != nil || r.PostForm.Get("loginId") != "alice" || r.PostForm.Get("client") != clientID {
			t.Errorf("unexpected callback form: %v, %v", r.PostForm, err)
		}
		return nil, errors.New("client offline")
	})
	login := performGinRequest(router, http.MethodPost, "/login", "loginId=alice")
	cookies := login.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("login cookies = %v", cookies)
	}
	for _, tc := range []struct {
		path          string
		authenticated bool
	}{
		{"/sso/logout", false},
		{"/sso/logout?loginId=someone-else", true},
	} {
		request := httptest.NewRequest(http.MethodGet, tc.path, nil)
		if tc.authenticated {
			request.AddCookie(cookies[0])
		}
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusUnauthorized || calls != 0 || len(recorder.Result().Cookies()) != 0 {
			t.Fatalf("untrusted logout: status=%d calls=%d", recorder.Code, calls)
		}
	}
	request := httptest.NewRequest(http.MethodPost, "/sso/logout", nil)
	request.AddCookie(cookies[0])
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	sessions, err := server.GetClientSessions(context.Background(), "alice")
	if err != nil {
		t.Fatal(err)
	}
	cleared := recorder.Result().Cookies()
	if recorder.Code != http.StatusOK || calls != 1 || len(sessions) != 0 || len(cleared) != 1 || cleared[0].Name != cookie.Name || cleared[0].Path != cookie.Path || cleared[0].MaxAge != -1 {
		t.Fatalf("best-effort logout: status=%d calls=%d sessions=%v cookies=%v", recorder.Code, calls, sessions, cleared)
	}
}

// TestGinRejectsPublicCookieKey verifies a published demo key no longer authenticates users. TestGinRejectsPublicCookieKey 验证公开示例密钥不能继续认证用户。
func TestGinRejectsPublicCookieKey(t *testing.T) {
	publicCookie := cookie
	publicCookie.SecretKey = "gin-demo-cookie-signing-secret"
	recorder := httptest.NewRecorder()
	sso.SetLoginIDCookie(recorder, publicCookie, "forged-user")
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(recorder.Result().Cookies()[0])
	if id, ok := sso.LoginIDFromCookie(cookie)(request); ok || id != "" {
		t.Fatalf("public key authenticated %q, %v", id, ok)
	}
}
