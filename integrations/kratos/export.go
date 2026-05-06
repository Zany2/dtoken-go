package kratos

import (
	"context"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/builder"
	"github.com/Zany2/dtoken-go/core/config"
	corecontext "github.com/Zany2/dtoken-go/core/context"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/core/oauth2"
	"github.com/Zany2/dtoken-go/dtoken"
)

// Re-exported types Re-exported types 重导出类型
type (
	Config = config.Config

	CookieConfig = config.CookieConfig

	SameSiteMode = config.SameSiteMode

	ConcurrencyScope = config.ConcurrencyScope

	Manager = manager.Manager

	DTokenContext = corecontext.DTokenContext

	TokenInfo = manager.TokenInfo

	DisableInfo = manager.DisableInfo

	ServiceDisableInfo = manager.ServiceDisableInfo

	Session = manager.Session

	TerminalInfo = manager.TerminalInfo

	TerminalVisitor = manager.TerminalVisitor

	TokenState = manager.TokenState

	Builder = builder.Builder

	DTokenError = derror.DTokenError

	TokenStyle = adapter.TokenStyle

	Storage = adapter.Storage

	RequestContext = adapter.RequestContext

	RequestContextExt = adapter.RequestContextExt

	CookieOptions = adapter.CookieOptions

	Codec = adapter.Codec

	Generator = adapter.Generator

	Log = adapter.Log

	LogControl = adapter.LogControl

	LogLevel = adapter.LogLevel

	Pool = adapter.Pool

	OAuth2Server = oauth2.OAuth2Server

	Client = oauth2.Client

	AuthorizationCode = oauth2.AuthorizationCode

	AccessToken = oauth2.AccessToken

	TokenRequest = oauth2.TokenRequest

	UserValidator = oauth2.UserValidator

	GrantType = oauth2.GrantType
)

// Error codes Error codes 错误码
const (
	CodeSuccess = derror.CodeSuccess

	CodeBadRequest = derror.CodeBadRequest

	CodeNotLogin = derror.CodeNotLogin

	CodePermissionDenied = derror.CodePermissionDenied

	CodeNotFound = derror.CodeNotFound

	CodeServerError = derror.CodeServerError

	CodeTokenInvalid = derror.CodeTokenInvalid

	CodeTokenExpired = derror.CodeTokenExpired

	CodeAccountDisabled = derror.CodeAccountDisabled

	CodeKickedOut = derror.CodeKickedOut

	CodeActiveTimeout = derror.CodeActiveTimeout

	CodeMaxLoginCount = derror.CodeMaxLoginCount

	CodeStorageError = derror.CodeStorageError

	CodeInvalidParameter = derror.CodeInvalidParameter
)

// Common errors Common errors 常用错误
var (
	ErrNotLogin = derror.ErrNotLogin

	ErrInvalidToken = derror.ErrInvalidToken

	ErrTokenExpired = derror.ErrTokenExpired

	ErrPermissionDenied = derror.ErrPermissionDenied

	ErrRoleDenied = derror.ErrRoleDenied

	ErrAccountDisabled = derror.ErrAccountDisabled

	ErrStorageUnavailable = derror.ErrStorageUnavailable

	ErrSerializeFailed = derror.ErrSerializeFailed

	ErrTypeConvert = derror.ErrTypeConvert

	ErrManagerNotFound = derror.ErrManagerNotFound

	ErrManagerInvalidType = derror.ErrManagerInvalidType

	ErrInvalidParam = derror.ErrInvalidParam

	ErrIDIsEmpty = derror.ErrIDIsEmpty

	ErrAccountNotDisabled = derror.ErrAccountNotDisabled

	ErrLoginLimitExceeded = derror.ErrLoginLimitExceeded

	ErrTokenKickout = derror.ErrTokenKickout

	ErrTokenReplaced = derror.ErrTokenReplaced

	ErrInvalidDevice = derror.ErrInvalidDevice

	ErrServiceDisabled = derror.ErrServiceDisabled

	ErrServiceNotDisabled = derror.ErrServiceNotDisabled

	ErrDisableLevelNotReached = derror.ErrDisableLevelNotReached

	ErrClientOrClientIDEmpty = derror.ErrClientOrClientIDEmpty

	ErrClientNotFound = derror.ErrClientNotFound

	ErrInvalidClientCredentials = derror.ErrInvalidClientCredentials

	ErrInvalidGrantType = derror.ErrInvalidGrantType

	ErrInvalidRedirectURI = derror.ErrInvalidRedirectURI

	ErrInvalidScope = derror.ErrInvalidScope

	ErrUserIDEmpty = derror.ErrUserIDEmpty

	ErrInvalidAuthCode = derror.ErrInvalidAuthCode

	ErrAuthCodeUsed = derror.ErrAuthCodeUsed

	ErrAuthCodeExpired = derror.ErrAuthCodeExpired

	ErrClientMismatch = derror.ErrClientMismatch

	ErrRedirectURIMismatch = derror.ErrRedirectURIMismatch

	ErrInvalidRefreshToken = derror.ErrInvalidRefreshToken

	ErrInvalidAccessToken = derror.ErrInvalidAccessToken

	ErrInvalidUserCredentials = derror.ErrInvalidUserCredentials

	ErrSessionNotFound = derror.ErrSessionNotFound

	ErrInvalidNonce = derror.ErrInvalidNonce
)

