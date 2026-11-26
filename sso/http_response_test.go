package sso

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// responseMarshaler exposes encoding attempts and injected failures. responseMarshaler 记录编码尝试并注入故障。
type responseMarshaler struct {
	calls   int
	failure error
}

// TestHTTPResponseNonceConsumption keeps accepted signed exchanges consumed after response encoding fails. TestHTTPResponseNonceConsumption 在响应编码失败后保持已接受的签名兑换已消费。
func TestHTTPResponseNonceConsumption(t *testing.T) {
	s := NewServer()
	defer s.Close()
	client := newTestClient()
	client.SecretKey = "key"
	if err := s.RegisterClient(client); err != nil {
		t.Fatal(err)
	}
	ticket, err := s.GenerateTicket(context.Background(), client.ClientID, "alice", client.RedirectURIs[0], nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	protection := newReplayTestProtection(t, nil)
	observed, enrichments := 0, 0
	var last error
	h := NewHTTPServer(s, HTTPOptions{
		ServerOptions: ServerOptions{CheckSign: true}, ReplayProtection: protection,
		EnrichExchangeResult: func(_ *http.Request, _ string, result *TicketExchangeResult) error {
			enrichments++
			result.Extra = map[string]any{"invalid": make(chan int)}
			return nil
		},
		OnHTTPError: func(_ *http.Request, _ int, err error) { observed++; last = err },
	})
	values := url.Values{"client": {client.ClientID}, "clientSecret": {client.ClientSecret}, "redirect": {client.RedirectURIs[0]}, "ticket": {ticket.Ticket}}
	values = NewSigner("key").AttachSign(protection.stamp(values, DefaultParamNames(), ActionToken))
	assertHTTPFailure(t, replayTestRequest(h.Handler(), "POST", "/sso/token", values), 500)
	if observed != 1 || enrichments != 1 || !errors.Is(last, ErrSerializeFailed) {
		t.Fatal("response encoding failure did not reach observer once")
	}
	assertHTTPFailure(t, replayTestRequest(h.Handler(), "POST", "/sso/token", values), 401)
	if observed != 2 || enrichments != 1 || !errors.Is(last, ErrRequestReplayed) {
		t.Fatal("encoding failure restored nonce")
	}
	values = NewSigner("key").AttachSign(protection.stamp(values, DefaultParamNames(), ActionToken))
	assertHTTPFailure(t, replayTestRequest(h.Handler(), "POST", "/sso/token", values), 400)
	if enrichments != 1 {
		t.Fatal("fresh nonce restored consumed ticket")
	}
}

// TestHTTPResponseLogoutReport preserves lifecycle diagnostics without serializing private errors. TestHTTPResponseLogoutReport 保留退出生命周期诊断且不序列化私有错误。
func TestHTTPResponseLogoutReport(t *testing.T) {
	for _, failed := range []bool{false, true} {
		private := errors.New("private-logout-diagnostic")
		result := LogoutResult{Stage: "complete", Result: ResultOK, LoginID: "private-alice"}
		if failed {
			result.Stage, result.Result, result.Cause = "cleanup", "", private
		}
		observed, lifecycle := 0, 0
		h := &HTTPServer{options: HTTPOptions{
			OnLogoutResult: func(_ *http.Request, got LogoutResult) {
				lifecycle++
				if got.Stage != result.Stage {
					t.Fatal("logout stage changed")
				}
			},
			OnHTTPError: func(_ *http.Request, _ int, err error) {
				observed++
				if !errors.Is(err, private) {
					t.Fatal("logout cause replaced")
				}
			},
		}}
		w := &responseWriterProbe{ResponseRecorder: httptest.NewRecorder()}
		h.writeLogoutResult(w, httptest.NewRequest("POST", "/sso/logout", nil), result)
		if !json.Valid(w.Body.Bytes()) || strings.Contains(w.Body.String(), "private-") || lifecycle != 1 || observed != map[bool]int{true: 1, false: 0}[failed] || w.headers != 1 || w.writes != 1 {
			t.Fatal("logout response or observation changed")
		}
	}
}

// MarshalJSON returns a deliberate private failure or valid JSON. MarshalJSON 返回刻意的私有故障或有效 JSON。
func (m *responseMarshaler) MarshalJSON() ([]byte, error) {
	m.calls++
	if m.failure != nil {
		return nil, m.failure
	}
	return []byte(`{"value":"<safe>"}`), nil
}

// responseWriterProbe tracks commitments and can simulate a broken connection. responseWriterProbe 记录响应提交并模拟连接中断。
type responseWriterProbe struct {
	*httptest.ResponseRecorder
	headers, writes int
	fail            bool
}

// WriteHeader counts status commitments. WriteHeader 记录状态提交次数。
func (w *responseWriterProbe) WriteHeader(status int) {
	w.headers++
	w.ResponseRecorder.WriteHeader(status)
}

// Write injects a partial transport failure without allowing a replacement response. Write 注入部分传输故障，不允许替换响应。
func (w *responseWriterProbe) Write(p []byte) (int, error) {
	w.writes++
	if w.fail {
		_, _ = w.ResponseRecorder.Write(p[:1])
		return 1, io.ErrClosedPipe
	}
	return w.ResponseRecorder.Write(p)
}

// TestHTTPResponseEncoding verifies atomic encoding, safe fallbacks, headers and diagnostic preservation. TestHTTPResponseEncoding 验证完整编码、安全回退、响应头及诊断保留。
func TestHTTPResponseEncoding(t *testing.T) {
	cycle := map[string]any{}
	cycle["self"] = cycle
	private := errors.New("private-encoder-secret")
	for _, tc := range []struct {
		name  string
		value any
	}{
		{"channel", make(chan int)}, {"function", func() {}}, {"NaN", math.NaN()},
		{"infinity", math.Inf(1)}, {"cycle", cycle}, {"raw JSON", json.RawMessage(`{"private-secret":`)},
		{"custom", &responseMarshaler{failure: private}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, center := range []bool{false, true} {
				w := &responseWriterProbe{ResponseRecorder: httptest.NewRecorder()}
				observed := 0
				h := &HTTPServer{options: HTTPOptions{OnHTTPError: func(_ *http.Request, status int, err error) {
					observed++
					if status != 500 || !errors.Is(err, ErrSerializeFailed) || w.headers != 0 || w.writes != 0 {
						t.Fatal("encoding error was not observed before commitment")
					}
					if tc.name == "custom" && !errors.Is(err, private) {
						t.Fatal("custom encoding cause lost")
					}
				}}}
				response := OKResponse(map[string]any{"identity": "private-alice", "extension": tc.value})
				if center {
					h.writeHTTPResponse(w, httptest.NewRequest("POST", "/sso/token", nil), 200, response)
				} else {
					writeJSON(w, 200, response)
				}
				assertHTTPFailure(t, w.ResponseRecorder, 500)
				if w.Body.String() != "{\"code\":500,\"message\":\"SSO response encoding failed\"}\n" || w.headers != 1 || w.writes != 1 || observed != map[bool]int{false: 0, true: 1}[center] {
					t.Fatal("partial data, repeated commitment or incorrect observation")
				}
			}
		})
	}
}

