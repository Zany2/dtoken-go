// @Author daixk 2026/1/22 17:33:00
package manager

import (
	"context"
	"fmt"
	djson "github.com/Zany2/dtoken-go/component/codec/json"
	"github.com/Zany2/dtoken-go/component/generator/dgenerator"
	"github.com/Zany2/dtoken-go/component/log/nop"
	"github.com/Zany2/dtoken-go/component/pool/ants"
	"github.com/Zany2/dtoken-go/component/storage/memory"
	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/errors"
	"github.com/Zany2/dtoken-go/core/utils"
	"strings"
	"time"
)

// NewManager 创建 Manager 实例
func NewManager(
	cfg *config.Config,
	generator adapter.Generator,
	storage adapter.Storage,
	serializer adapter.Codec,
	logger adapter.Log,
	pool adapter.Pool,
	customPermissionListFunc, CustomRoleListFunc func(loginID, authType string) ([]string, error),
) *Manager {

	// cfg 为 nil 时使用默认配置
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	// generator 为 nil 时创建默认 Token 生成器
	if generator == nil {
		generator = dgenerator.NewGenerator(cfg.Timeout, cfg.JwtSecretKey, cfg.TokenStyle)
	}

	// storage 为 nil 时使用内存存储
	if storage == nil {
		storage = memory.NewStorage()
	}

	// serializer 为 nil 时使用 JSON 序列化器
	if serializer == nil {
		serializer = djson.NewJSONSerializer()
	}

	// logger 为 nil 时使用空日志记录器
	if logger == nil {
		logger = nop.NewNopLogger()
	}

	// 启用自动续期且 pool 为 nil 时使用默认协程池
	if cfg.AutoRenew && pool == nil {
		pool = ants.NewRenewPoolManagerWithDefaultConfig()
	}

	// 返回初始化完成的 Manager 实例
	return &Manager{
		config:                   cfg,
		generator:                generator,
		storage:                  storage,
		serializer:               serializer,
		logger:                   logger,
		pool:                     pool,
		CustomPermissionListFunc: customPermissionListFunc,
		CustomRoleListFunc:       CustomRoleListFunc,
	}
}

// CloseManager 关闭管理器并释放所有资源
func (m *Manager) CloseManager() {
	// 若日志记录器实现了 LogControl 接口，则执行 Flush 和 Close
	if logControl, ok := m.logger.(adapter.LogControl); ok {
		logControl.Flush()
		logControl.Close()
	}

	// 安全关闭协程池并置空
	if m.pool != nil {
		m.pool.Stop()
		m.pool = nil
	}
}

// ---------------------------- 登录认证 ----------------------------

// Login 登录
func (m *Manager) Login(ctx context.Context, loginID string, deviceAndDeviceId ...string) (string, error) {
	// 检查ID
	if loginID == "" {
		return "", errors.ErrIDIsEmpty
	}

	// 检查账号是否被封禁
	isDisable := m.IsDisable(ctx, loginID)
	if isDisable {
		return "", errors.ErrAccountDisabled
	}

	// 获取设备类型和设备ID
	device, deviceId := m.getDeviceAndDeviceId(deviceAndDeviceId...)

	// 加载Session
	sess, err := loadFromStorage[Session](ctx, m, m.getSessionKey(loginID))
	if err == nil && sess != nil {
		// 不允许并发登录 根据作用域将所有Token设置为踢下线
		if !m.config.IsConcurrent { // m.config.IsConcurrent == false
			if m.config.ConcurrencyScope == config.ConcurrencyScopeAccount {
				_ = m.removeTerminalInfosAndTokens(ctx, sess)
			} else {
				_ = m.removeTerminalInfosAndTokens(ctx, sess, device)
			}
		} else if m.config.IsShare { // m.config.IsConcurrent == true && m.config.IsShare == true
			if m.config.ConcurrencyScope == config.ConcurrencyScopeAccount {
				if token := m.getTokenAndShare(ctx, sess); token != "" {
					return token, nil
				}
			} else {
				if token := m.getTokenAndShare(ctx, sess, device); token != "" {
					return token, nil
				}
			}
		} else if m.config.MaxLoginCount > 0 { // m.config.IsConcurrent == true && m.config.IsShare == true && m.config.MaxLoginCount > 0
			if int64(len(sess.TerminalInfos)) >= m.config.MaxLoginCount {
				if m.config.ConcurrencyScope == config.ConcurrencyScopeAccount {
					m.removeOldestTerminalInfosAndTokens(ctx, sess)
				} else {
					m.removeOldestTerminalInfosAndTokens(ctx, sess, device)
				}
			}
		}
	}

	// 生成Token
	token, err := m.generator.Generate(loginID, device, deviceId)
	if err != nil {
		return "", err
	}

	// 获取session并添加登录设备
	sess, err = m.getSession(ctx, loginID)
	if err != nil {
		return "", err
	}
	sess.TerminalInfos = append(sess.TerminalInfos, TerminalInfo{
		Token:    token,
		LoginID:  loginID,
		Device:   device,
		DeviceId: deviceId,
	})

	// 保存session
	err = saveToStorage[Session](ctx, m, m.getSessionKey(loginID), *sess, m.getExpiration())
	if err != nil {
		return "", err
	}
	// 保存Token
	err = saveToStorage[TokenInfo](ctx, m, m.getTokenKey(token), TokenInfo{
		AuthType:   m.config.AuthType,
		LoginID:    loginID,
		Device:     device,
		DeviceId:   deviceId,
		CreateTime: time.Now().Unix(),
	}, m.getExpiration())
	if err != nil {
		return "", err
	}

	return token, nil
}