// Token styles Token styles Token 风格
const (
	TokenStyleUUID = adapter.TokenStyleUUID

	TokenStyleSimple = adapter.TokenStyleSimple

	TokenStyleRandom32 = adapter.TokenStyleRandom32

	TokenStyleRandom64 = adapter.TokenStyleRandom64

	TokenStyleRandom128 = adapter.TokenStyleRandom128

	TokenStyleJWT = adapter.TokenStyleJWT

	TokenStyleHash = adapter.TokenStyleHash

	TokenStyleTimestamp = adapter.TokenStyleTimestamp

	TokenStyleTik = adapter.TokenStyleTik
)

// SameSite modes SameSite modes SameSite 模式
const (
	SameSiteStrict = config.SameSiteStrict

	SameSiteLax = config.SameSiteLax

	SameSiteNone = config.SameSiteNone
)

// Concurrency scopes Concurrency scopes 并发范围
const (
	ConcurrencyScopeAccount = config.ConcurrencyScopeAccount

	ConcurrencyScopeDevice = config.ConcurrencyScopeDevice
)

// Default values Default values 默认值
const (
	DefaultTokenName = config.DefaultTokenName

	DefaultKeyPrefix = config.DefaultKeyPrefix

	DefaultAuthType = config.DefaultAuthType

	DefaultTimeout = config.DefaultTimeout

	DefaultMaxLoginCount = config.DefaultMaxLoginCount

	DefaultCookiePath = config.DefaultCookiePath

	NoLimit = config.NoLimit
)

// Token states Token states Token 状态
const (
	TokenStateLogout = manager.TokenStateLogout

	TokenStateKickOut = manager.TokenStateKickOut

	TokenStateReplaced = manager.TokenStateReplaced
)

// OAuth2 grant types OAuth2 grant types OAuth2 授权类型
const (
	GrantTypeAuthorizationCode = oauth2.GrantTypeAuthorizationCode

	GrantTypeRefreshToken = oauth2.GrantTypeRefreshToken

	GrantTypeClientCredentials = oauth2.GrantTypeClientCredentials

	GrantTypePassword = oauth2.GrantTypePassword
)

// SetManager sets global manager SetManager 设置全局 Manager
func SetManager(mgr *manager.Manager) {
	dtoken.SetManager(mgr)
}

// GetManager gets manager by auth type GetManager 根据 authType 获取 Manager
func GetManager(authType ...string) (*manager.Manager, error) {
	return dtoken.GetManager(authType...)
}

// DeleteManager deletes manager by auth type DeleteManager 根据 authType 删除 Manager
func DeleteManager(authType ...string) error {
	return dtoken.DeleteManager(authType...)
}

// DeleteAllManager deletes all managers DeleteAllManager 删除全部 Manager
func DeleteAllManager() {
	dtoken.DeleteAllManager()
}

// NewDefaultBuilder creates default builder NewDefaultBuilder 创建默认 Builder
func NewDefaultBuilder() *builder.Builder {
	return builder.NewBuilder()
}

// NewDefaultConfig creates default config NewDefaultConfig 创建默认配置
func NewDefaultConfig() *config.Config {
	return config.DefaultConfig()
}

// NewDefaultCookieConfig creates default cookie config NewDefaultCookieConfig 创建默认 Cookie 配置
func NewDefaultCookieConfig() *config.CookieConfig {
	return config.DefaultCookieConfig()
}

// NewManager creates manager instance NewManager 创建 Manager 实例
func NewManager(
	cfg *config.Config,
	generator adapter.Generator,
	storage adapter.Storage,
	serializer adapter.Codec,
	logger adapter.Log,
	pool adapter.Pool,
	customPermissionListFunc, customRoleListFunc func(loginID, authType string) ([]string, error),
	customPermissionListExtFunc, customRoleListExtFunc func(loginID, device, deviceId, authType string) ([]string, error),
) *manager.Manager {
	return manager.NewManager(
		cfg,
		generator,
		storage,
		serializer,
		logger,
		pool,
		customPermissionListFunc,
		customRoleListFunc,
		customPermissionListExtFunc,
		customRoleListExtFunc,
	)
}

// NewContext creates dtoken context NewContext 创建 DTokenContext
func NewContext(reqCtx adapter.RequestContext, mgr *manager.Manager) *corecontext.DTokenContext {
	return corecontext.NewContext(reqCtx, mgr)
}

// NewDTokenError creates dtoken error NewDTokenError 创建 DTokenError
func NewDTokenError(code int, message string, err error) *derror.DTokenError {
	return derror.NewDTokenError(code, message, err)
}

// Login logs in and returns token Login 登录并返回 Token
func Login(ctx context.Context, loginID string, params ...string) (string, error) {
	return dtoken.Login(ctx, loginID, params...)
}

