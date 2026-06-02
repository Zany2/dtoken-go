// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"context"

	"github.com/Zany2/dtoken-go/core/derror"
)

// Terminate applies one terminal operation by options Terminate 根据选项执行一次终端下线操作
func (m *Manager) Terminate(ctx context.Context, opts TerminateOptions) error {
	// Use logout as the default terminal action 默认使用注销作为终端操作
	action := opts.Action
	if action == "" {
		action = TerminateActionLogout
	}

	// Prefer direct token operations when token is specified 指定 Token 时优先按 Token 操作
	if opts.Token != "" {
		switch action {
		case TerminateActionLogout:
			return m.Logout(ctx, opts.Token)
		case TerminateActionKickout:
			return m.Kickout(ctx, opts.Token)
		case TerminateActionReplace:
			return m.Replace(ctx, opts.Token)
		default:
			return derror.ErrInvalidParam
		}
	}

	// Require login ID for account or device scoped operations 账号或设备范围操作必须提供登录 ID
	if opts.LoginID == "" {
		return derror.ErrIDIsEmpty
	}

	// Dispatch operation by terminal action 根据终端操作类型分发处理
	switch action {
	case TerminateActionLogout:
		return m.terminateLogout(ctx, opts)
	case TerminateActionKickout:
		return m.terminateKickout(ctx, opts)
	case TerminateActionReplace:
		return m.terminateReplace(ctx, opts)
	default:
		return derror.ErrInvalidParam
	}
}

// terminateLogout dispatches logout by account or device scope terminateLogout 按账号或设备范围分发注销操作
func (m *Manager) terminateLogout(ctx context.Context, opts TerminateOptions) error {
	if opts.Device != "" && opts.DeviceID != "" {
		return m.LogoutByDeviceAndDeviceId(ctx, opts.LoginID, opts.Device, opts.DeviceID)
	}
	if opts.Device != "" {
		return m.LogoutByDevice(ctx, opts.LoginID, opts.Device)
	}
	return m.LogoutByLoginID(ctx, opts.LoginID)
}

// terminateKickout dispatches kickout by account or device scope terminateKickout 按账号或设备范围分发踢下线操作
func (m *Manager) terminateKickout(ctx context.Context, opts TerminateOptions) error {
	if opts.Device != "" && opts.DeviceID != "" {
		return m.KickoutByDeviceAndDeviceId(ctx, opts.LoginID, opts.Device, opts.DeviceID)
	}
	if opts.Device != "" {
		return m.KickoutByDevice(ctx, opts.LoginID, opts.Device)
	}
	return m.KickoutByLoginID(ctx, opts.LoginID)
}

// terminateReplace dispatches replace by account or device scope terminateReplace 按账号或设备范围分发顶替下线操作
func (m *Manager) terminateReplace(ctx context.Context, opts TerminateOptions) error {
	if opts.Device != "" && opts.DeviceID != "" {
		return m.ReplaceByDeviceAndDeviceId(ctx, opts.LoginID, opts.Device, opts.DeviceID)
	}
	if opts.Device != "" {
		return m.ReplaceByDevice(ctx, opts.LoginID, opts.Device)
	}
	return m.ReplaceByLoginID(ctx, opts.LoginID)
}
