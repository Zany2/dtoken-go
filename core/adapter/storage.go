// @Author daixk 2025/12/22 15:56:00
package adapter

import (
	"context"
	"time"
)

const (
	// TTLNoExpire means the key exists without expiration. TTLNoExpire 表示键存在且永不过期。
	TTLNoExpire = time.Duration(-1)
	// TTLNotFound means the key does not exist. TTLNotFound 表示键不存在。
	TTLNotFound = time.Duration(-2)
)

// Storage defines the minimal key-value contract used by core auth flows. Storage 定义核心鉴权流程使用的最小键值存储契约。
type Storage interface {
	// Set stores a key-value pair with optional expiration. Set 写入键值对，并可设置过期时间。
	Set(ctx context.Context, key string, value any, expiration time.Duration) error
	// Get gets value by key and returns nil, nil when missing. Get 根据键读取值，键不存在时返回 nil, nil。
	Get(ctx context.Context, key string) (any, error)
	// Delete deletes one or more keys. Delete 删除一个或多个键。
	Delete(ctx context.Context, keys ...string) error
	// Exists checks whether key exists. Exists 检查键是否存在。
	Exists(ctx context.Context, key string) bool
	// Expire sets key expiration and returns an error when the key is missing. Expire 设置键过期时间，键不存在时返回错误。
	Expire(ctx context.Context, key string, expiration time.Duration) error
	// TTL gets remaining lifetime of key using TTL sentinel values. TTL 获取键剩余生存时间，并使用 TTL 哨兵值表达特殊状态。
	TTL(ctx context.Context, key string) (time.Duration, error)
	// Ping checks whether storage is reachable. Ping 检查存储是否可达。
	Ping(ctx context.Context) error
}

// AtomicStorage defines optional storage operations that must be atomic. AtomicStorage 定义必须具备原子性的可选存储操作。
type AtomicStorage interface {
	// GetAndDelete gets and deletes key atomically. GetAndDelete 原子地读取并删除键。
	GetAndDelete(ctx context.Context, key string) (any, error)
	// GetAndDeleteMany gets and deletes key, then deletes extra keys atomically. GetAndDeleteMany 原子地读取并删除主键，同时删除附加键。
	GetAndDeleteMany(ctx context.Context, key string, deleteKeys ...string) (any, error)
	// SetIfAbsent stores a key only when it does not exist. SetIfAbsent 仅在键不存在时写入键值。
	SetIfAbsent(ctx context.Context, key string, value any, expiration time.Duration) (bool, error)
}

// ScannerStorage defines optional key scanning capability. ScannerStorage 定义可选的键扫描能力。
type ScannerStorage interface {
	// Keys gets all keys matching pattern. Keys 获取匹配模式的全部键。
	Keys(ctx context.Context, pattern string) ([]string, error)
}

// AdminStorage defines optional administrative storage capability. AdminStorage 定义可选的管理型存储能力。
type AdminStorage interface {
	// Clear clears all data. Clear 清空全部数据。
	Clear(ctx context.Context) error
}

// FullStorage groups all built-in storage capabilities. FullStorage 组合全部内置存储能力。
type FullStorage interface {
	Storage
	AtomicStorage
	ScannerStorage
	AdminStorage
}