// LoginWithTimeout logs in with custom timeout LoginWithTimeout 使用自定义时长登录
func LoginWithTimeout(ctx context.Context, loginID string, timeout time.Duration, params ...string) (string, error) {
	return dtoken.LoginWithTimeout(ctx, loginID, timeout, params...)
}

// LoginByToken logs in by token LoginByToken 使用 Token 登录
func LoginByToken(ctx context.Context, tokenValue string, authType ...string) error {
	return dtoken.LoginByToken(ctx, tokenValue, authType...)
}

// Logout logs out by token Logout 使用 Token 退出登录
func Logout(ctx context.Context, tokenValue string, authType ...string) error {
	return dtoken.Logout(ctx, tokenValue, authType...)
}

// LogoutByDeviceAndDeviceId logs out by device tuple LogoutByDeviceAndDeviceId 按设备和设备 ID 退出登录
func LogoutByDeviceAndDeviceId(ctx context.Context, loginID string, params ...string) error {
	return dtoken.LogoutByDeviceAndDeviceId(ctx, loginID, params...)
}

// LogoutByDevice logs out by device LogoutByDevice 按设备退出登录
func LogoutByDevice(ctx context.Context, loginID string, device string, authType ...string) error {
	return dtoken.LogoutByDevice(ctx, loginID, device, authType...)
}

// LogoutByLoginID logs out all terminals LogoutByLoginID 按登录 ID 退出全部终端
func LogoutByLoginID(ctx context.Context, loginID string, authType ...string) error {
	return dtoken.LogoutByLoginID(ctx, loginID, authType...)
}

// Kickout kicks out by token Kickout 使用 Token 踢下线
func Kickout(ctx context.Context, tokenValue string, authType ...string) error {
	return dtoken.Kickout(ctx, tokenValue, authType...)
}

// KickoutByDeviceAndDeviceId kicks out by device tuple KickoutByDeviceAndDeviceId 按设备和设备 ID 踢下线
func KickoutByDeviceAndDeviceId(ctx context.Context, loginID string, params ...string) error {
	return dtoken.KickoutByDeviceAndDeviceId(ctx, loginID, params...)
}

// KickoutByDevice kicks out by device KickoutByDevice 按设备踢下线
func KickoutByDevice(ctx context.Context, loginID string, device string, authType ...string) error {
	return dtoken.KickoutByDevice(ctx, loginID, device, authType...)
}

// KickoutByLoginID kicks out all terminals KickoutByLoginID 按登录 ID 踢全部终端下线
func KickoutByLoginID(ctx context.Context, loginID string, authType ...string) error {
	return dtoken.KickoutByLoginID(ctx, loginID, authType...)
}

// Replace replaces token session Replace 替换 Token 会话
func Replace(ctx context.Context, tokenValue string, authType ...string) error {
	return dtoken.Replace(ctx, tokenValue, authType...)
}

// ReplaceByDeviceAndDeviceId replaces by device tuple ReplaceByDeviceAndDeviceId 按设备和设备 ID 替换会话
func ReplaceByDeviceAndDeviceId(ctx context.Context, loginID string, params ...string) error {
	return dtoken.ReplaceByDeviceAndDeviceId(ctx, loginID, params...)
}

// ReplaceByDevice replaces by device ReplaceByDevice 按设备替换会话
func ReplaceByDevice(ctx context.Context, loginID string, device string, authType ...string) error {
	return dtoken.ReplaceByDevice(ctx, loginID, device, authType...)
}

// ReplaceByLoginID replaces all terminals ReplaceByLoginID 按登录 ID 替换全部终端会话
func ReplaceByLoginID(ctx context.Context, loginID string, authType ...string) error {
	return dtoken.ReplaceByLoginID(ctx, loginID, authType...)
}

// IsLogin checks login state IsLogin 检查登录状态
func IsLogin(ctx context.Context, tokenValue string, authType ...string) bool {
	return dtoken.IsLogin(ctx, tokenValue, authType...)
}

// CheckLogin checks login and returns error CheckLogin 校验登录并返回错误
func CheckLogin(ctx context.Context, tokenValue string, authType ...string) error {
	return dtoken.CheckLogin(ctx, tokenValue, authType...)
}

// GetLoginID gets login ID by token GetLoginID 根据 Token 获取登录 ID
func GetLoginID(ctx context.Context, tokenValue string, authType ...string) (string, error) {
	return dtoken.GetLoginID(ctx, tokenValue, authType...)
}

// GetTokenInfo gets token info GetTokenInfo 获取 Token 信息
func GetTokenInfo(ctx context.Context, tokenValue string, authType ...string) (*manager.TokenInfo, error) {
	return dtoken.GetTokenInfo(ctx, tokenValue, authType...)
}

// GetDevice gets device by token GetDevice 根据 Token 获取设备
func GetDevice(ctx context.Context, tokenValue string, authType ...string) (string, error) {
	return dtoken.GetDevice(ctx, tokenValue, authType...)
}

