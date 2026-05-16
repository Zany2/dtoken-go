// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"context"
	"fmt"

	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/listener"
)

// AddRoles adds roles to a user. AddRoles 为用户添加角色。
func (m *Manager) AddRoles(ctx context.Context, loginID string, roles []string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	unlock := m.lockLoginWrite(loginID)
	defer func() { unlock() }()

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		return err
	}

	sess.addRoles(roles...)
	return m.saveToStorage(ctx, m.getSessionKey(loginID), *sess)
}

// AddRolesByToken adds roles to a user by token. AddRolesByToken 根据 Token 为用户添加角色。
func (m *Manager) AddRolesByToken(ctx context.Context, tokenValue string, roles []string) error {
	_, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		return err
	}

	unlock := m.lockLoginWrite(tokenInfo.LoginID)
	defer func() { unlock() }()

	if err = m.ensureTerminalTokenAlive(ctx, tokenValue); err != nil {
		return err
	}
	sess, err := m.GetSessionByToken(ctx, tokenValue)
	if err != nil {
		return err
	}

	sess.addRoles(roles...)
	return m.saveToStorage(ctx, m.getSessionKey(sess.LoginID), *sess)
}

// RemoveRoles removes roles from a user. RemoveRoles 删除用户的指定角色。
func (m *Manager) RemoveRoles(ctx context.Context, loginID string, roles []string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	unlock := m.lockLoginWrite(loginID)
	defer func() { unlock() }()

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		return err
	}

	sess.removeRoles(roles...)
	return m.saveToStorage(ctx, m.getSessionKey(loginID), *sess)
}

// RemoveRolesByToken removes roles from a user by token. RemoveRolesByToken 根据 Token 删除用户的指定角色。
func (m *Manager) RemoveRolesByToken(ctx context.Context, tokenValue string, roles []string) error {
	_, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		return err
	}

	unlock := m.lockLoginWrite(tokenInfo.LoginID)
	defer func() { unlock() }()

	if err = m.ensureTerminalTokenAlive(ctx, tokenValue); err != nil {
		return err
	}
	sess, err := m.GetSessionByToken(ctx, tokenValue)
	if err != nil {
		return err
	}

	sess.removeRoles(roles...)
	return m.saveToStorage(ctx, m.getSessionKey(sess.LoginID), *sess)
}

// GetRoles retrieves the role list for a user. GetRoles 获取用户的角色列表。
func (m *Manager) GetRoles(ctx context.Context, loginID string) ([]string, error) {
	if loginID == "" {
		return nil, derror.ErrIDIsEmpty
	}
	return m.loadRolesByLoginID(ctx, loginID)
}

// GetRolesByToken retrieves the role list by token. GetRolesByToken 根据 Token 获取角色列表。
func (m *Manager) GetRolesByToken(ctx context.Context, tokenValue string) ([]string, error) {
	sess, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		return nil, err
	}

	return m.loadRoles(ctx, sess.Roles, AccessSubject{
		LoginID:  sess.LoginID,
		Device:   tokenInfo.Device,
		DeviceID: tokenInfo.DeviceId,
		Token:    tokenValue,
	})
}

// HasRole checks if a user has a specific role. HasRole 检查用户是否拥有指定角色。
func (m *Manager) HasRole(ctx context.Context, loginID string, role string) bool {
	if loginID == "" || role == "" {
		return false
	}

	roles, err := m.loadRolesByLoginID(ctx, loginID)
	if err != nil {
		m.logger.Errorf("manager.HasRole: failed to load roles, loginID=%s, error=%v", loginID, err)
		return false
	}
	hasRole := hasRoleInList(roles, role)

	m.triggerEvent(listener.EventRoleCheck, loginID, "", "", "", map[string]any{
		listener.ExtraKeyRole:   role,
		listener.ExtraKeyResult: hasRole,
	})

	return hasRole
}

// HasRoleByToken checks if a user has a specific role by token. HasRoleByToken 根据 Token 检查用户是否拥有指定角色。
func (m *Manager) HasRoleByToken(ctx context.Context, tokenValue string, role string) bool {
	if role == "" {
		return false
	}

	sess, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		m.logger.Errorf("manager.HasRoleByToken: failed to get token info, token=%s, error=%v", tokenValue, err)
		return false
	}

	device, deviceId := tokenInfo.Device, tokenInfo.DeviceId
	roles := m.resolveRoles(ctx, sess.Roles, AccessSubject{
		LoginID:  sess.LoginID,
		Device:   device,
		DeviceID: deviceId,
		Token:    tokenValue,
	})
	hasRole := hasRoleInList(roles, role)

	m.triggerEvent(listener.EventRoleCheck, sess.LoginID, device, deviceId, tokenValue, map[string]any{
		listener.ExtraKeyRole:   role,
		listener.ExtraKeyResult: hasRole,
	})

	return hasRole
}

