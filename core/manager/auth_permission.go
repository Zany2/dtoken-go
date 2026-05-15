// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"context"
	"fmt"
	"strings"

	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/listener"
)

// AddPermissions adds permissions to a user. AddPermissions 为用户添加权限。
func (m *Manager) AddPermissions(ctx context.Context, loginID string, permissions []string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	unlock := m.lockLoginWrite(loginID)
	defer unlock()

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		return err
	}

	sess.addPermissions(permissions...)
	return m.saveToStorage(ctx, m.getSessionKey(loginID), *sess)
}

// AddPermissionsByToken adds permissions to a user by token. AddPermissionsByToken 根据 Token 为用户添加权限。
func (m *Manager) AddPermissionsByToken(ctx context.Context, tokenValue string, permissions []string) error {
	_, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		return err
	}

	unlock := m.lockLoginWrite(tokenInfo.LoginID)
	defer unlock()

	if err = m.ensureTerminalTokenAlive(ctx, tokenValue); err != nil {
		return err
	}
	sess, err := m.GetSessionByToken(ctx, tokenValue)
	if err != nil {
		return err
	}

	sess.addPermissions(permissions...)
	return m.saveToStorage(ctx, m.getSessionKey(sess.LoginID), *sess)
}

// RemovePermissions removes permissions from a user. RemovePermissions 删除用户的指定权限。
func (m *Manager) RemovePermissions(ctx context.Context, loginID string, permissions []string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	unlock := m.lockLoginWrite(loginID)
	defer unlock()

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		return err
	}

	sess.removePermissions(permissions...)
	return m.saveToStorage(ctx, m.getSessionKey(loginID), *sess)
}

// RemovePermissionsByToken removes permissions from a user by token. RemovePermissionsByToken 根据 Token 删除用户的指定权限。
func (m *Manager) RemovePermissionsByToken(ctx context.Context, tokenValue string, permissions []string) error {
	_, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		return err
	}

	unlock := m.lockLoginWrite(tokenInfo.LoginID)
	defer unlock()

	if err = m.ensureTerminalTokenAlive(ctx, tokenValue); err != nil {
		return err
	}
	sess, err := m.GetSessionByToken(ctx, tokenValue)
	if err != nil {
		return err
	}

	sess.removePermissions(permissions...)
	return m.saveToStorage(ctx, m.getSessionKey(sess.LoginID), *sess)
}

// GetPermissions retrieves the permission list for a user. GetPermissions 获取用户的权限列表。
func (m *Manager) GetPermissions(ctx context.Context, loginID string) ([]string, error) {
	if loginID == "" {
		return nil, derror.ErrIDIsEmpty
	}

	subject := AccessSubject{LoginID: loginID}
	if m.accessProvider != nil {
		permissions, err := m.providerPermissions(ctx, nil, subject)
		if err != nil {
			return nil, err
		}
		if permissions != nil {
			return permissions, nil
		}
	}

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		return nil, err
	}

	return sess.Permissions, nil
}

// GetPermissionsByToken retrieves the permission list by token. GetPermissionsByToken 根据 Token 获取权限列表。
func (m *Manager) GetPermissionsByToken(ctx context.Context, tokenValue string) ([]string, error) {
	sess, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		return nil, err
	}

	return m.loadPermissions(ctx, sess.Permissions, AccessSubject{
		LoginID:  sess.LoginID,
		Device:   tokenInfo.Device,
		DeviceID: tokenInfo.DeviceId,
		Token:    tokenValue,
	})
}

// HasPermission checks if a user has a specific permission. HasPermission 检查用户是否拥有指定权限。
func (m *Manager) HasPermission(ctx context.Context, loginID string, permission string) bool {
	if loginID == "" {
		return false
	}

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		m.logger.Errorf("manager.HasPermission: failed to get session, loginID=%s, error=%v", loginID, err)
		return false
	}

	permissions := m.resolvePermissions(ctx, sess.Permissions, AccessSubject{LoginID: loginID})
	hasPermission := m.hasPermissionInList(permissions, permission)

	m.triggerEvent(listener.EventPermissionCheck, loginID, "", "", "", map[string]any{
		listener.ExtraKeyPermission: permission,
		listener.ExtraKeyResult:     hasPermission,
	})

	return hasPermission
}