// GetDeviceId gets device ID by token GetDeviceId 根据 Token 获取设备 ID
func GetDeviceId(ctx context.Context, tokenValue string, authType ...string) (string, error) {
	return dtoken.GetDeviceId(ctx, tokenValue, authType...)
}

// GetTokenCreateTime gets token create time GetTokenCreateTime 获取 Token 创建时间
func GetTokenCreateTime(ctx context.Context, tokenValue string, authType ...string) (int64, error) {
	return dtoken.GetTokenCreateTime(ctx, tokenValue, authType...)
}

// GetTokenTTL gets token ttl GetTokenTTL 获取 Token 剩余有效期
func GetTokenTTL(ctx context.Context, tokenValue string, authType ...string) (int64, error) {
	return dtoken.GetTokenTTL(ctx, tokenValue, authType...)
}

// GetOnlineTerminalCount gets online terminal count GetOnlineTerminalCount 获取在线终端数量
func GetOnlineTerminalCount(ctx context.Context, loginID string, authType ...string) (int, error) {
	return dtoken.GetOnlineTerminalCount(ctx, loginID, authType...)
}

// GetOnlineTerminalCountByDevice gets device terminal count GetOnlineTerminalCountByDevice 获取设备在线终端数量
func GetOnlineTerminalCountByDevice(ctx context.Context, loginID string, device string, authType ...string) (int, error) {
	return dtoken.GetOnlineTerminalCountByDevice(ctx, loginID, device, authType...)
}

// GetOnlineTerminalCountByDeviceAndDeviceId gets terminal count by device tuple GetOnlineTerminalCountByDeviceAndDeviceId 按设备和设备 ID 获取在线终端数量
func GetOnlineTerminalCountByDeviceAndDeviceId(ctx context.Context, loginID string, device string, deviceId string, authType ...string) (int, error) {
	return dtoken.GetOnlineTerminalCountByDeviceAndDeviceId(ctx, loginID, device, deviceId, authType...)
}

// Disable disables account Disable 封禁账号
func Disable(ctx context.Context, loginID string, duration time.Duration, reason string, authType ...string) error {
	return dtoken.Disable(ctx, loginID, duration, reason, authType...)
}

// Untie unties disabled account Untie 解封账号
func Untie(ctx context.Context, loginID string, authType ...string) error {
	return dtoken.Untie(ctx, loginID, authType...)
}

// IsDisable checks disable state IsDisable 检查账号是否被封禁
func IsDisable(ctx context.Context, loginID string, authType ...string) bool {
	return dtoken.IsDisable(ctx, loginID, authType...)
}

// GetDisableInfo gets disable info GetDisableInfo 获取账号封禁信息
func GetDisableInfo(ctx context.Context, loginID string, authType ...string) (*manager.DisableInfo, error) {
	return dtoken.GetDisableInfo(ctx, loginID, authType...)
}

// GetDisableTTL gets disable ttl GetDisableTTL 获取账号封禁剩余时间
func GetDisableTTL(ctx context.Context, loginID string, authType ...string) (int64, error) {
	return dtoken.GetDisableTTL(ctx, loginID, authType...)
}

// DisableService disables service DisableService 封禁指定服务
func DisableService(ctx context.Context, loginID, service string, duration time.Duration, authType ...string) error {
	return dtoken.DisableService(ctx, loginID, service, duration, authType...)
}

// DisableServiceWithReason disables service with reason DisableServiceWithReason 封禁指定服务并记录原因
func DisableServiceWithReason(ctx context.Context, loginID, service string, duration time.Duration, reason string, authType ...string) error {
	return dtoken.DisableServiceWithReason(ctx, loginID, service, duration, reason, authType...)
}

// DisableServiceLevel disables service by level DisableServiceLevel 按等级封禁指定服务
func DisableServiceLevel(ctx context.Context, loginID, service string, level int, duration time.Duration, authType ...string) error {
	return dtoken.DisableServiceLevel(ctx, loginID, service, level, duration, authType...)
}

// DisableServiceLevelWithReason disables service by level with reason DisableServiceLevelWithReason 按等级封禁指定服务并记录原因
func DisableServiceLevelWithReason(ctx context.Context, loginID, service string, level int, duration time.Duration, reason string, authType ...string) error {
	return dtoken.DisableServiceLevelWithReason(ctx, loginID, service, level, duration, reason, authType...)
}

// UntieService unties disabled service UntieService 解封指定服务
func UntieService(ctx context.Context, loginID, service string, authType ...string) error {
	return dtoken.UntieService(ctx, loginID, service, authType...)
}

// IsDisableService checks service disable state IsDisableService 检查指定服务是否被封禁
func IsDisableService(ctx context.Context, loginID, service string, authType ...string) bool {
	return dtoken.IsDisableService(ctx, loginID, service, authType...)
}

// IsDisableServiceLevel checks service disable level IsDisableServiceLevel 检查指定服务封禁等级
func IsDisableServiceLevel(ctx context.Context, loginID, service string, level int, authType ...string) bool {
	return dtoken.IsDisableServiceLevel(ctx, loginID, service, level, authType...)
}

