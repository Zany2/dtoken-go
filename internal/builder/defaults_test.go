package builder

import (
	"context"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/config"
	adapter "github.com/Zany2/dtoken-go/contract"
	"github.com/Zany2/dtoken-go/generator/dgenerator"
)

// TestDefaultFactoriesCreateUsableComponents verifies every bundled factory returns a usable component. TestDefaultFactoriesCreateUsableComponents 验证每个内置工厂都能创建可用组件。
func TestDefaultFactoriesCreateUsableComponents(t *testing.T) {
	ctx := context.Background()

	cfg := config.DefaultConfig()
	cfg.TokenStyle = adapter.TokenStyleJWT
	cfg.JwtSecretKey = "defaults-test-secret"
	cfg.Timeout = 60
	generator, err := DefaultGeneratorFactory(cfg)
	if err != nil || generator == nil {
		t.Fatalf("DefaultGeneratorFactory() = %v, %v; want usable generator", generator, err)
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

	storage, err := DefaultStorageFactory(cfg)
	if err != nil || storage == nil {
		t.Fatalf("DefaultStorageFactory() = %v, %v; want usable storage", storage, err)
	}
	if err = storage.Set(ctx, "defaults:test", "value", time.Minute); err != nil {
		t.Fatalf("default storage Set() error = %v", err)
	}
	stored, err := storage.Get(ctx, "defaults:test")
	if err != nil || stored != "value" {
		t.Fatalf("default storage Get() = %#v, %v; want value", stored, err)
	}

	codec, err := DefaultCodecFactory(cfg)
	if err != nil || codec == nil {
		t.Fatalf("DefaultCodecFactory() = %v, %v; want usable codec", codec, err)
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
	logger, err := DefaultLogFactory(nil)
	if err != nil || logger == nil {
		t.Fatalf("DefaultLogFactory() = %v, %v; want usable logger", logger, err)
	}
	loggerControl, ok := logger.(interface{ Close() })
	if !ok {
		t.Fatalf("default logger type = %T, want closeable logger", logger)
	}
	loggerControl.Close()

	pool, err := DefaultPoolFactory(nil)
	if err != nil || pool == nil {
		t.Fatalf("DefaultPoolFactory() = %v, %v; want usable pool", pool, err)
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
	generator, err := DefaultGeneratorFactory(cfg)
	if err != nil {
		t.Fatalf("DefaultGeneratorFactory(random32) error = %v", err)
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
	generator, err = DefaultGeneratorFactory(cfg)
	if err != nil {
		t.Fatalf("DefaultGeneratorFactory(jwt) error = %v", err)
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
