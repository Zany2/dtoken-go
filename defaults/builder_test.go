package defaults

import (
	"context"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/config"
)

// TestNewBuilderBuildsUsableManager verifies bundled defaults assemble a working manager. TestNewBuilderBuildsUsableManager 验证内置默认组件能装配可用管理器。
func TestNewBuilderBuildsUsableManager(t *testing.T) {
	mgr, err := NewBuilder().
		AuthType("defaults").
		IsPrintBanner(false).
		AutoRenew(false).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	defer mgr.CloseManager()

	if mgr.GetConfig().AuthType != "defaults:" {
		t.Fatalf("AuthType = %q, want defaults:", mgr.GetConfig().AuthType)
	}
	if mgr.GetGenerator() == nil {
		t.Fatal("GetGenerator() = nil")
	}
	if mgr.GetStorage() == nil {
		t.Fatal("GetStorage() = nil")
	}
	if mgr.GetSerializer() == nil {
		t.Fatal("GetSerializer() = nil")
	}
	if mgr.GetLogger() == nil {
		t.Fatal("GetLogger() = nil")
	}
	token, err := mgr.Login(context.Background(), "defaults-user", "web")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if !mgr.IsLogin(context.Background(), token) {
		t.Fatal("IsLogin() = false, want true")
	}
}

// TestNewBuilderAppliesDefaultConfig verifies defaults can still be overridden through core builder options. TestNewBuilderAppliesDefaultConfig 验证默认构建器仍可通过核心选项覆盖配置。
func TestNewBuilderAppliesDefaultConfig(t *testing.T) {
	mgr, err := NewBuilder().
		AuthType("defaults-config").
		IsPrintBanner(false).
		AutoRenew(false).
		TimeoutDuration(2 * time.Minute).
		CookiePath("/api").
		CookieSecure(true).
		CookieSameSite(config.SameSiteStrict).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	defer mgr.CloseManager()

	cfg := mgr.GetConfig()
	if cfg.Timeout != 120 {
		t.Fatalf("Timeout = %d, want 120", cfg.Timeout)
	}
	if cfg.CookieConfig == nil {
		t.Fatal("CookieConfig = nil")
	}
	if cfg.CookieConfig.Path != "/api" || !cfg.CookieConfig.Secure || cfg.CookieConfig.SameSite != config.SameSiteStrict {
		t.Fatalf("CookieConfig = %+v, want overridden cookie config", cfg.CookieConfig)
	}
}
