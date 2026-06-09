package shortkey

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
	if cfg == nil || cfg.TTL != DefaultTTL || cfg.Length != DefaultLength || cfg.MaxGenerateRetries <= 0 {
		t.Fatalf("DefaultConfig() = %+v, want defaults", cfg)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	cloned := cfg.Clone()
	cloned.Length++
	if cfg.Length == cloned.Length {
		t.Fatalf("Clone() shares state with original")
	}

	for name, cfg := range map[string]*Config{
		"ttl":     {TTL: 0, Length: 1, MaxGenerateRetries: 1},
		"length":  {TTL: time.Second, Length: 0, MaxGenerateRetries: 1},
		"retries": {TTL: time.Second, Length: 1, MaxGenerateRetries: 0},
	} {
		if err := cfg.Validate(); err == nil {
			t.Fatalf("Validate(%s) error = nil, want error", name)
		}
	}
}

func TestManagerCreateConfirmValidateConsume(t *testing.T) {
	ctx := context.Background()
	mgr := newTestShortKeyManager(time.Minute)

	created, err := mgr.Create(ctx, CreateOptions{
		Scene:     "login",
		SourceApp: "portal",
		TargetApp: "admin",
		Scopes:    []string{"profile"},
		Extra:     map[string]any{"trace": "abc"},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Key == "" || created.AuthType != "test" || created.Status != StatusPending {
		t.Fatalf("Create() = %+v, want pending short key", created)
	}
	if _, err = mgr.Validate(ctx, created.Key); !errors.Is(err, ErrShortKeyPending) {
		t.Fatalf("Validate(pending) error = %v, want ErrShortKeyPending", err)
	}

	confirmed, err := mgr.Confirm(ctx, created.Key, ConfirmOptions{
		LoginID:  "user-1",
		Device:   "web",
		DeviceId: "browser-1",
		Scopes:   []string{"profile", "email"},
		Extra:    map[string]any{"confirmed": true},
	})
	if err != nil {
		t.Fatalf("Confirm() error = %v", err)
	}
	if confirmed.Status != StatusConfirmed || confirmed.LoginID != "user-1" {
		t.Fatalf("Confirm() = %+v, want confirmed user key", confirmed)
	}

	validated, err := mgr.Validate(ctx, created.Key, ValidateOptions{
		LoginID:   "user-1",
		Device:    "web",
		Scene:     "login",
		TargetApp: "admin",
	})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if validated.DeviceId != "browser-1" || len(validated.Scopes) != 2 {
		t.Fatalf("Validate() = %+v, want confirmed data", validated)
	}

	result, err := mgr.Consume(ctx, created.Key, ValidateOptions{LoginID: "user-1"})
	if err != nil {
		t.Fatalf("Consume() error = %v", err)
	}
	if result.ShortKey.Status != StatusConsumed || result.ShortKey.LoginID != "user-1" {
		t.Fatalf("Consume() = %+v, want consumed short key", result.ShortKey)
	}

	status, err := mgr.Status(ctx, created.Key)
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status != StatusConsumed {
		t.Fatalf("Status() = %s, want %s", status, StatusConsumed)
	}
	if _, err = mgr.Validate(ctx, created.Key); !errors.Is(err, ErrShortKeyConsumed) {
		t.Fatalf("Validate(consumed) error = %v, want ErrShortKeyConsumed", err)
	}
}

func TestManagerBoundaries(t *testing.T) {
	ctx := context.Background()
	mgr := newTestShortKeyManager(time.Minute)

	if _, err := mgr.Validate(ctx, ""); !errors.Is(err, ErrInvalidShortKey) {
		t.Fatalf("Validate(empty) error = %v, want ErrInvalidShortKey", err)
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

	created, err := mgr.Create(ctx, CreateOptions{LoginID: "user-1", Scene: "login"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err = mgr.Validate(ctx, created.Key, ValidateOptions{Scene: "bind"}); !errors.Is(err, ErrShortKeyMismatch) {
		t.Fatalf("Validate(mismatch) error = %v, want ErrShortKeyMismatch", err)
	}

	if err = mgr.Revoke(ctx, created.Key); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
	if status, err = mgr.Status(ctx, created.Key); err != nil || status != StatusRevoked {
		t.Fatalf("Status(revoked) = %s, %v, want %s, nil", status, err, StatusRevoked)
	}
	if _, err = mgr.Validate(ctx, created.Key); !errors.Is(err, ErrShortKeyRevoked) {
		t.Fatalf("Validate(revoked) error = %v, want ErrShortKeyRevoked", err)
	}
	if err = mgr.Revoke(ctx, "missing"); err != nil {
		t.Fatalf("Revoke(missing) error = %v, want nil", err)
	}

	expired := &ShortKey{
		Key:        "expired-key",
		CreateTime: time.Now().Add(-2 * time.Second).Unix(),
		UpdateTime: time.Now().Add(-2 * time.Second).Unix(),
		ExpiresIn:  1,
		Status:     StatusConfirmed,
	}
	if err = mgr.save(ctx, expired, time.Minute); err != nil {
		t.Fatalf("save(expired) error = %v", err)
	}
	if _, err = mgr.Validate(ctx, expired.Key); !errors.Is(err, ErrShortKeyExpired) {
		t.Fatalf("Validate(expired) error = %v, want ErrShortKeyExpired", err)
	}
	if status, err = mgr.Status(ctx, expired.Key); err != nil || status != StatusExpired {
		t.Fatalf("Status(expired) = %s, %v, want %s, nil", status, err, StatusExpired)
	}
}

func TestConsumeConstraintMismatchDoesNotConsumeShortKey(t *testing.T) {
	ctx := context.Background()
	mgr := newTestShortKeyManager(time.Minute)

	created, err := mgr.Create(ctx, CreateOptions{LoginID: "user-1", Scene: "login"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if _, err = mgr.Consume(ctx, created.Key, ValidateOptions{Scene: "bind"}); !errors.Is(err, ErrShortKeyMismatch) {
		t.Fatalf("Consume(mismatch) error = %v, want ErrShortKeyMismatch", err)
	}

	validated, err := mgr.Validate(ctx, created.Key, ValidateOptions{Scene: "login"})
	if err != nil {
		t.Fatalf("Validate(after mismatch) error = %v", err)
	}
	if validated.Status != StatusConfirmed {
		t.Fatalf("Validate(after mismatch) status = %s, want %s", validated.Status, StatusConfirmed)
	}

	result, err := mgr.Consume(ctx, created.Key, ValidateOptions{Scene: "login"})
	if err != nil {
		t.Fatalf("Consume(after mismatch) error = %v", err)
	}
	if result.ShortKey.Status != StatusConsumed {
		t.Fatalf("Consume(after mismatch) status = %s, want %s", result.ShortKey.Status, StatusConsumed)
	}
}

func newTestShortKeyManager(ttl time.Duration) *Manager {
	return NewManagerWithConfig("test", "dt:", newShortKeyTestStorage(), shortKeyTestCodec{}, &Config{
		TTL:                ttl,
		Length:             DefaultLength,
		MaxGenerateRetries: 4,
	})
}

type shortKeyTestCodec struct{}

func (shortKeyTestCodec) Name() string { return "json-test" }

func (shortKeyTestCodec) Encode(v any) ([]byte, error) { return json.Marshal(v) }

func (shortKeyTestCodec) Decode(data []byte, v any) error { return json.Unmarshal(data, v) }

type shortKeyTestStorage struct {
	mu    sync.Mutex
	items map[string]shortKeyTestStorageItem
}

type shortKeyTestStorageItem struct {
	value     any
	expiresAt time.Time
}

func newShortKeyTestStorage() *shortKeyTestStorage {
	return &shortKeyTestStorage{items: make(map[string]shortKeyTestStorageItem)}
}

func (s *shortKeyTestStorage) Set(_ context.Context, key string, value any, expiration time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	item := shortKeyTestStorageItem{value: value}
	if expiration > 0 {
		item.expiresAt = time.Now().Add(expiration)
	}
	s.items[key] = item
	return nil
}

func (s *shortKeyTestStorage) Get(_ context.Context, key string) (any, error) {
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

func (s *shortKeyTestStorage) Delete(_ context.Context, keys ...string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, key := range keys {
		delete(s.items, key)
	}
	return nil
}

func (s *shortKeyTestStorage) Exists(_ context.Context, key string) bool {
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

func (s *shortKeyTestStorage) Expire(_ context.Context, key string, expiration time.Duration) error {
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

func (s *shortKeyTestStorage) TTL(_ context.Context, key string) (time.Duration, error) {
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

func (s *shortKeyTestStorage) Ping(context.Context) error { return nil }

func (s *shortKeyTestStorage) GetAndDelete(_ context.Context, key string) (any, error) {
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

func (item shortKeyTestStorageItem) expired() bool {
	return !item.expiresAt.IsZero() && time.Now().After(item.expiresAt)
}