// LoginByToken 根据Token登录
func (m *Manager) LoginByToken(ctx context.Context, tokenValue string) error {
	// 检查Token
	if tokenValue == "" {
		return errors.ErrInvalidToken
	}

	// 从存储中加载TokenInfo
	tokenInfo, err := loadFromStorage[TokenInfo](ctx, m, m.getTokenKey(tokenValue))
	if err != nil || tokenInfo == nil {
		return errors.ErrInvalidToken
	}
	// 检查Token状态
	err = m.checkTokenStatus(ctx, tokenInfo)
	if err != nil {
		return err
	}

	// 检查账号是否被封禁
	if m.IsDisable(ctx, tokenInfo.LoginID) {
		return errors.ErrAccountDisabled
	}

	// 续期Session
	_ = m.storage.Expire(ctx, m.getSessionKey(tokenInfo.LoginID), m.getExpiration())
	// 续期Token
	_ = m.storage.Expire(ctx, m.getTokenKey(tokenValue), m.getExpiration())

	return nil
}

// Logout 登出
func (m *Manager) Logout(ctx context.Context, tokenValue string) error {
	// 检查Token
	if tokenValue == "" {
		return errors.ErrInvalidToken
	}

	// 获取tokenInfo
	tokenInfo, err := loadFromStorage[TokenInfo](ctx, m, m.getTokenKey(tokenValue))
	if err != nil || tokenInfo == nil {
		return errors.ErrInvalidToken
	}

	// 从session中删除TerminalInfo
	sess, err := loadFromStorage[Session](ctx, m, m.getSessionKey(tokenInfo.LoginID))
	if err == nil && sess != nil {
		_, ok := sess.removeTerminalByToken(tokenValue)
		if ok {
			err = saveToStorage[Session](ctx, m, m.getSessionKey(tokenInfo.LoginID), *sess, m.getExpiration())
			if err != nil {
				return err
			}
		}
	}

	// 删除Token
	err = m.storage.Delete(ctx, m.getTokenKey(tokenValue))
	if err != nil {
		return fmt.Errorf("%w: %v", errors.ErrStorageUnavailable, err)
	}

	return nil
}

// LogoutByDeviceAndDeviceId 根据设备类型和设备ID登出
func (m *Manager) LogoutByDeviceAndDeviceId(ctx context.Context, loginID string, deviceAndDeviceId ...string) error {
	// 检查ID
	if loginID == "" {
		return errors.ErrIDIsEmpty
	}

	// 获取设备类型和设备ID
	device, deviceId := m.getDeviceAndDeviceId(deviceAndDeviceId...)

	// 加载 Session
	sess, err := loadFromStorage[Session](ctx, m, m.getSessionKey(loginID))
	if err != nil || sess == nil {
		return nil
	}

	// Session移除匹配的终端
	removedList := sess.removeTerminalByDeviceAndDeviceId(device, deviceId)

	if len(removedList) > 0 {
		// 保存session
		_ = saveToStorage[Session](ctx, m, m.getSessionKey(loginID), *sess, m.getExpiration())
		tokenList := make([]string, len(removedList))
		for _, info := range removedList {
			tokenList = append(tokenList, m.getTokenKey(info.Token))
		}
		// 删除对应的tokens
		_ = m.storage.Delete(ctx, tokenList...)
	}

	return nil
}

// ---------------------------- 在线状态管理 ----------------------------

