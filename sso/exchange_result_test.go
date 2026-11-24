package sso

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestExchangeResultEnrichment verifies trusted metadata, immutable authorization, and one-time failure semantics. TestExchangeResultEnrichment 验证可信元数据、不可变授权和失败时的一次性消费语义。
func TestExchangeResultEnrichment(t *testing.T) {
	type contextKey struct{}
	for _, mode := range []Mode{ModeTicket, ModeOAuth2} {
		for _, scenario := range []string{"enrich", "without callback", "nil hook", "hook error", "changed login", "changed center", "changed scopes", "invalid token ttl", "invalid session ttl", "invalid secret", "wrong redirect"} {
			t.Run(string(mode)+"/"+scenario, func(t *testing.T) {
				ctx := context.WithValue(context.Background(), contextKey{}, "exchange-context")
				s := NewServer()
				defer s.Close()
				client := newTestClient()
				client.Modes = []Mode{mode}
				if err := s.RegisterClient(client); err != nil {
					t.Fatal(err)
				}
				req := CredentialRequest{Mode: mode, RedirectURI: client.RedirectURIs[0]}
				extra := map[string]any{"sessionRef": "center-session-1", "existing": "preserved"}
				if mode == ModeTicket {
					ticket, err := s.GenerateTicket(ctx, client.ClientID, "user", req.RedirectURI, []string{"profile"}, extra)
					if err != nil {
						t.Fatal(err)
					}
					req.Ticket = ticket.Ticket
				} else {
					code, err := s.GenerateOAuth2Code(ctx, client.ClientID, "user", req.RedirectURI, []string{"profile"}, extra)
					if err != nil {
						t.Fatal(err)
					}
					req.Code = code.Code
				}
				calls := 0
				options := HTTPOptions{ServerOptions: ServerOptions{EnableSLO: true}}
				if scenario != "nil hook" {
					options.EnrichExchangeResult = func(r *http.Request, clientID string, result *TicketExchangeResult) error {
						calls++
						if clientID != client.ClientID || result.LoginID != "user" || result.CenterID != "user" || len(result.Scopes) != 1 || result.Scopes[0] != "profile" || result.Extra["sessionRef"] != "center-session-1" {
							t.Fatalf("incorrect verified input: client=%s result=%+v", clientID, result)
						}
						if r.FormValue("loginId") != "forged-user" || r.Context().Value(contextKey{}) != "exchange-context" {
							t.Fatal("hook did not receive the exchange request and context")
						}
						var err error
						if mode == ModeTicket {
							_, err = s.getTicket(ctx, req.Ticket)
						} else {
							_, err = s.getOAuth2Code(ctx, req.Code)
						}
						if err == nil {
							t.Fatal("hook ran before credential consumption")
						}
						sessions, err := s.GetClientSessions(ctx, "user")
						if err != nil || len(sessions) != 0 {
							t.Fatalf("hook ran after callback registration: %+v, %v", sessions, err)
						}
						result.TokenValue = "center-token"
						result.DeviceID = "browser"
						result.RemainTokenTimeout = 900
						result.RemainSessionTimeout = -1
						result.Extra["displayName"] = "Alice"
						switch scenario {
						case "hook error":
							return errors.New("center session expired")
						case "changed login":
							result.LoginID = "another-user"
						case "changed center":
							result.CenterID = "another-user"
						case "changed scopes":
							result.Scopes[0] = "admin"
						case "invalid token ttl":
							result.RemainTokenTimeout = -2
						case "invalid session ttl":
							result.RemainSessionTimeout = -2
						}
						return nil
					}
				}
				h := NewHTTPServer(s, options)
				var lastResponse *httptest.ResponseRecorder
				cfg := ClientConfig{
					ClientID: client.ClientID, ClientSecret: client.ClientSecret,
					ServerURL: "https://center.example.com", RegisterCallback: scenario != "without callback", LogoutCallbackURL: "https://app.example.com/logout",
					HTTPClient: &http.Client{Transport: logoutReviewTransport(func(r *http.Request) (*http.Response, error) {
						r.URL.RawQuery = "loginId=forged-user&deviceId=forged-device&remainTokenTimeout=999999"
						lastResponse = httptest.NewRecorder()
						h.HandleToken(lastResponse, r)
						return lastResponse.Result(), nil
					})},
				}
				if scenario == "invalid secret" {
					cfg.ClientSecret = "wrong-secret"
				}
				if scenario == "wrong redirect" {
					req.RedirectURI = "https://app.example.com/wrong"
				}
				app := NewClientApp(cfg)
				result, err := app.ExchangeCredential(ctx, req)
				enriched := scenario == "enrich" || scenario == "without callback"
				success := enriched || scenario == "nil hook"
				if (err == nil) != success || !success && result != nil {
					t.Fatalf("exchange = %+v, %v; want success=%v", result, err, success)
				}
				consumed := scenario != "invalid secret" && scenario != "wrong redirect"
				wantCalls := 0
				if consumed && scenario != "nil hook" {
					wantCalls = 1
				}
				if calls != wantCalls {
					t.Fatalf("hook calls = %d; want %d", calls, wantCalls)
				}
				if success && (result.LoginID != "user" || result.CenterID != "user" || len(result.Scopes) != 1 || result.Scopes[0] != "profile" || result.Extra["existing"] != "preserved") {
					t.Fatalf("base result changed: %+v", result)
				}
				if enriched && (result.TokenValue != "center-token" || result.DeviceID != "browser" || result.RemainTokenTimeout != 900 || result.RemainSessionTimeout != -1 || result.Extra["displayName"] != "Alice") {
					t.Fatalf("enriched fields were lost: %+v", result)
				}
				if scenario == "nil hook" && (result.TokenValue != "" || result.DeviceID != "" || result.RemainTokenTimeout != 0 || result.RemainSessionTimeout != 0) {
					t.Fatalf("request values changed the default result: %+v", result)
				}
				if !success && strings.Contains(lastResponse.Body.String(), "center-token") {
					t.Fatal("failed enrichment exposed partial result data")
				}
				sessions, err := s.GetClientSessions(ctx, "user")
				registered := success && scenario != "without callback"
				if err != nil || (len(sessions) == 1) != registered {
					t.Fatalf("callback registrations = %+v, %v", sessions, err)
				}
				if consumed {
					if _, err := app.ExchangeCredential(ctx, req); err == nil || calls != wantCalls {
						t.Fatalf("replay = %v, hook calls=%d", err, calls)
					}
				} else if mode == ModeTicket {
					if _, err := s.ConsumeTicket(ctx, req.Ticket, client.ClientID, client.ClientSecret, client.RedirectURIs[0]); err != nil {
						t.Fatalf("rejected request consumed ticket: %v", err)
					}
				} else if _, err := s.ConsumeOAuth2Code(ctx, req.Code, client.ClientID, client.ClientSecret, client.RedirectURIs[0]); err != nil {
					t.Fatalf("rejected request consumed code: %v", err)
				}
			})
		}
	}
}