// HasRolesAnd checks if a user has all specified roles. HasRolesAnd 检查用户是否拥有全部指定角色。
func (m *Manager) HasRolesAnd(ctx context.Context, loginID string, roles []string) bool {
	if loginID == "" {
		return false
	}

	roleList, err := m.loadRolesByLoginID(ctx, loginID)
	if err != nil {
		m.logger.Errorf("manager.HasRolesAnd: failed to load roles, loginID=%s, error=%v", loginID, err)
		return false
	}
	hasAll := hasAllRoles(roleList, roles)

	m.triggerEvent(listener.EventRoleCheck, loginID, "", "", "", map[string]any{
		listener.ExtraKeyRoles:  roles,
		listener.ExtraKeyLogic:  listener.LogicAnd,
		listener.ExtraKeyResult: hasAll,
	})

	return hasAll
}

// HasRolesAndByToken checks if a token user has all specified roles. HasRolesAndByToken 根据 Token 检查全部角色。
func (m *Manager) HasRolesAndByToken(ctx context.Context, tokenValue string, roles []string) bool {
	sess, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		m.logger.Errorf("manager.HasRolesAndByToken: failed to get token info, token=%s, error=%v", tokenValue, err)
		return false
	}

	device, deviceId := tokenInfo.Device, tokenInfo.DeviceId
	roleList := m.resolveRoles(ctx, sess.Roles, AccessSubject{
		LoginID:  sess.LoginID,
		Device:   device,
		DeviceID: deviceId,
		Token:    tokenValue,
	})
	hasAll := hasAllRoles(roleList, roles)

	m.triggerEvent(listener.EventRoleCheck, sess.LoginID, device, deviceId, tokenValue, map[string]any{
		listener.ExtraKeyRoles:  roles,
		listener.ExtraKeyLogic:  listener.LogicAnd,
		listener.ExtraKeyResult: hasAll,
	})

	return hasAll
}

// HasRolesOr checks if a user has any specified role. HasRolesOr 检查用户是否拥有任一指定角色。
func (m *Manager) HasRolesOr(ctx context.Context, loginID string, roles []string) bool {
	if loginID == "" {
		return false
	}

	roleList, err := m.loadRolesByLoginID(ctx, loginID)
	if err != nil {
		m.logger.Errorf("manager.HasRolesOr: failed to load roles, loginID=%s, error=%v", loginID, err)
		return false
	}
	hasAny := hasAnyRole(roleList, roles)

	m.triggerEvent(listener.EventRoleCheck, loginID, "", "", "", map[string]any{
		listener.ExtraKeyRoles:  roles,
		listener.ExtraKeyLogic:  listener.LogicOr,
		listener.ExtraKeyResult: hasAny,
	})

	return hasAny
}

// HasRolesOrByToken checks if a token user has any specified role. HasRolesOrByToken 根据 Token 检查任一角色。
func (m *Manager) HasRolesOrByToken(ctx context.Context, tokenValue string, roles []string) bool {
	sess, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		m.logger.Errorf("manager.HasRolesOrByToken: failed to get token info, token=%s, error=%v", tokenValue, err)
		return false
	}

	device, deviceId := tokenInfo.Device, tokenInfo.DeviceId
	roleList := m.resolveRoles(ctx, sess.Roles, AccessSubject{
		LoginID:  sess.LoginID,
		Device:   device,
		DeviceID: deviceId,
		Token:    tokenValue,
	})
	hasAny := hasAnyRole(roleList, roles)

	m.triggerEvent(listener.EventRoleCheck, sess.LoginID, device, deviceId, tokenValue, map[string]any{
		listener.ExtraKeyRoles:  roles,
		listener.ExtraKeyLogic:  listener.LogicOr,
		listener.ExtraKeyResult: hasAny,
	})

	return hasAny
}

// CheckRole checks if a user has a specific role. CheckRole 校验用户是否拥有指定角色。
func (m *Manager) CheckRole(ctx context.Context, loginID string, role string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	roles, err := m.loadRolesByLoginID(ctx, loginID)
	if err != nil {
		return err
	}
	hasRole := role != "" && hasRoleInList(roles, role)

	m.triggerEvent(listener.EventRoleCheck, loginID, "", "", "", map[string]any{
		listener.ExtraKeyRole:   role,
		listener.ExtraKeyResult: hasRole,
	})

	if !hasRole {
		return fmt.Errorf("%w: %s", derror.ErrRoleDenied, role)
	}
	return nil
}

