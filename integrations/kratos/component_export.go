// @Author daixk 2025/12/22 15:56:00
package kratos

import (
	base64codec "github.com/Zany2/dtoken-go/com/codec/base64"
	jsoncodec "github.com/Zany2/dtoken-go/com/codec/json"
	msgpackcodec "github.com/Zany2/dtoken-go/com/codec/msgpack"
	"github.com/Zany2/dtoken-go/com/generator/dgenerator"
	"github.com/Zany2/dtoken-go/com/log/dlog"
	"github.com/Zany2/dtoken-go/com/log/nop"
	"github.com/Zany2/dtoken-go/com/pool/ants"
	memorystorage "github.com/Zany2/dtoken-go/com/storage/memory"
	redisstorage "github.com/Zany2/dtoken-go/com/storage/redis"
	corepkg "github.com/Zany2/dtoken-go/core"
	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/builder"
	"github.com/Zany2/dtoken-go/core/config"
	corecontext "github.com/Zany2/dtoken-go/core/context"
	"github.com/Zany2/dtoken-go/core/listener"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/core/nonce"
	"github.com/Zany2/dtoken-go/core/oauth2"
	"github.com/Zany2/dtoken-go/dtoken"
)

// Core aliases keep framework imports self-contained Core 鍒悕璁╂鏋跺寘鍙嫭绔嬩娇鐢?
type (
	Builder                    = builder.Builder
	DTokenBuilder              = dtoken.Builder
	GeneratorFactory           = builder.GeneratorFactory
	StorageFactory             = builder.StorageFactory
	CodecFactory               = builder.CodecFactory
	LogFactory                 = builder.LogFactory
	PoolFactory                = builder.PoolFactory
	Components                 = builder.Components
	ComponentFactories         = builder.ComponentFactories
	Config                     = config.Config
	CookieConfig               = config.CookieConfig
	SameSiteMode               = config.SameSiteMode
	ConcurrencyScope           = config.ConcurrencyScope
	ReplacedLoginExitMode      = config.ReplacedLoginExitMode
	ReplacedLoginMode          = config.ReplacedLoginExitMode
	LogoutMode                 = config.LogoutMode
	Manager                    = manager.Manager
	ManagerOption              = manager.Option
	TokenInfo                  = manager.TokenInfo
	TokenIntrospection         = manager.TokenIntrospection
	ManagerRefreshTokenOptions = manager.RefreshTokenOptions
	RefreshTokenPair           = manager.RefreshTokenPair
	RefreshTokenInfo           = manager.RefreshTokenInfo
	Session                    = manager.Session
	TerminalInfo               = manager.TerminalInfo
	DisableInfo                = manager.DisableInfo
	ServiceDisableInfo         = manager.ServiceDisableInfo
	DeviceDisableInfo          = manager.DeviceDisableInfo
	TokenState                 = manager.TokenState
	TerminalRemovalFunc        = manager.TerminalRemovalFunc
	TerminalVisitor            = manager.TerminalVisitor
	AccessSubject              = manager.AccessSubject
	AccessProvider             = manager.AccessProvider
	AccessProviderFunc         = manager.AccessProviderFunc
	Generator                  = adapter.Generator
	TokenStyle                 = adapter.TokenStyle
	Storage                    = adapter.Storage
	AtomicStorage              = adapter.AtomicStorage
	ScannerStorage             = adapter.ScannerStorage
	AdminStorage               = adapter.AdminStorage
	FullStorage                = adapter.FullStorage
	Codec                      = adapter.Codec
	Log                        = adapter.Log
	LogLevel                   = adapter.LogLevel
	LogControl                 = adapter.LogControl
	Pool                       = adapter.Pool
	CookieOptions              = adapter.CookieOptions
	RequestContext             = adapter.RequestContext
	RequestContextExt          = adapter.RequestContextExt
	Event                      = listener.Event
	EventData                  = listener.EventData
	Listener                   = listener.Listener
	ListenerFunc               = listener.ListenerFunc
	ListenerConfig             = listener.ListenerConfig
	EventFilter                = listener.EventFilter
	EventStats                 = listener.EventStats
	EventManager               = listener.Manager
	NonceConfig                = nonce.Config
	NonceManager               = nonce.NonceManager
	OAuth2Config               = oauth2.Config
	OAuth2Client               = oauth2.Client
	AuthorizationCode          = oauth2.AuthorizationCode
	AccessToken                = oauth2.AccessToken
	TokenRequest               = oauth2.TokenRequest
	UserValidator              = oauth2.UserValidator
	OAuth2Server               = oauth2.OAuth2Server
	GrantType                  = oauth2.GrantType
	RedisConfig                = redisstorage.Config
	RedisStorage               = redisstorage.Storage
	MemoryStorage              = memorystorage.Storage
	JSONSerializer             = jsoncodec.JSONSerializer
	Base64Serializer           = base64codec.Base64Serializer
	MsgPackSerializer          = msgpackcodec.MsgPackSerializer
	DefaultGenerator           = dgenerator.Generator
	LoggerConfig               = dlog.LoggerConfig
	Logger                     = dlog.Logger
	RenewPoolConfig            = ants.RenewPoolConfig
	RenewPoolManager           = ants.RenewPoolManager
	NopLogger                  = nop.NopLogger
)

