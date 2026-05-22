// @Author daixk 2025/12/22 15:56:00
package builder

import (
	"fmt"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/banner"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/core/nonce"
	"github.com/Zany2/dtoken-go/core/oauth2"
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

// Builder builds a Manager from config and replaceable components Builder 根据配置和可替换组件构建 Manager
type Builder struct {
	// cfg stores mutable builder config cfg 存储 Builder 当前配置
	cfg *config.Config
	// components stores explicit runtime components components 存储显式设置的运行时组件
	components Components
	// factories stores default component factories factories 存储默认组件工厂
	factories ComponentFactories
	// managerOptions stores optional manager assembly options managerOptions 存储 Manager 可选模块装配项
	managerOptions []manager.Option
	// accessProvider stores permission and role provider accessProvider 存储权限与角色提供器
	accessProvider manager.AccessProvider

	// customPermissionListFunc custom account permission callback customPermissionListFunc 自定义账号权限回调
	customPermissionListFunc func(loginID, authType string) ([]string, error)
	// customRoleListFunc custom account role callback customRoleListFunc 自定义账号角色回调
	customRoleListFunc func(loginID, authType string) ([]string, error)
	// customPermissionListExtFunc custom terminal permission callback customPermissionListExtFunc 自定义终端权限回调
	customPermissionListExtFunc func(loginID, device, deviceId, authType string) ([]string, error)
	// customRoleListExtFunc custom terminal role callback customRoleListExtFunc 自定义终端角色回调
	customRoleListExtFunc func(loginID, device, deviceId, authType string) ([]string, error)
}

// NewBuilder creates a builder with the default config NewBuilder 使用默认配置创建 Builder
func NewBuilder() *Builder {
	return &Builder{cfg: config.DefaultConfig()}
}

// Config replaces the builder config with a clone of cfg Config 使用 cfg 的副本替换 Builder 配置
func (b *Builder) Config(cfg *config.Config) *Builder {
	if cfg == nil {
		b.cfg = config.DefaultConfig()
		return b
	}
	b.cfg = cfg.Clone()
	return b
}

// GetConfig returns the mutable builder config GetConfig 返回可修改的 Builder 配置
func (b *Builder) GetConfig() *config.Config {
	b.ensureConfig()
	return b.cfg
}

// AuthType sets auth type with suffix fix AuthType 设置认证体系类型并补齐后缀
func (b *Builder) AuthType(authType string) *Builder {
	b.ensureConfig()
	b.cfg.SetAuthType(authType)
	return b
}

// KeyPrefix sets key prefix with suffix fix KeyPrefix 设置存储键前缀并补齐后缀
func (b *Builder) KeyPrefix(keyPrefix string) *Builder {
	b.ensureConfig()
	b.cfg.SetKeyPrefix(keyPrefix)
	return b
}

// TokenName sets token name TokenName 设置 Token 名称
func (b *Builder) TokenName(name string) *Builder {
	b.ensureConfig()
	b.cfg.TokenName = name
	return b
}

// Timeout sets timeout seconds Timeout 设置超时时长秒数
func (b *Builder) Timeout(seconds int64) *Builder {
	b.ensureConfig()
	b.cfg.Timeout = seconds
	return b
}

// TimeoutDuration sets timeout by duration TimeoutDuration 使用时长设置超时时间
func (b *Builder) TimeoutDuration(d time.Duration) *Builder {
	b.ensureConfig()
	b.cfg.Timeout = durationToSeconds(d)
	return b
}

// AutoRenew sets auto renew switch AutoRenew 设置是否自动续期
func (b *Builder) AutoRenew(autoRenew bool) *Builder {
	b.ensureConfig()
	b.cfg.AutoRenew = autoRenew
	return b
}

// RenewMaxRefresh sets renew trigger threshold RenewMaxRefresh 设置自动续期触发阈值
func (b *Builder) RenewMaxRefresh(seconds int64) *Builder {
	b.ensureConfig()
	b.cfg.RenewMaxRefresh = seconds
	return b
}

// RenewInterval sets minimum renew interval RenewInterval 设置最小续期间隔
func (b *Builder) RenewInterval(seconds int64) *Builder {
	b.ensureConfig()
	b.cfg.RenewInterval = seconds
	return b
}

// ActiveTimeout sets max inactive duration ActiveTimeout 设置最大不活跃时长
func (b *Builder) ActiveTimeout(seconds int64) *Builder {
	b.ensureConfig()
	b.cfg.ActiveTimeout = seconds
	return b
}

// ConcurrencyScope sets concurrency scope ConcurrencyScope 设置并发控制作用域
func (b *Builder) ConcurrencyScope(concurrencyScope config.ConcurrencyScope) *Builder {
	b.ensureConfig()
	b.cfg.ConcurrencyScope = concurrencyScope
	return b
}

// IsConcurrent sets concurrent login switch IsConcurrent 设置是否允许并发登录
func (b *Builder) IsConcurrent(concurrent bool) *Builder {
	b.ensureConfig()
	b.cfg.IsConcurrent = concurrent
	return b
}

// IsShare sets shared token switch IsShare 设置是否共享 Token
func (b *Builder) IsShare(share bool) *Builder {
	b.ensureConfig()
	b.cfg.IsShare = share
	return b
}

// MaxLoginCount sets max login count MaxLoginCount 设置最大登录数量
func (b *Builder) MaxLoginCount(count int64) *Builder {
	b.ensureConfig()
	b.cfg.MaxLoginCount = count
	return b
}

// ReplacedLoginExitMode sets non-concurrent login strategy ReplacedLoginExitMode 设置非并发登录处理策略
func (b *Builder) ReplacedLoginExitMode(mode config.ReplacedLoginExitMode) *Builder {
	b.ensureConfig()
	b.cfg.ReplacedLoginExitMode = mode
	return b
}

// OverflowLogoutMode sets overflow logout mode OverflowLogoutMode 设置超限登录下线模式
func (b *Builder) OverflowLogoutMode(mode config.LogoutMode) *Builder {
	b.ensureConfig()
	b.cfg.OverflowLogoutMode = mode
	return b
}

// IsReadBody sets body read switch IsReadBody 设置是否从请求体读取 Token
func (b *Builder) IsReadBody(isRead bool) *Builder {
	b.ensureConfig()
	b.cfg.IsReadBody = isRead
	return b
}

// IsReadHeader sets header read switch IsReadHeader 设置是否从 Header 读取 Token
func (b *Builder) IsReadHeader(isRead bool) *Builder {
	b.ensureConfig()
	b.cfg.IsReadHeader = isRead
	return b
}

// IsReadCookie sets cookie read switch IsReadCookie 设置是否从 Cookie 读取 Token
func (b *Builder) IsReadCookie(isRead bool) *Builder {
	b.ensureConfig()
	b.cfg.IsReadCookie = isRead
	return b
}

// TokenStyle sets token style TokenStyle 设置 Token 生成风格
func (b *Builder) TokenStyle(style adapter.TokenStyle) *Builder {
	b.ensureConfig()
	b.cfg.TokenStyle = style
	return b
}

// JwtSecretKey sets JWT secret key JwtSecretKey 设置 JWT 密钥
func (b *Builder) JwtSecretKey(key string) *Builder {
	b.ensureConfig()
	b.cfg.JwtSecretKey = key
	return b
}

// IsLog sets log switch IsLog 设置是否开启日志
func (b *Builder) IsLog(isLog bool) *Builder {
	b.ensureConfig()
	b.cfg.IsLog = isLog
	return b
}

// IsPrintBanner sets banner print switch IsPrintBanner 设置是否打印 Banner
func (b *Builder) IsPrintBanner(isPrint bool) *Builder {
	b.ensureConfig()
	b.cfg.IsPrintBanner = isPrint
	return b
}

// AsyncEvent sets async event switch AsyncEvent 设置是否异步触发事件
func (b *Builder) AsyncEvent(asyncEvent bool) *Builder {
	b.ensureConfig()
	b.cfg.AsyncEvent = asyncEvent
	return b
}

// CookieDomain sets cookie domain CookieDomain 设置 Cookie 作用域
func (b *Builder) CookieDomain(domain string) *Builder {
	cc := b.ensureCookieConfig()
	cc.Domain = domain
	return b
}

// CookiePath sets cookie path CookiePath 设置 Cookie 路径
func (b *Builder) CookiePath(path string) *Builder {
	cc := b.ensureCookieConfig()
	cc.Path = path
	return b
}

// CookieSecure sets cookie secure switch CookieSecure 设置 Cookie 是否仅通过 HTTPS 传输
func (b *Builder) CookieSecure(secure bool) *Builder {
	cc := b.ensureCookieConfig()
	cc.Secure = secure
	return b
}

// CookieHttpOnly sets cookie httpOnly switch CookieHttpOnly 设置 Cookie 是否禁止 JavaScript 访问
func (b *Builder) CookieHttpOnly(httpOnly bool) *Builder {
	cc := b.ensureCookieConfig()
	cc.HttpOnly = httpOnly
	return b
}

// CookieSameSite sets cookie sameSite mode CookieSameSite 设置 Cookie SameSite 模式
func (b *Builder) CookieSameSite(sameSite config.SameSiteMode) *Builder {
	cc := b.ensureCookieConfig()
	cc.SameSite = sameSite
	return b
}

// CookieMaxAge sets cookie max age seconds CookieMaxAge 设置 Cookie 最大存活秒数
func (b *Builder) CookieMaxAge(maxAge int64) *Builder {
	cc := b.ensureCookieConfig()
	cc.MaxAge = maxAge
	return b
}

// CookieConfig sets full cookie config CookieConfig 设置完整 Cookie 配置
func (b *Builder) CookieConfig(cfg *config.CookieConfig) *Builder {
	b.ensureConfig()
	if cfg == nil {
		b.cfg.CookieConfig = nil
		return b
	}
	copyCfg := *cfg
	b.cfg.CookieConfig = &copyCfg
	return b
}

// SetGenerator sets token generator SetGenerator 设置 Token 生成器
func (b *Builder) SetGenerator(generator adapter.Generator) *Builder {
	b.components.Generator = generator
	return b
}

// SetStorage sets storage adapter SetStorage 设置存储适配器
func (b *Builder) SetStorage(storage adapter.Storage) *Builder {
	b.components.Storage = storage
	return b
}

// SetCodec sets codec adapter SetCodec 设置编解码适配器
func (b *Builder) SetCodec(codec adapter.Codec) *Builder {
	b.components.Codec = codec
	return b
}

// SetLog sets log adapter SetLog 设置日志适配器
func (b *Builder) SetLog(log adapter.Log) *Builder {
	b.components.Log = log
	return b
}

// SetPool sets async task pool SetPool 设置异步任务池
func (b *Builder) SetPool(pool adapter.Pool) *Builder {
	b.components.Pool = pool
	return b
}

// SetNonceManager sets optional nonce manager SetNonceManager 设置可选 Nonce 管理器
func (b *Builder) SetNonceManager(nonceManager *nonce.NonceManager) *Builder {
	return b.UseManagerOption(manager.WithNonceManager(nonceManager))
}

// SetOAuth2Manager sets optional OAuth2 manager SetOAuth2Manager 设置可选 OAuth2 管理器
func (b *Builder) SetOAuth2Manager(oauth2Manager *oauth2.OAuth2Server) *Builder {
	return b.UseManagerOption(manager.WithOAuth2Manager(oauth2Manager))
}

// UseManagerOption appends optional manager assembly option UseManagerOption 添加 Manager 可选模块装配项
func (b *Builder) UseManagerOption(option manager.Option) *Builder {
	if option != nil {
		b.managerOptions = append(b.managerOptions, option)
	}
	return b
}

// SetGeneratorFactory sets default generator factory SetGeneratorFactory 设置默认生成器工厂
func (b *Builder) SetGeneratorFactory(factory GeneratorFactory) *Builder {
	b.factories.Generator = factory
	return b
}

// SetStorageFactory sets default storage factory SetStorageFactory 设置默认存储工厂
func (b *Builder) SetStorageFactory(factory StorageFactory) *Builder {
	b.factories.Storage = factory
	return b
}

// SetCodecFactory sets default codec factory SetCodecFactory 设置默认编解码工厂
func (b *Builder) SetCodecFactory(factory CodecFactory) *Builder {
	b.factories.Codec = factory
	return b
}

// SetLogFactory sets default log factory SetLogFactory 设置默认日志工厂
func (b *Builder) SetLogFactory(factory LogFactory) *Builder {
	b.factories.Log = factory
	return b
}

// SetPoolFactory sets default pool factory SetPoolFactory 设置默认任务池工厂
func (b *Builder) SetPoolFactory(factory PoolFactory) *Builder {
	b.factories.Pool = factory
	return b
}

// SetAccessProvider sets the permission and role provider SetAccessProvider 设置权限与角色提供器
func (b *Builder) SetAccessProvider(provider manager.AccessProvider) *Builder {
	b.accessProvider = provider
	return b
}

// SetCustomPermissionListFunc sets permission callback SetCustomPermissionListFunc 设置权限回调
func (b *Builder) SetCustomPermissionListFunc(f func(loginID, authType string) ([]string, error)) *Builder {
	b.customPermissionListFunc = f
	return b
}

// SetCustomRoleListFunc sets role callback SetCustomRoleListFunc 设置角色回调
func (b *Builder) SetCustomRoleListFunc(f func(loginID, authType string) ([]string, error)) *Builder {
	b.customRoleListFunc = f
	return b
}

// SetCustomPermissionListExtFunc sets extended permission callback SetCustomPermissionListExtFunc 设置扩展权限回调
func (b *Builder) SetCustomPermissionListExtFunc(f func(loginID, device, deviceId, authType string) ([]string, error)) *Builder {
	b.customPermissionListExtFunc = f
	return b
}

// SetCustomRoleListExtFunc sets extended role callback SetCustomRoleListExtFunc 设置扩展角色回调
func (b *Builder) SetCustomRoleListExtFunc(f func(loginID, device, deviceId, authType string) ([]string, error)) *Builder {
	b.customRoleListExtFunc = f
	return b
}

// JwtSecret enables JWT style and sets secret JwtSecret 启用 JWT 风格并设置密钥
func (b *Builder) JwtSecret(secret string) *Builder {
	b.ensureConfig()
	b.cfg.TokenStyle = adapter.TokenStyleJWT
	b.cfg.JwtSecretKey = secret
	return b
}

// Clone clones builder with deep copy Clone 深拷贝 Builder
func (b *Builder) Clone() *Builder {
	clone := *b
	if b.cfg != nil {
		clone.cfg = b.cfg.Clone()
	}
	if len(b.managerOptions) > 0 {
		clone.managerOptions = append([]manager.Option(nil), b.managerOptions...)
	}
	return &clone
}

// Build builds manager and returns configuration errors Build 构建 Manager 并返回配置错误
func (b *Builder) Build() (*manager.Manager, error) {
	b.ensureConfig()
	cfg := b.cfg.Clone()

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("构建 Manager 失败，配置无效：%w", err)
	}

	// Resolve components per build so later config changes do not reuse stale factory products 每次构建独立装配组件，避免后续配置变化继续复用旧工厂产物
	components := b.components

	if components.Generator == nil {
		if b.factories.Generator != nil {
			generator, err := b.factories.Generator(cfg)
			if err != nil {
				return nil, fmt.Errorf("构建 Manager 失败，创建 Token 生成器失败：%w", err)
			}
			components.Generator = generator
		}
		if components.Generator == nil {
			return nil, fmt.Errorf("构建 Manager 失败，缺少 Token 生成器，请调用 SetGenerator 或 SetGeneratorFactory")
		}
	}

	if components.Storage == nil {
		if b.factories.Storage != nil {
			storage, err := b.factories.Storage(cfg)
			if err != nil {
				return nil, fmt.Errorf("构建 Manager 失败，创建存储适配器失败：%w", err)
			}
			components.Storage = storage
		}
		if components.Storage == nil {
			return nil, fmt.Errorf("构建 Manager 失败，缺少存储适配器，请调用 SetStorage 或 SetStorageFactory")
		}
	}
	if components.Codec == nil {
		if b.factories.Codec != nil {
			codec, err := b.factories.Codec(cfg)
			if err != nil {
				return nil, fmt.Errorf("构建 Manager 失败，创建编解码适配器失败：%w", err)
			}
			components.Codec = codec
		}
		if components.Codec == nil {
			return nil, fmt.Errorf("构建 Manager 失败，缺少编解码适配器，请调用 SetCodec 或 SetCodecFactory")
		}
	}

	if cfg.IsLog {
		if components.Log == nil {
			if b.factories.Log != nil {
				logger, err := b.factories.Log(cfg)
				if err != nil {
					return nil, fmt.Errorf("构建 Manager 失败，创建日志适配器失败：%w", err)
				}
				components.Log = logger
			}
			if components.Log == nil {
				return nil, fmt.Errorf("构建 Manager 失败，缺少日志适配器，请调用 SetLog 或 SetLogFactory")
			}
		}
	} else {
		components.Log = adapter.NewNopLogger()
	}

	if cfg.AutoRenew && components.Pool == nil && b.factories.Pool != nil {
		pool, err := b.factories.Pool(cfg)
		if err != nil {
			return nil, fmt.Errorf("构建 Manager 失败，创建续期任务池失败：%w", err)
		}
		components.Pool = pool
	}

	if cfg.IsPrintBanner {
		banner.PrintBanner(cfg)
	}

	accessProvider := b.accessProvider
	if accessProvider == nil {
		accessProvider = manager.NewLegacyAccessProvider(
			b.customPermissionListFunc,
			b.customRoleListFunc,
			b.customPermissionListExtFunc,
			b.customRoleListExtFunc,
		)
	}

	mgr := manager.NewManager(
		cfg,
		components.Generator,
		components.Storage,
		components.Codec,
		components.Log,
		components.Pool,
		accessProvider,
		b.managerOptions...,
	)

	return mgr, nil
}

// MustBuild builds manager and panics on error MustBuild 构建 Manager 并在失败时触发 panic
func (b *Builder) MustBuild() *manager.Manager {
	mgr, err := b.Build()
	if err != nil {
		panic(err)
	}
	return mgr
}

// ensureConfig initializes config when needed ensureConfig 在需要时初始化配置
func (b *Builder) ensureConfig() {
	if b.cfg == nil {
		b.cfg = config.DefaultConfig()
	}
}

// ensureCookieConfig initializes cookie config when needed ensureCookieConfig 在需要时初始化 Cookie 配置
func (b *Builder) ensureCookieConfig() *config.CookieConfig {
	b.ensureConfig()
	if b.cfg.CookieConfig == nil {
		b.cfg.CookieConfig = config.DefaultCookieConfig()
	}
	return b.cfg.CookieConfig
}

// durationToSeconds rounds positive durations up to whole seconds durationToSeconds 将正时长向上取整到整秒
func durationToSeconds(d time.Duration) int64 {
	if d <= 0 {
		return int64(d.Seconds())
	}

	seconds := int64(d / time.Second)
	if d%time.Second != 0 {
		seconds++
	}
	return seconds
}
