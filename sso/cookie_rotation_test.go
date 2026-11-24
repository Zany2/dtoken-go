package sso

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// rotationCookie issues a cookie through the public writer without exposing its value. rotationCookie 通过公开写入器签发 Cookie，不暴露其值。
func rotationCookie(t *testing.T, options CookieOptions) *http.Cookie {
	t.Helper()
	w := httptest.NewRecorder()
	SetLoginIDCookie(w, options, "alice")
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || cookies[0].MaxAge < 0 {
		t.Fatal("cookie was not issued")
	}
	return cookies[0]
}

// TestCookieRotationLifecycle verifies preparation, primary-only issuance, cutover and retirement. TestCookieRotationLifecycle 验证准备、仅主密钥签发、切换与退役。
func TestCookieRotationLifecycle(t *testing.T) {
	old := CookieOptions{SecretKey: "old-key"}
	prepare := CookieOptions{SecretKey: "old-key", VerificationKey: "new-key"}
	cutover := CookieOptions{SecretKey: "new-key", VerificationKey: "old-key"}
	retired := CookieOptions{SecretKey: "new-key"}
	oldCookie, newCookie := rotationCookie(t, prepare), rotationCookie(t, cutover)
	for _, tc := range []struct {
		name    string
		options CookieOptions
		oldOK   bool
		newOK   bool
	}{
		{"old only", old, true, false},
		{"prepare", prepare, true, true},
		{"cutover", cutover, true, true},
		{"retired", retired, false, true},
		{"duplicate keys", CookieOptions{SecretKey: "new-key", VerificationKey: "new-key"}, false, true},
		{"other application", CookieOptions{SecretKey: "other-key"}, false, false},
		{"verification only", CookieOptions{VerificationKey: "old-key"}, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for i, cookie := range []*http.Cookie{oldCookie, newCookie} {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.AddCookie(cookie)
				id, ok := LoginIDFromCookie(tc.options)(r)
				if ok != []bool{tc.oldOK, tc.newOK}[i] || (ok && id != "alice") {
					t.Fatalf("cookie %d accepted=%v", i, ok)
				}
			}
		})
	}
	w := httptest.NewRecorder()
	SetLoginIDCookie(w, CookieOptions{VerificationKey: "old-key"}, "alice")
	if cookies := w.Result().Cookies(); len(cookies) != 1 || cookies[0].MaxAge >= 0 {
		t.Fatal("verification-only key issued a cookie")
	}
}

// TestCookieRotationExpiryAndTampering preserves signed deadlines independently of the overlap window. TestCookieRotationExpiryAndTampering 独立于密钥过渡窗口保留签名截止时间。
func TestCookieRotationExpiryAndTampering(t *testing.T) {
	for _, key := range []string{"old-key", "new-key"} {
		valid := encodeLoginIDCookie("alice", key, time.Now().Add(time.Hour))
		for _, value := range []string{
			encodeLoginIDCookie("alice", key, time.Now().Add(-time.Second)),
			valid + "x", strings.Replace(valid, "v1.", "v2.", 1), "malformed",
		} {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.AddCookie(&http.Cookie{Name: DefaultCookieOptions().Name, Value: value})
			if _, ok := LoginIDFromCookie(CookieOptions{SecretKey: "new-key", VerificationKey: "old-key", MaxAge: 24 * time.Hour})(r); ok {
				t.Fatal("rotation accepted an expired or tampered cookie")
			}
		}
	}
}

