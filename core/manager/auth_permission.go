// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"context"
	"fmt"

	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/listener"
)

// AddPermissions adds permissions to a user. AddPermissions 为用户添加权限。
func (m *Manager) AddPermissions(ctx context.Context, loginID string, permissions []string) error {
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

	// Add permissions to session 向会话追加权限。
	sess.addPermissions(permissions...)
	// Persist updated session 持久化更新后的会话。
	if err = m.saveToStorage(ctx, m.getSessionKey(loginID), *sess); err != nil {
		return err
	}

	// Release lock before events 触发事件前释放锁。
	unlock()
	unlock = func() {}

	// Trigger permission change event 触发权限变更事件。
	m.triggerEvent(listener.EventPermissionChange, loginID, "", "", "", map[string]any{
		listener.ExtraKeyPermissions: permissions,
		listener.ExtraKeyAction:      listener.ActionAdd,
	})
	return nil
}

// AddPermissionsByToken adds permissions to a user by token. AddPermissionsByToken 根据 Token 为用户添加权限。
func (m *Manager) AddPermissionsByToken(ctx context.Context, tokenValue string, permissions []string) error {
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

	// Add permissions to session 向会话追加权限。
	sess.addPermissions(permissions...)
	// Persist updated session 持久化更新后的会话。
	if err = m.saveToStorage(ctx, m.getSessionKey(sess.LoginID), *sess); err != nil {
		return err
	}

	// Release lock before events 触发事件前释放锁。
	unlock()
	unlock = func() {}

	// Trigger permission change event 触发权限变更事件。
	m.triggerEvent(listener.EventPermissionChange, sess.LoginID, tokenInfo.Device, tokenInfo.DeviceId, tokenValue, map[string]any{
		listener.ExtraKeyPermissions: permissions,
		listener.ExtraKeyAction:      listener.ActionAdd,
	})
	return nil
}

// RemovePermissions removes permissions from a user. RemovePermissions 删除用户的指定权限。
func (m *Manager) RemovePermissions(ctx context.Context, loginID string, permissions []string) error {
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

	// Remove permissions from session 从会话移除权限。
	sess.removePermissions(permissions...)
	// Persist updated session 持久化更新后的会话。
	if err = m.saveToStorage(ctx, m.getSessionKey(loginID), *sess); err != nil {
		return err
	}

	// Release lock before events 触发事件前释放锁。
	unlock()
	unlock = func() {}

	// Trigger permission change event 触发权限变更事件。
	m.triggerEvent(listener.EventPermissionChange, loginID, "", "", "", map[string]any{
		listener.ExtraKeyPermissions: permissions,
		listener.ExtraKeyAction:      listener.ActionRemove,
	})
	return nil
}

// RemovePermissionsByToken removes permissions from a user by token. RemovePermissionsByToken 根据 Token 删除用户的指定权限。
func (m *Manager) RemovePermissionsByToken(ctx context.Context, tokenValue string, permissions []string) error {
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

	// Remove permissions from session 从会话移除权限。
	sess.removePermissions(permissions...)
	// Persist updated session 持久化更新后的会话。
	if err = m.saveToStorage(ctx, m.getSessionKey(sess.LoginID), *sess); err != nil {
		return err
	}

	// Release lock before events 触发事件前释放锁。
	unlock()
	unlock = func() {}

	// Trigger permission change event 触发权限变更事件。
	m.triggerEvent(listener.EventPermissionChange, sess.LoginID, tokenInfo.Device, tokenInfo.DeviceId, tokenValue, map[string]any{
		listener.ExtraKeyPermissions: permissions,
		listener.ExtraKeyAction:      listener.ActionRemove,
	})
	return nil
}

// GetPermissions retrieves the permission list for a user. GetPermissions 获取用户的权限列表。
func (m *Manager) GetPermissions(ctx context.Context, loginID string) ([]string, error) {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return nil, derror.ErrIDIsEmpty
	}
	// Load permissions by login ID 按登录 ID 加载权限。
	return m.loadPermissionsByLoginID(ctx, loginID)
}