// Kickout 踢人下线
func (m *Manager) Kickout(ctx context.Context, tokenValue string) error {
	// 获取tokenInfo
	tokenInfo, err := loadFromStorage[TokenInfo](ctx, m, m.getTokenKey(tokenValue))
	if err != nil {
		return err
	}

	// 加载Session
	sess, err := loadFromStorage[Session](ctx, m, m.getSessionKey(tokenInfo.LoginID))
	if err != nil {
		return err
	}
	if sess != nil {
		if _, ok := sess.removeTerminalByToken(tokenValue); ok {
			// 保存session
			err := saveToStorage(ctx, m, m.getSessionKey(tokenInfo.LoginID), *sess, m.getExpiration())
			if err != nil {
				return err
			}
		}
	}

	// 设置token状态为踢出
	err = m.storage.Set(ctx, m.getTokenKey(tokenValue), TokenStateKickOut, m.getExpiration())
	if err != nil {
		return fmt.Errorf("%w: %v", errors.ErrStorageUnavailable, err)
	}

	return nil
}

// KickoutByDeviceAndDeviceId 根据设备类型和设备ID踢人下线
func (m *Manager) KickoutByDeviceAndDeviceId(ctx context.Context, loginID string, deviceAndDeviceId ...string) error {
	return nil
}

// Replace 顶人下线
func (m *Manager) Replace(ctx context.Context, tokenValue string) error {
	// 获取tokenInfo
	tokenInfo, err := loadFromStorage[TokenInfo](ctx, m, m.getTokenKey(tokenValue))
	if err != nil {
		return err
	}

	// 加载Session
	sess, err := loadFromStorage[Session](ctx, m, m.getSessionKey(tokenInfo.LoginID))
	if err != nil {
		return err
	}
	if sess != nil {
		if _, ok := sess.removeTerminalByToken(tokenValue); ok {
			// 保存session
			err := saveToStorage(ctx, m, m.getSessionKey(tokenInfo.LoginID), *sess, m.getExpiration())
			if err != nil {
				return err
			}
		}
	}

	// 设置token状态为顶出
	err = m.storage.Set(ctx, m.getTokenKey(tokenValue), TokenStateReplaced, m.getExpiration())
	if err != nil {
		return fmt.Errorf("%w: %v", errors.ErrStorageUnavailable, err)
	}

	return nil
}

// ReplaceByDeviceAndDeviceId 根据设备类型和设备ID顶人下线
func (m *Manager) ReplaceByDeviceAndDeviceId(ctx context.Context, loginID string, deviceAndDeviceId ...string) error {
	return nil
}

// ---------------------------- Token 验证 ----------------------------

// IsLogin 检查用户是否登录
func (m *Manager) IsLogin(ctx context.Context, tokenValue string) bool {
	return m.storage.Exists(ctx, m.getTokenKey(tokenValue))
}

// CheckLogin 检查用户是否登录 根据情况返回err
func (m *Manager) CheckLogin(ctx context.Context, tokenValue string) error {
	tokenInfo, err := loadFromStorage[TokenInfo](ctx, m, m.getTokenKey(tokenValue))
	if err != nil {
		return err
	}
	return m.checkTokenStatus(ctx, tokenInfo)
}

// ---------------------------- Token 信息与解析 ----------------------------

// GetLoginID 根据Token获取登录ID
func (m *Manager) GetLoginID(ctx context.Context, tokenValue string) (string, error) {
	return "", nil
}

// GetLoginIDByDeviceAndDeviceId 根据设备类型和设备ID获取登录ID
func (m *Manager) GetLoginIDByDeviceAndDeviceId(ctx context.Context, deviceAndDeviceId ...string) (string, error) {
	return "", nil
}

// GetTokenInfo 根据Token获取TokenInfo信息
func (m *Manager) GetTokenInfo(ctx context.Context, tokenValue string) (*TokenInfo, error) {
	return nil, nil
}

// GetTokenInfoByDeviceAndDeviceId 根据设备类型和设备ID获取Token信息
func (m *Manager) GetTokenInfoByDeviceAndDeviceId(ctx context.Context, deviceAndDeviceId ...string) (*TokenInfo, error) {
	return nil, nil
}

// ---------------------------- 账号封禁 ----------------------------

// Disable 封禁账号
func (m *Manager) Disable(ctx context.Context, loginID string, duration time.Duration, reason ...string) error {
	return nil
}

// Untie 解封账号
func (m *Manager) Untie(ctx context.Context, loginID string) error {
	return nil
}

// IsDisable 检查账号是否被封禁
func (m *Manager) IsDisable(ctx context.Context, loginID string) bool {
	return false
}

// GetDisableInfo 获取封禁信息
func (m *Manager) GetDisableInfo(ctx context.Context, loginID string) (*DisableInfo, error) {
	return nil, nil
}

// GetDisableTTL 获取账号剩余封禁时间
func (m *Manager) GetDisableTTL(ctx context.Context, loginID string) (int64, error) {
	return 0, nil
}

// ---------------------------- Session 管理 ----------------------------

