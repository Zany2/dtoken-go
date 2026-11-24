package sso

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/contract"
)

// TestLogoutCallbackErrorsStorage keeps backend diagnostics local and stops application effects. TestLogoutCallbackErrorsStorage 将后端诊断保留在本地并阻止业务副作用。
func TestLogoutCallbackErrorsStorage(t *testing.T) {
	failure := errors.New("private-backend-address and credentials")
	for _, scenario := range []string{"read", "write", "atomic"} {
		t.Run(scenario, func(t *testing.T) {
			memory := NewMemoryStorage()
			basic := &replayFaultStorage{Storage: memory}
			var storage contract.Storage = basic
			switch scenario {
			case "read":
				basic.readErr = failure
			case "write":
				basic.writeErr = failure
			case "atomic":
				storage = &replayAtomicFaultStorage{MemoryStorage: memory}
			}
			protection := newReplayTestProtection(t, storage)
			app := NewClientApp(ClientConfig{ClientID: "app", CheckSign: true, SecretKey: "key", ReplayProtection: protection})
			values := protection.stamp(url.Values{"client": {"app"}, "loginId": {"alice"}}, DefaultParamNames(), ActionLogoutCallback)
			values = NewSigner("key").AttachSign(values)
			request := httptest.NewRequest(http.MethodPost, "/callback", strings.NewReader(values.Encode()))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			if _, err := app.VerifyLogoutCallback(request); !errors.Is(err, ErrStorageUnavailable) || (scenario != "atomic" && !errors.Is(err, failure)) {
				t.Fatalf("direct verification lost diagnostics: %v", err)
			}
			calls := 0
			handler := app.LogoutCallbackHandler(func(*http.Request, LogoutCallback) error { calls++; return nil })
			w := replayTestRequest(handler, http.MethodPost, "/callback", values)
			if w.Code != http.StatusInternalServerError || w.Body.String() != "Internal Server Error\n" || calls != 0 {
				t.Fatalf("storage failure response: status=%d body=%q calls=%d", w.Code, w.Body.String(), calls)
			}
			if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Pragma") != "no-cache" {
				t.Fatal("error response lost cache prevention")
			}
			if scenario != "atomic" {
				basic.readErr, basic.writeErr = nil, nil
				w = replayTestRequest(handler, http.MethodPost, "/callback", values)
				if w.Code != http.StatusOK || calls != 1 {
					t.Fatalf("failed storage claim consumed nonce: status=%d calls=%d", w.Code, calls)
				}
			}
		})
	}
}

// TestLogoutCallbackErrorsHook preserves nonce consumption after failure and permits fresh retries. TestLogoutCallbackErrorsHook 保留钩子失败后的随机值消费语义并允许新请求重试。
func TestLogoutCallbackErrorsHook(t *testing.T) {
	protection := newReplayTestProtection(t, nil)
	app := NewClientApp(ClientConfig{ClientID: "app", CheckSign: true, SecretKey: "key", ReplayProtection: protection})
	calls := 0
	handler := app.LogoutCallbackHandler(func(*http.Request, LogoutCallback) error {
		calls++
		if calls == 1 {
			return errors.New("private-session-store credentials")
		}
		return nil
	})
	values := protection.stamp(url.Values{"client": {"app"}, "loginId": {"alice"}}, DefaultParamNames(), ActionLogoutCallback)
	values = NewSigner("key").AttachSign(values)
	w := replayTestRequest(handler, http.MethodPost, "/callback", values)
	if w.Code != http.StatusInternalServerError || w.Body.String() != "logout callback failed\n" || calls != 1 {
		t.Fatalf("hook failure response: status=%d body=%q calls=%d", w.Code, w.Body.String(), calls)
	}
	w = replayTestRequest(handler, http.MethodPost, "/callback", values)
	if w.Code != http.StatusUnauthorized || w.Body.String() != "Unauthorized\n" || calls != 1 {
		t.Fatalf("failed callback replay: status=%d body=%q calls=%d", w.Code, w.Body.String(), calls)
	}
	values = NewSigner("key").AttachSign(protection.stamp(values, DefaultParamNames(), ActionLogoutCallback))
	w = replayTestRequest(handler, http.MethodPost, "/callback", values)
	if w.Code != http.StatusOK || calls != 2 {
		t.Fatalf("fresh retry: status=%d body=%q calls=%d", w.Code, w.Body.String(), calls)
	}
}

// TestLogoutCallbackErrorsStatus retains protocol status categories without executing the hook. TestLogoutCallbackErrorsStatus 保留协议状态分类且不执行钩子。
func TestLogoutCallbackErrorsStatus(t *testing.T) {
	for _, tc := range []struct {
		name   string
		method string
		edit   func(url.Values)
		status int
	}{
		{name: "method", method: http.MethodGet, status: http.StatusMethodNotAllowed},
		{name: "signature", edit: func(v url.Values) { v.Set("sign", "private-invalid-signature") }, status: http.StatusUnauthorized},
		{name: "identity", edit: func(v url.Values) { v.Set("loginId", "") }, status: http.StatusBadRequest},
		{name: "size", edit: func(v url.Values) { v.Set("extra", strings.Repeat("x", 1025)) }, status: http.StatusRequestEntityTooLarge},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := NewClientApp(ClientConfig{ClientID: "app", CheckSign: true, SecretKey: "key", RequestLimits: RequestLimits{MaxBodyBytes: 1024}})
			values := url.Values{"client": {"app"}, "loginId": {"alice"}, "timestamp": {time.Now().Format(time.RFC3339)}}
			if tc.edit != nil {
				tc.edit(values)
			}
			if tc.name != "signature" {
				values = NewSigner("key").AttachSign(values)
			}
			calls := 0
			handler := app.LogoutCallbackHandler(func(*http.Request, LogoutCallback) error { calls++; return nil })
			method := tc.method
			if method == "" {
				method = http.MethodPost
			}
			w := replayTestRequest(handler, method, "/callback", values)
			if w.Code != tc.status || w.Body.String() != http.StatusText(tc.status)+"\n" || calls != 0 {
				t.Fatalf("rejection: status=%d body=%q calls=%d", w.Code, w.Body.String(), calls)
			}
		})
	}
}
