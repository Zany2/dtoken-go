// @Author daixk 2025/12/22 15:56:00
package builder

import (
	"context"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/core/nonce"
	"github.com/Zany2/dtoken-go/core/oauth2"
	"github.com/Zany2/dtoken-go/core/shortkey"
	"github.com/Zany2/dtoken-go/core/ticket"
)

// TestBuildReturnsErrorForInvalidConfig verifies Build returns error instead of panic 测试 Build 在配置无效时返回错误而不是 panic
func TestBuildReturnsErrorForInvalidConfig(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Build should return error instead of panic: %v", r)
		}
	}()

	// Use invalid token name to trigger config validation 使用无效 Token 名称触发配置校验
	mgr, err := NewBuilder().TokenName("").Build()
	if err == nil {
		t.Fatal("Build should return config error")
	}
	if mgr != nil {
		t.Fatal("Build should not return manager when config is invalid")
	}
}

// TestBuildResolvesFactoryComponentsPerBuild verifies factory products follow the latest config TestBuildResolvesFactoryComponentsPerBuild 验证工厂组件会随最新配置重新装配
func TestBuildResolvesFactoryComponentsPerBuild(t *testing.T) {
	var generatorTimeouts []int64
	logFactoryCalls := 0

	b := NewBuilder().
		IsPrintBanner(false).
		SetGeneratorFactory(func(cfg *config.Config) (adapter.Generator, error) {
			generatorTimeouts = append(generatorTimeouts, cfg.Timeout)
			return &testGenerator{}, nil
		}).
		SetStorage(&testStorage{}).
		SetCodec(&testCodec{}).
		SetLogFactory(func(_ *config.Config) (adapter.Log, error) {
			logFactoryCalls++
			return &testLogger{}, nil
		})

	first, err := b.Timeout(10).RenewMaxRefresh(10).IsLog(false).Build()
	if err != nil {
		t.Fatalf("first Build() error = %v", err)
	}
	first.CloseManager()

	second, err := b.Timeout(20).RenewMaxRefresh(20).IsLog(true).Build()
	if err != nil {
		t.Fatalf("second Build() error = %v", err)
	}
	defer second.CloseManager()

	if len(generatorTimeouts) != 2 || generatorTimeouts[0] != 10 || generatorTimeouts[1] != 20 {
		t.Fatalf("generator timeouts = %v, want [10 20]", generatorTimeouts)
	}
	if logFactoryCalls != 1 {
		t.Fatalf("log factory calls = %d, want 1", logFactoryCalls)
	}
	if _, ok := second.GetLogger().(*testLogger); !ok {
		t.Fatalf("second logger type = %T, want *testLogger", second.GetLogger())
	}
}

// TestTimeoutDurationRoundsUp verifies sub-second durations remain valid TestTimeoutDurationRoundsUp 验证亚秒级时长会向上取整为有效秒数
func TestTimeoutDurationRoundsUp(t *testing.T) {
	cfg := NewBuilder().TimeoutDuration(1500 * time.Millisecond).GetConfig()
	if cfg.Timeout != 2 {
		t.Fatalf("Timeout = %d, want 2", cfg.Timeout)
	}
}

// TestBuildUsesInjectedOptionalModules verifies optional modules can be assembled externally TestBuildUsesInjectedOptionalModules 验证可选模块可以由外部装配后注入
func TestBuildUsesInjectedOptionalModules(t *testing.T) {
	storage := &testStorage{}
	codec := &testCodec{}
	nonceManager := nonce.NewNonceManager(config.DefaultAuthType, config.DefaultKeyPrefix, storage, time.Minute)
	oauth2Manager := oauth2.NewOAuth2Server(config.DefaultAuthType, config.DefaultKeyPrefix, storage, codec)
	ticketManager := ticket.NewDefaultManager(config.DefaultAuthType, config.DefaultKeyPrefix, storage, codec)
	shortKeyManager := shortkey.NewDefaultManager(config.DefaultAuthType, config.DefaultKeyPrefix, storage, codec)

	mgr, err := NewBuilder().
		IsPrintBanner(false).
		SetGenerator(&testGenerator{}).
		SetStorage(storage).
		SetCodec(codec).
		SetNonceManager(nonceManager).
		SetOAuth2Manager(oauth2Manager).
		SetTicketManager(ticketManager).
		SetShortKeyManager(shortKeyManager).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	defer mgr.CloseManager()

	if mgr.GetNonceManager() != nonceManager {
		t.Fatal("nonce manager was not injected")
	}
	if mgr.GetOAuth2Manager() != oauth2Manager {
		t.Fatal("oauth2 manager was not injected")
	}
	if mgr.GetTicketManager() != ticketManager {
		t.Fatal("ticket manager was not injected")
	}
	if mgr.GetShortKeyManager() != shortKeyManager {
		t.Fatal("short key manager was not injected")
	}
}

