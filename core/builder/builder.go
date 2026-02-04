package builder

import (
	djson "github.com/Zany2/dtoken-go/com/codec/json"
	"github.com/Zany2/dtoken-go/com/generator/dgenerator"
	"github.com/Zany2/dtoken-go/com/log/dlog"
	"github.com/Zany2/dtoken-go/com/log/nop"
	"github.com/Zany2/dtoken-go/com/pool/ants"
	"github.com/Zany2/dtoken-go/com/storage/memory"
	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/banner"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/manager"
	"strings"
	"time"
)

// Builder 构建器
type Builder struct {
	authType         string                  // authType 认证体系类型
	keyPrefix        string                  // keyPrefix 存储键的前缀
	tokenName        string                  // tokenName 客户端 Token 名称
	timeout          int64                   // timeout Token 过期时间（单位：秒）
	autoRenew        bool                    // autoRenew 是否启用自动续期
	renewMaxRefresh  int64                   // renewMaxRefresh 最大无感刷新时间（单位：秒）
	renewInterval    int64                   // renewInterval 最小续期间隔（单位：秒）
	activeTimeout    int64                   // activeTimeout 活跃超时时间（单位：秒，超过未访问则踢出）
	concurrencyScope config.ConcurrencyScope // concurrencyScope 并发控制的作用域
	isConcurrent     bool                    // isConcurrent 是否允许并发登录
	isShare          bool                    // isShare 是否共用同一个 Token
	maxLoginCount    int64                   // maxLoginCount 最大并发登录数量
	isReadBody       bool                    // isReadBody 是否从请求体读取 Token
	isReadHeader     bool                    // isReadHeader 是否从 HTTP Header 读取 Token
	isReadCookie     bool                    // isReadCookie 是否从 Cookie 读取 Token
	tokenStyle       adapter.TokenStyle      // tokenStyle Token 生成方式
	jwtSecretKey     string                  // jwtSecretKey JWT 密钥
	isLog            bool                    // isLog 是否开启日志输出
	isPrintBanner    bool                    // isPrintBanner 是否打印启动 Banner
	asyncEvent       bool                    // asyncEvent 是否异步触发事件

	cookieConfig    *config.CookieConfig  // cookieConfig Cookie 配置
	renewPoolConfig *ants.RenewPoolConfig // renewPoolConfig 续期协程池配置
	logConfig       *dlog.LoggerConfig    // logConfig 日志配置

	generator adapter.Generator // generator Token 生成器
	storage   adapter.Storage   // storage 存储适配器
	codec     adapter.Codec     // codec 编解码器适配器
	log       adapter.Log       // log 日志适配器
	pool      adapter.Pool      // pool 异步任务协程池组件

	customPermissionListFunc    func(loginID, authType string) ([]string, error)                   // customPermissionListFunc 自定义权限列表获取函数
	customRoleListFunc          func(loginID, authType string) ([]string, error)                   // customRoleListFunc 自定义角色列表获取函数
	customPermissionListExtFunc func(loginID, device, deviceId, authType string) ([]string, error) // customPermissionListExtFunc 自定义权限列表获取函数（扩展版本，支持设备信息）
	customRoleListExtFunc       func(loginID, device, deviceId, authType string) ([]string, error) // customRoleListExtFunc 自定义角色列表获取函数（扩展版本，支持设备信息）
}

// NewBuilder 创建新的构建器（使用默认配置）
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
		jwtSecretKey:     dgenerator.DefaultJWTSecret,
		isLog:            false,
		isPrintBanner:    true,
		asyncEvent:       true,

		cookieConfig:    config.DefaultCookieConfig(),
		renewPoolConfig: ants.DefaultRenewPoolConfig(),
	}
}

// AuthType 设置认证体系类型（自动补全冒号）
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

// KeyPrefix 设置存储键的前缀（自动补全冒号）
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

// TokenName 设置 Token 名称
func (b *Builder) TokenName(name string) *Builder {
	b.tokenName = name
	return b
}

// Timeout 设置超时时间（单位：秒）
func (b *Builder) Timeout(seconds int64) *Builder {
	b.timeout = seconds
	return b
}

// TimeoutDuration 设置超时时间（时间段）
func (b *Builder) TimeoutDuration(d time.Duration) *Builder {
	b.timeout = int64(d.Seconds())
	return b
}

