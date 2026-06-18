// @Author daixk 2025/12/22 15:56:00
package redis

import (
	"context"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/adapter/storagetest"
	redisv9 "github.com/redis/go-redis/v9"
)

const (
	testRedisAddr     = "192.168.19.104:6379"
	testRedisHost     = "192.168.19.104"
	testRedisPort     = 6379
	testRedisPassword = "root"
	testRedisDatabase = 0
)

// TestStorageContract verifies Redis storage follows the shared storage contract. TestStorageContract 验证 Redis 存储符合共享存储契约。
func TestStorageContract(t *testing.T) {
	storagetest.RunStorageContract(t, func(t *testing.T) adapter.FullStorage {
		storage, err := NewStorageFromConfig(&Config{
			Host:             testRedisHost,
			Port:             testRedisPort,
			Password:         testRedisPassword,
			Database:         testRedisDatabase,
			OperationTimeout: 3 * time.Second,
		})
		if err != nil {
			t.Skipf("skip redis storage contract: %v", err)
		}
		t.Cleanup(func() {
			_ = storage.Close()
		})
		return storage
	})
}

// TestNewStorageFromClient verifies client injection behavior 测试客户端注入行为
func TestNewStorageFromClient(t *testing.T) {
	client := redisv9.NewClient(&redisv9.Options{
		Addr:     testRedisAddr,
		Password: testRedisPassword,
		DB:       testRedisDatabase,
	})
	storage := NewStorageFromClient(client)

	if storage == nil {
		t.Fatal("NewStorageFromClient() returned nil")
	}
	if storage.GetClient() != client {
		t.Fatal("GetClient() did not return injected client")
	}
	if err := storage.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

// TestNewStorageFromConfigRejectsNil verifies nil config validation 测试空配置校验
func TestNewStorageFromConfigRejectsNil(t *testing.T) {
	storage, err := NewStorageFromConfig(nil)
	if err == nil {
		if storage != nil {
			_ = storage.Close()
		}
		t.Fatal("NewStorageFromConfig(nil) error = nil, want error")
	}
	if storage != nil {
		t.Fatalf("NewStorageFromConfig(nil) storage = %v, want nil", storage)
	}
}

// TestNewStorageFromClientHasNoOperationTimeout verifies client injection keeps caller context behavior 测试客户端注入不强制覆盖调用方上下文
func TestNewStorageFromClientHasNoOperationTimeout(t *testing.T) {
	storage := NewStorageFromClient(redisv9.NewClient(&redisv9.Options{
		Addr:     testRedisAddr,
		Password: testRedisPassword,
		DB:       testRedisDatabase,
	}))
	ctx, cancel := storage.withOperationTimeout(context.Background())
	defer cancel()

	if _, ok := ctx.Deadline(); ok {
		t.Fatal("NewStorageFromClient() should not set operation timeout by default")
	}
}

// TestWithOperationTimeoutAppliesConfiguredTimeout verifies configured operation timeout 测试配置的单次操作超时生效
func TestWithOperationTimeoutAppliesConfiguredTimeout(t *testing.T) {
	storage := &Storage{operationTimeout: time.Second}
	ctx, cancel := storage.withOperationTimeout(context.Background())
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("withOperationTimeout() should set deadline")
	}
	if remaining := time.Until(deadline); remaining <= 0 || remaining > time.Second {
		t.Fatalf("withOperationTimeout() remaining = %v, want within 1s", remaining)
	}
}

// TestNewStorageFromConfigConnects verifies configured Redis connectivity TestNewStorageFromConfigConnects 验证指定 Redis 配置可连接。
func TestNewStorageFromConfigConnects(t *testing.T) {
	storage, err := NewStorageFromConfig(&Config{
		Host:             testRedisHost,
		Port:             testRedisPort,
		Password:         testRedisPassword,
		Database:         testRedisDatabase,
		OperationTimeout: 3 * time.Second,
	})
	if err != nil {
		t.Skipf("skip redis storage connectivity test: %v", err)
	}
	defer func() {
		if err := storage.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}()

	if err := storage.Ping(context.Background()); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
}

// TestNormalizeTTL verifies Redis TTL sentinels match adapter contract TestNormalizeTTL 验证 Redis TTL 哨兵值符合适配器契约
func TestNormalizeTTL(t *testing.T) {
	tests := []struct {
		name string
		ttl  time.Duration
		want time.Duration
	}{
		{name: "no expire seconds", ttl: -time.Second, want: TTLNoExpire},
		{name: "not found seconds", ttl: -2 * time.Second, want: TTLNotFound},
		{name: "no expire milliseconds", ttl: -time.Millisecond, want: TTLNoExpire},
		{name: "not found milliseconds", ttl: -2 * time.Millisecond, want: TTLNotFound},
		{name: "no expire adapter", ttl: TTLNoExpire, want: TTLNoExpire},
		{name: "not found adapter", ttl: TTLNotFound, want: TTLNotFound},
		{name: "positive", ttl: time.Minute, want: time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeTTL(tt.ttl); got != tt.want {
				t.Fatalf("normalizeTTL() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestNormalizeSetExpiration verifies Set expiration matches memory storage semantics TestNormalizeSetExpiration 验证 Set 过期时间与内存存储语义一致。
func TestNormalizeSetExpiration(t *testing.T) {
	tests := []struct {
		name       string
		expiration time.Duration
		want       time.Duration
	}{
		{name: "zero means no expiration", expiration: 0, want: 0},
		{name: "negative means no expiration", expiration: -time.Second, want: 0},
		{name: "positive unchanged", expiration: time.Minute, want: time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeSetExpiration(tt.expiration); got != tt.want {
				t.Fatalf("normalizeSetExpiration() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestNewStorageRejectsInvalidURL verifies Redis URL parse errors 测试 Redis URL 解析错误
func TestNewStorageRejectsInvalidURL(t *testing.T) {
	storage, err := NewStorage("://bad-url")
	if err == nil {
		if storage != nil {
			_ = storage.Close()
		}
		t.Fatal("NewStorage() error = nil, want parse error")
	}
	if storage != nil {
		t.Fatalf("NewStorage() storage = %v, want nil", storage)
	}
}

// TestDeleteWithoutKeysSkipsClient verifies empty delete is a no-op 测试空删除不会访问客户端
func TestDeleteWithoutKeysSkipsClient(t *testing.T) {
	if err := (&Storage{}).Delete(context.Background()); err != nil {
		t.Fatalf("Delete() error = %v, want nil", err)
	}
}

// TestNilClientReturnsErrors verifies nil client does not panic TestNilClientReturnsErrors 验证空客户端不会 panic
func TestNilClientReturnsErrors(t *testing.T) {
	storage := NewStorageFromClient(nil)
	if storage == nil {
		t.Fatal("NewStorageFromClient(nil) returned nil")
	}
	if storage.GetClient() != nil {
		t.Fatal("GetClient() should return nil client")
	}
	if err := storage.Set(context.Background(), "k", "v", 0); err == nil {
		t.Fatal("Set() error = nil, want nil client error")
	}
	if _, err := storage.Get(context.Background(), "k"); err == nil {
		t.Fatal("Get() error = nil, want nil client error")
	}
	if storage.Exists(context.Background(), "k") {
		t.Fatal("Exists() should return false for nil client")
	}
	if err := storage.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}