// HasPermissionByToken checks if a user has a specific permission by token. HasPermissionByToken 根据 Token 检查用户是否拥有指定权限。
func (m *Manager) HasPermissionByToken(ctx context.Context, tokenValue string, permission string) bool {
	sess, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		m.logger.Errorf("manager.HasPermissionByToken: failed to get token info, token=%s, error=%v", tokenValue, err)
		return false
	}

	device, deviceId := tokenInfo.Device, tokenInfo.DeviceId
	permissions := m.resolvePermissions(ctx, sess.Permissions, AccessSubject{
		LoginID:  sess.LoginID,
		Device:   device,
		DeviceID: deviceId,
		Token:    tokenValue,
	})
	hasPermission := m.hasPermissionInList(permissions, permission)

	m.triggerEvent(listener.EventPermissionCheck, sess.LoginID, device, deviceId, tokenValue, map[string]any{
		listener.ExtraKeyPermission: permission,
		listener.ExtraKeyResult:     hasPermission,
	})

	return hasPermission
}

// HasPermissionsAnd checks if a user has all specified permissions. HasPermissionsAnd 检查用户是否拥有全部指定权限。
func (m *Manager) HasPermissionsAnd(ctx context.Context, loginID string, permissions []string) bool {
	if loginID == "" {
		return false
	}

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		m.logger.Errorf("manager.HasPermissionsAnd: failed to get session, loginID=%s, error=%v", loginID, err)
		return false
	}

	permList := m.resolvePermissions(ctx, sess.Permissions, AccessSubject{LoginID: loginID})
	hasAll := m.hasAllPermissions(permList, permissions)

	m.triggerEvent(listener.EventPermissionCheck, loginID, "", "", "", map[string]any{
		listener.ExtraKeyPermissions: permissions,
		listener.ExtraKeyLogic:       listener.LogicAnd,
		listener.ExtraKeyResult:      hasAll,
	})

	return hasAll
}

// HasPermissionsAndByToken checks if a token user has all specified permissions. HasPermissionsAndByToken 根据 Token 检查全部权限。
func (m *Manager) HasPermissionsAndByToken(ctx context.Context, tokenValue string, permissions []string) bool {
	sess, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		m.logger.Errorf("manager.HasPermissionsAndByToken: failed to get token info, token=%s, error=%v", tokenValue, err)
		return false
	}

	device, deviceId := tokenInfo.Device, tokenInfo.DeviceId
	permList := m.resolvePermissions(ctx, sess.Permissions, AccessSubject{
		LoginID:  sess.LoginID,
		Device:   device,
		DeviceID: deviceId,
		Token:    tokenValue,
	})
	hasAll := m.hasAllPermissions(permList, permissions)

	m.triggerEvent(listener.EventPermissionCheck, sess.LoginID, device, deviceId, tokenValue, map[string]any{
		listener.ExtraKeyPermissions: permissions,
		listener.ExtraKeyLogic:       listener.LogicAnd,
		listener.ExtraKeyResult:      hasAll,
	})

	return hasAll
}

// HasPermissionsOr checks if a user has any specified permission. HasPermissionsOr 检查用户是否拥有任一指定权限。
func (m *Manager) HasPermissionsOr(ctx context.Context, loginID string, permissions []string) bool {
	if loginID == "" {
		return false
	}
	if len(permissions) == 0 {
		return true
	}

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		m.logger.Errorf("manager.HasPermissionsOr: failed to get session, loginID=%s, error=%v", loginID, err)
		return false
	}

	permList := m.resolvePermissions(ctx, sess.Permissions, AccessSubject{LoginID: loginID})
	hasAny := m.hasAnyPermission(permList, permissions)

	m.triggerEvent(listener.EventPermissionCheck, loginID, "", "", "", map[string]any{
		listener.ExtraKeyPermissions: permissions,
		listener.ExtraKeyLogic:       listener.LogicOr,
		listener.ExtraKeyResult:      hasAny,
	})

	return hasAny
}

