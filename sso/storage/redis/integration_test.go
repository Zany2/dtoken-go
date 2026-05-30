package redis

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/sso"
)

func TestRedisSSOFlow(t *testing.T) {
	redisURL := os.Getenv("DTOKEN_SSO_REDIS")
	if redisURL == "" {
		t.Skip("set DTOKEN_SSO_REDIS to run Redis SSO integration test")
	}

	ctx := context.Background()
	server, err := NewServer(
		redisURL,
		sso.WithKeyPrefix("dtoken:test:"),
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
	client := &sso.Client{
		ClientID:     "redis-app",
		ClientSecret: "redis-secret",
		RedirectURIs: []string{
			"https://redis.example.com/sso/callback",
		},
		Modes: []sso.Mode{sso.ModeTicket, sso.ModeOAuth2},
	}
	if err = server.RegisterClient(client); err != nil {
		t.Fatalf("RegisterClient() error = %v", err)
	}
	t.Cleanup(func() {
		_ = server.UnregisterClient("redis-app")
		_ = server.ClearClientSessions(ctx, "user-redis")
	})

	ticket, err := server.GenerateTicket(ctx, "redis-app", "user-redis", "https://redis.example.com/sso/callback", nil, nil)
	if err != nil {
		t.Fatalf("GenerateTicket() error = %v", err)
	}
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
	consumedCode, err := server.ConsumeOAuth2Code(ctx, code.Code, "redis-app", "redis-secret", "https://redis.example.com/sso/callback")
	if err != nil {
		t.Fatalf("ConsumeOAuth2Code() error = %v", err)
	}
	if consumedCode.LoginID != "user-redis" || !consumedCode.Used {
		t.Fatalf("ConsumeOAuth2Code() = %+v, want consumed user-redis code", consumedCode)
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
