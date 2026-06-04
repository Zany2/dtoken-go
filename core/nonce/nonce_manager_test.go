package nonce

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/derror"
)

// TestNonceManagerGenerateVerifyAndConsume verifies nonce lifecycle semantics. TestNonceManagerGenerateVerifyAndConsume 验证 nonce 生命周期语义。
func TestNonceManagerGenerateVerifyAndConsume(t *testing.T) {
	ctx := context.Background()
	storage := newNonceTestStorage()
	manager := NewNonceManager("auth:", "dtoken:", storage, time.Minute)

	value, err := manager.Generate(ctx)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if value == "" {
		t.Fatal("Generate() returned empty nonce")
	}
	if !manager.IsValid(ctx, value) {
		t.Fatal("IsValid() = false, want true before consume")
	}
	if !manager.Verify(ctx, value) {
		t.Fatal("Verify() = false, want true on first consume")
	}
	if manager.IsValid(ctx, value) {
		t.Fatal("IsValid() = true after consume, want false")
	}
	if manager.Verify(ctx, value) {
		t.Fatal("Verify() = true on second consume, want false")
	}
	if err = manager.VerifyAndConsume(ctx, value); !errors.Is(err, derror.ErrInvalidNonce) {
		t.Fatalf("VerifyAndConsume() error = %v, want ErrInvalidNonce", err)
	}
}

// TestNonceManagerTTL verifies TTL sentinel mapping. TestNonceManagerTTL 验证 TTL 哨兵值映射。
func TestNonceManagerTTL(t *testing.T) {
	ctx := context.Background()
	manager := NewNonceManager("auth:", "dtoken:", newNonceTestStorage(), time.Minute)

	if ttl, err := manager.GetTTL(ctx, ""); err != nil || ttl != -2 {
		t.Fatalf("GetTTL(empty) = %d, %v; want -2, nil", ttl, err)
	}

	value, err := manager.GenerateWithTimeout(ctx, 30*time.Second)
	if err != nil {
		t.Fatalf("GenerateWithTimeout() error = %v", err)
	}
	ttl, err := manager.GetTTL(ctx, value)
	if err != nil {
		t.Fatalf("GetTTL() error = %v", err)
	}
	if ttl <= 0 || ttl > 30 {
		t.Fatalf("GetTTL() = %d, want 1..30", ttl)
	}
}

// TestNonceManagerRequiresAtomicStorage verifies non-atomic storage fails closed. TestNonceManagerRequiresAtomicStorage 验证非原子存储会安全失败。
func TestNonceManagerRequiresAtomicStorage(t *testing.T) {
	ctx := context.Background()
	storage := &nonceBasicStorage{values: map[string]any{}}
	manager := NewNonceManager("auth:", "dtoken:", storage, time.Minute)

	value, err := manager.Generate(ctx)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if !manager.IsValid(ctx, value) {
		t.Fatal("IsValid() = false, want true")
	}
	if manager.Verify(ctx, value) {
		t.Fatal("Verify() = true with non-atomic storage, want false")
	}
}

type nonceTestStorage struct {
	*nonceBasicStorage
}

func newNonceTestStorage() *nonceTestStorage {
	return &nonceTestStorage{nonceBasicStorage: &nonceBasicStorage{values: map[string]any{}, expires: map[string]time.Time{}}}
}

func (s *nonceTestStorage) GetAndDelete(ctx context.Context, key string) (any, error) {
	value, err := s.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	_ = s.Delete(ctx, key)
	return value, nil
}

type nonceBasicStorage struct {
	values  map[string]any
	expires map[string]time.Time
}

func (s *nonceBasicStorage) Set(_ context.Context, key string, value any, expiration time.Duration) error {
	if s.values == nil {
		s.values = map[string]any{}
	}
	if s.expires == nil {
		s.expires = map[string]time.Time{}
	}
	s.values[key] = value
	if expiration > 0 {
		s.expires[key] = time.Now().Add(expiration)
	} else {
		delete(s.expires, key)
	}
	return nil
}

func (s *nonceBasicStorage) Get(_ context.Context, key string) (any, error) {
	if s.isExpired(key) {
		return nil, nil
	}
	return s.values[key], nil
}

func (s *nonceBasicStorage) Delete(_ context.Context, keys ...string) error {
	for _, key := range keys {
		delete(s.values, key)
		delete(s.expires, key)
	}
	return nil
}

func (s *nonceBasicStorage) Exists(_ context.Context, key string) bool {
	if s.isExpired(key) {
		return false
	}
	_, ok := s.values[key]
	return ok
}

func (s *nonceBasicStorage) Expire(_ context.Context, key string, expiration time.Duration) error {
	if !s.Exists(context.Background(), key) {
		return derror.ErrInvalidToken
	}
	if expiration > 0 {
		s.expires[key] = time.Now().Add(expiration)
	} else {
		delete(s.expires, key)
	}
	return nil
}

func (s *nonceBasicStorage) TTL(_ context.Context, key string) (time.Duration, error) {
	if s.isExpired(key) {
		return adapter.TTLNotFound, nil
	}
	if _, ok := s.values[key]; !ok {
		return adapter.TTLNotFound, nil
	}
	expireAt, ok := s.expires[key]
	if !ok {
		return adapter.TTLNoExpire, nil
	}
	ttl := time.Until(expireAt)
	if ttl <= 0 {
		_ = s.Delete(context.Background(), key)
		return adapter.TTLNotFound, nil
	}
	return ttl, nil
}

func (s *nonceBasicStorage) Ping(context.Context) error { return nil }

func (s *nonceBasicStorage) isExpired(key string) bool {
	expireAt, ok := s.expires[key]
	if !ok || time.Now().Before(expireAt) {
		return false
	}
	_ = s.Delete(context.Background(), key)
	return true
}