// GetSession gets session by login ID | 获取Session
func (m *Manager) GetSession(ctx context.Context, loginID string) (*Session, error) {
	return nil, nil
}

// GetSessionByToken Gets session by token value | 通过Token值获取Session
func (m *Manager) GetSessionByToken(ctx context.Context, tokenValue string) (*Session, error) {
	return nil, nil
}

// ---------------------------- Token 与会话信息查询 ----------------------------

// GetTokenValueListByLoginID Gets all tokens for specified login ID | 获取指定登录ID的所有Token
func (m *Manager) GetTokenValueListByLoginID(ctx context.Context, loginID string, checkAlive ...bool) ([]string, error) {
	return nil, nil
}

// GetTokenValueListByDeviceAndDeviceId Gets all tokens for specified device and device ID | 获取指定设备类型和设备ID的所有Token
func (m *Manager) GetTokenValueListByDeviceAndDeviceId(ctx context.Context, loginID string, device, deviceId string, checkAlive ...bool) ([]string, error) {
	return nil, nil
}

// ---------------------------- 权限验证 ----------------------------

// SetPermissions Sets permissions for user | 设置权限
func (m *Manager) SetPermissions(ctx context.Context, loginID string, permissions []string) error {
	sess, err := m.GetSession(ctx, loginID)
	if err != nil {
		return err
	}

	permissionsFromSession, ok := sess.Get(SessionKeyPermissions)
	if ok {
		permissions = append(permissions, utils.ToStringSlice(permissionsFromSession)...)
		permissions = utils.UniqueStrings(permissions)
	}

	return sess.Set(ctx, SessionKeyPermissions, permissions, m.getExpiration())
}

// SetPermissionsByToken Sets permissions by token | 根据Token设置权限
func (m *Manager) SetPermissionsByToken(ctx context.Context, tokenValue string, permissions []string) error {
	loginID, err := m.GetLoginIDNotCheck(ctx, tokenValue)
	if err != nil {
		return err
	}

	return m.SetPermissions(ctx, loginID, permissions)
}

// RemovePermissions removes specified permissions for user | 删除用户指定权限
func (m *Manager) RemovePermissions(ctx context.Context, loginID string, permissions []string) error {
	sess, err := m.GetSession(ctx, loginID)
	if err != nil {
		return err
	}

	permissionsFromSession, ok := sess.Get(SessionKeyPermissions)
	if !ok {
		return nil
	}

	existingPerms := utils.ToStringSlice(permissionsFromSession)
	if len(existingPerms) == 0 {
		return nil
	}

	// Build a set for fast lookup of permissions to remove | 构建待删除权限集合
	removeSet := make(map[string]struct{}, len(permissions))
	for _, p := range permissions {
		removeSet[p] = struct{}{}
	}

	// Filter out permissions to be removed | 过滤掉需要删除的权限
	newPerms := make([]string, 0, len(existingPerms))
	for _, p := range existingPerms {
		if _, shouldRemove := removeSet[p]; !shouldRemove {
			newPerms = append(newPerms, p)
		}
	}

	return sess.Set(ctx, SessionKeyPermissions, newPerms, m.getExpiration())
}

// RemovePermissionsByToken removes specified permissions by token | 根据Token删除指定权限
func (m *Manager) RemovePermissionsByToken(ctx context.Context, tokenValue string, permissions []string) error {
	loginID, err := m.GetLoginIDNotCheck(ctx, tokenValue)
	if err != nil {
		return err
	}

	return m.RemovePermissions(ctx, loginID, permissions)
}

// GetPermissions Gets permission list | 获取权限列表
func (m *Manager) GetPermissions(ctx context.Context, loginID string) ([]string, error) {
	if m.CustomPermissionListFunc != nil {
		perms, err := m.CustomPermissionListFunc(loginID, m.config.AuthType)
		if err != nil {
			return nil, err
		}
		return perms, nil
	}

	sess, err := m.GetSession(ctx, loginID)
	if err != nil {
		return nil, err
	}

	perms, exists := sess.Get(SessionKeyPermissions)
	if !exists {
		return []string{}, nil
	}

	return utils.ToStringSlice(perms), nil
}

// GetPermissionsByToken Gets permission list by token | 根据Token获取权限列表
func (m *Manager) GetPermissionsByToken(ctx context.Context, tokenValue string) ([]string, error) {
	loginID, err := m.GetLoginIDNotCheck(ctx, tokenValue)
	if err != nil {
		return nil, err
	}

	return m.GetPermissions(ctx, loginID)
}

// HasPermission checks whether the specified loginID has the given permission | 检查指定账号是否拥有指定权限
func (m *Manager) HasPermission(ctx context.Context, loginID string, permission string) bool {
	perms, err := m.GetPermissions(ctx, loginID)
	if err != nil {
		return false
	}

	for _, p := range perms {
		if m.matchPermission(p, permission) {
			return true
		}
	}

	return false
}

