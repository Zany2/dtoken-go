package sso

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestCancellationReviewBrowserBegin suppresses cookies and redirects when state persistence finishes after cancellation. TestCancellationReviewBrowserBegin 在状态持久化取消后完成时禁止设置 Cookie 和跳转。
func TestCancellationReviewBrowserBegin(t *testing.T) {
	for _, early := range []bool{true, false} {
		ctx, cancel := context.WithCancel(context.Background())
		store := &cancellationReviewStorage{MemoryStorage: NewMemoryStorage(), afterSet: cancel}
		browser := newBrowserTestClient(t, store, nil)
		if early {
			cancel()
		}
		w := httptest.NewRecorder()
		err := browser.BeginLogin(w, httptest.NewRequest("GET", "/login", nil).WithContext(ctx), "/", nil)
		cancel()
		wantWrites := 1
		if early {
			wantWrites = 0
		}
		if !errors.Is(err, context.Canceled) || store.writes != wantWrites || len(w.Result().Cookies()) != 0 || w.Header().Get("Location") != "" {
			t.Fatalf("canceled begin: err=%v writes=%d headers=%v", err, store.writes, w.Header())
		}
	}
}

// TestCancellationReviewBrowserHooks never redirects after local login/logout hooks cancel the request. TestCancellationReviewBrowserHooks 本地登录或退出钩子取消请求后不再跳转。
func TestCancellationReviewBrowserHooks(t *testing.T) {
	browser := newBrowserTestClient(t, nil, nil)
	authorize, cookie := beginBrowserTestLogin(t, browser, "/protected")
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	login := browser.CallbackHandler(func(http.ResponseWriter, *http.Request, *BrowserLoginResult) error { calls++; cancel(); return nil })
	w := httptest.NewRecorder()
	login(w, browserTestCallback(browser, authorize, cookie).WithContext(ctx))
	cancel()
	if w.Code < 400 || calls != 1 || w.Header().Get("Location") != "" {
		t.Fatalf("late login success = %d, calls=%d", w.Code, calls)
	}
	if _, err := browser.CompleteLogin(httptest.NewRecorder(), browserTestCallback(browser, authorize, cookie)); !errors.Is(err, ErrInvalidLoginState) {
		t.Fatalf("state restored after hook cancellation: %v", err)
	}
	for _, global := range []bool{false, true} {
		for _, early := range []bool{false, true} {
			ctx, cancel := context.WithCancel(context.Background())
			calls := 0
			logout := browser.LogoutHandler(func(http.ResponseWriter, *http.Request) error { calls++; cancel(); return nil }, global, "/")
			r := httptest.NewRequest("POST", "https://app.example/logout", nil).WithContext(ctx)
			r.Header.Set("Origin", "https://app.example")
			if early {
				cancel()
			}
			w := httptest.NewRecorder()
			logout(w, r)
			cancel()
			wantCalls := 1
			if early {
				wantCalls = 0
			}
			if w.Code < 400 || calls != wantCalls || w.Header().Get("Location") != "" {
				t.Fatalf("late logout global=%v early=%v status=%d calls=%d", global, early, w.Code, calls)
			}
		}
	}
}
