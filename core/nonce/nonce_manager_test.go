package nonce

import (
	"context"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/derror"
)

// TestNonceManagerConstructorsAndStorageKey verifies constructor defaults and generated key layout. TestNonceManagerConstructorsAndStorageKey 验证构造器默认值与生成键格式。
func TestNonceManagerConstructorsAndStorageKey(t *testing.T) {
	ctx := context.Background()
	storage := newNonceTestStorage()
	manager := NewDefaultNonceManager("auth", "prefix:", storage)
	if manager.TTL() != DefaultNonceTTL {
		t.Fatalf("NewDefaultNonceManager().TTL() = %s, want %s", manager.TTL(), DefaultNonceTTL)
	}

	value, err := manager.Generate(ctx)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(value) != hex.EncodedLen(NonceLength) {
		t.Fatalf("generated nonce length = %d, want %d", len(value), hex.EncodedLen(NonceLength))
	}
	if _, err = hex.DecodeString(value); err != nil {
		t.Fatalf("generated nonce is not hexadecimal: %v", err)
	}

	key := "prefix:auth" + NonceKeySuffix + value
	stored, ok := storage.values[key]
	if !ok {
		t.Fatalf("generated nonce key %q was not stored", key)
	}
	if _, ok = stored.(int64); !ok {
		t.Fatalf("stored nonce value type = %T, want int64", stored)
	}
	if remaining := time.Until(storage.expires[key]); remaining <= 0 || remaining > DefaultNonceTTL {
		t.Fatalf("stored nonce ttl remaining = %s, want (0, %s]", remaining, DefaultNonceTTL)
	}

	customConfig := &Config{TTL: 7 * time.Second}
	custom := NewNonceManagerWithConfig("auth", "prefix:", storage, customConfig)
	if custom.TTL() != customConfig.TTL {
		t.Fatalf("NewNonceManagerWithConfig().TTL() = %s, want %s", custom.TTL(), customConfig.TTL)
	}
	customConfig.TTL = time.Hour
	if custom.TTL() != 7*time.Second {
		t.Fatalf("constructor should copy config ttl, got %s", custom.TTL())
	}

	for name, cfg := range map[string]*Config{
		"nil":     nil,
		"invalid": {TTL: -time.Second},
	} {
		t.Run(name, func(t *testing.T) {
			got := NewNonceManagerWithConfig("auth", "prefix:", storage, cfg).TTL()
			if got != DefaultNonceTTL {
				t.Fatalf("NewNonceManagerWithConfig(%s).TTL() = %s, want %s", name, got, DefaultNonceTTL)
			}
		})
	}
}

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

// TestNonceManagerTimeoutFallback verifies non-positive timeout fallback. TestNonceManagerTimeoutFallback 验证非正有效期会回退默认值。
func TestNonceManagerTimeoutFallback(t *testing.T) {
	manager := NewNonceManager("auth:", "dtoken:", newNonceTestStorage(), -time.Second)
	if manager.TTL() != DefaultNonceTTL {
		t.Fatalf("NewNonceManager() ttl = %s, want %s", manager.TTL(), DefaultNonceTTL)
	}

	storage := newNonceTestStorage()
	manager = NewNonceManager("auth:", "dtoken:", storage, time.Minute)
	value, err := manager.GenerateWithTimeout(context.Background(), 0)
	if err != nil {
		t.Fatalf("GenerateWithTimeout(zero) error = %v", err)
	}
	if remaining := time.Until(storage.expires[manager.getNonceKey(value)]); remaining <= 0 || remaining > time.Minute {
		t.Fatalf("GenerateWithTimeout(zero) ttl remaining = %s, want (0, 1m]", remaining)
	}
}

