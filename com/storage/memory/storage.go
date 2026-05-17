// @Author daixk 2025/12/22 15:56:00
package memory

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/patrickmn/go-cache"
)

// ErrKeyNotFound indicates the key is missing or expired 表示键不存在或已过期。
var ErrKeyNotFound = errors.New("key not found")

// TTL constants define memory TTL sentinel values TTL 常量定义内存 TTL 哨兵值
const (
	TTLNoExpire = adapter.TTLNoExpire
	TTLNotFound = adapter.TTLNotFound
)

// Storage implements in-memory storage with go-cache 基于 go-cache 的内存存储实现。
type Storage struct {
	c  *cache.Cache
	mu sync.Mutex
}

// Interface assertion keeps storage contract checked at compile time 接口断言在编译期检查存储契约
var _ adapter.Storage = (*Storage)(nil)
var _ adapter.AtomicStorage = (*Storage)(nil)
var _ adapter.FullStorage = (*Storage)(nil)

// NewStorage creates a new memory storage instance 创建一个新的内存存储实例
func NewStorage() *Storage {
	return &Storage{
		c: cache.New(time.Duration(time.Minute*10), time.Duration(time.Minute*10)),
	}
}

// Set stores a key value pair 设置键值对
func (s *Storage) Set(_ context.Context, key string, value any, expiration time.Duration) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	if expiration <= 0 {
		s.c.Set(key, value, cache.NoExpiration) // Keep the key without expiration 永不过期
	} else {
		s.c.Set(key, value, expiration)
	}
	return nil
}

// Get retrieves the value for a key 获取指定键的值
func (s *Storage) Get(_ context.Context, key string) (any, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	if val, found := s.c.Get(key); found {
		return val, nil
	}
	// Return nil nil when the key is missing 键不存在时返回 nil, nil（这是正常情况，不是错误）
	return nil, nil
}

// GetAndDelete atomically gets and deletes a key 原子地获取并删除指定键
func (s *Storage) GetAndDelete(_ context.Context, key string) (any, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	val, found := s.c.Get(key)
	if !found {
		// Return nil nil when the key is missing 键不存在时返回 nil, nil（这是正常情况，不是错误）
		return nil, nil
	}

	s.c.Delete(key)
	return val, nil
}

// Delete removes one or more keys 删除一个或多个键
func (s *Storage) Delete(_ context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	if err := s.ensureReady(); err != nil {
		return err
	}
	for _, key := range keys {
		s.c.Delete(key)
	}
	return nil
}

// Exists checks whether a key exists and is not expired 检查键是否存在且未过期
func (s *Storage) Exists(_ context.Context, key string) bool {
	if err := s.ensureReady(); err != nil {
		return false
	}
	_, found := s.c.Get(key)
	return found
}

// Keys returns all keys matching the pattern 返回匹配指定模式的所有键
func (s *Storage) Keys(_ context.Context, pattern string) ([]string, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	if pattern == "" {
		pattern = "*"
	}
	items := s.c.Items()
	now := time.Now().UnixNano()
	keys := make([]string, 0, len(items))

	for k, it := range items {
		// Check whether the entry is expired 检查是否已过期（Expiration > 0 表示有 TTL）
		if it.Expiration > 0 && now >= it.Expiration {
			// Delete expired entry proactively 主动清理（避免后续重复处理）
			s.c.Delete(k)
			continue
		}
		if matchPattern(k, pattern) {
			keys = append(keys, k)
		}
	}
	return keys, nil
}

// Expire sets a new expiration for the key 为指定键设置新的过期时间
func (s *Storage) Expire(_ context.Context, key string, expiration time.Duration) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	_, found := s.c.Get(key)
	if !found {
		return ErrKeyNotFound
	}

	if expiration <= 0 {
		s.c.Delete(key) // Delete the key for immediate expiration 立即过期等于删除
	} else {
		// Reset the value with a new TTL 重新设置值 + 新 TTL
		val, _ := s.c.Get(key) // The key is confirmed to exist 已确认存在
		s.c.Set(key, val, expiration)
	}

	return nil
}

// TTL returns the remaining lifetime for a key 获取指定键的剩余生存时间
func (s *Storage) TTL(_ context.Context, key string) (time.Duration, error) {
	if err := s.ensureReady(); err != nil {
		return 0, err
	}
	_, expirationTime, found := s.c.GetWithExpiration(key)
	if !found {
		return TTLNotFound, nil
	}
	if expirationTime.IsZero() {
		return TTLNoExpire, nil
	}
	ttl := time.Until(expirationTime)
	if ttl <= 0 {
		// Handle unlikely expired edge cases defensively 边缘情况兜底：理论上 go-cache 不会返回已过期项，但防御性处理
		return TTLNotFound, nil
	}
	return ttl, nil
}

// Clear removes all stored data 清空所有数据。
func (s *Storage) Clear(_ context.Context) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	s.c.Flush()
	return nil
}

// Ping checks whether storage is available 检查存储是否可用
func (s *Storage) Ping(_ context.Context) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	return nil
}

// ensureReady checks storage dependencies ensureReady 检查存储依赖是否可用
func (s *Storage) ensureReady() error {
	if s == nil || s.c == nil {
		return errors.New("memory storage cache is nil")
	}
	return nil
}

// matchPattern implements Redis style wildcard matching 实现 Redis 风格的通配符匹配（支持 *, ?, \ 转义）
func matchPattern(key, pattern string) bool {
	return wildcardMatch(key, pattern, 0, 0)
}

// wildcardMatch performs recursive backtracking matching 递归回溯匹配
func wildcardMatch(key, pattern string, i, j int) bool {
	for j < len(pattern) {
		switch pattern[j] {
		case '\\':
			// Escape the next character 转义下一个字符
			if j+1 >= len(pattern) {
				return i == len(key) // Match only the end when pattern ends with \ 以 \ 结尾，只匹配到末尾
			}
			j++
			if i >= len(key) || key[i] != pattern[j] {
				return false
			}
			i++
			j++

		case '?':
			if i >= len(key) {
				return false
			}
			i++
			j++

		case '*':
			// Skip consecutive wildcard stars 跳过连续的 *
			for j < len(pattern) && pattern[j] == '*' {
				j++
			}
			if j == len(pattern) {
				return true // Match the remaining suffix when * is the last character * 是最后一个字符，匹配剩余所有
			}
			// Try matching the remaining pattern from the current position 尝试从当前位置开始匹配剩余 pattern
			for i <= len(key) {
				if wildcardMatch(key, pattern, i, j) {
					return true
				}
				i++
			}
			return false

		default:
			if i >= len(key) || key[i] != pattern[j] {
				return false
			}
			i++
			j++
		}
	}
	return i == len(key)
}
