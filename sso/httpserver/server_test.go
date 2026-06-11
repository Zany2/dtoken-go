package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/sso"
)

// TestWrapperDefaultsAndNew verifies httpserver package delegates to SSO HTTP helpers. TestWrapperDefaultsAndNew 验证 httpserver 包转发 SSO HTTP 辅助能力。
func TestWrapperDefaultsAndNew(t *testing.T) {
	opts := DefaultOptions()
	if opts.ServerOptions.Endpoints.Authorize == "" || opts.Cookie.Name == "" {
		t.Fatalf("DefaultOptions() = %+v, want endpoints and cookie defaults", opts)
	}

	server := sso.NewServer()
	httpServer := New(server, Options{})
	if httpServer == nil {
		t.Fatal("New() = nil, want HTTP server")
	}

	mux := http.NewServeMux()
	httpServer.Register(mux)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, opts.ServerOptions.Endpoints.Authorize, nil)
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("registered authorize route status = %d, want method not allowed", recorder.Code)
	}
}

// TestCookieWrappers verifies shared-cookie helper wrappers. TestCookieWrappers 验证共享 Cookie 包装方法。
func TestCookieWrappers(t *testing.T) {
	options := DefaultCookieOptions()
	options.Name = "sso_login"
	options.Path = "/auth"
	options.MaxAge = time.Minute

	recorder := httptest.NewRecorder()
	SetLoginIDCookie(recorder, options, "user-1")
	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("SetLoginIDCookie wrote %d cookies, want 1", len(cookies))
	}
	if cookies[0].Name != "sso_login" || cookies[0].Value != "user-1" || cookies[0].Path != "/auth" {
		t.Fatalf("cookie = %+v, want configured login cookie", cookies[0])
	}

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(cookies[0])
	loginID, ok := LoginIDFromCookie(options)(request)
	if !ok || loginID != "user-1" {
		t.Fatalf("LoginIDFromCookie() = %q, %v, want user-1 true", loginID, ok)
	}

	clearRecorder := httptest.NewRecorder()
	ClearLoginIDCookie(clearRecorder, options)
	clearCookies := clearRecorder.Result().Cookies()
	if len(clearCookies) != 1 || clearCookies[0].MaxAge != -1 {
		t.Fatalf("ClearLoginIDCookie cookies = %+v, want expired cookie", clearCookies)
	}
}
