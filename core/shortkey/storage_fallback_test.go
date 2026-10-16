package shortkey

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/derror"
)

// TestCreateFallbackPreservesReadErrors verifies failed collision checks cannot write a credential. TestCreateFallbackPreservesReadErrors 验证碰撞检查失败时不会写入凭证。
func TestCreateFallbackPreservesReadErrors(t *testing.T) {
	storage := &shortKeyReadFailStorage{Storage: newShortKeyTestStorage()}
	mgr := NewDefaultManager("test", "dt:", storage, shortKeyTestCodec{})
	created, err := mgr.Create(context.Background(), CreateOptions{})
	if !errors.Is(err, derror.ErrStorageUnavailable) || created != nil || storage.writes != 0 {
		t.Fatalf("Create() = %v, %v, writes = %d; want storage error and no write", created, err, storage.writes)
	}
}

// TestCreateFallbackPreservesExistingKey verifies ordinary storage distinguishes collisions from absence. TestCreateFallbackPreservesExistingKey 验证普通存储区分碰撞与键不存在。
func TestCreateFallbackPreservesExistingKey(t *testing.T) {
	ctx := context.Background()
	storage := struct{ adapter.Storage }{newShortKeyTestStorage()}
	mgr := NewDefaultManager("test", "dt:", storage, shortKeyTestCodec{})
	first := &ShortKey{Key: "same", AuthType: "test", LoginID: "original", Status: StatusConfirmed}
	if saved, err := mgr.saveIfAbsent(ctx, first, time.Minute); err != nil || !saved {
		t.Fatalf("first save = %v, %v", saved, err)
	}
	replacement := *first
	replacement.LoginID = "replacement"
	if saved, err := mgr.saveIfAbsent(ctx, &replacement, time.Minute); err != nil || saved {
		t.Fatalf("collision save = %v, %v", saved, err)
	}
	got, err := mgr.get(ctx, first.Key)
	if err != nil || got.LoginID != "original" {
		t.Fatalf("stored credential = %v, %v", got, err)
	}
}

type shortKeyReadFailStorage struct {
	adapter.Storage
	writes int
}

func (s *shortKeyReadFailStorage) Get(context.Context, string) (any, error) {
	return nil, errors.New("read unavailable")
}

func (s *shortKeyReadFailStorage) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	s.writes++
	return s.Storage.Set(ctx, key, value, ttl)
}