// TestHTTPResponseSuccessAndDisconnect preserves JSON fidelity and never retries after transport commitment. TestHTTPResponseSuccessAndDisconnect 保留 JSON 精度且不在传输提交后重试。
func TestHTTPResponseSuccessAndDisconnect(t *testing.T) {
	for _, disconnect := range []bool{false, true} {
		marshaler := &responseMarshaler{}
		response := OKResponse(map[string]any{"id": json.Number("9007199254740993"), "custom": marshaler})
		w := &responseWriterProbe{ResponseRecorder: httptest.NewRecorder(), fail: disconnect}
		observed := 0
		h := &HTTPServer{options: HTTPOptions{OnHTTPError: func(*http.Request, int, error) { observed++ }}}
		h.writeHTTPResponse(w, httptest.NewRequest("POST", "/", nil), 200, response)
		if marshaler.calls != 1 || w.headers != 1 || w.writes != 1 || observed != 0 || w.Code != 200 {
			t.Fatal("response was encoded or committed more than once")
		}
		if !disconnect && (!json.Valid(w.Body.Bytes()) || !strings.Contains(w.Body.String(), "9007199254740993") || strings.Contains(w.Body.String(), "<safe>")) {
			t.Fatal("successful JSON precision or escaping changed")
		}
	}
}

