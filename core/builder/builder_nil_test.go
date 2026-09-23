package builder

import (
	"strings"
	"testing"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
)

// TestBuildRejectsTypedNilInjection verifies invalid explicit adapters never reach Manager or cleanup. TestBuildRejectsTypedNilInjection 验证显式注入的空指针不会进入 Manager 或资源清理。
func TestBuildRejectsTypedNilInjection(t *testing.T) {
	tests := []struct {
		name   string
		inject func(*Builder)
		want   string
	}{
		{name: "generator", inject: func(b *Builder) { b.SetGenerator((*testGenerator)(nil)) }, want: "token generator"},
		{name: "storage", inject: func(b *Builder) { b.SetStorage((*builderClosableStorage)(nil)) }, want: "storage adapter"},
		{name: "codec", inject: func(b *Builder) { b.SetCodec((*testCodec)(nil)) }, want: "codec adapter"},
		{name: "logger", inject: func(b *Builder) { b.SetLog((*builderClosableLogger)(nil)) }, want: "log adapter"},
		{name: "pool", inject: func(b *Builder) { b.SetPool((*testPool)(nil)) }, want: "task pool"},
		{name: "provider", inject: func(b *Builder) { b.SetAccessProvider((*testAccessProvider)(nil)) }, want: "access provider"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &builderClosableStorage{}
			logger := &builderClosableLogger{}
			b := NewBuilder().IsPrintBanner(false).IsLog(true).
				SetGenerator(&testGenerator{}).SetStorage(storage).SetCodec(&testCodec{}).SetLog(logger)
			tt.inject(b)

			mgr, err := b.Build()
			if mgr != nil || err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Build() = (%v, %v), want nil Manager and %q error", mgr, err, tt.want)
			}
			if storage.closeCalls != 0 || logger.flushCalls != 0 || logger.closeCalls != 0 {
				t.Fatal("failed Build closed explicitly injected resources")
			}
		})
	}
}

// TestBuildRejectsTypedNilFactoryResults verifies failed assembly releases earlier factory resources safely. TestBuildRejectsTypedNilFactoryResults 验证空指针工厂产物会使装配失败并安全释放此前创建的资源。
func TestBuildRejectsTypedNilFactoryResults(t *testing.T) {
	for _, stage := range []string{"generator", "storage", "codec", "logger", "pool"} {
		t.Run(stage, func(t *testing.T) {
			storage := &builderClosableStorage{}
			logger := &builderClosableLogger{}
			b := NewBuilder().IsPrintBanner(false).IsLog(true).
				SetGeneratorFactory(func(*config.Config) (adapter.Generator, error) {
					if stage == "generator" {
						return (*testGenerator)(nil), nil
					}
					return &testGenerator{}, nil
				}).
				SetStorageFactory(func(*config.Config) (adapter.Storage, error) {
					if stage == "storage" {
						return (*builderClosableStorage)(nil), nil
					}
					return storage, nil
				}).
				SetCodecFactory(func(*config.Config) (adapter.Codec, error) {
					if stage == "codec" {
						return (*testCodec)(nil), nil
					}
					return &testCodec{}, nil
				}).
				SetLogFactory(func(*config.Config) (adapter.Log, error) {
					if stage == "logger" {
						return (*builderClosableLogger)(nil), nil
					}
					return logger, nil
				}).
				SetPoolFactory(func(*config.Config) (adapter.Pool, error) {
					return (*testPool)(nil), nil
				})

			mgr, err := b.Build()
			wantError := map[string]string{
				"generator": "token generator", "storage": "storage adapter", "codec": "codec adapter",
				"logger": "log adapter", "pool": "task pool",
			}[stage]
			if mgr != nil || err == nil || !strings.Contains(err.Error(), wantError) {
				t.Fatalf("Build() = (%v, %v), want nil Manager and %q error", mgr, err, wantError)
			}
			wantStorageClose, wantLoggerClose := 0, 0
			if stage == "codec" || stage == "logger" || stage == "pool" {
				wantStorageClose = 1
			}
			if stage == "pool" {
				wantLoggerClose = 1
			}
			if storage.closeCalls != wantStorageClose || logger.closeCalls != wantLoggerClose || logger.flushCalls != wantLoggerClose {
				t.Fatalf("cleanup calls = storage %d, logger close %d flush %d; want %d, %d, %d",
					storage.closeCalls, logger.closeCalls, logger.flushCalls, wantStorageClose, wantLoggerClose, wantLoggerClose)
			}
		})
	}
}

// TestBuildPreservesNilAndValueComponentSemantics verifies optional nils, ignored logs, and value adapters remain supported. TestBuildPreservesNilAndValueComponentSemantics 验证可选 nil、禁用日志及值类型适配器仍然兼容。
func TestBuildPreservesNilAndValueComponentSemantics(t *testing.T) {
	poolFactoryCalled := false
	generator := builderValueGenerator{testGenerator: &testGenerator{}}
	mgr, err := NewBuilder().IsPrintBanner(false).IsLog(false).
		SetGenerator(nil).
		SetGeneratorFactory(func(*config.Config) (adapter.Generator, error) { return generator, nil }).
		SetStorage(&testStorage{}).SetCodec(&testCodec{}).
		SetLog((*builderClosableLogger)(nil)).SetAccessProvider(nil).SetPool(nil).
		SetPoolFactory(func(*config.Config) (adapter.Pool, error) {
			poolFactoryCalled = true
			return nil, nil
		}).Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	defer mgr.CloseManager()
	if !poolFactoryCalled || mgr.GetPool() != nil || mgr.GetAccessProvider() != nil {
		t.Fatal("optional nil component semantics changed")
	}
}

// builderValueGenerator exercises a non-pointer adapter implementation. builderValueGenerator 用于验证非指针适配器实现。
type builderValueGenerator struct {
	*testGenerator
}