// CheckDisableService checks service disable CheckDisableService 校验服务封禁状态
func CheckDisableService(ctx context.Context, loginID string, services []string, authType ...string) error {
	return dtoken.CheckDisableService(ctx, loginID, services, authType...)
}

// CheckDisableServiceLevel checks service disable level CheckDisableServiceLevel 校验服务封禁等级
func CheckDisableServiceLevel(ctx context.Context, loginID, service string, level int, authType ...string) error {
	return dtoken.CheckDisableServiceLevel(ctx, loginID, service, level, authType...)
}

// GetDisableServiceInfo gets service disable info GetDisableServiceInfo 获取服务封禁信息
func GetDisableServiceInfo(ctx context.Context, loginID, service string, authType ...string) (*manager.ServiceDisableInfo, error) {
	return dtoken.GetDisableServiceInfo(ctx, loginID, service, authType...)
}

// GetDisableServiceTTL gets service disable ttl GetDisableServiceTTL 获取服务封禁剩余时间
func GetDisableServiceTTL(ctx context.Context, loginID, service string, authType ...string) (int64, error) {
	return dtoken.GetDisableServiceTTL(ctx, loginID, service, authType...)
}

// GetSession gets session by login ID GetSession 根据登录 ID 获取 Session
func GetSession(ctx context.Context, loginID string, authType ...string) (*manager.Session, error) {
	return dtoken.GetSession(ctx, loginID, authType...)
}

// GetSessionByToken gets session by token GetSessionByToken 根据 Token 获取 Session
func GetSessionByToken(ctx context.Context, tokenValue string, authType ...string) (*manager.Session, error) {
	return dtoken.GetSessionByToken(ctx, tokenValue, authType...)
}

// ForEachTerminal iterates terminals ForEachTerminal 遍历终端信息
func ForEachTerminal(ctx context.Context, loginID string, visitor manager.TerminalVisitor, authType ...string) error {
	return dtoken.ForEachTerminal(ctx, loginID, visitor, authType...)
}

// ForEachTerminalByDevice iterates terminals by device ForEachTerminalByDevice 按设备遍历终端信息
func ForEachTerminalByDevice(ctx context.Context, loginID, device string, visitor manager.TerminalVisitor, authType ...string) error {
	return dtoken.ForEachTerminalByDevice(ctx, loginID, device, visitor, authType...)
}

// GetTokenValueListByLoginID gets token list by login ID GetTokenValueListByLoginID 根据登录 ID 获取 Token 列表
func GetTokenValueListByLoginID(ctx context.Context, loginID string, checkAlive bool, authType ...string) ([]string, error) {
	return dtoken.GetTokenValueListByLoginID(ctx, loginID, checkAlive, authType...)
}

// GetTokenValueListByDevice gets token list by device GetTokenValueListByDevice 按设备获取 Token 列表
func GetTokenValueListByDevice(ctx context.Context, loginID string, device string, checkAlive bool, authType ...string) ([]string, error) {
	return dtoken.GetTokenValueListByDevice(ctx, loginID, device, checkAlive, authType...)
}

// GetTokenValueListByDeviceAndDeviceId gets token list by device tuple GetTokenValueListByDeviceAndDeviceId 按设备和设备 ID 获取 Token 列表
func GetTokenValueListByDeviceAndDeviceId(ctx context.Context, loginID string, device string, deviceId string, checkAlive bool, authType ...string) ([]string, error) {
	return dtoken.GetTokenValueListByDeviceAndDeviceId(ctx, loginID, device, deviceId, checkAlive, authType...)
}

// GetTerminalListByLoginID gets terminal list by login ID GetTerminalListByLoginID 根据登录 ID 获取终端列表
func GetTerminalListByLoginID(ctx context.Context, loginID string, authType ...string) ([]manager.TerminalInfo, error) {
	return dtoken.GetTerminalListByLoginID(ctx, loginID, authType...)
}

// GetTerminalListByLoginIDAndDevice gets terminal list by device GetTerminalListByLoginIDAndDevice 按设备获取终端列表
func GetTerminalListByLoginIDAndDevice(ctx context.Context, loginID string, device string, authType ...string) ([]manager.TerminalInfo, error) {
	return dtoken.GetTerminalListByLoginIDAndDevice(ctx, loginID, device, authType...)
}

// GetTerminalInfoByToken gets terminal info by token GetTerminalInfoByToken 根据 Token 获取终端信息
func GetTerminalInfoByToken(ctx context.Context, tokenValue string, authType ...string) (*manager.TerminalInfo, error) {
	return dtoken.GetTerminalInfoByToken(ctx, tokenValue, authType...)
}

// GetTokenValueByLoginID gets latest token by login ID GetTokenValueByLoginID 根据登录 ID 获取最新 Token
func GetTokenValueByLoginID(ctx context.Context, loginID string, authType ...string) (string, error) {
	return dtoken.GetTokenValueByLoginID(ctx, loginID, authType...)
}

// GetTokenValueByLoginIDAndDevice gets latest token by device GetTokenValueByLoginIDAndDevice 按设备获取最新 Token
func GetTokenValueByLoginIDAndDevice(ctx context.Context, loginID string, device string, authType ...string) (string, error) {
	return dtoken.GetTokenValueByLoginIDAndDevice(ctx, loginID, device, authType...)
}

