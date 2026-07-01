// @Author daixk 2025/12/22 15:56:00
package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/redis/go-redis/v9"
)

// TTL constants define Redis TTL sentinel values TTL 常量定义 Redis TTL 哨兵值
const (
	TTLNoExpire = adapter.TTLNoExpire
	TTLNotFound = adapter.TTLNotFound
)

// Config defines Redis storage configuration Redis 存储配置
type Config struct {
	// Host specifies the Redis server host Redis 服务器主机地址，例如 "127.0.0.1" 或 "redis.examples.com"
	Host string
	// Port specifies the Redis server port Redis 服务器端口号，默认通常为 6379
	Port int
	// Password specifies the Redis auth password Redis 认证密码，若未设置鉴权可留空
	Password string
	// Database specifies the Redis database index Redis 数据库索引（0 到 15），默认使用 0
	Database int
	// PoolSize specifies the maximum active connections 连接池中最大活跃连接数，影响并发性能；建议根据业务负载调整
	PoolSize int
	// DialTimeout specifies the TCP dial timeout 建立 TCP 连接的超时时间，0 表示无限制（不推荐）
	DialTimeout time.Duration
	// ReadTimeout specifies the Redis read timeout 从 Redis 读取响应的超时时间，0 表示无限制
	ReadTimeout time.Duration
	// WriteTimeout specifies the Redis write timeout 向 Redis 发送命令的超时时间，0 表示无限制
	WriteTimeout time.Duration
	// PoolTimeout specifies the wait timeout for pool acquisition 从连接池获取连接时的等待超时时间
	PoolTimeout time.Duration
	// OperationTimeout specifies the timeout for each storage operation 每个存储操作（如 Get/Set/Delete）的上下文超时时间。
	OperationTimeout time.Duration
}

// Storage implements Redis backed storage Redis 存储实现
type Storage struct {
	client           *redis.Client // client stores Redis client. client 存储 Redis 客户端。
	operationTimeout time.Duration // operationTimeout stores per-operation timeout. operationTimeout 存储单次操作超时时间。
}

// Interface assertion keeps storage contract checked at compile time 接口断言在编译期检查存储契约
var _ adapter.Storage = (*Storage)(nil)
var _ adapter.AtomicStorage = (*Storage)(nil)
var _ adapter.FullStorage = (*Storage)(nil)

var getAndDeleteManyScript = redis.NewScript(`
local value = redis.call("GET", KEYS[1])
if not value then
	return false
end
redis.call("DEL", unpack(KEYS))
return value
`)

// NewStorage creates storage from a Redis URL 通过 Redis URL 创建存储
func NewStorage(url string) (*Storage, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis url: %w", err)
	}
	opTimeout := 3 * time.Second

	client := redis.NewClient(opts)
	pingCtx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()

	// Test Redis connectivity 测试连接
	if err = client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &Storage{
		client:           client,
		operationTimeout: opTimeout,
	}, nil
}

// NewStorageFromConfig creates storage from config 通过配置创建存储
func NewStorageFromConfig(cfg *Config) (*Storage, error) {
	if cfg == nil {
		return nil, errors.New("redis config is nil")
	}

	opTimeout := cfg.OperationTimeout
	if opTimeout <= 0 {
		opTimeout = 3 * time.Second
	}
	pingCtx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()

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

	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &Storage{
		client:           client,
		operationTimeout: opTimeout,
	}, nil
}

// NewStorageFromClient creates storage from an existing Redis client 从已有的 Redis 客户端创建存储
func NewStorageFromClient(client *redis.Client) *Storage {
	return &Storage{
		client: client,
	}
}

// Set stores a key value pair 设置键值对
func (s *Storage) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	ctx, cancel := s.withOperationTimeout(ctx)
	defer cancel()

	return s.client.Set(ctx, key, value, normalizeSetExpiration(expiration)).Err()
}

// Get retrieves the value 获取值
func (s *Storage) Get(ctx context.Context, key string) (any, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	ctx, cancel := s.withOperationTimeout(ctx)
	defer cancel()

	val, err := s.client.Get(ctx, key).Result()
	if err != nil {
		// Return nil nil when the key is missing 键不存在时返回 nil, nil（这是正常情况，不是错误）
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	return val, nil
}

// GetAndDelete atomically gets and deletes the key 原子获取并删除键
func (s *Storage) GetAndDelete(ctx context.Context, key string) (any, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	ctx, cancel := s.withOperationTimeout(ctx)
	defer cancel()

	val, err := s.client.GetDel(ctx, key).Result()
	if err != nil {
		// Return nil nil when the key is missing 键不存在时返回 nil, nil（这是正常情况，不是错误）
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	return val, nil
}

// GetAndDeleteMany atomically gets and deletes a key and deletes extra keys 原子获取并删除主键，同时删除附加键
func (s *Storage) GetAndDeleteMany(ctx context.Context, key string, deleteKeys ...string) (any, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	ctx, cancel := s.withOperationTimeout(ctx)
	defer cancel()

	keys := make([]string, 0, len(deleteKeys)+1)
	keys = append(keys, key)
	keys = append(keys, deleteKeys...)
	val, err := getAndDeleteManyScript.Run(ctx, s.client, keys).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	if val == nil {
		return nil, nil
	}
	if missing, ok := val.(bool); ok && !missing {
		return nil, nil
	}
	return val, nil
}

// SetIfAbsent stores a key only when it does not exist 仅当键不存在时写入
func (s *Storage) SetIfAbsent(ctx context.Context, key string, value any, expiration time.Duration) (bool, error) {
	if err := s.ensureReady(); err != nil {
		return false, err
	}
	ctx, cancel := s.withOperationTimeout(ctx)
	defer cancel()

	return s.client.SetNX(ctx, key, value, normalizeSetExpiration(expiration)).Result()
}

// Delete removes keys 删除键
func (s *Storage) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	if err := s.ensureReady(); err != nil {
		return err
	}
	ctx, cancel := s.withOperationTimeout(ctx)
	defer cancel()

	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = key
	}

	return s.client.Del(ctx, fullKeys...).Err()
}

