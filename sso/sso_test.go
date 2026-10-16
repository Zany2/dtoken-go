package sso

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestSSONewServerDefaults verifies the SSO New Server Defaults scenario. TestSSONewServerDefaults 验证对应的 SSO 服务端场景。
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

// TestSSONewServerWithConfigFallsBackToBuiltIns verifies the SSO New Server With Config Falls Back To Built Ins scenario. TestSSONewServerWithConfigFallsBackToBuiltIns 验证对应的 SSO 服务端场景。
func TestSSONewServerWithConfigFallsBackToBuiltIns(t *testing.T) {
	ctx := context.Background()
	server := NewServerWithConfig(DefaultAuthType, DefaultKeyPrefix, nil, nil, nil)
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
}

// TestSSOServerCloseOwnership verifies server-owned storage is closed exactly once. TestSSOServerCloseOwnership 验证 Server 仅关闭自有存储且只关闭一次。
func TestSSOServerCloseOwnership(t *testing.T) {
	externalStorage := &closeTrackingStorage{MemoryStorage: NewMemoryStorage()}
	externalServer := NewServer(WithStorage(externalStorage))
	if err := externalServer.Close(); err != nil {
		t.Fatalf("Close() external storage error = %v", err)
	}
	if externalStorage.closeCount != 0 {
		t.Fatalf("Close() external storage count = %d, want 0", externalStorage.closeCount)
	}

	closeErr := errors.New("close failed")
	ownedStorage := &closeTrackingStorage{
		MemoryStorage: NewMemoryStorage(),
		closeErr:      closeErr,
	}
	ownedServer := NewServer(WithStorage(ownedStorage), WithStorageOwnership(true))
	if err := ownedServer.Close(); !errors.Is(err, closeErr) {
		t.Fatalf("Close() owned storage error = %v, want %v", err, closeErr)
	}
	if err := ownedServer.Close(); !errors.Is(err, closeErr) {
		t.Fatalf("Close() owned storage second error = %v, want %v", err, closeErr)
	}
	if ownedStorage.closeCount != 1 {
		t.Fatalf("Close() owned storage count = %d, want 1", ownedStorage.closeCount)
	}
}

// TestSSOClientLifecycle verifies the SSO Client Lifecycle scenario. TestSSOClientLifecycle 验证对应的 SSO 服务端场景。
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

// TestSSOTicketGenerateValidateAndConsume verifies the SSO Ticket Generate Validate And Consume scenario. TestSSOTicketGenerateValidateAndConsume 验证对应的 SSO 服务端场景。
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

// TestSSOTicketErrorBoundaries verifies the SSO Ticket Error Boundaries scenario. TestSSOTicketErrorBoundaries 验证对应的 SSO 服务端场景。
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
	if consumed, err := server.ConsumeTicket(ctx, ticket.Ticket, "app-a", "secret-a", "https://app.example.com/sso/callback"); err != nil || consumed == nil || !consumed.Used {
		t.Fatalf("ConsumeTicket() after rejected request = %+v, %v, want successful consume", consumed, err)
	}
}

// TestSSOTicketTTLRevokeAndExpire verifies the SSO Ticket TTL Revoke And Expire scenario. TestSSOTicketTTLRevokeAndExpire 验证对应的 SSO 服务端场景。
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

// TestSSOModeCompatibility verifies the SSO Mode Compatibility scenario. TestSSOModeCompatibility 验证对应的 SSO 服务端场景。
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

