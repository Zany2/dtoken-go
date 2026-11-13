// @Author daixk 2026/05/15
package dtoken

import (
	"fmt"
	"time"

	"github.com/Zany2/dtoken-go/config"
	adapter "github.com/Zany2/dtoken-go/contract"
	"github.com/Zany2/dtoken-go/internal/builder"
	"github.com/Zany2/dtoken-go/internal/defaults"
	manager "github.com/Zany2/dtoken-go/internal/engine"
	"github.com/Zany2/dtoken-go/logging/dlog"
	"github.com/Zany2/dtoken-go/pool/ants"
)

// Builder constructs an Auth instance with explicit component ownership. Builder 按显式组件所有权构建 Auth 实例。
type Builder struct {
	inner *builder.Builder // inner stores core builder inner 存储底层构建器

	renewPoolConfig *ants.RenewPoolConfig // renewPoolConfig stores renew pool config renewPoolConfig 存储续期池配置
	loggerConfig    *dlog.LoggerConfig    // loggerConfig stores logger config loggerConfig 存储日志配置

	customLogFactory  bool // customLogFactory marks user logger factory customLogFactory 标记用户自定义日志工厂
	customPoolFactory bool // customPoolFactory marks user pool factory customPoolFactory 标记用户自定义续期池工厂

	managerOptions []manager.Option // managerOptions stores delayed user options managerOptions 存储延迟执行的用户装配选项
}

// NewBuilder creates a high-level builder with bundled default configs NewBuilder 创建高层默认配置构建器
func NewBuilder() *Builder {
	return &Builder{
		inner:           defaults.NewBuilder(),
		renewPoolConfig: ants.DefaultRenewPoolConfig(),
		loggerConfig:    dlog.DefaultLoggerConfig(),
	}
}

// Config replaces core config Config 替换核心配置
func (b *Builder) Config(cfg *config.Config) *Builder {
	b.ensureBuilder()
	b.inner.Config(cfg)
	return b
}

// GetConfig returns mutable core config GetConfig 返回可变核心配置
func (b *Builder) GetConfig() *config.Config {
	b.ensureBuilder()
	return b.inner.GetConfig()
}

// AuthType sets auth type AuthType 设置认证体系类型
func (b *Builder) AuthType(authType string) *Builder {
	b.ensureBuilder()
	b.inner.AuthType(authType)
	return b
}

// KeyPrefix sets key prefix KeyPrefix 设置存储键前缀
func (b *Builder) KeyPrefix(keyPrefix string) *Builder {
	b.ensureBuilder()
	b.inner.KeyPrefix(keyPrefix)
	return b
}

// TokenName sets token name TokenName 设置 Token 名称
func (b *Builder) TokenName(name string) *Builder {
	b.ensureBuilder()
	b.inner.TokenName(name)
	return b
}

// Timeout sets token timeout seconds Timeout 设置 Token 超时时间秒数
func (b *Builder) Timeout(seconds int64) *Builder {
	b.ensureBuilder()
	b.inner.Timeout(seconds)
	return b
}

// RefreshTokenTimeout sets refresh token timeout seconds RefreshTokenTimeout 设置刷新令牌超时时间秒数
func (b *Builder) RefreshTokenTimeout(seconds int64) *Builder {
	b.ensureBuilder()
	b.inner.RefreshTokenTimeout(seconds)
	return b
}

// TimeoutDuration sets token timeout by duration TimeoutDuration 使用时长设置 Token 超时时间
func (b *Builder) TimeoutDuration(d time.Duration) *Builder {
	b.ensureBuilder()
	b.inner.TimeoutDuration(d)
	return b
}

// RefreshTokenTimeoutDuration sets refresh token timeout by duration RefreshTokenTimeoutDuration 使用时长设置刷新令牌超时时间
func (b *Builder) RefreshTokenTimeoutDuration(d time.Duration) *Builder {
	b.ensureBuilder()
	b.inner.RefreshTokenTimeoutDuration(d)
	return b
}

// AutoRenew sets auto renew switch AutoRenew 设置自动续期开关
func (b *Builder) AutoRenew(autoRenew bool) *Builder {
	b.ensureBuilder()
	b.inner.AutoRenew(autoRenew)
	return b
}