// TestCookieRotationConfiguration keeps cookie trust independent of protocol companion keys. TestCookieRotationConfiguration 保持 Cookie 信任独立于协议过渡密钥。
func TestCookieRotationConfiguration(t *testing.T) {
	options := HTTPOptions{Cookie: CookieOptions{VerificationKey: "private-companion"}, LoginIDResolver: func(*http.Request) (string, bool) { return "alice", true }}
	err := options.Validate()
	if !errors.Is(err, ErrSSOConfig) || !errors.Is(err, ErrSignSecretRequired) || strings.Contains(err.Error(), "private-companion") {
		t.Fatalf("orphan cookie key diagnostic: %v", err)
	}
	options.SecretKey = "fallback-cookie-key"
	if err := options.Validate(); err != nil {
		t.Fatal(err)
	}
	s := NewServer()
	defer s.Close()
	h, err := NewHTTPServerChecked(s, HTTPOptions{ServerOptions: ServerOptions{SecretKey: "new-key", VerificationKey: "old-key", CheckSign: true}})
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(rotationCookie(t, CookieOptions{SecretKey: "old-key"}))
	if _, ok := h.options.LoginIDResolver(r); ok {
		t.Fatal("protocol companion implicitly expanded cookie trust")
	}
	h, err = NewHTTPServerChecked(s, HTTPOptions{ServerOptions: ServerOptions{SecretKey: "new-key"}, Cookie: CookieOptions{VerificationKey: "old-key"}})
	if err != nil {
		t.Fatal(err)
	}
	if id, ok := h.options.LoginIDResolver(r); !ok || id != "alice" {
		t.Fatal("explicit cookie companion did not apply to the default resolver")
	}
	b := newBrowserTestClient(t, nil, nil)
	stateOptions := b.options
	stateOptions.StateCookie.SecretKey = ""
	stateOptions.StateCookie.VerificationKey = "old-key"
	if _, err := NewBrowserClient(b.app, stateOptions); !errors.Is(err, ErrBrowserConfig) {
		t.Fatal("browser accepted a verification-only state key")
	}
}

// TestCookieRotationBrowserFlows shares pending PKCE flows across key generations without restoring consumed state. TestCookieRotationBrowserFlows 跨密钥代际共享待完成 PKCE 流程，不恢复已消费状态。
func TestCookieRotationBrowserFlows(t *testing.T) {
	base := newBrowserTestClient(t, nil, nil)
	makeBrowser := func(primary, companion string) *BrowserClient {
		t.Helper()
		options := base.options
		options.PKCE = true
		options.StateCookie.SecretKey, options.StateCookie.VerificationKey = primary, companion
		b, err := NewBrowserClient(base.app, options)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	old, prepared, switched, retired := makeBrowser("old-key", ""), makeBrowser("old-key", "new-key"), makeBrowser("new-key", "old-key"), makeBrowser("new-key", "")
	for _, pair := range [][2]*BrowserClient{{old, switched}, {switched, prepared}} {
		auth, cookie := beginBrowserTestLogin(t, pair[0], "/orders")
		result, err := pair[1].CompleteLogin(httptest.NewRecorder(), browserTestCallback(pair[1], auth, cookie))
		if err != nil || result.Identity.LoginID != "alice" || result.ReturnTo != "/orders" {
			t.Fatalf("cross-generation flow: %v", err)
		}
		if _, err := pair[0].CompleteLogin(httptest.NewRecorder(), browserTestCallback(pair[0], auth, cookie)); !errors.Is(err, ErrInvalidLoginState) {
			t.Fatal("rotation restored consumed state")
		}
	}
	auth, cookie := beginBrowserTestLogin(t, old, "/orders")
	if _, err := retired.CompleteLogin(httptest.NewRecorder(), browserTestCallback(retired, auth, cookie)); !errors.Is(err, ErrInvalidLoginState) {
		t.Fatal("retired key accepted a pending login")
	}
	if _, err := switched.CompleteLogin(httptest.NewRecorder(), browserTestCallback(switched, auth, cookie)); err != nil {
		t.Fatal("rejected old key consumed pending state")
	}
	auth, cookie = beginBrowserTestLogin(t, old, "/orders")
	key := old.options.KeyPrefix + auth.Query().Get(old.app.config.Params.Back)
	raw, err := old.options.Storage.Get(context.Background(), key)
	if err != nil {
		t.Fatal(err)
	}
	var record browserLoginState
	payload, ok := raw.(string)
	if !ok || json.Unmarshal([]byte(payload), &record) != nil {
		t.Fatal("pending state was not stored")
	}
	record.ExpiresAt = time.Now().Add(-time.Minute)
	encoded, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := old.options.Storage.Set(context.Background(), key, string(encoded), time.Hour); err != nil {
		t.Fatal(err)
	}
	if _, err := switched.CompleteLogin(httptest.NewRecorder(), browserTestCallback(switched, auth, cookie)); !errors.Is(err, ErrInvalidLoginState) {
		t.Fatal("rotation revived expired state retained by storage")
	}
}
