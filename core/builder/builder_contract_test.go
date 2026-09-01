// @Author daixk 2026/08/31
package builder

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/manager"
)

// TestBuilderConfigCopiesInput verifies Config does not retain caller-owned mutable state. TestBuilderConfigCopiesInput 验证 Config 不会保留调用方持有的可变状态。
func TestBuilderConfigCopiesInput(t *testing.T) {
	source := config.DefaultConfig()
	source.AuthType = "source:"
	source.CookieConfig.Path = "/source"

	b := NewBuilder().Config(source)
	source.AuthType = "mutated:"
	source.CookieConfig.Path = "/mutated"

	got := b.GetConfig()
	if got == source {
		t.Fatal("Config() retained the source config pointer")
	}
	if got.CookieConfig == source.CookieConfig {
		t.Fatal("Config() retained the source CookieConfig pointer")
	}
	if got.AuthType != "source:" || got.CookieConfig.Path != "/source" {
		t.Fatalf("builder config changed after source mutation: %+v", got)
	}
}

// TestBuilderConfigNilRestoresDefaults verifies nil Config resets only the mutable config. TestBuilderConfigNilRestoresDefaults 验证 Config(nil) 会重置可变配置。
func TestBuilderConfigNilRestoresDefaults(t *testing.T) {
	b := NewBuilder().AuthType("custom").KeyPrefix("app").Config(nil)
	got := b.GetConfig()

	if got.AuthType != config.DefaultAuthType || got.KeyPrefix != config.DefaultKeyPrefix {
		t.Fatalf("Config(nil) values = %q %q, want defaults", got.AuthType, got.KeyPrefix)
	}
	if got.CookieConfig == nil || got.CookieConfig.Path != config.DefaultCookiePath {
		t.Fatalf("Config(nil) CookieConfig = %+v, want defaults", got.CookieConfig)
	}
}

// TestBuilderCloneCopiesMutableState verifies cloned builders can diverge without changing the original. TestBuilderCloneCopiesMutableState 验证克隆 Builder 可以独立变化。
func TestBuilderCloneCopiesMutableState(t *testing.T) {
	original := NewBuilder().AuthType("original").CookiePath("/original")
	original.UseManagerOption(func(_ *manager.Manager) {})
	clone := original.Clone()

	clone.AuthType("clone").CookiePath("/clone")
	clone.UseManagerOption(func(_ *manager.Manager) {})

	if original.GetConfig().AuthType != "original:" || original.GetConfig().CookieConfig.Path != "/original" {
		t.Fatalf("original config changed after clone mutation: %+v", original.GetConfig())
	}
	if clone.GetConfig().AuthType != "clone:" || clone.GetConfig().CookieConfig.Path != "/clone" {
		t.Fatalf("clone config = %+v, want clone values", clone.GetConfig())
	}
	if len(original.managerOptions) != 1 || len(clone.managerOptions) != 2 {
		t.Fatalf("manager option lengths = %d %d, want 1 2", len(original.managerOptions), len(clone.managerOptions))
	}
}

// TestBuilderCookieConfigCopiesInput verifies CookieConfig copies the caller value. TestBuilderCookieConfigCopiesInput 验证 CookieConfig 会复制调用方传入的值。
func TestBuilderCookieConfigCopiesInput(t *testing.T) {
	source := &config.CookieConfig{Domain: "example.com", Path: "/source", SameSite: config.SameSiteLax}
	b := NewBuilder().CookieConfig(source)
	source.Path = "/mutated"
	source.Domain = "mutated.example.com"

	got := b.GetConfig().CookieConfig
	if got == source {
		t.Fatal("CookieConfig() retained the source pointer")
	}
	if got.Path != "/source" || got.Domain != "example.com" {
		t.Fatalf("builder cookie config changed after source mutation: %+v", got)
	}
}

