package manager

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/redis/go-redis/v9"
)

const (
	managerRedisTestAddr     = "192.168.19.104:6379"
	managerRedisTestPassword = "root"
	managerRedisTestDatabase = 0
)

var managerRedisGetAndDeleteManyScript = redis.NewScript(`
local value = redis.call("GET", KEYS[1])
if not value then
	return false
end
redis.call("DEL", unpack(KEYS))
return value
`)

var _ adapter.FullStorage = (*managerRedisTestStorage)(nil)

type managerRedisTestStorage struct {
	client *redis.Client
	prefix string
}

func newManagerRedisTestStorage(t interface {
	Helper()
	Fatalf(string, ...any)
	Cleanup(func())
}, cfg *config.Config) adapter.FullStorage {
	t.Helper()
	client := redis.NewClient(&redis.Options{
		Addr:     managerRedisTestAddr,
		Password: managerRedisTestPassword,
		DB:       managerRedisTestDatabase,
	})
	storage := &managerRedisTestStorage{client: client, prefix: cfg.KeyPrefix}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := storage.Ping(ctx); err != nil {
		_ = client.Close()
		t.Fatalf("connect manager redis test storage error = %v", err)
	}
	_ = storage.Clear(ctx)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cleanupCancel()
		_ = storage.Clear(cleanupCtx)
		_ = client.Close()
	})
	return storage
}

func (s *managerRedisTestStorage) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	return s.client.Set(ctx, key, value, normalizeManagerRedisTestExpiration(expiration)).Err()
}

func (s *managerRedisTestStorage) Get(ctx context.Context, key string) (any, error) {
	value, err := s.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (s *managerRedisTestStorage) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return s.client.Del(ctx, keys...).Err()
}

func (s *managerRedisTestStorage) Exists(ctx context.Context, key string) bool {
	count, err := s.client.Exists(ctx, key).Result()
	return err == nil && count > 0
}

func (s *managerRedisTestStorage) Expire(ctx context.Context, key string, expiration time.Duration) error {
	if expiration <= 0 {
		ok, err := s.client.Persist(ctx, key).Result()
		if err != nil {
			return err
		}
		if !ok && !s.Exists(ctx, key) {
			return derror.ErrKeyNotFound
		}
		return nil
	}
	ok, err := s.client.PExpire(ctx, key, expiration).Result()
	if err != nil {
		return err
	}
	if !ok {
		return derror.ErrKeyNotFound
	}
	return nil
}

func (s *managerRedisTestStorage) TTL(ctx context.Context, key string) (time.Duration, error) {
	ttl, err := s.client.PTTL(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	return normalizeManagerRedisTestTTL(ttl), nil
}

func (s *managerRedisTestStorage) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

func (s *managerRedisTestStorage) GetAndDelete(ctx context.Context, key string) (any, error) {
	value, err := s.client.GetDel(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (s *managerRedisTestStorage) GetAndDeleteMany(ctx context.Context, key string, deleteKeys ...string) (any, error) {
	keys := make([]string, 0, len(deleteKeys)+1)
	keys = append(keys, key)
	keys = append(keys, deleteKeys...)
	value, err := managerRedisGetAndDeleteManyScript.Run(ctx, s.client, keys).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, nil
	}
	if missing, ok := value.(bool); ok && !missing {
		return nil, nil
	}
	return value, nil
}

func (s *managerRedisTestStorage) SetIfAbsent(ctx context.Context, key string, value any, expiration time.Duration) (bool, error) {
	return s.client.SetNX(ctx, key, value, normalizeManagerRedisTestExpiration(expiration)).Result()
}

func (s *managerRedisTestStorage) Keys(ctx context.Context, pattern string) ([]string, error) {
	if pattern == "" {
		pattern = "*"
	}
	var (
		cursor uint64
		keys   []string
	)
	for {
		matched, next, err := s.client.Scan(ctx, cursor, pattern, 1000).Result()
		if err != nil {
			return nil, err
		}
		keys = append(keys, matched...)
		cursor = next
		if cursor == 0 {
			break
		}
	}
	sort.Strings(keys)
	return keys, nil
}

func (s *managerRedisTestStorage) Clear(ctx context.Context) error {
	keys, err := s.Keys(ctx, s.prefix+"*")
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	return s.Delete(ctx, keys...)
}

func normalizeManagerRedisTestExpiration(expiration time.Duration) time.Duration {
	if expiration <= 0 {
		return 0
	}
	return expiration
}

func normalizeManagerRedisTestTTL(ttl time.Duration) time.Duration {
	switch ttl {
	case -time.Second, -time.Millisecond, adapter.TTLNoExpire:
		return adapter.TTLNoExpire
	case -2 * time.Second, -2 * time.Millisecond, adapter.TTLNotFound:
		return adapter.TTLNotFound
	default:
		return ttl
	}
}
