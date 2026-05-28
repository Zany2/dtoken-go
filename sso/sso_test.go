package sso

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestSSONewServerDefaults(t *testing.T) {
	ctx := context.Background()
	server := NewServer()

	client := newTestClient()
	if err := server.RegisterClient(client); err != nil {
		t.Fatalf("RegisterClient() error = %v", err)
	}

	ticket, err := server.GenerateTicket(ctx, client.ClientID, "user-1001", client.RedirectURIs[0], nil, nil)
	if err != nil {
		t.Fatalf("GenerateTicket() error = %v", err)
	}
	if ticket.Ticket == "" {
		t.Fatal("GenerateTicket() returned empty ticket")
	}

	info, err := server.ConsumeTicket(ctx, ticket.Ticket, client.ClientID, client.ClientSecret, client.RedirectURIs[0])
	if err != nil {
		t.Fatalf("ConsumeTicket() error = %v", err)
	}
	if info.LoginID != "user-1001" {
		t.Fatalf("ConsumeTicket() loginID = %q, want user-1001", info.LoginID)
	}
}

func TestSSOClientLifecycle(t *testing.T) {
	server := newTestServer()

	client := newTestClient()
	if err := server.RegisterClient(client); err != nil {
		t.Fatalf("RegisterClient() error = %v", err)
	}

	got, err := server.GetClient(client.ClientID)
	if err != nil {
		t.Fatalf("GetClient() error = %v", err)
	}
	if got.ClientID != client.ClientID || got.ClientSecret != client.ClientSecret {
		t.Fatalf("GetClient() = %+v, want client id and secret preserved", got)
	}

	if err = server.UnregisterClient(client.ClientID); err != nil {
		t.Fatalf("UnregisterClient() error = %v", err)
	}
	if _, err = server.GetClient(client.ClientID); !errors.Is(err, ErrClientNotFound) {
		t.Fatalf("GetClient() after unregister error = %v, want ErrClientNotFound", err)
	}
}

func TestSSOTicketGenerateValidateAndConsume(t *testing.T) {
	ctx := context.Background()
	server := newTestServer()
	registerTestClient(t, server)

	ticket, err := server.GenerateTicket(ctx, "app-a", "user-1001", "https://app.example.com/sso/callback", []string{"profile"}, map[string]any{"scene": "web"})
	if err != nil {
		t.Fatalf("GenerateTicket() error = %v", err)
	}
	if ticket.Ticket == "" || ticket.Mode != ModeTicket || ticket.LoginID != "user-1001" {
		t.Fatalf("GenerateTicket() = %+v, want valid ticket data", ticket)
	}

	validated, err := server.ValidateTicket(ctx, ticket.Ticket)
	if err != nil {
		t.Fatalf("ValidateTicket() error = %v", err)
	}
	if validated.Ticket != ticket.Ticket {
		t.Fatalf("ValidateTicket() ticket = %q, want %q", validated.Ticket, ticket.Ticket)
	}

	consumed, err := server.ConsumeTicket(ctx, ticket.Ticket, "app-a", "secret-a", "https://app.example.com/sso/callback")
	if err != nil {
		t.Fatalf("ConsumeTicket() error = %v", err)
	}
	if !consumed.Used || consumed.LoginID != "user-1001" {
		t.Fatalf("ConsumeTicket() = %+v, want used ticket for user-1001", consumed)
	}

	if _, err = server.ConsumeTicket(ctx, ticket.Ticket, "app-a", "secret-a", "https://app.example.com/sso/callback"); !errors.Is(err, ErrInvalidTicket) {
		t.Fatalf("ConsumeTicket() second error = %v, want ErrInvalidTicket", err)
	}
}

