package dtoken_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/Zany2/dtoken-go"
	derror "github.com/Zany2/dtoken-go/errors"
	"github.com/Zany2/dtoken-go/global"
	"github.com/Zany2/dtoken-go/storage/memory"
)

// lifecycleStorage observes the resource owner without replacing storage behavior. lifecycleStorage 在保留存储行为的同时观察资源所有权。
type lifecycleStorage struct {
	*memory.Storage
	closes atomic.Int32
}

func (s *lifecycleStorage) Close() error {
	s.closes.Add(1)
	return nil
}

// TestRegistryReplacementPreservesCallerOwnedStorage verifies the public ownership contract across two Auth instances. TestRegistryReplacementPreservesCallerOwnedStorage 验证两个 Auth 实例间的公开资源所有权契约。
func TestRegistryReplacementPreservesCallerOwnedStorage(t *testing.T) {
	global.DeleteAllManager()
	t.Cleanup(global.DeleteAllManager)
	shared := &lifecycleStorage{Storage: memory.NewStorage()}
	t.Cleanup(func() { _ = shared.Close() })
	build := func() *dtoken.Auth {
		t.Helper()
		auth, err := dtoken.NewBuilder().AuthType("shared-lifecycle").
			Storage(shared).IsPrintBanner(false).IsLog(false).AutoRenew(false).
			UseManagerOption(dtoken.WithComponentOwnership(dtoken.ComponentOwnership{Storage: false, Logger: true, Pool: true})).Build()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(auth.Close)
		return auth
	}
	first, second := build(), build()
	alias := dtoken.New(first.Manager())
	token, err := first.LoginID(context.Background(), "user")
	if err != nil {
		t.Fatal(err)
	}
	global.SetAuth(first)
	global.SetAuth(first)
	if first.IsClosed() {
		t.Fatal("duplicate registration closed the instance")
	}
	global.SetAuth(second)
	if !first.IsClosed() || !alias.IsClosed() || second.IsClosed() {
		t.Fatal("replacement did not preserve shared engine lifecycle")
	}
	if first.Config() != nil || first.Storage() != nil || first.Codec() != nil || first.EventManager() != nil {
		t.Fatal("closed Auth exposed borrowed components")
	}
	if _, err := alias.LoginID(context.Background(), "other"); !errors.Is(err, derror.ErrManagerNotFound) {
		t.Fatalf("closed alias login = %v", err)
	}
	if err := second.CheckLogin(context.Background(), token); err != nil {
		t.Fatalf("replacement lost caller-owned credentials: %v", err)
	}
	first.Close()
	alias.Close()
	if err := global.DeleteManager("shared-lifecycle"); err != nil {
		t.Fatal(err)
	}
	second.Close()
	if shared.closes.Load() != 0 {
		t.Fatal("Auth or registry closed caller-owned storage")
	}
}

// TestAuthOwnsStorageOnce verifies repeated closure and wrapper aliases release owned resources once. TestAuthOwnsStorageOnce 验证重复关闭与门面别名仅释放一次自有资源。
func TestAuthOwnsStorageOnce(t *testing.T) {
	storage := &lifecycleStorage{Storage: memory.NewStorage()}
	auth, err := dtoken.NewBuilder().Storage(storage).IsPrintBanner(false).IsLog(false).AutoRenew(false).Build()
	if err != nil {
		t.Fatal(err)
	}
	alias := dtoken.New(auth.Manager())
	auth.Close()
	alias.Close()
	auth.Close()
	if storage.closes.Load() != 1 {
		t.Fatalf("storage closes = %d, want 1", storage.closes.Load())
	}
	var missing *dtoken.Auth
	if !missing.IsClosed() || !(&dtoken.Auth{}).IsClosed() {
		t.Fatal("empty Auth should be unavailable")
	}
	global.SetAuth(missing)
	global.SetAuth(auth)
}