// HasPermissionByToken checks whether the current token subject has the specified permission | 根据当前 Token 判断是否拥有指定权限
func (m *Manager) HasPermissionByToken(ctx context.Context, tokenValue string, permission string) bool {
	loginID, err := m.GetLoginIDNotCheck(ctx, tokenValue)
	if err != nil {
		return false
	}

	return m.HasPermission(ctx, loginID, permission)
}

// HasPermissionsAnd Checks whether the user has all permissions (AND) | 是否拥有所有权限（AND）
func (m *Manager) HasPermissionsAnd(ctx context.Context, loginID string, permissions []string) bool {
	userPerms, err := m.GetPermissions(ctx, loginID)
	if err != nil || len(userPerms) == 0 {
		return false
	}

	// Check every required permission | 校验每一个必需权限
	for _, need := range permissions {
		if !m.hasPermissionInList(userPerms, need) {
			return false
		}
	}

	return true
}

// HasPermissionsAndByToken checks whether the current token subject has all specified permissions (AND) | 根据当前 Token 判断是否拥有所有指定权限（AND）
func (m *Manager) HasPermissionsAndByToken(ctx context.Context, tokenValue string, permissions []string) bool {
	loginID, err := m.GetLoginIDNotCheck(ctx, tokenValue)
	if err != nil {
		return false
	}

	return m.HasPermissionsAnd(ctx, loginID, permissions)
}

// HasPermissionsOr Checks whether the user has any permission (OR) | 是否拥有任一权限（OR）
func (m *Manager) HasPermissionsOr(ctx context.Context, loginID string, permissions []string) bool {
	// Get all permissions once | 一次性获取用户权限
	userPerms, err := m.GetPermissions(ctx, loginID)
	if err != nil || len(userPerms) == 0 {
		return false
	}

	// Check if any permission matches | 任一权限匹配即通过
	for _, need := range permissions {
		if m.hasPermissionInList(userPerms, need) {
			return true
		}
	}
	return false
}

// HasPermissionsOrByToken checks whether the current token subject has any of the specified permissions (OR) | 根据当前 Token 判断是否拥有任一指定权限（OR）
func (m *Manager) HasPermissionsOrByToken(ctx context.Context, tokenValue string, permissions []string) bool {
	loginID, err := m.GetLoginIDNotCheck(ctx, tokenValue)
	if err != nil {
		return false
	}

	return m.HasPermissionsOr(ctx, loginID, permissions)
}

// matchPermission Matches permission with wildcards support | 权限匹配（支持通配符）
func (m *Manager) matchPermission(pattern, permission string) bool {
	// Exact match or wildcard | 精确匹配或通配符
	if pattern == PermissionWildcard || pattern == permission {
		return true
	}

	// Pattern like "user:*" matches "user:add", "user:delete", etc. | 支持通配符，例如 user:* 匹配 user:add, user:delete等
	wildcardSuffix := PermissionSeparator + PermissionWildcard
	if strings.HasSuffix(pattern, wildcardSuffix) {
		prefix := strings.TrimSuffix(pattern, PermissionWildcard)
		return strings.HasPrefix(permission, prefix)
	}

	// Pattern like "user:*:view" | 支持 user:*:view 这样的模式
	if strings.Contains(pattern, PermissionWildcard) {
		parts := strings.Split(pattern, PermissionSeparator)
		permParts := strings.Split(permission, PermissionSeparator)
		if len(parts) != len(permParts) {
			return false
		}
		for i, part := range parts {
			if part != PermissionWildcard && part != permParts[i] {
				return false
			}
		}
		return true
	}

	return false
}

// hasPermissionInList checks whether permission exists in permission list | 判断权限是否存在于权限列表中
func (m *Manager) hasPermissionInList(perms []string, permission string) bool {
	for _, p := range perms {
		if m.matchPermission(p, permission) {
			return true
		}
	}
	return false
}

// ---------------------------- 角色验证 ----------------------------

// SetRoles Sets roles for user | 设置角色
func (m *Manager) SetRoles(ctx context.Context, loginID string, roles []string) error {
	sess, err := m.GetSession(ctx, loginID)
	if err != nil {
		return err
	}

	rolesFromSession, ok := sess.Get(SessionKeyRoles)
	if ok {
		roles = append(roles, utils.ToStringSlice(rolesFromSession)...)
		roles = utils.UniqueStrings(roles)
	}

	return sess.Set(ctx, SessionKeyRoles, roles, m.getExpiration())
}

