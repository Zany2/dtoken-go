package sso

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// logoutAcknowledgementBody tracks response reads and closure, including injected read failures. logoutAcknowledgementBody 记录响应读取与关闭，并支持注入读取故障。
type logoutAcknowledgementBody struct {
	reader    io.Reader
	readErr   error
	readBytes int
	closed    bool
}

// Read delegates to the body reader or returns the injected failure. Read 转发响应读取或返回注入的故障。
func (b *logoutAcknowledgementBody) Read(p []byte) (int, error) {
	if b.readErr != nil {
		return 0, b.readErr
	}
	n, err := b.reader.Read(p)
	b.readBytes += n
	return n, err
}

// Close records that response resources were released. Close 记录响应资源已释放。
func (b *logoutAcknowledgementBody) Close() error {
	b.closed = true
	return nil
}

// TestLogoutAcknowledgementLifecycle verifies business failures preserve strict logout state. TestLogoutAcknowledgementLifecycle 验证业务失败会保留严格注销模式的状态。
func TestLogoutAcknowledgementLifecycle(t *testing.T) {
	for _, tc := range []struct {
		name, payload string
		status        int
		success       bool
		readErr       error
	}{
		{name: "legacy empty", status: http.StatusOK, success: true},
		{name: "no content", status: http.StatusNoContent, success: true},
		{name: "plain ok", payload: " \nok\r\n", status: http.StatusOK, success: true},
		{name: "json success", payload: `{"code":0,"data":{"result":"ok"}}`, status: http.StatusOK, success: true},
		{name: "limit boundary", payload: `{"code":0}` + strings.Repeat(" ", (64<<10)-len(`{"code":0}`)), status: http.StatusOK, success: true},
		{name: "business failure", payload: `{"code":1,"message":"local logout failed"}`, status: http.StatusOK},
		{name: "missing code", payload: `{}`, status: http.StatusOK},
		{name: "null code", payload: `{"code":null}`, status: http.StatusOK},
		{name: "null response", payload: `null`, status: http.StatusOK},
		{name: "string code", payload: `{"code":"0"}`, status: http.StatusOK},
		{name: "array response", payload: `[]`, status: http.StatusOK},
		{name: "html page", payload: `<html>Login required</html>`, status: http.StatusOK},
		{name: "truncated json", payload: `{"code":`, status: http.StatusOK},
		{name: "trailing response", payload: `{"code":0}{"code":1}`, status: http.StatusOK},
		{name: "oversized response", payload: `{"code":0}` + strings.Repeat(" ", 128<<10), status: http.StatusOK},
		{name: "read failure", status: http.StatusOK, readErr: io.ErrUnexpectedEOF},
		{name: "http failure", payload: `{"code":0}`, status: http.StatusServiceUnavailable},
	} {
		for _, bestEffort := range []bool{false, true} {
			policy := "strict"
			if bestEffort {
				policy = "best effort"
			}
			t.Run(tc.name+"/"+policy, func(t *testing.T) {
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
				body := &logoutAcknowledgementBody{reader: strings.NewReader(tc.payload), readErr: tc.readErr}
				centerCalls := 0
				h := NewHTTPServer(s, HTTPOptions{
					ServerOptions: ServerOptions{
						EnableSLO: true, LogoutCallbackBestEffort: bestEffort,
						LogoutHTTPClient: &http.Client{Transport: logoutReviewTransport(func(*http.Request) (*http.Response, error) {
							return &http.Response{StatusCode: tc.status, Body: body}, nil
						})},
					},
					LoginIDResolver: func(*http.Request) (string, bool) { return "user", true },
					CenterLogout:    func(*http.Request, string) error { centerCalls++; return nil },
				})
				response := protocolReviewPost(h.HandleLogout, nil)
				completed := tc.success || bestEffort
				wantStatus := http.StatusInternalServerError
				if completed {
					wantStatus = http.StatusOK
				}
				if response.Code != wantStatus {
					t.Fatalf("logout status = %d; want %d: %s", response.Code, wantStatus, response.Body.String())
				}
				if !completed && !strings.Contains(response.Body.String(), client.ClientID) {
					t.Fatal("callback failure did not identify the client")
				}
				sessions, err := s.GetClientSessions(ctx, "user")
				if err != nil || (len(sessions) == 0) != completed || (centerCalls == 1) != completed || (len(response.Result().Cookies()) == 1) != completed {
					t.Fatalf("cleanup mismatch: sessions=%+v error=%v center calls=%d", sessions, err, centerCalls)
				}
				if !body.closed || body.readBytes > (64<<10)+1 {
					t.Fatalf("response handling: closed=%v bytes read=%d", body.closed, body.readBytes)
				}

				// A business failure must remain retryable after the client recovers. 客户端恢复后，业务失败必须仍可重试。
				if tc.name == "business failure" && !bestEffort {
					body = &logoutAcknowledgementBody{reader: strings.NewReader(`{"code":0}`)}
					retry := protocolReviewPost(h.HandleLogout, nil)
					sessions, err = s.GetClientSessions(ctx, "user")
					if retry.Code != http.StatusOK || centerCalls != 1 || err != nil || len(sessions) != 0 || !body.closed {
						t.Fatalf("retry failed: status=%d sessions=%+v error=%v", retry.Code, sessions, err)
					}
				}
			})
		}
	}
}

// TestLogoutReportsAllFailedClients verifies fan-out errors retain every failed target and cause. TestLogoutReportsAllFailedClients 验证并发通知错误保留所有失败目标及原因。
func TestLogoutReportsAllFailedClients(t *testing.T) {
	s := NewServer()
	defer s.Close()
	causes := map[string]error{"app-a": errors.New("first offline"), "app-b": errors.New("second offline")}
	for _, id := range []string{"app-a", "app-b", "app-c"} {
		client := newTestClient()
		client.ClientID = id
		if err := s.RegisterClient(client); err != nil {
			t.Fatal(err)
		}
		if _, err := s.RegisterClientSession(context.Background(), "user", id, client.RedirectURIs[0]); err != nil {
			t.Fatal(err)
		}
	}
	var calls atomic.Int32
	h := NewHTTPServer(s, HTTPOptions{ServerOptions: ServerOptions{
		LogoutHTTPClient: &http.Client{Transport: logoutReviewTransport(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if err := r.ParseForm(); err != nil {
				return nil, err
			}
			if err := causes[r.Form.Get("client")]; err != nil {
				return nil, err
			}
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"code":0}`))}, nil
		})},
	}})
	err := h.pushLogoutCallbacks(httptest.NewRequest(http.MethodPost, "/sso/logout", nil), "user")
	if err == nil || calls.Load() != 3 {
		t.Fatalf("notification error=%v calls=%d", err, calls.Load())
	}
	for id, cause := range causes {
		if !errors.Is(err, cause) || !strings.Contains(err.Error(), id) {
			t.Fatalf("missing failure for %s: %v", id, err)
		}
	}
}