// TestNonceManagerGenerateStorageError verifies generate maps storage failures. TestNonceManagerGenerateStorageError 验证生成时正确映射存储错误。
func TestNonceManagerGenerateStorageError(t *testing.T) {
	storage := &nonceFailingSetStorage{nonceBasicStorage: newNonceTestStorage().nonceBasicStorage, err: errors.New("set failed")}
	manager := NewNonceManager("auth", "prefix:", storage, time.Minute)

	value, err := manager.Generate(context.Background())
	if value != "" {
		t.Fatalf("Generate() value = %q on storage error, want empty", value)
	}
	if !errors.Is(err, derror.ErrStorageUnavailable) {
		t.Fatalf("Generate() error = %v, want ErrStorageUnavailable", err)
	}
	if len(storage.values) != 0 {
		t.Fatalf("Generate() stored %d values after storage error, want 0", len(storage.values))
	}
}

// TestNonceManagerExpiredNonce verifies expired nonce cannot be consumed. TestNonceManagerExpiredNonce 验证过期 nonce 不可消费。
func TestNonceManagerExpiredNonce(t *testing.T) {
	ctx := context.Background()
	manager := NewNonceManager("auth:", "dtoken:", newNonceTestStorage(), time.Minute)

	value, err := manager.GenerateWithTimeout(ctx, time.Nanosecond)
	if err != nil {
		t.Fatalf("GenerateWithTimeout() error = %v", err)
	}
	time.Sleep(time.Millisecond)

	if manager.IsValid(ctx, value) {
		t.Fatal("IsValid(expired) = true, want false")
	}
	if manager.Verify(ctx, value) {
		t.Fatal("Verify(expired) = true, want false")
	}
	if err = manager.VerifyAndConsume(ctx, value); !errors.Is(err, derror.ErrInvalidNonce) {
		t.Fatalf("VerifyAndConsume(expired) error = %v, want ErrInvalidNonce", err)
	}
}

// TestNonceManagerGetTTLSentinels verifies all storage TTL sentinel mappings and errors. TestNonceManagerGetTTLSentinels 验证全部存储 TTL 哨兵映射及错误处理。
func TestNonceManagerGetTTLSentinels(t *testing.T) {
	ctx := context.Background()
	for _, tt := range []struct {
		name string
		raw  time.Duration
		want int64
	}{
		{name: "not found", raw: adapter.TTLNotFound, want: -2},
		{name: "no expire", raw: adapter.TTLNoExpire, want: -1},
		{name: "positive", raw: 3 * time.Second, want: 3},
		{name: "zero", raw: 0, want: 0},
		{name: "other negative", raw: -3 * time.Second, want: 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			storage := &nonceTTLStorage{nonceBasicStorage: newNonceTestStorage().nonceBasicStorage, ttl: tt.raw}
			manager := NewNonceManager("auth", "prefix:", storage, time.Minute)
			got, err := manager.GetTTL(ctx, "nonce")
			if err != nil || got != tt.want {
				t.Fatalf("GetTTL() = %d, %v; want %d, nil", got, err, tt.want)
			}
		})
	}

	storageErr := errors.New("ttl failed")
	manager := NewNonceManager("auth", "prefix:", &nonceTTLStorage{
		nonceBasicStorage: newNonceTestStorage().nonceBasicStorage,
		err:               storageErr,
	}, time.Minute)
	if got, err := manager.GetTTL(ctx, "nonce"); got != 0 || !errors.Is(err, derror.ErrStorageUnavailable) {
		t.Fatalf("GetTTL(storage error) = %d, %v; want 0 and ErrStorageUnavailable", got, err)
	}
	if got, err := manager.GetTTL(ctx, ""); got != -2 || err != nil {
		t.Fatalf("GetTTL(empty with failing storage) = %d, %v; want -2, nil", got, err)
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
	if err = manager.VerifyAndConsume(ctx, value); !errors.Is(err, derror.ErrStorageCapabilityUnsupported) {
		t.Fatalf("VerifyAndConsume() error = %v, want ErrStorageCapabilityUnsupported", err)
	}
}

