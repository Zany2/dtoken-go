package dtoken

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/Zany2/dtoken-go/config"
	"github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
	"github.com/Zany2/dtoken-go/storage/memory"
)

// registryBuilder avoids background work in registry lifecycle checks. registryBuilder 在注册表生命周期检查中禁用后台任务。
func registryBuilder() *Builder {
	return NewBuilder().IsPrintBanner(false).IsLog(false).AutoRenew(false).AsyncEvent(false)
}

// TestBuildRegistersBothFacadeStyles verifies default routing, namespace isolation and direct instance identity. TestBuildRegistersBothFacadeStyles 验证默认路由、体系隔离与直接实例身份。
func TestBuildRegistersBothFacadeStyles(t *testing.T) {
	CloseAll()
	t.Cleanup(CloseAll)
	ctx := context.Background()
	defaultAuth := registryBuilder().MustBuild()
	userAuth := registryBuilder().AuthType("user").MustBuild()

	if AuthType("").Manager() != defaultAuth.Manager() || AuthType(" user: ").Manager() != userAuth.Manager() {
		t.Fatal("selectors do not resolve the built instances")
	}
	defaultToken, err := Login(ctx, LoginOptions{LoginID: "default-user", AuthType: "user"})
	if err != nil {
		t.Fatal(err)
	}
	userToken, err := AuthType("user").LoginID(ctx, "user")
	if err != nil {
		t.Fatal(err)
	}
	if !defaultAuth.IsLogin(ctx, defaultToken) || !IsLogin(ctx, defaultToken) || !userAuth.IsLogin(ctx, userToken) {
		t.Fatal("direct and registry facades do not share credentials")
	}
	if IsLogin(ctx, userToken) || userAuth.IsLogin(ctx, defaultToken) || AuthType("missing").IsLogin(ctx, defaultToken) {
		t.Fatal("token crossed authentication namespaces")
	}
	if err := AuthType("missing").CheckLogin(ctx, defaultToken); !errors.Is(err, derror.ErrManagerNotFound) {
		t.Fatalf("missing namespace: %v", err)
	}

	Close()
	Close()
	if !defaultAuth.IsClosed() || userAuth.IsClosed() {
		t.Fatal("default Close affected the wrong instance")
	}
	CloseAll()
	CloseAll()
	if !userAuth.IsClosed() {
		t.Fatal("CloseAll left a registered instance open")
	}
}

// TestBuildDuplicatePreservesExistingInstance verifies duplicate names fail before component creation. TestBuildDuplicatePreservesExistingInstance 验证重复名称在组件创建前失败并保留已有实例。
func TestBuildDuplicatePreservesExistingInstance(t *testing.T) {
	CloseAll()
	t.Cleanup(CloseAll)
	first := registryBuilder().AuthType("duplicate").MustBuild()
	var factories atomic.Int32
	b := registryBuilder().AuthType(" duplicate: ").SetStorageFactory(func(*config.Config) (contract.Storage, error) {
		factories.Add(1)
		return memory.NewStorage(), nil
	})
	if auth, err := b.Build(); auth != nil || !errors.Is(err, derror.ErrManagerAlreadyRegistered) {
		t.Fatalf("duplicate Build = %v, %v", auth, err)
	}
	if factories.Load() != 0 || first.IsClosed() || AuthType("duplicate").Manager() != first.Manager() {
		t.Fatal("duplicate build allocated resources or replaced the original")
	}
	defer func() {
		err, ok := recover().(error)
		if !ok || !errors.Is(err, derror.ErrManagerAlreadyRegistered) {
			t.Fatalf("MustBuild panic = %v", err)
		}
	}()
	b.MustBuild()
}

// TestFailedBuildAllowsRetry verifies failed assembly leaves no registration and a corrected builder can be reused. TestFailedBuildAllowsRetry 验证装配失败不遗留注册项，修正后的构建器可以重试。
func TestFailedBuildAllowsRetry(t *testing.T) {
	CloseAll()
	t.Cleanup(CloseAll)
	buildErr := errors.New("storage factory failed")
	b := registryBuilder().AuthType("retry").SetStorageFactory(func(*config.Config) (contract.Storage, error) {
		return nil, buildErr
	})
	if auth, err := b.Build(); auth != nil || !errors.Is(err, buildErr) {
		t.Fatalf("failed Build = %v, %v", auth, err)
	}
	if _, err := GetManager("retry"); !errors.Is(err, derror.ErrManagerNotFound) {
		t.Fatalf("failed build left a registration: %v", err)
	}
	b.SetStorageFactory(func(*config.Config) (contract.Storage, error) {
		return memory.NewStorage(), nil
	})
	auth, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	if AuthType("retry").Manager() != auth.Manager() {
		t.Fatal("corrected builder did not register its engine")
	}
}

