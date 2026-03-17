package fiber

import (
	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/builder"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/core/oauth2"
	"github.com/Zany2/dtoken-go/dtoken"
)

// -------------------------------------------------- Type Aliases - 类型别名 --------------------------------------------------
type (
	Config             = config.Config
	Manager            = manager.Manager
	TokenInfo          = manager.TokenInfo
	DisableInfo        = manager.DisableInfo
	ServiceDisableInfo = manager.ServiceDisableInfo
	Session            = manager.Session
	TerminalInfo       = manager.TerminalInfo
	Builder            = builder.Builder
	DTokenError        = derror.DTokenError
	TokenStyle         = adapter.TokenStyle
	OAuth2Client       = oauth2.Client
	OAuth2AccessToken  = oauth2.AccessToken
	OAuth2TokenRequest = oauth2.TokenRequest
	OAuth2AuthCode     = oauth2.AuthorizationCode
)

// -------------------------------------------------- Error Code Aliases - 错误码别名 --------------------------------------------------
const (
	CodeSuccess          = derror.CodeSuccess
	CodeBadRequest       = derror.CodeBadRequest
	CodeNotLogin         = derror.CodeNotLogin
	CodePermissionDenied = derror.CodePermissionDenied
	CodeNotFound         = derror.CodeNotFound
	CodeServerError      = derror.CodeServerError
	CodeTokenInvalid     = derror.CodeTokenInvalid
	CodeTokenExpired     = derror.CodeTokenExpired
	CodeAccountDisabled  = derror.CodeAccountDisabled
)

// -------------------------------------------------- Token Style Aliases - Token 风格别名 --------------------------------------------------
const (
	TokenStyleUUID      = adapter.TokenStyleUUID
	TokenStyleSimple    = adapter.TokenStyleSimple
	TokenStyleRandom32  = adapter.TokenStyleRandom32
	TokenStyleRandom64  = adapter.TokenStyleRandom64
	TokenStyleRandom128 = adapter.TokenStyleRandom128
	TokenStyleJWT       = adapter.TokenStyleJWT
)

