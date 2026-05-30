// @Author daixk 2026/05/29
package sso

import (
	"github.com/Zany2/dtoken-go/core/adapter"
	jsoncodec "github.com/Zany2/dtoken-go/sso/codec/json"
	"github.com/Zany2/dtoken-go/sso/storage/memory"
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
type JSONCodec = jsoncodec.Codec

// MemoryStorage is the built-in in-memory storage. MemoryStorage 是内置内存存储。
type MemoryStorage = memory.Storage

// NewMemoryStorage creates built-in in-memory storage. NewMemoryStorage 创建内置内存存储。
func NewMemoryStorage() *MemoryStorage {
	return memory.New()
}
