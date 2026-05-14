package builder

import (
	"fmt"
	"strings"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/banner"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/manager"
)

// GeneratorFactory creates default token generator GeneratorFactory 创建默认 Token 生成器
type GeneratorFactory func(cfg *config.Config) (adapter.Generator, error)

// StorageFactory creates default storage adapter StorageFactory 创建默认存储适配器
type StorageFactory func(cfg *config.Config) (adapter.Storage, error)

// CodecFactory creates default codec adapter CodecFactory 创建默认编解码器适配器
type CodecFactory func(cfg *config.Config) (adapter.Codec, error)

// LogFactory creates default log adapter LogFactory 创建默认日志适配器
type LogFactory func(cfg *config.Config) (adapter.Log, error)

// PoolFactory creates default async task pool PoolFactory 创建默认异步任务池
type PoolFactory func(cfg *config.Config) (adapter.Pool, error)

// Builder defines manager builder Builder 定义 Manager 构建器
type Builder struct {
	authType         string                  // authType stores auth type authType 存储认证体系类型
	keyPrefix        string                  // keyPrefix stores storage key prefix keyPrefix 存储存储键前缀
	tokenName        string                  // tokenName stores token name tokenName 存储 Token 名称
	timeout          int64                   // timeout stores token timeout seconds timeout 存储 Token 超时时间秒数
	autoRenew        bool                    // autoRenew controls auto renew autoRenew 控制是否启用自动续期
	renewMaxRefresh  int64                   // renewMaxRefresh stores renew trigger threshold renewMaxRefresh 存储续期触发阈值
	renewInterval    int64                   // renewInterval stores minimum renew interval renewInterval 存储最小续期间隔
	activeTimeout    int64                   // activeTimeout stores max inactive duration activeTimeout 存储最大不活跃时长
	concurrencyScope config.ConcurrencyScope // concurrencyScope stores concurrency scope concurrencyScope 存储并发控制作用域
	isConcurrent     bool                    // isConcurrent controls concurrent login isConcurrent 控制是否允许并发登录
	isShare          bool                    // isShare controls shared token isShare 控制是否共享同一 Token
	maxLoginCount    int64                   // maxLoginCount stores max login count maxLoginCount 存储最大并发登录数量
	isReadBody       bool                    // isReadBody controls body token read isReadBody 控制是否从请求体读取 Token
	isReadHeader     bool                    // isReadHeader controls header token read isReadHeader 控制是否从 Header 读取 Token
	isReadCookie     bool                    // isReadCookie controls cookie token read isReadCookie 控制是否从 Cookie 读取 Token
	tokenStyle       adapter.TokenStyle      // tokenStyle stores token style tokenStyle 存储 Token 生成风格
	jwtSecretKey     string                  // jwtSecretKey stores JWT secret key jwtSecretKey 存储 JWT 密钥
	isLog            bool                    // isLog controls logging isLog 控制是否开启日志输出
	isPrintBanner    bool                    // isPrintBanner controls banner print isPrintBanner 控制是否打印启动 Banner
	asyncEvent       bool                    // asyncEvent controls async event asyncEvent 控制是否异步触发事件

	cookieConfig *config.CookieConfig // cookieConfig stores cookie config cookieConfig 存储 Cookie 配置

	generator adapter.Generator // generator stores token generator generator 存储 Token 生成器
	storage   adapter.Storage   // storage stores storage adapter storage 存储存储适配器
	codec     adapter.Codec     // codec stores codec adapter codec 存储编解码器适配器
	log       adapter.Log       // log stores log adapter log 存储日志适配器
	pool      adapter.Pool      // pool stores async task pool pool 存储异步任务协程池组件

	generatorFactory GeneratorFactory // generatorFactory creates default generator generatorFactory 创建默认生成器
	storageFactory   StorageFactory   // storageFactory creates default storage storageFactory 创建默认存储
	codecFactory     CodecFactory     // codecFactory creates default codec codecFactory 创建默认编解码器
	logFactory       LogFactory       // logFactory creates default logger logFactory 创建默认日志
	poolFactory      PoolFactory      // poolFactory creates default pool poolFactory 创建默认协程池

	customPermissionListFunc    func(loginID, authType string) ([]string, error)                   // customPermissionListFunc stores custom permission callback customPermissionListFunc 存储自定义权限列表回调
	customRoleListFunc          func(loginID, authType string) ([]string, error)                   // customRoleListFunc stores custom role callback customRoleListFunc 存储自定义角色列表回调
	customPermissionListExtFunc func(loginID, device, deviceId, authType string) ([]string, error) // customPermissionListExtFunc stores custom extended permission callback customPermissionListExtFunc 存储扩展权限列表回调
	customRoleListExtFunc       func(loginID, device, deviceId, authType string) ([]string, error) // customRoleListExtFunc stores custom extended role callback customRoleListExtFunc 存储扩展角色列表回调
}

