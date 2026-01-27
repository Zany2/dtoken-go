package adapter

import (
	"context"
	"time"
)

// Storage 定义存储接口，用于存储Token和Session数据
type Storage interface {
	// ============== 基本操作 ==============

	// Set 设置键值对，可选过期时间（0表示永不过期）
	Set(ctx context.Context, key string, value any, expiration time.Duration) error

	// Get 获取键对应的值，键不存在时返回nil
	Get(ctx context.Context, key string) (any, error)

	// GetAndDelete 原子获取并删除键
	GetAndDelete(ctx context.Context, key string) (any, error)

	// Delete 删除一个或多个键
	Delete(ctx context.Context, keys ...string) error

	// Exists 检查键是否存在
	Exists(ctx context.Context, key string) bool

	// ============== 键管理 ==============

	// Keys 获取匹配模式的所有键（如："user:*"）
	Keys(ctx context.Context, pattern string) ([]string, error)

	// Expire 设置键的过期时间
	Expire(ctx context.Context, key string, expiration time.Duration) error

	// TTL 获取键的剩余生存时间（-1表示永不过期，-2表示键不存在）
	TTL(ctx context.Context, key string) (time.Duration, error)

	// ============== 工具方法 ==============

	// Clear 清空所有数据（谨慎使用，主要用于测试）
	Clear(ctx context.Context) error

	// Ping 检查存储是否可访问
	Ping(ctx context.Context) error
}
