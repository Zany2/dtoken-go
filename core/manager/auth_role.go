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
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	// Lock account writes 锁定账号写操作。
	unlock := m.lockLoginWrite(loginID)
	// Release lock on function exit 函数退出时释放锁。
	defer func() { unlock() }()

	// Load session 加载会话。
	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		return err
	}

	// Add roles to session 向会话追加角色。
	sess.addRoles(roles...)
	// Persist updated session 持久化更新后的会话。
	if err = m.saveToStorage(ctx, m.getSessionKey(loginID), *sess); err != nil {
		return err
	}

	// Release lock before events 触发事件前释放锁。
	unlock()
	unlock = func() {}

	// Trigger role change event 触发角色变更事件。
	m.triggerEvent(listener.EventRoleChange, loginID, "", "", "", map[string]any{
		listener.ExtraKeyRoles:  roles,
		listener.ExtraKeyAction: listener.ActionAdd,
	})
	return nil
}

// AddRolesByToken adds roles to a user by token. AddRolesByToken 根据 Token 为用户添加角色。
func (m *Manager) AddRolesByToken(ctx context.Context, tokenValue string, roles []string) error {
	// Validate token and load context 校验 Token 并加载上下文。
	_, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		return err
	}

	// Lock account writes 锁定账号写操作。
	unlock := m.lockLoginWrite(tokenInfo.LoginID)
	// Release lock on function exit 函数退出时释放锁。
	defer func() { unlock() }()

	// Ensure token is still alive 确认 Token 仍然有效。
	if err = m.ensureTerminalTokenAlive(ctx, tokenValue); err != nil {
		return err
	}
	// Load session by token 根据 Token 加载会话。
	sess, err := m.GetSessionByToken(ctx, tokenValue)
	if err != nil {
		return err
	}

	// Add roles to session 向会话追加角色。
	sess.addRoles(roles...)
	// Persist updated session 持久化更新后的会话。
	if err = m.saveToStorage(ctx, m.getSessionKey(sess.LoginID), *sess); err != nil {
		return err
	}

	// Release lock before events 触发事件前释放锁。
	unlock()
	unlock = func() {}

	// Trigger role change event 触发角色变更事件。
	m.triggerEvent(listener.EventRoleChange, sess.LoginID, tokenInfo.Device, tokenInfo.DeviceId, tokenValue, map[string]any{
		listener.ExtraKeyRoles:  roles,
		listener.ExtraKeyAction: listener.ActionAdd,
	})
	return nil
}

// RemoveRoles removes roles from a user. RemoveRoles 删除用户的指定角色。
func (m *Manager) RemoveRoles(ctx context.Context, loginID string, roles []string) error {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	// Lock account writes 锁定账号写操作。
	unlock := m.lockLoginWrite(loginID)
	// Release lock on function exit 函数退出时释放锁。
	defer func() { unlock() }()

	// Load session 加载会话。
	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		return err
	}

	// Remove roles from session 从会话移除角色。
	sess.removeRoles(roles...)
	// Persist updated session 持久化更新后的会话。
	if err = m.saveToStorage(ctx, m.getSessionKey(loginID), *sess); err != nil {
		return err
	}

	// Release lock before events 触发事件前释放锁。
	unlock()
	unlock = func() {}

	// Trigger role change event 触发角色变更事件。
	m.triggerEvent(listener.EventRoleChange, loginID, "", "", "", map[string]any{
		listener.ExtraKeyRoles:  roles,
		listener.ExtraKeyAction: listener.ActionRemove,
	})
	return nil
}