// NewBuilder creates builder with default config NewBuilder 创建使用默认配置的构建器
func NewBuilder() *Builder {
	return &Builder{
		authType:         config.DefaultAuthType,
		keyPrefix:        config.DefaultKeyPrefix,
		tokenName:        config.DefaultTokenName,
		timeout:          config.DefaultTimeout,
		autoRenew:        true,
		renewMaxRefresh:  config.DefaultTimeout / 2,
		renewInterval:    config.NoLimit,
		activeTimeout:    config.NoLimit,
		concurrencyScope: config.ConcurrencyScopeAccount,
		isConcurrent:     true,
		isShare:          false,
		maxLoginCount:    config.NoLimit,
		isReadBody:       false,
		isReadHeader:     true,
		isReadCookie:     false,
		tokenStyle:       adapter.TokenStyleUUID,
		jwtSecretKey:     config.DefaultJWTSecretKey,
		isLog:            false,
		isPrintBanner:    true,
		asyncEvent:       true,

		cookieConfig: config.DefaultCookieConfig(),
	}
}

// AuthType sets auth type with suffix fix AuthType 设置认证体系类型并自动补全冒号
func (b *Builder) AuthType(authType string) *Builder {
	if authType == "" {
		b.authType = config.DefaultAuthType
	} else if !strings.HasSuffix(authType, ":") {
		b.authType = authType + ":"
	} else {
		b.authType = authType
	}
	return b
}

// KeyPrefix sets key prefix with suffix fix KeyPrefix 设置存储键前缀并自动补全冒号
func (b *Builder) KeyPrefix(keyPrefix string) *Builder {
	if keyPrefix == "" {
		b.keyPrefix = config.DefaultKeyPrefix
	} else if !strings.HasSuffix(keyPrefix, ":") {
		b.keyPrefix = keyPrefix + ":"
	} else {
		b.keyPrefix = keyPrefix
	}
	return b
}

// TokenName sets token name TokenName 设置 Token 名称
func (b *Builder) TokenName(name string) *Builder {
	b.tokenName = name
	return b
}

// Timeout sets timeout seconds Timeout 设置超时时间秒数
func (b *Builder) Timeout(seconds int64) *Builder {
	b.timeout = seconds
	return b
}

// TimeoutDuration sets timeout by duration TimeoutDuration 按时间段设置超时时间
func (b *Builder) TimeoutDuration(d time.Duration) *Builder {
	b.timeout = int64(d.Seconds())
	return b
}

// AutoRenew sets auto renew switch AutoRenew 设置是否启用自动续期
func (b *Builder) AutoRenew(autoRenew bool) *Builder {
	b.autoRenew = autoRenew
	return b
}

// RenewMaxRefresh sets renew trigger threshold RenewMaxRefresh 设置自动续期触发阈值
func (b *Builder) RenewMaxRefresh(seconds int64) *Builder {
	b.renewMaxRefresh = seconds
	return b
}

// RenewInterval sets minimum renew interval RenewInterval 设置最小续期间隔
func (b *Builder) RenewInterval(seconds int64) *Builder {
	b.renewInterval = seconds
	return b
}

// ActiveTimeout sets max inactive duration ActiveTimeout 设置最大不活跃时长
func (b *Builder) ActiveTimeout(seconds int64) *Builder {
	b.activeTimeout = seconds
	return b
}

