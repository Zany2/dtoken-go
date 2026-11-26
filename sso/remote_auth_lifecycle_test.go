package sso

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestRemoteAuthCenterLifecycle exercises signed introspection, current application state, individual revocation, and account logout. TestRemoteAuthCenterLifecycle 验证签名检查、当前应用状态、单凭证撤销与账号注销。
func TestRemoteAuthCenterLifecycle(t *testing.T) {
	for _, mode := range []Mode{ModeRemoteSession, ModeSharedToken} {
		t.Run(string(mode), func(t *testing.T) {
			s := NewServer()
			defer s.Close()
			client := newTestClient()
			client.Modes, client.SecretKey = []Mode{mode}, "client-sign-key"
			if err := s.RegisterClient(client); err != nil {
				t.Fatal(err)
			}
			active := true
			var lookupError error
			params := normalizeParamNames(ParamNames{Client: "application", SessionID: "sid", TokenValue: "credential"})
			paths := DefaultEndpoints().AddPrefix("/gateway")
			h := NewHTTPServer(s, HTTPOptions{
				ServerOptions:         ServerOptions{CheckSign: true, Params: params, Endpoints: paths},
				ReplayProtection:      newReplayTestProtection(t, nil),
				LoginIDResolver:       func(*http.Request) (string, bool) { return "alice", true },
				CheckCredentialActive: func(*http.Request, CredentialInfo) (bool, error) { return active, lookupError },
			})
			requests := 0
			seenNonces := make(map[string]bool)
			app := NewClientApp(ClientConfig{
				Mode: mode, ClientID: client.ClientID, ClientSecret: client.ClientSecret, SecretKey: client.SecretKey, CheckSign: true,
				ServerURL: "https://center.example", Endpoints: paths, Params: params,
				ReplayProtection: newReplayTestProtection(t, nil),
				HTTPClient: &http.Client{Transport: logoutReviewTransport(func(r *http.Request) (*http.Response, error) {
					requests++
					if err := r.ParseForm(); err != nil {
						t.Fatal(err)
					}
					nonce := r.PostForm.Get(params.Nonce)
					if nonce == "" || seenNonces[nonce] || !NewSignerWithParams(client.SecretKey, params).Verify(r.PostForm) {
						t.Fatal("each remote check must carry a fresh signed request")
					}
					seenNonces[nonce] = true
					out := httptest.NewRecorder()
					h.Handler().ServeHTTP(out, r)
					return out.Result(), nil
				})},
			})
			issue := func(loginID string) string {
				t.Helper()
				if mode == ModeRemoteSession {
					value, err := s.CreateRemoteSession(context.Background(), client.ClientID, loginID, []string{"profile"}, map[string]any{"team": "orders"})
					if err != nil {
						t.Fatal(err)
					}
					return value.SessionID
				}
				value, err := s.GenerateSharedToken(context.Background(), client.ClientID, loginID, []string{"profile"}, map[string]any{"team": "orders"})
				if err != nil {
					t.Fatal(err)
				}
				return value.Token
			}
			alice, bob := issue("alice"), issue("bob")
			a, err := NewRemoteAuth(app, RemoteAuthOptions{RequiredScopes: []string{"profile"}})
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			protected := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				info, ok := RemoteIdentityFromContext(r.Context())
				if !ok || info.Mode != mode || info.Extra["team"] != "orders" {
					t.Fatal("identity metadata lost")
				}
				_, _ = w.Write([]byte(info.LoginID))
			}))
			check := func(credential string, want int) {
				t.Helper()
				beforeCalls, beforeRequests := calls, requests
				r := remoteAuthTestRequest()
				r.Header.Set("Authorization", "Bearer "+credential)
				out := httptest.NewRecorder()
				protected.ServeHTTP(out, r)
				if out.Code != want || requests != beforeRequests+1 || (want != 200 && calls != beforeCalls) {
					t.Fatalf("lifecycle: status=%d requests=%d calls=%d body=%s", out.Code, requests, calls, out.Body.String())
				}
			}
			check(alice, 200)
			check(alice, 200)
			active = false
			check(alice, 401)
			active, lookupError = true, errors.New("private store outage")
			check(alice, 503)
			lookupError = nil
			check(alice, 200)
			req := CredentialRequest{Mode: mode}
			if mode == ModeRemoteSession {
				req.SessionID = alice
			} else {
				req.TokenValue = alice
			}
			if err := app.Revoke(context.Background(), req); err != nil {
				t.Fatal(err)
			}
			check(alice, 401)
			alice = issue("alice")
			check(alice, 200)
			logoutURL, err := app.SignoutURL("alice", nil)
			if err != nil {
				t.Fatal(err)
			}
			out := httptest.NewRecorder()
			h.HandleLogout(out, httptest.NewRequest("GET", logoutURL, nil))
			if out.Code != http.StatusOK {
				t.Fatalf("center logout: %d %s", out.Code, out.Body.String())
			}
			check(alice, 401)
			check(bob, 200)
			check(issue("alice"), 200)
		})
	}
}

