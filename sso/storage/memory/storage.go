// @Author daixk 2026/05/29
package memory

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
)

// Storage is the built-in in-memory SSO storage. Storage 是 SSO 内置内存存储。
type Storage struct {
	mu     sync.RWMutex
	values map[string]item
}

type item struct {
	value    any
	expireAt time.Time
}

// New creates built-in in-memory storage. New 创建内置内存存储。
func New() *Storage {
	return &Storage{values: make(map[string]item)}
}

// Set stores a key-value pair with optional expiration. Set 写入键值对，并可设置过期时间。
func (s *Storage) Set(_ context.Context, key string, value any, expiration time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	stored := item{value: value}
	if expiration > 0 {
		stored.expireAt = time.Now().Add(expiration)
	}
	s.values[key] = stored
	return nil
}

// Get gets value by key and returns nil when missing. Get 根据键读取值，键不存在时返回 nil。
func (s *Storage) Get(_ context.Context, key string) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	stored, ok := s.values[key]
	if !ok {
		return nil, nil
	}
	if !stored.expireAt.IsZero() && time.Now().After(stored.expireAt) {
		delete(s.values, key)
		return nil, nil
	}
	return stored.value, nil
}

// GetAndDelete gets and deletes key atomically. GetAndDelete 原子地读取并删除键。
func (s *Storage) GetAndDelete(_ context.Context, key string) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	stored, ok := s.values[key]
	if !ok {
		return nil, nil
	}
	if !stored.expireAt.IsZero() && time.Now().After(stored.expireAt) {
		delete(s.values, key)
		return nil, nil
	}
	delete(s.values, key)
	return stored.value, nil
}

// Delete deletes one or more keys. Delete 删除一个或多个键。
func (s *Storage) Delete(_ context.Context, keys ...string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, key := range keys {
		delete(s.values, key)
	}
	return nil
}

// Exists checks whether key exists. Exists 检查键是否存在。
func (s *Storage) Exists(ctx context.Context, key string) bool {
	value, _ := s.Get(ctx, key)
	return value != nil
}

// Expire sets key expiration and returns an error when the key is missing. Expire 设置键过期时间，键不存在时返回错误。
func (s *Storage) Expire(_ context.Context, key string, expiration time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	stored, ok := s.values[key]
	if !ok {
		return errors.New("key not found")
	}
	if expiration <= 0 {
		delete(s.values, key)
		return nil
	}
	stored.expireAt = time.Now().Add(expiration)
	s.values[key] = stored
	return nil
}

// TTL gets remaining lifetime of key using TTL sentinel values. TTL 获取键剩余生存时间。
func (s *Storage) TTL(_ context.Context, key string) (time.Duration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	stored := s.values[key]
	if stored.value == nil {
		return adapter.TTLNotFound, nil
	}
	if !stored.expireAt.IsZero() && time.Now().After(stored.expireAt) {
		delete(s.values, key)
		return adapter.TTLNotFound, nil
	}
	if stored.expireAt.IsZero() {
		return adapter.TTLNoExpire, nil
	}
	return time.Until(stored.expireAt), nil
}

// Ping checks whether storage is reachable. Ping 检查存储是否可达。
func (s *Storage) Ping(context.Context) error { return nil }
