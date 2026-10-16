// @Author daixk 2026/05/29
package redis

import (
	"github.com/Zany2/dtoken-go/com/storage/redis"
	"github.com/Zany2/dtoken-go/sso"
	goredis "github.com/redis/go-redis/v9"
)

// NewServer creates an SSO server backed by Redis storage. NewServer 创建使用 Redis 存储的 SSO 服务端。
// The created storage and its ownership take precedence over options. 新建存储及其所有权优先于传入选项。
func NewServer(redisURL string, options ...sso.Option) (*sso.Server, error) {
	storage, err := redis.NewStorage(redisURL)
	if err != nil {
		return nil, err
	}
	return newServerWithOwnedStorage(storage, options...), nil
}

// NewServerFromConfig creates an SSO server from Redis config. NewServerFromConfig 使用 Redis 配置创建 SSO 服务端。
// The created storage and its ownership take precedence over options. 新建存储及其所有权优先于传入选项。
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

// NewServerFromClient creates an SSO server from an existing Redis client. NewServerFromClient 使用已有 Redis 客户端创建 SSO 服务端。
// It does not ping or close the client by default; options may override storage and ownership. 默认不探测或关闭客户端；选项可覆盖存储和所有权。
func NewServerFromClient(client *goredis.Client, options ...sso.Option) *sso.Server {
	storage := redis.NewStorageFromClient(client)
	options = append([]sso.Option{sso.WithStorage(storage)}, options...)
	return sso.NewServer(options...)
}

// NewServerFromStorage creates an SSO server from an existing Redis storage. NewServerFromStorage 使用已有 Redis 存储创建 SSO 服务端。
// It does not ping or close the storage by default; options may override storage and ownership. 默认不探测或关闭存储；选项可覆盖存储和所有权。
func NewServerFromStorage(storage *redis.Storage, options ...sso.Option) *sso.Server {
	options = append([]sso.Option{sso.WithStorage(storage)}, options...)
	return sso.NewServer(options...)
}
