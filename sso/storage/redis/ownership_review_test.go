package redis

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"

	redisstorage "github.com/Zany2/dtoken-go/com/storage/redis"
	"github.com/Zany2/dtoken-go/sso"
	goredis "github.com/redis/go-redis/v9"
)

// TestOwnedRedisStorageCannotBeOverridden verifies constructor options cannot orphan newly created storage. TestOwnedRedisStorageCannotBeOverridden 验证构造选项不能使新建存储失去使用及关闭责任方。
func TestOwnedRedisStorageCannotBeOverridden(t *testing.T) {
	dialErr := errors.New("review dial blocked")
	client := goredis.NewClient(&goredis.Options{
		Addr:       "redis.invalid:6379",
		MaxRetries: -1,
		Dialer: func(context.Context, string, string) (net.Conn, error) {
			return nil, dialErr
		},
	})
	t.Cleanup(func() { _ = client.Close() })
	storage := redisstorage.NewStorageFromClient(client)
	options := make([]sso.Option, 2, 4)
	options[0] = sso.WithStorage(sso.NewMemoryStorage())
	options[1] = sso.WithStorageOwnership(false)

	// Both URL and config constructors delegate to this ownership boundary. URL 和 Config 构造器共用此所有权边界。
	server := newServerWithOwnedStorage(storage, options...)
	t.Cleanup(func() { _ = server.Close() })
	if err := server.RegisterClient(&sso.Client{ClientID: "app"}); !errors.Is(err, sso.ErrStorageUnavailable) || !strings.Contains(err.Error(), dialErr.Error()) {
		t.Fatalf("RegisterClient() error = %v, want the supplied Redis storage error", err)
	}
	backing := options[:cap(options)]
	if backing[2] != nil || backing[3] != nil {
		t.Fatal("constructor overwrote the caller's option backing array")
	}
	for i := 0; i < 2; i++ {
		if err := server.Close(); err != nil {
			t.Fatalf("Close() call %d error = %v", i+1, err)
		}
	}
	if err := client.Close(); !errors.Is(err, goredis.ErrClosed) {
		t.Fatalf("client.Close() error = %v, want already closed by Server", err)
	}
}

// TestInjectedRedisOwnership verifies caller-owned clients stay open unless ownership is explicitly transferred. TestInjectedRedisOwnership 验证外部客户端默认不被关闭，显式转移所有权后才关闭。
func TestInjectedRedisOwnership(t *testing.T) {
	constructors := []struct {
		name string
		new  func(*goredis.Client, ...sso.Option) *sso.Server
	}{
		{"client", NewServerFromClient},
		{"storage", func(client *goredis.Client, options ...sso.Option) *sso.Server {
			return NewServerFromStorage(redisstorage.NewStorageFromClient(client), options...)
		}},
	}
	for _, constructor := range constructors {
		for _, owned := range []bool{false, true} {
			name := constructor.name + "/borrowed"
			if owned {
				name = constructor.name + "/owned"
			}
			t.Run(name, func(t *testing.T) {
				client := goredis.NewClient(&goredis.Options{
					Addr: "redis.invalid:6379",
					Dialer: func(context.Context, string, string) (net.Conn, error) {
						t.Error("injection or Close unexpectedly attempted a connection")
						return nil, errors.New("unexpected connection")
					},
				})
				t.Cleanup(func() { _ = client.Close() })
				var options []sso.Option
				if owned {
					options = append(options, sso.WithStorageOwnership(true))
				}
				server := constructor.new(client, options...)
				for i := 0; i < 2; i++ {
					if err := server.Close(); err != nil {
						t.Fatalf("Close() call %d error = %v", i+1, err)
					}
				}
				err := client.Close()
				if owned && !errors.Is(err, goredis.ErrClosed) {
					t.Fatalf("owned client Close() = %v, want ErrClosed", err)
				}
				if !owned && err != nil {
					t.Fatalf("borrowed client was already closed: %v", err)
				}
			})
		}
	}
}

// TestInjectedNilRedisFailsWithoutMemoryFallback verifies invalid injected dependencies fail on storage use. TestInjectedNilRedisFailsWithoutMemoryFallback 验证无效注入依赖在使用时报错，不静默回退到内存。
func TestInjectedNilRedisFailsWithoutMemoryFallback(t *testing.T) {
	for _, server := range []*sso.Server{NewServerFromClient(nil), NewServerFromStorage(nil)} {
		if err := server.RegisterClient(&sso.Client{ClientID: "app"}); !errors.Is(err, sso.ErrStorageUnavailable) {
			t.Fatalf("RegisterClient() error = %v, want ErrStorageUnavailable", err)
		}
		if err := server.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}
}