// AutoRenew 设置是否启用自动续期
func (b *Builder) AutoRenew(autoRenew bool) *Builder {
	b.autoRenew = autoRenew
	return b
}

// RenewMaxRefresh 设置 Token 自动续期触发阈值（单位：秒）
func (b *Builder) RenewMaxRefresh(seconds int64) *Builder {
	b.renewMaxRefresh = seconds
	return b
}

// RenewInterval 设置最小续期间隔（单位：秒）
func (b *Builder) RenewInterval(seconds int64) *Builder {
	b.renewInterval = seconds
	return b
}

// ActiveTimeout 设置最大不活跃时长（单位：秒）
func (b *Builder) ActiveTimeout(seconds int64) *Builder {
	b.activeTimeout = seconds
	return b
}

// ConcurrencyScope 并发控制的作用域
func (b *Builder) ConcurrencyScope(concurrencyScope config.ConcurrencyScope) *Builder {
	b.concurrencyScope = concurrencyScope
	return b
}

// IsConcurrent 设置是否允许并发登录
func (b *Builder) IsConcurrent(concurrent bool) *Builder {
	b.isConcurrent = concurrent
	return b
}

// IsShare 设置是否共用同一个 Token
func (b *Builder) IsShare(share bool) *Builder {
	b.isShare = share
	return b
}

// MaxLoginCount 设置最大并发登录数量
func (b *Builder) MaxLoginCount(count int64) *Builder {
	b.maxLoginCount = count
	return b
}

// IsReadBody 设置是否从请求体读取 Token
func (b *Builder) IsReadBody(isRead bool) *Builder {
	b.isReadBody = isRead
	return b
}

// IsReadHeader 设置是否从 HTTP Header 读取 Token
func (b *Builder) IsReadHeader(isRead bool) *Builder {
	b.isReadHeader = isRead
	return b
}

// IsReadCookie 设置是否从 Cookie 读取 Token
func (b *Builder) IsReadCookie(isRead bool) *Builder {
	b.isReadCookie = isRead
	return b
}

// TokenStyle 设置 Token 生成风格
func (b *Builder) TokenStyle(style adapter.TokenStyle) *Builder {
	b.tokenStyle = style
	return b
}

// JwtSecretKey 设置 JWT 密钥
func (b *Builder) JwtSecretKey(key string) *Builder {
	b.jwtSecretKey = key
	return b
}

// IsLog 设置是否开启日志输出
func (b *Builder) IsLog(isLog bool) *Builder {
	b.isLog = isLog
	return b
}

// IsPrintBanner 设置是否打印启动 Banner
func (b *Builder) IsPrintBanner(isPrint bool) *Builder {
	b.isPrintBanner = isPrint
	return b
}

// AsyncEvent 设置是否异步触发事件
func (b *Builder) AsyncEvent(asyncEvent bool) *Builder {
	b.asyncEvent = asyncEvent
	return b
}

// CookieDomain 设置 Cookie 作用域（Domain）
func (b *Builder) CookieDomain(domain string) *Builder {
	if b.cookieConfig == nil {
		b.cookieConfig = &config.CookieConfig{}
	}
	b.cookieConfig.Domain = domain
	return b
}

// CookiePath 设置 Cookie 路径
func (b *Builder) CookiePath(path string) *Builder {
	if b.cookieConfig == nil {
		b.cookieConfig = &config.CookieConfig{}
	}
	b.cookieConfig.Path = path
	return b
}

// CookieSecure 设置 Cookie 是否仅在 HTTPS 下生效
func (b *Builder) CookieSecure(secure bool) *Builder {
	if b.cookieConfig == nil {
		b.cookieConfig = &config.CookieConfig{}
	}
	b.cookieConfig.Secure = secure
	return b
}

// CookieHttpOnly 设置 Cookie 是否禁止 JavaScript 访问
func (b *Builder) CookieHttpOnly(httpOnly bool) *Builder {
	if b.cookieConfig == nil {
		b.cookieConfig = &config.CookieConfig{}
	}
	b.cookieConfig.HttpOnly = httpOnly
	return b
}

