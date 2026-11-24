package sso

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// lateAcknowledgement returns valid bytes only after the attempt deadline when wait is enabled. lateAcknowledgement 在启用等待时仅于尝试截止后返回有效确认数据。
type lateAcknowledgement struct {
	ctx          context.Context
	reader       io.Reader
	wait, closed bool
}

// Read simulates a body reader that ignores cancellation and returns a successful acknowledgement. Read 模拟忽略取消且返回成功确认的响应体读取器。
func (b *lateAcknowledgement) Read(p []byte) (int, error) {
	if b.wait {
		<-b.ctx.Done()
		b.wait = false
	}
	return b.reader.Read(p)
}

// Close records response cleanup for every attempt. Close 记录每次尝试的响应体释放。
func (b *lateAcknowledgement) Close() error { b.closed = true; return nil }

// TestCancellationReviewNotificationDeadline rejects late transport/body successes and permits a fresh bounded retry. TestCancellationReviewNotificationDeadline 拒绝延迟传输或响应体成功，并允许新的有限重试。
func TestCancellationReviewNotificationDeadline(t *testing.T) {
	for _, phase := range []string{"transport", "body"} {
		for _, recover := range []bool{false, true} {
			t.Run(phase+map[bool]string{false: "/exhausted", true: "/recovered"}[recover], func(t *testing.T) {
				s := NewServer()
				defer s.Close()
				registerNotificationClients(t, s, "app-a")
				var bodies []*lateAcknowledgement
				h := NewHTTPServer(s, HTTPOptions{ServerOptions: ServerOptions{
					LogoutCallbackMaxAttempts: 2, LogoutCallbackRetryInterval: time.Nanosecond, LogoutCallbackTimeout: 50 * time.Millisecond,
					LogoutHTTPClient: &http.Client{Transport: logoutReviewTransport(func(r *http.Request) (*http.Response, error) {
						late := !recover || len(bodies) == 0
						body := &lateAcknowledgement{ctx: r.Context(), reader: strings.NewReader(`{"code":0}`), wait: late && phase == "body"}
						bodies = append(bodies, body)
						if late && phase == "transport" {
							<-r.Context().Done()
						}
						return &http.Response{StatusCode: http.StatusOK, Body: body}, nil
					})},
				}})
				report, err := h.NotifyLogoutClients(httptest.NewRequest("POST", "/logout", nil), "alice")
				if report == nil || report.Complete != recover || len(bodies) != 2 || report.Clients[0].Attempts != 2 {
					t.Fatalf("late report=%+v error=%v attempts=%d", report, err, len(bodies))
				}
				if recover {
					if err != nil || report.Clients[0].Status != LogoutCallbackDelivered {
						t.Fatalf("fresh retry failed: %+v, %v", report, err)
					}
				} else if !errors.Is(err, context.DeadlineExceeded) || report.Clients[0].Status != LogoutCallbackFailed || report.Clients[0].Reason != "timeout" {
					t.Fatalf("late success acknowledged: %+v, %v", report, err)
				}
				for _, body := range bodies {
					if !body.closed {
						t.Fatal("attempt leaked response body")
					}
				}
			})
		}
	}
}

// TestCancellationReviewNotificationParent keeps parent cancellation distinct from delivered callbacks and attempt timeouts. TestCancellationReviewNotificationParent 将父请求取消与已投递及单次超时区分。
func TestCancellationReviewNotificationParent(t *testing.T) {
	s := NewServer()
	defer s.Close()
	registerNotificationClients(t, s, "app-a")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	body := &lateAcknowledgement{reader: strings.NewReader("ok")}
	calls := 0
	h := NewHTTPServer(s, HTTPOptions{ServerOptions: ServerOptions{LogoutCallbackMaxAttempts: 2, LogoutHTTPClient: &http.Client{Transport: logoutReviewTransport(func(*http.Request) (*http.Response, error) {
		calls++
		cancel()
		return &http.Response{StatusCode: 200, Body: body}, nil
	})}}})
	report, err := h.NotifyLogoutClients(httptest.NewRequest("POST", "/logout", nil).WithContext(ctx), "alice")
	if !errors.Is(err, context.Canceled) || report == nil || report.Complete || report.Clients[0].Status != LogoutCallbackCanceled || report.Clients[0].Reason != "canceled" || calls != 1 || !body.closed {
		t.Fatalf("parent cancellation = %+v, %v, calls=%d closed=%v", report, err, calls, body.closed)
	}
}