// RemoveRolesByToken removes roles from a user by token. RemoveRolesByToken 根据 Token 删除用户的指定角色。
func (m *Manager) RemoveRolesByToken(ctx context.Context, tokenValue string, roles []string) error {
	// Validate token and load context 校验 Token 并加载上下文。
	_, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		return err
	}

	// Lock account writes 锁定账号写操作。
	unlock := m.lockLoginWrite(tokenInfo.LoginID)
	// Release lock on function exit 函数退出时释放锁。
	defer func() { unlock() }()

	// Ensure token is still alive 确认 Token 仍然有效。
	if err = m.ensureTerminalTokenAlive(ctx, tokenValue); err != nil {
		return err
	}
	// Load session by token 根据 Token 加载会话。
	sess, err := m.GetSessionByToken(ctx, tokenValue)
	if err != nil {
		return err
	}

	// Remove roles from session 从会话移除角色。
	sess.removeRoles(roles...)
	// Persist updated session 持久化更新后的会话。
	if err = m.saveToStorage(ctx, m.getSessionKey(sess.LoginID), *sess); err != nil {
		return err
	}

	// Release lock before events 触发事件前释放锁。
	unlock()
	unlock = func() {}

	// Trigger role change event 触发角色变更事件。
	m.triggerEvent(listener.EventRoleChange, sess.LoginID, tokenInfo.Device, tokenInfo.DeviceId, tokenValue, map[string]any{
		listener.ExtraKeyRoles:  roles,
		listener.ExtraKeyAction: listener.ActionRemove,
	})
	return nil
}

// GetRoles retrieves the role list for a user. GetRoles 获取用户的角色列表。
func (m *Manager) GetRoles(ctx context.Context, loginID string) ([]string, error) {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return nil, derror.ErrIDIsEmpty
	}
	// Load roles by login ID 按登录 ID 加载角色。
	return m.loadRolesByLoginID(ctx, loginID)
}

// GetRolesByToken retrieves the role list by token. GetRolesByToken 根据 Token 获取角色列表。
func (m *Manager) GetRolesByToken(ctx context.Context, tokenValue string) ([]string, error) {
	// Validate token and load context 校验 Token 并加载上下文。
	sess, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		return nil, err
	}

	// Resolve roles by token 按 Token 解析角色。
	return m.loadRoles(ctx, sess.Roles, AccessSubject{
		LoginID:  sess.LoginID,
		Device:   tokenInfo.Device,
		DeviceID: tokenInfo.DeviceId,
		Token:    tokenValue,
	})
}

// HasRole checks if a user has a specific role. HasRole 检查用户是否拥有指定角色。
func (m *Manager) HasRole(ctx context.Context, loginID string, role string) bool {
	// Validate required parameters 校验必要参数。
	if loginID == "" || role == "" {
		return false
	}

	// Load roles 加载角色。
	roles, err := m.loadRolesByLoginID(ctx, loginID)
	if err != nil {
		m.logger.Errorf("manager.HasRole: failed to load roles, loginID=%s, error=%v", loginID, err)
		return false
	}
	// Calculate role result 计算角色结果。
	hasRole := hasRoleInList(roles, role)

	// Trigger role check event 触发角色校验事件。
	m.triggerEvent(listener.EventRoleCheck, loginID, "", "", "", map[string]any{
		listener.ExtraKeyRole:   role,
		listener.ExtraKeyResult: hasRole,
	})

	return hasRole
}

// HasRoleByToken checks if a user has a specific role by token. HasRoleByToken 根据 Token 检查用户是否拥有指定角色。
func (m *Manager) HasRoleByToken(ctx context.Context, tokenValue string, role string) bool {
	// Validate role 校验角色。
	if role == "" {
		return false
	}

	// Validate token and load context 校验 Token 并加载上下文。
	sess, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		m.logger.Errorf("manager.HasRoleByToken: failed to get token info, token=%s, error=%v", tokenValue, err)
		return false
	}

	// Build access subject 构建访问主体。
	device, deviceId := tokenInfo.Device, tokenInfo.DeviceId
	// Resolve roles by token 按 Token 解析角色。
	roles := m.resolveRoles(ctx, sess.Roles, AccessSubject{
		LoginID:  sess.LoginID,
		Device:   device,
		DeviceID: deviceId,
		Token:    tokenValue,
	})
	// Calculate role result 计算角色结果。
	hasRole := hasRoleInList(roles, role)

	// Trigger role check event 触发角色校验事件。
	m.triggerEvent(listener.EventRoleCheck, sess.LoginID, device, deviceId, tokenValue, map[string]any{
		listener.ExtraKeyRole:   role,
		listener.ExtraKeyResult: hasRole,
	})

	return hasRole
}