// GetPermissionsByToken retrieves the permission list by token. GetPermissionsByToken 根据 Token 获取权限列表。
func (m *Manager) GetPermissionsByToken(ctx context.Context, tokenValue string) ([]string, error) {
	// Validate token and load context 校验 Token 并加载上下文。
	sess, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		return nil, err
	}

	// Resolve permissions by token 按 Token 解析权限。
	return m.loadPermissions(ctx, sess.Permissions, AccessSubject{
		LoginID:  sess.LoginID,
		Device:   tokenInfo.Device,
		DeviceID: tokenInfo.DeviceId,
		Token:    tokenValue,
	})
}

// HasPermission checks if a user has a specific permission. HasPermission 检查用户是否拥有指定权限。
func (m *Manager) HasPermission(ctx context.Context, loginID string, permission string) bool {
	// Validate required parameters 校验必要参数。
	if loginID == "" || permission == "" {
		return false
	}

	// Load permissions 加载权限。
	permissions, err := m.loadPermissionsByLoginID(ctx, loginID)
	if err != nil {
		m.logger.Errorf("manager.HasPermission: failed to load permissions, loginID=%s, error=%v", loginID, err)
		return false
	}
	// Calculate permission result 计算权限结果。
	hasPermission := m.hasPermissionInList(permissions, permission)

	// Trigger permission check event 触发权限校验事件。
	m.triggerEvent(listener.EventPermissionCheck, loginID, "", "", "", map[string]any{
		listener.ExtraKeyPermission: permission,
		listener.ExtraKeyResult:     hasPermission,
	})

	return hasPermission
}

// HasPermissionByToken checks if a user has a specific permission by token. HasPermissionByToken 根据 Token 检查用户是否拥有指定权限。
func (m *Manager) HasPermissionByToken(ctx context.Context, tokenValue string, permission string) bool {
	// Validate permission 校验权限。
	if permission == "" {
		return false
	}

	// Validate token and load context 校验 Token 并加载上下文。
	sess, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		m.logger.Errorf("manager.HasPermissionByToken: failed to get token info, token=%s, error=%v", tokenValue, err)
		return false
	}

	// Build access subject 构建访问主体。
	device, deviceId := tokenInfo.Device, tokenInfo.DeviceId
	// Resolve permissions by token 按 Token 解析权限。
	permissions := m.resolvePermissions(ctx, sess.Permissions, AccessSubject{
		LoginID:  sess.LoginID,
		Device:   device,
		DeviceID: deviceId,
		Token:    tokenValue,
	})
	// Calculate permission result 计算权限结果。
	hasPermission := m.hasPermissionInList(permissions, permission)

	// Trigger permission check event 触发权限校验事件。
	m.triggerEvent(listener.EventPermissionCheck, sess.LoginID, device, deviceId, tokenValue, map[string]any{
		listener.ExtraKeyPermission: permission,
		listener.ExtraKeyResult:     hasPermission,
	})

	return hasPermission
}

// HasPermissionsAnd checks if a user has all specified permissions. HasPermissionsAnd 检查用户是否拥有全部指定权限。
func (m *Manager) HasPermissionsAnd(ctx context.Context, loginID string, permissions []string) bool {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return false
	}

	// Load permissions 加载权限。
	permList, err := m.loadPermissionsByLoginID(ctx, loginID)
	if err != nil {
		m.logger.Errorf("manager.HasPermissionsAnd: failed to load permissions, loginID=%s, error=%v", loginID, err)
		return false
	}
	// Calculate AND result 计算 AND 结果。
	hasAll := m.hasAllPermissions(permList, permissions)

	// Trigger permission check event 触发权限校验事件。
	m.triggerEvent(listener.EventPermissionCheck, loginID, "", "", "", map[string]any{
		listener.ExtraKeyPermissions: permissions,
		listener.ExtraKeyLogic:       listener.LogicAnd,
		listener.ExtraKeyResult:      hasAll,
	})

	return hasAll
}