func TestSSOTicketErrorBoundaries(t *testing.T) {
	ctx := context.Background()
	server := newTestServer()
	registerTestClient(t, server)

	if _, err := server.GenerateTicket(ctx, "app-a", "", "https://app.example.com/sso/callback", nil, nil); !errors.Is(err, ErrUserIDEmpty) {
		t.Fatalf("GenerateTicket() empty login error = %v, want ErrUserIDEmpty", err)
	}
	if _, err := server.GenerateTicket(ctx, "app-a", "user-1001", "https://evil.example.com/callback", nil, nil); !errors.Is(err, ErrInvalidRedirectURI) {
		t.Fatalf("GenerateTicket() redirect error = %v, want ErrInvalidRedirectURI", err)
	}
	if _, err := server.GenerateTicket(ctx, "app-a", "user-1001", "https://app.example.com/sso/callback", []string{"admin"}, nil); !errors.Is(err, ErrInvalidScope) {
		t.Fatalf("GenerateTicket() scope error = %v, want ErrInvalidScope", err)
	}

	ticket, err := server.GenerateTicket(ctx, "app-a", "user-1001", "https://app.example.com/sso/callback", nil, nil)
	if err != nil {
		t.Fatalf("GenerateTicket() error = %v", err)
	}
	if _, err = server.ConsumeTicket(ctx, ticket.Ticket, "app-a", "bad-secret", "https://app.example.com/sso/callback"); !errors.Is(err, ErrInvalidClientCredentials) {
		t.Fatalf("ConsumeTicket() secret error = %v, want ErrInvalidClientCredentials", err)
	}
	if _, err = server.ConsumeTicket(ctx, ticket.Ticket, "app-a", "secret-a", "https://other.example.com/callback"); !errors.Is(err, ErrRedirectURIMismatch) {
		t.Fatalf("ConsumeTicket() redirect mismatch error = %v, want ErrRedirectURIMismatch", err)
	}
}

func TestSSOTicketTTLRevokeAndExpire(t *testing.T) {
	ctx := context.Background()
	server := newTestServer()
	registerTestClient(t, server)

	ticket, err := server.GenerateTicketWithTimeout(ctx, "app-a", "user-1001", "https://app.example.com/sso/callback", nil, nil, time.Second)
	if err != nil {
		t.Fatalf("GenerateTicketWithTimeout() error = %v", err)
	}

	ttl, err := server.GetTicketTTL(ctx, ticket.Ticket)
	if err != nil {
		t.Fatalf("GetTicketTTL() error = %v", err)
	}
	if ttl < 0 || ttl > 1 {
		t.Fatalf("GetTicketTTL() = %d, want 0..1", ttl)
	}

	time.Sleep(1100 * time.Millisecond)
	if _, err = server.ValidateTicket(ctx, ticket.Ticket); !errors.Is(err, ErrInvalidTicket) {
		t.Fatalf("ValidateTicket() expired error = %v, want ErrInvalidTicket", err)
	}

	ticket, err = server.GenerateTicket(ctx, "app-a", "user-1001", "https://app.example.com/sso/callback", nil, nil)
	if err != nil {
		t.Fatalf("GenerateTicket() error = %v", err)
	}
	if err = server.RevokeTicket(ctx, ticket.Ticket); err != nil {
		t.Fatalf("RevokeTicket() error = %v", err)
	}
	ttl, err = server.GetTicketTTL(ctx, ticket.Ticket)
	if err != nil {
		t.Fatalf("GetTicketTTL() after revoke error = %v", err)
	}
	if ttl != -2 {
		t.Fatalf("GetTicketTTL() after revoke = %d, want -2", ttl)
	}
}

func TestSSOModeCompatibility(t *testing.T) {
	ctx := context.Background()
	server := newTestServer()

	defaultClient := newTestClient()
	defaultClient.ClientID = "default-app"
	defaultClient.Modes = nil
	if err := server.RegisterClient(defaultClient); err != nil {
		t.Fatalf("RegisterClient() default error = %v", err)
	}
	if _, err := server.GenerateTicket(ctx, defaultClient.ClientID, "user-1001", "https://app.example.com/sso/callback", nil, nil); err != nil {
		t.Fatalf("GenerateTicket() with empty modes error = %v, want nil", err)
	}
	if _, err := server.GenerateSharedToken(ctx, defaultClient.ClientID, "user-1001", nil, nil); !errors.Is(err, ErrModeUnsupported) {
		t.Fatalf("GenerateSharedToken() with empty modes error = %v, want ErrModeUnsupported", err)
	}

	client := newTestClient()
	client.Modes = []Mode{ModeSharedToken}
	if err := server.RegisterClient(client); err != nil {
		t.Fatalf("RegisterClient() error = %v", err)
	}

	if _, err := server.GenerateTicket(ctx, client.ClientID, "user-1001", "https://app.example.com/sso/callback", nil, nil); !errors.Is(err, ErrModeUnsupported) {
		t.Fatalf("GenerateTicket() unsupported mode error = %v, want ErrModeUnsupported", err)
	}
}