// Exists checks whether the key exists 检查键是否存在
func (s *Storage) Exists(ctx context.Context, key string) bool {
	if err := s.ensureReady(); err != nil {
		return false
	}
	ctx, cancel := s.withOperationTimeout(ctx)
	defer cancel()

	result, err := s.client.Exists(ctx, key).Result()
	return err == nil && result > 0
}

// Keys gets all keys matching the pattern 获取匹配模式的所有键
func (s *Storage) Keys(ctx context.Context, pattern string) ([]string, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	if pattern == "" {
		pattern = "*"
	}
	ctx, cancel := s.withOperationTimeout(ctx)
	defer cancel()

	var (
		cursor uint64
		result = make([]string, 0)
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

// Expire sets the expiration for the key 设置键的过期时间
func (s *Storage) Expire(ctx context.Context, key string, expiration time.Duration) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	ctx, cancel := s.withOperationTimeout(ctx)
	defer cancel()

	if expiration <= 0 {
		deleted, err := s.client.Del(ctx, key).Result()
		if err != nil {
			return err
		}
		if deleted == 0 {
			return derror.ErrKeyNotFound
		}
		return nil
	}

	ok, err := s.client.PExpire(ctx, key, expiration).Result()
	if err != nil {
		// Return network or Redis command errors 网络错误、Redis 报错（如 EXPIRE -1）等
		return err
	}
	if !ok {
		// Handle Redis zero result when the key is missing Redis 返回 0：key 不存在或已过期
		return derror.ErrKeyNotFound
	}
	return nil
}

// TTL gets the remaining lifetime for the key 获取键的剩余生存时间
func (s *Storage) TTL(ctx context.Context, key string) (time.Duration, error) {
	if err := s.ensureReady(); err != nil {
		return 0, err
	}
	ctx, cancel := s.withOperationTimeout(ctx)
	defer cancel()

	ttl, err := s.client.PTTL(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	return normalizeTTL(ttl), nil
}

// Clear removes all stored data 清空所有数据
func (s *Storage) Clear(ctx context.Context) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	ctx, cancel := s.withOperationTimeout(ctx)
	defer cancel()

	return s.client.FlushDB(ctx).Err()
}

// Ping checks the Redis connection 检查连接
func (s *Storage) Ping(ctx context.Context) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	ctx, cancel := s.withOperationTimeout(ctx)
	defer cancel()

	return s.client.Ping(ctx).Err()
}

// Close closes the Redis connection 关闭连接
func (s *Storage) Close() error {
	if s == nil || s.client == nil {
		return nil
	}
	return s.client.Close()
}

// GetClient returns the Redis client 获取 Redis 客户端
func (s *Storage) GetClient() *redis.Client {
	if s == nil {
		return nil
	}
	return s.client
}

// ensureReady checks storage dependencies ensureReady 检查存储依赖是否可用
func (s *Storage) ensureReady() error {
	if s == nil || s.client == nil {
		return errors.New("redis storage client is nil")
	}
	return nil
}

// normalizeTTL converts Redis sentinel durations to adapter sentinels normalizeTTL 转换 Redis TTL 哨兵值
func normalizeTTL(ttl time.Duration) time.Duration {
	switch ttl {
	case -time.Second, -time.Millisecond, adapter.TTLNoExpire:
		return adapter.TTLNoExpire
	case -2 * time.Second, -2 * time.Millisecond, adapter.TTLNotFound:
		return adapter.TTLNotFound
	default:
		return ttl
	}
}

// normalizeSetExpiration converts non-positive set TTL to no expiration. normalizeSetExpiration 将 Set 的非正过期时间转换为永不过期。
func normalizeSetExpiration(expiration time.Duration) time.Duration {
	if expiration <= 0 {
		return 0
	}
	return expiration
}

// withOperationTimeout applies configured operation timeout. withOperationTimeout 应用已配置的单次操作超时时间。
func (s *Storage) withOperationTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if s == nil || s.operationTimeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, s.operationTimeout)
}
