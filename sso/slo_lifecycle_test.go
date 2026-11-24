package sso

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
)

// TestExchangeRegistersLogoutCallback verifies direct exchanges register only authenticated identities. TestExchangeRegistersLogoutCallback 验证直接兑换仅登记认证成功的身份。
func TestExchangeRegistersLogoutCallback(t *testing.T) {
	for _, mode := range []Mode{ModeTicket, ModeOAuth2} {
		for _, scenario := range []string{"success", "invalid callback", "invalid secret", "wrong client", "wrong redirect", "limit", "update at limit", "disabled", "opt out"} {
			t.Run(string(mode)+"/"+scenario, func(t *testing.T) {
				ctx := context.Background()
				s := NewServer()
				defer s.Close()
				client := newTestClient()
				client.Modes = []Mode{mode}
				if err := s.RegisterClient(client); err != nil {
					t.Fatal(err)
				}
				if scenario == "limit" || scenario == "update at limit" {
					existing := *client
					if scenario == "limit" {
						existing.ClientID = "other-app"
					}
					if err := s.RegisterClient(&existing); err != nil {
						t.Fatal(err)
					}
					if _, err := s.RegisterClientSession(ctx, "user", existing.ClientID, existing.RedirectURIs[0]); err != nil {
						t.Fatal(err)
					}
				}
				params := ParamNames{Callback: "logoutTarget", Client: "app", Sign: "signature"}
				h := NewHTTPServer(s, HTTPOptions{ServerOptions: ServerOptions{
					EnableSLO: scenario != "disabled", MaxRegisteredClient: 1,
					CheckSign: true, SecretKey: "sign-secret", Params: params,
				}})
				req := CredentialRequest{Mode: mode, RedirectURI: client.RedirectURIs[0]}
				if mode == ModeTicket {
					ticket, err := s.GenerateTicket(ctx, client.ClientID, "user", req.RedirectURI, nil, nil)
					if err != nil {
						t.Fatal(err)
					}
					req.Ticket = ticket.Ticket
				} else {
					code, err := s.GenerateOAuth2Code(ctx, client.ClientID, "user", req.RedirectURI, nil, nil)
					if err != nil {
						t.Fatal(err)
					}
					req.Code = code.Code
				}
				cfg := ClientConfig{
					ClientID: client.ClientID, ClientSecret: client.ClientSecret,
					ServerURL: "https://sso.example.com", CheckSign: true, SecretKey: "sign-secret", Params: params,
					RegisterCallback: scenario != "opt out", LogoutCallbackURL: "https://app.example.com/logout",
					HTTPClient: &http.Client{Transport: logoutReviewTransport(func(r *http.Request) (*http.Response, error) {
						w := httptest.NewRecorder()
						h.HandleToken(w, r)
						return w.Result(), nil
					})},
				}
				switch scenario {
				case "invalid callback":
					cfg.LogoutCallbackURL = "https://unregistered.example.com/logout"
				case "invalid secret":
					cfg.ClientSecret = "wrong"
				case "wrong client":
					other := *client
					other.ClientID = "other-app"
					if err := s.RegisterClient(&other); err != nil {
						t.Fatal(err)
					}
					cfg.ClientID = other.ClientID
				case "wrong redirect":
					req.RedirectURI = "https://app.example.com/wrong"
				}
				app := NewClientApp(cfg)
				result, err := app.ExchangeCredential(ctx, req)
				wantError := scenario == "invalid callback" || scenario == "invalid secret" || scenario == "wrong client" || scenario == "wrong redirect" || scenario == "limit"
				if (err != nil) != wantError {
					t.Fatalf("exchange = %+v, %v; want error = %v", result, err, wantError)
				}
				if !wantError && (result == nil || result.LoginID != "user") {
					t.Fatalf("unexpected exchange identity: %+v", result)
				}
				sessions, err := s.GetClientSessions(ctx, "user")
				if err != nil {
					t.Fatal(err)
				}
				registered := !wantError && scenario != "disabled" && scenario != "opt out"
				wantCount := 0
				if registered || scenario == "limit" {
					wantCount = 1
				}
				if len(sessions) != wantCount {
					t.Fatalf("sessions = %+v; want %d", sessions, wantCount)
				}
				if registered && (sessions[0].ClientID != client.ClientID || sessions[0].LogoutCallbackURL != cfg.LogoutCallbackURL) {
					t.Fatalf("incorrect binding: %+v", sessions[0])
				}

				// Retry with valid inputs: rejected requests must leave the credential available. 使用有效参数重试：被拒绝的请求必须保留凭证。
				if mode == ModeTicket {
					_, err = s.ConsumeTicket(ctx, req.Ticket, client.ClientID, client.ClientSecret, client.RedirectURIs[0])
				} else {
					_, err = s.ConsumeOAuth2Code(ctx, req.Code, client.ClientID, client.ClientSecret, client.RedirectURIs[0])
				}
				if (err == nil) != wantError {
					t.Fatalf("credential retry error = %v; first request rejected = %v", err, wantError)
				}
			})
		}
	}
}

