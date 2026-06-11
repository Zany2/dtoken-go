package redis

import (
	"testing"

	redisstorage "github.com/Zany2/dtoken-go/com/storage/redis"
	"github.com/Zany2/dtoken-go/sso"
	goredis "github.com/redis/go-redis/v9"
)

// TestNewServerRejectsInvalidRedisURL verifies invalid Redis URL is rejected. TestNewServerRejectsInvalidRedisURL 验证非法 Redis 地址会被拒绝。
func TestNewServerRejectsInvalidRedisURL(t *testing.T) {
	if server, err := NewServer("://bad-url"); err == nil || server != nil {
		t.Fatalf("NewServer(invalid) = %v, %v, want nil error", server, err)
	}
}

// TestNewServerFromConfigRejectsNil verifies nil config is rejected. TestNewServerFromConfigRejectsNil 验证空配置会被拒绝。
func TestNewServerFromConfigRejectsNil(t *testing.T) {
	if server, err := NewServerFromConfig(nil); err == nil || server != nil {
		t.Fatalf("NewServerFromConfig(nil) = %v, %v, want nil error", server, err)
	}
}

// TestNewServerFromClientAndStorage verifies constructor wrappers use supplied storage. TestNewServerFromClientAndStorage 验证构造包装方法使用传入存储。
func TestNewServerFromClientAndStorage(t *testing.T) {
	client := goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:6379"})
	t.Cleanup(func() {
		_ = client.Close()
	})

	server := NewServerFromClient(client, sso.WithAuthType("redis-sso"))
	if server == nil {
		t.Fatal("NewServerFromClient() = nil, want server")
	}

	storage := redisstorage.NewStorageFromClient(client)
	server = NewServerFromStorage(storage, sso.WithKeyPrefix("dtoken:sso:test:"))
	if server == nil {
		t.Fatal("NewServerFromStorage() = nil, want server")
	}
}