// responseFaultCodec models a custom storage codec returning values unsupported by JSON transport. responseFaultCodec 模拟自定义存储编解码器返回 JSON 传输不支持的值。
type responseFaultCodec struct {
	JSONCodec
	fail bool
}

// Decode injects an invalid extension only into credential records after ordinary decoding. Decode 在普通解码后仅向凭证记录注入无效扩展。
func (c *responseFaultCodec) Decode(data []byte, value any) error {
	if err := c.JSONCodec.Decode(data, value); err != nil {
		return err
	}
	if !c.fail {
		return nil
	}
	extra := map[string]any{"private-extension": make(chan int)}
	switch v := value.(type) {
	case *Ticket:
		v.Extra = extra
	case *OAuth2Code:
		v.Extra = extra
	case *SharedToken:
		v.Extra = extra
	case *RemoteSession:
		v.Extra = extra
	}
	return nil
}

// TestHTTPResponseCredentialLifecycle checks inspection recovery and consumed exchange failures across modes. TestHTTPResponseCredentialLifecycle 检查各模式的查询恢复及已消费兑换失败。
func TestHTTPResponseCredentialLifecycle(t *testing.T) {
	for _, mode := range []Mode{ModeTicket, ModeOAuth2, ModeSharedToken, ModeRemoteSession} {
		t.Run(string(mode), func(t *testing.T) {
			s := NewServer()
			defer s.Close()
			codec := &responseFaultCodec{}
			s.serializer = codec
			client := newTestClient()
			client.Modes = []Mode{mode}
			if err := s.RegisterClient(client); err != nil {
				t.Fatal(err)
			}
			observed := 0
			h := NewHTTPServer(s, HTTPOptions{
				LoginIDResolver: func(*http.Request) (string, bool) { return "alice", true },
				OnHTTPError: func(_ *http.Request, status int, err error) {
					observed++
					if status != 500 || !errors.Is(err, ErrSerializeFailed) {
						t.Fatal("encoding classification lost")
					}
				},
			})
			issue := url.Values{"client": {client.ClientID}, "redirect": {client.RedirectURIs[0]}, "mode": {string(mode)}}
			out := replayTestRequest(h.Handler(), "GET", "/sso/authorize", issue)
			location, err := url.Parse(out.Header().Get("Location"))
			if err != nil || out.Code != 302 {
				t.Fatal("issuance failed")
			}
			values := location.Query()
			values.Set("mode", string(mode))
			values.Set("client", client.ClientID)
			values.Set("clientSecret", client.ClientSecret)
			values.Set("redirect", client.RedirectURIs[0])
			for _, path := range []string{"/sso/introspect", "/sso/userinfo"} {
				before := observed
				codec.fail = true
				assertHTTPFailure(t, replayTestRequest(h.Handler(), "POST", path, values), 500)
				if observed != before+1 {
					t.Fatal("encoding observation count changed")
				}
				codec.fail = false
				if w := replayTestRequest(h.Handler(), "POST", path, values); w.Code != 200 || observed != before+1 {
					t.Fatal("response failure revoked an inspection credential")
				}
			}
			if mode == ModeTicket || mode == ModeOAuth2 {
				h.options.EnrichExchangeResult = func(_ *http.Request, _ string, result *TicketExchangeResult) error {
					result.Extra = map[string]any{"private-extension": math.NaN()}
					return nil
				}
				assertHTTPFailure(t, replayTestRequest(h.Handler(), "POST", "/sso/token", values), 500)
				h.options.OnHTTPError = nil
				assertHTTPFailure(t, replayTestRequest(h.Handler(), "POST", "/sso/token", values), 400)
			}
		})
	}
}