// RenewMaxRefresh sets renew trigger threshold RenewMaxRefresh 设置续期触发阈值秒数
func (b *Builder) RenewMaxRefresh(seconds int64) *Builder {
	b.ensureBuilder()
	b.inner.RenewMaxRefresh(seconds)
	return b
}

// RenewInterval sets minimum renew interval RenewInterval 设置最小续期间隔秒数
func (b *Builder) RenewInterval(seconds int64) *Builder {
	b.ensureBuilder()
	b.inner.RenewInterval(seconds)
	return b
}

// ActiveTimeout sets max inactive duration ActiveTimeout 设置最大不活跃时长秒数
func (b *Builder) ActiveTimeout(seconds int64) *Builder {
	b.ensureBuilder()
	b.inner.ActiveTimeout(seconds)
	return b
}

// ConcurrencyScope sets concurrency scope ConcurrencyScope 设置并发控制作用域
func (b *Builder) ConcurrencyScope(concurrencyScope config.ConcurrencyScope) *Builder {
	b.ensureBuilder()
	b.inner.ConcurrencyScope(concurrencyScope)
	return b
}

// IsConcurrent sets concurrent login switch IsConcurrent 设置是否允许并发登录
func (b *Builder) IsConcurrent(concurrent bool) *Builder {
	b.ensureBuilder()
	b.inner.IsConcurrent(concurrent)
	return b
}

// IsShare sets shared token switch IsShare 设置是否共享 Token
func (b *Builder) IsShare(share bool) *Builder {
	b.ensureBuilder()
	b.inner.IsShare(share)
	return b
}

// MaxLoginCount sets max login count MaxLoginCount 设置最大登录数量
func (b *Builder) MaxLoginCount(count int64) *Builder {
	b.ensureBuilder()
	b.inner.MaxLoginCount(count)
	return b
}

// ReplacedLoginExitMode sets replaced login strategy ReplacedLoginExitMode 设置非并发登录处理策略
func (b *Builder) ReplacedLoginExitMode(mode config.ReplacedLoginExitMode) *Builder {
	b.ensureBuilder()
	b.inner.ReplacedLoginExitMode(mode)
	return b
}

// OverflowLogoutMode sets overflow logout mode OverflowLogoutMode 设置超限登录下线模式
func (b *Builder) OverflowLogoutMode(mode config.LogoutMode) *Builder {
	b.ensureBuilder()
	b.inner.OverflowLogoutMode(mode)
	return b
}

// IsReadBody sets body read switch IsReadBody 设置是否从请求体读取 Token
func (b *Builder) IsReadBody(isRead bool) *Builder {
	b.ensureBuilder()
	b.inner.IsReadBody(isRead)
	return b
}

// IsReadQuery sets query read switch IsReadQuery 设置是否从 Query 读取 Token
func (b *Builder) IsReadQuery(isRead bool) *Builder {
	b.ensureBuilder()
	b.inner.IsReadQuery(isRead)
	return b
}

// IsReadHeader sets header read switch IsReadHeader 设置是否从 Header 读取 Token
func (b *Builder) IsReadHeader(isRead bool) *Builder {
	b.ensureBuilder()
	b.inner.IsReadHeader(isRead)
	return b
}

// IsReadCookie sets cookie read switch IsReadCookie 设置是否从 Cookie 读取 Token
func (b *Builder) IsReadCookie(isRead bool) *Builder {
	b.ensureBuilder()
	b.inner.IsReadCookie(isRead)
	return b
}

// TokenStyle sets token style TokenStyle 设置 Token 生成风格
func (b *Builder) TokenStyle(style adapter.TokenStyle) *Builder {
	b.ensureBuilder()
	b.inner.TokenStyle(style)
	return b
}

// JwtSecretKey sets JWT secret key JwtSecretKey 设置 JWT 密钥
func (b *Builder) JwtSecretKey(key string) *Builder {
	b.ensureBuilder()
	b.inner.JwtSecretKey(key)
	return b
}

// IsLog sets log switch IsLog 设置是否开启日志
func (b *Builder) IsLog(isLog bool) *Builder {
	b.ensureBuilder()
	b.inner.IsLog(isLog)
	return b
}