// TestCancellationReviewCenterLogoutStages stops browser logout where cancellation occurred without clearing its cookie. TestCancellationReviewCenterLogoutStages 在浏览器退出发生取消的阶段停止且不清理 Cookie。
func TestCancellationReviewCenterLogoutStages(t *testing.T) {
	for _, stage := range []string{"center_logout", "credentials", "cleanup"} {
		t.Run(stage, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			store := &accountCancelStorage{MemoryStorage: NewMemoryStorage(), stage: stage, cancel: cancel}
			s := NewServer(WithStorage(store))
			defer s.Close()
			var result LogoutResult
			observations := 0
			h := NewHTTPServer(s, HTTPOptions{
				ServerOptions: ServerOptions{EnableSLO: true}, LoginIDResolver: func(*http.Request) (string, bool) { return "alice", true },
				CenterLogout: func(*http.Request, string) error {
					if stage == "center_logout" {
						cancel()
					}
					return nil
				},
				OnLogoutResult: func(_ *http.Request, got LogoutResult) { observations++; result = got },
			})
			w := httptest.NewRecorder()
			h.HandleLogout(w, httptest.NewRequest("POST", "/sso/logout", nil).WithContext(ctx))
			if w.Code < 400 || len(w.Result().Cookies()) != 0 || observations != 1 || result.Stage != stage || result.Result != "" || !errors.Is(result.Cause, context.Canceled) {
				t.Fatalf("logout cancellation = %+v, status=%d observers=%d", result, w.Code, observations)
			}
		})
	}
}

// TestCancellationReviewClientAcknowledgement rejects canceled hooks and retains signed nonce consumption semantics. TestCancellationReviewClientAcknowledgement 拒绝已取消钩子的成功确认，保留签名随机值消费语义。
func TestCancellationReviewClientAcknowledgement(t *testing.T) {
	for _, early := range []bool{true, false} {
		ctx, cancel := context.WithCancel(context.Background())
		protection := newReplayTestProtection(t, nil)
		app := NewClientApp(ClientConfig{ClientID: "app", CheckSign: true, SecretKey: "signing-key", ReplayProtection: protection})
		calls, cancelHook := 0, !early
		handler := app.LogoutCallbackHandler(func(*http.Request, LogoutCallback) error {
			calls++
			if cancelHook {
				cancel()
			}
			return nil
		})
		values := protection.stamp(url.Values{"client": {"app"}, "loginId": {"alice"}}, DefaultParamNames(), ActionLogoutCallback)
		values = NewSigner("signing-key").AttachSign(values)
		request := func(ctx context.Context) *http.Request {
			r := httptest.NewRequest("POST", "/callback", strings.NewReader(values.Encode())).WithContext(ctx)
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			return r
		}
		if early {
			cancel()
		}
		w := httptest.NewRecorder()
		handler(w, request(ctx))
		cancel()
		wantCalls := 1
		if early {
			wantCalls = 0
		}
		if w.Code < 400 || calls != wantCalls {
			t.Fatalf("canceled acknowledgement = %d, calls=%d", w.Code, calls)
		}
		cancelHook = false
		w = httptest.NewRecorder()
		handler(w, request(context.Background()))
		if early && (w.Code != 200 || calls != 1) {
			t.Fatal("pre-cancellation consumed nonce")
		}
		if !early && (w.Code != http.StatusUnauthorized || calls != 1) {
			t.Fatal("completed hook restored consumed nonce")
		}
	}
}
