package memory

import (
	"context"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
)

// TestWritesReclaimUnvisitedExpiredKeys verifies both write paths reclaim expired entries without reading them. TestWritesReclaimUnvisitedExpiredKeys 验证两种写入路径无需读取旧键即可回收过期条目。
func TestWritesReclaimUnvisitedExpiredKeys(t *testing.T) {
	ctx := context.Background()
	for _, operation := range []string{"set", "set-if-absent", "set-if-present"} {
		t.Run(operation, func(t *testing.T) {
			storage := New()
			if err := storage.Set(ctx, "permanent", "keep", 0); err != nil {
				t.Fatal(err)
			}
			if err := storage.Set(ctx, "live", "keep", time.Hour); err != nil {
				t.Fatal(err)
			}
			// Seed elapsed deadlines without relying on wall-clock sleeps. 直接设置已过去的截止时间，避免依赖等待。
			storage.values["expired"] = item{value: "old", expireAt: time.Now().Add(-time.Hour)}
			storage.nextCleanup = time.Now().Add(-time.Minute)
			switch operation {
			case "set":
				if err := storage.Set(ctx, "new", "value", time.Hour); err != nil {
					t.Fatal(err)
				}
			case "set-if-absent":
				if stored, err := storage.SetIfAbsent(ctx, "new", "value", time.Hour); err != nil || !stored {
					t.Fatalf("SetIfAbsent = %v, %v", stored, err)
				}
			case "set-if-present":
				if stored, err := storage.SetIfAbsent(ctx, "live", "replacement", time.Hour); err != nil || stored {
					t.Fatalf("SetIfAbsent(existing) = %v, %v", stored, err)
				}
			}
			if _, exists := storage.values["expired"]; exists {
				t.Fatal("unvisited expired key remains allocated")
			}
			for _, key := range []string{"permanent", "live"} {
				if value, err := storage.Get(ctx, key); err != nil || value != "keep" {
					t.Fatalf("Get(%s) = %v, %v", key, value, err)
				}
			}
		})
	}
}

// TestNilValuesRetainKeySemantics verifies nil payloads still have consistent existence and TTL. TestNilValuesRetainKeySemantics 验证 nil 载荷仍具有一致的存在性和有效期。
func TestNilValuesRetainKeySemantics(t *testing.T) {
	ctx := context.Background()
	storage := New()
	if err := storage.Set(ctx, "nil", nil, 0); err != nil {
		t.Fatal(err)
	}
	if !storage.Exists(ctx, "nil") {
		t.Fatal("nil payload should not hide an existing key")
	}
	if ttl, err := storage.TTL(ctx, "nil"); err != nil || ttl != adapter.TTLNoExpire {
		t.Fatalf("TTL(nil) = %v, %v", ttl, err)
	}
	if stored, err := storage.SetIfAbsent(ctx, "nil", "replacement", 0); err != nil || stored {
		t.Fatalf("SetIfAbsent(nil) = %v, %v", stored, err)
	}
	if err := storage.Expire(ctx, "nil", time.Hour); err != nil {
		t.Fatal(err)
	}
	if ttl, err := storage.TTL(ctx, "nil"); err != nil || ttl <= 0 {
		t.Fatalf("TTL(nil with expiry) = %v, %v", ttl, err)
	}
	storage.values["nil"] = item{expireAt: time.Now().Add(-time.Hour)}
	if storage.Exists(ctx, "nil") {
		t.Fatal("expired nil payload should not exist")
	}
	if _, exists := storage.values["nil"]; exists {
		t.Fatal("expired nil payload was not reclaimed")
	}
}
