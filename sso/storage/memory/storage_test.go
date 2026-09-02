package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/derror"
)

// TestStorageBasicSemantics verifies the Storage Basic Semantics scenario. TestStorageBasicSemantics 验证对应的内存存储场景。
func TestStorageBasicSemantics(t *testing.T) {
	ctx := context.Background()
	storage := New()

	if err := storage.Set(ctx, "token", "value", time.Second); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	value, err := storage.Get(ctx, "token")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if value != "value" {
		t.Fatalf("Get() = %v, want value", value)
	}

	deleted, err := storage.GetAndDelete(ctx, "token")
	if err != nil {
		t.Fatalf("GetAndDelete() error = %v", err)
	}
	if deleted != "value" {
		t.Fatalf("GetAndDelete() = %v, want value", deleted)
	}
	value, err = storage.Get(ctx, "token")
	if err != nil {
		t.Fatalf("Get() after delete error = %v", err)
	}
	if value != nil {
		t.Fatalf("Get() after delete = %v, want nil", value)
	}
}

// TestStorageAtomicAndDeleteSemantics verifies conditional writes, existence checks, and multi-key deletion. TestStorageAtomicAndDeleteSemantics 验证条件写入、存在性检查和多键删除。
func TestStorageAtomicAndDeleteSemantics(t *testing.T) {
	ctx := context.Background()
	storage := New()

	stored, err := storage.SetIfAbsent(ctx, "atomic", "first", 0)
	if err != nil || !stored {
		t.Fatalf("SetIfAbsent(first) = %v, %v, want true nil", stored, err)
	}
	stored, err = storage.SetIfAbsent(ctx, "atomic", "second", 0)
	if err != nil || stored {
		t.Fatalf("SetIfAbsent(second) = %v, %v, want false nil", stored, err)
	}
	if !storage.Exists(ctx, "atomic") {
		t.Fatal("Exists(atomic) = false, want true")
	}
	value, err := storage.Get(ctx, "atomic")
	if err != nil || value != "first" {
		t.Fatalf("Get(atomic) = %v, %v, want first nil", value, err)
	}
	if err = storage.Set(ctx, "other", "value", 0); err != nil {
		t.Fatalf("Set(other) error = %v", err)
	}
	if err = storage.Delete(ctx, "atomic", "other"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if storage.Exists(ctx, "atomic") || storage.Exists(ctx, "other") {
		t.Fatal("Delete() should remove every requested key")
	}

	if err = storage.Set(ctx, "expired-atomic", "value", time.Nanosecond); err != nil {
		t.Fatalf("Set(expired-atomic) error = %v", err)
	}
	time.Sleep(time.Millisecond)
	value, err = storage.GetAndDelete(ctx, "expired-atomic")
	if err != nil || value != nil {
		t.Fatalf("GetAndDelete(expired) = %v, %v, want nil nil", value, err)
	}
	stored, err = storage.SetIfAbsent(ctx, "expired-atomic", "replacement", 0)
	if err != nil || !stored {
		t.Fatalf("SetIfAbsent(expired) = %v, %v, want true nil", stored, err)
	}
}

// TestStorageTTLAndExpireSemantics verifies the Storage TTL And Expire Semantics scenario. TestStorageTTLAndExpireSemantics 验证对应的内存存储场景。
func TestStorageTTLAndExpireSemantics(t *testing.T) {
	ctx := context.Background()
	storage := New()

	if ttl, err := storage.TTL(ctx, "missing"); err != nil || ttl != adapter.TTLNotFound {
		t.Fatalf("TTL() missing = %v, %v; want TTLNotFound nil", ttl, err)
	}
	if err := storage.Set(ctx, "forever", "value", 0); err != nil {
		t.Fatalf("Set() no expire error = %v", err)
	}
	if ttl, err := storage.TTL(ctx, "forever"); err != nil || ttl != adapter.TTLNoExpire {
		t.Fatalf("TTL() no expire = %v, %v; want TTLNoExpire nil", ttl, err)
	}
	if err := storage.Expire(ctx, "forever", time.Second); err != nil {
		t.Fatalf("Expire() error = %v", err)
	}
	if ttl, err := storage.TTL(ctx, "forever"); err != nil || ttl <= 0 || ttl > time.Second {
		t.Fatalf("TTL() after expire = %v, %v; want 0..1s nil", ttl, err)
	}
	if err := storage.Expire(ctx, "forever", 0); err != nil {
		t.Fatalf("Expire() delete error = %v", err)
	}
	if ttl, err := storage.TTL(ctx, "forever"); err != nil || ttl != adapter.TTLNotFound {
		t.Fatalf("TTL() after delete = %v, %v; want TTLNotFound nil", ttl, err)
	}
	if err := storage.Expire(ctx, "missing", time.Second); !errors.Is(err, derror.ErrKeyNotFound) {
		t.Fatalf("Expire() missing error = %v, want ErrKeyNotFound", err)
	}
}

// TestStorageExpiredKeyCannotBeRenewed verifies the Storage Expired Key Cannot Be Renewed scenario. TestStorageExpiredKeyCannotBeRenewed 验证对应的内存存储场景。
func TestStorageExpiredKeyCannotBeRenewed(t *testing.T) {
	ctx := context.Background()
	storage := New()

	if err := storage.Set(ctx, "expired", "value", time.Nanosecond); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	time.Sleep(time.Millisecond)
	if err := storage.Expire(ctx, "expired", time.Second); !errors.Is(err, derror.ErrKeyNotFound) {
		t.Fatalf("Expire() expired error = %v, want ErrKeyNotFound", err)
	}
}

// TestStorageContextAndNilSemantics verifies the Storage Context And Nil Semantics scenario. TestStorageContextAndNilSemantics 验证对应的内存存储场景。
func TestStorageContextAndNilSemantics(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	storage := New()
	if err := storage.Set(ctx, "key", "value", 0); !errors.Is(err, context.Canceled) {
		t.Fatalf("Set() canceled error = %v, want context.Canceled", err)
	}

	var nilStorage *Storage
	if err := nilStorage.Ping(context.Background()); err == nil {
		t.Fatal("Ping() nil storage error = nil, want error")
	}
}
