package defaults

import (
	djson "github.com/Zany2/dtoken-go/com/codec/json"
	"github.com/Zany2/dtoken-go/com/generator/dgenerator"
	"github.com/Zany2/dtoken-go/com/log/dlog"
	"github.com/Zany2/dtoken-go/com/pool/ants"
	"github.com/Zany2/dtoken-go/com/storage/memory"
	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/builder"
	"github.com/Zany2/dtoken-go/core/config"
)

// NewBuilder creates a builder wired with bundled default components NewBuilder 创建已装配内置默认组件的构建器
func NewBuilder() *builder.Builder {
	return builder.NewBuilder().
		SetGeneratorFactory(defaultGeneratorFactory).
		SetStorageFactory(defaultStorageFactory).
		SetCodecFactory(defaultCodecFactory).
		SetLogFactory(defaultLogFactory).
		SetPoolFactory(defaultPoolFactory)
}

// defaultGeneratorFactory creates the bundled token generator defaultGeneratorFactory 创建内置 Token 生成器
func defaultGeneratorFactory(cfg *config.Config) (adapter.Generator, error) {
	return dgenerator.NewGenerator(cfg.Timeout, cfg.JwtSecretKey, cfg.TokenStyle), nil
}

// defaultStorageFactory creates the bundled memory storage defaultStorageFactory 创建内置内存存储
func defaultStorageFactory(_ *config.Config) (adapter.Storage, error) {
	return memory.NewStorage(), nil
}

// defaultCodecFactory creates the bundled JSON codec defaultCodecFactory 创建内置 JSON 编解码器
func defaultCodecFactory(_ *config.Config) (adapter.Codec, error) {
	return djson.NewJSONSerializer(), nil
}

// defaultLogFactory creates the bundled logger defaultLogFactory 创建内置日志器
func defaultLogFactory(_ *config.Config) (adapter.Log, error) {
	return dlog.NewLoggerWithConfig(dlog.DefaultLoggerConfig())
}

// defaultPoolFactory creates the bundled renew pool defaultPoolFactory 创建内置续期协程池
func defaultPoolFactory(_ *config.Config) (adapter.Pool, error) {
	return ants.NewRenewPoolManagerWithConfig(ants.DefaultRenewPoolConfig())
}
