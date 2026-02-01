// @Author daixk 2026/1/21 13:28:00
package redis

import (
	"context"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"
)

// ErrKeyNotFound 表示键不存在或已过期。
var ErrKeyNotFound = errors.New("key not found")

// Config Redis配置
type Config struct {
	// Host Redis 服务器主机地址，例如 "127.0.0.1" 或 "redis.example.com"
	Host string
	// Port Redis 服务器端口号，默认通常为 6379
	Port int
	// Password Redis 认证密码，若未设置鉴权可留空
	Password string
	// Database Redis 数据库索引（0 到 15），默认使用 0
	Database int
	// PoolSize 连接池中最大活跃连接数，影响并发性能；建议根据业务负载调整
	PoolSize int
	// DialTimeout 建立 TCP 连接的超时时间，0 表示无限制（不推荐）
	DialTimeout time.Duration
	// ReadTimeout 从 Redis 读取响应的超时时间，0 表示无限制
	ReadTimeout time.Duration
	// WriteTimeout 向 Redis 发送命令的超时时间，0 表示无限制
	WriteTimeout time.Duration
	// PoolTimeout 从连接池获取连接时的等待超时时间，
	PoolTimeout time.Duration
	// OperationTimeout 每个存储操作（如 Get/Set/Delete）的上下文超时时间。
	OperationTimeout time.Duration
}

// Storage Redis存储实现
type Storage struct {
	client *redis.Client
}

// NewStorage 通过Redis URL创建存储
func NewStorage(url string) (*Storage, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis url: %w", err)
	}

	client := redis.NewClient(opts)

	// 测试连接
	if err = client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &Storage{
		client: client,
	}, nil
}

// NewStorageFromConfig 通过配置创建存储
func NewStorageFromConfig(cfg *Config) (*Storage, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.Database,
		PoolSize:     cfg.PoolSize,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		PoolTimeout:  cfg.PoolTimeout,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	opTimeout := cfg.OperationTimeout
	if opTimeout <= 0 {
		opTimeout = 3 * time.Second
	}

	return &Storage{
		client: client,
	}, nil
}

// NewStorageFromClient 从已有的Redis客户端创建存储
func NewStorageFromClient(client *redis.Client) *Storage {
	return &Storage{
		client: client,
	}
}

// Set 设置键值对
func (s *Storage) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	return s.client.Set(ctx, key, value, expiration).Err()
}

// Get 获取值
func (s *Storage) Get(ctx context.Context, key string) (any, error) {
	val, err := s.client.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	return val, nil
}

// GetAndDelete 原子获取并删除键
func (s *Storage) GetAndDelete(ctx context.Context, key string) (any, error) {
	return s.client.GetDel(ctx, key).Result()
}

// Delete 删除键
func (s *Storage) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = key
	}

	return s.client.Del(ctx, fullKeys...).Err()
}

// Exists 检查键是否存在
func (s *Storage) Exists(ctx context.Context, key string) bool {
	result, err := s.client.Exists(ctx, key).Result()
	return err == nil && result > 0
}

// Keys 获取匹配模式的所有键
func (s *Storage) Keys(ctx context.Context, pattern string) ([]string, error) {
	var (
		cursor uint64
		result []string
	)

	for {
		keys, next, err := s.client.Scan(ctx, cursor, pattern, 1000).Result()
		if err != nil {
			return nil, err
		}
		result = append(result, keys...)
		cursor = next
		if cursor == 0 {
			break
		}
	}

	return result, nil
}

// Expire 设置键的过期时间
func (s *Storage) Expire(ctx context.Context, key string, expiration time.Duration) error {
	ok, err := s.client.Expire(ctx, key, expiration).Result()
	if err != nil {
		// 网络错误、Redis 报错（如 EXPIRE -1）等
		return err
	}
	if !ok {
		// Redis 返回 0：key 不存在或已过期
		return ErrKeyNotFound
	}
	return nil
}

// TTL 获取键的剩余生存时间
func (s *Storage) TTL(ctx context.Context, key string) (time.Duration, error) {
	return s.client.TTL(ctx, key).Result()
}

// Clear 清空所有数据
func (s *Storage) Clear(ctx context.Context) error {
	var cursor uint64
	for {
		keys, next, err := s.client.Scan(ctx, cursor, "*", 1000).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			if err := s.client.Unlink(ctx, keys...).Err(); err != nil {
				return err
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return nil
}

// Ping 检查连接
func (s *Storage) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

// Close 关闭连接
func (s *Storage) Close() error {
	return s.client.Close()
}

// GetClient 获取Redis客户端
func (s *Storage) GetClient() *redis.Client {
	return s.client
}
