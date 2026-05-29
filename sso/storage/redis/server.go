// @Author daixk 2026/05/29
package redis

import (
	"github.com/Zany2/dtoken-go/com/storage/redis"
	"github.com/Zany2/dtoken-go/sso"
)

// NewServer creates an SSO server backed by Redis storage. NewServer 创建使用 Redis 存储的 SSO 服务端。
func NewServer(redisURL string, options ...sso.Option) (*sso.Server, error) {
	storage, err := redis.NewStorage(redisURL)
	if err != nil {
		return nil, err
	}
	options = append([]sso.Option{sso.WithStorage(storage)}, options...)
	return sso.NewServer(options...), nil
}

// NewServerFromConfig creates an SSO server from Redis config. NewServerFromConfig 使用 Redis 配置创建 SSO 服务端。
func NewServerFromConfig(cfg *redis.Config, options ...sso.Option) (*sso.Server, error) {
	storage, err := redis.NewStorageFromConfig(cfg)
	if err != nil {
		return nil, err
	}
	options = append([]sso.Option{sso.WithStorage(storage)}, options...)
	return sso.NewServer(options...), nil
}

// NewServerFromStorage creates an SSO server from an existing Redis storage. NewServerFromStorage 使用已有 Redis 存储创建 SSO 服务端。
func NewServerFromStorage(storage *redis.Storage, options ...sso.Option) *sso.Server {
	options = append([]sso.Option{sso.WithStorage(storage)}, options...)
	return sso.NewServer(options...)
}