// HasPermissionsAndByToken checks if a token user has all specified permissions. HasPermissionsAndByToken 根据 Token 检查全部权限。
func (m *Manager) HasPermissionsAndByToken(ctx context.Context, tokenValue string, permissions []string) bool {
	// Validate token and load context 校验 Token 并加载上下文。
	sess, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		m.logger.Errorf("manager.HasPermissionsAndByToken: failed to get token info, token=%s, error=%v", tokenValue, err)
		return false
	}

	// Build access subject 构建访问主体。
	device, deviceId := tokenInfo.Device, tokenInfo.DeviceId
	// Resolve permissions by token 按 Token 解析权限。
	permList := m.resolvePermissions(ctx, sess.Permissions, AccessSubject{
		LoginID:  sess.LoginID,
		Device:   device,
		DeviceID: deviceId,
		Token:    tokenValue,
	})
	// Calculate AND result 计算 AND 结果。
	hasAll := m.hasAllPermissions(permList, permissions)

	// Trigger permission check event 触发权限校验事件。
	m.triggerEvent(listener.EventPermissionCheck, sess.LoginID, device, deviceId, tokenValue, map[string]any{
		listener.ExtraKeyPermissions: permissions,
		listener.ExtraKeyLogic:       listener.LogicAnd,
		listener.ExtraKeyResult:      hasAll,
	})

	return hasAll
}

// HasPermissionsOr checks if a user has any specified permission. HasPermissionsOr 检查用户是否拥有任一指定权限。
func (m *Manager) HasPermissionsOr(ctx context.Context, loginID string, permissions []string) bool {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return false
	}

	// Load permissions 加载权限。
	permList, err := m.loadPermissionsByLoginID(ctx, loginID)
	if err != nil {
		m.logger.Errorf("manager.HasPermissionsOr: failed to load permissions, loginID=%s, error=%v", loginID, err)
		return false
	}
	// Calculate OR result 计算 OR 结果。
	hasAny := m.hasAnyPermission(permList, permissions)

	// Trigger permission check event 触发权限校验事件。
	m.triggerEvent(listener.EventPermissionCheck, loginID, "", "", "", map[string]any{
		listener.ExtraKeyPermissions: permissions,
		listener.ExtraKeyLogic:       listener.LogicOr,
		listener.ExtraKeyResult:      hasAny,
	})

	return hasAny
}

// HasPermissionsOrByToken checks if a token user has any specified permission. HasPermissionsOrByToken 根据 Token 检查任一权限。
func (m *Manager) HasPermissionsOrByToken(ctx context.Context, tokenValue string, permissions []string) bool {
	// Validate token and load context 校验 Token 并加载上下文。
	sess, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		m.logger.Errorf("manager.HasPermissionsOrByToken: failed to get token info, token=%s, error=%v", tokenValue, err)
		return false
	}

	// Build access subject 构建访问主体。
	device, deviceId := tokenInfo.Device, tokenInfo.DeviceId
	// Resolve permissions by token 按 Token 解析权限。
	permList := m.resolvePermissions(ctx, sess.Permissions, AccessSubject{
		LoginID:  sess.LoginID,
		Device:   device,
		DeviceID: deviceId,
		Token:    tokenValue,
	})
	// Calculate OR result 计算 OR 结果。
	hasAny := m.hasAnyPermission(permList, permissions)

	// Trigger permission check event 触发权限校验事件。
	m.triggerEvent(listener.EventPermissionCheck, sess.LoginID, device, deviceId, tokenValue, map[string]any{
		listener.ExtraKeyPermissions: permissions,
		listener.ExtraKeyLogic:       listener.LogicOr,
		listener.ExtraKeyResult:      hasAny,
	})

	return hasAny
}

