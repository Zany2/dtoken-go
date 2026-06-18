// @Author daixk 2025/12/22 15:56:00
package echo

import (
	"github.com/Zany2/dtoken-go/core/builder"
	corecontext "github.com/Zany2/dtoken-go/core/context"
	"github.com/Zany2/dtoken-go/defaults"
	"github.com/Zany2/dtoken-go/dtoken"
)

// DTokenContext exposes the request-scoped DToken context type. DTokenContext 暴露请求级 DToken 上下文类型。
type DTokenContext = corecontext.DTokenContext

// NewBuilder creates a default DToken builder. NewBuilder 创建默认 DToken 构建器。
func NewBuilder() *builder.Builder {
	return defaults.NewBuilder()
}

// DToken operations forward all exported dtoken APIs. DToken 操作转发 dtoken 的全部公开 API。
var (
	SetManager                                = dtoken.SetManager
	GetManager                                = dtoken.GetManager
	GetEventManager                           = dtoken.GetEventManager
	DeleteManager                             = dtoken.DeleteManager
	DeleteAllManager                          = dtoken.DeleteAllManager
	Login                                     = dtoken.Login
	LoginWithTimeout                          = dtoken.LoginWithTimeout
	LoginWithRefreshToken                     = dtoken.LoginWithRefreshToken
	RefreshToken                              = dtoken.RefreshToken
	RevokeRefreshToken                        = dtoken.RevokeRefreshToken
	GetRefreshTokenTTL                        = dtoken.GetRefreshTokenTTL
	IntrospectToken                           = dtoken.IntrospectToken
	LoginByToken                              = dtoken.LoginByToken
	Logout                                    = dtoken.Logout
	LogoutByDeviceAndDeviceId                 = dtoken.LogoutByDeviceAndDeviceId
	LogoutByDevice                            = dtoken.LogoutByDevice
	LogoutByLoginID                           = dtoken.LogoutByLoginID
	Kickout                                   = dtoken.Kickout
	Replace                                   = dtoken.Replace
	KickoutByDeviceAndDeviceId                = dtoken.KickoutByDeviceAndDeviceId
	KickoutByDevice                           = dtoken.KickoutByDevice
	KickoutByLoginID                          = dtoken.KickoutByLoginID
	ReplaceByDeviceAndDeviceId                = dtoken.ReplaceByDeviceAndDeviceId
	ReplaceByDevice                           = dtoken.ReplaceByDevice
	ReplaceByLoginID                          = dtoken.ReplaceByLoginID
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
	Disable                                   = dtoken.Disable
	Untie                                     = dtoken.Untie
	IsDisable                                 = dtoken.IsDisable
	GetDisableInfo                            = dtoken.GetDisableInfo
	GetDisableTTL                             = dtoken.GetDisableTTL
	DisableService                            = dtoken.DisableService
	DisableServiceWithReason                  = dtoken.DisableServiceWithReason
	DisableServiceLevel                       = dtoken.DisableServiceLevel
	DisableServiceLevelWithReason             = dtoken.DisableServiceLevelWithReason
	UntieService                              = dtoken.UntieService
	IsDisableService                          = dtoken.IsDisableService
	IsDisableServiceLevel                     = dtoken.IsDisableServiceLevel
	CheckDisableService                       = dtoken.CheckDisableService
	CheckDisableServiceLevel                  = dtoken.CheckDisableServiceLevel
	GetDisableServiceInfo                     = dtoken.GetDisableServiceInfo
	GetDisableServiceTTL                      = dtoken.GetDisableServiceTTL
	DisableDevice                             = dtoken.DisableDevice
	DisableDeviceWithReason                   = dtoken.DisableDeviceWithReason
	DisableDeviceAndDeviceId                  = dtoken.DisableDeviceAndDeviceId
	DisableDeviceAndDeviceIdWithReason        = dtoken.DisableDeviceAndDeviceIdWithReason
	UntieDevice                               = dtoken.UntieDevice
	UntieDeviceAndDeviceId                    = dtoken.UntieDeviceAndDeviceId
	IsDisableDevice                           = dtoken.IsDisableDevice
	IsDisableDeviceAndDeviceId                = dtoken.IsDisableDeviceAndDeviceId
	CheckDisableDevice                        = dtoken.CheckDisableDevice
	CheckDisableDeviceAndDeviceId             = dtoken.CheckDisableDeviceAndDeviceId
	GetDisableDeviceInfo                      = dtoken.GetDisableDeviceInfo
	GetDisableDeviceAndDeviceIdInfo           = dtoken.GetDisableDeviceAndDeviceIdInfo
	GetDisableDeviceTTL                       = dtoken.GetDisableDeviceTTL
	GetDisableDeviceAndDeviceIdTTL            = dtoken.GetDisableDeviceAndDeviceIdTTL
	CheckPermission                           = dtoken.CheckPermission
	CheckPermissionAnd                        = dtoken.CheckPermissionAnd
	CheckPermissionOr                         = dtoken.CheckPermissionOr
	CheckRole                                 = dtoken.CheckRole
	CheckRoleAnd                              = dtoken.CheckRoleAnd
	CheckRoleOr                               = dtoken.CheckRoleOr
	CheckDisable                              = dtoken.CheckDisable
	RenewTimeout                              = dtoken.RenewTimeout
	ForEachTerminal                           = dtoken.ForEachTerminal
	ForEachTerminalByDevice                   = dtoken.ForEachTerminalByDevice
	GetSession                                = dtoken.GetSession
	GetSessionByToken                         = dtoken.GetSessionByToken
	GetTokenValueListByLoginID                = dtoken.GetTokenValueListByLoginID
	GetTokenValueListByDeviceAndDeviceId      = dtoken.GetTokenValueListByDeviceAndDeviceId
	GetTokenValueListByDevice                 = dtoken.GetTokenValueListByDevice
	GetTerminalListByLoginID                  = dtoken.GetTerminalListByLoginID
	GetTerminalListByLoginIDAndDevice         = dtoken.GetTerminalListByLoginIDAndDevice
	GetTerminalInfoByToken                    = dtoken.GetTerminalInfoByToken
	GetTokenValueByLoginID                    = dtoken.GetTokenValueByLoginID
	GetTokenValueByLoginIDAndDevice           = dtoken.GetTokenValueByLoginIDAndDevice
	SearchTokenValue                          = dtoken.SearchTokenValue
	SearchSessionId                           = dtoken.SearchSessionId
	AddPermissions                            = dtoken.AddPermissions
	AddPermissionsByToken                     = dtoken.AddPermissionsByToken
	RemovePermissions                         = dtoken.RemovePermissions
	RemovePermissionsByToken                  = dtoken.RemovePermissionsByToken
	GetPermissions                            = dtoken.GetPermissions
	GetPermissionsByToken                     = dtoken.GetPermissionsByToken
	HasPermission                             = dtoken.HasPermission
	HasPermissionByToken                      = dtoken.HasPermissionByToken
	HasPermissionsAnd                         = dtoken.HasPermissionsAnd
	HasPermissionsAndByToken                  = dtoken.HasPermissionsAndByToken
	HasPermissionsOr                          = dtoken.HasPermissionsOr
	HasPermissionsOrByToken                   = dtoken.HasPermissionsOrByToken
	AddRoles                                  = dtoken.AddRoles
	AddRolesByToken                           = dtoken.AddRolesByToken
	RemoveRoles                               = dtoken.RemoveRoles
	RemoveRolesByToken                        = dtoken.RemoveRolesByToken
	GetRoles                                  = dtoken.GetRoles
	GetRolesByToken                           = dtoken.GetRolesByToken
	HasRole                                   = dtoken.HasRole
	HasRoleByToken                            = dtoken.HasRoleByToken
	HasRolesAnd                               = dtoken.HasRolesAnd
	HasRolesAndByToken                        = dtoken.HasRolesAndByToken
	HasRolesOr                                = dtoken.HasRolesOr
	HasRolesOrByToken                         = dtoken.HasRolesOrByToken
	GenerateNonce                             = dtoken.GenerateNonce
	GenerateNonceWithTimeout                  = dtoken.GenerateNonceWithTimeout
	VerifyNonce                               = dtoken.VerifyNonce
	VerifyAndConsumeNonce                     = dtoken.VerifyAndConsumeNonce
	IsNonceValid                              = dtoken.IsNonceValid
	GetNonceTTL                               = dtoken.GetNonceTTL
	RegisterOAuth2Client                      = dtoken.RegisterOAuth2Client
	UnregisterOAuth2Client                    = dtoken.UnregisterOAuth2Client
	GetOAuth2Client                           = dtoken.GetOAuth2Client
	OAuth2Token                               = dtoken.OAuth2Token
	GenerateOAuth2AuthorizationCode           = dtoken.GenerateOAuth2AuthorizationCode
	GenerateOAuth2AuthorizationCodeWithPKCE   = dtoken.GenerateOAuth2AuthorizationCodeWithPKCE
	ExchangeOAuth2CodeForToken                = dtoken.ExchangeOAuth2CodeForToken
	ExchangeOAuth2CodeForTokenWithPKCE        = dtoken.ExchangeOAuth2CodeForTokenWithPKCE
	OAuth2ClientCredentialsToken              = dtoken.OAuth2ClientCredentialsToken
	OAuth2PasswordGrantToken                  = dtoken.OAuth2PasswordGrantToken
	RefreshOAuth2AccessToken                  = dtoken.RefreshOAuth2AccessToken
	ValidateOAuth2AccessToken                 = dtoken.ValidateOAuth2AccessToken
	ValidateOAuth2AccessTokenAndGetInfo       = dtoken.ValidateOAuth2AccessTokenAndGetInfo
	RevokeOAuth2Token                         = dtoken.RevokeOAuth2Token
)
