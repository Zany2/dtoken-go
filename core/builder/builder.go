package builder

import (
	"fmt"
	"strings"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/banner"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
)

// GeneratorFactory creates a token generator from config.
type GeneratorFactory func(cfg *config.Config) (adapter.Generator, error)

// StorageFactory creates a storage adapter from config.
type StorageFactory func(cfg *config.Config) (adapter.Storage, error)

// CodecFactory creates a codec adapter from config.
type CodecFactory func(cfg *config.Config) (adapter.Codec, error)

// LogFactory creates a logger adapter from config.
type LogFactory func(cfg *config.Config) (adapter.Log, error)

// PoolFactory creates an async task pool from config.
type PoolFactory func(cfg *config.Config) (adapter.Pool, error)

// Components groups pluggable runtime components.
type Components struct {
	Generator adapter.Generator
	Storage   adapter.Storage
	Codec     adapter.Codec
	Log       adapter.Log
	Pool      adapter.Pool
}

// ComponentFactories groups default component factories.
type ComponentFactories struct {
	Generator GeneratorFactory
	Storage   StorageFactory
	Codec     CodecFactory
	Log       LogFactory
	Pool      PoolFactory
}

// Builder builds a Manager from config and replaceable components.
type Builder struct {
	cfg        *config.Config
	components Components
	factories  ComponentFactories

	accessProvider manager.AccessProvider

	customPermissionListFunc    func(loginID, authType string) ([]string, error)
	customRoleListFunc          func(loginID, authType string) ([]string, error)
	customPermissionListExtFunc func(loginID, device, deviceId, authType string) ([]string, error)
	customRoleListExtFunc       func(loginID, device, deviceId, authType string) ([]string, error)
}

// NewBuilder creates a builder with the default config.
func NewBuilder() *Builder {
	return &Builder{cfg: config.DefaultConfig()}
}

// Config replaces the builder config with a clone of cfg.
func (b *Builder) Config(cfg *config.Config) *Builder {
	if cfg == nil {
		b.cfg = config.DefaultConfig()
		return b
	}
	b.cfg = cfg.Clone()
	return b
}

// GetConfig returns the mutable builder config.
func (b *Builder) GetConfig() *config.Config {
	b.ensureConfig()
	return b.cfg
}

// AuthType sets auth type with suffix fix.
func (b *Builder) AuthType(authType string) *Builder {
	b.ensureConfig()
	if authType == "" {
		b.cfg.AuthType = config.DefaultAuthType
	} else if !strings.HasSuffix(authType, ":") {
		b.cfg.AuthType = authType + ":"
	} else {
		b.cfg.AuthType = authType
	}
	return b
}

// KeyPrefix sets key prefix with suffix fix.
func (b *Builder) KeyPrefix(keyPrefix string) *Builder {
	b.ensureConfig()
	if keyPrefix == "" {
		b.cfg.KeyPrefix = config.DefaultKeyPrefix
	} else if !strings.HasSuffix(keyPrefix, ":") {
		b.cfg.KeyPrefix = keyPrefix + ":"
	} else {
		b.cfg.KeyPrefix = keyPrefix
	}
	return b
}

// TokenName sets token name.
func (b *Builder) TokenName(name string) *Builder {
	b.ensureConfig()
	b.cfg.TokenName = name
	return b
}

// Timeout sets timeout seconds.
func (b *Builder) Timeout(seconds int64) *Builder {
	b.ensureConfig()
	b.cfg.Timeout = seconds
	return b
}

// TimeoutDuration sets timeout by duration.
func (b *Builder) TimeoutDuration(d time.Duration) *Builder {
	b.ensureConfig()
	b.cfg.Timeout = int64(d.Seconds())
	return b
}

// AutoRenew sets auto renew switch.
func (b *Builder) AutoRenew(autoRenew bool) *Builder {
	b.ensureConfig()
	b.cfg.AutoRenew = autoRenew
	return b
}

// RenewMaxRefresh sets renew trigger threshold.
func (b *Builder) RenewMaxRefresh(seconds int64) *Builder {
	b.ensureConfig()
	b.cfg.RenewMaxRefresh = seconds
	return b
}

// RenewInterval sets minimum renew interval.
func (b *Builder) RenewInterval(seconds int64) *Builder {
	b.ensureConfig()
	b.cfg.RenewInterval = seconds
	return b
}

// ActiveTimeout sets max inactive duration.
func (b *Builder) ActiveTimeout(seconds int64) *Builder {
	b.ensureConfig()
	b.cfg.ActiveTimeout = seconds
	return b
}

// ConcurrencyScope sets concurrency scope.
func (b *Builder) ConcurrencyScope(concurrencyScope config.ConcurrencyScope) *Builder {
	b.ensureConfig()
	b.cfg.ConcurrencyScope = concurrencyScope
	return b
}

// IsConcurrent sets concurrent login switch.
func (b *Builder) IsConcurrent(concurrent bool) *Builder {
	b.ensureConfig()
	b.cfg.IsConcurrent = concurrent
	return b
}

