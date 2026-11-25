// @Author daixk 2025/12/22 15:56:00
package builder

import (
	djson "github.com/Zany2/dtoken-go/codec/json"
	"github.com/Zany2/dtoken-go/config"
	adapter "github.com/Zany2/dtoken-go/contract"
	"github.com/Zany2/dtoken-go/generator/dgenerator"
	"github.com/Zany2/dtoken-go/logging/dlog"
	"github.com/Zany2/dtoken-go/pool/ants"
	"github.com/Zany2/dtoken-go/storage/memory"
)

// DefaultGeneratorFactory creates the bundled token generator DefaultGeneratorFactory 创建内置 Token 生成器
func DefaultGeneratorFactory(cfg *config.Config) (adapter.Generator, error) {
	return dgenerator.NewGenerator(cfg.Timeout, cfg.JwtSecretKey, cfg.TokenStyle), nil
}

// DefaultStorageFactory creates the bundled memory storage DefaultStorageFactory 创建内置内存存储
func DefaultStorageFactory(_ *config.Config) (adapter.Storage, error) {
	return memory.NewStorage(), nil
}

// DefaultCodecFactory creates the bundled JSON codec DefaultCodecFactory 创建内置 JSON 编解码器
func DefaultCodecFactory(_ *config.Config) (adapter.Codec, error) {
	return djson.NewJSONSerializer(), nil
}

// DefaultLogFactory creates the bundled logger DefaultLogFactory 创建内置日志器
func DefaultLogFactory(cfg *dlog.LoggerConfig) (adapter.Log, error) {
	return dlog.NewLoggerWithConfig(cfg)
}

// DefaultPoolFactory creates the bundled renew pool DefaultPoolFactory 创建内置续期协程池
func DefaultPoolFactory(cfg *ants.RenewPoolConfig) (adapter.Pool, error) {
	return ants.NewRenewPoolManagerWithConfig(cfg)
}
