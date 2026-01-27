// @Author daixk 2025/12/4 17:58:00
package manager

// 存储键和默认值常量
const (
	DisableKeyPrefix = "disable:" // DisableKeyPrefix 禁用状态存储前缀
	SessionKeyPrefix = "session:" // SessionKeyPrefix 会话存储前缀
	RenewKeyPrefix   = "renew:"   // RenewKeyPrefix Token 续期存储前缀

	SessionKeyLoginID     = "loginId"     // SessionKeyLoginID 登录 ID
	SessionKeyDevice      = "device"      // SessionKeyDevice 设备类型
	SessionKeyLoginTime   = "loginTime"   // SessionKeyLoginTime 登录时间
	SessionKeyPermissions = "permissions" // SessionKeyPermissions 权限列表
	SessionKeyRoles       = "roles"       // SessionKeyRoles 角色列表

	PermissionWildcard  = "*" // PermissionWildcard 全局权限通配符
	PermissionSeparator = ":" // PermissionSeparator 权限段分隔符
)

// TokenState 表示 Token 的逻辑状态
type TokenState string

const (
	TokenStateLogout   TokenState = "LOGOUT"   // TokenStateLogout 主动登出
	TokenStateKickOut  TokenState = "KICK_OUT" // TokenStateKickOut 被踢下线
	TokenStateReplaced TokenState = "REPLACED" // TokenStateReplaced 被顶下线
)
