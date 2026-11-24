package sso

import (
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"strings"
	"testing"
	"testing/iotest"
)

// boundedClientCall exercises each public protocol entry point and rejects partial results on error. boundedClientCall 调用各公开协议入口，并检查失败时不返回部分结果。
func boundedClientCall(t *testing.T, app *ClientApp, operation string) error {
	t.Helper()
	req := CredentialRequest{Mode: ModeTicket, Ticket: "ticket"}
	switch operation {
	case "ticket", "code":
		var result *TicketExchangeResult
		var err error
		if operation == "ticket" {
			result, err = app.ExchangeTicket(nil, "ticket", "")
		} else {
			result, err = app.ExchangeOAuth2Code(nil, "code", "")
		}
		if err != nil && result != nil {
			t.Fatal("exchange returned partial data")
		}
		return err
	case "introspect", "userinfo":
		var result *CredentialInfo
		var err error
		if operation == "introspect" {
			result, err = app.Introspect(nil, req)
		} else {
			result, err = app.UserInfo(nil, req)
		}
		if err != nil && result != nil {
			t.Fatal("read returned partial data")
		}
		return err
	default:
		return app.Revoke(nil, req)
	}
}

// TestClientResponseSizeLimits verifies actual byte limits for successful and failed responses across all endpoints. TestClientResponseSizeLimits 验证所有端点成功及失败响应的实际字节上限。
func TestClientResponseSizeLimits(t *testing.T) {
	const limit = 128
	payload := `{"code":0,"data":{"loginId":"alice","active":true}}`
	for _, operation := range []string{"ticket", "code", "introspect", "userinfo", "revoke"} {
		for _, tc := range []struct {
			name          string
			size, status  int
			contentLength int64
			success       bool
		}{
			{"below", limit - 1, http.StatusOK, -1, true},
			{"exact", limit, http.StatusOK, -1, true},
			{"over", limit + 1, http.StatusOK, -1, false},
			{"long stream", limit * 10, http.StatusOK, -1, false},
			{"understated length", limit + 1, http.StatusOK, 1, false},
			{"overstated length", limit - 1, http.StatusOK, limit * 10, true},
			{"large HTTP error", limit * 10, http.StatusBadGateway, -1, false},
			{"small HTTP error", limit - 1, http.StatusBadGateway, -1, false},
		} {
			t.Run(operation+"/"+tc.name, func(t *testing.T) {
				body := &logoutAcknowledgementBody{reader: strings.NewReader(payload + strings.Repeat(" ", tc.size-len(payload)))}
				app := NewClientApp(ClientConfig{ServerURL: "https://center.example.com", MaxResponseBytes: limit,
					HTTPClient: &http.Client{Transport: logoutReviewTransport(func(*http.Request) (*http.Response, error) {
						return &http.Response{StatusCode: tc.status, ContentLength: tc.contentLength, Body: body}, nil
					})},
				})
				err := boundedClientCall(t, app, operation)
				if (err == nil) != tc.success {
					t.Fatalf("success=%v error=%v", tc.success, err)
				}
				if tc.size > limit && (err == nil || !strings.Contains(err.Error(), "exceeds")) {
					t.Fatalf("missing size rejection: %v", err)
				}
				if !body.closed || body.readBytes > limit+1 {
					t.Fatalf("body closed=%v read=%d", body.closed, body.readBytes)
				}
				if tc.size > limit && body.readBytes != limit+1 {
					t.Fatalf("unexpected probe: %d", body.readBytes)
				}
			})
		}
	}
}