// HasPermissionsOrByToken checks if a token user has any specified permission. HasPermissionsOrByToken 根据 Token 检查任一权限。
func (m *Manager) HasPermissionsOrByToken(ctx context.Context, tokenValue string, permissions []string) bool {
	if len(permissions) == 0 {
		return true
	}

	sess, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		m.logger.Errorf("manager.HasPermissionsOrByToken: failed to get token info, token=%s, error=%v", tokenValue, err)
		return false
	}

	device, deviceId := tokenInfo.Device, tokenInfo.DeviceId
	permList := m.resolvePermissions(ctx, sess.Permissions, AccessSubject{
		LoginID:  sess.LoginID,
		Device:   device,
		DeviceID: deviceId,
		Token:    tokenValue,
	})
	hasAny := m.hasAnyPermission(permList, permissions)

	m.triggerEvent(listener.EventPermissionCheck, sess.LoginID, device, deviceId, tokenValue, map[string]any{
		listener.ExtraKeyPermissions: permissions,
		listener.ExtraKeyLogic:       listener.LogicOr,
		listener.ExtraKeyResult:      hasAny,
	})

	return hasAny
}

// CheckPermission checks if a user has a specific permission. CheckPermission 校验用户是否拥有指定权限。
func (m *Manager) CheckPermission(ctx context.Context, loginID string, permission string) error {
	if !m.HasPermission(ctx, loginID, permission) {
		return fmt.Errorf("%w: %s", derror.ErrPermissionDenied, permission)
	}
	return nil
}

// CheckPermissionAnd checks if a user has all specified permissions. CheckPermissionAnd 校验用户是否拥有全部权限。
func (m *Manager) CheckPermissionAnd(ctx context.Context, loginID string, permissions []string) error {
	if !m.HasPermissionsAnd(ctx, loginID, permissions) {
		return derror.ErrPermissionDenied
	}
	return nil
}

// CheckPermissionOr checks if a user has any specified permission. CheckPermissionOr 校验用户是否拥有任一权限。
func (m *Manager) CheckPermissionOr(ctx context.Context, loginID string, permissions []string) error {
	if !m.HasPermissionsOr(ctx, loginID, permissions) {
		return derror.ErrPermissionDenied
	}
	return nil
}

// matchPermission matches permission with wildcard support. matchPermission 支持通配符权限匹配。
func (m *Manager) matchPermission(pattern, permission string) bool {
	if pattern == PermissionWildcard || pattern == permission {
		return true
	}
	if !strings.Contains(pattern, PermissionWildcard) {
		return false
	}

	separator := PermissionSeparator
	if strings.Contains(pattern, "/") {
		separator = "/"
	}

	patternParts := strings.Split(pattern, separator)
	permParts := strings.Split(permission, separator)
	if len(patternParts) != len(permParts) {
		return false
	}

	for i := range patternParts {
		if patternParts[i] == PermissionWildcard {
			continue
		}
		if patternParts[i] != permParts[i] {
			return false
		}
	}
	return true
}

// hasPermissionInList checks if permission exists in permission list. hasPermissionInList 判断权限是否存在。
func (m *Manager) hasPermissionInList(perms []string, permission string) bool {
	for _, p := range perms {
		if m.matchPermission(p, permission) {
			return true
		}
	}
	return false
}

func (m *Manager) hasAllPermissions(perms []string, required []string) bool {
	for _, need := range required {
		if !m.hasPermissionInList(perms, need) {
			return false
		}
	}
	return true
}

func (m *Manager) hasAnyPermission(perms []string, required []string) bool {
	for _, need := range required {
		if m.hasPermissionInList(perms, need) {
			return true
		}
	}
	return false
}