// TestSSOClientSessionFlow verifies the SSO Client Session Flow scenario. TestSSOClientSessionFlow 验证对应的 SSO 服务端场景。
func TestSSOClientSessionFlow(t *testing.T) {
	ctx := context.Background()
	server := newTestServer()
	registerTestClient(t, server)

	session, err := server.RegisterClientSession(ctx, "user-1001", "app-a", "https://app.example.com/sso/logout-callback")
	if err != nil {
		t.Fatalf("RegisterClientSession() error = %v", err)
	}
	if session.LoginID != "user-1001" || session.ClientID != "app-a" || session.LogoutCallbackURL == "" {
		t.Fatalf("RegisterClientSession() = %+v, want user app-a callback", session)
	}

	sessions, err := server.GetClientSessions(ctx, "user-1001")
	if err != nil {
		t.Fatalf("GetClientSessions() error = %v", err)
	}
	if len(sessions) != 1 || sessions[0].ClientID != "app-a" {
		t.Fatalf("GetClientSessions() = %+v, want one app-a session", sessions)
	}

	updated, err := server.RegisterClientSession(ctx, "user-1001", "app-a", "https://app.example.com/sso/logout-callback-2")
	if err != nil {
		t.Fatalf("RegisterClientSession() update error = %v", err)
	}
	if updated.CreateTime != session.CreateTime {
		t.Fatalf("RegisterClientSession() update createTime = %d, want %d", updated.CreateTime, session.CreateTime)
	}

	sessions, err = server.GetClientSessions(ctx, "user-1001")
	if err != nil {
		t.Fatalf("GetClientSessions() after update error = %v", err)
	}
	if len(sessions) != 1 || sessions[0].LogoutCallbackURL != "https://app.example.com/sso/logout-callback-2" {
		t.Fatalf("GetClientSessions() after update = %+v, want one updated session", sessions)
	}

	if _, err = server.RegisterClientSession(ctx, "", "app-a", "https://app.example.com/sso/logout-callback"); !errors.Is(err, ErrUserIDEmpty) {
		t.Fatalf("RegisterClientSession() empty login error = %v, want ErrUserIDEmpty", err)
	}
	if _, err = server.RegisterClientSession(ctx, "user-1001", "", "https://app.example.com/sso/logout-callback"); !errors.Is(err, ErrClientOrClientIDEmpty) {
		t.Fatalf("RegisterClientSession() empty client error = %v, want ErrClientOrClientIDEmpty", err)
	}
	if _, err = server.RegisterClientSession(ctx, "user-1001", "app-a", "https://evil.example.com/sso/logout-callback"); !errors.Is(err, ErrInvalidCallbackURL) {
		t.Fatalf("RegisterClientSession() invalid callback error = %v, want ErrInvalidCallbackURL", err)
	}

	if err = server.ClearClientSessions(ctx, "user-1001"); err != nil {
		t.Fatalf("ClearClientSessions() error = %v", err)
	}
	sessions, err = server.GetClientSessions(ctx, "user-1001")
	if err != nil {
		t.Fatalf("GetClientSessions() after clear error = %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("GetClientSessions() after clear = %+v, want empty", sessions)
	}
}

// TestSSOClientSessionConcurrentRegistration preserves all local index updates. TestSSOClientSessionConcurrentRegistration 验证并发注册不会丢失当前实例内的索引更新。
func TestSSOClientSessionConcurrentRegistration(t *testing.T) {
	ctx := context.Background()
	server := NewServer()
	const clientCount = 32

	for i := 0; i < clientCount; i++ {
		clientID := fmt.Sprintf("app-%d", i)
		if err := server.RegisterClient(&Client{
			ClientID:     clientID,
			RedirectURIs: []string{fmt.Sprintf("https://%s.example.com/callback", clientID)},
			Modes:        []Mode{ModeTicket},
		}); err != nil {
			t.Fatalf("RegisterClient(%q) error = %v", clientID, err)
		}
	}

	var wg sync.WaitGroup
	for i := 0; i < clientCount; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			clientID := fmt.Sprintf("app-%d", i)
			callback := fmt.Sprintf("https://%s.example.com/callback", clientID)
			if _, err := server.RegisterClientSession(ctx, "user-concurrent", clientID, callback); err != nil {
				t.Errorf("RegisterClientSession(%q) error = %v", clientID, err)
			}
		}()
	}
	wg.Wait()

	sessions, err := server.GetClientSessions(ctx, "user-concurrent")
	if err != nil {
		t.Fatalf("GetClientSessions() error = %v", err)
	}
	if len(sessions) != clientCount {
		t.Fatalf("GetClientSessions() count = %d, want %d", len(sessions), clientCount)
	}
	seen := make(map[string]bool, len(sessions))
	for _, session := range sessions {
		seen[session.ClientID] = true
	}
	for i := 0; i < clientCount; i++ {
		clientID := fmt.Sprintf("app-%d", i)
		if !seen[clientID] {
			t.Fatalf("GetClientSessions() missing client %q", clientID)
		}
	}
}