// IsShare sets shared token switch.
func (b *Builder) IsShare(share bool) *Builder {
	b.ensureConfig()
	b.cfg.IsShare = share
	return b
}

// MaxLoginCount sets max login count.
func (b *Builder) MaxLoginCount(count int64) *Builder {
	b.ensureConfig()
	b.cfg.MaxLoginCount = count
	return b
}

// IsReadBody sets body read switch.
func (b *Builder) IsReadBody(isRead bool) *Builder {
	b.ensureConfig()
	b.cfg.IsReadBody = isRead
	return b
}

// IsReadHeader sets header read switch.
func (b *Builder) IsReadHeader(isRead bool) *Builder {
	b.ensureConfig()
	b.cfg.IsReadHeader = isRead
	return b
}

// IsReadCookie sets cookie read switch.
func (b *Builder) IsReadCookie(isRead bool) *Builder {
	b.ensureConfig()
	b.cfg.IsReadCookie = isRead
	return b
}

// TokenStyle sets token style.
func (b *Builder) TokenStyle(style adapter.TokenStyle) *Builder {
	b.ensureConfig()
	b.cfg.TokenStyle = style
	return b
}

// JwtSecretKey sets JWT secret key.
func (b *Builder) JwtSecretKey(key string) *Builder {
	b.ensureConfig()
	b.cfg.JwtSecretKey = key
	return b
}

// IsLog sets log switch.
func (b *Builder) IsLog(isLog bool) *Builder {
	b.ensureConfig()
	b.cfg.IsLog = isLog
	return b
}

// IsPrintBanner sets banner print switch.
func (b *Builder) IsPrintBanner(isPrint bool) *Builder {
	b.ensureConfig()
	b.cfg.IsPrintBanner = isPrint
	return b
}

// AsyncEvent sets async event switch.
func (b *Builder) AsyncEvent(asyncEvent bool) *Builder {
	b.ensureConfig()
	b.cfg.AsyncEvent = asyncEvent
	return b
}

// CookieDomain sets cookie domain.
func (b *Builder) CookieDomain(domain string) *Builder {
	cc := b.ensureCookieConfig()
	cc.Domain = domain
	return b
}

// CookiePath sets cookie path.
func (b *Builder) CookiePath(path string) *Builder {
	cc := b.ensureCookieConfig()
	cc.Path = path
	return b
}

// CookieSecure sets cookie secure switch.
func (b *Builder) CookieSecure(secure bool) *Builder {
	cc := b.ensureCookieConfig()
	cc.Secure = secure
	return b
}

// CookieHttpOnly sets cookie httpOnly switch.
func (b *Builder) CookieHttpOnly(httpOnly bool) *Builder {
	cc := b.ensureCookieConfig()
	cc.HttpOnly = httpOnly
	return b
}

// CookieSameSite sets cookie sameSite mode.
func (b *Builder) CookieSameSite(sameSite config.SameSiteMode) *Builder {
	cc := b.ensureCookieConfig()
	cc.SameSite = sameSite
	return b
}

// CookieMaxAge sets cookie max age seconds.
func (b *Builder) CookieMaxAge(maxAge int64) *Builder {
	cc := b.ensureCookieConfig()
	cc.MaxAge = maxAge
	return b
}

// CookieConfig sets full cookie config.
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

// SetGenerator sets token generator.
func (b *Builder) SetGenerator(generator adapter.Generator) *Builder {
	b.components.Generator = generator
	return b
}

// SetStorage sets storage adapter.
func (b *Builder) SetStorage(storage adapter.Storage) *Builder {
	b.components.Storage = storage
	return b
}

// SetCodec sets codec adapter.
func (b *Builder) SetCodec(codec adapter.Codec) *Builder {
	b.components.Codec = codec
	return b
}

// SetLog sets log adapter.
func (b *Builder) SetLog(log adapter.Log) *Builder {
	b.components.Log = log
	return b
}

// SetPool sets async task pool.
func (b *Builder) SetPool(pool adapter.Pool) *Builder {
	b.components.Pool = pool
	return b
}

// SetGeneratorFactory sets default generator factory.
func (b *Builder) SetGeneratorFactory(factory GeneratorFactory) *Builder {
	b.factories.Generator = factory
	return b
}

// SetStorageFactory sets default storage factory.
func (b *Builder) SetStorageFactory(factory StorageFactory) *Builder {
	b.factories.Storage = factory
	return b
}

// SetCodecFactory sets default codec factory.
func (b *Builder) SetCodecFactory(factory CodecFactory) *Builder {
	b.factories.Codec = factory
	return b
}

// SetLogFactory sets default log factory.
func (b *Builder) SetLogFactory(factory LogFactory) *Builder {
	b.factories.Log = factory
	return b
}

// SetPoolFactory sets default pool factory.
func (b *Builder) SetPoolFactory(factory PoolFactory) *Builder {
	b.factories.Pool = factory
	return b
}

