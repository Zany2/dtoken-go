// @Author daixk 2026/05/15
package dtoken

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/core/nonce"
)

// TestNewBuilderReturnsDefaultBuilder verifies the facade exposes a ready builder TestNewBuilderReturnsDefaultBuilder 验证门面入口会返回可用的 Builder
func TestNewBuilderReturnsDefaultBuilder(t *testing.T) {
	if NewBuilder() == nil {
		t.Fatal("NewBuilder() returned nil")
	}
}

// TestZeroValueBuilderInitializesDefaults verifies a zero-value builder lazily initializes every config. TestZeroValueBuilderInitializesDefaults 验证零值 Builder 会惰性初始化全部配置。
func TestZeroValueBuilderInitializesDefaults(t *testing.T) {
	var coreOnly Builder
	if coreOnly.GetConfig() == nil || coreOnly.GetRenewPoolConfig() == nil || coreOnly.GetLoggerConfig() == nil {
		t.Fatal("zero-value Builder should initialize core configs")
	}

	mgr, err := coreOnly.IsPrintBanner(false).AutoRenew(false).Build()
	if err != nil {
		t.Fatalf("zero-value Build() error = %v", err)
	}
	defer mgr.CloseManager()
	if mgr.GetNonceManager() != nil || mgr.GetOAuth2Manager() != nil || mgr.GetTicketManager() != nil || mgr.GetShortKeyManager() != nil {
		t.Fatal("zero-value Builder should keep optional modules disabled")
	}

	var optional Builder
	if optional.GetNonceConfig() == nil || optional.GetOAuth2Config() == nil || optional.GetTicketConfig() == nil || optional.GetShortKeyConfig() == nil {
		t.Fatal("zero-value Builder should initialize optional module configs")
	}
	optional.IsPrintBanner(false).AutoRenew(false)
	optionalManager, err := optional.Build()
	if err != nil {
		t.Fatalf("zero-value optional Build() error = %v", err)
	}
	defer optionalManager.CloseManager()
	if optionalManager.GetNonceManager() == nil || optionalManager.GetOAuth2Manager() == nil || optionalManager.GetTicketManager() == nil || optionalManager.GetShortKeyManager() == nil {
		t.Fatal("optional config getters should enable their modules")
	}
}

// TestBuilderNilConfigsRestoreDefaults verifies nil config setters restore independent defaults. TestBuilderNilConfigsRestoreDefaults 验证传入 nil 会恢复独立的默认配置。
func TestBuilderNilConfigsRestoreDefaults(t *testing.T) {
	b := NewBuilder().
		RenewPoolMinSize(99).
		LoggerPrefix("custom").
		NonceTTL(time.Second).
		OAuth2TokenExpiration(time.Minute).
		TicketTTL(time.Second).
		ShortKeyLength(3).
		RenewPoolConfig(nil).
		LoggerConfig(nil).
		NonceConfig(nil).
		OAuth2Config(nil).
		TicketConfig(nil).
		ShortKeyConfig(nil)

	defaults := NewBuilder()
	if b.GetRenewPoolConfig().MinSize != defaults.GetRenewPoolConfig().MinSize {
		t.Fatalf("renew pool MinSize = %d, want default %d", b.GetRenewPoolConfig().MinSize, defaults.GetRenewPoolConfig().MinSize)
	}
	if b.GetLoggerConfig().Prefix != defaults.GetLoggerConfig().Prefix {
		t.Fatalf("logger Prefix = %q, want default %q", b.GetLoggerConfig().Prefix, defaults.GetLoggerConfig().Prefix)
	}
	if b.GetNonceConfig().TTL != defaults.GetNonceConfig().TTL || b.GetOAuth2Config().TokenExpiration != defaults.GetOAuth2Config().TokenExpiration || b.GetTicketConfig().TTL != defaults.GetTicketConfig().TTL || b.GetShortKeyConfig().Length != defaults.GetShortKeyConfig().Length {
		t.Fatal("nil optional configs should restore module defaults")
	}
}