// ConcurrencyScope sets concurrency scope ConcurrencyScope 设置并发控制作用域
func (b *Builder) ConcurrencyScope(concurrencyScope config.ConcurrencyScope) *Builder {
	b.concurrencyScope = concurrencyScope
	return b
}

// IsConcurrent sets concurrent login switch IsConcurrent 设置是否允许并发登录
func (b *Builder) IsConcurrent(concurrent bool) *Builder {
	b.isConcurrent = concurrent
	return b
}

// IsShare sets shared token switch IsShare 设置是否共享同一 Token
func (b *Builder) IsShare(share bool) *Builder {
	b.isShare = share
	return b
}

// MaxLoginCount sets max login count MaxLoginCount 设置最大并发登录数量
func (b *Builder) MaxLoginCount(count int64) *Builder {
	b.maxLoginCount = count
	return b
}

// IsReadBody sets body read switch IsReadBody 设置是否从请求体读取 Token
func (b *Builder) IsReadBody(isRead bool) *Builder {
	b.isReadBody = isRead
	return b
}

// IsReadHeader sets header read switch IsReadHeader 设置是否从 HTTP Header 读取 Token
func (b *Builder) IsReadHeader(isRead bool) *Builder {
	b.isReadHeader = isRead
	return b
}

// IsReadCookie sets cookie read switch IsReadCookie 设置是否从 Cookie 读取 Token
func (b *Builder) IsReadCookie(isRead bool) *Builder {
	b.isReadCookie = isRead
	return b
}

// TokenStyle sets token style TokenStyle 设置 Token 生成风格
func (b *Builder) TokenStyle(style adapter.TokenStyle) *Builder {
	b.tokenStyle = style
	return b
}

// JwtSecretKey sets JWT secret key JwtSecretKey 设置 JWT 密钥
func (b *Builder) JwtSecretKey(key string) *Builder {
	b.jwtSecretKey = key
	return b
}

// IsLog sets log switch IsLog 设置是否开启日志输出
func (b *Builder) IsLog(isLog bool) *Builder {
	b.isLog = isLog
	return b
}

// IsPrintBanner sets banner print switch IsPrintBanner 设置是否打印启动 Banner
func (b *Builder) IsPrintBanner(isPrint bool) *Builder {
	b.isPrintBanner = isPrint
	return b
}

// AsyncEvent sets async event switch AsyncEvent 设置是否异步触发事件
func (b *Builder) AsyncEvent(asyncEvent bool) *Builder {
	b.asyncEvent = asyncEvent
	return b
}

// CookieDomain sets cookie domain CookieDomain 设置 Cookie 作用域 Domain
func (b *Builder) CookieDomain(domain string) *Builder {
	if b.cookieConfig == nil {
		b.cookieConfig = &config.CookieConfig{}
	}
	b.cookieConfig.Domain = domain
	return b
}

// CookiePath sets cookie path CookiePath 设置 Cookie 路径
func (b *Builder) CookiePath(path string) *Builder {
	if b.cookieConfig == nil {
		b.cookieConfig = &config.CookieConfig{}
	}
	b.cookieConfig.Path = path
	return b
}

// CookieSecure sets cookie secure switch CookieSecure 设置 Cookie 是否仅在 HTTPS 下生效
func (b *Builder) CookieSecure(secure bool) *Builder {
	if b.cookieConfig == nil {
		b.cookieConfig = &config.CookieConfig{}
	}
	b.cookieConfig.Secure = secure
	return b
}

// CookieHttpOnly sets cookie httpOnly switch CookieHttpOnly 设置 Cookie 是否禁止 JavaScript 访问
func (b *Builder) CookieHttpOnly(httpOnly bool) *Builder {
	if b.cookieConfig == nil {
		b.cookieConfig = &config.CookieConfig{}
	}
	b.cookieConfig.HttpOnly = httpOnly
	return b
}

// CookieSameSite sets cookie sameSite CookieSameSite 设置 Cookie SameSite 属性
func (b *Builder) CookieSameSite(sameSite config.SameSiteMode) *Builder {
	if b.cookieConfig == nil {
		b.cookieConfig = &config.CookieConfig{}
	}
	b.cookieConfig.SameSite = sameSite
	return b
}

