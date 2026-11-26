// @Author daixk 2026/05/29
package redis

import (
	"github.com/Zany2/dtoken-go/sso"
	"github.com/Zany2/dtoken-go/storage/redis"
	goredis "github.com/redis/go-redis/v9"
)

// NewServer creates a Redis-backed SSO server whose new storage and ownership take precedence over options. NewServer 创建 Redis SSO 服务端，新建存储及其所有权优先于传入选项。
func NewServer(redisURL string, options ...sso.Option) (*sso.Server, error) {
	storage, err := redis.NewStorage(redisURL)
	if err != nil {
		return nil, err
	}
	return newServerWithOwnedStorage(storage, options...), nil
}

// NewServerFromConfig creates an SSO server from Redis config, giving the new storage and its ownership priority over options. NewServerFromConfig 按 Redis 配置创建 SSO 服务端，新建存储及其所有权优先于传入选项。
func NewServerFromConfig(cfg *redis.Config, options ...sso.Option) (*sso.Server, error) {
	storage, err := redis.NewStorageFromConfig(cfg)
	if err != nil {
		return nil, err
	}
	return newServerWithOwnedStorage(storage, options...), nil
}

// newServerWithOwnedStorage keeps a newly created Redis connection owned and reachable. newServerWithOwnedStorage 确保新建 Redis 连接始终由 Server 使用并负责关闭。
func newServerWithOwnedStorage(storage *redis.Storage, options ...sso.Option) *sso.Server {
	// Copy before appending so the caller's option slice remains unchanged. 追加前复制，避免修改调用方选项切片。
	options = append([]sso.Option(nil), options...)
	options = append(options, sso.WithStorage(storage), sso.WithStorageOwnership(true))
	return sso.NewServer(options...)
}

// NewServerFromClient wraps an existing Redis client without pinging or closing it by default; options may override storage and ownership. NewServerFromClient 使用已有 Redis 客户端创建 SSO 服务端，默认不探测或关闭客户端；选项可覆盖存储和所有权。
func NewServerFromClient(client *goredis.Client, options ...sso.Option) *sso.Server {
	storage := redis.NewStorageFromClient(client)
	options = append([]sso.Option{sso.WithStorage(storage)}, options...)
	return sso.NewServer(options...)
}

// NewServerFromStorage wraps existing Redis storage without pinging or closing it by default; options may override storage and ownership. NewServerFromStorage 使用已有 Redis 存储创建 SSO 服务端，默认不探测或关闭存储；选项可覆盖存储和所有权。
func NewServerFromStorage(storage *redis.Storage, options ...sso.Option) *sso.Server {
	options = append([]sso.Option{sso.WithStorage(storage)}, options...)
	return sso.NewServer(options...)
}