// SetRolesByToken Sets roles by token | 根据Token设置角色
func (m *Manager) SetRolesByToken(ctx context.Context, tokenValue string, roles []string) error {
	loginID, err := m.GetLoginIDNotCheck(ctx, tokenValue)
	if err != nil {
		return err
	}

	return m.SetRoles(ctx, loginID, roles)
}

// RemoveRoles removes specified roles for user | 删除用户指定角色
func (m *Manager) RemoveRoles(ctx context.Context, loginID string, roles []string) error {
	sess, err := m.GetSession(ctx, loginID)
	if err != nil {
		return err
	}

	// Load existing roles | 加载已有角色
	rolesFromSession, ok := sess.Get(SessionKeyRoles)
	if !ok {
		return nil // No roles to remove | 没有角色可删除
	}

	existingRoles := utils.ToStringSlice(rolesFromSession)
	if len(existingRoles) == 0 {
		return nil
	}

	// Build lookup set for roles to remove | 构建待删除角色集合
	removeSet := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		removeSet[r] = struct{}{}
	}

	// Filter existing roles | 过滤掉需要删除的角色
	newRoles := make([]string, 0, len(existingRoles))
	for _, r := range existingRoles {
		if _, remove := removeSet[r]; !remove {
			newRoles = append(newRoles, r)
		}
	}

	// Save updated roles | 保存更新后的角色列表
	return sess.Set(ctx, SessionKeyRoles, newRoles, m.getExpiration())
}

// RemoveRolesByToken removes specified roles by token | 根据Token删除指定角色
func (m *Manager) RemoveRolesByToken(ctx context.Context, tokenValue string, roles []string) error {
	loginID, err := m.GetLoginIDNotCheck(ctx, tokenValue)
	if err != nil {
		return err
	}

	return m.RemoveRoles(ctx, loginID, roles)
}

// GetRoles gets role list for the specified loginID | 获取指定账号的角色列表
func (m *Manager) GetRoles(ctx context.Context, loginID string) ([]string, error) {
	if m.CustomRoleListFunc != nil {
		perms, err := m.CustomRoleListFunc(loginID, m.config.AuthType)
		if err != nil {
			return nil, err
		}
		return perms, nil
	}

	sess, err := m.GetSession(ctx, loginID)
	if err != nil {
		return nil, err
	}

	roles, exists := sess.Get(SessionKeyRoles)
	if !exists {
		return []string{}, nil
	}

	return utils.ToStringSlice(roles), nil
}

// GetRolesByToken Gets role list by token | 根据Token获取角色列表
func (m *Manager) GetRolesByToken(ctx context.Context, tokenValue string) ([]string, error) {
	loginID, err := m.GetLoginIDNotCheck(ctx, tokenValue)
	if err != nil {
		return nil, err
	}

	return m.GetRoles(ctx, loginID)
}

