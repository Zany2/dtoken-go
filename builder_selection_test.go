package dtoken

import (
	"testing"

	"github.com/Zany2/dtoken-go/config"
	adapter "github.com/Zany2/dtoken-go/contract"
)

// TestBuilderNilFactoriesRestoreDefaults verifies clearing overrides restores fresh default components. TestBuilderNilFactoriesRestoreDefaults 验证清除工厂覆盖后恢复独立的默认组件。
func TestBuilderNilFactoriesRestoreDefaults(t *testing.T) {
	b := NewBuilder().IsPrintBanner(false).IsLog(true).LoggerStdoutOnly(true).
		SetGeneratorFactory(func(*config.Config) (adapter.Generator, error) {
			t.Fatal("cleared generator factory called")
			return nil, nil
		}).
		SetStorageFactory(func(*config.Config) (adapter.Storage, error) {
			t.Fatal("cleared storage factory called")
			return nil, nil
		}).
		SetCodecFactory(func(*config.Config) (adapter.Codec, error) {
			t.Fatal("cleared codec factory called")
			return nil, nil
		}).
		SetLogFactory(func(*config.Config) (adapter.Log, error) {
			t.Fatal("cleared log factory called")
			return nil, nil
		}).
		SetPoolFactory(func(*config.Config) (adapter.Pool, error) {
			t.Fatal("cleared pool factory called")
			return nil, nil
		}).
		SetGeneratorFactory(nil).SetStorageFactory(nil).SetCodecFactory(nil).SetLogFactory(nil).SetPoolFactory(nil)
	first, err := b.BuildInstance()
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	second, err := b.BuildInstance()
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	if first.Manager().GetPool() == nil || first.Manager().GetStorage() == second.Manager().GetStorage() || first.Manager().GetPool() == second.Manager().GetPool() {
		t.Fatal("repeated builds did not create independent default components")
	}
}

// TestBuilderIgnoresUnusedBundledConfigs verifies custom or disabled components ignore bundled implementation settings. TestBuilderIgnoresUnusedBundledConfigs 验证自定义或禁用组件不受内置实现配置影响。
func TestBuilderIgnoresUnusedBundledConfigs(t *testing.T) {
	for _, mode := range []string{"disabled", "instance", "factory"} {
		t.Run(mode, func(t *testing.T) {
			b := NewBuilder().IsPrintBanner(false).LoggerMaxSize(-1).RenewPoolMinSize(0)
			logger, pool := &testLogger{}, &testPool{}
			switch mode {
			case "disabled":
				b.IsLog(false).AutoRenew(false)
			case "instance":
				b.IsLog(true).SetLog(logger).SetPool(pool)
			case "factory":
				b.IsLog(true).
					SetLogFactory(func(*config.Config) (adapter.Log, error) { return logger, nil }).
					SetPoolFactory(func(*config.Config) (adapter.Pool, error) { return pool, nil })
			}
			auth, err := b.BuildInstance()
			if err != nil {
				t.Fatal(err)
			}
			defer auth.Close()
			if mode != "disabled" && (auth.Manager().GetLogger() != logger || auth.Manager().GetPool() != pool) {
				t.Fatal("custom components were replaced by defaults")
			}
		})
	}
}

// TestBuilderExplicitComponentsPrecedeFactories verifies factories are not invoked for injected adapters. TestBuilderExplicitComponentsPrecedeFactories 验证显式适配器存在时不会调用工厂。
func TestBuilderExplicitComponentsPrecedeFactories(t *testing.T) {
	generator, storage, codec := &testGenerator{}, &testStorage{}, &testCodec{}
	logger, pool := &testLogger{}, &testPool{}
	auth, err := NewBuilder().IsPrintBanner(false).IsLog(true).
		SetGenerator(generator).Storage(storage).SetCodec(codec).SetLog(logger).SetPool(pool).
		SetGeneratorFactory(func(*config.Config) (adapter.Generator, error) {
			t.Fatal("generator factory called")
			return nil, nil
		}).
		SetStorageFactory(func(*config.Config) (adapter.Storage, error) {
			t.Fatal("storage factory called")
			return nil, nil
		}).
		SetCodecFactory(func(*config.Config) (adapter.Codec, error) {
			t.Fatal("codec factory called")
			return nil, nil
		}).
		SetLogFactory(func(*config.Config) (adapter.Log, error) {
			t.Fatal("log factory called")
			return nil, nil
		}).
		SetPoolFactory(func(*config.Config) (adapter.Pool, error) {
			t.Fatal("pool factory called")
			return nil, nil
		}).BuildInstance()
	if err != nil {
		t.Fatal(err)
	}
	defer auth.Close()
	mgr := auth.Manager()
	if mgr.GetGenerator() != generator || mgr.GetStorage() != storage || mgr.GetSerializer() != codec || mgr.GetLogger() != logger || mgr.GetPool() != pool {
		t.Fatal("explicit component identity changed")
	}
}

// TestBuilderRejectsInvalidDefaultPool verifies the bundled pool still validates its own configuration. TestBuilderRejectsInvalidDefaultPool 验证默认池仍校验自身配置。
func TestBuilderRejectsInvalidDefaultPool(t *testing.T) {
	auth, err := NewBuilder().IsPrintBanner(false).RenewPoolMinSize(0).BuildInstance()
	if auth != nil {
		auth.Close()
	}
	if auth != nil || err == nil {
		t.Fatalf("BuildInstance() = (%v, %v), want invalid pool error", auth, err)
	}
}
