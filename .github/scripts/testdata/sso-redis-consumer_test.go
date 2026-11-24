package consumer_test

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/Zany2/dtoken-go/contract"
	"github.com/Zany2/dtoken-go/sso"
	ssoredis "github.com/Zany2/dtoken-go/sso/storage/redis"
	redisstorage "github.com/Zany2/dtoken-go/storage/redis"
	goredis "github.com/redis/go-redis/v9"
)

// TestSSOConsumerRedisComposition checks published constructors and ownership without a Redis service. TestSSOConsumerRedisComposition 无需 Redis 服务即可检查发布构造入口及所有权。
func TestSSOConsumerRedisComposition(t *testing.T) {
	var _ contract.Storage = (*redisstorage.Storage)(nil)
	var _ contract.AtomicStorage = (*redisstorage.Storage)(nil)
	if server, err := ssoredis.NewServer("://invalid"); err == nil || server != nil {
		t.Fatal("invalid Redis URL accepted")
	}
	if server, err := ssoredis.NewServerFromConfig(nil); err == nil || server != nil {
		t.Fatal("nil Redis config accepted")
	}
	for _, fromStorage := range []bool{false, true} {
		for _, owned := range []bool{false, true} {
			client := goredis.NewClient(&goredis.Options{
				Addr: "redis.invalid:6379", MaxRetries: -1,
				Dialer: func(context.Context, string, string) (net.Conn, error) {
					t.Error("constructor or Close attempted a Redis connection")
					return nil, errors.New("network disabled in consumer fixture")
				},
			})
			t.Cleanup(func() { _ = client.Close() })
			options := []sso.Option{sso.WithStorageOwnership(owned)}
			var server *sso.Server
			if fromStorage {
				server = ssoredis.NewServerFromStorage(redisstorage.NewStorageFromClient(client), options...)
			} else {
				server = ssoredis.NewServerFromClient(client, options...)
			}
			if err := server.Close(); err != nil {
				t.Fatal(err)
			}
			err := client.Close()
			if owned && !errors.Is(err, goredis.ErrClosed) || !owned && err != nil {
				t.Fatalf("storage=%v owned=%v close=%v", fromStorage, owned, err)
			}
		}
	}
}
