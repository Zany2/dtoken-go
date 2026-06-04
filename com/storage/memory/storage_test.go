// @Author daixk 2025/12/22 15:56:00
package memory

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/adapter/storagetest"
	"github.com/Zany2/dtoken-go/core/derror"
)

// TestStorageContract verifies memory storage follows the shared storage contract. TestStorageContract 验证内存存储符合共享存储契约。
func TestStorageContract(t *testing.T) {
	storagetest.RunStorageContract(t, func(t *testing.T) adapter.FullStorage {
		return NewStorage()
	})
}

// TestStorageBasicOperations verifies memory storage CRUD behavior 测试内存存储基础增删查行为
func TestStorageBasicOperations(t *testing.T) {
	ctx := context.Background()
	s := NewStorage()

	if err := s.Set(ctx, "user:1", "alice", 0); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if !s.Exists(ctx, "user:1") {
		t.Fatal("Exists() should return true")
	}
	got, err := s.Get(ctx, "user:1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != "alice" {
		t.Fatalf("Get() = %v, want alice", got)
	}

	got, err = s.GetAndDelete(ctx, "user:1")
	if err != nil {
		t.Fatalf("GetAndDelete() error = %v", err)
	}
	if got != "alice" || s.Exists(ctx, "user:1") {
		t.Fatalf("GetAndDelete() got=%v exists=%v", got, s.Exists(ctx, "user:1"))
	}

	got, err = s.Get(ctx, "missing")
	if err != nil || got != nil {
		t.Fatalf("Get(missing) got=%v err=%v, want nil nil", got, err)
	}
}

// TestStorageKeysAndPatterns verifies wildcard pattern matching 测试通配符匹配
func TestStorageKeysAndPatterns(t *testing.T) {
	ctx := context.Background()
	s := NewStorage()
	_ = s.Set(ctx, "user:1", "a", 0)
	_ = s.Set(ctx, "user:2", "b", 0)
	_ = s.Set(ctx, "team:1", "c", 0)
	_ = s.Set(ctx, "user:*", "literal", 0)

	keys, err := s.Keys(ctx, "user:?")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}
	sort.Strings(keys)
	if !reflect.DeepEqual(keys, []string{"user:*", "user:1", "user:2"}) {
		t.Fatalf("Keys(user:?) = %v", keys)
	}

	keys, err = s.Keys(ctx, `user:\*`)
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}
	if !reflect.DeepEqual(keys, []string{"user:*"}) {
		t.Fatalf("Keys(user:\\*) = %v", keys)
	}

	keys, err = s.Keys(ctx, "")
	if err != nil {
		t.Fatalf("Keys(empty) error = %v", err)
	}
	sort.Strings(keys)
	if !reflect.DeepEqual(keys, []string{"team:1", "user:*", "user:1", "user:2"}) {
		t.Fatalf("Keys(empty) = %v", keys)
	}
}

// TestStorageTTLAndExpire verifies TTL and expiration behavior 测试 TTL 与过期行为
func TestStorageTTLAndExpire(t *testing.T) {
	ctx := context.Background()
	s := NewStorage()

	if ttl, err := s.TTL(ctx, "missing"); err != nil || ttl != TTLNotFound {
		t.Fatalf("TTL(missing) = %v, %v, want %v nil", ttl, err, TTLNotFound)
	}

	_ = s.Set(ctx, "forever", "value", 0)
	if ttl, err := s.TTL(ctx, "forever"); err != nil || ttl != TTLNoExpire {
		t.Fatalf("TTL(forever) = %v, %v, want %v nil", ttl, err, TTLNoExpire)
	}

	if err := s.Expire(ctx, "forever", 50*time.Millisecond); err != nil {
		t.Fatalf("Expire() error = %v", err)
	}
	if ttl, err := s.TTL(ctx, "forever"); err != nil || ttl <= 0 {
		t.Fatalf("TTL(expiring) = %v, %v, want positive nil", ttl, err)
	}

	if err := s.Expire(ctx, "forever", 0); err != nil {
		t.Fatalf("Expire(immediate) error = %v", err)
	}
	if s.Exists(ctx, "forever") {
		t.Fatal("Expire(immediate) should delete key")
	}
	if err := s.Expire(ctx, "missing", time.Second); !errors.Is(err, derror.ErrKeyNotFound) {
		t.Fatalf("Expire(missing) error = %v, want %v", err, derror.ErrKeyNotFound)
	}
}

// TestStorageClearAndPing verifies Clear and Ping 测试清空与连通检查
func TestStorageClearAndPing(t *testing.T) {
	ctx := context.Background()
	s := NewStorage()
	_ = s.Set(ctx, "a", 1, 0)
	_ = s.Set(ctx, "b", 2, 0)

	if err := s.Ping(ctx); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
	if err := s.Clear(ctx); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}
	keys, err := s.Keys(ctx, "*")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}
	if len(keys) != 0 {
		t.Fatalf("Clear() left keys: %v", keys)
	}
}

// TestZeroValueStorageReturnsErrors verifies zero value storage is safe TestZeroValueStorageReturnsErrors 验证零值存储调用安全
func TestZeroValueStorageReturnsErrors(t *testing.T) {
	var s Storage
	ctx := context.Background()

	if err := s.Set(ctx, "k", "v", 0); err == nil {
		t.Fatal("Set() error = nil, want nil cache error")
	}
	if _, err := s.Get(ctx, "k"); err == nil {
		t.Fatal("Get() error = nil, want nil cache error")
	}
	if s.Exists(ctx, "k") {
		t.Fatal("Exists() should return false for nil cache")
	}
	if _, err := s.TTL(ctx, "k"); err == nil {
		t.Fatal("TTL() error = nil, want nil cache error")
	}
	if err := s.Delete(ctx); err != nil {
		t.Fatalf("Delete(empty) error = %v, want nil", err)
	}
	if err := s.Delete(ctx, "k"); err == nil {
		t.Fatal("Delete() error = nil, want nil cache error")
	}
}

// TestStorageContextCancellation verifies memory storage respects context cancellation TestStorageContextCancellation 验证内存存储遵守上下文取消语义。
func TestStorageContextCancellation(t *testing.T) {
	s := NewStorage()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := s.Set(ctx, "k", "v", 0); !errors.Is(err, context.Canceled) {
		t.Fatalf("Set(canceled) error = %v, want %v", err, context.Canceled)
	}
	if _, err := s.Get(ctx, "k"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Get(canceled) error = %v, want %v", err, context.Canceled)
	}
	if s.Exists(ctx, "k") {
		t.Fatal("Exists(canceled) should return false")
	}
	if _, err := s.Keys(ctx, "*"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Keys(canceled) error = %v, want %v", err, context.Canceled)
	}
	if err := s.Expire(ctx, "k", time.Second); !errors.Is(err, context.Canceled) {
		t.Fatalf("Expire(canceled) error = %v, want %v", err, context.Canceled)
	}
	if _, err := s.TTL(ctx, "k"); !errors.Is(err, context.Canceled) {
		t.Fatalf("TTL(canceled) error = %v, want %v", err, context.Canceled)
	}
	if err := s.Clear(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Clear(canceled) error = %v, want %v", err, context.Canceled)
	}
	if err := s.Ping(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Ping(canceled) error = %v, want %v", err, context.Canceled)
	}
}