// SearchTokenValue searches token values SearchTokenValue 搜索 Token 值
func SearchTokenValue(ctx context.Context, keyword string, start, size int, authType ...string) ([]string, error) {
	return dtoken.SearchTokenValue(ctx, keyword, start, size, authType...)
}

// SearchSessionId searches session IDs SearchSessionId 搜索 Session ID
func SearchSessionId(ctx context.Context, keyword string, start, size int, authType ...string) ([]string, error) {
	return dtoken.SearchSessionId(ctx, keyword, start, size, authType...)
}

// CheckPermission checks single permission CheckPermission 校验单个权限
func CheckPermission(ctx context.Context, loginID string, permission string, authType ...string) error {
	return dtoken.CheckPermission(ctx, loginID, permission, authType...)
}

// CheckPermissionAnd checks all permissions CheckPermissionAnd 校验全部权限
func CheckPermissionAnd(ctx context.Context, loginID string, permissions []string, authType ...string) error {
	return dtoken.CheckPermissionAnd(ctx, loginID, permissions, authType...)
}

// CheckPermissionOr checks any permission CheckPermissionOr 校验任一权限
func CheckPermissionOr(ctx context.Context, loginID string, permissions []string, authType ...string) error {
	return dtoken.CheckPermissionOr(ctx, loginID, permissions, authType...)
}

// AddPermissions adds permissions AddPermissions 添加权限
func AddPermissions(ctx context.Context, loginID string, permissions []string, authType ...string) error {
	return dtoken.AddPermissions(ctx, loginID, permissions, authType...)
}

// AddPermissionsByToken adds permissions by token AddPermissionsByToken 按 Token 添加权限
func AddPermissionsByToken(ctx context.Context, tokenValue string, permissions []string, authType ...string) error {
	return dtoken.AddPermissionsByToken(ctx, tokenValue, permissions, authType...)
}

// RemovePermissions removes permissions RemovePermissions 移除权限
func RemovePermissions(ctx context.Context, loginID string, permissions []string, authType ...string) error {
	return dtoken.RemovePermissions(ctx, loginID, permissions, authType...)
}

// RemovePermissionsByToken removes permissions by token RemovePermissionsByToken 按 Token 移除权限
func RemovePermissionsByToken(ctx context.Context, tokenValue string, permissions []string, authType ...string) error {
	return dtoken.RemovePermissionsByToken(ctx, tokenValue, permissions, authType...)
}

// GetPermissions gets permission list GetPermissions 获取权限列表
func GetPermissions(ctx context.Context, loginID string, authType ...string) ([]string, error) {
	return dtoken.GetPermissions(ctx, loginID, authType...)
}

// GetPermissionsByToken gets permission list by token GetPermissionsByToken 根据 Token 获取权限列表
func GetPermissionsByToken(ctx context.Context, tokenValue string, authType ...string) ([]string, error) {
	return dtoken.GetPermissionsByToken(ctx, tokenValue, authType...)
}

// HasPermission checks permission HasPermission 检查是否拥有权限
func HasPermission(ctx context.Context, loginID string, permission string, authType ...string) bool {
	return dtoken.HasPermission(ctx, loginID, permission, authType...)
}

// HasPermissionByToken checks permission by token HasPermissionByToken 按 Token 检查权限
func HasPermissionByToken(ctx context.Context, tokenValue string, permission string, authType ...string) bool {
	return dtoken.HasPermissionByToken(ctx, tokenValue, permission, authType...)
}

// HasPermissionsAnd checks all permissions HasPermissionsAnd 检查是否拥有全部权限
func HasPermissionsAnd(ctx context.Context, loginID string, permissions []string, authType ...string) bool {
	return dtoken.HasPermissionsAnd(ctx, loginID, permissions, authType...)
}

// HasPermissionsAndByToken checks all permissions by token HasPermissionsAndByToken 按 Token 检查是否拥有全部权限
func HasPermissionsAndByToken(ctx context.Context, tokenValue string, permissions []string, authType ...string) bool {
	return dtoken.HasPermissionsAndByToken(ctx, tokenValue, permissions, authType...)
}

// HasPermissionsOr checks any permission HasPermissionsOr 检查是否拥有任一权限
func HasPermissionsOr(ctx context.Context, loginID string, permissions []string, authType ...string) bool {
	return dtoken.HasPermissionsOr(ctx, loginID, permissions, authType...)
}

// HasPermissionsOrByToken checks any permission by token HasPermissionsOrByToken 按 Token 检查是否拥有任一权限
func HasPermissionsOrByToken(ctx context.Context, tokenValue string, permissions []string, authType ...string) bool {
	return dtoken.HasPermissionsOrByToken(ctx, tokenValue, permissions, authType...)
}

// AddRoles adds roles AddRoles 添加角色
func AddRoles(ctx context.Context, loginID string, roles []string, authType ...string) error {
	return dtoken.AddRoles(ctx, loginID, roles, authType...)
}