// CookieSameSite 设置 Cookie SameSite 属性（Strict/Lax/None）
func (b *Builder) CookieSameSite(sameSite config.SameSiteMode) *Builder {
	if b.cookieConfig == nil {
		b.cookieConfig = &config.CookieConfig{}
	}
	b.cookieConfig.SameSite = sameSite
	return b
}

// CookieMaxAge 设置 Cookie 过期时间（单位：秒）
func (b *Builder) CookieMaxAge(maxAge int64) *Builder {
	if b.cookieConfig == nil {
		b.cookieConfig = &config.CookieConfig{}
	}
	b.cookieConfig.MaxAge = maxAge
	return b
}

// CookieConfig 设置完整的 Cookie 配置
func (b *Builder) CookieConfig(cfg *config.CookieConfig) *Builder {
	b.cookieConfig = cfg
	return b
}

// RenewPoolMinSize 设置续期协程池最小协程数
func (b *Builder) RenewPoolMinSize(size int) *Builder {
	if b.renewPoolConfig == nil {
		b.renewPoolConfig = &ants.RenewPoolConfig{}
	}
	b.renewPoolConfig.MinSize = size
	return b
}

// RenewPoolMaxSize 设置续期协程池最大协程数
func (b *Builder) RenewPoolMaxSize(size int) *Builder {
	if b.renewPoolConfig == nil {
		b.renewPoolConfig = &ants.RenewPoolConfig{}
	}
	b.renewPoolConfig.MaxSize = size
	return b
}

// RenewPoolScaleUpRate 设置协程池扩容触发比例
func (b *Builder) RenewPoolScaleUpRate(rate float64) *Builder {
	if b.renewPoolConfig == nil {
		b.renewPoolConfig = &ants.RenewPoolConfig{}
	}
	b.renewPoolConfig.ScaleUpRate = rate
	return b
}

// RenewPoolScaleDownRate 设置协程池缩容触发比例
func (b *Builder) RenewPoolScaleDownRate(rate float64) *Builder {
	if b.renewPoolConfig == nil {
		b.renewPoolConfig = &ants.RenewPoolConfig{}
	}
	b.renewPoolConfig.ScaleDownRate = rate
	return b
}

// RenewPoolCheckInterval 设置协程池扩缩容检查间隔
func (b *Builder) RenewPoolCheckInterval(interval time.Duration) *Builder {
	if b.renewPoolConfig == nil {
		b.renewPoolConfig = &ants.RenewPoolConfig{}
	}
	b.renewPoolConfig.CheckInterval = interval
	return b
}

// RenewPoolExpiry 设置协程池空闲协程过期时间
func (b *Builder) RenewPoolExpiry(duration time.Duration) *Builder {
	if b.renewPoolConfig == nil {
		b.renewPoolConfig = &ants.RenewPoolConfig{}
	}
	b.renewPoolConfig.Expiry = duration
	return b
}

// RenewPoolPrintStatusInterval 设置协程池状态打印间隔
func (b *Builder) RenewPoolPrintStatusInterval(interval time.Duration) *Builder {
	if b.renewPoolConfig == nil {
		b.renewPoolConfig = &ants.RenewPoolConfig{}
	}
	b.renewPoolConfig.PrintStatusInterval = interval
	return b
}

// RenewPoolPreAlloc 设置协程池是否预分配内存
func (b *Builder) RenewPoolPreAlloc(preAlloc bool) *Builder {
	if b.renewPoolConfig == nil {
		b.renewPoolConfig = &ants.RenewPoolConfig{}
	}
	b.renewPoolConfig.PreAlloc = preAlloc
	return b
}

// RenewPoolNonBlocking 设置协程池是否为非阻塞模式
func (b *Builder) RenewPoolNonBlocking(nonBlocking bool) *Builder {
	if b.renewPoolConfig == nil {
		b.renewPoolConfig = &ants.RenewPoolConfig{}
	}
	b.renewPoolConfig.NonBlocking = nonBlocking
	return b
}

// RenewPoolConfig 设置完整的续期协程池配置
func (b *Builder) RenewPoolConfig(cfg *ants.RenewPoolConfig) *Builder {
	b.renewPoolConfig = cfg
	return b
}

