// @Author daixk 2026/1/21 10:43:00
package manager

import (
	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
)

// Manager 认证管理器
type Manager struct {
	config *config.Config // config 全局认证配置

	generator  adapter.Generator // generator Token 生成器
	storage    adapter.Storage   // storage 存储适配器
	serializer adapter.Codec     // serializer 编解码器适配器
	logger     adapter.Log       // logger 日志适配器
	pool       adapter.Pool      // pool 异步任务协程池组件

	CustomPermissionListFunc func(loginID, authType string) ([]string, error) // CustomPermissionListFunc 自定义权限列表获取函数
	CustomRoleListFunc       func(loginID, authType string) ([]string, error) // CustomRoleListFunc 自定义角色列表获取函数
}

// TokenInfo Token 信息
type TokenInfo struct {
	AuthType   string `json:"authType"`   // AuthType 认证体系类型
	LoginID    string `json:"loginId"`    // LoginID 登录 ID
	Device     string `json:"device"`     // Device 设备类型
	DeviceId   string `json:"deviceId"`   // DeviceId 设备 ID
	CreateTime int64  `json:"createTime"` // CreateTime 创建时间戳
}

// Session 会话对象，用于存储用户数据
type Session struct {
	AuthType      string         `json:"authType"`      // AuthType 认证体系类型
	LoginID       string         `json:"loginID"`       // LoginID 登录 ID
	CreateTime    int64          `json:"createTime"`    // CreateTime 创建时间
	TerminalInfos []TerminalInfo `json:"terminalInfos"` // TerminalInfos 终端信息列表
	Permissions   []string       `json:"permissions"`   // Permissions 权限列表
	Roles         []string       `json:"roles"`         // Roles 角色列表
}

// TerminalInfo 终端信息
type TerminalInfo struct {
	Token      string `json:"token"`      // Token 令牌值
	LoginID    string `json:"loginId"`    // LoginID 登录 ID
	Device     string `json:"device"`     // Device 设备类型
	DeviceId   string `json:"deviceId"`   // DeviceId 设备 ID
	CreateTime int64  `json:"createTime"` // CreateTime Token 创建时间戳
}

// DisableInfo 账号封禁信息
type DisableInfo struct {
	DisableTime   int64  `json:"disableTime"`   // DisableTime 封禁时间戳
	DisableReason string `json:"disableReason"` // DisableReason 封禁原因
}