// TestSelectorFollowsReplacement verifies cached selectors follow replacement and old wrappers cannot remove the new engine. TestSelectorFollowsReplacement 验证缓存选择器跟随替换，旧门面不能移除新引擎。
func TestSelectorFollowsReplacement(t *testing.T) {
	CloseAll()
	t.Cleanup(CloseAll)
	selected := AuthType("replace")
	first := registryBuilder().AuthType("replace").MustBuild()
	old := New(first.Manager())
	second := registryBuilder().AuthType("replace").MustBuildInstance()
	t.Cleanup(second.Close)
	SetAuth(second)
	if !first.IsClosed() || selected.Manager() != second.Manager() {
		t.Fatal("selector did not follow explicit replacement")
	}
	first.Close()
	old.Close()
	if selected.Manager() != second.Manager() || second.IsClosed() {
		t.Fatal("closing an old wrapper removed its replacement")
	}
	selected.Close()
	selected.Close()
	if !second.IsClosed() {
		t.Fatal("selected Close did not close the current instance")
	}
	third := registryBuilder().AuthType("replace").MustBuild()
	if selected.Manager() != third.Manager() {
		t.Fatal("selector failed to resolve a newly registered instance")
	}
}

// TestBuildInstanceAndImmediateUnregister verifies isolated builds and identity-based eager removal. TestBuildInstanceAndImmediateUnregister 验证独立构建及按实例身份立即注销。
func TestBuildInstanceAndImmediateUnregister(t *testing.T) {
	CloseAll()
	t.Cleanup(CloseAll)
	registered := registryBuilder().MustBuild()
	isolated := registryBuilder().MustBuildInstance()
	t.Cleanup(isolated.Close)
	if AuthType("").Manager() != registered.Manager() {
		t.Fatal("isolated build changed the registry")
	}
	CloseAll()
	if isolated.IsClosed() {
		t.Fatal("CloseAll closed an unregistered instance")
	}
	SetAuth(isolated)
	isolated.Close()
	if _, exists := globalManagerMap.Load(config.DefaultAuthType); exists {
		t.Fatal("Auth.Close left an entry until the next lookup")
	}
	SetAuth(isolated)
	if _, err := GetManager(); !errors.Is(err, derror.ErrManagerNotFound) {
		t.Fatalf("closed Auth was registered: %v", err)
	}
}

// TestRegistryInvalidValue preserves error reporting after moving registry ownership. TestRegistryInvalidValue 在迁移注册表归属后保留损坏值错误报告。
func TestRegistryInvalidValue(t *testing.T) {
	CloseAll()
	t.Cleanup(CloseAll)
	globalManagerMap.Store(config.DefaultAuthType, struct{}{})
	if err := AuthType("").CheckLogin(context.Background(), "token"); !errors.Is(err, derror.ErrManagerInvalidType) {
		t.Fatalf("selector error = %v", err)
	}
	if _, err := registryBuilder().Build(); !errors.Is(err, derror.ErrManagerInvalidType) {
		t.Fatalf("Build error = %v", err)
	}
	if err := DeleteManager(); !errors.Is(err, derror.ErrManagerInvalidType) {
		t.Fatalf("DeleteManager error = %v", err)
	}
}

// registryOwnedStorage tracks cleanup of fully assembled candidates. registryOwnedStorage 跟踪已完成装配的候选实例清理。
type registryOwnedStorage struct {
	*memory.Storage
	closes atomic.Int32
}

func (s *registryOwnedStorage) Close() error {
	s.closes.Add(1)
	return nil
}

// TestConcurrentBuildPublishesOnce verifies the loser releases its resources without closing the winner. TestConcurrentBuildPublishesOnce 验证并发失败方释放自有资源且不关闭获胜者。
func TestConcurrentBuildPublishesOnce(t *testing.T) {
	CloseAll()
	t.Cleanup(CloseAll)
	ready := make(chan struct{}, 2)
	proceed := make(chan struct{})
	type result struct {
		auth    *Auth
		err     error
		storage *registryOwnedStorage
	}
	results := make(chan result, 2)
	for range 2 {
		go func() {
			storage := &registryOwnedStorage{Storage: memory.NewStorage()}
			auth, err := registryBuilder().SetStorageFactory(func(*config.Config) (contract.Storage, error) {
				ready <- struct{}{}
				<-proceed
				return storage, nil
			}).Build()
			results <- result{auth, err, storage}
		}()
	}
	<-ready
	<-ready
	close(proceed)
	winners := 0
	for range 2 {
		result := <-results
		if result.err == nil {
			winners++
			if result.auth == nil || result.auth.IsClosed() || result.storage.closes.Load() != 0 {
				t.Error("winning instance was closed")
			}
		} else if !errors.Is(result.err, derror.ErrManagerAlreadyRegistered) || result.auth != nil || result.storage.closes.Load() != 1 {
			t.Errorf("losing build = %v, %v, closes=%d", result.auth, result.err, result.storage.closes.Load())
		}
	}
	if winners != 1 {
		t.Fatalf("successful builds = %d, want 1", winners)
	}
}