// TestNonceManagerVerifyStorageError verifies storage errors are preserved. TestNonceManagerVerifyStorageError 验证存储错误会保留。
func TestNonceManagerVerifyStorageError(t *testing.T) {
	ctx := context.Background()
	manager := NewNonceManager("auth:", "dtoken:", &nonceFailingAtomicStorage{}, time.Minute)

	if manager.Verify(ctx, "nonce") {
		t.Fatal("Verify() = true on storage error, want false")
	}
	if err := manager.VerifyAndConsume(ctx, "nonce"); !errors.Is(err, derror.ErrStorageUnavailable) {
		t.Fatalf("VerifyAndConsume() error = %v, want ErrStorageUnavailable", err)
	}
}

// TestNonceManagerInvalidInputs verifies empty nonce handling for consuming and TTL lookup. TestNonceManagerInvalidInputs 验证消费与 TTL 查询的空 nonce 处理。
func TestNonceManagerInvalidInputs(t *testing.T) {
	manager := NewNonceManager("auth", "prefix:", newNonceTestStorage(), time.Minute)
	if err := manager.VerifyAndConsume(context.Background(), ""); !errors.Is(err, derror.ErrInvalidNonce) {
		t.Fatalf("VerifyAndConsume(empty) error = %v, want ErrInvalidNonce", err)
	}
	if manager.IsValid(context.Background(), "") {
		t.Fatal("IsValid(empty) = true, want false")
	}
	if got, err := manager.GetTTL(context.Background(), "missing"); got != -2 || err != nil {
		t.Fatalf("GetTTL(missing) = %d, %v; want -2, nil", got, err)
	}
}

// TestNonceManagerConcurrentVerifyConsumesOnce verifies one-time consumption under concurrent callers. TestNonceManagerConcurrentVerifyConsumesOnce 验证并发调用下 nonce 只会成功消费一次。
func TestNonceManagerConcurrentVerifyConsumesOnce(t *testing.T) {
	ctx := context.Background()
	manager := NewNonceManager("auth", "prefix:", newNonceTestStorage(), time.Minute)
	value, err := manager.Generate(ctx)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	start := make(chan struct{})
	errs := make(chan error, 2)
	for range 2 {
		go func() {
			<-start
			errs <- manager.VerifyAndConsume(ctx, value)
		}()
	}
	close(start)

	successes := 0
	invalid := 0
	for range 2 {
		switch err := <-errs; {
		case err == nil:
			successes++
		case errors.Is(err, derror.ErrInvalidNonce):
			invalid++
		default:
			t.Fatalf("concurrent VerifyAndConsume() error = %v, want one success and one ErrInvalidNonce", err)
		}
	}
	if successes != 1 || invalid != 1 {
		t.Fatalf("concurrent VerifyAndConsume() results = %d success, %d invalid; want 1 each", successes, invalid)
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

func (s *nonceTestStorage) SetIfAbsent(ctx context.Context, key string, value any, expiration time.Duration) (bool, error) {
	if s.Exists(ctx, key) {
		return false, nil
	}
	if err := s.Set(ctx, key, value, expiration); err != nil {
		return false, err
	}
	return true, nil
}

type nonceFailingAtomicStorage struct {
	nonceBasicStorage
}

func (nonceFailingAtomicStorage) GetAndDelete(context.Context, string) (any, error) {
	return nil, errors.New("storage down")
}

func (nonceFailingAtomicStorage) SetIfAbsent(context.Context, string, any, time.Duration) (bool, error) {
	return false, errors.New("storage down")
}

type nonceFailingSetStorage struct {
	*nonceBasicStorage
	err error
}

func (s *nonceFailingSetStorage) Set(context.Context, string, any, time.Duration) error {
	return s.err
}

type nonceTTLStorage struct {
	*nonceBasicStorage
	ttl time.Duration
	err error
}

func (s *nonceTTLStorage) TTL(context.Context, string) (time.Duration, error) {
	return s.ttl, s.err
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