// TestBuildUsesInjectedAccessProvider verifies access provider injection TestBuildUsesInjectedAccessProvider 验证访问提供器注入
func TestBuildUsesInjectedAccessProvider(t *testing.T) {
	provider := &testAccessProvider{}

	mgr, err := NewBuilder().
		IsPrintBanner(false).
		SetGenerator(&testGenerator{}).
		SetStorage(&testStorage{}).
		SetCodec(&testCodec{}).
		SetAccessProvider(provider).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	defer mgr.CloseManager()

	if mgr.GetAccessProvider() != provider {
		t.Fatalf("access provider = %T, want injected provider", mgr.GetAccessProvider())
	}
}

// TestBuildUsesInjectedPool verifies explicit pool reaches manager TestBuildUsesInjectedPool 验证显式任务池会传递给 Manager
func TestBuildUsesInjectedPool(t *testing.T) {
	pool := &testPool{}

	mgr, err := NewBuilder().
		IsPrintBanner(false).
		SetGenerator(&testGenerator{}).
		SetStorage(&testStorage{}).
		SetCodec(&testCodec{}).
		SetPool(pool).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	defer mgr.CloseManager()

	if mgr.GetPool() != pool {
		t.Fatalf("pool = %T, want injected pool", mgr.GetPool())
	}
}

// TestBuildRejectsInvalidNamespaceSetters verifies Builder does not hide invalid namespace values TestBuildRejectsInvalidNamespaceSetters 验证 Builder 不会吞掉非法命名空间值
func TestBuildRejectsInvalidNamespaceSetters(t *testing.T) {
	tests := []struct {
		name  string
		build func() (*manager.Manager, error)
	}{
		{
			name: "empty auth type",
			build: func() (*manager.Manager, error) {
				return NewBuilder().AuthType("").Build()
			},
		},
		{
			name: "empty key prefix",
			build: func() (*manager.Manager, error) {
				return NewBuilder().KeyPrefix("").Build()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr, err := tt.build()
			if err == nil {
				t.Fatal("Build() error = nil, want invalid namespace error")
			}
			if mgr != nil {
				t.Fatal("Build() should not return manager for invalid namespace")
			}
		})
	}
}

// TestBuildPreservesNilCookieConfig verifies explicit nil cookie config reaches validation TestBuildPreservesNilCookieConfig 验证显式 nil Cookie 配置会进入校验
func TestBuildPreservesNilCookieConfig(t *testing.T) {
	mgr, err := NewBuilder().
		CookieConfig(nil).
		IsReadCookie(true).
		Build()
	if err == nil {
		t.Fatal("Build() error = nil, want nil cookie config error")
	}
	if mgr != nil {
		t.Fatal("Build() should not return manager for nil cookie config")
	}
}

// TestBuildAllowsCoreStorageWithoutAtomicCapability verifies core build does not require nonce capability TestBuildAllowsCoreStorageWithoutAtomicCapability 验证核心构建不依赖 nonce 原子能力
func TestBuildAllowsCoreStorageWithoutAtomicCapability(t *testing.T) {
	mgr, err := NewBuilder().
		IsPrintBanner(false).
		SetGenerator(&testGenerator{}).
		SetStorage(&testBasicStorage{}).
		SetCodec(&testCodec{}).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	defer mgr.CloseManager()
}

// TestBuilderSetsCoreConcurrencyModes verifies core concurrency options stay on Builder TestBuilderSetsCoreConcurrencyModes 验证核心并发配置仍由 Builder 暴露
func TestBuilderSetsCoreConcurrencyModes(t *testing.T) {
	cfg := NewBuilder().
		ReplacedLoginExitMode(config.ReplacedLoginExitModeNewDevice).
		OverflowLogoutMode(config.LogoutModeReplaced).
		GetConfig()

	if cfg.ReplacedLoginExitMode != config.ReplacedLoginExitModeNewDevice {
		t.Fatalf("ReplacedLoginExitMode = %q, want %q", cfg.ReplacedLoginExitMode, config.ReplacedLoginExitModeNewDevice)
	}
	if cfg.OverflowLogoutMode != config.LogoutModeReplaced {
		t.Fatalf("OverflowLogoutMode = %q, want %q", cfg.OverflowLogoutMode, config.LogoutModeReplaced)
	}
}

// testGenerator provides a minimal generator for builder tests testGenerator 为 Builder 测试提供最小生成器
type testGenerator struct{}

// Generate returns a deterministic token Generate 返回固定 Token
func (g *testGenerator) Generate(loginID, device, deviceID string) (string, error) {
	return loginID + device + deviceID, nil
}

// testCodec provides a minimal codec for builder tests testCodec 为 Builder 测试提供最小编解码器
type testCodec struct{}

// Name returns the codec name Name 返回编解码器名称
func (c *testCodec) Name() string { return "test" }

// Encode returns an empty payload Encode 返回空载荷
func (c *testCodec) Encode(v any) ([]byte, error) { return []byte{}, nil }

// Decode accepts any payload Decode 接受任意载荷
func (c *testCodec) Decode(data []byte, v any) error { return nil }

// testStorage provides a minimal atomic storage for builder tests testStorage 为 Builder 测试提供最小原子存储
type testStorage struct{}

