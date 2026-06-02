// @Author daixk 2025/12/22 15:56:00
package gin

import "github.com/Zany2/dtoken-go/dtoken"

// Auth exposes the instance-oriented dtoken facade.
type Auth = dtoken.Auth

// Typed option aliases keep framework imports self-contained.
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

// Instance and typed global operations.
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