// TestRemoteAuthCancellation rejects expired resolution and late transport success without retrying. TestRemoteAuthCancellation 拒绝超时解析与迟到的传输成功，且不重试。
func TestRemoteAuthCancellation(t *testing.T) {
	for _, stage := range []string{"resolver", "transport", "late success", "already canceled"} {
		t.Run(stage, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			requests, resolutions := 0, 0
			var transportContext context.Context
			app := newRemoteAuthTestApp(logoutReviewTransport(func(r *http.Request) (*http.Response, error) {
				requests++
				transportContext = r.Context()
				if stage == "late success" {
					cancel()
					out := httptest.NewRecorder()
					writeJSON(out, 200, OKResponse(CredentialInfo{Active: true, ClientID: "app", Mode: ModeRemoteSession, LoginID: "alice", ExpiresIn: 60}))
					return out.Result(), nil
				}
				<-r.Context().Done()
				return nil, r.Context().Err()
			}))
			opts := RemoteAuthOptions{Timeout: 20 * time.Millisecond, ResolveCredential: func(r *http.Request) (string, error) {
				resolutions++
				if stage == "resolver" {
					<-r.Context().Done()
				}
				return "opaque", nil
			}}
			if stage == "late success" {
				opts.Timeout = time.Second
			}
			a, err := NewRemoteAuth(app, opts)
			if err != nil {
				t.Fatal(err)
			}
			if stage == "already canceled" {
				cancel()
			}
			info, err := a.Authenticate(remoteAuthTestRequest().WithContext(ctx))
			cause := context.DeadlineExceeded
			if stage == "already canceled" || stage == "late success" {
				cause = context.Canceled
			}
			if info != nil || !errors.Is(err, ErrRemoteUnavailable) || !errors.Is(err, cause) {
				t.Fatalf("cancellation lost: %v", err)
			}
			if requests > 1 || (stage == "resolver" && requests != 0) || (stage == "already canceled" && resolutions != 0) {
				t.Fatal("canceled operation continued or retried")
			}
			if transportContext != nil && transportContext.Err() == nil {
				t.Fatal("outbound context not canceled")
			}
		})
	}
}

// TestRemoteAuthTransportSafety preserves response limits and prevents implicit redirect forwarding. TestRemoteAuthTransportSafety 保留响应大小限制并阻止隐式重定向转发。
func TestRemoteAuthTransportSafety(t *testing.T) {
	for _, redirect := range []bool{false, true} {
		requests := 0
		app := newRemoteAuthTestApp(logoutReviewTransport(func(*http.Request) (*http.Response, error) {
			requests++
			out := httptest.NewRecorder()
			if redirect {
				out.Header().Set("Location", "https://other.example/collect")
				out.WriteHeader(http.StatusTemporaryRedirect)
			} else {
				_, _ = out.WriteString(strings.Repeat("x", 65))
			}
			return out.Result(), nil
		}))
		app.config.MaxResponseBytes = 64
		a, err := NewRemoteAuth(app, RemoteAuthOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := a.Authenticate(remoteAuthTestRequest()); !errors.Is(err, ErrRemoteUnavailable) || requests != 1 {
			t.Fatalf("response or redirect accepted: %v, requests=%d", err, requests)
		}
		if app.config.HTTPClient.CheckRedirect != nil {
			t.Fatal("borrowed client was modified")
		}
	}
}