// IsPrintBanner sets banner print switch IsPrintBanner 设置是否打印 Banner
func (b *Builder) IsPrintBanner(isPrint bool) *Builder {
	b.ensureBuilder()
	b.inner.IsPrintBanner(isPrint)
	return b
}

// AsyncEvent sets async event switch AsyncEvent 设置是否异步触发事件
func (b *Builder) AsyncEvent(asyncEvent bool) *Builder {
	b.ensureBuilder()
	b.inner.AsyncEvent(asyncEvent)
	return b
}

// CookieDomain sets cookie domain CookieDomain 设置 Cookie 作用域
func (b *Builder) CookieDomain(domain string) *Builder {
	b.ensureBuilder()
	b.inner.CookieDomain(domain)
	return b
}

// CookiePath sets cookie path CookiePath 设置 Cookie 路径
func (b *Builder) CookiePath(path string) *Builder {
	b.ensureBuilder()
	b.inner.CookiePath(path)
	return b
}

// CookieSecure sets cookie secure switch CookieSecure 设置 Cookie HTTPS 限制
func (b *Builder) CookieSecure(secure bool) *Builder {
	b.ensureBuilder()
	b.inner.CookieSecure(secure)
	return b
}

// CookieHttpOnly sets cookie httpOnly switch CookieHttpOnly 设置 Cookie HttpOnly
func (b *Builder) CookieHttpOnly(httpOnly bool) *Builder {
	b.ensureBuilder()
	b.inner.CookieHttpOnly(httpOnly)
	return b
}

// CookieSameSite sets cookie sameSite mode CookieSameSite 设置 Cookie SameSite 模式
func (b *Builder) CookieSameSite(sameSite config.SameSiteMode) *Builder {
	b.ensureBuilder()
	b.inner.CookieSameSite(sameSite)
	return b
}

// CookieMaxAge sets cookie max age seconds CookieMaxAge 设置 Cookie 最大存活秒数
func (b *Builder) CookieMaxAge(maxAge int64) *Builder {
	b.ensureBuilder()
	b.inner.CookieMaxAge(maxAge)
	return b
}

// CookieConfig sets full cookie config CookieConfig 设置完整 Cookie 配置
func (b *Builder) CookieConfig(cfg *config.CookieConfig) *Builder {
	b.ensureBuilder()
	b.inner.CookieConfig(cfg)
	return b
}

// RenewPoolConfig replaces renew pool config RenewPoolConfig 替换续期池配置
func (b *Builder) RenewPoolConfig(cfg *ants.RenewPoolConfig) *Builder {
	b.ensureBuilder()
	if cfg == nil {
		b.renewPoolConfig = ants.DefaultRenewPoolConfig()
		return b
	}
	b.renewPoolConfig = cfg.Clone()
	return b
}

// GetRenewPoolConfig returns mutable renew pool config GetRenewPoolConfig 返回可变续期池配置
func (b *Builder) GetRenewPoolConfig() *ants.RenewPoolConfig {
	b.ensureBuilder()
	return b.renewPoolConfig
}

// RenewPoolMinSize sets renew pool minimum size RenewPoolMinSize 设置续期池最小协程数
func (b *Builder) RenewPoolMinSize(size int) *Builder {
	b.ensureBuilder()
	b.renewPoolConfig.MinSize = size
	return b
}

// RenewPoolMaxSize sets renew pool maximum size RenewPoolMaxSize 设置续期池最大协程数
func (b *Builder) RenewPoolMaxSize(size int) *Builder {
	b.ensureBuilder()
	b.renewPoolConfig.MaxSize = size
	return b
}

// RenewPoolScaleUpRate sets renew pool scale up rate RenewPoolScaleUpRate 设置续期池扩容阈值
func (b *Builder) RenewPoolScaleUpRate(rate float64) *Builder {
	b.ensureBuilder()
	b.renewPoolConfig.ScaleUpRate = rate
	return b
}

// RenewPoolScaleDownRate sets renew pool scale down rate RenewPoolScaleDownRate 设置续期池缩容阈值
func (b *Builder) RenewPoolScaleDownRate(rate float64) *Builder {
	b.ensureBuilder()
	b.renewPoolConfig.ScaleDownRate = rate
	return b
}