// TestCenterLogoutLifecycle verifies ordering, trusted identity, and retry behavior. TestCenterLogoutLifecycle 验证注销顺序、可信身份和重试行为。
func TestCenterLogoutLifecycle(t *testing.T) {
	for _, scenario := range []string{"success", "center failure", "best effort center failure", "callback failure", "best effort", "disabled", "mismatched identity", "unauthenticated"} {
		t.Run(scenario, func(t *testing.T) {
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
			var notifications atomic.Int32
			centerCalls := 0
			active := true
			centerFails := scenario == "center failure" || scenario == "best effort center failure"
			h := NewHTTPServer(s, HTTPOptions{
				ServerOptions: ServerOptions{
					EnableSLO: scenario != "disabled", LogoutCallbackBestEffort: strings.HasPrefix(scenario, "best effort"),
					LogoutHTTPClient: &http.Client{Transport: logoutReviewTransport(func(*http.Request) (*http.Response, error) {
						notifications.Add(1)
						status := http.StatusOK
						if scenario == "callback failure" || scenario == "best effort" {
							status = http.StatusServiceUnavailable
						}
						return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(""))}, nil
					})},
				},
				LoginIDResolver: func(*http.Request) (string, bool) { return "user", active && scenario != "unauthenticated" },
				CenterLogout: func(r *http.Request, loginID string) error {
					centerCalls++
					if loginID != "user" || r.Context() == nil {
						t.Fatalf("invalid center logout identity: %q", loginID)
					}
					if scenario != "disabled" && notifications.Load() == 0 {
						t.Fatal("center logout ran before notifications")
					}
					if centerFails {
						return errors.New("center unavailable")
					}
					active = false
					return nil
				},
			})
			values := url.Values{}
			if scenario == "mismatched identity" {
				values.Set("loginId", "another-user")
			}
			w := protocolReviewPost(h.HandleLogout, values)
			wantStatus := http.StatusOK
			switch scenario {
			case "center failure", "best effort center failure", "callback failure":
				wantStatus = http.StatusInternalServerError
			case "mismatched identity", "unauthenticated":
				wantStatus = http.StatusUnauthorized
			}
			if w.Code != wantStatus {
				t.Fatalf("status = %d; want %d: %s", w.Code, wantStatus, w.Body.String())
			}
			wantCenterCalls := 1
			if scenario == "callback failure" || wantStatus == http.StatusUnauthorized {
				wantCenterCalls = 0
			}
			if centerCalls != wantCenterCalls || active != (wantStatus != http.StatusOK) {
				t.Fatalf("center calls = %d, active = %v", centerCalls, active)
			}
			if (scenario == "disabled" || wantStatus == http.StatusUnauthorized) && notifications.Load() != 0 {
				t.Fatal("unexpected client notification")
			}
			sessions, err := s.GetClientSessions(ctx, "user")
			cleared := wantStatus == http.StatusOK && scenario != "disabled"
			if err != nil || (len(sessions) == 0) != cleared {
				t.Fatalf("sessions = %+v, %v; cleared = %v", sessions, err, cleared)
			}
			if (len(w.Result().Cookies()) != 0) != (wantStatus == http.StatusOK) {
				t.Fatal("cookie cleanup did not match logout result")
			}
			if centerFails {
				centerFails = false
				retry := protocolReviewPost(h.HandleLogout, nil)
				sessions, err = s.GetClientSessions(ctx, "user")
				if retry.Code != http.StatusOK || active || centerCalls != 2 || notifications.Load() != 2 || err != nil || len(sessions) != 0 {
					t.Fatalf("retry failed: status = %d, sessions = %+v, error = %v", retry.Code, sessions, err)
				}
			}
		})
	}
}