// Common constants forward core configuration values Common 甯搁噺杞彂鏍稿績閰嶇疆鍊?
const (
	Version                        = corepkg.Version
	SameSiteStrict                 = config.SameSiteStrict
	SameSiteLax                    = config.SameSiteLax
	SameSiteNone                   = config.SameSiteNone
	ConcurrencyScopeAccount        = config.ConcurrencyScopeAccount
	ConcurrencyScopeDevice         = config.ConcurrencyScopeDevice
	ReplacedLoginModeOldDevice     = config.ReplacedLoginExitModeOldDevice
	ReplacedLoginModeNewDevice     = config.ReplacedLoginExitModeNewDevice
	ReplacedLoginExitModeOldDevice = config.ReplacedLoginExitModeOldDevice
	ReplacedLoginExitModeNewDevice = config.ReplacedLoginExitModeNewDevice
	LogoutModeLogout               = config.LogoutModeLogout
	LogoutModeKickout              = config.LogoutModeKickout
	LogoutModeReplaced             = config.LogoutModeReplaced
	DefaultTokenName               = config.DefaultTokenName
	DefaultKeyPrefix               = config.DefaultKeyPrefix
	DefaultAuthType                = config.DefaultAuthType
	TokenKeyPrefix                 = config.TokenKeyPrefix
	DefaultTimeout                 = config.DefaultTimeout
	DefaultRefreshTokenTimeout     = config.DefaultRefreshTokenTimeout
	DefaultJWTSecretKey            = config.DefaultJWTSecretKey
	DefaultMaxLoginCount           = config.DefaultMaxLoginCount
	DefaultCookiePath              = config.DefaultCookiePath
	NoLimit                        = config.NoLimit
	DefaultNonceTTL                = nonce.DefaultNonceTTL
	NonceLength                    = nonce.NonceLength
	NonceKeySuffix                 = nonce.NonceKeySuffix
	DefaultAccessTTL               = nonce.DefaultAccessTTL
	TokenStyleUUID                 = adapter.TokenStyleUUID
	TokenStyleSimple               = adapter.TokenStyleSimple
	TokenStyleRandom32             = adapter.TokenStyleRandom32
	TokenStyleRandom64             = adapter.TokenStyleRandom64
	TokenStyleRandom128            = adapter.TokenStyleRandom128
	TokenStyleJWT                  = adapter.TokenStyleJWT
	TokenStyleHash                 = adapter.TokenStyleHash
	TokenStyleTimestamp            = adapter.TokenStyleTimestamp
	TokenStyleTik                  = adapter.TokenStyleTik
	LogLevelDebug                  = adapter.LogLevelDebug
	LogLevelInfo                   = adapter.LogLevelInfo
	LogLevelWarn                   = adapter.LogLevelWarn
	LogLevelError                  = adapter.LogLevelError
	EventLogin                     = listener.EventLogin
	EventLogout                    = listener.EventLogout
	EventKickout                   = listener.EventKickout
	EventReplace                   = listener.EventReplace
	EventDisable                   = listener.EventDisable
	EventUntie                     = listener.EventUntie
	EventRenew                     = listener.EventRenew
	EventCreateSession             = listener.EventCreateSession
	EventDestroySession            = listener.EventDestroySession
	EventPermissionCheck           = listener.EventPermissionCheck
	EventPermissionChange          = listener.EventPermissionChange
	EventRoleCheck                 = listener.EventRoleCheck
	EventRoleChange                = listener.EventRoleChange
	EventDisableService            = listener.EventDisableService
	EventUntieService              = listener.EventUntieService
	EventDisableDevice             = listener.EventDisableDevice
	EventUntieDevice               = listener.EventUntieDevice
	EventAll                       = listener.EventAll
	ExtraKeyPermission             = listener.ExtraKeyPermission
	ExtraKeyPermissions            = listener.ExtraKeyPermissions
	ExtraKeyRole                   = listener.ExtraKeyRole
	ExtraKeyRoles                  = listener.ExtraKeyRoles
	ExtraKeyLogic                  = listener.ExtraKeyLogic
	ExtraKeyResult                 = listener.ExtraKeyResult
	ExtraKeyAction                 = listener.ExtraKeyAction
	ExtraKeyShared                 = listener.ExtraKeyShared
	ExtraKeyService                = listener.ExtraKeyService
	ExtraKeyLevel                  = listener.ExtraKeyLevel
	EventLogicAnd                  = listener.LogicAnd
	EventLogicOr                   = listener.LogicOr
	ActionAdd                      = listener.ActionAdd
	ActionRemove                   = listener.ActionRemove
	DisableKeyPrefix               = manager.DisableKeyPrefix
	DisableServiceKeyPrefix        = manager.DisableServiceKeyPrefix
	DisableDeviceKeyPrefix         = manager.DisableDeviceKeyPrefix
	DisableDeviceIDKeyPrefix       = manager.DisableDeviceIDKeyPrefix
	SessionKeyPrefix               = manager.SessionKeyPrefix
	RenewKeyPrefix                 = manager.RenewKeyPrefix
	ActivePrefix                   = manager.ActivePrefix
	RefreshTokenKeyPrefix          = manager.RefreshTokenKeyPrefix
	TokenRefreshKeyPrefix          = manager.TokenRefreshKeyPrefix
	SessionKeyLoginID              = manager.SessionKeyLoginID
	SessionKeyDevice               = manager.SessionKeyDevice
	SessionKeyLoginTime            = manager.SessionKeyLoginTime
	SessionKeyPermissions          = manager.SessionKeyPermissions
	SessionKeyRoles                = manager.SessionKeyRoles
	PermissionWildcard             = manager.PermissionWildcard
	PermissionSeparator            = manager.PermissionSeparator
	TokenStateLogout               = manager.TokenStateLogout
	TokenStateKickOut              = manager.TokenStateKickOut
	TokenStateReplaced             = manager.TokenStateReplaced
	TokenStateActiveTimeout        = manager.TokenStateActiveTimeout
	DefaultCodeExpiration          = oauth2.DefaultCodeExpiration
	DefaultTokenExpiration         = oauth2.DefaultTokenExpiration
	DefaultRefreshTTL              = oauth2.DefaultRefreshTTL
	CodeLength                     = oauth2.CodeLength
	AccessTokenLength              = oauth2.AccessTokenLength
	RefreshTokenLength             = oauth2.RefreshTokenLength
	CodeKeySuffix                  = oauth2.CodeKeySuffix
	OAuth2TokenKeySuffix           = oauth2.TokenKeySuffix
	OAuth2RefreshKeySuffix         = oauth2.RefreshKeySuffix
	ClientKeySuffix                = oauth2.ClientKeySuffix
	GrantTypeAuthorizationCode     = oauth2.GrantTypeAuthorizationCode
	GrantTypeRefreshToken          = oauth2.GrantTypeRefreshToken
	GrantTypeClientCredentials     = oauth2.GrantTypeClientCredentials
	GrantTypePassword              = oauth2.GrantTypePassword
	TokenTypeBearer                = oauth2.TokenTypeBearer
	TTLNoExpire                    = adapter.TTLNoExpire
	TTLNotFound                    = adapter.TTLNotFound
)