// CheckPermission checks if a user has a specific permission. CheckPermission 校验用户是否拥有指定权限。
func (m *Manager) CheckPermission(ctx context.Context, loginID string, permission string) error {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	// Load permissions 加载权限。
	permissions, err := m.loadPermissionsByLoginID(ctx, loginID)
	if err != nil {
		return err
	}
	// Calculate permission result 计算权限结果。
	hasPermission := permission != "" && m.hasPermissionInList(permissions, permission)

	// Trigger permission check event 触发权限校验事件。
	m.triggerEvent(listener.EventPermissionCheck, loginID, "", "", "", map[string]any{
		listener.ExtraKeyPermission: permission,
		listener.ExtraKeyResult:     hasPermission,
	})

	if !hasPermission {
		return fmt.Errorf("%w: %s", derror.ErrPermissionDenied, permission)
	}
	return nil
}

// CheckPermissionByToken checks if a token user has a specific permission. CheckPermissionByToken 根据 Token 校验用户是否拥有指定权限。
func (m *Manager) CheckPermissionByToken(ctx context.Context, tokenValue string, permission string) error {
	// Validate token and load context 校验 Token 并加载上下文。
	sess, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		return err
	}
	// Build access subject 构建访问主体。
	device, deviceId := tokenInfo.Device, tokenInfo.DeviceId
	// Load permissions by token 按 Token 加载权限。
	permissions, err := m.loadPermissions(ctx, sess.Permissions, AccessSubject{
		LoginID:  sess.LoginID,
		Device:   device,
		DeviceID: deviceId,
		Token:    tokenValue,
	})
	if err != nil {
		return err
	}
	// Calculate permission result 计算权限结果。
	hasPermission := permission != "" && m.hasPermissionInList(permissions, permission)

	// Trigger permission check event 触发权限校验事件。
	m.triggerEvent(listener.EventPermissionCheck, sess.LoginID, device, deviceId, tokenValue, map[string]any{
		listener.ExtraKeyPermission: permission,
		listener.ExtraKeyResult:     hasPermission,
	})

	if !hasPermission {
		return fmt.Errorf("%w: %s", derror.ErrPermissionDenied, permission)
	}
	return nil
}

// CheckPermissionAnd checks if a user has all specified permissions. CheckPermissionAnd 校验用户是否拥有全部权限。
func (m *Manager) CheckPermissionAnd(ctx context.Context, loginID string, permissions []string) error {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	// Load permissions 加载权限。
	permList, err := m.loadPermissionsByLoginID(ctx, loginID)
	if err != nil {
		return err
	}
	// Calculate AND result 计算 AND 结果。
	hasAll := m.hasAllPermissions(permList, permissions)

	// Trigger permission check event 触发权限校验事件。
	m.triggerEvent(listener.EventPermissionCheck, loginID, "", "", "", map[string]any{
		listener.ExtraKeyPermissions: permissions,
		listener.ExtraKeyLogic:       listener.LogicAnd,
		listener.ExtraKeyResult:      hasAll,
	})

	if !hasAll {
		return derror.ErrPermissionDenied
	}
	return nil
}

// CheckPermissionAndByToken checks if a token user has all specified permissions. CheckPermissionAndByToken 根据 Token 校验用户是否拥有全部权限。
func (m *Manager) CheckPermissionAndByToken(ctx context.Context, tokenValue string, permissions []string) error {
	// Validate token and load context 校验 Token 并加载上下文。
	sess, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		return err
	}
	// Build access subject 构建访问主体。
	device, deviceId := tokenInfo.Device, tokenInfo.DeviceId
	// Load permissions by token 按 Token 加载权限。
	permList, err := m.loadPermissions(ctx, sess.Permissions, AccessSubject{
		LoginID:  sess.LoginID,
		Device:   device,
		DeviceID: deviceId,
		Token:    tokenValue,
	})
	if err != nil {
		return err
	}
	// Calculate AND result 计算 AND 结果。
	hasAll := m.hasAllPermissions(permList, permissions)

	// Trigger permission check event 触发权限校验事件。
	m.triggerEvent(listener.EventPermissionCheck, sess.LoginID, device, deviceId, tokenValue, map[string]any{
		listener.ExtraKeyPermissions: permissions,
		listener.ExtraKeyLogic:       listener.LogicAnd,
		listener.ExtraKeyResult:      hasAll,
	})

	if !hasAll {
		return derror.ErrPermissionDenied
	}
	return nil
}