// RenewPoolCheckInterval sets renew pool check interval RenewPoolCheckInterval 设置续期池检查间隔
func (b *Builder) RenewPoolCheckInterval(interval time.Duration) *Builder {
	b.ensureBuilder()
	b.renewPoolConfig.CheckInterval = interval
	return b
}

// RenewPoolExpiry sets renew pool worker expiry RenewPoolExpiry 设置续期池空闲过期时间
func (b *Builder) RenewPoolExpiry(expiry time.Duration) *Builder {
	b.ensureBuilder()
	b.renewPoolConfig.Expiry = expiry
	return b
}

// RenewPoolPrintStatusInterval sets renew pool status interval RenewPoolPrintStatusInterval 设置续期池状态打印间隔
func (b *Builder) RenewPoolPrintStatusInterval(interval time.Duration) *Builder {
	b.ensureBuilder()
	b.renewPoolConfig.PrintStatusInterval = interval
	return b
}

// RenewPoolPreAlloc sets renew pool pre allocation RenewPoolPreAlloc 设置续期池是否预分配
func (b *Builder) RenewPoolPreAlloc(preAlloc bool) *Builder {
	b.ensureBuilder()
	b.renewPoolConfig.PreAlloc = preAlloc
	return b
}

// RenewPoolNonBlocking sets renew pool non-blocking mode RenewPoolNonBlocking 设置续期池非阻塞模式
func (b *Builder) RenewPoolNonBlocking(nonBlocking bool) *Builder {
	b.ensureBuilder()
	b.renewPoolConfig.NonBlocking = nonBlocking
	return b
}

// LoggerConfig replaces logger config LoggerConfig 替换日志配置
func (b *Builder) LoggerConfig(cfg *dlog.LoggerConfig) *Builder {
	b.ensureBuilder()
	if cfg == nil {
		b.loggerConfig = dlog.DefaultLoggerConfig()
		return b
	}
	b.loggerConfig = cfg.Clone()
	return b
}

// GetLoggerConfig returns mutable logger config GetLoggerConfig 返回可变日志配置
func (b *Builder) GetLoggerConfig() *dlog.LoggerConfig {
	b.ensureBuilder()
	return b.loggerConfig
}

// LoggerPath sets log output path LoggerPath 设置日志输出目录
func (b *Builder) LoggerPath(path string) *Builder {
	b.ensureBuilder()
	b.loggerConfig.Path = path
	return b
}

// LoggerFileFormat sets log file format LoggerFileFormat 设置日志文件格式
func (b *Builder) LoggerFileFormat(format string) *Builder {
	b.ensureBuilder()
	b.loggerConfig.FileFormat = format
	return b
}

// LoggerPrefix sets log prefix LoggerPrefix 设置日志前缀
func (b *Builder) LoggerPrefix(prefix string) *Builder {
	b.ensureBuilder()
	b.loggerConfig.Prefix = prefix
	return b
}

// LoggerLevel sets log level LoggerLevel 设置日志级别
func (b *Builder) LoggerLevel(level dlog.LogLevel) *Builder {
	b.ensureBuilder()
	b.loggerConfig.Level = level
	return b
}

// LoggerTimeFormat sets log time format LoggerTimeFormat 设置日志时间格式
func (b *Builder) LoggerTimeFormat(format string) *Builder {
	b.ensureBuilder()
	b.loggerConfig.TimeFormat = format
	return b
}

// LoggerStdout sets stdout output LoggerStdout 设置是否输出到控制台
func (b *Builder) LoggerStdout(stdout bool) *Builder {
	b.ensureBuilder()
	b.loggerConfig.Stdout = stdout
	return b
}

// LoggerStdoutOnly sets stdout-only mode LoggerStdoutOnly 设置仅控制台输出模式
func (b *Builder) LoggerStdoutOnly(stdoutOnly bool) *Builder {
	b.ensureBuilder()
	b.loggerConfig.StdoutOnly = stdoutOnly
	if stdoutOnly {
		b.loggerConfig.Stdout = true
	}
	return b
}

// LoggerQueueSize sets async log queue size LoggerQueueSize 设置异步日志队列大小
func (b *Builder) LoggerQueueSize(size int) *Builder {
	b.ensureBuilder()
	b.loggerConfig.QueueSize = size
	return b
}