// -------------------------------------------------- Exported Aliases - 对外快捷入口与 API 别名 --------------------------------------------------
var (
	ErrNotLogin         = derror.ErrNotLogin
	ErrInvalidToken     = derror.ErrInvalidToken
	ErrTokenExpired     = derror.ErrTokenExpired
	ErrPermissionDenied = derror.ErrPermissionDenied
	ErrRoleDenied       = derror.ErrRoleDenied
	ErrAccountDisabled  = derror.ErrAccountDisabled

	NewDefaultBuilder = builder.NewBuilder
	NewDefaultConfig  = config.DefaultConfig

	SetManager       = dtoken.SetManager
	GetManager       = dtoken.GetManager
	DeleteManager    = dtoken.DeleteManager
	DeleteAllManager = dtoken.DeleteAllManager

	Login                      = dtoken.Login
	LoginWithTimeout           = dtoken.LoginWithTimeout
	LoginByToken               = dtoken.LoginByToken
	Logout                     = dtoken.Logout
	LogoutByDeviceAndDeviceId  = dtoken.LogoutByDeviceAndDeviceId
	LogoutByDevice             = dtoken.LogoutByDevice
	LogoutByLoginID            = dtoken.LogoutByLoginID
	Kickout                    = dtoken.Kickout
	Replace                    = dtoken.Replace
	KickoutByDeviceAndDeviceId = dtoken.KickoutByDeviceAndDeviceId
	KickoutByDevice            = dtoken.KickoutByDevice
	KickoutByLoginID           = dtoken.KickoutByLoginID
	ReplaceByDeviceAndDeviceId = dtoken.ReplaceByDeviceAndDeviceId
	ReplaceByDevice            = dtoken.ReplaceByDevice
	ReplaceByLoginID           = dtoken.ReplaceByLoginID

	IsLogin                                   = dtoken.IsLogin
	CheckLogin                                = dtoken.CheckLogin
	GetLoginID                                = dtoken.GetLoginID
	GetTokenInfo                              = dtoken.GetTokenInfo
	GetDevice                                 = dtoken.GetDevice
	GetDeviceId                               = dtoken.GetDeviceId
	GetTokenCreateTime                        = dtoken.GetTokenCreateTime
	GetTokenTTL                               = dtoken.GetTokenTTL
	GetOnlineTerminalCount                    = dtoken.GetOnlineTerminalCount
	GetOnlineTerminalCountByDevice            = dtoken.GetOnlineTerminalCountByDevice
	GetOnlineTerminalCountByDeviceAndDeviceId = dtoken.GetOnlineTerminalCountByDeviceAndDeviceId

	Disable                       = dtoken.Disable
	Untie                         = dtoken.Untie
	IsDisable                     = dtoken.IsDisable
	GetDisableInfo                = dtoken.GetDisableInfo
	GetDisableTTL                 = dtoken.GetDisableTTL
	DisableService                = dtoken.DisableService
	DisableServiceWithReason      = dtoken.DisableServiceWithReason
	DisableServiceLevel           = dtoken.DisableServiceLevel
	DisableServiceLevelWithReason = dtoken.DisableServiceLevelWithReason
	UntieService                  = dtoken.UntieService
	IsDisableService              = dtoken.IsDisableService
	IsDisableServiceLevel         = dtoken.IsDisableServiceLevel
	CheckDisableService           = dtoken.CheckDisableService
	CheckDisableServiceLevel      = dtoken.CheckDisableServiceLevel
	GetDisableServiceInfo         = dtoken.GetDisableServiceInfo
	GetDisableServiceTTL          = dtoken.GetDisableServiceTTL

	CheckPermission    = dtoken.CheckPermission
	CheckPermissionAnd = dtoken.CheckPermissionAnd
	CheckPermissionOr  = dtoken.CheckPermissionOr
	CheckRole          = dtoken.CheckRole
	CheckRoleAnd       = dtoken.CheckRoleAnd
	CheckRoleOr        = dtoken.CheckRoleOr
	CheckDisable       = dtoken.CheckDisable
	RenewTimeout       = dtoken.RenewTimeout

	ForEachTerminal                      = dtoken.ForEachTerminal
	ForEachTerminalByDevice              = dtoken.ForEachTerminalByDevice
	GetSession                           = dtoken.GetSession
	GetSessionByToken                    = dtoken.GetSessionByToken
	GetTokenValueListByLoginID           = dtoken.GetTokenValueListByLoginID
	GetTokenValueListByDeviceAndDeviceId = dtoken.GetTokenValueListByDeviceAndDeviceId
	GetTokenValueListByDevice            = dtoken.GetTokenValueListByDevice
	GetTerminalListByLoginID             = dtoken.GetTerminalListByLoginID
	GetTerminalListByLoginIDAndDevice    = dtoken.GetTerminalListByLoginIDAndDevice
	GetTerminalInfoByToken               = dtoken.GetTerminalInfoByToken
	GetTokenValueByLoginID               = dtoken.GetTokenValueByLoginID
	GetTokenValueByLoginIDAndDevice      = dtoken.GetTokenValueByLoginIDAndDevice
	SearchTokenValue                     = dtoken.SearchTokenValue
	SearchSessionId                      = dtoken.SearchSessionId

	AddPermissions           = dtoken.AddPermissions
	AddPermissionsByToken    = dtoken.AddPermissionsByToken
	RemovePermissions        = dtoken.RemovePermissions
	RemovePermissionsByToken = dtoken.RemovePermissionsByToken
	GetPermissions           = dtoken.GetPermissions
	GetPermissionsByToken    = dtoken.GetPermissionsByToken
	HasPermission            = dtoken.HasPermission
	HasPermissionByToken     = dtoken.HasPermissionByToken
	HasPermissionsAnd        = dtoken.HasPermissionsAnd
	HasPermissionsAndByToken = dtoken.HasPermissionsAndByToken
	HasPermissionsOr         = dtoken.HasPermissionsOr
	HasPermissionsOrByToken  = dtoken.HasPermissionsOrByToken

	AddRoles           = dtoken.AddRoles
	AddRolesByToken    = dtoken.AddRolesByToken
	RemoveRoles        = dtoken.RemoveRoles
	RemoveRolesByToken = dtoken.RemoveRolesByToken
	GetRoles           = dtoken.GetRoles
	GetRolesByToken    = dtoken.GetRolesByToken
	HasRole            = dtoken.HasRole
	HasRoleByToken     = dtoken.HasRoleByToken
	HasRolesAnd        = dtoken.HasRolesAnd
	HasRolesAndByToken = dtoken.HasRolesAndByToken
	HasRolesOr         = dtoken.HasRolesOr
	HasRolesOrByToken  = dtoken.HasRolesOrByToken

	GenerateNonce            = dtoken.GenerateNonce
	GenerateNonceWithTimeout = dtoken.GenerateNonceWithTimeout
	VerifyNonce              = dtoken.VerifyNonce
	VerifyAndConsumeNonce    = dtoken.VerifyAndConsumeNonce
	IsNonceValid             = dtoken.IsNonceValid
	GetNonceTTL              = dtoken.GetNonceTTL

	RegisterOAuth2Client                = dtoken.RegisterOAuth2Client
	UnregisterOAuth2Client              = dtoken.UnregisterOAuth2Client
	GetOAuth2Client                     = dtoken.GetOAuth2Client
	OAuth2Token                         = dtoken.OAuth2Token
	GenerateOAuth2AuthorizationCode     = dtoken.GenerateOAuth2AuthorizationCode
	ExchangeOAuth2CodeForToken          = dtoken.ExchangeOAuth2CodeForToken
	OAuth2ClientCredentialsToken        = dtoken.OAuth2ClientCredentialsToken
	OAuth2PasswordGrantToken            = dtoken.OAuth2PasswordGrantToken
	RefreshOAuth2AccessToken            = dtoken.RefreshOAuth2AccessToken
	ValidateOAuth2AccessToken           = dtoken.ValidateOAuth2AccessToken
	ValidateOAuth2AccessTokenAndGetInfo = dtoken.ValidateOAuth2AccessTokenAndGetInfo
	RevokeOAuth2Token                   = dtoken.RevokeOAuth2Token
)