// SetAccessProvider sets the permission and role provider.
func (b *Builder) SetAccessProvider(provider manager.AccessProvider) *Builder {
	b.accessProvider = provider
	return b
}

// SetCustomPermissionListFunc sets permission callback.
func (b *Builder) SetCustomPermissionListFunc(f func(loginID, authType string) ([]string, error)) *Builder {
	b.customPermissionListFunc = f
	return b
}

// SetCustomRoleListFunc sets role callback.
func (b *Builder) SetCustomRoleListFunc(f func(loginID, authType string) ([]string, error)) *Builder {
	b.customRoleListFunc = f
	return b
}

// SetCustomPermissionListExtFunc sets extended permission callback.
func (b *Builder) SetCustomPermissionListExtFunc(f func(loginID, device, deviceId, authType string) ([]string, error)) *Builder {
	b.customPermissionListExtFunc = f
	return b
}

// SetCustomRoleListExtFunc sets extended role callback.
func (b *Builder) SetCustomRoleListExtFunc(f func(loginID, device, deviceId, authType string) ([]string, error)) *Builder {
	b.customRoleListExtFunc = f
	return b
}

// JwtSecret enables JWT style and sets secret.
func (b *Builder) JwtSecret(secret string) *Builder {
	b.ensureConfig()
	b.cfg.TokenStyle = adapter.TokenStyleJWT
	b.cfg.JwtSecretKey = secret
	return b
}

// Clone clones builder with deep copy.
func (b *Builder) Clone() *Builder {
	clone := *b
	if b.cfg != nil {
		clone.cfg = b.cfg.Clone()
	}
	return &clone
}

// Build builds manager and returns configuration errors.
func (b *Builder) Build() (*manager.Manager, error) {
	b.ensureConfig()
	cfg := b.cfg.Clone()
	if cfg.CookieConfig == nil {
		cfg.CookieConfig = config.DefaultCookieConfig()
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("build manager invalid config: %w", err)
	}

	if b.components.Generator == nil {
		if b.factories.Generator != nil {
			generator, err := b.factories.Generator(cfg)
			if err != nil {
				return nil, fmt.Errorf("build manager create generator: %w", err)
			}
			b.components.Generator = generator
		}
		if b.components.Generator == nil {
			return nil, fmt.Errorf("build manager missing generator: call SetGenerator or SetGeneratorFactory")
		}
	}

	if b.components.Storage == nil {
		if b.factories.Storage != nil {
			storage, err := b.factories.Storage(cfg)
			if err != nil {
				return nil, fmt.Errorf("build manager create storage: %w", err)
			}
			b.components.Storage = storage
		}
		if b.components.Storage == nil {
			return nil, fmt.Errorf("build manager missing storage: call SetStorage or SetStorageFactory")
		}
	}
	if _, ok := b.components.Storage.(adapter.AtomicStorage); !ok {
		return nil, fmt.Errorf("%w: nonce verification requires adapter.AtomicStorage", derror.ErrStorageCapabilityUnsupported)
	}

	if b.components.Codec == nil {
		if b.factories.Codec != nil {
			codec, err := b.factories.Codec(cfg)
			if err != nil {
				return nil, fmt.Errorf("build manager create codec: %w", err)
			}
			b.components.Codec = codec
		}
		if b.components.Codec == nil {
			return nil, fmt.Errorf("build manager missing codec: call SetCodec or SetCodecFactory")
		}
	}

	if cfg.IsLog {
		if b.components.Log == nil {
			if b.factories.Log != nil {
				logger, err := b.factories.Log(cfg)
				if err != nil {
					return nil, fmt.Errorf("build manager create logger: %w", err)
				}
				b.components.Log = logger
			}
			if b.components.Log == nil {
				return nil, fmt.Errorf("build manager missing logger: call SetLog or SetLogFactory")
			}
		}
	} else {
		b.components.Log = adapter.NewNopLogger()
	}

	if cfg.AutoRenew && b.components.Pool == nil && b.factories.Pool != nil {
		pool, err := b.factories.Pool(cfg)
		if err != nil {
			return nil, fmt.Errorf("build manager create renew pool: %w", err)
		}
		b.components.Pool = pool
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
		b.components.Generator,
		b.components.Storage,
		b.components.Codec,
		b.components.Log,
		b.components.Pool,
		accessProvider,
	)

	return mgr, nil
}

// MustBuild builds manager and panics on error.
func (b *Builder) MustBuild() *manager.Manager {
	mgr, err := b.Build()
	if err != nil {
		panic(err)
	}
	return mgr
}

func (b *Builder) ensureConfig() {
	if b.cfg == nil {
		b.cfg = config.DefaultConfig()
	}
}

func (b *Builder) ensureCookieConfig() *config.CookieConfig {
	b.ensureConfig()
	if b.cfg.CookieConfig == nil {
		b.cfg.CookieConfig = config.DefaultCookieConfig()
	}
	return b.cfg.CookieConfig
}
