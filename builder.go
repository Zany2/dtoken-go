// @Author daixk 2026/05/15
package dtoken

import (
	"fmt"
	"time"

	"github.com/Zany2/dtoken-go/config"
	adapter "github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
	"github.com/Zany2/dtoken-go/internal/builder"
	manager "github.com/Zany2/dtoken-go/internal/engine"
	"github.com/Zany2/dtoken-go/logging/dlog"
	"github.com/Zany2/dtoken-go/pool/ants"
)

// Builder constructs an Auth instance with explicit component ownership. Builder 按显式组件所有权构建 Auth 实例。
type Builder struct {
	cfg            *config.Config     // cfg stores mutable authentication settings. cfg 保存可修改的认证配置。
	components     Components         // components stores explicitly injected adapters. components 保存显式注入的适配器。
	factories      ComponentFactories // factories stores user factories; nil selects bundled defaults. factories 保存用户工厂，nil 使用内置默认实现。
	accessProvider AccessProvider     // accessProvider supplies permissions and roles. accessProvider 提供权限与角色。

	renewPoolConfig *ants.RenewPoolConfig // renewPoolConfig stores renew pool config renewPoolConfig 存储续期池配置
	loggerConfig    *dlog.LoggerConfig    // loggerConfig stores logger config loggerConfig 存储日志配置

	managerOptions []manager.Option // managerOptions stores delayed user options managerOptions 存储延迟执行的用户装配选项
}

// NewBuilder creates a high-level builder with bundled default configs NewBuilder 创建高层默认配置构建器
func NewBuilder() *Builder {
	return &Builder{
		cfg:             config.DefaultConfig(),
		renewPoolConfig: ants.DefaultRenewPoolConfig(),
		loggerConfig:    dlog.DefaultLoggerConfig(),
	}
}

// Config replaces core config Config 替换核心配置
func (b *Builder) Config(cfg *config.Config) *Builder {
	// Reset to default config when input is nil 输入为 nil 时重置为默认配置
	if cfg == nil {
		b.cfg = config.DefaultConfig()
		return b
	}

	// Store a cloned config to avoid external mutation 保存副本以避免外部修改影响 Builder
	b.cfg = cfg.Clone()
	return b
}

// GetConfig returns mutable core config GetConfig 返回可变核心配置
func (b *Builder) GetConfig() *config.Config {
	b.ensureBuilder()
	return b.cfg
}

// AuthType sets auth type AuthType 设置认证体系类型
func (b *Builder) AuthType(authType string) *Builder {
	b.ensureBuilder()
	b.cfg.SetAuthType(authType)
	return b
}

// KeyPrefix sets key prefix KeyPrefix 设置存储键前缀
func (b *Builder) KeyPrefix(keyPrefix string) *Builder {
	b.ensureBuilder()
	b.cfg.SetKeyPrefix(keyPrefix)
	return b
}

// TokenName sets token name TokenName 设置 Token 名称
func (b *Builder) TokenName(name string) *Builder {
	b.ensureBuilder()
	b.cfg.TokenName = name
	return b
}

// Timeout sets token timeout seconds Timeout 设置 Token 超时时间秒数
func (b *Builder) Timeout(seconds int64) *Builder {
	b.ensureBuilder()
	b.cfg.Timeout = seconds
	return b
}

// RefreshTokenTimeout sets refresh token timeout seconds RefreshTokenTimeout 设置刷新令牌超时时间秒数
func (b *Builder) RefreshTokenTimeout(seconds int64) *Builder {
	b.ensureBuilder()
	b.cfg.RefreshTokenTimeout = seconds
	return b
}

// TimeoutDuration sets token timeout by duration TimeoutDuration 使用时长设置 Token 超时时间
func (b *Builder) TimeoutDuration(d time.Duration) *Builder {
	b.ensureBuilder()
	b.cfg.Timeout = durationToSeconds(d)
	return b
}

// RefreshTokenTimeoutDuration sets refresh token timeout by duration RefreshTokenTimeoutDuration 使用时长设置刷新令牌超时时间
func (b *Builder) RefreshTokenTimeoutDuration(d time.Duration) *Builder {
	b.ensureBuilder()
	b.cfg.RefreshTokenTimeout = durationToSeconds(d)
	return b
}

