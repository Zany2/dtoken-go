package sso

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/derror"
)

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
	if _, err = server.GetClient(client.ClientID); !errors.Is(err, derror.ErrClientNotFound) {
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

	if _, err := server.GenerateTicket(ctx, "app-a", "", "https://app.example.com/sso/callback", nil, nil); !errors.Is(err, derror.ErrUserIDEmpty) {
		t.Fatalf("GenerateTicket() empty login error = %v, want ErrUserIDEmpty", err)
	}
	if _, err := server.GenerateTicket(ctx, "app-a", "user-1001", "https://evil.example.com/callback", nil, nil); !errors.Is(err, derror.ErrInvalidRedirectURI) {
		t.Fatalf("GenerateTicket() redirect error = %v, want ErrInvalidRedirectURI", err)
	}
	if _, err := server.GenerateTicket(ctx, "app-a", "user-1001", "https://app.example.com/sso/callback", []string{"admin"}, nil); !errors.Is(err, derror.ErrInvalidScope) {
		t.Fatalf("GenerateTicket() scope error = %v, want ErrInvalidScope", err)
	}

	ticket, err := server.GenerateTicket(ctx, "app-a", "user-1001", "https://app.example.com/sso/callback", nil, nil)
	if err != nil {
		t.Fatalf("GenerateTicket() error = %v", err)
	}
	if _, err = server.ConsumeTicket(ctx, ticket.Ticket, "app-a", "bad-secret", "https://app.example.com/sso/callback"); !errors.Is(err, derror.ErrInvalidClientCredentials) {
		t.Fatalf("ConsumeTicket() secret error = %v, want ErrInvalidClientCredentials", err)
	}
	if _, err = server.ConsumeTicket(ctx, ticket.Ticket, "app-a", "secret-a", "https://other.example.com/callback"); !errors.Is(err, derror.ErrRedirectURIMismatch) {
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

	client := newTestClient()
	client.Modes = []Mode{ModeSharedToken}
	if err := server.RegisterClient(client); err != nil {
		t.Fatalf("RegisterClient() error = %v", err)
	}

	if _, err := server.GenerateTicket(ctx, client.ClientID, "user-1001", "https://app.example.com/sso/callback", nil, nil); !errors.Is(err, ErrModeUnsupported) {
		t.Fatalf("GenerateTicket() unsupported mode error = %v, want ErrModeUnsupported", err)
	}
}

func newTestServer() *Server {
	return NewServerWithConfig("login:", "dtoken:", newSSOTestStorage(), ssoTestCodec{}, &Config{TicketExpiration: time.Minute})
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

type ssoTestCodec struct{}

func (ssoTestCodec) Name() string { return "json-test" }

func (ssoTestCodec) Encode(v any) ([]byte, error) { return json.Marshal(v) }

func (ssoTestCodec) Decode(data []byte, v any) error { return json.Unmarshal(data, v) }

type ssoTestStorage struct {
	values map[string]ssoTestStorageItem
}

type ssoTestStorageItem struct {
	value    any
	expireAt time.Time
}

func newSSOTestStorage() *ssoTestStorage {
	return &ssoTestStorage{values: make(map[string]ssoTestStorageItem)}
}

func (s *ssoTestStorage) Set(_ context.Context, key string, value any, expiration time.Duration) error {
	item := ssoTestStorageItem{value: value}
	if expiration > 0 {
		item.expireAt = time.Now().Add(expiration)
	}
	s.values[key] = item
	return nil
}

func (s *ssoTestStorage) Get(_ context.Context, key string) (any, error) {
	item, ok := s.values[key]
	if !ok {
		return nil, nil
	}
	if !item.expireAt.IsZero() && time.Now().After(item.expireAt) {
		delete(s.values, key)
		return nil, nil
	}
	return item.value, nil
}

func (s *ssoTestStorage) GetAndDelete(ctx context.Context, key string) (any, error) {
	value, err := s.Get(ctx, key)
	if err != nil || value == nil {
		return value, err
	}
	delete(s.values, key)
	return value, nil
}

func (s *ssoTestStorage) Delete(_ context.Context, keys ...string) error {
	for _, key := range keys {
		delete(s.values, key)
	}
	return nil
}

func (s *ssoTestStorage) Exists(ctx context.Context, key string) bool {
	value, _ := s.Get(ctx, key)
	return value != nil
}

func (s *ssoTestStorage) Expire(_ context.Context, key string, expiration time.Duration) error {
	item, ok := s.values[key]
	if !ok {
		return errors.New("key not found")
	}
	if expiration <= 0 {
		delete(s.values, key)
		return nil
	}
	item.expireAt = time.Now().Add(expiration)
	s.values[key] = item
	return nil
}

func (s *ssoTestStorage) TTL(ctx context.Context, key string) (time.Duration, error) {
	value, err := s.Get(ctx, key)
	if err != nil {
		return 0, err
	}
	if value == nil {
		return adapter.TTLNotFound, nil
	}
	item := s.values[key]
	if item.expireAt.IsZero() {
		return adapter.TTLNoExpire, nil
	}
	return time.Until(item.expireAt), nil
}

func (s *ssoTestStorage) Ping(context.Context) error {
	return nil
}
