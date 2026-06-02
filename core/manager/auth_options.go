// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"time"

	"github.com/Zany2/dtoken-go/core/config"
)

// LoginOptions describes one login operation LoginOptions 描述一次登录操作
type LoginOptions struct {
	// LoginID stores the subject id LoginID 存储登录主体 ID
	LoginID string `json:"loginId"`
	// Device stores the device type Device 存储设备类型
	Device string `json:"device"`
	// DeviceID stores the concrete device id DeviceID 存储具体设备 ID
	DeviceID string `json:"deviceId"`
	// Timeout overrides token timeout for this login Timeout 覆盖本次登录的 token 有效期
	Timeout time.Duration `json:"timeout"`
	// ActiveTimeout overrides active timeout for this token ActiveTimeout 覆盖本次 token 的活跃超时
	ActiveTimeout time.Duration `json:"activeTimeout"`
	// Token uses a pre-created token value Token 使用预生成 token 值
	Token string `json:"token"`
	// Extra stores token extension data Extra 存储 token 扩展数据
	Extra map[string]any `json:"extra,omitempty"`
	// TerminalExtra stores terminal extension data TerminalExtra 存储终端扩展数据
	TerminalExtra map[string]any `json:"terminalExtra,omitempty"`
	// IsConcurrent overrides concurrent login switch IsConcurrent 覆盖并发登录开关
	IsConcurrent *bool `json:"isConcurrent,omitempty"`
	// IsShare overrides shared token switch IsShare 覆盖共享 token 开关
	IsShare *bool `json:"isShare,omitempty"`
	// MaxLoginCount overrides max login count MaxLoginCount 覆盖最大登录数量
	MaxLoginCount *int64 `json:"maxLoginCount,omitempty"`
	// ReplacedLoginExitMode overrides non-concurrent strategy ReplacedLoginExitMode 覆盖非并发登录处理策略
	ReplacedLoginExitMode *config.ReplacedLoginExitMode `json:"replacedLoginExitMode,omitempty"`
	// OverflowLogoutMode overrides overflow terminal mode OverflowLogoutMode 覆盖超限终端处理模式
	OverflowLogoutMode *config.LogoutMode `json:"overflowLogoutMode,omitempty"`
}

// TerminateAction defines terminal operation behavior TerminateAction 定义终端操作行为
type TerminateAction string

const (
	// TerminateActionLogout deletes terminal token TerminateActionLogout 删除终端 token
	TerminateActionLogout TerminateAction = "logout"
	// TerminateActionKickout marks terminal as kicked out TerminateActionKickout 标记终端被踢下线
	TerminateActionKickout TerminateAction = "kickout"
	// TerminateActionReplace marks terminal as replaced TerminateActionReplace 标记终端被顶替
	TerminateActionReplace TerminateAction = "replace"
)

// TerminateOptions describes one terminal operation TerminateOptions 描述一次终端下线操作
type TerminateOptions struct {
	// Action stores terminal operation behavior Action 存储终端操作行为
	Action TerminateAction `json:"action"`
	// LoginID stores the subject id LoginID 存储登录主体 ID
	LoginID string `json:"loginId"`
	// Token stores the target token Token 存储目标 token
	Token string `json:"token"`
	// Device stores target device type Device 存储目标设备类型
	Device string `json:"device"`
	// DeviceID stores target concrete device id DeviceID 存储目标具体设备 ID
	DeviceID string `json:"deviceId"`
}

type loginPolicy struct {
	isConcurrent          bool
	isShare               bool
	maxLoginCount         int64
	replacedLoginExitMode config.ReplacedLoginExitMode
	overflowLogoutMode    config.LogoutMode
}

// resolveLoginPolicy merges global config with per-login overrides resolveLoginPolicy 合并全局配置和单次登录覆盖项
func (m *Manager) resolveLoginPolicy(opts LoginOptions) loginPolicy {
	policy := loginPolicy{
		isConcurrent:          m.config.IsConcurrent,
		isShare:               m.config.IsShare,
		maxLoginCount:         m.config.MaxLoginCount,
		replacedLoginExitMode: m.config.ReplacedLoginExitMode,
		overflowLogoutMode:    m.config.OverflowLogoutMode,
	}
	if opts.IsConcurrent != nil {
		policy.isConcurrent = *opts.IsConcurrent
	}
	if opts.IsShare != nil {
		policy.isShare = *opts.IsShare
	}
	if opts.MaxLoginCount != nil {
		policy.maxLoginCount = *opts.MaxLoginCount
	}
	if opts.ReplacedLoginExitMode != nil {
		policy.replacedLoginExitMode = *opts.ReplacedLoginExitMode
	}
	if opts.OverflowLogoutMode != nil {
		policy.overflowLogoutMode = *opts.OverflowLogoutMode
	}
	return policy
}