// AutoRenew sets auto renew switch AutoRenew 设置自动续期开关
func (b *Builder) AutoRenew(autoRenew bool) *Builder {
	b.ensureBuilder()
	b.cfg.AutoRenew = autoRenew
	return b
}

// RenewMaxRefresh sets renew trigger threshold RenewMaxRefresh 设置续期触发阈值秒数
func (b *Builder) RenewMaxRefresh(seconds int64) *Builder {
	b.ensureBuilder()
	b.cfg.RenewMaxRefresh = seconds
	return b
}

// RenewInterval sets minimum renew interval RenewInterval 设置最小续期间隔秒数
func (b *Builder) RenewInterval(seconds int64) *Builder {
	b.ensureBuilder()
	b.cfg.RenewInterval = seconds
	return b
}

// ActiveTimeout sets max inactive duration ActiveTimeout 设置最大不活跃时长秒数
func (b *Builder) ActiveTimeout(seconds int64) *Builder {
	b.ensureBuilder()
	b.cfg.ActiveTimeout = seconds
	return b
}

// ConcurrencyScope sets concurrency scope ConcurrencyScope 设置并发控制作用域
func (b *Builder) ConcurrencyScope(concurrencyScope config.ConcurrencyScope) *Builder {
	b.ensureBuilder()
	b.cfg.ConcurrencyScope = concurrencyScope
	return b
}

// IsConcurrent sets concurrent login switch IsConcurrent 设置是否允许并发登录
func (b *Builder) IsConcurrent(concurrent bool) *Builder {
	b.ensureBuilder()
	b.cfg.IsConcurrent = concurrent
	return b
}

// IsShare sets shared token switch IsShare 设置是否共享 Token
func (b *Builder) IsShare(share bool) *Builder {
	b.ensureBuilder()
	b.cfg.IsShare = share
	return b
}

// MaxLoginCount sets max login count MaxLoginCount 设置最大登录数量
func (b *Builder) MaxLoginCount(count int64) *Builder {
	b.ensureBuilder()
	b.cfg.MaxLoginCount = count
	return b
}

// ReplacedLoginExitMode sets replaced login strategy ReplacedLoginExitMode 设置非并发登录处理策略
func (b *Builder) ReplacedLoginExitMode(mode config.ReplacedLoginExitMode) *Builder {
	b.ensureBuilder()
	b.cfg.ReplacedLoginExitMode = mode
	return b
}

// OverflowLogoutMode sets overflow logout mode OverflowLogoutMode 设置超限登录下线模式
func (b *Builder) OverflowLogoutMode(mode config.LogoutMode) *Builder {
	b.ensureBuilder()
	b.cfg.OverflowLogoutMode = mode
	return b
}

// IsReadBody sets body read switch IsReadBody 设置是否从请求体读取 Token
func (b *Builder) IsReadBody(isRead bool) *Builder {
	b.ensureBuilder()
	b.cfg.IsReadBody = isRead
	return b
}

// IsReadQuery sets query read switch IsReadQuery 设置是否从 Query 读取 Token
func (b *Builder) IsReadQuery(isRead bool) *Builder {
	b.ensureBuilder()
	b.cfg.IsReadQuery = isRead
	return b
}

// IsReadHeader sets header read switch IsReadHeader 设置是否从 Header 读取 Token
func (b *Builder) IsReadHeader(isRead bool) *Builder {
	b.ensureBuilder()
	b.cfg.IsReadHeader = isRead
	return b
}

// IsReadCookie sets cookie read switch IsReadCookie 设置是否从 Cookie 读取 Token
func (b *Builder) IsReadCookie(isRead bool) *Builder {
	b.ensureBuilder()
	b.cfg.IsReadCookie = isRead
	return b
}

// TokenStyle sets token style TokenStyle 设置 Token 生成风格
func (b *Builder) TokenStyle(style adapter.TokenStyle) *Builder {
	b.ensureBuilder()
	b.cfg.TokenStyle = style
	return b
}

// JwtSecretKey sets JWT secret key JwtSecretKey 设置 JWT 密钥
func (b *Builder) JwtSecretKey(key string) *Builder {
	b.ensureBuilder()
	b.cfg.JwtSecretKey = key
	return b
}