// AddRolesByToken adds roles by token AddRolesByToken 按 Token 添加角色
func AddRolesByToken(ctx context.Context, tokenValue string, roles []string, authType ...string) error {
	return dtoken.AddRolesByToken(ctx, tokenValue, roles, authType...)
}

// RemoveRoles removes roles RemoveRoles 移除角色
func RemoveRoles(ctx context.Context, loginID string, roles []string, authType ...string) error {
	return dtoken.RemoveRoles(ctx, loginID, roles, authType...)
}

// RemoveRolesByToken removes roles by token RemoveRolesByToken 按 Token 移除角色
func RemoveRolesByToken(ctx context.Context, tokenValue string, roles []string, authType ...string) error {
	return dtoken.RemoveRolesByToken(ctx, tokenValue, roles, authType...)
}

// GetRoles gets role list GetRoles 获取角色列表
func GetRoles(ctx context.Context, loginID string, authType ...string) ([]string, error) {
	return dtoken.GetRoles(ctx, loginID, authType...)
}

// GetRolesByToken gets role list by token GetRolesByToken 根据 Token 获取角色列表
func GetRolesByToken(ctx context.Context, tokenValue string, authType ...string) ([]string, error) {
	return dtoken.GetRolesByToken(ctx, tokenValue, authType...)
}

// HasRole checks role HasRole 检查是否拥有角色
func HasRole(ctx context.Context, loginID string, role string, authType ...string) bool {
	return dtoken.HasRole(ctx, loginID, role, authType...)
}

// HasRoleByToken checks role by token HasRoleByToken 按 Token 检查角色
func HasRoleByToken(ctx context.Context, tokenValue string, role string, authType ...string) bool {
	return dtoken.HasRoleByToken(ctx, tokenValue, role, authType...)
}

// HasRolesAnd checks all roles HasRolesAnd 检查是否拥有全部角色
func HasRolesAnd(ctx context.Context, loginID string, roles []string, authType ...string) bool {
	return dtoken.HasRolesAnd(ctx, loginID, roles, authType...)
}

// HasRolesAndByToken checks all roles by token HasRolesAndByToken 按 Token 检查是否拥有全部角色
func HasRolesAndByToken(ctx context.Context, tokenValue string, roles []string, authType ...string) bool {
	return dtoken.HasRolesAndByToken(ctx, tokenValue, roles, authType...)
}

// HasRolesOr checks any role HasRolesOr 检查是否拥有任一角色
func HasRolesOr(ctx context.Context, loginID string, roles []string, authType ...string) bool {
	return dtoken.HasRolesOr(ctx, loginID, roles, authType...)
}

// HasRolesOrByToken checks any role by token HasRolesOrByToken 按 Token 检查是否拥有任一角色
func HasRolesOrByToken(ctx context.Context, tokenValue string, roles []string, authType ...string) bool {
	return dtoken.HasRolesOrByToken(ctx, tokenValue, roles, authType...)
}

// CheckRole checks single role CheckRole 校验单个角色
func CheckRole(ctx context.Context, loginID string, role string, authType ...string) error {
	return dtoken.CheckRole(ctx, loginID, role, authType...)
}

// CheckRoleAnd checks all roles CheckRoleAnd 校验全部角色
func CheckRoleAnd(ctx context.Context, loginID string, roles []string, authType ...string) error {
	return dtoken.CheckRoleAnd(ctx, loginID, roles, authType...)
}

// CheckRoleOr checks any role CheckRoleOr 校验任一角色
func CheckRoleOr(ctx context.Context, loginID string, roles []string, authType ...string) error {
	return dtoken.CheckRoleOr(ctx, loginID, roles, authType...)
}

// CheckDisable checks disable state CheckDisable 校验封禁状态
func CheckDisable(ctx context.Context, loginID string, authType ...string) error {
	return dtoken.CheckDisable(ctx, loginID, authType...)
}

// RenewTimeout renews token timeout RenewTimeout 续期 Token 超时时间
func RenewTimeout(ctx context.Context, tokenValue string, timeout time.Duration, authType ...string) error {
	return dtoken.RenewTimeout(ctx, tokenValue, timeout, authType...)
}

// GenerateNonce generates nonce GenerateNonce 生成 Nonce
func GenerateNonce(ctx context.Context, authType ...string) (string, error) {
	return dtoken.GenerateNonce(ctx, authType...)
}

// GenerateNonceWithTimeout generates nonce with timeout GenerateNonceWithTimeout 按时长生成 Nonce
func GenerateNonceWithTimeout(ctx context.Context, timeout time.Duration, authType ...string) (string, error) {
	return dtoken.GenerateNonceWithTimeout(ctx, timeout, authType...)
}

// VerifyNonce verifies nonce VerifyNonce 校验 Nonce
func VerifyNonce(ctx context.Context, nonce string, authType ...string) bool {
	return dtoken.VerifyNonce(ctx, nonce, authType...)
}