// CookieMaxAge sets cookie max age CookieMaxAge 设置 Cookie 过期时间秒数
func (b *Builder) CookieMaxAge(maxAge int64) *Builder {
	if b.cookieConfig == nil {
		b.cookieConfig = &config.CookieConfig{}
	}
	b.cookieConfig.MaxAge = maxAge
	return b
}

// CookieConfig sets full cookie config CookieConfig 设置完整 Cookie 配置
func (b *Builder) CookieConfig(cfg *config.CookieConfig) *Builder {
	b.cookieConfig = cfg
	return b
}

// SetGenerator sets token generator SetGenerator 设置 Token 生成器
func (b *Builder) SetGenerator(generator adapter.Generator) *Builder {
	b.generator = generator
	return b
}

// SetStorage sets storage adapter SetStorage 设置存储适配器
func (b *Builder) SetStorage(storage adapter.Storage) *Builder {
	b.storage = storage
	return b
}

// SetCodec sets codec adapter SetCodec 设置编解码器适配器
func (b *Builder) SetCodec(codec adapter.Codec) *Builder {
	b.codec = codec
	return b
}

// SetLog sets log adapter SetLog 设置日志适配器
func (b *Builder) SetLog(log adapter.Log) *Builder {
	b.log = log
	return b
}

// SetPool sets async task pool SetPool 设置异步任务协程池
func (b *Builder) SetPool(pool adapter.Pool) *Builder {
	b.pool = pool
	return b
}

// SetGeneratorFactory sets default generator factory SetGeneratorFactory 设置默认 Token 生成器工厂
func (b *Builder) SetGeneratorFactory(factory GeneratorFactory) *Builder {
	b.generatorFactory = factory
	return b
}

// SetStorageFactory sets default storage factory SetStorageFactory 设置默认存储工厂
func (b *Builder) SetStorageFactory(factory StorageFactory) *Builder {
	b.storageFactory = factory
	return b
}

// SetCodecFactory sets default codec factory SetCodecFactory 设置默认编解码器工厂
func (b *Builder) SetCodecFactory(factory CodecFactory) *Builder {
	b.codecFactory = factory
	return b
}

// SetLogFactory sets default log factory SetLogFactory 设置默认日志工厂
func (b *Builder) SetLogFactory(factory LogFactory) *Builder {
	b.logFactory = factory
	return b
}

// SetPoolFactory sets default pool factory SetPoolFactory 设置默认协程池工厂
func (b *Builder) SetPoolFactory(factory PoolFactory) *Builder {
	b.poolFactory = factory
	return b
}

// SetCustomPermissionListFunc sets permission callback SetCustomPermissionListFunc 设置自定义权限列表获取函数
func (b *Builder) SetCustomPermissionListFunc(f func(loginID, authType string) ([]string, error)) *Builder {
	b.customPermissionListFunc = f
	return b
}

// SetCustomRoleListFunc sets role callback SetCustomRoleListFunc 设置自定义角色列表获取函数
func (b *Builder) SetCustomRoleListFunc(f func(loginID, authType string) ([]string, error)) *Builder {
	b.customRoleListFunc = f
	return b
}

// SetCustomPermissionListExtFunc sets extended permission callback SetCustomPermissionListExtFunc 设置扩展权限列表获取函数
func (b *Builder) SetCustomPermissionListExtFunc(f func(loginID, device, deviceId, authType string) ([]string, error)) *Builder {
	b.customPermissionListExtFunc = f
	return b
}

// SetCustomRoleListExtFunc sets extended role callback SetCustomRoleListExtFunc 设置扩展角色列表获取函数
func (b *Builder) SetCustomRoleListExtFunc(f func(loginID, device, deviceId, authType string) ([]string, error)) *Builder {
	b.customRoleListExtFunc = f
	return b
}

// JwtSecret enables JWT style and sets secret JwtSecret 设置 JWT 模式并指定密钥
func (b *Builder) JwtSecret(secret string) *Builder {
	b.tokenStyle = adapter.TokenStyleJWT
	b.jwtSecretKey = secret
	return b
}