// IsLog sets log switch IsLog 设置是否开启日志
func (b *Builder) IsLog(isLog bool) *Builder {
	b.ensureBuilder()
	b.cfg.IsLog = isLog
	return b
}

// IsPrintBanner sets banner print switch IsPrintBanner 设置是否打印 Banner
func (b *Builder) IsPrintBanner(isPrint bool) *Builder {
	b.ensureBuilder()
	b.cfg.IsPrintBanner = isPrint
	return b
}

// AsyncEvent sets async event switch AsyncEvent 设置是否异步触发事件
func (b *Builder) AsyncEvent(asyncEvent bool) *Builder {
	b.ensureBuilder()
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

// CookieSecure sets cookie secure switch CookieSecure 设置 Cookie HTTPS 限制
func (b *Builder) CookieSecure(secure bool) *Builder {
	cc := b.ensureCookieConfig()
	cc.Secure = secure
	return b
}

// CookieHttpOnly sets cookie httpOnly switch CookieHttpOnly 设置 Cookie HttpOnly
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
	b.ensureBuilder()

	// Preserve explicit nil for validation 保留显式 nil 以交给配置校验处理
	if cfg == nil {
		b.cfg.CookieConfig = nil
		return b
	}

	// Store a copied cookie config 保存 Cookie 配置副本
	copyCfg := *cfg
	b.cfg.CookieConfig = &copyCfg
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

// RenewPoolPrintStatusInterval enables periodic pool status logging when positive and logging is enabled; zero disables it. RenewPoolPrintStatusInterval 为正且启用日志时开启池状态定时打印，零值禁用。
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

// RenewPoolNonBlocking sets submission mode; false fails validation for the bundled pool. RenewPoolNonBlocking 设置提交模式，内置池配置为 false 时校验失败。
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

// LoggerFilename sets the fixed active log filename. LoggerFilename 设置固定的活动日志文件名。
func (b *Builder) LoggerFilename(filename string) *Builder {
	b.ensureBuilder()
	b.loggerConfig.Filename = filename
	return b
}

// LoggerPrefix sets the prefix before the level label and message. LoggerPrefix 设置级别标签和消息之前的前缀。
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

// LoggerMaxSize limits each log file in MiB. LoggerMaxSize 以 MiB 限制单个日志文件大小。
func (b *Builder) LoggerMaxSize(size int) *Builder {
	b.ensureBuilder()
	b.loggerConfig.MaxSize = size
	return b
}

// LoggerMaxBackups limits historical backups excluding the active file; zero disables the count limit. LoggerMaxBackups 限制历史备份数量，不包含活动文件；零值表示不按数量限制。
func (b *Builder) LoggerMaxBackups(count int) *Builder {
	b.ensureBuilder()
	b.loggerConfig.MaxBackups = count
	return b
}

// LoggerMaxAge limits backup retention in 24-hour days since rotation; zero disables age-based deletion. LoggerMaxAge 从轮转时间起以 24 小时为一天限制备份保留时间，零值表示不按时间删除。
func (b *Builder) LoggerMaxAge(days int) *Builder {
	b.ensureBuilder()
	b.loggerConfig.MaxAge = days
	return b
}

// LoggerCompress enables gzip compression of rotated log files. LoggerCompress 启用轮转日志文件的 gzip 压缩。
func (b *Builder) LoggerCompress(enable bool) *Builder {
	b.ensureBuilder()
	b.loggerConfig.Compress = enable
	return b
}

// LoggerLocalTime uses local time in rotated log filenames. LoggerLocalTime 在轮转日志文件名中使用本地时间。
func (b *Builder) LoggerLocalTime(enable bool) *Builder {
	b.ensureBuilder()
	b.loggerConfig.LocalTime = enable
	return b
}

// SetGenerator sets token generator SetGenerator 设置 Token 生成器
func (b *Builder) SetGenerator(generator adapter.Generator) *Builder {
	b.components.Generator = generator
	return b
}

// SetStorage sets a manager-owned storage adapter by default. SetStorage 设置默认由 Manager 持有的存储适配器。
func (b *Builder) Storage(storage adapter.Storage) *Builder {
	b.components.Storage = storage
	return b
}

// SetCodec sets codec adapter SetCodec 设置编解码器
func (b *Builder) SetCodec(codec adapter.Codec) *Builder {
	b.components.Codec = codec
	return b
}

// SetLog sets a manager-owned log adapter by default. SetLog 设置默认由 Manager 持有的日志适配器。
func (b *Builder) SetLog(log adapter.Log) *Builder {
	b.components.Log = log
	return b
}

// SetPool sets a manager-owned async task pool by default. SetPool 设置默认由 Manager 持有的异步任务池。
func (b *Builder) SetPool(pool adapter.Pool) *Builder {
	b.components.Pool = pool
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

// SetGeneratorFactory sets a custom factory; nil restores the bundled default. SetGeneratorFactory 设置自定义工厂，nil 恢复内置默认实现。
func (b *Builder) SetGeneratorFactory(factory GeneratorFactory) *Builder {
	b.factories.Generator = factory
	return b
}

// SetStorageFactory sets a custom factory; nil restores the bundled default. SetStorageFactory 设置自定义工厂，nil 恢复内置默认实现。
func (b *Builder) SetStorageFactory(factory StorageFactory) *Builder {
	b.factories.Storage = factory
	return b
}

// SetCodecFactory sets a custom factory; nil restores the bundled default. SetCodecFactory 设置自定义工厂，nil 恢复内置默认实现。
func (b *Builder) SetCodecFactory(factory CodecFactory) *Builder {
	b.factories.Codec = factory
	return b
}

// SetLogFactory sets a custom factory; nil restores the bundled default. SetLogFactory 设置自定义工厂，nil 恢复内置默认实现。
func (b *Builder) SetLogFactory(factory LogFactory) *Builder {
	b.factories.Log = factory
	return b
}

// SetPoolFactory sets a custom factory; nil restores the bundled default. SetPoolFactory 设置自定义工厂，nil 恢复内置默认实现。
func (b *Builder) SetPoolFactory(factory PoolFactory) *Builder {
	b.factories.Pool = factory
	return b
}

// SetAccessProvider sets permission and role provider SetAccessProvider 设置权限与角色提供器
func (b *Builder) SetAccessProvider(provider AccessProvider) *Builder {
	b.accessProvider = provider
	return b
}

// JwtSecret enables JWT style and sets secret JwtSecret 启用 JWT 风格并设置密钥
func (b *Builder) JwtSecret(secret string) *Builder {
	b.ensureBuilder()
	b.cfg.TokenStyle = adapter.TokenStyleJWT
	b.cfg.JwtSecretKey = secret
	return b
}

// Clone copies configs and options while sharing injected components and callbacks; use factories for independent resources. Clone 复制配置和选项，共享注入组件和回调；独立资源应使用工厂。
func (b *Builder) Clone() *Builder {
	b.ensureBuilder()
	clone := *b
	clone.cfg = b.cfg.Clone()
	clone.renewPoolConfig = b.renewPoolConfig.Clone()
	clone.loggerConfig = b.loggerConfig.Clone()

	if len(b.managerOptions) > 0 {
		clone.managerOptions = append([]manager.Option(nil), b.managerOptions...)
	}
	return &clone
}

// buildManager selects defaults and delegates component assembly. buildManager 选择默认组件并调用装配函数。
func (b *Builder) buildManager() (*manager.Manager, error) {
	b.ensureBuilder()

	// Keep each build independent of later builder changes. 每次构建使用独立的组件配置快照。
	loggerConfig := b.loggerConfig.Clone()
	renewPoolConfig := b.renewPoolConfig.Clone()
	factories := b.factories
	if factories.Generator == nil {
		factories.Generator = builder.DefaultGeneratorFactory
	}
	if factories.Storage == nil {
		factories.Storage = builder.DefaultStorageFactory
	}
	if factories.Codec == nil {
		factories.Codec = builder.DefaultCodecFactory
	}
	if factories.Log == nil {
		factories.Log = func(_ *config.Config) (adapter.Log, error) {
			return builder.DefaultLogFactory(loggerConfig)
		}
	}
	if factories.Pool == nil {
		factories.Pool = func(_ *config.Config) (adapter.Pool, error) {
			return builder.DefaultPoolFactory(renewPoolConfig)
		}
	}

	// Reporting applies to custom pools too, independently of their capacity settings. 状态报告也适用于自定义池，与池容量配置无关。
	if renewPoolConfig.PrintStatusInterval < 0 {
		return nil, fmt.Errorf("build manager failed: RenewPoolConfig.PrintStatusInterval must not be negative")
	}
	mgr, err := builder.Build(b.cfg, b.components, factories, b.accessProvider, b.managerOptions...)
	if err != nil {
		return nil, err
	}

	// Manager owns status reporting and stops it during shutdown. Manager 管理状态报告并在关闭时停止。
	if mgr.GetConfig().IsLog && renewPoolConfig.PrintStatusInterval > 0 {
		mgr.StartRenewPoolStatusLogger(renewPoolConfig.PrintStatusInterval)
	}
	return mgr, nil
}

// Build creates and registers an Auth, rejecting duplicate auth types; use BuildInstance for isolated ownership. Build 创建并注册 Auth，拒绝同名体系；独立实例使用 BuildInstance。
func (b *Builder) Build() (*Auth, error) {
	// Reject existing names before allocating owned resources. 在分配自有资源前拒绝已注册名称。
	authType := getAutoType(b.GetConfig().AuthType)
	if _, err := GetManager(authType); err == nil {
		return nil, fmt.Errorf("%w: %s", derror.ErrManagerAlreadyRegistered, authType)
	} else if err != derror.ErrManagerNotFound {
		return nil, err
	}

	auth, err := b.BuildInstance()
	if err != nil {
		return nil, err
	}

	// Publish atomically; a concurrent winner keeps its registration while this candidate releases its owned resources. 原子发布；并发获胜者保留注册，本次候选释放其自有资源。
	if err := registerManager(auth.Manager()); err != nil {
		auth.Close()
		return nil, err
	}
	return auth, nil
}

// BuildInstance creates an isolated Auth without registry access. BuildInstance 创建独立 Auth，不访问注册表。
func (b *Builder) BuildInstance() (*Auth, error) {
	mgr, err := b.buildManager()
	if err != nil {
		return nil, err
	}
	return New(mgr), nil
}

// ensureBuilder initializes missing internals ensureBuilder 初始化缺失的内部配置
func (b *Builder) ensureBuilder() {
	if b.cfg == nil {
		b.cfg = config.DefaultConfig()
	}
	if b.renewPoolConfig == nil {
		b.renewPoolConfig = ants.DefaultRenewPoolConfig()
	}
	if b.loggerConfig == nil {
		b.loggerConfig = dlog.DefaultLoggerConfig()
	}
}

// ensureCookieConfig initializes cookie config when needed ensureCookieConfig 在需要时初始化 Cookie 配置
func (b *Builder) ensureCookieConfig() *config.CookieConfig {
	// Ensure parent config first 先确保父级配置存在
	b.ensureBuilder()

	// Restore default cookie config when missing 缺失 Cookie 配置时恢复默认配置
	if b.cfg.CookieConfig == nil {
		b.cfg.CookieConfig = config.DefaultCookieConfig()
	}

	return b.cfg.CookieConfig
}

// durationToSeconds rounds positive durations up and keeps negative durations invalid. durationToSeconds 将正时长向上取整，并保持负时长无效。
func durationToSeconds(d time.Duration) int64 {
	// Keep zero invalid and map negative durations below NoLimit so validation cannot mistake them for unlimited. 保留零值无效，并将负时长映射到 NoLimit 以下，避免被误认为无限制。
	if d <= 0 {
		if d == 0 {
			return 0
		}

		seconds := int64(d / time.Second)
		if d%time.Second != 0 {
			seconds--
		}
		if seconds >= config.NoLimit {
			return config.NoLimit - 1
		}
		return seconds
	}

	// Convert whole seconds first 先转换整秒部分
	seconds := int64(d / time.Second)

	// Round up when sub-second remainder exists 存在亚秒余数时向上取整
	if d%time.Second != 0 {
		seconds++
	}

	return seconds
}
