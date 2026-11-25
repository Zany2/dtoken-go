// @Author daixk 2025/12/22 15:56:00
package builder

import (
	"fmt"

	"github.com/Zany2/dtoken-go/config"
	adapter "github.com/Zany2/dtoken-go/contract"
	"github.com/Zany2/dtoken-go/internal/banner"
	manager "github.com/Zany2/dtoken-go/internal/engine"
	"github.com/Zany2/dtoken-go/internal/utils"
)

// GeneratorFactory creates a token generator from config GeneratorFactory 根据配置创建 Token 生成器
type GeneratorFactory func(cfg *config.Config) (adapter.Generator, error)

// StorageFactory creates a storage adapter from config StorageFactory 根据配置创建存储适配器
type StorageFactory func(cfg *config.Config) (adapter.Storage, error)

// CodecFactory creates a codec adapter from config CodecFactory 根据配置创建编解码适配器
type CodecFactory func(cfg *config.Config) (adapter.Codec, error)

// LogFactory creates a logger adapter from config LogFactory 根据配置创建日志适配器
type LogFactory func(cfg *config.Config) (adapter.Log, error)

// PoolFactory creates an async task pool from config PoolFactory 根据配置创建异步任务池
type PoolFactory func(cfg *config.Config) (adapter.Pool, error)

// Components groups pluggable runtime components Components 聚合可替换的运行时组件
type Components struct {
	// Generator stores token generator Generator 存储 Token 生成器
	Generator adapter.Generator

	// Storage stores storage adapter Storage 存储存储适配器
	Storage adapter.Storage

	// Codec stores codec adapter Codec 存储编解码适配器
	Codec adapter.Codec

	// Log stores logger adapter Log 存储日志适配器
	Log adapter.Log

	// Pool stores async task pool Pool 存储异步任务池
	Pool adapter.Pool
}

// ComponentFactories groups default component factories ComponentFactories 聚合默认组件工厂
type ComponentFactories struct {
	// Generator stores generator factory Generator 存储生成器工厂
	Generator GeneratorFactory

	// Storage stores storage factory Storage 存储存储工厂
	Storage StorageFactory

	// Codec stores codec factory Codec 存储编解码工厂
	Codec CodecFactory

	// Log stores logger factory Log 存储日志工厂
	Log LogFactory

	// Pool stores pool factory Pool 存储任务池工厂
	Pool PoolFactory
}

// Build validates a config snapshot and resolves components without retaining builder state. Build 校验配置快照并解析组件，不持有构建器状态。
func Build(cfg *config.Config, components Components, factories ComponentFactories, accessProvider manager.AccessProvider, options ...manager.Option) (*manager.Manager, error) {
	cfg = cfg.Clone()

	// Validate config before resolving components 装配组件前先校验配置
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("build manager failed: invalid config: %w", err)
	}

	// A plain nil provider is optional; an interface containing nil is an invalid injection. 普通 nil 提供器表示未配置，接口内的 nil 则属于无效注入。
	if accessProvider != nil && utils.IsNil(accessProvider) {
		return nil, fmt.Errorf("build manager failed: access provider contains nil")
	}

	// Resolve components per build so later config changes do not reuse stale factory products 每次构建独立装配组件，避免后续配置变化继续复用旧工厂产物
	var createdStorage adapter.Storage
	var createdLogger adapter.Log
	var createdPool adapter.Pool
	cleanup := func() {
		// Stop factory-created pools before closing their dependent storage. 先停止工厂创建的协程池，再关闭其依赖的存储。
		if !utils.IsNil(createdPool) {
			createdPool.Stop()
			createdPool = nil
		}

		// Close factory-created storage only; explicit components remain caller-owned on failed builds. 仅关闭工厂创建的存储，构建失败时显式组件仍由调用方持有。
		if storageCloser, ok := createdStorage.(interface{ Close() error }); ok && !utils.IsNil(createdStorage) {
			_ = storageCloser.Close()
		}
		createdStorage = nil

		// Flush and close factory-created loggers when they expose lifecycle control. 工厂创建的日志器支持生命周期控制时先刷新再关闭。
		if logControl, ok := createdLogger.(adapter.LogControl); ok && !utils.IsNil(createdLogger) {
			logControl.Flush()
			logControl.Close()
		}
		createdLogger = nil
	}

	// Resolve token generator 解析 Token 生成器
	if components.Generator == nil {
		if factories.Generator != nil {
			generator, err := factories.Generator(cfg.Clone())
			if err != nil {
				cleanup()
				return nil, fmt.Errorf("build manager failed: create token generator failed: %w", err)
			}
			components.Generator = generator
		}
	}
	if utils.IsNil(components.Generator) {
		cleanup()
		return nil, fmt.Errorf("build manager failed: token generator is missing, call SetGenerator or SetGeneratorFactory")
	}

	// Resolve storage adapter 解析存储适配器
	if components.Storage == nil {
		if factories.Storage != nil {
			storage, err := factories.Storage(cfg.Clone())
			if err != nil {
				cleanup()
				return nil, fmt.Errorf("build manager failed: create storage adapter failed: %w", err)
			}
			components.Storage = storage
			createdStorage = storage
		}
	}
	if utils.IsNil(components.Storage) {
		cleanup()
		return nil, fmt.Errorf("build manager failed: storage adapter is missing, call SetStorage or SetStorageFactory")
	}

	// Resolve codec adapter 解析编解码适配器
	if components.Codec == nil {
		if factories.Codec != nil {
			codec, err := factories.Codec(cfg.Clone())
			if err != nil {
				cleanup()
				return nil, fmt.Errorf("build manager failed: create codec adapter failed: %w", err)
			}
			components.Codec = codec
		}
	}
	if utils.IsNil(components.Codec) {
		cleanup()
		return nil, fmt.Errorf("build manager failed: codec adapter is missing, call SetCodec or SetCodecFactory")
	}

	// Resolve logger adapter 解析日志适配器
	if cfg.IsLog {
		if components.Log == nil {
			if factories.Log != nil {
				logger, err := factories.Log(cfg.Clone())
				if err != nil {
					cleanup()
					return nil, fmt.Errorf("build manager failed: create log adapter failed: %w", err)
				}
				components.Log = logger
				createdLogger = logger
			}
		}
		if utils.IsNil(components.Log) {
			cleanup()
			return nil, fmt.Errorf("build manager failed: log adapter is missing, call SetLog or SetLogFactory")
		}
	} else {
		// Use no-op logger when logging is disabled 日志关闭时使用空日志器
		components.Log = adapter.NewNopLogger()
	}

	// Resolve optional renew pool 解析可选续期任务池
	if cfg.AutoRenew && components.Pool == nil && factories.Pool != nil {
		pool, err := factories.Pool(cfg.Clone())
		if err != nil {
			cleanup()
			return nil, fmt.Errorf("build manager failed: create renew task pool failed: %w", err)
		}
		components.Pool = pool
		createdPool = pool
	}
	if components.Pool != nil && utils.IsNil(components.Pool) {
		cleanup()
		return nil, fmt.Errorf("build manager failed: renew task pool contains nil")
	}

	// Print banner when enabled 开启时打印 Banner
	if cfg.IsPrintBanner {
		banner.PrintBanner(cfg)
	}

	// Assemble manager with resolved components 使用解析后的组件装配 Manager
	mgr := manager.NewManager(
		cfg,
		components.Generator,
		components.Storage,
		components.Codec,
		components.Log,
		components.Pool,
		accessProvider,
		options...,
	)

	return mgr, nil
}