// CheckRoleByToken checks if a token user has a specific role. CheckRoleByToken 根据 Token 校验用户是否拥有指定角色。
func (m *Manager) CheckRoleByToken(ctx context.Context, tokenValue string, role string) error {
	sess, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		return err
	}
	device, deviceId := tokenInfo.Device, tokenInfo.DeviceId
	roles, err := m.loadRoles(ctx, sess.Roles, AccessSubject{
		LoginID:  sess.LoginID,
		Device:   device,
		DeviceID: deviceId,
		Token:    tokenValue,
	})
	if err != nil {
		return err
	}
	hasRole := role != "" && hasRoleInList(roles, role)

	m.triggerEvent(listener.EventRoleCheck, sess.LoginID, device, deviceId, tokenValue, map[string]any{
		listener.ExtraKeyRole:   role,
		listener.ExtraKeyResult: hasRole,
	})

	if !hasRole {
		return fmt.Errorf("%w: %s", derror.ErrRoleDenied, role)
	}
	return nil
}

// CheckRoleAnd checks if a user has all specified roles. CheckRoleAnd 校验用户是否拥有全部角色。
func (m *Manager) CheckRoleAnd(ctx context.Context, loginID string, roles []string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	roleList, err := m.loadRolesByLoginID(ctx, loginID)
	if err != nil {
		return err
	}
	hasAll := hasAllRoles(roleList, roles)

	m.triggerEvent(listener.EventRoleCheck, loginID, "", "", "", map[string]any{
		listener.ExtraKeyRoles:  roles,
		listener.ExtraKeyLogic:  listener.LogicAnd,
		listener.ExtraKeyResult: hasAll,
	})

	if !hasAll {
		return derror.ErrRoleDenied
	}
	return nil
}

// CheckRoleAndByToken checks if a token user has all specified roles. CheckRoleAndByToken 根据 Token 校验用户是否拥有全部角色。
func (m *Manager) CheckRoleAndByToken(ctx context.Context, tokenValue string, roles []string) error {
	sess, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		return err
	}
	device, deviceId := tokenInfo.Device, tokenInfo.DeviceId
	roleList, err := m.loadRoles(ctx, sess.Roles, AccessSubject{
		LoginID:  sess.LoginID,
		Device:   device,
		DeviceID: deviceId,
		Token:    tokenValue,
	})
	if err != nil {
		return err
	}
	hasAll := hasAllRoles(roleList, roles)

	m.triggerEvent(listener.EventRoleCheck, sess.LoginID, device, deviceId, tokenValue, map[string]any{
		listener.ExtraKeyRoles:  roles,
		listener.ExtraKeyLogic:  listener.LogicAnd,
		listener.ExtraKeyResult: hasAll,
	})

	if !hasAll {
		return derror.ErrRoleDenied
	}
	return nil
}

// CheckRoleOr checks if a user has any specified role. CheckRoleOr 校验用户是否拥有任一角色。
func (m *Manager) CheckRoleOr(ctx context.Context, loginID string, roles []string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	roleList, err := m.loadRolesByLoginID(ctx, loginID)
	if err != nil {
		return err
	}
	hasAny := hasAnyRole(roleList, roles)

	m.triggerEvent(listener.EventRoleCheck, loginID, "", "", "", map[string]any{
		listener.ExtraKeyRoles:  roles,
		listener.ExtraKeyLogic:  listener.LogicOr,
		listener.ExtraKeyResult: hasAny,
	})

	if !hasAny {
		return derror.ErrRoleDenied
	}
	return nil
}

// CheckRoleOrByToken checks if a token user has any specified role. CheckRoleOrByToken 根据 Token 校验用户是否拥有任一角色。
func (m *Manager) CheckRoleOrByToken(ctx context.Context, tokenValue string, roles []string) error {
	sess, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		return err
	}
	device, deviceId := tokenInfo.Device, tokenInfo.DeviceId
	roleList, err := m.loadRoles(ctx, sess.Roles, AccessSubject{
		LoginID:  sess.LoginID,
		Device:   device,
		DeviceID: deviceId,
		Token:    tokenValue,
	})
	if err != nil {
		return err
	}
	hasAny := hasAnyRole(roleList, roles)

	m.triggerEvent(listener.EventRoleCheck, sess.LoginID, device, deviceId, tokenValue, map[string]any{
		listener.ExtraKeyRoles:  roles,
		listener.ExtraKeyLogic:  listener.LogicOr,
		listener.ExtraKeyResult: hasAny,
	})

	if !hasAny {
		return derror.ErrRoleDenied
	}
	return nil
}

func hasRoleInList(roles []string, role string) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

func hasAllRoles(roles []string, required []string) bool {
	if len(required) == 0 {
		return false
	}
	for _, need := range required {
		if need == "" {
			return false
		}
		if !hasRoleInList(roles, need) {
			return false
		}
	}
	return true
}

func hasAnyRole(roles []string, required []string) bool {
	for _, need := range required {
		if need == "" {
			continue
		}
		if hasRoleInList(roles, need) {
			return true
		}
	}
	return false
}
