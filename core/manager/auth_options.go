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

// loginPolicy stores the effective login policy for one login call. loginPolicy 存储单次登录最终生效的登录策略。
type loginPolicy struct {
	isConcurrent          bool                         // isConcurrent controls whether multiple terminals may coexist. isConcurrent 控制是否允许多端共存。
	isShare               bool                         // isShare controls whether same-device login may reuse a token. isShare 控制同设备登录是否可复用 Token。
	maxLoginCount         int64                        // maxLoginCount limits terminal count in the configured scope. maxLoginCount 限制配置作用域内的终端数量。
	replacedLoginExitMode config.ReplacedLoginExitMode // replacedLoginExitMode controls non-concurrent replacement behavior. replacedLoginExitMode 控制非并发登录的顶替行为。
	overflowLogoutMode    config.LogoutMode            // overflowLogoutMode controls how overflowed terminals are retired. overflowLogoutMode 控制超限终端的下线方式。
}

// loginInternalOptions carries manager-only login controls. loginInternalOptions 承载 manager 内部登录控制参数。
type loginInternalOptions struct {
	skipConcurrencyControl bool // skipConcurrencyControl skips login concurrency handling. skipConcurrencyControl 跳过登录并发策略处理。
}

// concurrencyResult describes the outcome of concurrency handling. concurrencyResult 描述并发策略处理结果。
type concurrencyResult struct {
	reuseToken       string // reuseToken stores a shared token when login can reuse one. reuseToken 保存可复用的共享 Token。
	handled          bool   // handled reports whether concurrency logic already took an action. handled 表示并发逻辑是否已经处理。
	destroyedSession bool   // destroyedSession reports whether the whole old session was removed. destroyedSession 表示是否移除了整个旧会话。
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