// Component constructors forward bundled implementations Component 鏋勯€犲櫒杞彂鍐呯疆瀹炵幇
var (
	NewDTokenBuilder               = dtoken.NewBuilder
	BuildAndSetManager             = dtoken.BuildAndSetManager
	DefaultConfig                  = config.DefaultConfig
	DefaultCookieConfig            = config.DefaultCookieConfig
	NewContext                     = corecontext.NewContext
	NewManager                     = manager.NewManager
	WithNonceManager               = manager.WithNonceManager
	WithOAuth2Manager              = manager.WithOAuth2Manager
	NewLegacyAccessProvider        = manager.NewLegacyAccessProvider
	NewEventManager                = listener.NewManager
	DefaultNonceConfig             = nonce.DefaultConfig
	NewDefaultNonceManager         = nonce.NewDefaultNonceManager
	NewNonceManagerWithConfig      = nonce.NewNonceManagerWithConfig
	NewNonceManager                = nonce.NewNonceManager
	DefaultOAuth2Config            = oauth2.DefaultConfig
	NewDefaultOAuth2Server         = oauth2.NewDefaultOAuth2Server
	NewOAuth2ServerWithConfig      = oauth2.NewOAuth2ServerWithConfig
	NewOAuth2Server                = oauth2.NewOAuth2Server
	NewRedisStorage                = redisstorage.NewStorage
	NewRedisStorageFromConfig      = redisstorage.NewStorageFromConfig
	NewRedisStorageFromClient      = redisstorage.NewStorageFromClient
	NewStorageFromConfig           = redisstorage.NewStorageFromConfig
	NewStorageFromClient           = redisstorage.NewStorageFromClient
	NewMemoryStorage               = memorystorage.NewStorage
	NewJSONSerializer              = jsoncodec.NewJSONSerializer
	NewBase64Serializer            = base64codec.NewBase64Serializer
	NewMsgPackSerializer           = msgpackcodec.NewMsgPackSerializer
	NewDefaultTokenGenerator       = dgenerator.NewDefaultGenerator
	NewTokenGenerator              = dgenerator.NewGenerator
	DefaultLoggerConfig            = dlog.DefaultLoggerConfig
	NewLoggerWithConfig            = dlog.NewLoggerWithConfig
	DefaultRenewPoolConfig         = ants.DefaultRenewPoolConfig
	NewRenewPoolManagerWithDefault = ants.NewRenewPoolManagerWithDefaultConfig
	NewRenewPoolManagerWithConfig  = ants.NewRenewPoolManagerWithConfig
	NewNopLogger                   = nop.NewNopLogger
	NewAdapterNopLogger            = adapter.NewNopLogger
	ErrRedisKeyNotFound            = redisstorage.ErrKeyNotFound
	ErrMemoryKeyNotFound           = memorystorage.ErrKeyNotFound
)
