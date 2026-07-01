// @Author daixk 2026/05/29
package memory

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/derror"
)

// Storage is the built-in in-memory SSO storage. Storage 是 SSO 内置内存存储。
type Storage struct {
	mu     sync.RWMutex
	values map[string]item
}

var _ adapter.Storage = (*Storage)(nil)
var _ adapter.AtomicStorage = (*Storage)(nil)

type item struct {
	value    any
	expireAt time.Time
}

// New creates built-in in-memory storage. New 创建内置内存存储。
func New() *Storage {
	return &Storage{values: make(map[string]item)}
}

// Set stores a key-value pair with optional expiration. Set 写入键值对，并可设置过期时间。
func (s *Storage) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	if err := checkContext(ctx); err != nil {
		return err
	}
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
func (s *Storage) Get(ctx context.Context, key string) (any, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
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
func (s *Storage) GetAndDelete(ctx context.Context, key string) (any, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
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

// GetAndDeleteMany gets and deletes key and extra keys atomically. GetAndDeleteMany 原子地读取并删除主键，同时删除附加键。
func (s *Storage) GetAndDeleteMany(ctx context.Context, key string, deleteKeys ...string) (any, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
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
	for _, deleteKey := range deleteKeys {
		delete(s.values, deleteKey)
	}
	return stored.value, nil
}

// SetIfAbsent stores a key only when it does not exist. SetIfAbsent 仅在键不存在时写入。
func (s *Storage) SetIfAbsent(ctx context.Context, key string, value any, expiration time.Duration) (bool, error) {
	if err := s.ensureReady(); err != nil {
		return false, err
	}
	if err := checkContext(ctx); err != nil {
		return false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	stored, ok := s.values[key]
	if ok && (stored.expireAt.IsZero() || time.Now().Before(stored.expireAt)) {
		return false, nil
	}
	next := item{value: value}
	if expiration > 0 {
		next.expireAt = time.Now().Add(expiration)
	}
	s.values[key] = next
	return true, nil
}

// Delete deletes one or more keys. Delete 删除一个或多个键。
func (s *Storage) Delete(ctx context.Context, keys ...string) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	if err := checkContext(ctx); err != nil {
		return err
	}
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
func (s *Storage) Expire(ctx context.Context, key string, expiration time.Duration) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	if err := checkContext(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	stored, ok := s.values[key]
	if !ok {
		return derror.ErrKeyNotFound
	}
	if !stored.expireAt.IsZero() && time.Now().After(stored.expireAt) {
		delete(s.values, key)
		return derror.ErrKeyNotFound
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
func (s *Storage) TTL(ctx context.Context, key string) (time.Duration, error) {
	if err := s.ensureReady(); err != nil {
		return 0, err
	}
	if err := checkContext(ctx); err != nil {
		return 0, err
	}
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
func (s *Storage) Ping(ctx context.Context) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	return checkContext(ctx)
}

func (s *Storage) ensureReady() error {
	if s == nil || s.values == nil {
		return errors.New("sso memory storage is nil")
	}
	return nil
}

func checkContext(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}