// Set stores one value Set 保存单个值
func (s *testStorage) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	return nil
}

// Get gets one value Get 读取单个值
func (s *testStorage) Get(ctx context.Context, key string) (any, error) { return nil, nil }

// Delete removes keys Delete 删除键
func (s *testStorage) Delete(ctx context.Context, keys ...string) error { return nil }

// Exists checks key presence Exists 检查键是否存在
func (s *testStorage) Exists(ctx context.Context, key string) bool { return false }

// Expire updates key expiration Expire 更新键过期时间
func (s *testStorage) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return nil
}

// TTL gets key lifetime TTL 获取键剩余时间
func (s *testStorage) TTL(ctx context.Context, key string) (time.Duration, error) {
	return adapter.TTLNotFound, nil
}

// Ping checks storage health Ping 检查存储健康状态
func (s *testStorage) Ping(ctx context.Context) error { return nil }

// GetAndDelete gets and deletes a key atomically GetAndDelete 原子读取并删除键
func (s *testStorage) GetAndDelete(ctx context.Context, key string) (any, error) { return nil, nil }

// GetAndDeleteMany gets and deletes a key and extra keys atomically GetAndDeleteMany 原子读取并删除主键和附加键
func (s *testStorage) GetAndDeleteMany(ctx context.Context, key string, deleteKeys ...string) (any, error) {
	return nil, nil
}

// SetIfAbsent stores a key only when absent SetIfAbsent 仅当键不存在时写入
func (s *testStorage) SetIfAbsent(ctx context.Context, key string, value any, expiration time.Duration) (bool, error) {
	return true, nil
}

// testBasicStorage provides storage without optional atomic capability testBasicStorage 提供不带可选原子能力的存储
type testBasicStorage struct{}

// Set stores one value Set 保存单个值
func (s *testBasicStorage) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	return nil
}

// Get gets one value Get 读取单个值
func (s *testBasicStorage) Get(ctx context.Context, key string) (any, error) { return nil, nil }

// Delete removes keys Delete 删除键
func (s *testBasicStorage) Delete(ctx context.Context, keys ...string) error { return nil }

// Exists checks key presence Exists 检查键是否存在
func (s *testBasicStorage) Exists(ctx context.Context, key string) bool { return false }

// Expire updates key expiration Expire 更新键过期时间
func (s *testBasicStorage) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return nil
}

// TTL gets key lifetime TTL 获取键剩余时间
func (s *testBasicStorage) TTL(ctx context.Context, key string) (time.Duration, error) {
	return adapter.TTLNotFound, nil
}

// Ping checks storage health Ping 检查存储健康状态
func (s *testBasicStorage) Ping(ctx context.Context) error { return nil }

// testLogger provides a minimal logger for builder tests testLogger 为 Builder 测试提供最小日志器
type testLogger struct{}

// Print writes a plain message Print 输出普通消息
func (l *testLogger) Print(v ...any) {}

// Printf writes a formatted plain message Printf 输出格式化普通消息
func (l *testLogger) Printf(format string, v ...any) {}

// Debug writes a debug message Debug 输出调试消息
func (l *testLogger) Debug(v ...any) {}

// Debugf writes a formatted debug message Debugf 输出格式化调试消息
func (l *testLogger) Debugf(format string, v ...any) {}

// Info writes an info message Info 输出信息消息
func (l *testLogger) Info(v ...any) {}

// Infof writes a formatted info message Infof 输出格式化信息消息
func (l *testLogger) Infof(format string, v ...any) {}

// Warn writes a warning message Warn 输出警告消息
func (l *testLogger) Warn(v ...any) {}

// Warnf writes a formatted warning message Warnf 输出格式化警告消息
func (l *testLogger) Warnf(format string, v ...any) {}

// Error writes an error message Error 输出错误消息
func (l *testLogger) Error(v ...any) {}

// Errorf writes a formatted error message Errorf 输出格式化错误消息
func (l *testLogger) Errorf(format string, v ...any) {}

// testAccessProvider provides a comparable provider for builder tests testAccessProvider 为 Builder 测试提供可比较的访问提供器
type testAccessProvider struct{}

// Permissions returns no provider permissions Permissions 不返回提供器权限
func (p *testAccessProvider) Permissions(ctx context.Context, subject manager.AccessSubject) ([]string, error) {
	return nil, nil
}

// Roles returns no provider roles Roles 不返回提供器角色
func (p *testAccessProvider) Roles(ctx context.Context, subject manager.AccessSubject) ([]string, error) {
	return nil, nil
}

// testPool provides a minimal pool for builder tests testPool 为 Builder 测试提供最小任务池
type testPool struct {
	stopped bool
}

// Submit runs task immediately Submit 立即执行任务
func (p *testPool) Submit(task func()) error {
	if task != nil {
		task()
	}
	return nil
}

// Stop records stop state Stop 记录停止状态
func (p *testPool) Stop() {
	p.stopped = true
}

// Stats returns fixed pool stats Stats 返回固定任务池状态
func (p *testPool) Stats() (running, capacity int, usage float64) {
	return 0, 1, 0
}