// Clone clones builder with deep copy Clone 深拷贝当前构建器
func (b *Builder) Clone() *Builder {
	clone := *b
	if b.cookieConfig != nil {
		cookieCopy := *b.cookieConfig
		clone.cookieConfig = &cookieCopy
	}
	return &clone
}

// Build builds manager and returns configuration errors Build 构建 Manager 实例并返回配置错误
func (b *Builder) Build() (*manager.Manager, error) {
	if b.cookieConfig == nil {
		b.cookieConfig = config.DefaultCookieConfig()
	}

	cfg := &config.Config{
		AuthType:         b.authType,
		KeyPrefix:        b.keyPrefix,
		TokenName:        b.tokenName,
		Timeout:          b.timeout,
		AutoRenew:        b.autoRenew,
		RenewMaxRefresh:  b.renewMaxRefresh,
		RenewInterval:    b.renewInterval,
		ActiveTimeout:    b.activeTimeout,
		ConcurrencyScope: b.concurrencyScope,
		IsConcurrent:     b.isConcurrent,
		IsShare:          b.isShare,
		MaxLoginCount:    b.maxLoginCount,
		IsReadBody:       b.isReadBody,
		IsReadHeader:     b.isReadHeader,
		IsReadCookie:     b.isReadCookie,
		TokenStyle:       b.tokenStyle,
		JwtSecretKey:     b.jwtSecretKey,
		IsLog:            b.isLog,
		IsPrintBanner:    b.isPrintBanner,
		AsyncEvent:       b.asyncEvent,
		CookieConfig:     b.cookieConfig,
	}

	err := cfg.Validate()
	if err != nil {
		return nil, fmt.Errorf("build manager invalid config: %w", err)
	}

	if b.generator == nil {
		if b.generatorFactory != nil {
			b.generator, err = b.generatorFactory(cfg)
			if err != nil {
				return nil, fmt.Errorf("build manager create generator: %w", err)
			}
		}
		if b.generator == nil {
			return nil, fmt.Errorf("build manager missing generator: call SetGenerator or SetGeneratorFactory")
		}
	}

	if b.storage == nil {
		if b.storageFactory != nil {
			b.storage, err = b.storageFactory(cfg)
			if err != nil {
				return nil, fmt.Errorf("build manager create storage: %w", err)
			}
		}
		if b.storage == nil {
			return nil, fmt.Errorf("build manager missing storage: call SetStorage or SetStorageFactory")
		}
	}

	if b.codec == nil {
		if b.codecFactory != nil {
			b.codec, err = b.codecFactory(cfg)
			if err != nil {
				return nil, fmt.Errorf("build manager create codec: %w", err)
			}
		}
		if b.codec == nil {
			return nil, fmt.Errorf("build manager missing codec: call SetCodec or SetCodecFactory")
		}
	}

	if b.isLog {
		if b.log == nil {
			if b.logFactory != nil {
				b.log, err = b.logFactory(cfg)
			}
			if err != nil {
				return nil, fmt.Errorf("build manager create logger: %w", err)
			}
			if b.log == nil {
				return nil, fmt.Errorf("build manager missing logger: call SetLog or SetLogFactory")
			}
		}
	} else {
		b.log = adapter.NewNopLogger()
	}

	if b.autoRenew && b.pool == nil && b.poolFactory != nil {
		b.pool, err = b.poolFactory(cfg)
		if err != nil {
			return nil, fmt.Errorf("build manager create renew pool: %w", err)
		}
	}

	if b.isPrintBanner {
		banner.PrintBanner(cfg)
	}

	mgr := manager.NewManager(cfg, b.generator, b.storage, b.codec, b.log, b.pool, b.customPermissionListFunc, b.customRoleListFunc, b.customPermissionListExtFunc, b.customRoleListExtFunc)

	return mgr, nil
}

// MustBuild builds manager and panics on error MustBuild 构建 Manager，失败时触发 panic
func (b *Builder) MustBuild() *manager.Manager {
	mgr, err := b.Build()
	if err != nil {
		panic(err)
	}
	return mgr
}