// HasRolesAnd checks if a user has all specified roles. HasRolesAnd 检查用户是否拥有全部指定角色。
func (m *Manager) HasRolesAnd(ctx context.Context, loginID string, roles []string) bool {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return false
	}

	// Load roles 加载角色。
	roleList, err := m.loadRolesByLoginID(ctx, loginID)
	if err != nil {
		m.logger.Errorf("manager.HasRolesAnd: failed to load roles, loginID=%s, error=%v", loginID, err)
		return false
	}
	// Calculate AND result 计算 AND 结果。
	hasAll := hasAllRoles(roleList, roles)

	// Trigger role check event 触发角色校验事件。
	m.triggerEvent(listener.EventRoleCheck, loginID, "", "", "", map[string]any{
		listener.ExtraKeyRoles:  roles,
		listener.ExtraKeyLogic:  listener.LogicAnd,
		listener.ExtraKeyResult: hasAll,
	})

	return hasAll
}

// HasRolesAndByToken checks if a token user has all specified roles. HasRolesAndByToken 根据 Token 检查全部角色。
func (m *Manager) HasRolesAndByToken(ctx context.Context, tokenValue string, roles []string) bool {
	// Validate token and load context 校验 Token 并加载上下文。
	sess, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		m.logger.Errorf("manager.HasRolesAndByToken: failed to get token info, token=%s, error=%v", tokenValue, err)
		return false
	}

	// Build access subject 构建访问主体。
	device, deviceId := tokenInfo.Device, tokenInfo.DeviceId
	// Resolve roles by token 按 Token 解析角色。
	roleList := m.resolveRoles(ctx, sess.Roles, AccessSubject{
		LoginID:  sess.LoginID,
		Device:   device,
		DeviceID: deviceId,
		Token:    tokenValue,
	})
	// Calculate AND result 计算 AND 结果。
	hasAll := hasAllRoles(roleList, roles)

	// Trigger role check event 触发角色校验事件。
	m.triggerEvent(listener.EventRoleCheck, sess.LoginID, device, deviceId, tokenValue, map[string]any{
		listener.ExtraKeyRoles:  roles,
		listener.ExtraKeyLogic:  listener.LogicAnd,
		listener.ExtraKeyResult: hasAll,
	})

	return hasAll
}

// HasRolesOr checks if a user has any specified role. HasRolesOr 检查用户是否拥有任一指定角色。
func (m *Manager) HasRolesOr(ctx context.Context, loginID string, roles []string) bool {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return false
	}

	// Load roles 加载角色。
	roleList, err := m.loadRolesByLoginID(ctx, loginID)
	if err != nil {
		m.logger.Errorf("manager.HasRolesOr: failed to load roles, loginID=%s, error=%v", loginID, err)
		return false
	}
	// Calculate OR result 计算 OR 结果。
	hasAny := hasAnyRole(roleList, roles)

	// Trigger role check event 触发角色校验事件。
	m.triggerEvent(listener.EventRoleCheck, loginID, "", "", "", map[string]any{
		listener.ExtraKeyRoles:  roles,
		listener.ExtraKeyLogic:  listener.LogicOr,
		listener.ExtraKeyResult: hasAny,
	})

	return hasAny
}

// HasRolesOrByToken checks if a token user has any specified role. HasRolesOrByToken 根据 Token 检查任一角色。
func (m *Manager) HasRolesOrByToken(ctx context.Context, tokenValue string, roles []string) bool {
	// Validate token and load context 校验 Token 并加载上下文。
	sess, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		m.logger.Errorf("manager.HasRolesOrByToken: failed to get token info, token=%s, error=%v", tokenValue, err)
		return false
	}

	// Build access subject 构建访问主体。
	device, deviceId := tokenInfo.Device, tokenInfo.DeviceId
	// Resolve roles by token 按 Token 解析角色。
	roleList := m.resolveRoles(ctx, sess.Roles, AccessSubject{
		LoginID:  sess.LoginID,
		Device:   device,
		DeviceID: deviceId,
		Token:    tokenValue,
	})
	// Calculate OR result 计算 OR 结果。
	hasAny := hasAnyRole(roleList, roles)

	// Trigger role check event 触发角色校验事件。
	m.triggerEvent(listener.EventRoleCheck, sess.LoginID, device, deviceId, tokenValue, map[string]any{
		listener.ExtraKeyRoles:  roles,
		listener.ExtraKeyLogic:  listener.LogicOr,
		listener.ExtraKeyResult: hasAny,
	})

	return hasAny
}

