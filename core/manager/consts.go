// @Author daixk 2025/12/22 15:56:00
package manager

const (
	// DisableKeyPrefix stores disable key prefix. DisableKeyPrefix 存储账号封禁键前缀。
	DisableKeyPrefix = "disable:"
	// DisableServiceKeyPrefix stores service disable key prefix. DisableServiceKeyPrefix 存储服务封禁键前缀。
	DisableServiceKeyPrefix = "disable:service:"
	// DisableDeviceKeyPrefix stores device disable key prefix. DisableDeviceKeyPrefix 存储设备封禁键前缀。
	DisableDeviceKeyPrefix = "disable:device:"
	// DisableDeviceIDKeyPrefix stores concrete device disable key prefix. DisableDeviceIDKeyPrefix 存储具体设备封禁键前缀。
	DisableDeviceIDKeyPrefix = "disable:device:id:"
	// SessionKeyPrefix stores session key prefix. SessionKeyPrefix 存储会话键前缀。
	SessionKeyPrefix = "session:"
	// RenewKeyPrefix stores renew key prefix. RenewKeyPrefix 存储 Token 续期键前缀。
	RenewKeyPrefix = "renew:"
	// ActivePrefix stores active key prefix. ActivePrefix 存储 Token 活跃时间键前缀。
	ActivePrefix = "active:"
	// RefreshTokenKeyPrefix stores refresh token key prefix RefreshTokenKeyPrefix 存储刷新令牌键前缀
	RefreshTokenKeyPrefix = "refresh:"
	// TokenRefreshKeyPrefix stores access token to refresh token key prefix TokenRefreshKeyPrefix 存储访问令牌到刷新令牌的键前缀
	TokenRefreshKeyPrefix = "refresh:token:"

	// SessionKeyLoginID stores session login id key. SessionKeyLoginID 存储登录 ID 字段名。
	SessionKeyLoginID = "loginId"
	// SessionKeyDevice stores session device key. SessionKeyDevice 存储设备类型字段名。
	SessionKeyDevice = "device"
	// SessionKeyLoginTime stores session login time key. SessionKeyLoginTime 存储登录时间字段名。
	SessionKeyLoginTime = "loginTime"
	// SessionKeyPermissions stores permissions key. SessionKeyPermissions 存储权限列表字段名。
	SessionKeyPermissions = "permissions"
	// SessionKeyRoles stores roles key. SessionKeyRoles 存储角色列表字段名。
	SessionKeyRoles = "roles"

	// PermissionWildcard stores permission wildcard. PermissionWildcard 存储权限通配符。
	PermissionWildcard = "*"
	// PermissionSeparator stores permission separator. PermissionSeparator 存储权限段分隔符。
	PermissionSeparator = ":"
)

// TokenState defines token logical state. TokenState 定义 Token 逻辑状态。
type TokenState string

const (
	// TokenStateLogout indicates logout state. TokenStateLogout 表示主动登出状态。
	TokenStateLogout TokenState = "LOGOUT"
	// TokenStateKickOut indicates kickout state. TokenStateKickOut 表示被踢下线状态。
	TokenStateKickOut TokenState = "KICK_OUT"
	// TokenStateReplaced indicates replaced state. TokenStateReplaced 表示被顶下线状态。
	TokenStateReplaced TokenState = "REPLACED"
	// TokenStateActiveTimeout indicates inactive timeout state. TokenStateActiveTimeout 表示不活跃超时状态。
	TokenStateActiveTimeout TokenState = "ACTIVE_TIMEOUT"
)
