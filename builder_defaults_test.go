package dtoken

import (
	"context"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/config"
)

// TestJWTLoginRespectsTokenSharing verifies unique JWT issuance does not change Manager token sharing. TestJWTLoginRespectsTokenSharing 验证 JWT 独立签发不会改变 Manager 的 Token 共享策略。
func TestJWTLoginRespectsTokenSharing(t *testing.T) {
	for _, shared := range []bool{false, true} {
		name := "independent"
		if shared {
			name = "shared"
		}
		t.Run(name, func(t *testing.T) {
			mgr, err := NewBuilder().
				IsPrintBanner(false).
				AutoRenew(false).
				AsyncEvent(false).
				IsShare(shared).
				JwtSecret("jwt-sharing-test-secret").
				buildManager()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(mgr.CloseManager)
			ctx := context.Background()
			first, err := mgr.Login(ctx, "jwt-user", "web", "browser")
			if err != nil {
				t.Fatal(err)
			}
			second, err := mgr.Login(ctx, "jwt-user", "web", "browser")
			if err != nil {
				t.Fatalf("repeated JWT login failed: %v", err)
			}
			if (first == second) != shared {
				t.Fatalf("token reuse = %v, want %v", first == second, shared)
			}
		})
	}
}

// TestNewBuilderBuildsUsableManager verifies bundled defaults assemble a working manager. TestNewBuilderBuildsUsableManager 验证内置默认组件能装配可用管理器。
func TestNewBuilderBuildsUsableManager(t *testing.T) {
	mgr, err := NewBuilder().
		AuthType("defaults").
		IsPrintBanner(false).
		AutoRenew(false).
		buildManager()
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
		buildManager()
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

// TestNewBuilderCreatesDefaultRenewPool verifies AutoRenew enables the bundled pool factory. TestNewBuilderCreatesDefaultRenewPool 验证开启自动续期时会装配内置协程池。
func TestNewBuilderCreatesDefaultRenewPool(t *testing.T) {
	mgr, err := NewBuilder().
		AuthType("defaults-pool").
		IsPrintBanner(false).
		buildManager()
	if err != nil {
		t.Fatalf("Build(AutoRenew) error = %v", err)
	}
	defer mgr.CloseManager()

	pool := mgr.GetPool()
	if pool == nil {
		t.Fatal("GetPool() = nil with AutoRenew enabled")
	}
	_, capacity, _ := pool.Stats()
	if capacity <= 0 {
		t.Fatalf("default renew pool capacity = %d, want positive", capacity)
	}
}
