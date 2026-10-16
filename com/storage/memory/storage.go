// @Author daixk 2025/12/22 15:56:00
package memory

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/patrickmn/go-cache"
)

// TTL constants define memory TTL sentinel values TTL 常量定义内存 TTL 哨兵值
const (
	TTLNoExpire = adapter.TTLNoExpire
	TTLNotFound = adapter.TTLNotFound
)

// Storage implements in-memory storage with go-cache 基于 go-cache 的内存存储实现。
type Storage struct {
	c  *cache.Cache // c stores the underlying cache instance c 存储底层缓存实例。
	mu sync.Mutex   // mu serializes all mutations with compound storage operations. mu 将所有修改与复合存储操作串行化。
}

// Interface assertion keeps storage contract checked at compile time 接口断言在编译期检查存储契约
var (
	_ adapter.Storage       = (*Storage)(nil)
	_ adapter.AtomicStorage = (*Storage)(nil)
	_ adapter.FullStorage   = (*Storage)(nil)
)

// NewStorage creates a new memory storage instance 创建一个新的内存存储实例
func NewStorage() *Storage {
	return &Storage{
		c: cache.New(time.Minute*10, time.Minute*10),
	}
}

// Set stores a key value pair 设置键值对
func (s *Storage) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	if err := checkContext(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := checkContext(ctx); err != nil {
		return err
	}
	if expiration <= 0 {
		s.c.Set(key, value, cache.NoExpiration) // Keep the key without expiration 永不过期
	} else {
		if err := checkExpiration(expiration); err != nil {
			return err
		}
		s.c.Set(key, value, expiration)
	}
	return nil
}

// Get retrieves the value for a key 获取指定键的值
func (s *Storage) Get(ctx context.Context, key string) (any, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	if val, found := s.c.Get(key); found {
		return val, nil
	}
	// Return nil nil when the key is missing 键不存在时返回 nil, nil（这是正常情况，不是错误）
	return nil, nil
}

// GetAndDelete atomically gets and deletes a key 原子地获取并删除指定键
func (s *Storage) GetAndDelete(ctx context.Context, key string) (any, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	val, found := s.c.Get(key)
	if !found {
		// Return nil nil when the key is missing 键不存在时返回 nil, nil（这是正常情况，不是错误）
		return nil, nil
	}

	s.c.Delete(key)
	return val, nil
}