// LoggerRotateSize sets log rotate size LoggerRotateSize 设置日志滚动大小
func (b *Builder) LoggerRotateSize(size int64) *Builder {
	b.ensureBuilder()
	b.loggerConfig.RotateSize = size
	return b
}

// LoggerRotateExpire sets log rotate interval LoggerRotateExpire 设置日志滚动间隔
func (b *Builder) LoggerRotateExpire(expire time.Duration) *Builder {
	b.ensureBuilder()
	b.loggerConfig.RotateExpire = expire
	return b
}

// LoggerRotateBackupLimit sets log backup count LoggerRotateBackupLimit 设置日志备份数量
func (b *Builder) LoggerRotateBackupLimit(limit int) *Builder {
	b.ensureBuilder()
	b.loggerConfig.RotateBackupLimit = limit
	return b
}

// LoggerRotateBackupDays sets log retention days LoggerRotateBackupDays 设置日志保留天数
func (b *Builder) LoggerRotateBackupDays(days int) *Builder {
	b.ensureBuilder()
	b.loggerConfig.RotateBackupDays = days
	return b
}

// SetGenerator sets token generator SetGenerator 设置 Token 生成器
func (b *Builder) SetGenerator(generator adapter.Generator) *Builder {
	b.ensureBuilder()
	b.inner.SetGenerator(generator)
	return b
}

// SetStorage sets a manager-owned storage adapter by default. SetStorage 设置默认由 Manager 持有的存储适配器。
func (b *Builder) Storage(storage adapter.Storage) *Builder {
	b.ensureBuilder()
	b.inner.Storage(storage)
	return b
}

// SetCodec sets codec adapter SetCodec 设置编解码器
func (b *Builder) SetCodec(codec adapter.Codec) *Builder {
	b.ensureBuilder()
	b.inner.SetCodec(codec)
	return b
}

// SetLog sets a manager-owned log adapter by default. SetLog 设置默认由 Manager 持有的日志适配器。
func (b *Builder) SetLog(log adapter.Log) *Builder {
	b.ensureBuilder()
	b.inner.SetLog(log)
	return b
}

// SetPool sets a manager-owned async task pool by default. SetPool 设置默认由 Manager 持有的异步任务池。
func (b *Builder) SetPool(pool adapter.Pool) *Builder {
	b.ensureBuilder()
	b.inner.SetPool(pool)
	return b
}

// UseManagerOption appends manager option UseManagerOption 添加 Manager 装配选项
func (b *Builder) UseManagerOption(option ManagerOption) *Builder {
	b.ensureBuilder()
	if option != nil {
		b.managerOptions = append(b.managerOptions, option)
	}
	return b
}

// SetGeneratorFactory sets default generator factory SetGeneratorFactory 设置默认生成器工厂
func (b *Builder) SetGeneratorFactory(factory GeneratorFactory) *Builder {
	b.ensureBuilder()
	b.inner.SetGeneratorFactory(factory)
	return b
}

// SetStorageFactory sets a factory for storage adapters owned by Manager by default. SetStorageFactory 设置创建默认由 Manager 持有的存储适配器的工厂。
func (b *Builder) SetStorageFactory(factory StorageFactory) *Builder {
	b.ensureBuilder()
	b.inner.SetStorageFactory(factory)
	return b
}

// SetCodecFactory sets default codec factory SetCodecFactory 设置默认编解码器工厂
func (b *Builder) SetCodecFactory(factory CodecFactory) *Builder {
	b.ensureBuilder()
	b.inner.SetCodecFactory(factory)
	return b
}

// SetLogFactory sets a factory for log adapters owned by Manager by default. SetLogFactory 设置创建默认由 Manager 持有的日志适配器的工厂。
func (b *Builder) SetLogFactory(factory LogFactory) *Builder {
	b.ensureBuilder()
	b.customLogFactory = factory != nil
	b.inner.SetLogFactory(factory)
	return b
}

// SetPoolFactory sets a factory for task pools owned by Manager by default. SetPoolFactory 设置创建默认由 Manager 持有的任务池的工厂。
func (b *Builder) SetPoolFactory(factory PoolFactory) *Builder {
	b.ensureBuilder()
	b.customPoolFactory = factory != nil
	b.inner.SetPoolFactory(factory)
	return b
}

