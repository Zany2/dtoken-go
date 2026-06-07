// @Author daixk 2026/06/06
package beego

import "github.com/Zany2/dtoken-go/dtoken"

// Auth exposes the instance-oriented dtoken facade Auth 暴露面向实例的 dtoken 门面
type Auth = dtoken.Auth

// Typed option aliases keep framework imports self-contained 类型别名让框架包自包含
type (
	LoginOptions          = dtoken.LoginOptions
	RefreshTokenOptions   = dtoken.RefreshTokenOptions
	LogoutOptions         = dtoken.LogoutOptions
	DisableOptions        = dtoken.DisableOptions
	ServiceDisableOptions = dtoken.ServiceDisableOptions
	DeviceDisableOptions  = dtoken.DeviceDisableOptions
	PermissionOptions     = dtoken.PermissionOptions
	RoleOptions           = dtoken.RoleOptions
)

// Instance and typed global operations 实例和强类型全局操作
var (
	New                            = dtoken.New
	Default                        = dtoken.Default
	MustDefault                    = dtoken.MustDefault
	NewByAuthType                  = dtoken.NewByAuthType
	LoginWithOptions               = dtoken.LoginWithOptions
	LoginWithRefreshTokenOptions   = dtoken.LoginWithRefreshTokenOptions
	LogoutWithOptions              = dtoken.LogoutWithOptions
	KickoutWithOptions             = dtoken.KickoutWithOptions
	ReplaceWithOptions             = dtoken.ReplaceWithOptions
	DisableWithOptions             = dtoken.DisableWithOptions
	DisableServiceWithOptions      = dtoken.DisableServiceWithOptions
	DisableDeviceWithOptions       = dtoken.DisableDeviceWithOptions
	AddPermissionsWithOptions      = dtoken.AddPermissionsWithOptions
	RemovePermissionsWithOptions   = dtoken.RemovePermissionsWithOptions
	CheckPermissionWithOptions     = dtoken.CheckPermissionWithOptions
	CheckPermissionsAndWithOptions = dtoken.CheckPermissionsAndWithOptions
	CheckPermissionsOrWithOptions  = dtoken.CheckPermissionsOrWithOptions
	AddRolesWithOptions            = dtoken.AddRolesWithOptions
	RemoveRolesWithOptions         = dtoken.RemoveRolesWithOptions
	CheckRoleWithOptions           = dtoken.CheckRoleWithOptions
	CheckRolesAndWithOptions       = dtoken.CheckRolesAndWithOptions
	CheckRolesOrWithOptions        = dtoken.CheckRolesOrWithOptions
)
