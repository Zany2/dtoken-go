package redis

import (
	"context"
	"crypto/rand"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/sso"
)

// TestRedisSSOFlow verifies the Redis SSO Flow scenario. TestRedisSSOFlow 验证对应的 Redis SSO 集成场景。
func TestRedisSSOFlow(t *testing.T) {
	redisURL := strings.TrimSpace(os.Getenv("DTOKEN_REDIS_URL"))
	if redisURL == "" {
		t.Skip("set DTOKEN_REDIS_URL to run Redis SSO integration test")
	}

	ctx := context.Background()
	server, err := NewServer(
		redisURL,
		sso.WithKeyPrefix("dtoken:test:"+rand.Text()+":"),
		sso.WithAuthType("sso:"),
		sso.WithConfig(&sso.Config{
			TicketExpiration:        time.Minute,
			SharedTokenExpiration:   time.Minute,
			RemoteSessionExpiration: time.Minute,
			OAuth2CodeExpiration:    time.Minute,
		}),
	)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	// Cleanup callbacks run in reverse order, so data is removed before closing Redis. 清理回调逆序执行，保证先清理数据再关闭 Redis。
	t.Cleanup(func() {
		if err := server.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})
	t.Cleanup(func() {
		if err := server.ClearClientSessions(ctx, "user-redis"); err != nil {
			t.Errorf("ClearClientSessions() cleanup error = %v", err)
		}
		if err := server.UnregisterClient("redis-app"); err != nil {
			t.Errorf("UnregisterClient() cleanup error = %v", err)
		}
	})
	client := &sso.Client{
		ClientID:     "redis-app",
		ClientSecret: "redis-secret",
		RedirectURIs: []string{
			"https://redis.example.com/sso/callback",
		},
		Modes: []sso.Mode{sso.ModeTicket, sso.ModeOAuth2, sso.ModeSharedToken, sso.ModeRemoteSession},
	}
	if err = server.RegisterClient(client); err != nil {
		t.Fatalf("RegisterClient() error = %v", err)
	}

	ticket, err := server.GenerateTicket(ctx, "redis-app", "user-redis", "https://redis.example.com/sso/callback", nil, nil)
	if err != nil {
		t.Fatalf("GenerateTicket() error = %v", err)
	}
	t.Cleanup(func() {
		if err := server.RevokeTicket(ctx, ticket.Ticket); err != nil {
			t.Errorf("RevokeTicket() cleanup error = %v", err)
		}
	})
	consumedTicket, err := server.ConsumeTicket(ctx, ticket.Ticket, "redis-app", "redis-secret", "https://redis.example.com/sso/callback")
	if err != nil {
		t.Fatalf("ConsumeTicket() error = %v", err)
	}
	if consumedTicket.LoginID != "user-redis" || !consumedTicket.Used {
		t.Fatalf("ConsumeTicket() = %+v, want consumed user-redis ticket", consumedTicket)
	}
	if _, err = server.ValidateTicket(ctx, ticket.Ticket); !errors.Is(err, sso.ErrInvalidTicket) {
		t.Fatalf("ValidateTicket() after consume error = %v, want ErrInvalidTicket", err)
	}

	code, err := server.GenerateOAuth2Code(ctx, "redis-app", "user-redis", "https://redis.example.com/sso/callback", nil, nil)
	if err != nil {
		t.Fatalf("GenerateOAuth2Code() error = %v", err)
	}
	t.Cleanup(func() {
		if err := server.RevokeOAuth2Code(ctx, code.Code); err != nil {
			t.Errorf("RevokeOAuth2Code() cleanup error = %v", err)
		}
	})
	consumedCode, err := server.ConsumeOAuth2Code(ctx, code.Code, "redis-app", "redis-secret", "https://redis.example.com/sso/callback")
	if err != nil {
		t.Fatalf("ConsumeOAuth2Code() error = %v", err)
	}
	if consumedCode.LoginID != "user-redis" || !consumedCode.Used {
		t.Fatalf("ConsumeOAuth2Code() = %+v, want consumed user-redis code", consumedCode)
	}
	if _, err = server.ConsumeOAuth2Code(ctx, code.Code, client.ClientID, client.ClientSecret, client.RedirectURIs[0]); !errors.Is(err, sso.ErrInvalidOAuth2Code) {
		t.Fatalf("replayed OAuth2 code error = %v, want ErrInvalidOAuth2Code", err)
	}

	// Verify reusable credentials and renewed deadlines against the same Redis backend. 在同一 Redis 后端验证可复用凭证及续期截止时间。
	shared, err := server.GenerateSharedToken(ctx, client.ClientID, "user-redis", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := server.RevokeSharedToken(ctx, shared.Token); err != nil {
			t.Errorf("RevokeSharedToken() cleanup error = %v", err)
		}
	})
	if loaded, err := server.ValidateSharedToken(ctx, shared.Token, client.ClientID); err != nil || loaded.LoginID != "user-redis" || !loaded.ExpiresAt.Equal(shared.ExpiresAt) {
		t.Fatalf("shared token round trip = %+v, %v", loaded, err)
	}
	if ttl, err := server.GetSharedTokenTTL(ctx, shared.Token); err != nil || ttl <= 0 || ttl > 60 {
		t.Fatalf("shared token TTL = %d, %v", ttl, err)
	}
	if err := server.RevokeSharedToken(ctx, shared.Token); err != nil {
		t.Fatal(err)
	}
	if _, err := server.ValidateSharedToken(ctx, shared.Token, client.ClientID); !errors.Is(err, sso.ErrInvalidSharedToken) {
		t.Fatalf("revoked shared token error = %v", err)
	}

	remote, err := server.CreateRemoteSession(ctx, client.ClientID, "user-redis", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := server.RevokeRemoteSession(ctx, remote.SessionID); err != nil {
			t.Errorf("RevokeRemoteSession() cleanup error = %v", err)
		}
	})
	if err := server.RenewRemoteSession(ctx, remote.SessionID, 2*time.Minute); err != nil {
		t.Fatal(err)
	}
	if renewed, err := server.ValidateRemoteSession(ctx, remote.SessionID, client.ClientID); err != nil || renewed.LoginID != "user-redis" || !renewed.ExpiresAt.After(remote.ExpiresAt) || renewed.ExpiresIn != 120 {
		t.Fatalf("renewed remote session = %+v, %v", renewed, err)
	}
	if ttl, err := server.GetRemoteSessionTTL(ctx, remote.SessionID); err != nil || ttl <= 60 || ttl > 120 {
		t.Fatalf("renewed remote session TTL = %d, %v", ttl, err)
	}
	if err := server.RevokeRemoteSession(ctx, remote.SessionID); err != nil {
		t.Fatal(err)
	}
	if _, err := server.ValidateRemoteSession(ctx, remote.SessionID, client.ClientID); !errors.Is(err, sso.ErrInvalidRemoteSession) {
		t.Fatalf("revoked remote session error = %v", err)
	}
	if ttl, err := server.GetRemoteSessionTTL(ctx, remote.SessionID); err != nil || ttl != -2 {
		t.Fatalf("revoked remote session TTL = %d, %v, want -2", ttl, err)
	}

	session, err := server.RegisterClientSession(ctx, "user-redis", "redis-app", "https://redis.example.com/sso/logout-callback")
	if err != nil {
		t.Fatalf("RegisterClientSession() error = %v", err)
	}
	if !strings.Contains(session.LogoutCallbackURL, "/sso/logout-callback") {
		t.Fatalf("RegisterClientSession() callback = %q, want logout callback", session.LogoutCallbackURL)
	}
	sessions, err := server.GetClientSessions(ctx, "user-redis")
	if err != nil {
		t.Fatalf("GetClientSessions() error = %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("GetClientSessions() = %+v, want one session", sessions)
	}
	if err = server.ClearClientSessions(ctx, "user-redis"); err != nil {
		t.Fatalf("ClearClientSessions() error = %v", err)
	}
}