// TestBuilderCookieSettersInitializeConfig verifies cookie setters restore a missing cookie config. TestBuilderCookieSettersInitializeConfig 验证 Cookie Setter 会恢复缺失的 Cookie 配置。
func TestBuilderCookieSettersInitializeConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.CookieConfig = nil
	b := NewBuilder().Config(cfg)
	b.CookieDomain("example.com").CookiePath("/admin").CookieSecure(true).CookieHttpOnly(false).CookieSameSite(config.SameSiteNone).CookieMaxAge(3600)

	got := b.GetConfig().CookieConfig
	if got == nil {
		t.Fatal("cookie setters left CookieConfig nil")
	}
	if got.Domain != "example.com" || got.Path != "/admin" || !got.Secure || got.HttpOnly || got.SameSite != config.SameSiteNone || got.MaxAge != 3600 {
		t.Fatalf("cookie setter values = %+v", got)
	}
}

// TestDurationToSecondsBoundaries verifies exact and fractional positive durations. TestDurationToSecondsBoundaries 验证整秒和亚秒正时长转换边界。
func TestDurationToSecondsBoundaries(t *testing.T) {
	tests := []struct {
		name string
		in   time.Duration
		want int64
	}{
		{name: "zero", in: 0, want: 0},
		{name: "exact second", in: time.Second, want: 1},
		{name: "fractional second", in: time.Second + time.Nanosecond, want: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := durationToSeconds(tt.in); got != tt.want {
				t.Fatalf("durationToSeconds(%s) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

// TestNegativeDurationCannotBecomeNoLimit verifies negative durations never become the -1 sentinel. TestNegativeDurationCannotBecomeNoLimit 验证负时长不会误变成 -1 无限值。
func TestNegativeDurationCannotBecomeNoLimit(t *testing.T) {
	for _, duration := range []time.Duration{-time.Second, -1500 * time.Millisecond} {
		t.Run(duration.String(), func(t *testing.T) {
			_, err := NewBuilder().
				IsPrintBanner(false).
				AutoRenew(false).
				RefreshTokenTimeoutDuration(duration).
				Build()
			if err == nil {
				t.Fatalf("Build() accepted negative duration %s as unlimited", duration)
			}
			if !strings.Contains(err.Error(), "RefreshTokenTimeout") {
				t.Fatalf("Build() error = %q, want RefreshTokenTimeout validation", err)
			}
		})
	}
}

// TestBuildRejectsMissingComponentsByStage verifies required components fail at the correct stage. TestBuildRejectsMissingComponentsByStage 验证缺失组件会在对应阶段失败。
func TestBuildRejectsMissingComponentsByStage(t *testing.T) {
	tests := []struct {
		name  string
		build func() (*manager.Manager, error)
		want  string
	}{
		{
			name: "generator",
			build: func() (*manager.Manager, error) {
				return NewBuilder().IsPrintBanner(false).Build()
			},
			want: "token generator is missing",
		},
		{
			name: "storage",
			build: func() (*manager.Manager, error) {
				return NewBuilder().IsPrintBanner(false).SetGenerator(&testGenerator{}).Build()
			},
			want: "storage adapter is missing",
		},
		{
			name: "codec",
			build: func() (*manager.Manager, error) {
				return NewBuilder().IsPrintBanner(false).SetGenerator(&testGenerator{}).SetStorage(&testStorage{}).Build()
			},
			want: "codec adapter is missing",
		},
		{
			name: "logger",
			build: func() (*manager.Manager, error) {
				return NewBuilder().IsPrintBanner(false).IsLog(true).SetGenerator(&testGenerator{}).SetStorage(&testStorage{}).SetCodec(&testCodec{}).Build()
			},
			want: "log adapter is missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr, err := tt.build()
			if err == nil {
				t.Fatal("Build() error = nil, want missing component error")
			}
			if mgr != nil {
				t.Fatal("Build() returned manager on missing component")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Build() error = %q, want %q", err, tt.want)
			}
		})
	}
}

// TestBuildPoolFactoryFollowsAutoRenew verifies pool factories are only used when auto-renew is enabled. TestBuildPoolFactoryFollowsAutoRenew 验证协程池工厂只在启用自动续期时调用。
func TestBuildPoolFactoryFollowsAutoRenew(t *testing.T) {
	pool := &testPool{}
	calls := 0
	b := NewBuilder().
		IsPrintBanner(false).
		SetGenerator(&testGenerator{}).
		SetStorage(&testStorage{}).
		SetCodec(&testCodec{}).
		SetPoolFactory(func(_ *config.Config) (adapter.Pool, error) {
			calls++
			return pool, nil
		})

	b.AutoRenew(false)
	withoutRenew, err := b.Build()
	if err != nil {
		t.Fatalf("Build() with AutoRenew(false) error = %v", err)
	}
	if withoutRenew.GetPool() != nil || calls != 0 {
		t.Fatalf("pool = %v, factory calls = %d with AutoRenew(false), want nil and 0", withoutRenew.GetPool(), calls)
	}
	withoutRenew.CloseManager()

	b.AutoRenew(true)
	withRenew, err := b.Build()
	if err != nil {
		t.Fatalf("Build() with AutoRenew(true) error = %v", err)
	}
	defer withRenew.CloseManager()
	if withRenew.GetPool() != pool || calls != 1 {
		t.Fatalf("pool = %v, factory calls = %d with AutoRenew(true), want injected pool and 1", withRenew.GetPool(), calls)
	}
}

// TestBuildClosesFactoryStorageOnLaterFailure verifies factory-created storage is released when assembly fails later. TestBuildClosesFactoryStorageOnLaterFailure 验证后续装配失败时会释放工厂创建的存储。
func TestBuildClosesFactoryStorageOnLaterFailure(t *testing.T) {
	storage := &builderClosableStorage{}
	sentinel := errors.New("codec factory failed")

	mgr, err := NewBuilder().
		IsPrintBanner(false).
		SetGenerator(&testGenerator{}).
		SetStorageFactory(func(_ *config.Config) (adapter.Storage, error) {
			return storage, nil
		}).
		SetCodecFactory(func(_ *config.Config) (adapter.Codec, error) {
			return nil, sentinel
		}).
		Build()

	if mgr != nil {
		t.Fatal("Build() returned manager after codec factory failure")
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("Build() error = %v, want wrapped codec factory error", err)
	}
	if storage.closeCalls != 1 {
		t.Fatalf("factory storage close calls = %d, want 1", storage.closeCalls)
	}
}

// TestBuildClosesFactoryResourcesOnPoolFailure verifies storage and logger are released when pool assembly fails. TestBuildClosesFactoryResourcesOnPoolFailure 验证协程池装配失败时会释放存储和日志资源。
func TestBuildClosesFactoryResourcesOnPoolFailure(t *testing.T) {
	storage := &builderClosableStorage{}
	logger := &builderClosableLogger{}
	sentinel := errors.New("pool factory failed")

	mgr, err := NewBuilder().
		IsPrintBanner(false).
		IsLog(true).
		SetGenerator(&testGenerator{}).
		SetStorageFactory(func(_ *config.Config) (adapter.Storage, error) {
			return storage, nil
		}).
		SetCodec(&testCodec{}).
		SetLogFactory(func(_ *config.Config) (adapter.Log, error) {
			return logger, nil
		}).
		SetPoolFactory(func(_ *config.Config) (adapter.Pool, error) {
			return nil, sentinel
		}).
		Build()

	if mgr != nil {
		t.Fatal("Build() returned manager after pool factory failure")
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("Build() error = %v, want wrapped pool factory error", err)
	}
	if storage.closeCalls != 1 {
		t.Fatalf("factory storage close calls = %d, want 1", storage.closeCalls)
	}
	if logger.flushCalls != 1 || logger.closeCalls != 1 {
		t.Fatalf("factory logger lifecycle = flush %d close %d, want 1 1", logger.flushCalls, logger.closeCalls)
	}
}

type builderClosableStorage struct {
	testStorage
	closeCalls int
}

func (s *builderClosableStorage) Close() error {
	s.closeCalls++
	return nil
}

type builderClosableLogger struct {
	testLogger
	flushCalls int
	closeCalls int
}

func (l *builderClosableLogger) Close() {
	l.closeCalls++
}

func (l *builderClosableLogger) Flush() {
	l.flushCalls++
}

func (l *builderClosableLogger) SetLevel(adapter.LogLevel) {}

func (l *builderClosableLogger) SetPrefix(string) {}

func (l *builderClosableLogger) SetStdout(bool) {}

func (l *builderClosableLogger) LogPath() string { return "" }

func (l *builderClosableLogger) DropCount() uint64 { return 0 }