// VerifyAndConsumeNonce verifies and consumes nonce VerifyAndConsumeNonce 校验并消费 Nonce
func VerifyAndConsumeNonce(ctx context.Context, nonce string, authType ...string) error {
	return dtoken.VerifyAndConsumeNonce(ctx, nonce, authType...)
}

// IsNonceValid checks nonce validity IsNonceValid 检查 Nonce 是否有效
func IsNonceValid(ctx context.Context, nonce string, authType ...string) bool {
	return dtoken.IsNonceValid(ctx, nonce, authType...)
}

// GetNonceTTL gets nonce ttl GetNonceTTL 获取 Nonce 剩余有效期
func GetNonceTTL(ctx context.Context, nonce string, authType ...string) (int64, error) {
	return dtoken.GetNonceTTL(ctx, nonce, authType...)
}

// RegisterOAuth2Client registers oauth2 client RegisterOAuth2Client 注册 OAuth2 客户端
func RegisterOAuth2Client(client *oauth2.Client, authType ...string) error {
	return dtoken.RegisterOAuth2Client(client, authType...)
}

// UnregisterOAuth2Client unregisters oauth2 client UnregisterOAuth2Client 注销 OAuth2 客户端
func UnregisterOAuth2Client(clientID string, authType ...string) error {
	return dtoken.UnregisterOAuth2Client(clientID, authType...)
}

// GetOAuth2Client gets oauth2 client GetOAuth2Client 获取 OAuth2 客户端
func GetOAuth2Client(clientID string, authType ...string) (*oauth2.Client, error) {
	return dtoken.GetOAuth2Client(clientID, authType...)
}

// OAuth2Token issues oauth2 token OAuth2Token 签发 OAuth2 Token
func OAuth2Token(ctx context.Context, req *oauth2.TokenRequest, validateUser oauth2.UserValidator, authType ...string) (*oauth2.AccessToken, error) {
	return dtoken.OAuth2Token(ctx, req, validateUser, authType...)
}

// GenerateOAuth2AuthorizationCode generates auth code GenerateOAuth2AuthorizationCode 生成 OAuth2 授权码
func GenerateOAuth2AuthorizationCode(ctx context.Context, clientID, userID, redirectURI string, scopes []string, authType ...string) (*oauth2.AuthorizationCode, error) {
	return dtoken.GenerateOAuth2AuthorizationCode(ctx, clientID, userID, redirectURI, scopes, authType...)
}

// ExchangeOAuth2CodeForToken exchanges code for token ExchangeOAuth2CodeForToken 使用授权码换取 Token
func ExchangeOAuth2CodeForToken(ctx context.Context, code, clientID, clientSecret, redirectURI string, authType ...string) (*oauth2.AccessToken, error) {
	return dtoken.ExchangeOAuth2CodeForToken(ctx, code, clientID, clientSecret, redirectURI, authType...)
}

// OAuth2ClientCredentialsToken issues client token OAuth2ClientCredentialsToken 签发客户端模式 Token
func OAuth2ClientCredentialsToken(ctx context.Context, clientID, clientSecret string, scopes []string, authType ...string) (*oauth2.AccessToken, error) {
	return dtoken.OAuth2ClientCredentialsToken(ctx, clientID, clientSecret, scopes, authType...)
}

// OAuth2PasswordGrantToken issues password token OAuth2PasswordGrantToken 签发密码模式 Token
func OAuth2PasswordGrantToken(ctx context.Context, clientID, clientSecret, username, password string, scopes []string, validateUser oauth2.UserValidator, authType ...string) (*oauth2.AccessToken, error) {
	return dtoken.OAuth2PasswordGrantToken(ctx, clientID, clientSecret, username, password, scopes, validateUser, authType...)
}

// RefreshOAuth2AccessToken refreshes access token RefreshOAuth2AccessToken 刷新 OAuth2 访问令牌
func RefreshOAuth2AccessToken(ctx context.Context, clientID, refreshToken, clientSecret string, authType ...string) (*oauth2.AccessToken, error) {
	return dtoken.RefreshOAuth2AccessToken(ctx, clientID, refreshToken, clientSecret, authType...)
}

// ValidateOAuth2AccessToken validates access token ValidateOAuth2AccessToken 校验 OAuth2 访问令牌
func ValidateOAuth2AccessToken(ctx context.Context, accessToken string, authType ...string) bool {
	return dtoken.ValidateOAuth2AccessToken(ctx, accessToken, authType...)
}

// ValidateOAuth2AccessTokenAndGetInfo validates token and gets info ValidateOAuth2AccessTokenAndGetInfo 校验 OAuth2 访问令牌并获取信息
func ValidateOAuth2AccessTokenAndGetInfo(ctx context.Context, accessToken string, authType ...string) (*oauth2.AccessToken, error) {
	return dtoken.ValidateOAuth2AccessTokenAndGetInfo(ctx, accessToken, authType...)
}

// RevokeOAuth2Token revokes access token RevokeOAuth2Token 撤销 OAuth2 访问令牌
func RevokeOAuth2Token(ctx context.Context, accessToken string, authType ...string) error {
	return dtoken.RevokeOAuth2Token(ctx, accessToken, authType...)
}
