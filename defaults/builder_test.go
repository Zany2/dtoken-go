package defaults

import (
	"context"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/com/generator/dgenerator"
	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
)

// TestDefaultFactoriesCreateUsableComponents verifies every bundled factory returns a usable component. TestDefaultFactoriesCreateUsableComponents 验证每个内置工厂都能创建可用组件。
func TestDefaultFactoriesCreateUsableComponents(t *testing.T) {
	ctx := context.Background()

	cfg := config.DefaultConfig()
	cfg.TokenStyle = adapter.TokenStyleJWT
	cfg.JwtSecretKey = "defaults-test-secret"
	cfg.Timeout = 60
	generator, err := defaultGeneratorFactory(cfg)
	if err != nil || generator == nil {
		t.Fatalf("defaultGeneratorFactory() = %v, %v; want usable generator", generator, err)
	}
	token, err := generator.Generate("factory-user", "web", "browser-1")
	if err != nil || token == "" {
		t.Fatalf("default generator Generate() = %q, %v; want token", token, err)
	}
	_, ok := generator.(*dgenerator.Generator)
	if !ok {
		t.Fatalf("default generator type = %T, want *dgenerator.Generator", generator)
	}
	parser := dgenerator.NewGenerator(60, "defaults-test-secret", adapter.TokenStyleJWT)
	claims, err := parser.ParseJWT(token)
	if err != nil || claims["loginId"] != "factory-user" {
		t.Fatalf("default generator ParseJWT() = %#v, %v; want factory-user claim", claims, err)
	}

	storage, err := defaultStorageFactory(cfg)
	if err != nil || storage == nil {
		t.Fatalf("defaultStorageFactory() = %v, %v; want usable storage", storage, err)
	}
	if err = storage.Set(ctx, "defaults:test", "value", time.Minute); err != nil {
		t.Fatalf("default storage Set() error = %v", err)
	}
	stored, err := storage.Get(ctx, "defaults:test")
	if err != nil || stored != "value" {
		t.Fatalf("default storage Get() = %#v, %v; want value", stored, err)
	}

	codec, err := defaultCodecFactory(cfg)
	if err != nil || codec == nil {
		t.Fatalf("defaultCodecFactory() = %v, %v; want usable codec", codec, err)
	}
	if codec.Name() != "json" {
		t.Fatalf("default codec Name() = %q, want json", codec.Name())
	}
	payload, err := codec.Encode(map[string]string{"name": "defaults"})
	if err != nil {
		t.Fatalf("default codec Encode() error = %v", err)
	}
	decoded := map[string]string{}
	if err = codec.Decode(payload, &decoded); err != nil || decoded["name"] != "defaults" {
		t.Fatalf("default codec Decode() = %#v, %v; want defaults", decoded, err)
	}

	tempDir := t.TempDir()
	t.Chdir(tempDir)
	logger, err := defaultLogFactory(cfg)
	if err != nil || logger == nil {
		t.Fatalf("defaultLogFactory() = %v, %v; want usable logger", logger, err)
	}
	loggerControl, ok := logger.(interface{ Close() })
	if !ok {
		t.Fatalf("default logger type = %T, want closeable logger", logger)
	}
	loggerControl.Close()

	pool, err := defaultPoolFactory(cfg)
	if err != nil || pool == nil {
		t.Fatalf("defaultPoolFactory() = %v, %v; want usable pool", pool, err)
	}
	defer pool.Stop()
	_, capacity, _ := pool.Stats()
	if capacity <= 0 {
		t.Fatalf("default pool capacity = %d, want positive", capacity)
	}
}

// TestDefaultGeneratorFactoryUsesConfig verifies generator factory forwards token style and JWT settings. TestDefaultGeneratorFactoryUsesConfig 验证生成器工厂透传 Token 风格和 JWT 配置。
func TestDefaultGeneratorFactoryUsesConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.TokenStyle = adapter.TokenStyleRandom32
	generator, err := defaultGeneratorFactory(cfg)
	if err != nil {
		t.Fatalf("defaultGeneratorFactory(random32) error = %v", err)
	}
	token, err := generator.Generate("style-user", "", "")
	if err != nil {
		t.Fatalf("random32 generator Generate() error = %v", err)
	}
	if len(token) != 32 {
		t.Fatalf("random32 token length = %d, want 32", len(token))
	}

	cfg.TokenStyle = adapter.TokenStyleJWT
	cfg.JwtSecretKey = "forwarded-secret"
	cfg.Timeout = 90
	generator, err = defaultGeneratorFactory(cfg)
	if err != nil {
		t.Fatalf("defaultGeneratorFactory(jwt) error = %v", err)
	}
	_, ok := generator.(*dgenerator.Generator)
	if !ok {
		t.Fatalf("default JWT generator type = %T, want *dgenerator.Generator", generator)
	}
	token, err = generator.Generate("jwt-user", "web", "device-1")
	if err != nil {
		t.Fatalf("JWT generator Generate() error = %v", err)
	}
	parser := dgenerator.NewGenerator(90, "forwarded-secret", adapter.TokenStyleJWT)
	if _, err = parser.ParseJWT(token); err != nil {
		t.Fatalf("JWT generated with forwarded secret could not be parsed = %v", err)
	}
}

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

// TestNewBuilderCreatesDefaultRenewPool verifies AutoRenew enables the bundled pool factory. TestNewBuilderCreatesDefaultRenewPool 验证开启自动续期时会装配内置协程池。
func TestNewBuilderCreatesDefaultRenewPool(t *testing.T) {
	mgr, err := NewBuilder().
		AuthType("defaults-pool").
		IsPrintBanner(false).
		Build()
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