func TestSSOSharedTokenFlow(t *testing.T) {
	ctx := context.Background()
	server := newTestServer()
	client := newTestClient()
	client.Modes = []Mode{ModeSharedToken}
	if err := server.RegisterClient(client); err != nil {
		t.Fatalf("RegisterClient() error = %v", err)
	}

	token, err := server.GenerateSharedToken(ctx, "app-a", "user-1001", []string{"profile"}, map[string]any{"scene": "shared"})
	if err != nil {
		t.Fatalf("GenerateSharedToken() error = %v", err)
	}
	if token.Token == "" || token.Mode != ModeSharedToken || token.LoginID != "user-1001" {
		t.Fatalf("GenerateSharedToken() = %+v, want valid shared token", token)
	}

	validated, err := server.ValidateSharedToken(ctx, token.Token, "app-a")
	if err != nil {
		t.Fatalf("ValidateSharedToken() error = %v", err)
	}
	if validated.Token != token.Token || validated.LoginID != "user-1001" {
		t.Fatalf("ValidateSharedToken() = %+v, want original token info", validated)
	}
	if _, err = server.ValidateSharedToken(ctx, token.Token, "app-b"); !errors.Is(err, ErrClientMismatch) {
		t.Fatalf("ValidateSharedToken() client mismatch error = %v, want ErrClientMismatch", err)
	}

	if err = server.RevokeSharedToken(ctx, token.Token); err != nil {
		t.Fatalf("RevokeSharedToken() error = %v", err)
	}
	if _, err = server.ValidateSharedToken(ctx, token.Token, "app-a"); !errors.Is(err, ErrInvalidSharedToken) {
		t.Fatalf("ValidateSharedToken() after revoke error = %v, want ErrInvalidSharedToken", err)
	}
}

func TestSSOSharedTokenTTLAndBoundaries(t *testing.T) {
	ctx := context.Background()
	server := newTestServer()
	client := newTestClient()
	client.Modes = []Mode{ModeSharedToken}
	if err := server.RegisterClient(client); err != nil {
		t.Fatalf("RegisterClient() error = %v", err)
	}

	if _, err := server.GenerateSharedToken(ctx, "app-a", "", nil, nil); !errors.Is(err, ErrUserIDEmpty) {
		t.Fatalf("GenerateSharedToken() empty login error = %v, want ErrUserIDEmpty", err)
	}
	if _, err := server.GenerateSharedToken(ctx, "app-a", "user-1001", []string{"admin"}, nil); !errors.Is(err, ErrInvalidScope) {
		t.Fatalf("GenerateSharedToken() scope error = %v, want ErrInvalidScope", err)
	}

	token, err := server.GenerateSharedTokenWithTimeout(ctx, "app-a", "user-1001", nil, nil, time.Second)
	if err != nil {
		t.Fatalf("GenerateSharedTokenWithTimeout() error = %v", err)
	}
	ttl, err := server.GetSharedTokenTTL(ctx, token.Token)
	if err != nil {
		t.Fatalf("GetSharedTokenTTL() error = %v", err)
	}
	if ttl < 0 || ttl > 1 {
		t.Fatalf("GetSharedTokenTTL() = %d, want 0..1", ttl)
	}

	time.Sleep(1100 * time.Millisecond)
	if _, err = server.ValidateSharedToken(ctx, token.Token, "app-a"); !errors.Is(err, ErrInvalidSharedToken) {
		t.Fatalf("ValidateSharedToken() expired error = %v, want ErrInvalidSharedToken", err)
	}
}

