// @Author daixk 2025/12/22 15:56:00
package storagetest

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/derror"
)

// StorageFactory creates a fresh storage instance for each contract run. StorageFactory 为每次契约测试创建新的存储实例。
type StorageFactory func(t *testing.T) adapter.FullStorage

// RunStorageContract verifies shared storage semantics. RunStorageContract 验证通用存储语义契约。
func RunStorageContract(t *testing.T, factory StorageFactory) {
	t.Helper()

	t.Run("basic operations", func(t *testing.T) {
		ctx := context.Background()
		storage := factory(t)

		if err := storage.Set(ctx, "contract:user:1", "alice", 0); err != nil {
			t.Fatalf("Set() error = %v", err)
		}
		if !storage.Exists(ctx, "contract:user:1") {
			t.Fatal("Exists() should return true")
		}
		got, err := storage.Get(ctx, "contract:user:1")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if got != "alice" {
			t.Fatalf("Get() = %v, want alice", got)
		}

		got, err = storage.Get(ctx, "contract:missing")
		if err != nil || got != nil {
			t.Fatalf("Get(missing) = %v, %v, want nil nil", got, err)
		}
	})

	t.Run("atomic get and delete", func(t *testing.T) {
		ctx := context.Background()
		storage := factory(t)

		if err := storage.Set(ctx, "contract:atomic", "value", 0); err != nil {
			t.Fatalf("Set() error = %v", err)
		}
		got, err := storage.GetAndDelete(ctx, "contract:atomic")
		if err != nil {
			t.Fatalf("GetAndDelete() error = %v", err)
		}
		if got != "value" {
			t.Fatalf("GetAndDelete() = %v, want value", got)
		}
		if storage.Exists(ctx, "contract:atomic") {
			t.Fatal("GetAndDelete() should remove key")
		}

		got, err = storage.GetAndDelete(ctx, "contract:missing")
		if err != nil || got != nil {
			t.Fatalf("GetAndDelete(missing) = %v, %v, want nil nil", got, err)
		}
	})

	t.Run("atomic get delete many", func(t *testing.T) {
		ctx := context.Background()
		storage := factory(t)

		if err := storage.Set(ctx, "contract:atomic:primary", "value", 0); err != nil {
			t.Fatalf("Set(primary) error = %v", err)
		}
		if err := storage.Set(ctx, "contract:atomic:extra1", "a", 0); err != nil {
			t.Fatalf("Set(extra1) error = %v", err)
		}
		if err := storage.Set(ctx, "contract:atomic:extra2", "b", 0); err != nil {
			t.Fatalf("Set(extra2) error = %v", err)
		}
		got, err := storage.GetAndDeleteMany(ctx, "contract:atomic:primary", "contract:atomic:extra1", "contract:atomic:extra2")
		if err != nil {
			t.Fatalf("GetAndDeleteMany() error = %v", err)
		}
		if got != "value" {
			t.Fatalf("GetAndDeleteMany() = %v, want value", got)
		}
		if storage.Exists(ctx, "contract:atomic:primary") ||
			storage.Exists(ctx, "contract:atomic:extra1") ||
			storage.Exists(ctx, "contract:atomic:extra2") {
			t.Fatal("GetAndDeleteMany() should remove primary and extra keys")
		}

		if err := storage.Set(ctx, "contract:atomic:extra3", "c", 0); err != nil {
			t.Fatalf("Set(extra3) error = %v", err)
		}
		got, err = storage.GetAndDeleteMany(ctx, "contract:atomic:missing", "contract:atomic:extra3")
		if err != nil || got != nil {
			t.Fatalf("GetAndDeleteMany(missing) = %v, %v, want nil nil", got, err)
		}
		if !storage.Exists(ctx, "contract:atomic:extra3") {
			t.Fatal("GetAndDeleteMany(missing) should keep extra keys")
		}
	})

	t.Run("atomic set if absent", func(t *testing.T) {
		ctx := context.Background()
		storage := factory(t)

		ok, err := storage.SetIfAbsent(ctx, "contract:setnx", "first", 0)
		if err != nil {
			t.Fatalf("SetIfAbsent(first) error = %v", err)
		}
		if !ok {
			t.Fatal("SetIfAbsent(first) = false, want true")
		}
		ok, err = storage.SetIfAbsent(ctx, "contract:setnx", "second", 0)
		if err != nil {
			t.Fatalf("SetIfAbsent(second) error = %v", err)
		}
		if ok {
			t.Fatal("SetIfAbsent(second) = true, want false")
		}
		got, err := storage.Get(ctx, "contract:setnx")
		if err != nil {
			t.Fatalf("Get(setnx) error = %v", err)
		}
		if got != "first" {
			t.Fatalf("Get(setnx) = %v, want first", got)
		}
	})

	t.Run("ttl and expire", func(t *testing.T) {
		ctx := context.Background()
		storage := factory(t)

		if ttl, err := storage.TTL(ctx, "contract:missing"); err != nil || ttl != adapter.TTLNotFound {
			t.Fatalf("TTL(missing) = %v, %v, want %v nil", ttl, err, adapter.TTLNotFound)
		}

		if err := storage.Set(ctx, "contract:forever", "value", 0); err != nil {
			t.Fatalf("Set(no-expire) error = %v", err)
		}
		if ttl, err := storage.TTL(ctx, "contract:forever"); err != nil || ttl != adapter.TTLNoExpire {
			t.Fatalf("TTL(no-expire) = %v, %v, want %v nil", ttl, err, adapter.TTLNoExpire)
		}

		if err := storage.Set(ctx, "contract:ttl", "value", time.Second); err != nil {
			t.Fatalf("Set(ttl) error = %v", err)
		}
		if ttl, err := storage.TTL(ctx, "contract:ttl"); err != nil || ttl <= 0 || ttl > time.Second {
			t.Fatalf("TTL(ttl) = %v, %v, want within (0, 1s]", ttl, err)
		}

		if err := storage.Expire(ctx, "contract:ttl", 500*time.Millisecond); err != nil {
			t.Fatalf("Expire() error = %v", err)
		}
		if ttl, err := storage.TTL(ctx, "contract:ttl"); err != nil || ttl <= 0 || ttl > 500*time.Millisecond {
			t.Fatalf("TTL(after Expire) = %v, %v, want within (0, 500ms]", ttl, err)
		}

		if err := storage.Expire(ctx, "contract:ttl", 0); err != nil {
			t.Fatalf("Expire(immediate) error = %v", err)
		}
		if storage.Exists(ctx, "contract:ttl") {
			t.Fatal("Expire(immediate) should remove key")
		}
		if ttl, err := storage.TTL(ctx, "contract:ttl"); err != nil || ttl != adapter.TTLNotFound {
			t.Fatalf("TTL(after immediate Expire) = %v, %v, want %v nil", ttl, err, adapter.TTLNotFound)
		}

		if err := storage.Expire(ctx, "contract:missing", time.Second); !errors.Is(err, derror.ErrKeyNotFound) {
			t.Fatalf("Expire(missing) error = %v, want %v", err, derror.ErrKeyNotFound)
		}
	})

	t.Run("keys delete and clear", func(t *testing.T) {
		ctx := context.Background()
		storage := factory(t)

		if err := storage.Set(ctx, "contract:user:1", "a", 0); err != nil {
			t.Fatalf("Set() error = %v", err)
		}
		if err := storage.Set(ctx, "contract:user:2", "b", 0); err != nil {
			t.Fatalf("Set() error = %v", err)
		}
		if err := storage.Set(ctx, "contract:team:1", "c", 0); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		keys, err := storage.Keys(ctx, "contract:user:?")
		if err != nil {
			t.Fatalf("Keys() error = %v", err)
		}
		sort.Strings(keys)
		if !reflect.DeepEqual(keys, []string{"contract:user:1", "contract:user:2"}) {
			t.Fatalf("Keys(user:?) = %v", keys)
		}

		keys, err = storage.Keys(ctx, "contract:missing:*")
		if err != nil {
			t.Fatalf("Keys(missing) error = %v", err)
		}
		if keys == nil || len(keys) != 0 {
			t.Fatalf("Keys(missing) = %#v, want empty non-nil slice", keys)
		}

		if err := storage.Delete(ctx); err != nil {
			t.Fatalf("Delete(empty) error = %v, want nil", err)
		}
		if err := storage.Delete(ctx, "contract:user:1", "contract:user:2"); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}
		if storage.Exists(ctx, "contract:user:1") || storage.Exists(ctx, "contract:user:2") {
			t.Fatal("Delete() should remove provided keys")
		}

		if err := storage.Clear(ctx); err != nil {
			t.Fatalf("Clear() error = %v", err)
		}
		keys, err = storage.Keys(ctx, "contract:*")
		if err != nil {
			t.Fatalf("Keys(after Clear) error = %v", err)
		}
		if keys == nil || len(keys) != 0 {
			t.Fatalf("Keys(after Clear) = %#v, want empty non-nil slice", keys)
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		storage := factory(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		if err := storage.Set(ctx, "contract:canceled", "v", 0); !errors.Is(err, context.Canceled) {
			t.Fatalf("Set(canceled) error = %v, want %v", err, context.Canceled)
		}
		if _, err := storage.Get(ctx, "contract:canceled"); !errors.Is(err, context.Canceled) {
			t.Fatalf("Get(canceled) error = %v, want %v", err, context.Canceled)
		}
		if storage.Exists(ctx, "contract:canceled") {
			t.Fatal("Exists(canceled) should return false")
		}
		if _, err := storage.Keys(ctx, "contract:*"); !errors.Is(err, context.Canceled) {
			t.Fatalf("Keys(canceled) error = %v, want %v", err, context.Canceled)
		}
		if err := storage.Expire(ctx, "contract:canceled", time.Second); !errors.Is(err, context.Canceled) {
			t.Fatalf("Expire(canceled) error = %v, want %v", err, context.Canceled)
		}
		if _, err := storage.TTL(ctx, "contract:canceled"); !errors.Is(err, context.Canceled) {
			t.Fatalf("TTL(canceled) error = %v, want %v", err, context.Canceled)
		}
		if err := storage.Clear(ctx); !errors.Is(err, context.Canceled) {
			t.Fatalf("Clear(canceled) error = %v, want %v", err, context.Canceled)
		}
		if err := storage.Ping(ctx); !errors.Is(err, context.Canceled) {
			t.Fatalf("Ping(canceled) error = %v, want %v", err, context.Canceled)
		}
	})
}
