package sso

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// cancelingResponse simulates a transport body that ignores cancellation and still reports success. cancelingResponse 模拟忽略取消并继续返回成功的传输响应体。
type cancelingResponse struct {
	reader io.Reader
	cancel context.CancelFunc
	closed bool
}

// Read cancels the request while yielding the complete successful payload. Read 在提供完整成功载荷时取消请求。
func (b *cancelingResponse) Read(p []byte) (int, error) { b.cancel(); return b.reader.Read(p) }

// Close records body release after a rejected response. Close 记录拒绝响应后的响应体释放。
func (b *cancelingResponse) Close() error { b.closed = true; return nil }

// TestClientCancellationRejectsLateSuccess covers all client operations with custom transports. TestClientCancellationRejectsLateSuccess 覆盖自定义传输下全部客户端操作的延迟成功拒绝。
func TestClientCancellationRejectsLateSuccess(t *testing.T) {
	for _, operation := range []string{"ticket", "code", "introspect", "userinfo", "revoke"} {
		for _, early := range []bool{true, false} {
			t.Run(operation+map[bool]string{true: "/before transport", false: "/during body"}[early], func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				body := &cancelingResponse{reader: strings.NewReader(`{"code":0,"data":{"active":true,"loginId":"alice"}}`), cancel: cancel}
				calls := 0
				app := NewClientApp(ClientConfig{ServerURL: "https://center.example", HTTPClient: &http.Client{Transport: logoutReviewTransport(func(*http.Request) (*http.Response, error) {
					calls++
					return &http.Response{StatusCode: http.StatusOK, Body: body}, nil
				})}})
				if early {
					cancel()
				}
				var err error
				switch operation {
				case "ticket", "code":
					var result *TicketExchangeResult
					if operation == "ticket" {
						result, err = app.ExchangeTicket(ctx, "ticket", "")
					} else {
						result, err = app.ExchangeOAuth2Code(ctx, "code", "")
					}
					if result != nil {
						t.Fatal("canceled exchange returned identity")
					}
				case "introspect", "userinfo":
					var info *CredentialInfo
					if operation == "introspect" {
						info, err = app.Introspect(ctx, CredentialRequest{})
					} else {
						info, err = app.UserInfo(ctx, CredentialRequest{})
					}
					if info != nil {
						t.Fatal("canceled lookup returned identity")
					}
				default:
					err = app.Revoke(ctx, CredentialRequest{})
				}
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("cancel error = %v", err)
				}
				if early && calls != 0 {
					t.Fatal("already canceled request reached transport")
				}
				if !early && (calls != 1 || !body.closed) {
					t.Fatal("late canceled response body was not closed")
				}
			})
		}
	}
}

// TestCanceledBrowserExchangeDoesNotEstablishSession keeps consumed browser state consumed without invoking the session hook. TestCanceledBrowserExchangeDoesNotEstablishSession 保持浏览器状态已消费且不调用会话建立钩子。
func TestCanceledBrowserExchangeDoesNotEstablishSession(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	body := &cancelingResponse{reader: strings.NewReader(`{"code":0,"data":{"loginId":"alice"}}`), cancel: cancel}
	browser := newBrowserTestClient(t, nil, logoutReviewTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: body}, nil
	}))
	authorize, cookie := beginBrowserTestLogin(t, browser, "/protected")
	called := false
	handler := browser.CallbackHandler(func(http.ResponseWriter, *http.Request, *BrowserLoginResult) error { called = true; return nil })
	w := httptest.NewRecorder()
	handler(w, browserTestCallback(browser, authorize, cookie).WithContext(ctx))
	if called || w.Code == http.StatusSeeOther || !body.closed {
		t.Fatal("canceled exchange established browser session")
	}
	if _, err := browser.CompleteLogin(httptest.NewRecorder(), browserTestCallback(browser, authorize, cookie)); !errors.Is(err, ErrInvalidLoginState) {
		t.Fatalf("consumed state after cancellation = %v", err)
	}
}
