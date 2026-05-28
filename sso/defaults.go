// @Author daixk 2026/05/29
package sso

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
)

const (
	// DefaultAuthType stores the default SSO namespace. DefaultAuthType 存储默认 SSO 命名空间。
	DefaultAuthType = "sso:"
	// DefaultKeyPrefix stores the default SSO storage key prefix. DefaultKeyPrefix 存储默认 SSO 存储键前缀。
	DefaultKeyPrefix = "dtoken:"
)

// Option configures a Server during construction. Option 配置 Server 构建过程。
type Option func(*serverBuildOptions)

type serverBuildOptions struct {
	authType   string
	keyPrefix  string
	storage    adapter.Storage
	serializer adapter.Codec
	config     *Config
}

// NewServer creates a Server with built-in JSON codec and memory storage. NewServer 使用内置 JSON 编解码和内存存储创建 Server。
func NewServer(options ...Option) *Server {
	opts := serverBuildOptions{
		authType:   DefaultAuthType,
		keyPrefix:  DefaultKeyPrefix,
		storage:    NewMemoryStorage(),
		serializer: JSONCodec{},
		config:     DefaultConfig(),
	}
	for _, option := range options {
		if option != nil {
			option(&opts)
		}
	}
	return NewServerWithConfig(opts.authType, opts.keyPrefix, opts.storage, opts.serializer, opts.config)
}

// WithAuthType sets the SSO namespace. WithAuthType 设置 SSO 命名空间。
func WithAuthType(authType string) Option {
	return func(opts *serverBuildOptions) {
		if authType != "" {
			opts.authType = authType
		}
	}
}

// WithKeyPrefix sets the storage key prefix. WithKeyPrefix 设置存储键前缀。
func WithKeyPrefix(keyPrefix string) Option {
	return func(opts *serverBuildOptions) {
		if keyPrefix != "" {
			opts.keyPrefix = keyPrefix
		}
	}
}

// WithStorage sets the storage adapter. WithStorage 设置存储适配器。
func WithStorage(storage adapter.Storage) Option {
	return func(opts *serverBuildOptions) {
		if storage != nil {
			opts.storage = storage
		}
	}
}

// WithCodec sets the codec adapter. WithCodec 设置编解码适配器。
func WithCodec(codec adapter.Codec) Option {
	return func(opts *serverBuildOptions) {
		if codec != nil {
			opts.serializer = codec
		}
	}
}

// WithConfig sets SSO config. WithConfig 设置 SSO 配置。
func WithConfig(cfg *Config) Option {
	return func(opts *serverBuildOptions) {
		if cfg != nil {
			opts.config = cfg
		}
	}
}

// JSONCodec is the built-in JSON codec. JSONCodec 是内置 JSON 编解码器。
type JSONCodec struct{}

// Name returns codec name. Name 返回编解码器名称。
func (JSONCodec) Name() string { return "json" }

// Encode encodes value to JSON. Encode 将值编码为 JSON。
func (JSONCodec) Encode(v any) ([]byte, error) { return json.Marshal(v) }

// Decode decodes JSON bytes into value. Decode 将 JSON 字节解码到目标值。
func (JSONCodec) Decode(data []byte, v any) error { return json.Unmarshal(data, v) }

// MemoryStorage is the built-in in-memory storage. MemoryStorage 是内置内存存储。
type MemoryStorage struct {
	mu     sync.RWMutex
	values map[string]memoryItem
}

type memoryItem struct {
	value    any
	expireAt time.Time
}

// NewMemoryStorage creates built-in in-memory storage. NewMemoryStorage 创建内置内存存储。
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{values: make(map[string]memoryItem)}
}

// Set stores a key-value pair with optional expiration. Set 写入键值对，并可设置过期时间。
func (s *MemoryStorage) Set(_ context.Context, key string, value any, expiration time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	item := memoryItem{value: value}
	if expiration > 0 {
		item.expireAt = time.Now().Add(expiration)
	}
	s.values[key] = item
	return nil
}

// Get gets value by key and returns nil when missing. Get 根据键读取值，键不存在时返回 nil。
func (s *MemoryStorage) Get(_ context.Context, key string) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.values[key]
	if !ok {
		return nil, nil
	}
	if !item.expireAt.IsZero() && time.Now().After(item.expireAt) {
		delete(s.values, key)
		return nil, nil
	}
	return item.value, nil
}

// GetAndDelete gets and deletes key atomically. GetAndDelete 原子地读取并删除键。
func (s *MemoryStorage) GetAndDelete(ctx context.Context, key string) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.values[key]
	if !ok {
		return nil, nil
	}
	if !item.expireAt.IsZero() && time.Now().After(item.expireAt) {
		delete(s.values, key)
		return nil, nil
	}
	delete(s.values, key)
	return item.value, nil
}

// Delete deletes one or more keys. Delete 删除一个或多个键。
func (s *MemoryStorage) Delete(_ context.Context, keys ...string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, key := range keys {
		delete(s.values, key)
	}
	return nil
}

// Exists checks whether key exists. Exists 检查键是否存在。
func (s *MemoryStorage) Exists(ctx context.Context, key string) bool {
	value, _ := s.Get(ctx, key)
	return value != nil
}

// Expire sets key expiration and returns an error when the key is missing. Expire 设置键过期时间，键不存在时返回错误。
func (s *MemoryStorage) Expire(_ context.Context, key string, expiration time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.values[key]
	if !ok {
		return errors.New("key not found")
	}
	if expiration <= 0 {
		delete(s.values, key)
		return nil
	}
	item.expireAt = time.Now().Add(expiration)
	s.values[key] = item
	return nil
}

// TTL gets remaining lifetime of key using TTL sentinel values. TTL 获取键剩余生存时间。
func (s *MemoryStorage) TTL(ctx context.Context, key string) (time.Duration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item := s.values[key]
	if item.value == nil {
		return adapter.TTLNotFound, nil
	}
	if !item.expireAt.IsZero() && time.Now().After(item.expireAt) {
		delete(s.values, key)
		return adapter.TTLNotFound, nil
	}
	if item.expireAt.IsZero() {
		return adapter.TTLNoExpire, nil
	}
	return time.Until(item.expireAt), nil
}

// Ping checks whether storage is reachable. Ping 检查存储是否可达。
func (s *MemoryStorage) Ping(context.Context) error { return nil }