// LoggerPath 设置日志文件目录路径
func (b *Builder) LoggerPath(path string) *Builder {
	if b.logConfig == nil {
		b.logConfig = &dlog.LoggerConfig{}
	}
	b.logConfig.Path = path
	return b
}

// LoggerFileFormat 设置日志文件命名格式
func (b *Builder) LoggerFileFormat(format string) *Builder {
	if b.logConfig == nil {
		b.logConfig = &dlog.LoggerConfig{}
	}
	b.logConfig.FileFormat = format
	return b
}

// LoggerPrefix 设置日志行前缀
func (b *Builder) LoggerPrefix(prefix string) *Builder {
	if b.logConfig == nil {
		b.logConfig = &dlog.LoggerConfig{}
	}
	b.logConfig.Prefix = prefix
	return b
}

// LoggerLevel 设置日志最低输出级别
func (b *Builder) LoggerLevel(level dlog.LogLevel) *Builder {
	if b.logConfig == nil {
		b.logConfig = &dlog.LoggerConfig{}
	}
	b.logConfig.Level = level
	return b
}

// LoggerTimeFormat 设置日志时间戳格式
func (b *Builder) LoggerTimeFormat(format string) *Builder {
	if b.logConfig == nil {
		b.logConfig = &dlog.LoggerConfig{}
	}
	b.logConfig.TimeFormat = format
	return b
}

// LoggerStdout 设置是否将日志输出到控制台
func (b *Builder) LoggerStdout(stdout bool) *Builder {
	if b.logConfig == nil {
		b.logConfig = &dlog.LoggerConfig{}
	}
	b.logConfig.Stdout = stdout
	return b
}

// LoggerStdoutOnly 设置是否仅输出到控制台（不写入文件）
func (b *Builder) LoggerStdoutOnly(stdoutOnly bool) *Builder {
	if b.logConfig == nil {
		b.logConfig = &dlog.LoggerConfig{}
	}
	b.logConfig.StdoutOnly = stdoutOnly
	return b
}

// LoggerQueueSize 设置日志异步写入队列大小
func (b *Builder) LoggerQueueSize(size int) *Builder {
	if b.logConfig == nil {
		b.logConfig = &dlog.LoggerConfig{}
	}
	b.logConfig.QueueSize = size
	return b
}

// LoggerRotateSize 设置日志文件滚动大小阈值（字节）
func (b *Builder) LoggerRotateSize(size int64) *Builder {
	if b.logConfig == nil {
		b.logConfig = &dlog.LoggerConfig{}
	}
	b.logConfig.RotateSize = size
	return b
}

// LoggerRotateExpire 设置日志文件按时间滚动间隔
func (b *Builder) LoggerRotateExpire(expire time.Duration) *Builder {
	if b.logConfig == nil {
		b.logConfig = &dlog.LoggerConfig{}
	}
	b.logConfig.RotateExpire = expire
	return b
}

// LoggerRotateBackupLimit 设置日志备份文件最大数量
func (b *Builder) LoggerRotateBackupLimit(limit int) *Builder {
	if b.logConfig == nil {
		b.logConfig = &dlog.LoggerConfig{}
	}
	b.logConfig.RotateBackupLimit = limit
	return b
}

// LoggerRotateBackupDays 设置日志备份文件保留天数
func (b *Builder) LoggerRotateBackupDays(days int) *Builder {
	if b.logConfig == nil {
		b.logConfig = &dlog.LoggerConfig{}
	}
	b.logConfig.RotateBackupDays = days
	return b
}

// LoggerConfig 设置完整的日志配置
func (b *Builder) LoggerConfig(cfg *dlog.LoggerConfig) *Builder {
	b.logConfig = cfg
	return b
}

// SetGenerator 设置 Token 生成器
func (b *Builder) SetGenerator(generator adapter.Generator) *Builder {
	b.generator = generator
	return b
}

// SetStorage 设置存储适配器
func (b *Builder) SetStorage(storage adapter.Storage) *Builder {
	b.storage = storage
	return b
}

// SetCodec 设置编解码器适配器
func (b *Builder) SetCodec(codec adapter.Codec) *Builder {
	b.codec = codec
	return b
}

// SetLog 设置日志适配器
func (b *Builder) SetLog(log adapter.Log) *Builder {
	b.log = log
	return b
}

