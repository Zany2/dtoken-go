package ticket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
)

func TestConfigValidateAndClone(t *testing.T) {
	cfg := DefaultConfig()
	if cfg == nil || cfg.TTL != DefaultTicketTTL {
		t.Fatalf("DefaultConfig() = %+v, want default ttl", cfg)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	cloned := cfg.Clone()
	cloned.TTL = time.Second
	if cfg.TTL == cloned.TTL {
		t.Fatalf("Clone() shares state with original")
	}

	if err := (&Config{}).Validate(); err == nil {
		t.Fatalf("Validate() error = nil, want invalid ttl error")
	}
}

func TestManagerCreateValidateConsume(t *testing.T) {
	ctx := context.Background()
	mgr := newTestTicketManager(time.Minute)

	created, err := mgr.Create(ctx, CreateOptions{
		LoginID:   "user-1",
		Device:    "web",
		DeviceId:  "browser-1",
		Source:    "qr",
		SourceApp: "portal",
		TargetApp: "admin",
		Scopes:    []string{"profile"},
		Extra:     map[string]any{"trace": "abc"},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Ticket == "" || created.AuthType != "test" || created.Status != StatusValid {
		t.Fatalf("Create() = %+v, want valid ticket", created)
	}

	validated, err := mgr.Validate(ctx, created.Ticket, ValidateOptions{
		LoginID:   "user-1",
		Device:    "web",
		TargetApp: "admin",
	})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if validated.LoginID != "user-1" || validated.TargetApp != "admin" {
		t.Fatalf("Validate() = %+v, want preserved ticket data", validated)
	}

	result, err := mgr.Consume(ctx, created.Ticket, ValidateOptions{LoginID: "user-1"})
	if err != nil {
		t.Fatalf("Consume() error = %v", err)
	}
	if result.Ticket.Status != StatusConsumed || result.Ticket.LoginID != "user-1" {
		t.Fatalf("Consume() = %+v, want consumed ticket", result.Ticket)
	}

	status, err := mgr.Status(ctx, created.Ticket)
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status != StatusConsumed {
		t.Fatalf("Status() = %s, want %s", status, StatusConsumed)
	}
	if _, err = mgr.Consume(ctx, created.Ticket); !errors.Is(err, ErrTicketConsumed) {
		t.Fatalf("Consume() second error = %v, want ErrTicketConsumed", err)
	}
}

func TestManagerBoundaries(t *testing.T) {
	ctx := context.Background()
	mgr := newTestTicketManager(time.Minute)

	if _, err := mgr.Validate(ctx, ""); !errors.Is(err, ErrInvalidTicket) {
		t.Fatalf("Validate(empty) error = %v, want ErrInvalidTicket", err)
	}
	status, err := mgr.Status(ctx, "missing")
	if err != nil {
		t.Fatalf("Status(missing) error = %v", err)
	}
	if status != StatusInvalid {
		t.Fatalf("Status(missing) = %s, want %s", status, StatusInvalid)
	}
	if ttl, err := mgr.GetTTL(ctx, ""); err != nil || ttl != -2 {
		t.Fatalf("GetTTL(empty) = %d, %v, want -2, nil", ttl, err)
	}

	created, err := mgr.Create(ctx, CreateOptions{LoginID: "user-1", TargetApp: "admin"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err = mgr.Validate(ctx, created.Ticket, ValidateOptions{TargetApp: "console"}); !errors.Is(err, ErrTicketMismatch) {
		t.Fatalf("Validate(mismatch) error = %v, want ErrTicketMismatch", err)
	}

	if err = mgr.Revoke(ctx, created.Ticket); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
	if status, err = mgr.Status(ctx, created.Ticket); err != nil || status != StatusRevoked {
		t.Fatalf("Status(revoked) = %s, %v, want %s, nil", status, err, StatusRevoked)
	}
	if _, err = mgr.Validate(ctx, created.Ticket); !errors.Is(err, ErrTicketRevoked) {
		t.Fatalf("Validate(revoked) error = %v, want ErrTicketRevoked", err)
	}
	if err = mgr.Revoke(ctx, "missing"); err != nil {
		t.Fatalf("Revoke(missing) error = %v, want nil", err)
	}

	expired := &Ticket{
		Ticket:     "expired-ticket",
		CreateTime: time.Now().Add(-2 * time.Second).Unix(),
		ExpiresIn:  1,
		Status:     StatusValid,
	}
	if err = mgr.save(ctx, expired, time.Minute); err != nil {
		t.Fatalf("save(expired) error = %v", err)
	}
	if _, err = mgr.Validate(ctx, expired.Ticket); !errors.Is(err, ErrTicketExpired) {
		t.Fatalf("Validate(expired) error = %v, want ErrTicketExpired", err)
	}
	if status, err = mgr.Status(ctx, expired.Ticket); err != nil || status != StatusExpired {
		t.Fatalf("Status(expired) = %s, %v, want %s, nil", status, err, StatusExpired)
	}
}

func TestConsumeConstraintMismatchDoesNotConsumeTicket(t *testing.T) {
	ctx := context.Background()
	mgr := newTestTicketManager(time.Minute)

	created, err := mgr.Create(ctx, CreateOptions{LoginID: "user-1", TargetApp: "admin"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if _, err = mgr.Consume(ctx, created.Ticket, ValidateOptions{TargetApp: "console"}); !errors.Is(err, ErrTicketMismatch) {
		t.Fatalf("Consume(mismatch) error = %v, want ErrTicketMismatch", err)
	}

	validated, err := mgr.Validate(ctx, created.Ticket, ValidateOptions{TargetApp: "admin"})
	if err != nil {
		t.Fatalf("Validate(after mismatch) error = %v", err)
	}
	if validated.Status != StatusValid {
		t.Fatalf("Validate(after mismatch) status = %s, want %s", validated.Status, StatusValid)
	}

	result, err := mgr.Consume(ctx, created.Ticket, ValidateOptions{TargetApp: "admin"})
	if err != nil {
		t.Fatalf("Consume(after mismatch) error = %v", err)
	}
	if result.Ticket.Status != StatusConsumed {
		t.Fatalf("Consume(after mismatch) status = %s, want %s", result.Ticket.Status, StatusConsumed)
	}
}

func newTestTicketManager(ttl time.Duration) *Manager {
	return NewManagerWithConfig("test", "dt:", newTicketTestStorage(), ticketTestCodec{}, &Config{TTL: ttl})
}

type ticketTestCodec struct{}

func (ticketTestCodec) Name() string { return "json-test" }

func (ticketTestCodec) Encode(v any) ([]byte, error) { return json.Marshal(v) }

func (ticketTestCodec) Decode(data []byte, v any) error { return json.Unmarshal(data, v) }

type ticketTestStorage struct {
	mu    sync.Mutex
	items map[string]ticketTestStorageItem
}

type ticketTestStorageItem struct {
	value     any
	expiresAt time.Time
}

func newTicketTestStorage() *ticketTestStorage {
	return &ticketTestStorage{items: make(map[string]ticketTestStorageItem)}
}

func (s *ticketTestStorage) Set(_ context.Context, key string, value any, expiration time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	item := ticketTestStorageItem{value: value}
	if expiration > 0 {
		item.expiresAt = time.Now().Add(expiration)
	}
	s.items[key] = item
	return nil
}

func (s *ticketTestStorage) Get(_ context.Context, key string) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.items[key]
	if !ok {
		return nil, nil
	}
	if item.expired() {
		delete(s.items, key)
		return nil, nil
	}
	return item.value, nil
}

func (s *ticketTestStorage) Delete(_ context.Context, keys ...string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, key := range keys {
		delete(s.items, key)
	}
	return nil
}

func (s *ticketTestStorage) Exists(_ context.Context, key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.items[key]
	if !ok {
		return false
	}
	if item.expired() {
		delete(s.items, key)
		return false
	}
	return true
}

func (s *ticketTestStorage) Expire(_ context.Context, key string, expiration time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.items[key]
	if !ok || item.expired() {
		delete(s.items, key)
		return fmt.Errorf("key not found")
	}
	if expiration > 0 {
		item.expiresAt = time.Now().Add(expiration)
	} else {
		item.expiresAt = time.Time{}
	}
	s.items[key] = item
	return nil
}

func (s *ticketTestStorage) TTL(_ context.Context, key string) (time.Duration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.items[key]
	if !ok {
		return adapter.TTLNotFound, nil
	}
	if item.expired() {
		delete(s.items, key)
		return adapter.TTLNotFound, nil
	}
	if item.expiresAt.IsZero() {
		return adapter.TTLNoExpire, nil
	}
	return time.Until(item.expiresAt), nil
}

func (s *ticketTestStorage) Ping(context.Context) error { return nil }

func (s *ticketTestStorage) GetAndDelete(_ context.Context, key string) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.items[key]
	if !ok {
		return nil, nil
	}
	delete(s.items, key)
	if item.expired() {
		return nil, nil
	}
	return item.value, nil
}

func (item ticketTestStorageItem) expired() bool {
	return !item.expiresAt.IsZero() && time.Now().After(item.expiresAt)
}