// HasRole checks whether the specified loginID has the given role | 检查指定账号是否拥有指定角色
func (m *Manager) HasRole(ctx context.Context, loginID string, role string) bool {
	roles, err := m.GetRoles(ctx, loginID)
	if err != nil {
		return false
	}

	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasRoleByToken checks whether the current token subject has the specified role | 根据当前 Token 判断是否拥有指定角色
func (m *Manager) HasRoleByToken(ctx context.Context, tokenValue string, role string) bool {
	loginID, err := m.GetLoginIDNotCheck(ctx, tokenValue)
	if err != nil {
		return false
	}

	return m.HasRole(ctx, loginID, role)
}

// HasRolesAnd Checks whether the user has all roles (AND) | 是否拥有所有角色（AND）
func (m *Manager) HasRolesAnd(ctx context.Context, loginID string, roles []string) bool {
	userRoles, err := m.GetRoles(ctx, loginID)
	if err != nil || len(userRoles) == 0 {
		return false
	}

	for _, need := range roles {
		if !m.hasRoleInList(userRoles, need) {
			return false
		}
	}
	return true
}

// HasRolesAndByToken checks whether the current token subject has all specified roles (AND) | 根据当前 Token 判断是否拥有所有指定角色（AND）
func (m *Manager) HasRolesAndByToken(ctx context.Context, tokenValue string, roles []string) bool {
	loginID, err := m.GetLoginIDNotCheck(ctx, tokenValue)
	if err != nil {
		return false
	}

	return m.HasRolesAnd(ctx, loginID, roles)
}

// HasRolesOr Checks whether the user has any role (OR) | 是否拥有任一角色（OR）
func (m *Manager) HasRolesOr(ctx context.Context, loginID string, roles []string) bool {
	userRoles, err := m.GetRoles(ctx, loginID)
	if err != nil || len(userRoles) == 0 {
		return false
	}

	for _, need := range roles {
		if m.hasRoleInList(userRoles, need) {
			return true
		}
	}
	return false
}

// HasRolesOrByToken checks whether the current token subject has any of the specified roles (OR) | 根据当前 Token 判断是否拥有任一指定角色（OR）
func (m *Manager) HasRolesOrByToken(ctx context.Context, tokenValue string, roles []string) bool {
	loginID, err := m.GetLoginIDNotCheck(ctx, tokenValue)
	if err != nil {
		return false
	}

	return m.HasRolesOr(ctx, loginID, roles)
}

// hasPermissionInList checks whether permission exists in permission list | 判断权限是否存在于权限列表中
func (m *Manager) hasRoleInList(roles []string, role string) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

// ---------------------------- Token 标签 ----------------------------

// ---------------------------- 安全特性 ----------------------------

// ---------------------------- OAuth2 特性 ----------------------------

// ---------------------------- 公共获取器 ----------------------------

// ---------------------------- 内部方法 ----------------------------

// ---------------------------- 内部辅助方法 ----------------------------

// getTokenKey 获取 Token 存储键
func (m *Manager) getTokenKey(tokenValue string) string {
	return m.config.KeyPrefix + m.config.AuthType + tokenValue
}

// getSessionKey 获取会话存储键
func (m *Manager) getSessionKey(loginID string) string {
	return m.config.KeyPrefix + m.config.AuthType + SessionKeyPrefix + loginID
}

// getRenewKey 获取 Token 续期追踪键
func (m *Manager) getRenewKey(tokenValue string) string {
	return m.config.KeyPrefix + m.config.AuthType + RenewKeyPrefix + tokenValue
}

// getDisableKey 获取账号禁用状态存储键
func (m *Manager) getDisableKey(loginID string) string {
	return m.config.KeyPrefix + m.config.AuthType + DisableKeyPrefix + loginID
}

// getExpiration 从配置中计算 Token 过期时长
func (m *Manager) getExpiration() time.Duration {
	if m.config.Timeout > 0 {
		return time.Duration(m.config.Timeout) * time.Second
	}
	return 0
}

// getDeviceAndDeviceId 获取设备类型和设备ID
func (m *Manager) getDeviceAndDeviceId(deviceAndDeviceId ...string) (string, string) {
	device := ""
	deviceId := ""

	if len(deviceAndDeviceId) > 0 {
		if val := strings.TrimSpace(deviceAndDeviceId[0]); val != "" {
			device = val
		}
	}
	if len(deviceAndDeviceId) > 1 {
		deviceId = strings.TrimSpace(deviceAndDeviceId[1])
	}

	return device, deviceId
}

// removeTerminalInfosAndTokens 移除终端信息和Token
func (m *Manager) removeTerminalInfosAndTokens(ctx context.Context, sess *Session, device ...string) error {
	if len(device) > 0 {
		// 移除指定设备类型的终端信息
		terminalInfos := sess.removeTerminalByDevice(device[0])

		// 保存会话数据
		err := saveToStorage[Session](ctx, m, m.getSessionKey(sess.LoginID), *sess, m.getExpiration())
		if err != nil {
			return err
		}

		// 将所有的Token设置为踢出
		for _, info := range terminalInfos {
			err = m.storage.Set(ctx, m.getTokenKey(info.Token), TokenStateKickOut, m.getExpiration())
			if err != nil {
				return fmt.Errorf("%w: %v", errors.ErrStorageUnavailable, err)
			}
		}

		return nil
	}

	// 获取旧的终端信息
	oldTerminalInfos := sess.TerminalInfos

	// 移除终端信息
	sess.TerminalInfos = make([]TerminalInfo, 0)

	// 保存会话数据
	err := saveToStorage[Session](ctx, m, m.getSessionKey(sess.LoginID), *sess, m.getExpiration())
	if err != nil {
		return err
	}
	// 将所有的Token设置为踢出
	for _, terminalInfo := range oldTerminalInfos {
		err = m.storage.Set(ctx, m.getTokenKey(terminalInfo.Token), TokenStateKickOut, m.getExpiration())
		if err != nil {
			return fmt.Errorf("%w: %v", errors.ErrStorageUnavailable, err)
		}
	}

	return nil
}

// getTokenAndShare 获取Token并共享
func (m *Manager) getTokenAndShare(ctx context.Context, sess *Session, device ...string) string {
	if len(sess.TerminalInfos) > 0 {
		if len(device) > 0 {
			terminalInfo, ok := sess.getLatestTerminalByDevice(device[0])
			if ok {
				// 续期session
				_ = m.storage.Expire(ctx, m.getSessionKey(terminalInfo.LoginID), m.getExpiration())
				// 存储Token 这里重新Set 防止Token为非正常登录状态
				_ = saveToStorage[TokenInfo](ctx, m, m.getTokenKey(terminalInfo.Token), TokenInfo{
					AuthType:   m.config.AuthType,
					LoginID:    terminalInfo.LoginID,
					Device:     terminalInfo.Device,
					DeviceId:   terminalInfo.DeviceId,
					CreateTime: time.Now().Unix(),
				}, m.getExpiration())

				return terminalInfo.Token
			}
		}

		// 如果存在设备列表信息 那么取最后一个
		terminalInfo := sess.TerminalInfos[len(sess.TerminalInfos)-1]

		// 续期session
		_ = m.storage.Expire(ctx, m.getSessionKey(terminalInfo.LoginID), m.getExpiration())
		// 存储Token 这里重新Set 防止Token为非正常登录状态
		_ = saveToStorage[TokenInfo](ctx, m, m.getTokenKey(terminalInfo.Token), TokenInfo{
			AuthType:   m.config.AuthType,
			LoginID:    terminalInfo.LoginID,
			Device:     terminalInfo.Device,
			DeviceId:   terminalInfo.DeviceId,
			CreateTime: time.Now().Unix(),
		}, m.getExpiration())

		return terminalInfo.Token
	}

	return ""
}

// removeOldestTerminalInfosAndTokens 移除最旧的终端信息并删除对应的Token
func (m *Manager) removeOldestTerminalInfosAndTokens(ctx context.Context, sess *Session, device ...string) {
	if len(device) > 0 {
		terminalInfo, ok := sess.removeOldestTerminal(device...)
		if ok {
			// 保存会话数据
			_ = saveToStorage[Session](ctx, m, m.getSessionKey(sess.LoginID), *sess, m.getExpiration())
			// 设置token状态为踢出
			_ = m.storage.Set(ctx, m.getTokenKey(terminalInfo.Token), TokenStateKickOut, m.getExpiration())
		}
	}

	terminalInfo, ok := sess.removeOldestTerminal()
	if ok {
		// 保存会话数据
		_ = saveToStorage[Session](ctx, m, m.getSessionKey(sess.LoginID), *sess, m.getExpiration())
		// 设置token状态为踢出
		_ = m.storage.Set(ctx, m.getTokenKey(terminalInfo.Token), TokenStateKickOut, m.getExpiration())
	}
}

// getSession 获取会话信息
func (m *Manager) getSession(ctx context.Context, loginID string) (*Session, error) {
	sess, err := loadFromStorage[Session](ctx, m, m.getSessionKey(loginID))
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return &Session{
			AuthType:      m.config.AuthType,
			LoginID:       loginID,
			CreateTime:    time.Now().Unix(),
			TerminalInfos: make([]TerminalInfo, 0),
			Permissions:   make([]string, 0),
			Roles:         make([]string, 0),
		}, nil
	}
	return sess, nil
}

