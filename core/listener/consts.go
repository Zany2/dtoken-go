// @Author daixk 2025/12/14 20:49:00
package listener

// Event represents the type of authentication event.
// Event 认证事件类型。
type Event string

const (
	// EventLogin fired when a user logs in.
	// EventLogin 用户登录事件。
	EventLogin Event = "login"

	// EventLogout fired when a user logs out.
	// EventLogout 用户登出事件。
	EventLogout Event = "logout"

	// EventKickout fired when a user is forcibly logged out.
	// EventKickout 用户被踢下线事件。
	EventKickout Event = "kickout"

	// EventReplace fired when a user is replaced by a new login.
	// EventReplace 用户被顶下线事件。
	EventReplace Event = "replace"

	// EventDisable fired when an account is disabled.
	// EventDisable 账号被禁用事件。
	EventDisable Event = "disable"

	// EventUntie fired when an account is re-enabled.
	// EventUntie 账号解禁事件。
	EventUntie Event = "untie"

	// EventRenew fired when a token is renewed.
	// EventRenew Token 续期事件。
	EventRenew Event = "renew"

	// EventCreateSession fired when a new session is created.
	// EventCreateSession Session 创建事件。
	EventCreateSession Event = "createSession"

	// EventDestroySession fired when a session is destroyed.
	// EventDestroySession Session 销毁事件。
	EventDestroySession Event = "destroySession"

	// EventPermissionCheck fired when a permission check is performed.
	// EventPermissionCheck 权限检查事件。
	EventPermissionCheck Event = "permissionCheck"

	// EventRoleCheck fired when a role check is performed.
	// EventRoleCheck 角色检查事件。
	EventRoleCheck Event = "roleCheck"

	// EventDisableService fired when a service is disabled for an account.
	// EventDisableService 账号服务被封禁事件。
	EventDisableService Event = "disableService"

	// EventUntieService fired when a service is re-enabled for an account.
	// EventUntieService 账号服务解禁事件。
	EventUntieService Event = "untieService"

	// EventAll is a wildcard event that matches all events.
	// EventAll 通配符事件（匹配所有事件）。
	EventAll Event = "*"
)

// Extra field keys for event data
// 事件数据 Extra 字段的键名常量
const (
	// ExtraKeyPermission 单个权限字段
	ExtraKeyPermission = "permission"

	// ExtraKeyPermissions 多个权限字段
	ExtraKeyPermissions = "permissions"

	// ExtraKeyRole 单个角色字段
	ExtraKeyRole = "role"

	// ExtraKeyRoles 多个角色字段
	ExtraKeyRoles = "roles"

	// ExtraKeyLogic 逻辑类型字段（AND/OR）
	ExtraKeyLogic = "logic"

	// ExtraKeyResult 检查结果字段
	ExtraKeyResult = "result"

	// ExtraKeyService 服务/业务模块字段
	ExtraKeyService = "service"

	// ExtraKeyLevel 封禁等级字段
	ExtraKeyLevel = "level"
)

// Logic types for permission/role checks
// 权限/角色检查的逻辑类型常量
const (
	// LogicAnd AND 逻辑（需要满足所有条件）
	LogicAnd = "AND"

	// LogicOr OR 逻辑（满足任一条件即可）
	LogicOr = "OR"
)