func TestSSORemoteSessionFlow(t *testing.T) {
	ctx := context.Background()
	server := newTestServer()
	client := newTestClient()
	client.Modes = []Mode{ModeRemoteSession}
	if err := server.RegisterClient(client); err != nil {
		t.Fatalf("RegisterClient() error = %v", err)
	}

	session, err := server.CreateRemoteSessionWithTimeout(ctx, "app-a", "user-1001", []string{"profile"}, nil, time.Second)
	if err != nil {
		t.Fatalf("CreateRemoteSessionWithTimeout() error = %v", err)
	}
	if session.SessionID == "" || session.Mode != ModeRemoteSession || session.LoginID != "user-1001" {
		t.Fatalf("CreateRemoteSessionWithTimeout() = %+v, want valid remote session", session)
	}

	validated, err := server.ValidateRemoteSession(ctx, session.SessionID, "app-a")
	if err != nil {
		t.Fatalf("ValidateRemoteSession() error = %v", err)
	}
	if validated.SessionID != session.SessionID {
		t.Fatalf("ValidateRemoteSession() sessionID = %q, want %q", validated.SessionID, session.SessionID)
	}

	if err = server.RenewRemoteSession(ctx, session.SessionID, 2*time.Second); err != nil {
		t.Fatalf("RenewRemoteSession() error = %v", err)
	}
	ttl, err := server.GetRemoteSessionTTL(ctx, session.SessionID)
	if err != nil {
		t.Fatalf("GetRemoteSessionTTL() error = %v", err)
	}
	if ttl < 0 || ttl > 2 {
		t.Fatalf("GetRemoteSessionTTL() = %d, want 0..2", ttl)
	}

	if err = server.RevokeRemoteSession(ctx, session.SessionID); err != nil {
		t.Fatalf("RevokeRemoteSession() error = %v", err)
	}
	if _, err = server.ValidateRemoteSession(ctx, session.SessionID, "app-a"); !errors.Is(err, ErrInvalidRemoteSession) {
		t.Fatalf("ValidateRemoteSession() after revoke error = %v, want ErrInvalidRemoteSession", err)
	}
}

func TestSSORemoteSessionBoundaries(t *testing.T) {
	ctx := context.Background()
	server := newTestServer()
	client := newTestClient()
	client.Modes = []Mode{ModeRemoteSession}
	if err := server.RegisterClient(client); err != nil {
		t.Fatalf("RegisterClient() error = %v", err)
	}

	if _, err := server.CreateRemoteSession(ctx, "app-a", "", nil, nil); !errors.Is(err, ErrUserIDEmpty) {
		t.Fatalf("CreateRemoteSession() empty login error = %v, want ErrUserIDEmpty", err)
	}
	if _, err := server.CreateRemoteSession(ctx, "app-a", "user-1001", []string{"admin"}, nil); !errors.Is(err, ErrInvalidScope) {
		t.Fatalf("CreateRemoteSession() scope error = %v, want ErrInvalidScope", err)
	}
	if err := server.RenewRemoteSession(ctx, "", time.Second); !errors.Is(err, ErrInvalidRemoteSession) {
		t.Fatalf("RenewRemoteSession() empty session error = %v, want ErrInvalidRemoteSession", err)
	}

	session, err := server.CreateRemoteSessionWithTimeout(ctx, "app-a", "user-1001", nil, nil, time.Second)
	if err != nil {
		t.Fatalf("CreateRemoteSessionWithTimeout() error = %v", err)
	}
	if _, err = server.ValidateRemoteSession(ctx, session.SessionID, "app-b"); !errors.Is(err, ErrClientMismatch) {
		t.Fatalf("ValidateRemoteSession() client mismatch error = %v, want ErrClientMismatch", err)
	}

	time.Sleep(1100 * time.Millisecond)
	if _, err = server.ValidateRemoteSession(ctx, session.SessionID, "app-a"); !errors.Is(err, ErrInvalidRemoteSession) {
		t.Fatalf("ValidateRemoteSession() expired error = %v, want ErrInvalidRemoteSession", err)
	}
}

func TestSSOOAuth2CodeFlow(t *testing.T) {
	ctx := context.Background()
	server := newTestServer()
	client := newTestClient()
	client.Modes = []Mode{ModeOAuth2}
	if err := server.RegisterClient(client); err != nil {
		t.Fatalf("RegisterClient() error = %v", err)
	}

	code, err := server.GenerateOAuth2Code(ctx, "app-a", "user-1001", "https://app.example.com/sso/callback", []string{"profile"}, nil)
	if err != nil {
		t.Fatalf("GenerateOAuth2Code() error = %v", err)
	}
	if code.Code == "" || code.Mode != ModeOAuth2 || code.LoginID != "user-1001" {
		t.Fatalf("GenerateOAuth2Code() = %+v, want valid OAuth2 code", code)
	}

	consumed, err := server.ConsumeOAuth2Code(ctx, code.Code, "app-a", "secret-a", "https://app.example.com/sso/callback")
	if err != nil {
		t.Fatalf("ConsumeOAuth2Code() error = %v", err)
	}
	if !consumed.Used || consumed.LoginID != "user-1001" {
		t.Fatalf("ConsumeOAuth2Code() = %+v, want used OAuth2 code", consumed)
	}
	if _, err = server.ConsumeOAuth2Code(ctx, code.Code, "app-a", "secret-a", "https://app.example.com/sso/callback"); !errors.Is(err, ErrInvalidOAuth2Code) {
		t.Fatalf("ConsumeOAuth2Code() second error = %v, want ErrInvalidOAuth2Code", err)
	}
}