// SetPool 设置异步任务协程池
func (b *Builder) SetPool(pool adapter.Pool) *Builder {
	b.pool = pool
	return b
}

// SetCustomPermissionListFunc 设置自定义权限列表获取函数
func (b *Builder) SetCustomPermissionListFunc(f func(loginID, authType string) ([]string, error)) *Builder {
	b.customPermissionListFunc = f
	return b
}

// SetCustomRoleListFunc 设置自定义角色列表获取函数
func (b *Builder) SetCustomRoleListFunc(f func(loginID, authType string) ([]string, error)) *Builder {
	b.customRoleListFunc = f
	return b
}

// SetCustomPermissionListExtFunc 设置自定义权限列表获取函数（扩展版本，支持设备信息）
func (b *Builder) SetCustomPermissionListExtFunc(f func(loginID, device, deviceId, authType string) ([]string, error)) *Builder {
	b.customPermissionListExtFunc = f
	return b
}

// SetCustomRoleListExtFunc 设置自定义角色列表获取函数（扩展版本，支持设备信息）
func (b *Builder) SetCustomRoleListExtFunc(f func(loginID, device, deviceId, authType string) ([]string, error)) *Builder {
	b.customRoleListExtFunc = f
	return b
}

// JwtSecret 设置为 JWT 模式并指定密钥
func (b *Builder) JwtSecret(secret string) *Builder {
	b.tokenStyle = adapter.TokenStyleJWT
	b.jwtSecretKey = secret
	return b
}

// Clone 克隆当前构建器（深拷贝可变字段）
func (b *Builder) Clone() *Builder {
	clone := *b
	if b.cookieConfig != nil {
		cookieCopy := *b.cookieConfig
		clone.cookieConfig = &cookieCopy
	}
	if b.renewPoolConfig != nil {
		poolCopy := *b.renewPoolConfig
		clone.renewPoolConfig = &poolCopy
	}
	if b.logConfig != nil {
		logCopy := *b.logConfig
		clone.logConfig = &logCopy
	}
	return &clone
}

// Build 构建 Manager 实例并打印启动 Banner
func (b *Builder) Build() *manager.Manager {
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
		panic("Build Manager Invalid config err: " + err.Error())
	}

	if b.generator == nil {
		b.generator = dgenerator.NewGenerator(b.timeout, b.jwtSecretKey, b.tokenStyle)
	}
	if b.storage == nil {
		b.storage = memory.NewStorage()
	}
	if b.codec == nil {
		b.codec = djson.NewJSONSerializer()
	}

	if b.isLog {
		if b.log == nil {
			if b.logConfig == nil {
				b.logConfig = dlog.DefaultLoggerConfig()
			}
			b.log, err = dlog.NewLoggerWithConfig(b.logConfig)
			if err != nil {
				panic("Build Manager Invalid LoggerConfig err: " + err.Error())
			}
		}
	} else {
		b.log = nop.NewNopLogger()
	}

	if b.autoRenew {
		if b.pool == nil {
			if b.renewPoolConfig == nil {
				b.renewPoolConfig = ants.DefaultRenewPoolConfig()
			}
			err = b.renewPoolConfig.Validate()
			if err != nil {
				panic("Build Manager Invalid RenewPoolConfig err: " + err.Error())
			}
			b.pool, err = ants.NewRenewPoolManagerWithConfig(b.renewPoolConfig)
			if err != nil {
				panic("Build Manager NewRenewPoolManagerWithConfig err: " + err.Error())
			}
		}

		if b.renewPoolConfig.PrintStatusInterval > 0 {
			ticker := time.NewTicker(b.renewPoolConfig.PrintStatusInterval)
			go func() {
				defer ticker.Stop()
				for {
					select {
					case <-ticker.C:
						running, capacity, usage := b.pool.Stats()
						b.log.Infof(
							"RenewPool Status: Capacity=%d, Running=%d, Usage=%.2f%%",
							capacity, running, usage*100,
						)
					}
				}
			}()
		}
	}

	if b.isPrintBanner {
		banner.PrintBanner(cfg)
	}

	return manager.NewManager(cfg, b.generator, b.storage, b.codec, b.log, b.pool, b.customPermissionListFunc, b.customRoleListFunc, b.customPermissionListExtFunc, b.customRoleListExtFunc)
}