// TestSSOClientSessionAllowOrigin verifies the SSO Client Session Allow Origin scenario. TestSSOClientSessionAllowOrigin 验证对应的 SSO 服务端场景。
func TestSSOClientSessionAllowOrigin(t *testing.T) {
	ctx := context.Background()
	server := newTestServer()
	client := newTestClient()
	client.AllowOrigins = []string{"https://logout.example.com"}
	if err := server.RegisterClient(client); err != nil {
		t.Fatalf("RegisterClient() error = %v", err)
	}

	session, err := server.RegisterClientSession(ctx, "user-1001", "app-a", "https://logout.example.com/sso/logout-callback")
	if err != nil {
		t.Fatalf("RegisterClientSession() allow origin error = %v", err)
	}
	if session.LogoutCallbackURL != "https://logout.example.com/sso/logout-callback" {
		t.Fatalf("RegisterClientSession() callback = %q, want allow-origin callback", session.LogoutCallbackURL)
	}
}

// TestSSOSharedTokenFlow verifies the SSO Shared Token Flow scenario. TestSSOSharedTokenFlow 验证对应的 SSO 服务端场景。
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

// TestSSOSharedTokenTTLAndBoundaries verifies the SSO Shared Token TTL And Boundaries scenario. TestSSOSharedTokenTTLAndBoundaries 验证对应的 SSO 服务端场景。
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

// TestSSORemoteSessionFlow verifies the SSO Remote Session Flow scenario. TestSSORemoteSessionFlow 验证对应的 SSO 服务端场景。
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

	if err = server.RenewRemoteSession(ctx, session.SessionID, 3*time.Second); err != nil {
		t.Fatalf("RenewRemoteSession() error = %v", err)
	}
	ttl, err := server.GetRemoteSessionTTL(ctx, session.SessionID)
	if err != nil {
		t.Fatalf("GetRemoteSessionTTL() error = %v", err)
	}
	if ttl < 0 || ttl > 3 {
		t.Fatalf("GetRemoteSessionTTL() = %d, want 0..3", ttl)
	}
	time.Sleep(1100 * time.Millisecond)
	if validated, err = server.ValidateRemoteSession(ctx, session.SessionID, "app-a"); err != nil || validated.ExpiresIn != 3 {
		t.Fatalf("ValidateRemoteSession() after original deadline = %+v, %v, want renewed session", validated, err)
	}

	if err = server.RevokeRemoteSession(ctx, session.SessionID); err != nil {
		t.Fatalf("RevokeRemoteSession() error = %v", err)
	}
	if _, err = server.ValidateRemoteSession(ctx, session.SessionID, "app-a"); !errors.Is(err, ErrInvalidRemoteSession) {
		t.Fatalf("ValidateRemoteSession() after revoke error = %v, want ErrInvalidRemoteSession", err)
	}
}

// TestSSORemoteSessionBoundaries verifies the SSO Remote Session Boundaries scenario. TestSSORemoteSessionBoundaries 验证对应的 SSO 服务端场景。
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

// TestSSOOAuth2CodeFlow verifies the SSO OAuth2 authorization-code flow. TestSSOOAuth2CodeFlow 验证 SSO OAuth2 授权码流程。
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

// TestSSOOAuth2CodeBoundaries verifies SSO OAuth2 Code error boundaries. TestSSOOAuth2CodeBoundaries 验证 SSO OAuth2 授权码错误边界。
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
	if consumed, err := server.ConsumeOAuth2Code(ctx, code.Code, "app-a", "secret-a", "https://app.example.com/sso/callback"); err != nil || consumed == nil || !consumed.Used {
		t.Fatalf("ConsumeOAuth2Code() after rejected request = %+v, %v, want successful consume", consumed, err)
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

// newTestServer creates an isolated SSO server for tests. newTestServer 为测试创建隔离的 SSO 服务端。
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

// newTestClient creates the default SSO test client. newTestClient 创建默认 SSO 测试客户端。
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

// registerTestClient registers the default client for a test. registerTestClient 为测试注册默认客户端。
func registerTestClient(t *testing.T, server *Server) {
	t.Helper()
	if err := server.RegisterClient(newTestClient()); err != nil {
		t.Fatalf("RegisterClient() error = %v", err)
	}
}

// closeTrackingStorage records close calls for lifecycle tests. closeTrackingStorage 为生命周期测试记录关闭调用。
type closeTrackingStorage struct {
	*MemoryStorage
	closeCount int
	closeErr   error
}

// Close records one close call and returns the configured error. Close 记录一次关闭调用并返回预设错误。
func (s *closeTrackingStorage) Close() error {
	s.closeCount++
	return s.closeErr
}