func TestSSOOAuth2CodeBoundaries(t *testing.T) {
	ctx := context.Background()
	server := newTestServer()
	client := newTestClient()
	client.Modes = []Mode{ModeOAuth2}
	if err := server.RegisterClient(client); err != nil {
		t.Fatalf("RegisterClient() error = %v", err)
	}

	if _, err := server.GenerateOAuth2Code(ctx, "app-a", "", "https://app.example.com/sso/callback", nil, nil); !errors.Is(err, ErrUserIDEmpty) {
		t.Fatalf("GenerateOAuth2Code() empty login error = %v, want ErrUserIDEmpty", err)
	}
	if _, err := server.GenerateOAuth2Code(ctx, "app-a", "user-1001", "https://evil.example.com/callback", nil, nil); !errors.Is(err, ErrInvalidRedirectURI) {
		t.Fatalf("GenerateOAuth2Code() redirect error = %v, want ErrInvalidRedirectURI", err)
	}
	if _, err := server.GenerateOAuth2Code(ctx, "app-a", "user-1001", "https://app.example.com/sso/callback", []string{"admin"}, nil); !errors.Is(err, ErrInvalidScope) {
		t.Fatalf("GenerateOAuth2Code() scope error = %v, want ErrInvalidScope", err)
	}

	code, err := server.GenerateOAuth2Code(ctx, "app-a", "user-1001", "https://app.example.com/sso/callback", nil, nil)
	if err != nil {
		t.Fatalf("GenerateOAuth2Code() error = %v", err)
	}
	if _, err = server.ConsumeOAuth2Code(ctx, code.Code, "app-a", "bad-secret", "https://app.example.com/sso/callback"); !errors.Is(err, ErrInvalidClientCredentials) {
		t.Fatalf("ConsumeOAuth2Code() secret error = %v, want ErrInvalidClientCredentials", err)
	}
	if _, err = server.ConsumeOAuth2Code(ctx, code.Code, "app-a", "secret-a", "https://other.example.com/callback"); !errors.Is(err, ErrRedirectURIMismatch) {
		t.Fatalf("ConsumeOAuth2Code() redirect mismatch error = %v, want ErrRedirectURIMismatch", err)
	}

	expiringCode, err := server.GenerateOAuth2CodeWithTimeout(ctx, "app-a", "user-1001", "https://app.example.com/sso/callback", nil, nil, time.Second)
	if err != nil {
		t.Fatalf("GenerateOAuth2CodeWithTimeout() error = %v", err)
	}
	time.Sleep(1100 * time.Millisecond)
	if _, err = server.ConsumeOAuth2Code(ctx, expiringCode.Code, "app-a", "secret-a", "https://app.example.com/sso/callback"); !errors.Is(err, ErrInvalidOAuth2Code) {
		t.Fatalf("ConsumeOAuth2Code() expired error = %v, want ErrInvalidOAuth2Code", err)
	}
}

func newTestServer() *Server {
	return NewServer(
		WithAuthType("login:"),
		WithKeyPrefix("dtoken:"),
		WithConfig(&Config{
			TicketExpiration:        time.Minute,
			SharedTokenExpiration:   time.Minute,
			RemoteSessionExpiration: time.Minute,
			OAuth2CodeExpiration:    time.Minute,
		}),
	)
}

func newTestClient() *Client {
	return &Client{
		ClientID:     "app-a",
		ClientSecret: "secret-a",
		Name:         "App A",
		RedirectURIs: []string{"https://app.example.com/sso/callback"},
		Modes:        []Mode{ModeTicket},
		Scopes:       []string{"profile", "email"},
	}
}

func registerTestClient(t *testing.T, server *Server) {
	t.Helper()
	if err := server.RegisterClient(newTestClient()); err != nil {
		t.Fatalf("RegisterClient() error = %v", err)
	}
}
