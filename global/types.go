package global

import "github.com/Zany2/dtoken-go"

// Options reuse the instance API contracts. 选项复用实例 API 的契约。
type (
	Auth                  = dtoken.Auth
	Builder               = dtoken.Builder
	LoginOptions          = dtoken.LoginOptions
	RefreshTokenOptions   = dtoken.RefreshTokenOptions
	LogoutOptions         = dtoken.LogoutOptions
	DisableOptions        = dtoken.DisableOptions
	ServiceDisableOptions = dtoken.ServiceDisableOptions
	DeviceDisableOptions  = dtoken.DeviceDisableOptions
	PermissionOptions     = dtoken.PermissionOptions
	RoleOptions           = dtoken.RoleOptions
)

// New wraps an existing authentication engine. New 包装既有认证引擎。
var New = dtoken.New