// CheckPermissionOr checks if a user has any specified permission. CheckPermissionOr 校验用户是否拥有任一权限。
func (m *Manager) CheckPermissionOr(ctx context.Context, loginID string, permissions []string) error {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	// Load permissions 加载权限。
	permList, err := m.loadPermissionsByLoginID(ctx, loginID)
	if err != nil {
		return err
	}
	// Calculate OR result 计算 OR 结果。
	hasAny := m.hasAnyPermission(permList, permissions)

	// Trigger permission check event 触发权限校验事件。
	m.triggerEvent(listener.EventPermissionCheck, loginID, "", "", "", map[string]any{
		listener.ExtraKeyPermissions: permissions,
		listener.ExtraKeyLogic:       listener.LogicOr,
		listener.ExtraKeyResult:      hasAny,
	})

	if !hasAny {
		return derror.ErrPermissionDenied
	}
	return nil
}

// CheckPermissionOrByToken checks if a token user has any specified permission. CheckPermissionOrByToken 根据 Token 校验用户是否拥有任一权限。
func (m *Manager) CheckPermissionOrByToken(ctx context.Context, tokenValue string, permissions []string) error {
	// Validate token and load context 校验 Token 并加载上下文。
	sess, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		return err
	}
	// Build access subject 构建访问主体。
	device, deviceId := tokenInfo.Device, tokenInfo.DeviceId
	// Load permissions by token 按 Token 加载权限。
	permList, err := m.loadPermissions(ctx, sess.Permissions, AccessSubject{
		LoginID:  sess.LoginID,
		Device:   device,
		DeviceID: deviceId,
		Token:    tokenValue,
	})
	if err != nil {
		return err
	}
	// Calculate OR result 计算 OR 结果。
	hasAny := m.hasAnyPermission(permList, permissions)

	// Trigger permission check event 触发权限校验事件。
	m.triggerEvent(listener.EventPermissionCheck, sess.LoginID, device, deviceId, tokenValue, map[string]any{
		listener.ExtraKeyPermissions: permissions,
		listener.ExtraKeyLogic:       listener.LogicOr,
		listener.ExtraKeyResult:      hasAny,
	})

	if !hasAny {
		return derror.ErrPermissionDenied
	}
	return nil
}

// matchPermission matches permission with wildcard support. matchPermission 支持通配符权限匹配。
func (m *Manager) matchPermission(pattern, permission string) bool {
	return m.strategy.normalize().PermissionMatcher(pattern, permission)
}

// hasPermissionInList checks if permission exists in permission list. hasPermissionInList 判断权限是否存在。
func (m *Manager) hasPermissionInList(perms []string, permission string) bool {
	// Check each permission 逐个检查权限。
	for _, p := range perms {
		if m.matchPermission(p, permission) {
			return true
		}
	}
	return false
}

// hasAllPermissions checks whether all permissions exist. hasAllPermissions 检查是否拥有全部权限。
func (m *Manager) hasAllPermissions(perms []string, required []string) bool {
	// Reject empty requirement 空需求直接拒绝。
	if len(required) == 0 {
		return false
	}
	// Check each required permission 逐个检查必需权限。
	for _, need := range required {
		if need == "" {
			return false
		}
		if !m.hasPermissionInList(perms, need) {
			return false
		}
	}
	return true
}

// hasAnyPermission checks whether any permission exists. hasAnyPermission 检查是否拥有任一权限。
func (m *Manager) hasAnyPermission(perms []string, required []string) bool {
	// Check each candidate 逐个检查候选权限。
	for _, need := range required {
		if need == "" {
			continue
		}
		if m.hasPermissionInList(perms, need) {
			return true
		}
	}
	return false
}