// TestClientResponseLimitConfiguration checks default normalization, customization, and maximum integer safety. TestClientResponseLimitConfiguration 检查默认归一化、自定义上限和最大整数安全性。
func TestClientResponseLimitConfiguration(t *testing.T) {
	if DefaultClientConfig().MaxResponseBytes != 1<<20 {
		t.Fatal("unexpected default limit")
	}
	payload := `{"code":0,"data":{"loginId":"alice"}}`
	for _, configured := range []int64{0, -1, 128, math.MaxInt64} {
		body := &logoutAcknowledgementBody{reader: strings.NewReader(payload)}
		app := NewClientApp(ClientConfig{ServerURL: "https://center.example.com", MaxResponseBytes: configured,
			HTTPClient: &http.Client{Transport: logoutReviewTransport(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK, Body: body}, nil
			})},
		})
		want := configured
		if want <= 0 {
			want = DefaultClientMaxResponseBytes
		}
		if app.Config().MaxResponseBytes != want {
			t.Fatalf("normalized limit=%d, want=%d", app.Config().MaxResponseBytes, want)
		}
		if err := boundedClientCall(t, app, "ticket"); err != nil || !body.closed {
			t.Fatalf("configured=%d error=%v closed=%v", configured, err, body.closed)
		}
	}
	body := &logoutAcknowledgementBody{reader: strings.NewReader(payload + strings.Repeat(" ", int(DefaultClientMaxResponseBytes)))}
	app := NewClientApp(ClientConfig{ServerURL: "https://center.example.com",
		HTTPClient: &http.Client{Transport: logoutReviewTransport(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: body}, nil
		})},
	})
	if err := boundedClientCall(t, app, "revoke"); err == nil || !strings.Contains(err.Error(), "exceeds") || !body.closed || int64(body.readBytes) != DefaultClientMaxResponseBytes+1 {
		t.Fatalf("default bound failed: %v, bytes=%d", err, body.readBytes)
	}
}

// TestClientResponseReadFailures verifies failures before and at the limit are preserved and bodies close. TestClientResponseReadFailures 验证上限前及边界处的读取故障正常上报并关闭响应体。
func TestClientResponseReadFailures(t *testing.T) {
	const limit = 128
	payload := `{"code":0,"data":{"loginId":"alice","active":true}}`
	readErr := errors.New("response stream interrupted")
	for _, operation := range []string{"ticket", "code", "introspect", "userinfo", "revoke"} {
		for _, size := range []int{0, len(payload), limit} {
			prefix := ""
			if size > 0 {
				prefix = payload + strings.Repeat(" ", size-len(payload))
			}
			body := &logoutAcknowledgementBody{reader: io.MultiReader(strings.NewReader(prefix), iotest.ErrReader(readErr))}
			app := NewClientApp(ClientConfig{ServerURL: "https://center.example.com", MaxResponseBytes: limit,
				HTTPClient: &http.Client{Transport: logoutReviewTransport(func(*http.Request) (*http.Response, error) {
					return &http.Response{StatusCode: http.StatusOK, Body: body}, nil
				})},
			})
			if err := boundedClientCall(t, app, operation); !errors.Is(err, readErr) || !body.closed {
				t.Fatalf("%s at %d: error=%v closed=%v", operation, size, err, body.closed)
			}
		}
	}
}

// TestClientExchangeTimeoutValidation enforces the protocol sentinel range without losing large integer precision. TestClientExchangeTimeoutValidation 校验协议特殊值范围且不丢失大整数精度。
func TestClientExchangeTimeoutValidation(t *testing.T) {
	for _, operation := range []string{"ticket", "code"} {
		for _, tc := range []struct {
			token, session int64
			invalid        bool
		}{
			{0, 0, false}, {-1, -1, false}, {1, 60, false}, {math.MaxInt64, math.MaxInt64, false},
			{-2, 60, true}, {60, -2, true}, {math.MinInt64, 0, true},
		} {
			want := TicketExchangeResult{LoginID: "alice", RemainTokenTimeout: tc.token, RemainSessionTimeout: tc.session}
			payload, err := json.Marshal(OKResponse(want))
			if err != nil {
				t.Fatal(err)
			}
			body := &logoutAcknowledgementBody{reader: strings.NewReader(string(payload))}
			app := NewClientApp(ClientConfig{ServerURL: "https://center.example.com",
				HTTPClient: &http.Client{Transport: logoutReviewTransport(func(*http.Request) (*http.Response, error) {
					return &http.Response{StatusCode: http.StatusOK, Body: body}, nil
				})},
			})
			var got *TicketExchangeResult
			if operation == "ticket" {
				got, err = app.ExchangeTicket(nil, "ticket", "")
			} else {
				got, err = app.ExchangeOAuth2Code(nil, "code", "")
			}
			if tc.invalid {
				if err == nil || got != nil {
					t.Fatalf("accepted invalid timeout: %+v, %v", got, err)
				}
			} else if err != nil || got == nil || got.RemainTokenTimeout != want.RemainTokenTimeout || got.RemainSessionTimeout != want.RemainSessionTimeout {
				t.Fatalf("valid timeout changed: %+v, %v", got, err)
			}
			if !body.closed {
				t.Fatal("response body not closed")
			}
		}
	}
}