// SetIfAbsent stores a key only when it does not exist 仅当键不存在时写入
func (s *Storage) SetIfAbsent(ctx context.Context, key string, value any, expiration time.Duration) (bool, error) {
	if err := s.ensureReady(); err != nil {
		return false, err
	}
	if err := checkContext(ctx); err != nil {
		return false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := checkContext(ctx); err != nil {
		return false, err
	}
	if _, found := s.c.Get(key); found {
		return false, nil
	}
	if expiration <= 0 {
		s.c.Set(key, value, cache.NoExpiration)
	} else {
		if err := checkExpiration(expiration); err != nil {
			return false, err
		}
		s.c.Set(key, value, expiration)
	}
	return true, nil
}

// Delete removes one or more keys 删除一个或多个键
func (s *Storage) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	if err := s.ensureReady(); err != nil {
		return err
	}
	if err := checkContext(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := checkContext(ctx); err != nil {
		return err
	}
	for _, key := range keys {
		s.c.Delete(key)
	}
	return nil
}

// Exists checks whether a key exists and is not expired 检查键是否存在且未过期
func (s *Storage) Exists(ctx context.Context, key string) bool {
	if err := s.ensureReady(); err != nil {
		return false
	}
	if err := checkContext(ctx); err != nil {
		return false
	}
	_, found := s.c.Get(key)
	return found
}

// Keys returns all keys matching the pattern 返回匹配指定模式的所有键
func (s *Storage) Keys(ctx context.Context, pattern string) ([]string, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	if pattern == "" {
		pattern = "*"
	}
	items := s.c.Items()
	now := time.Now().UnixNano()
	keys := make([]string, 0, len(items))

	for k, it := range items {
		if err := checkContext(ctx); err != nil {
			return nil, err
		}

		// Check whether the entry is expired 检查是否已过期（Expiration > 0 表示有 TTL）
		if it.Expiration > 0 && now >= it.Expiration {
			// A snapshot cannot authorize deleting a concurrently replaced value. 快照不能用于删除可能已被并发替换的值。
			continue
		}
		if matchPattern(k, pattern) {
			keys = append(keys, k)
		}
	}
	return keys, nil
}

// Expire sets a new expiration for the key 为指定键设置新的过期时间
func (s *Storage) Expire(ctx context.Context, key string, expiration time.Duration) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	if err := checkContext(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := checkContext(ctx); err != nil {
		return err
	}
	val, found := s.c.Get(key)
	if !found {
		return derror.ErrKeyNotFound
	}

	if expiration <= 0 {
		s.c.Delete(key) // Delete the key for immediate expiration 立即过期等于删除
	} else {
		// Reset the value with a new TTL 重新设置值 + 新 TTL
		if err := checkExpiration(expiration); err != nil {
			return err
		}
		s.c.Set(key, val, expiration)
	}

	return nil
}

// TTL returns the remaining lifetime for a key 获取指定键的剩余生存时间
func (s *Storage) TTL(ctx context.Context, key string) (time.Duration, error) {
	if err := s.ensureReady(); err != nil {
		return 0, err
	}
	if err := checkContext(ctx); err != nil {
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
func (s *Storage) Clear(ctx context.Context) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	if err := checkContext(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := checkContext(ctx); err != nil {
		return err
	}
	s.c.Flush()
	return nil
}

// Ping checks whether storage is available 检查存储是否可用
func (s *Storage) Ping(ctx context.Context) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	if err := checkContext(ctx); err != nil {
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

// checkExpiration prevents go-cache's absolute UnixNano deadline from wrapping into a non-expiring value. checkExpiration 防止 go-cache 的绝对 UnixNano 截止时间溢出为永不过期。
func checkExpiration(expiration time.Duration) error {
	if time.Now().Add(expiration).After(time.Unix(0, math.MaxInt64)) {
		return fmt.Errorf("%w: memory storage expiration exceeds the UnixNano range", derror.ErrInvalidParam)
	}
	return nil
}

// checkContext returns context cancellation errors consistently with remote storage. checkContext 返回上下文取消错误以对齐远程存储语义。
func checkContext(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

// matchPattern implements Redis-style byte glob matching. matchPattern 实现 Redis 风格的字节通配符匹配。
func matchPattern(key, pattern string) bool {
	memo := make(map[[2]int]bool)
	visited := make(map[[2]int]bool)
	return wildcardMatch(key, pattern, 0, 0, memo, visited)
}

// wildcardMatch performs memoized glob matching. wildcardMatch 使用记忆化方式执行通配符匹配。
func wildcardMatch(key, pattern string, i, j int, memo, visited map[[2]int]bool) bool {
	state := [2]int{i, j}
	if visited[state] {
		return memo[state]
	}
	visited[state] = true

	matched := false
	switch {
	case j == len(pattern):
		matched = i == len(key)

	case pattern[j] == '*':
		for j < len(pattern) && pattern[j] == '*' {
			j++
		}
		if j == len(pattern) {
			matched = true
			break
		}
		for next := i; next <= len(key); next++ {
			if wildcardMatch(key, pattern, next, j, memo, visited) {
				matched = true
				break
			}
		}

	case i == len(key):
		matched = false

	case pattern[j] == '?':
		matched = wildcardMatch(key, pattern, i+1, j+1, memo, visited)

	case pattern[j] == '[':
		classMatched, next, valid := matchCharacterClass(key[i], pattern, j+1)
		matched = valid && classMatched && wildcardMatch(key, pattern, i+1, next, memo, visited)

	case pattern[j] == '\\':
		// A trailing escape is a literal backslash, matching Redis glob behavior. 末尾转义符按反斜杠字面量处理，与 Redis 通配规则一致。
		literalIndex := j
		if j+1 < len(pattern) {
			literalIndex = j + 1
		}
		matched = key[i] == pattern[literalIndex] && wildcardMatch(key, pattern, i+1, literalIndex+1, memo, visited)

	default:
		matched = key[i] == pattern[j] && wildcardMatch(key, pattern, i+1, j+1, memo, visited)
	}

	memo[state] = matched
	return matched
}

// matchCharacterClass matches Redis-style ranges, negation, and escapes. matchCharacterClass 匹配 Redis 风格的范围、取反与转义。
func matchCharacterClass(value byte, pattern string, start int) (matched bool, next int, valid bool) {
	index := start
	negated := index < len(pattern) && pattern[index] == '^'
	if negated {
		index++
	}

	for index < len(pattern) {
		if pattern[index] == ']' {
			if negated {
				matched = !matched
			}
			return matched, index + 1, true
		}

		current := pattern[index]
		if current == '\\' && index+1 < len(pattern) {
			index++
			current = pattern[index]
			if current == value {
				matched = true
			}
			index++
			continue
		}

		if index+2 < len(pattern) && pattern[index+1] == '-' && pattern[index+2] != ']' {
			end := pattern[index+2]
			if current > end {
				current, end = end, current
			}
			if value >= current && value <= end {
				matched = true
			}
			index += 3
			continue
		}

		if current == value {
			matched = true
		}
		index++
	}

	return false, 0, false
}