// SetAccessProvider sets permission and role provider SetAccessProvider 设置权限与角色提供器
func (b *Builder) SetAccessProvider(provider AccessProvider) *Builder {
	b.ensureBuilder()
	b.inner.SetAccessProvider(provider)
	return b
}

// JwtSecret enables JWT style and sets secret JwtSecret 启用 JWT 风格并设置密钥
func (b *Builder) JwtSecret(secret string) *Builder {
	b.ensureBuilder()
	b.inner.JwtSecret(secret)
	return b
}

// Clone clones builder with module configs Clone 克隆构建器和模块配置
func (b *Builder) Clone() *Builder {
	b.ensureBuilder()
	clone := *b
	clone.inner = b.inner.Clone()
	clone.renewPoolConfig = b.renewPoolConfig.Clone()
	clone.loggerConfig = b.loggerConfig.Clone()

	if len(b.managerOptions) > 0 {
		clone.managerOptions = append([]manager.Option(nil), b.managerOptions...)
	}
	return &clone
}

// Build validates module configs and builds manager Build 校验模块配置并构建 Manager
func (b *Builder) buildManager() (*manager.Manager, error) {
	b.ensureBuilder()

	// Clone configs so validation and assembly preserve reusable builder state. 克隆配置，避免校验和装配修改可复用的构建器状态。
	coreConfig := b.inner.GetConfig().Clone()
	renewPoolConfig := b.renewPoolConfig.Clone()
	loggerConfig := b.loggerConfig.Clone()

	// Validate every config before wiring components. 在装配组件前校验全部配置。
	if err := coreConfig.Validate(); err != nil {
		return nil, fmt.Errorf("build manager failed: invalid core config: %w", err)
	}
	if err := renewPoolConfig.Validate(); err != nil {
		return nil, fmt.Errorf("build manager failed: invalid renew pool config: %w", err)
	}
	if err := loggerConfig.Validate(); err != nil {
		return nil, fmt.Errorf("build manager failed: invalid logger config: %w", err)
	}

	// Wire authentication defaults before applying user overrides. 先装配认证默认组件，再应用用户覆盖项。
	coreBuilder := b.inner.Clone().Config(coreConfig)
	b.applyDefaultFactories(coreBuilder, renewPoolConfig, loggerConfig)

	b.applyUserManagerOptions(coreBuilder)

	return coreBuilder.Build()
}

// Build creates an authentication instance. Build 创建认证实例。
func (b *Builder) Build() (*Auth, error) {
	mgr, err := b.buildManager()
	if err != nil {
		return nil, err
	}
	return New(mgr), nil
}

// MustBuild builds manager and panics on error MustBuild 构建 Manager，失败时 panic
func (b *Builder) MustBuild() *Auth {
	mgr, err := b.Build()
	if err != nil {
		panic(err)
	}
	return mgr
}

// applyDefaultFactories wires config-aware default factories applyDefaultFactories 装配读取配置的默认工厂
func (b *Builder) applyDefaultFactories(coreBuilder *builder.Builder, renewPoolConfig *ants.RenewPoolConfig, loggerConfig *dlog.LoggerConfig) {
	if !b.customLogFactory {
		coreBuilder.SetLogFactory(func(_ *config.Config) (adapter.Log, error) {
			return dlog.NewLoggerWithConfig(loggerConfig.Clone())
		})
	}
	if !b.customPoolFactory {
		coreBuilder.SetPoolFactory(func(_ *config.Config) (adapter.Pool, error) {
			return ants.NewRenewPoolManagerWithConfig(renewPoolConfig.Clone())
		})
	}
}

// applyUserManagerOptions appends user assembly options. applyUserManagerOptions 追加用户装配选项。
func (b *Builder) applyUserManagerOptions(coreBuilder *builder.Builder) {
	for _, option := range b.managerOptions {
		coreBuilder.UseManagerOption(option)
	}
}

// ensureBuilder initializes missing internals ensureBuilder 初始化缺失的内部配置
func (b *Builder) ensureBuilder() {
	if b.inner == nil {
		b.inner = defaults.NewBuilder()
	}
	if b.renewPoolConfig == nil {
		b.renewPoolConfig = ants.DefaultRenewPoolConfig()
	}
	if b.loggerConfig == nil {
		b.loggerConfig = dlog.DefaultLoggerConfig()
	}
}
