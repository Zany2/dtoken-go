// @Author daixk 2025/12/22 15:56:00
package dtoken

import "time"

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
