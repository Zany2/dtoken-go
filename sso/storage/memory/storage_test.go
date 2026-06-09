package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/derror"
)

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