// CheckRole checks if a user has a specific role. CheckRole 校验用户是否拥有指定角色。
func (m *Manager) CheckRole(ctx context.Context, loginID string, role string) error {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	// Load roles 加载角色。
	roles, err := m.loadRolesByLoginID(ctx, loginID)
	if err != nil {
		return err
	}
	// Calculate role result 计算角色结果。
	hasRole := role != "" && hasRoleInList(roles, role)

	// Trigger role check event 触发角色校验事件。
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
	// Validate token and load context 校验 Token 并加载上下文。
	sess, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		return err
	}
	// Build access subject 构建访问主体。
	device, deviceId := tokenInfo.Device, tokenInfo.DeviceId
	// Load roles by token 按 Token 加载角色。
	roles, err := m.loadRoles(ctx, sess.Roles, AccessSubject{
		LoginID:  sess.LoginID,
		Device:   device,
		DeviceID: deviceId,
		Token:    tokenValue,
	})
	if err != nil {
		return err
	}
	// Calculate role result 计算角色结果。
	hasRole := role != "" && hasRoleInList(roles, role)

	// Trigger role check event 触发角色校验事件。
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
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	// Load roles 加载角色。
	roleList, err := m.loadRolesByLoginID(ctx, loginID)
	if err != nil {
		return err
	}
	// Calculate AND result 计算 AND 结果。
	hasAll := hasAllRoles(roleList, roles)

	// Trigger role check event 触发角色校验事件。
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
	// Validate token and load context 校验 Token 并加载上下文。
	sess, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		return err
	}
	// Build access subject 构建访问主体。
	device, deviceId := tokenInfo.Device, tokenInfo.DeviceId
	// Load roles by token 按 Token 加载角色。
	roleList, err := m.loadRoles(ctx, sess.Roles, AccessSubject{
		LoginID:  sess.LoginID,
		Device:   device,
		DeviceID: deviceId,
		Token:    tokenValue,
	})
	if err != nil {
		return err
	}
	// Calculate AND result 计算 AND 结果。
	hasAll := hasAllRoles(roleList, roles)

	// Trigger role check event 触发角色校验事件。
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
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	// Load roles 加载角色。
	roleList, err := m.loadRolesByLoginID(ctx, loginID)
	if err != nil {
		return err
	}
	// Calculate OR result 计算 OR 结果。
	hasAny := hasAnyRole(roleList, roles)

	// Trigger role check event 触发角色校验事件。
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
	// Validate token and load context 校验 Token 并加载上下文。
	sess, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		return err
	}
	// Build access subject 构建访问主体。
	device, deviceId := tokenInfo.Device, tokenInfo.DeviceId
	// Load roles by token 按 Token 加载角色。
	roleList, err := m.loadRoles(ctx, sess.Roles, AccessSubject{
		LoginID:  sess.LoginID,
		Device:   device,
		DeviceID: deviceId,
		Token:    tokenValue,
	})
	if err != nil {
		return err
	}
	// Calculate OR result 计算 OR 结果。
	hasAny := hasAnyRole(roleList, roles)

	// Trigger role check event 触发角色校验事件。
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

// hasRoleInList checks whether role exists. hasRoleInList 检查角色是否存在。
func hasRoleInList(roles []string, role string) bool {
	// Check each role 逐个检查角色。
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

// hasAllRoles checks whether all roles exist. hasAllRoles 检查是否拥有全部角色。
func hasAllRoles(roles []string, required []string) bool {
	// Reject empty requirement 空需求直接拒绝。
	if len(required) == 0 {
		return false
	}
	// Check each required role 逐个检查必需角色。
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

// hasAnyRole checks whether any role exists. hasAnyRole 检查是否拥有任一角色。
func hasAnyRole(roles []string, required []string) bool {
	// Check each candidate 逐个检查候选角色。
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