// TestBuilderBuildsCoreOnlyByDefault verifies optional modules are opt-in TestBuilderBuildsCoreOnlyByDefault 验证默认只装配核心能力。
func TestBuilderBuildsCoreOnlyByDefault(t *testing.T) {
	mgr, err := NewBuilder().
		IsPrintBanner(false).
		AutoRenew(false).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	defer mgr.CloseManager()

	if mgr.GetNonceManager() != nil || mgr.GetOAuth2Manager() != nil || mgr.GetTicketManager() != nil || mgr.GetShortKeyManager() != nil {
		t.Fatal("Build() should not attach optional managers by default")
	}
	if _, err = mgr.GenerateNonce(context.Background()); !errors.Is(err, derror.ErrModuleNotEnabled) {
		t.Fatalf("GenerateNonce() error = %v, want ErrModuleNotEnabled", err)
	}
}

// TestBuilderBuildsWithModuleConfig verifies high-level module config chain TestBuilderBuildsWithModuleConfig 验证高层模块配置链路
func TestBuilderBuildsWithModuleConfig(t *testing.T) {
	mgr, err := NewBuilder().
		IsPrintBanner(false).
		AutoRenew(false).
		Timeout(3600).
		CookiePath("/").
		RenewPoolMinSize(2).
		RenewPoolMaxSize(4).
		LoggerQueueSize(2048).
		NonceTTL(5 * time.Minute).
		OAuth2TokenExpiration(2 * time.Hour).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	defer mgr.CloseManager()

	if mgr.GetNonceManager() == nil || mgr.GetOAuth2Manager() == nil {
		t.Fatal("Build() should attach nonce and OAuth2 managers")
	}
}

// TestBuilderRejectsInvalidModuleConfig verifies module validation runs before assembly TestBuilderRejectsInvalidModuleConfig 验证装配前会执行模块校验
func TestBuilderRejectsInvalidModuleConfig(t *testing.T) {
	if _, err := NewBuilder().IsPrintBanner(false).RenewPoolMinSize(0).Build(); err == nil {
		t.Fatal("Build() error = nil, want invalid renew pool config error")
	}
	if _, err := NewBuilder().IsPrintBanner(false).LoggerFileFormat("logs/app.log").Build(); err == nil {
		t.Fatal("Build() error = nil, want invalid logger config error")
	}
	if _, err := NewBuilder().IsPrintBanner(false).NonceTTL(0).Build(); err == nil {
		t.Fatal("Build() error = nil, want invalid nonce config error")
	}
	if _, err := NewBuilder().IsPrintBanner(false).OAuth2TokenExpiration(0).Build(); err == nil {
		t.Fatal("Build() error = nil, want invalid OAuth2 config error")
	}
	if _, err := NewBuilder().IsPrintBanner(false).TicketTTL(0).Build(); err == nil {
		t.Fatal("Build() error = nil, want invalid ticket config error")
	}
	if _, err := NewBuilder().IsPrintBanner(false).ShortKeyLength(0).Build(); err == nil {
		t.Fatal("Build() error = nil, want invalid short key config error")
	}
}

// TestBuilderKeepsEnabledModulesWithExtraOption verifies generic options do not disable enabled modules TestBuilderKeepsEnabledModulesWithExtraOption 验证通用选项不会关闭已启用模块
func TestBuilderKeepsEnabledModulesWithExtraOption(t *testing.T) {
	mgr, err := NewBuilder().
		IsPrintBanner(false).
		AutoRenew(false).
		NonceTTL(time.Minute).
		OAuth2TokenExpiration(2 * time.Hour).
		UseManagerOption(func(m *manager.Manager) {}).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	defer mgr.CloseManager()

	if mgr.GetNonceManager() == nil || mgr.GetOAuth2Manager() == nil {
		t.Fatal("Build() should keep enabled nonce and OAuth2 managers when extra options are used")
	}
}

// TestBuilderAppliesUserOptionsAfterEnabledModules verifies user options can still override enabled defaults TestBuilderAppliesUserOptionsAfterEnabledModules 验证用户选项仍可覆盖已启用模块
func TestBuilderAppliesUserOptionsAfterEnabledModules(t *testing.T) {
	customNonce := nonce.NewNonceManager(
		config.DefaultAuthType,
		config.DefaultKeyPrefix,
		nil,
		time.Minute,
	)

	mgr, err := NewBuilder().
		IsPrintBanner(false).
		AutoRenew(false).
		EnableNonce().
		UseManagerOption(manager.WithNonceManager(customNonce)).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	defer mgr.CloseManager()

	if mgr.GetNonceManager() != customNonce {
		t.Fatal("Build() should apply user manager options after enabled defaults")
	}
}
