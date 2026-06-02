// @Author daixk 2025/12/22 15:56:00
package dtoken

import (
	"time"

	"github.com/Zany2/dtoken-go/core/config"
)

// LoginOptions describes a typed login request. LoginOptions 描述类型化登录请求。
type LoginOptions struct {
	// AuthType stores the auth namespace. AuthType 存储认证命名空间。
	AuthType string
	// LoginID stores the subject id. LoginID 存储主体 ID。
	LoginID string
	// Device stores the device type. Device 存储设备类型。
	Device string
	// DeviceID stores the concrete device id. DeviceID 存储具体设备 ID。
	DeviceID string
	// Timeout stores custom token timeout. Timeout 存储自定义 Token 过期时间。
	Timeout time.Duration
	// ActiveTimeout stores custom inactive timeout. ActiveTimeout 存储自定义不活跃超时。
	ActiveTimeout time.Duration
	// Token stores a pre-created token. Token 存储预创建 Token。
	Token string
	// Extra stores token extension data. Extra 存储 Token 扩展数据。
	Extra map[string]any
	// TerminalExtra stores terminal extension data. TerminalExtra 存储终端扩展数据。
	TerminalExtra map[string]any
	// IsConcurrent overrides concurrent login switch. IsConcurrent 覆盖并发登录开关。
	IsConcurrent *bool
	// IsShare overrides shared token switch. IsShare 覆盖共享 Token 开关。
	IsShare *bool
	// MaxLoginCount overrides max login count. MaxLoginCount 覆盖最大登录数量。
	MaxLoginCount *int64
	// ReplacedLoginExitMode overrides non-concurrent strategy. ReplacedLoginExitMode 覆盖非并发登录处理策略。
	ReplacedLoginExitMode *config.ReplacedLoginExitMode
	// OverflowLogoutMode overrides overflow terminal mode. OverflowLogoutMode 覆盖超限终端处理模式。
	OverflowLogoutMode *config.LogoutMode
}

// LogoutOptions describes a typed logout or terminal operation. LogoutOptions 描述类型化登出或终端操作。
type LogoutOptions struct {
	// AuthType stores the auth namespace. AuthType 存储认证命名空间。
	AuthType string
	// LoginID stores the subject id. LoginID 存储主体 ID。
	LoginID string
	// Token stores the target token. Token 存储目标 Token。
	Token string
	// Device stores the device type. Device 存储设备类型。
	Device string
	// DeviceID stores the concrete device id. DeviceID 存储具体设备 ID。
	DeviceID string
}

// TerminateOptions describes a typed terminal operation. TerminateOptions 描述类型化终端操作。
type TerminateOptions = LogoutOptions

// DisableOptions describes an account disable request. DisableOptions 描述账号封禁请求。
type DisableOptions struct {
	// AuthType stores the auth namespace. AuthType 存储认证命名空间。
	AuthType string
	// LoginID stores the subject id. LoginID 存储主体 ID。
	LoginID string
	// Duration stores disable duration. Duration 存储封禁时长。
	Duration time.Duration
	// Reason stores disable reason. Reason 存储封禁原因。
	Reason string
}

// ServiceDisableOptions describes a service-scoped disable request. ServiceDisableOptions 描述服务维度封禁请求。
type ServiceDisableOptions struct {
	// AuthType stores the auth namespace. AuthType 存储认证命名空间。
	AuthType string
	// LoginID stores the subject id. LoginID 存储主体 ID。
	LoginID string
	// Service stores disabled service name. Service 存储被封禁服务名。
	Service string
	// Level stores disable level. Level 存储封禁等级。
	Level int
	// Duration stores disable duration. Duration 存储封禁时长。
	Duration time.Duration
	// Reason stores disable reason. Reason 存储封禁原因。
	Reason string
}

// DeviceDisableOptions describes a device-scoped disable request. DeviceDisableOptions 描述设备维度封禁请求。
type DeviceDisableOptions struct {
	// AuthType stores the auth namespace. AuthType 存储认证命名空间。
	AuthType string
	// LoginID stores the subject id. LoginID 存储主体 ID。
	LoginID string
	// Device stores disabled device type. Device 存储被封禁设备类型。
	Device string
	// DeviceID stores disabled concrete device id. DeviceID 存储被封禁具体设备 ID。
	DeviceID string
	// Duration stores disable duration. Duration 存储封禁时长。
	Duration time.Duration
	// Reason stores disable reason. Reason 存储封禁原因。
	Reason string
}

// PermissionOptions describes permission mutations and checks. PermissionOptions 描述权限变更和校验。
type PermissionOptions struct {
	// AuthType stores the auth namespace. AuthType 存储认证命名空间。
	AuthType string
	// LoginID stores the subject id. LoginID 存储主体 ID。
	LoginID string
	// Token stores the target token. Token 存储目标 Token。
	Token string
	// Permission stores one permission. Permission 存储单个权限。
	Permission string
	// Permissions stores multiple permissions. Permissions 存储多个权限。
	Permissions []string
}

// RoleOptions describes role mutations and checks. RoleOptions 描述角色变更和校验。
type RoleOptions struct {
	// AuthType stores the auth namespace. AuthType 存储认证命名空间。
	AuthType string
	// LoginID stores the subject id. LoginID 存储主体 ID。
	LoginID string
	// Token stores the target token. Token 存储目标 Token。
	Token string
	// Role stores one role. Role 存储单个角色。
	Role string
	// Roles stores multiple roles. Roles 存储多个角色。
	Roles []string
}
