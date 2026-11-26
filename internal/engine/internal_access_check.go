package engine

import (
	"context"

	derror "github.com/Zany2/dtoken-go/errors"
	listener "github.com/Zany2/dtoken-go/event"
)

// CheckTokenAccess validates a token once for a combined check; a nonempty login ID distinguishes subsequent session or provider errors from credential failures. CheckTokenAccess 为组合检查单次校验 Token；非空登录 ID 用于区分后续 Session、提供器错误与凭证失败。
func CheckTokenAccess(ctx context.Context, m *Manager, tokenValue string, permissions, roles []string, matchAll, checkDisable bool) (string, error) {
	if m.IsClosed() {
		return "", derror.ErrManagerNotFound
	}
	sess, tokenInfo, err := m.checkLoginAndGetContextWithOptions(ctx, tokenValue, checkLoginOptions{tokenOnly: true, allowRenew: true, checkDisable: checkDisable})
	if err != nil {
		return "", err
	}
	loginID := tokenInfo.LoginID
	if len(permissions) == 0 && len(roles) == 0 {
		return loginID, nil
	}

	subject := AccessSubject{
		AuthType: m.config.AuthType,
		LoginID:  loginID,
		Device:   tokenInfo.Device,
		DeviceID: tokenInfo.DeviceID,
		Token:    tokenValue,
	}
	// Share fallback only within this operation, preserving permission-first short circuiting. 仅在本次操作内共享回退数据，保留权限优先的短路顺序。
	if len(permissions) > 0 {
		if err := m.checkPermissionList(ctx, subject, &sess, permissions, matchAll); err != nil {
			return loginID, err
		}
	}
	if len(roles) > 0 {
		if err := m.checkRoleList(ctx, subject, &sess, roles, matchAll); err != nil {
			return loginID, err
		}
	}
	return loginID, nil
}

// checkPermissionList applies permission policy to an already validated subject. checkPermissionList 对已校验的主体应用权限策略。
func (m *Manager) checkPermissionList(ctx context.Context, subject AccessSubject, cached **Session, permissions []string, matchAll bool) error {
	values, err := m.loadPermissions(ctx, cached, subject)
	if err != nil {
		return err
	}
	var allowed bool
	logic := listener.LogicOr
	if matchAll {
		allowed = m.hasAllPermissions(values, permissions)
		logic = listener.LogicAnd
	} else {
		allowed = m.hasAnyPermission(values, permissions)
	}
	m.triggerEvent(listener.EventPermissionCheck, subject.LoginID, subject.Device, subject.DeviceID, subject.Token, map[string]any{
		listener.ExtraKeyPermissions: permissions,
		listener.ExtraKeyLogic:       logic,
		listener.ExtraKeyResult:      allowed,
	})
	if !allowed {
		return derror.ErrPermissionDenied
	}
	return nil
}

// checkRoleList applies role policy to an already validated subject. checkRoleList 对已校验的主体应用角色策略。
func (m *Manager) checkRoleList(ctx context.Context, subject AccessSubject, cached **Session, roles []string, matchAll bool) error {
	values, err := m.loadRoles(ctx, cached, subject)
	if err != nil {
		return err
	}
	var allowed bool
	logic := listener.LogicOr
	if matchAll {
		allowed = m.hasAllRoles(values, roles)
		logic = listener.LogicAnd
	} else {
		allowed = m.hasAnyRole(values, roles)
	}
	m.triggerEvent(listener.EventRoleCheck, subject.LoginID, subject.Device, subject.DeviceID, subject.Token, map[string]any{
		listener.ExtraKeyRoles:  roles,
		listener.ExtraKeyLogic:  logic,
		listener.ExtraKeyResult: allowed,
	})
	if !allowed {
		return derror.ErrRoleDenied
	}
	return nil
}
