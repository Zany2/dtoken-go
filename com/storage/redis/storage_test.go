package redis

import (
	"context"
	"testing"

	redisv9 "github.com/redis/go-redis/v9"
)

// TestNewStorageFromClient verifies client injection behavior 测试客户端注入行为
func TestNewStorageFromClient(t *testing.T) {
	client := redisv9.NewClient(&redisv9.Options{Addr: "127.0.0.1:0"})
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
