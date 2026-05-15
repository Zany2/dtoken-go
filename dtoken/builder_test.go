// @Author daixk 2026/05/15
package dtoken

import (
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/core/nonce"
)

// TestNewBuilderReturnsDefaultBuilder verifies the facade exposes a ready builder TestNewBuilderReturnsDefaultBuilder 验证门面入口会返回可用的 Builder
func TestNewBuilderReturnsDefaultBuilder(t *testing.T) {
	if NewBuilder() == nil {
		t.Fatal("NewBuilder() returned nil")
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
	if _, err := NewBuilder().IsPrintBanner(false).LoggerQueueSize(0).Build(); err == nil {
		t.Fatal("Build() error = nil, want invalid logger config error")
	}
	if _, err := NewBuilder().IsPrintBanner(false).NonceTTL(0).Build(); err == nil {
		t.Fatal("Build() error = nil, want invalid nonce config error")
	}
	if _, err := NewBuilder().IsPrintBanner(false).OAuth2TokenExpiration(0).Build(); err == nil {
		t.Fatal("Build() error = nil, want invalid OAuth2 config error")
	}
}

// TestBuilderKeepsDefaultModulesWithExtraOption verifies generic options do not disable configured modules TestBuilderKeepsDefaultModulesWithExtraOption 验证通用选项不会关闭已配置的默认模块
func TestBuilderKeepsDefaultModulesWithExtraOption(t *testing.T) {
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
		t.Fatal("Build() should keep default nonce and OAuth2 managers when extra options are used")
	}
}

// TestBuilderAppliesUserOptionsAfterDefaultModules verifies user options can still override defaults TestBuilderAppliesUserOptionsAfterDefaultModules 验证用户选项仍可覆盖默认模块
func TestBuilderAppliesUserOptionsAfterDefaultModules(t *testing.T) {
	customNonce := nonce.NewNonceManager(
		config.DefaultAuthType,
		config.DefaultKeyPrefix,
		nil,
		time.Minute,
	)

	mgr, err := NewBuilder().
		IsPrintBanner(false).
		AutoRenew(false).
		UseManagerOption(manager.WithNonceManager(customNonce)).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	defer mgr.CloseManager()

	if mgr.GetNonceManager() != customNonce {
		t.Fatal("Build() should apply user manager options after configured defaults")
	}
}