// checkTokenStatus 检查Token状态
func (m *Manager) checkTokenStatus(_ context.Context, tokenInfo *TokenInfo) error {
	bytesData, err := utils.ToBytes(tokenInfo)
	if err != nil {
		return fmt.Errorf("%w: %v", errors.ErrSerializeFailed, err)
	}

	switch string(bytesData) {
	case string(TokenStateLogout):
		return errors.ErrInvalidToken
	case string(TokenStateKickOut):
		return errors.ErrTokenKickout
	case string(TokenStateReplaced):
		return errors.ErrTokenReplaced
	default:
		return nil
	}
}

// loadFromStorage 从存储中加载并反序列化为指定类型
func loadFromStorage[T any](
	ctx context.Context,
	manager *Manager,
	key string,
) (*T, error) {
	data, err := manager.storage.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errors.ErrStorageUnavailable, err)
	}

	if data == nil {
		return nil, nil // key 不存在
	}

	bytesData, err := utils.ToBytes(data)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errors.ErrSerializeFailed, err)
	}

	var result T
	if err = manager.serializer.Decode(bytesData, &result); err != nil {
		return nil, fmt.Errorf("%w: %v", errors.ErrSerializeFailed, err)
	}

	return &result, nil
}

// saveToStorage 将指定类型的数据序列化并存储到存储后端
func saveToStorage[T any](
	ctx context.Context,
	manager *Manager,
	key string,
	value T,
	expiration time.Duration,
) error {
	// 序列化为字节
	bytesData, err := manager.serializer.Encode(value)
	if err != nil {
		return fmt.Errorf("%w: %v", errors.ErrSerializeFailed, err)
	}

	// 存储到后端
	if err = manager.storage.Set(ctx, key, bytesData, expiration); err != nil {
		return fmt.Errorf("%w: %v", errors.ErrStorageUnavailable, err)
	}

	return nil
}
