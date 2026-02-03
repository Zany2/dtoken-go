// @Author daixk 2026/1/22 17:33:00
package manager

import (
	"context"
	"errors"
	"fmt"
	"github.com/Zany2/dtoken-go/core/listener"
	"github.com/Zany2/dtoken-go/core/nonce"
	"github.com/Zany2/dtoken-go/core/oauth2"
	"strings"
	"time"

	djson "github.com/Zany2/dtoken-go/com/codec/json"
	"github.com/Zany2/dtoken-go/com/generator/dgenerator"
	"github.com/Zany2/dtoken-go/com/log/nop"
	"github.com/Zany2/dtoken-go/com/pool/ants"
	"github.com/Zany2/dtoken-go/com/storage/memory"
	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/utils"
)

// ============================================================================
// PUBLIC METHODS - 公开方法
// ============================================================================

// ============================================================================
// Initialization & Lifecycle - 初始化与生命周期
// ============================================================================

// NewManager creates a new Manager instance with the provided components.
// NewManager 创建一个新的 Manager 实例,使用提供的组件。
func NewManager(
	cfg *config.Config,
	generator adapter.Generator,
	storage adapter.Storage,
	serializer adapter.Codec,
	logger adapter.Log,
	pool adapter.Pool,
	customPermissionListFunc, CustomRoleListFunc func(loginID, authType string) ([]string, error),
) *Manager {

	// Use default config if cfg is nil
	// cfg 为 nil 时使用默认配置
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	// Create default token generator if generator is nil
	// generator 为 nil 时创建默认 Token 生成器
	if generator == nil {
		generator = dgenerator.NewGenerator(cfg.Timeout, cfg.JwtSecretKey, cfg.TokenStyle)
	}

	// Use memory storage if storage is nil
	// storage 为 nil 时使用内存存储
	if storage == nil {
		storage = memory.NewStorage()
	}

	// Use JSON serializer if serializer is nil
	// serializer 为 nil 时使用 JSON 序列化器
	if serializer == nil {
		serializer = djson.NewJSONSerializer()
	}

	// Use nop logger if logger is nil
	// logger 为 nil 时使用空日志记录器
	if logger == nil {
		logger = nop.NewNopLogger()
	}

	// Create default goroutine pool if AutoRenew is enabled and pool is nil
	// 启用自动续期且 pool 为 nil 时使用默认协程池
	if cfg.AutoRenew && pool == nil {
		pool = ants.NewRenewPoolManagerWithDefaultConfig()
	}

	// Return initialized Manager instance
	// 返回初始化完成的 Manager 实例
	return &Manager{
		config:                   cfg,
		generator:                generator,
		storage:                  storage,
		serializer:               serializer,
		logger:                   logger,
		pool:                     pool,
		nonceManager:             nonce.NewNonceManager(cfg.AuthType, cfg.KeyPrefix, storage, nonce.DefaultNonceTTL),
		oauth2Manager:            oauth2.NewOAuth2Server(cfg.AuthType, cfg.KeyPrefix, storage, serializer),
		eventManager:             listener.NewManager(logger),
		CustomPermissionListFunc: customPermissionListFunc,
		CustomRoleListFunc:       CustomRoleListFunc,
	}
}

// CloseManager closes the manager and releases all resources.
// CloseManager 关闭管理器并释放所有资源。
func (m *Manager) CloseManager() {
	// Flush and close logger if it implements LogControl interface
	// 若日志记录器实现了 LogControl 接口则执行 Flush 和 Close
	if logControl, ok := m.logger.(adapter.LogControl); ok {
		logControl.Flush()
		logControl.Close()
	}

	// Safely stop goroutine pool and set to nil
	// 安全关闭协程池并置空
	if m.pool != nil {
		m.pool.Stop()
		m.pool = nil
	}
}

// ============================================================================
// Login & Authentication - 登录与认证
// ============================================================================

// Login performs user login and returns a token.
// Login 执行用户登录并返回 token。
func (m *Manager) Login(ctx context.Context, loginID string, deviceAndDeviceId ...string) (string, error) {
	if loginID == "" {
		return "", derror.ErrIDIsEmpty
	}

	if m.isDisable(ctx, loginID) {
		return "", derror.ErrAccountDisabled
	}

	device, deviceId := m.getDeviceAndDeviceId(deviceAndDeviceId...)

	// 尝试加载现有 session
	sess, err := m.loginGetSession(ctx, loginID)
	if err != nil {
		return "", err
	}

	// 处理并发策略
	if sess != nil {
		token, handled, handleErr := m.handleConcurrency(ctx, sess, loginID, device)
		if handleErr != nil {
			return "", handleErr
		}
		if handled {
			if token != "" {
				return token, nil // 复用 token
			}
		}
	}

	// 生成新 token
	token, err := m.generator.Generate(loginID, device, deviceId)
	if err != nil {
		return "", err
	}

	// 记录创建时间
	createTime := time.Now().Unix()

	// 获取或创建 session
	sess, err = m.getSession(ctx, loginID, true)
	if err != nil {
		return "", err
	}

	// 递增历史终端计数
	sess.HistoryTerminalCount++

	// 添加终端信息
	sess.TerminalInfos = append(sess.TerminalInfos, TerminalInfo{
		Token:      token,
		LoginID:    loginID,
		Device:     device,
		DeviceId:   deviceId,
		CreateTime: createTime,
		Index:      sess.HistoryTerminalCount, // 设置历史登录顺序索引
	})

	// 保存 session
	if err = m.saveToStorage(ctx, m.getSessionKey(loginID), *sess); err != nil {
		return "", err
	}

	// 保存 token info
	if err = m.saveToStorage(ctx, m.getTokenKey(token), TokenInfo{
		AuthType:   m.config.AuthType,
		LoginID:    loginID,
		Device:     device,
		DeviceId:   deviceId,
		CreateTime: createTime,
	}); err != nil {
		return "", err
	}

	// 触发登录事件
	m.triggerEvent(listener.EventLogin, loginID, device, deviceId, token, nil)

	return token, nil
}

// LoginByToken performs login renewal based on an existing token.
// LoginByToken 根据 Token 续期登录。
func (m *Manager) LoginByToken(ctx context.Context, tokenValue string) error {
	if tokenValue == "" {
		return derror.ErrInvalidToken
	}

	tokenInfo, err := m.getTokenInfo(ctx, tokenValue)
	if err != nil {
		return err
	}

	// 检查账号是否被封禁
	if m.isDisable(ctx, tokenInfo.LoginID) {
		return derror.ErrAccountDisabled
	}

	sess, err := m.getSession(ctx, tokenInfo.LoginID)
	if err != nil {
		return err
	}

	// 验证 token 是否在 session 的 TerminalInfos 中
	found := false
	for _, ti := range sess.TerminalInfos {
		if ti.Token == tokenValue {
			found = true
			break
		}
	}
	if !found {
		return derror.ErrInvalidToken
	}

	sessionKey := m.getSessionKey(tokenInfo.LoginID)
	tokenKey := m.getTokenKey(tokenValue)

	if err = m.storage.Expire(ctx, sessionKey, m.getExpiration()); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	if err = m.storage.Expire(ctx, tokenKey, m.getExpiration()); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	// 更新 metadata
	if m.config.RenewInterval > 0 {
		_ = m.storage.Set(ctx, m.getRenewKey(tokenValue), time.Now().Unix(), time.Duration(m.config.RenewInterval)*time.Second)
	}
	if m.config.ActiveTimeout > 0 {
		_ = m.storage.Set(ctx, m.getActiveKey(tokenValue), time.Now().Unix(), m.getExpiration())
	}

	// 触发续期事件
	m.triggerEvent(listener.EventRenew, tokenInfo.LoginID, tokenInfo.Device, tokenInfo.DeviceId, tokenValue, nil)

	return nil
}

// Logout logs out a user by token.
// Logout 根据 Token 登出用户。
func (m *Manager) Logout(ctx context.Context, tokenValue string) error {
	if tokenValue == "" {
		return derror.ErrInvalidToken
	}

	tokenInfo, err := m.getTokenInfo(ctx, tokenValue)
	if err != nil {
		return err
	}

	return m.logoutTerminals(ctx, tokenInfo.LoginID, func(sess *Session) []TerminalInfo {
		if info, ok := sess.removeTerminalByToken(tokenValue); ok {
			return []TerminalInfo{info}
		}
		return nil
	})
}

// LogoutByDevice logs out all terminals of a specific device type.
// LogoutByDevice 根据设备类型登出所有该设备的终端。
func (m *Manager) LogoutByDevice(ctx context.Context, loginID string, device string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	return m.logoutTerminals(ctx, loginID, func(sess *Session) []TerminalInfo {
		return sess.removeTerminalByDevice(device)
	})
}

// LogoutByDeviceAndDeviceId logs out a user by device type and device ID.
// LogoutByDeviceAndDeviceId 根据设备类型和设备ID登出用户。
func (m *Manager) LogoutByDeviceAndDeviceId(ctx context.Context, loginID string, deviceAndDeviceId ...string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	device, deviceId := m.getDeviceAndDeviceId(deviceAndDeviceId...)
	return m.logoutTerminals(ctx, loginID, func(sess *Session) []TerminalInfo {
		return sess.removeTerminalByDeviceAndDeviceId(device, deviceId)
	})
}

// ============================================================================
// Online Status Management - 在线状态管理
// ============================================================================

// Kickout kicks out a user by token.
// Kickout 根据 Token 踢人下线。
func (m *Manager) Kickout(ctx context.Context, tokenValue string) error {
	tokenInfo, err := m.getTokenInfo(ctx, tokenValue)
	if err != nil {
		return err
	}

	return m.processTerminals(ctx, tokenInfo.LoginID, func(sess *Session) []TerminalInfo {
		if info, ok := sess.removeTerminalByToken(tokenValue); ok {
			return []TerminalInfo{info}
		}
		return nil
	}, TokenStateKickOut)
}

// KickoutByDevice kicks out all terminals of a specific device type.
// KickoutByDevice 根据设备类型踢人下线（踢掉该设备类型的所有终端）。
func (m *Manager) KickoutByDevice(ctx context.Context, loginID string, device string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	return m.processTerminals(ctx, loginID, func(sess *Session) []TerminalInfo {
		return sess.removeTerminalByDevice(device)
	}, TokenStateKickOut)
}

// KickoutByDeviceAndDeviceId kicks out a user by device type and device ID.
// KickoutByDeviceAndDeviceId 根据设备类型和设备ID踢人下线。
func (m *Manager) KickoutByDeviceAndDeviceId(ctx context.Context, loginID string, deviceAndDeviceId ...string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	device, deviceId := m.getDeviceAndDeviceId(deviceAndDeviceId...)

	return m.processTerminals(ctx, loginID, func(sess *Session) []TerminalInfo {
		return sess.removeTerminalByDeviceAndDeviceId(device, deviceId)
	}, TokenStateKickOut)
}

// Replace replaces a user session by token.
// Replace 根据 Token 顶人下线。
func (m *Manager) Replace(ctx context.Context, tokenValue string) error {
	tokenInfo, err := m.getTokenInfo(ctx, tokenValue)
	if err != nil {
		return err
	}

	return m.processTerminals(ctx, tokenInfo.LoginID, func(sess *Session) []TerminalInfo {
		if info, ok := sess.removeTerminalByToken(tokenValue); ok {
			return []TerminalInfo{info}
		}
		return nil
	}, TokenStateReplaced)
}

// ReplaceByDevice replaces all terminals of a specific device type.
// ReplaceByDevice 根据设备类型顶人下线（顶掉该设备类型的所有终端）。
func (m *Manager) ReplaceByDevice(ctx context.Context, loginID string, device string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	return m.processTerminals(ctx, loginID, func(sess *Session) []TerminalInfo {
		return sess.removeTerminalByDevice(device)
	}, TokenStateReplaced)
}

// ReplaceByDeviceAndDeviceId replaces a user session by device type and device ID.
// ReplaceByDeviceAndDeviceId 根据设备类型和设备ID顶人下线。
func (m *Manager) ReplaceByDeviceAndDeviceId(ctx context.Context, loginID string, deviceAndDeviceId ...string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	device, deviceId := m.getDeviceAndDeviceId(deviceAndDeviceId...)
	return m.processTerminals(ctx, loginID, func(sess *Session) []TerminalInfo {
		return sess.removeTerminalByDeviceAndDeviceId(device, deviceId)
	}, TokenStateReplaced)
}

// ============================================================================
// Token Validation - Token 验证
// ============================================================================

// IsLogin checks if a user is logged in.
// IsLogin 检查用户是否登录。
func (m *Manager) IsLogin(ctx context.Context, tokenValue string) bool {
	return m.checkLoginInternal(ctx, tokenValue) == nil
}

// CheckLogin checks if a user is logged in and returns an error if not.
// CheckLogin 检查用户是否登录，如果未登录则返回错误。
func (m *Manager) CheckLogin(ctx context.Context, tokenValue string) error {
	return m.checkLoginInternal(ctx, tokenValue)
}

// ============================================================================
// Token Information - Token 信息
// ============================================================================

// GetLoginID retrieves the login ID from a token.
// GetLoginID 根据 Token 获取登录 ID。
func (m *Manager) GetLoginID(ctx context.Context, tokenValue string) (string, error) {
	// 获取tokenInfo
	tokenInfo, err := m.getTokenInfo(ctx, tokenValue)
	if err != nil {
		return "", err
	}

	return tokenInfo.LoginID, nil
}

// GetTokenInfo retrieves token information.
// GetTokenInfo 根据 Token 获取 TokenInfo 信息。
func (m *Manager) GetTokenInfo(ctx context.Context, tokenValue string) (*TokenInfo, error) {
	return m.getTokenInfo(ctx, tokenValue)
}

// GetDevice retrieves the device type for a token.
// GetDevice 获取 Token 的设备类型。
func (m *Manager) GetDevice(ctx context.Context, tokenValue string) (string, error) {
	tokenInfo, err := m.getTokenInfo(ctx, tokenValue)
	if err != nil {
		return "", err
	}
	return tokenInfo.Device, nil
}

// GetDeviceId retrieves the device ID for a token.
// GetDeviceId 获取 Token 的设备 ID。
func (m *Manager) GetDeviceId(ctx context.Context, tokenValue string) (string, error) {
	tokenInfo, err := m.getTokenInfo(ctx, tokenValue)
	if err != nil {
		return "", err
	}
	return tokenInfo.DeviceId, nil
}

// GetTokenCreateTime retrieves the creation time for a token.
// GetTokenCreateTime 获取 Token 的创建时间戳。
func (m *Manager) GetTokenCreateTime(ctx context.Context, tokenValue string) (int64, error) {
	tokenInfo, err := m.getTokenInfo(ctx, tokenValue)
	if err != nil {
		return 0, err
	}
	return tokenInfo.CreateTime, nil
}

// GetTokenTTL retrieves the remaining time-to-live for a token in seconds.
// GetTokenTTL 获取 Token 的剩余有效时间（秒）。
func (m *Manager) GetTokenTTL(ctx context.Context, tokenValue string) (int64, error) {
	ttl, err := m.storage.TTL(ctx, m.getTokenKey(tokenValue))
	if err != nil {
		return 0, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	// 标准 Redis TTL 语义：
	// -2: key 不存在
	// -1: key 存在但无过期时间
	// >0: 剩余秒数
	return int64(ttl.Seconds()), nil
}

// ============================================================================
// Account Disable Management - 账号封禁管理
// ============================================================================

// Disable disables an account for a specified duration.
// Disable 封禁账号指定时长。
func (m *Manager) Disable(ctx context.Context, loginID string, duration time.Duration, reason ...string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	// 1. 先尝试加载 Session（如果存储出错，在保存封禁信息前就返回，保证原子性）
	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		// 如果只是 session 不存在，不算错误；其他存储错误则返回
		if !errors.Is(err, derror.ErrSessionNotFound) {
			return err
		}
		// 否则 sess == nil，继续执行封禁操作（幂等）
	}

	// 2. 构建并保存封禁信息
	disableInfo := DisableInfo{
		DisableTime: time.Now().Unix(),
	}
	if len(reason) > 0 && reason[0] != "" {
		disableInfo.DisableReason = reason[0]
	}

	if err = m.saveToStorage(ctx, m.getDisableKey(loginID), disableInfo, duration); err != nil {
		return err
	}

	// 3. 删除 Session
	if err = m.storage.Delete(ctx, m.getSessionKey(loginID)); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	// 4. 如果有终端信息，清理所有相关 token 数据
	if sess != nil && len(sess.TerminalInfos) > 0 {
		tokens := make([]string, len(sess.TerminalInfos))
		tokenKeys := make([]string, len(sess.TerminalInfos))
		for i, info := range sess.TerminalInfos {
			tokens[i] = info.Token
			tokenKeys[i] = m.getTokenKey(info.Token)
		}

		// 删除主 token keys
		if err = m.storage.Delete(ctx, tokenKeys...); err != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}

		// 清理附属 metadata（续期、活跃时间）
		if err = m.cleanTokenMetadata(ctx, tokens); err != nil {
			return err
		}
	}

	// 触发封禁事件
	m.triggerEvent(listener.EventDisable, loginID, "", "", "", map[string]any{
		"reason":   disableInfo.DisableReason,
		"duration": duration.Seconds(),
	})

	return nil
}

// Untie removes the disable status from an account.
// Untie 解封账号。
func (m *Manager) Untie(ctx context.Context, loginID string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	if err := m.storage.Delete(ctx, m.getDisableKey(loginID)); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	// 触发解禁事件
	m.triggerEvent(listener.EventUntie, loginID, "", "", "", nil)

	return nil
}

// IsDisable checks if an account is disabled.
// IsDisable 检查账号是否被封禁。
func (m *Manager) IsDisable(ctx context.Context, loginID string) bool {
	return m.isDisable(ctx, loginID)
}

// GetDisableInfo retrieves disable information for an account.
// GetDisableInfo 获取账号的封禁信息。
func (m *Manager) GetDisableInfo(ctx context.Context, loginID string) (*DisableInfo, error) {
	disableInfoData, err := m.storage.Get(ctx, m.getDisableKey(loginID))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	// 如果 key 不存在（用户未被封禁），返回明确的错误
	if disableInfoData == nil {
		return nil, derror.ErrAccountNotDisabled
	}

	bytesData, err := utils.ToBytes(disableInfoData)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrTypeConvert, err)
	}

	var disableInfo DisableInfo
	if err = m.serializer.Decode(bytesData, &disableInfo); err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}

	return &disableInfo, nil
}

// GetDisableTTL retrieves the remaining disable time for an account in seconds.
// GetDisableTTL 获取账号剩余封禁时间（秒）。
func (m *Manager) GetDisableTTL(ctx context.Context, loginID string) (int64, error) {
	ttl, err := m.storage.TTL(ctx, m.getDisableKey(loginID))
	if err != nil {
		return 0, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	// 标准 Redis TTL 语义：
	// -2: key 不存在（未封禁或已解封）
	// -1: key 存在但无过期时间（理论上不应出现）
	// >0: 剩余秒数
	switch {
	case ttl == -2*time.Second:
		return -2, nil // 未封禁
	case ttl == -1*time.Second:
		return -1, nil // 永久封禁（无 TTL）
	case ttl > 0:
		return int64(ttl.Seconds()), nil
	default:
		return 0, nil
	}
}

// ============================================================================
// Session Management - 会话管理
// ============================================================================

// GetSession retrieves session information for a login ID.
// GetSession 获取指定登录 ID 的会话信息。
func (m *Manager) GetSession(ctx context.Context, loginID string) (*Session, error) {
	if loginID == "" {
		return nil, derror.ErrIDIsEmpty
	}
	return m.getSession(ctx, loginID)
}

// GetSessionByToken retrieves session information by token.
// GetSessionByToken 通过 Token 值获取会话信息。
func (m *Manager) GetSessionByToken(ctx context.Context, tokenValue string) (*Session, error) {
	// 获取tokenInfo
	tokenInfo, err := m.getTokenInfo(ctx, tokenValue)
	if err != nil {
		return nil, err
	}

	return m.getSession(ctx, tokenInfo.LoginID)
}

// GetTokenValueListByLoginID retrieves all tokens for a login ID.
// GetTokenValueListByLoginID 获取指定登录 ID 的所有 Token。
func (m *Manager) GetTokenValueListByLoginID(ctx context.Context, loginID string, checkAlive ...bool) ([]string, error) {
	if loginID == "" {
		return nil, derror.ErrIDIsEmpty
	}

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		// 仅当存储层真正出错时才返回 error；session 不存在视为 nil
		if !errors.Is(err, derror.ErrSessionNotFound) {
			return nil, err
		}
		return []string{}, nil
	}
	if sess == nil {
		return []string{}, nil
	}

	return m.filterTokens(ctx, sess.TerminalInfos, len(checkAlive) > 0 && checkAlive[0])
}

// GetTokenValueListByDevice retrieves all tokens for a specific device type.
// GetTokenValueListByDevice 获取指定设备类型的所有 Token。
func (m *Manager) GetTokenValueListByDevice(ctx context.Context, loginID, device string, checkAlive ...bool) ([]string, error) {
	if loginID == "" {
		return []string{}, derror.ErrIDIsEmpty
	}
	if device == "" {
		return []string{}, derror.ErrInvalidToken
	}

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		if !errors.Is(err, derror.ErrSessionNotFound) {
			return nil, err
		}
		return []string{}, nil
	}
	if sess == nil {
		return []string{}, nil
	}

	matched := sess.getTerminalsByDevice(device)
	return m.filterTokens(ctx, matched, len(checkAlive) > 0 && checkAlive[0])
}

// GetTokenValueListByDeviceAndDeviceId retrieves all tokens for a specific device type and device ID.
// GetTokenValueListByDeviceAndDeviceId 获取指定设备类型和设备 ID 的所有 Token。
func (m *Manager) GetTokenValueListByDeviceAndDeviceId(ctx context.Context, loginID, device, deviceId string, checkAlive ...bool) ([]string, error) {
	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		if !errors.Is(err, derror.ErrSessionNotFound) {
			return nil, err
		}
		return []string{}, nil
	}
	if sess == nil {
		return []string{}, nil
	}

	matched := sess.getTerminalsByDeviceAndDeviceId(device, deviceId)
	return m.filterTokens(ctx, matched, len(checkAlive) > 0 && checkAlive[0])
}

// GetOnlineTerminalCount retrieves the count of online terminals for a user.
// GetOnlineTerminalCount 获取用户的在线终端数量。
func (m *Manager) GetOnlineTerminalCount(ctx context.Context, loginID string) (int, error) {
	if loginID == "" {
		return 0, derror.ErrIDIsEmpty
	}

	tokens, err := m.GetTokenValueListByLoginID(ctx, loginID, true)
	if err != nil {
		return 0, err
	}
	return len(tokens), nil
}

// GetOnlineTerminalCountByDevice retrieves the count of online terminals for a specific device type.
// GetOnlineTerminalCountByDevice 获取用户在指定设备类型的在线终端数量。
func (m *Manager) GetOnlineTerminalCountByDevice(ctx context.Context, loginID, device string) (int, error) {
	tokens, err := m.GetTokenValueListByDevice(ctx, loginID, device, true)
	if err != nil {
		return 0, err
	}
	return len(tokens), nil
}

// GetOnlineTerminalCountByDeviceAndDeviceId retrieves the count of online terminals for a specific device type and device ID.
// GetOnlineTerminalCountByDeviceAndDeviceId 获取用户在指定设备类型和设备ID的在线终端数量。
func (m *Manager) GetOnlineTerminalCountByDeviceAndDeviceId(ctx context.Context, loginID, device, deviceId string) (int, error) {
	tokens, err := m.GetTokenValueListByDeviceAndDeviceId(ctx, loginID, device, deviceId, true)
	if err != nil {
		return 0, err
	}
	return len(tokens), nil
}

// ============================================================================
// Permission Management - 权限管理
// ============================================================================

// AddPermissions adds permissions to a user.
// AddPermissions 为用户添加权限。
func (m *Manager) AddPermissions(ctx context.Context, loginID string, permissions []string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		return err
	}

	sess.addPermissions(permissions...)
	err = m.saveToStorage(ctx, m.getSessionKey(loginID), *sess)
	if err != nil {
		return err
	}

	return nil
}

// AddPermissionsByToken adds permissions to a user by token.
// AddPermissionsByToken 根据 Token 为用户添加权限。
func (m *Manager) AddPermissionsByToken(ctx context.Context, tokenValue string, permissions []string) error {
	sess, err := m.GetSessionByToken(ctx, tokenValue)
	if err != nil {
		return err
	}

	// 添加权限
	sess.addPermissions(permissions...)
	// 保存Session
	err = m.saveToStorage(ctx, m.getSessionKey(sess.LoginID), *sess)
	if err != nil {
		return err
	}

	return nil
}

// RemovePermissions removes permissions from a user.
// RemovePermissions 删除用户的指定权限。
func (m *Manager) RemovePermissions(ctx context.Context, loginID string, permissions []string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		return err
	}

	sess.removePermissions(permissions...)
	err = m.saveToStorage(ctx, m.getSessionKey(loginID), *sess)
	if err != nil {
		return err
	}

	return nil
}

// RemovePermissionsByToken removes permissions from a user by token.
// RemovePermissionsByToken 根据 Token 删除用户的指定权限。
func (m *Manager) RemovePermissionsByToken(ctx context.Context, tokenValue string, permissions []string) error {
	sess, err := m.GetSessionByToken(ctx, tokenValue)
	if err != nil {
		return err
	}

	// 删除权限
	sess.removePermissions(permissions...)
	// 保存Session
	err = m.saveToStorage(ctx, m.getSessionKey(sess.LoginID), *sess)
	if err != nil {
		return err
	}

	return nil
}

// GetPermissions retrieves the permission list for a user.
// GetPermissions 获取用户的权限列表。
func (m *Manager) GetPermissions(ctx context.Context, loginID string) ([]string, error) {
	if loginID == "" {
		return nil, derror.ErrIDIsEmpty
	}

	// 自定义权限列表获取函数
	if m.CustomPermissionListFunc != nil {
		return m.CustomPermissionListFunc(loginID, m.config.AuthType)
	}

	// 获取Session
	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		return nil, err
	}

	return sess.Permissions, nil
}

// GetPermissionsByToken retrieves the permission list by token.
// GetPermissionsByToken 根据 Token 获取权限列表。
func (m *Manager) GetPermissionsByToken(ctx context.Context, tokenValue string) ([]string, error) {
	// 获取Session
	sess, err := m.GetSessionByToken(ctx, tokenValue)
	if err != nil {
		return nil, err
	}

	return sess.Permissions, nil
}

// HasPermission checks if a user has a specific permission.
// HasPermission 检查用户是否拥有指定权限。
func (m *Manager) HasPermission(ctx context.Context, loginID string, permission string) bool {
	if loginID == "" {
		return false
	}

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		return false
	}

	for _, p := range sess.Permissions {
		if m.matchPermission(p, permission) {
			return true
		}
	}

	return false
}

// HasPermissionByToken checks if a user has a specific permission by token.
// HasPermissionByToken 根据 Token 检查用户是否拥有指定权限。
func (m *Manager) HasPermissionByToken(ctx context.Context, tokenValue string, permission string) bool {
	sess, err := m.GetSessionByToken(ctx, tokenValue)
	if err != nil {
		return false
	}

	for _, p := range sess.Permissions {
		if m.matchPermission(p, permission) {
			return true
		}
	}

	return false
}

// HasPermissionsAnd checks if a user has all specified permissions (AND logic).
// HasPermissionsAnd 检查用户是否拥有所有指定权限（AND 逻辑）。
func (m *Manager) HasPermissionsAnd(ctx context.Context, loginID string, permissions []string) bool {
	if loginID == "" {
		return false
	}

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		return false
	}

	// 校验每一个必需权限
	for _, need := range permissions {
		if !m.hasPermissionInList(sess.Permissions, need) {
			return false
		}
	}

	return true
}

// HasPermissionsAndByToken checks if a user has all specified permissions by token (AND logic).
// HasPermissionsAndByToken 根据 Token 检查用户是否拥有所有指定权限（AND 逻辑）。
func (m *Manager) HasPermissionsAndByToken(ctx context.Context, tokenValue string, permissions []string) bool {
	sess, err := m.GetSessionByToken(ctx, tokenValue)
	if err != nil {
		return false
	}

	// 校验每一个必需权限
	for _, need := range permissions {
		if !m.hasPermissionInList(sess.Permissions, need) {
			return false
		}
	}

	return true
}

// HasPermissionsOr checks if a user has any of the specified permissions (OR logic).
// HasPermissionsOr 检查用户是否拥有任一指定权限（OR 逻辑）。
func (m *Manager) HasPermissionsOr(ctx context.Context, loginID string, permissions []string) bool {
	if loginID == "" {
		return false
	}

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		return false
	}

	// 任一权限匹配即通过
	for _, need := range permissions {
		if m.hasPermissionInList(sess.Permissions, need) {
			return true
		}
	}

	return false
}

// HasPermissionsOrByToken checks if a user has any of the specified permissions by token (OR logic).
// HasPermissionsOrByToken 根据 Token 检查用户是否拥有任一指定权限（OR 逻辑）。
func (m *Manager) HasPermissionsOrByToken(ctx context.Context, tokenValue string, permissions []string) bool {
	sess, err := m.GetSessionByToken(ctx, tokenValue)
	if err != nil {
		return false
	}

	// 任一权限匹配即通过
	for _, need := range permissions {
		if m.hasPermissionInList(sess.Permissions, need) {
			return true
		}
	}

	return false
}

// ============================================================================
// Role Management - 角色管理
// ============================================================================

// AddRoles adds roles to a user.
// AddRoles 为用户添加角色。
func (m *Manager) AddRoles(ctx context.Context, loginID string, roles []string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	// 获取Session
	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		return err
	}

	sess.addRoles(roles...)
	err = m.saveToStorage(ctx, m.getSessionKey(loginID), *sess)
	if err != nil {
		return err
	}

	return nil
}

// AddRolesByToken adds roles to a user by token.
// AddRolesByToken 根据 Token 为用户添加角色。
func (m *Manager) AddRolesByToken(ctx context.Context, tokenValue string, roles []string) error {
	// 获取Session
	sess, err := m.GetSessionByToken(ctx, tokenValue)
	if err != nil {
		return err
	}

	// 添加角色
	sess.addRoles(roles...)
	// 保存Session
	err = m.saveToStorage(ctx, m.getSessionKey(sess.LoginID), *sess)
	if err != nil {
		return err
	}

	return nil
}

// RemoveRoles removes roles from a user.
// RemoveRoles 删除用户的指定角色。
func (m *Manager) RemoveRoles(ctx context.Context, loginID string, roles []string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	// 获取Session
	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		return err
	}

	sess.removeRoles(roles...)
	err = m.saveToStorage(ctx, m.getSessionKey(loginID), *sess)
	if err != nil {
		return err
	}

	return nil
}

// RemoveRolesByToken removes roles from a user by token.
// RemoveRolesByToken 根据 Token 删除用户的指定角色。
func (m *Manager) RemoveRolesByToken(ctx context.Context, tokenValue string, roles []string) error {
	// 获取Session
	sess, err := m.GetSessionByToken(ctx, tokenValue)
	if err != nil {
		return err
	}

	// 删除角色
	sess.removeRoles(roles...)
	// 保存Session
	err = m.saveToStorage(ctx, m.getSessionKey(sess.LoginID), *sess)
	if err != nil {
		return err
	}

	return nil
}

// GetRoles retrieves the role list for a user.
// GetRoles 获取用户的角色列表。
func (m *Manager) GetRoles(ctx context.Context, loginID string) ([]string, error) {
	if loginID == "" {
		return nil, derror.ErrIDIsEmpty
	}

	// 自定义角色列表获取函数
	if m.CustomRoleListFunc != nil {
		return m.CustomRoleListFunc(loginID, m.config.AuthType)
	}

	// 获取Session
	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		return nil, err
	}

	return sess.Roles, nil
}

// GetRolesByToken retrieves the role list by token.
// GetRolesByToken 根据 Token 获取角色列表。
func (m *Manager) GetRolesByToken(ctx context.Context, tokenValue string) ([]string, error) {
	// 获取Session
	sess, err := m.GetSessionByToken(ctx, tokenValue)
	if err != nil {
		return nil, err
	}

	return sess.Roles, nil
}

// HasRole checks if a user has a specific role.
// HasRole 检查用户是否拥有指定角色。
func (m *Manager) HasRole(ctx context.Context, loginID string, role string) bool {
	if loginID == "" {
		return false
	}

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		return false
	}

	for _, r := range sess.Roles {
		if r == role {
			return true
		}
	}

	return false
}

// HasRoleByToken checks if a user has a specific role by token.
// HasRoleByToken 根据 Token 检查用户是否拥有指定角色。
func (m *Manager) HasRoleByToken(ctx context.Context, tokenValue string, role string) bool {
	sess, err := m.GetSessionByToken(ctx, tokenValue)
	if err != nil {
		return false
	}

	for _, r := range sess.Roles {
		if r == role {
			return true
		}
	}

	return false
}

// HasRolesAnd checks if a user has all specified roles (AND logic).
// HasRolesAnd 检查用户是否拥有所有指定角色（AND 逻辑）。
func (m *Manager) HasRolesAnd(ctx context.Context, loginID string, roles []string) bool {
	if loginID == "" {
		return false
	}

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		return false
	}

	// 校验每一个必需角色
	for _, need := range roles {
		found := false
		for _, r := range sess.Roles {
			if r == need {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// HasRolesAndByToken checks if a user has all specified roles by token (AND logic).
// HasRolesAndByToken 根据 Token 检查用户是否拥有所有指定角色（AND 逻辑）。
func (m *Manager) HasRolesAndByToken(ctx context.Context, tokenValue string, roles []string) bool {
	sess, err := m.GetSessionByToken(ctx, tokenValue)
	if err != nil {
		return false
	}

	// 校验每一个必需角色
	for _, need := range roles {
		found := false
		for _, r := range sess.Roles {
			if r == need {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// HasRolesOr checks if a user has any of the specified roles (OR logic).
// HasRolesOr 检查用户是否拥有任一指定角色（OR 逻辑）。
func (m *Manager) HasRolesOr(ctx context.Context, loginID string, roles []string) bool {
	if loginID == "" {
		return false
	}

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		return false
	}

	// 任一角色匹配即通过
	for _, need := range roles {
		for _, r := range sess.Roles {
			if r == need {
				return true
			}
		}
	}

	return false
}

// HasRolesOrByToken checks if a user has any of the specified roles by token (OR logic).
// HasRolesOrByToken 根据 Token 检查用户是否拥有任一指定角色（OR 逻辑）。
func (m *Manager) HasRolesOrByToken(ctx context.Context, tokenValue string, roles []string) bool {
	sess, err := m.GetSessionByToken(ctx, tokenValue)
	if err != nil {
		return false
	}

	// 任一角色匹配即通过
	for _, need := range roles {
		for _, r := range sess.Roles {
			if r == need {
				return true
			}
		}
	}

	return false
}

// ============================================================================
// Component Getters - 组件获取器
// ============================================================================

// GetConfig retrieves the manager configuration.
// GetConfig 获取管理器配置。
func (m *Manager) GetConfig() *config.Config {
	return m.config
}

// GetGenerator retrieves the token generator.
// GetGenerator 获取 Token 生成器。
func (m *Manager) GetGenerator() adapter.Generator {
	return m.generator
}

// GetStorage retrieves the storage adapter.
// GetStorage 获取存储适配器。
func (m *Manager) GetStorage() adapter.Storage {
	return m.storage
}

// GetSerializer retrieves the serializer adapter.
// GetSerializer 获取序列化器适配器。
func (m *Manager) GetSerializer() adapter.Codec {
	return m.serializer
}

// GetLogger retrieves the logger adapter.
// GetLogger 获取日志适配器。
func (m *Manager) GetLogger() adapter.Log {
	return m.logger
}

// GetPool retrieves the goroutine pool.
// GetPool 获取协程池。
func (m *Manager) GetPool() adapter.Pool {
	return m.pool
}

// GetCustomPermissionListFunc retrieves the custom permission list function.
// GetCustomPermissionListFunc 获取自定义权限列表获取函数。
func (m *Manager) GetCustomPermissionListFunc() func(loginID, authType string) ([]string, error) {
	return m.CustomPermissionListFunc
}

// GetCustomRoleListFunc retrieves the custom role list function.
// GetCustomRoleListFunc 获取自定义角色列表获取函数。
func (m *Manager) GetCustomRoleListFunc() func(loginID, authType string) ([]string, error) {
	return m.CustomRoleListFunc
}

// GetNonceManager retrieves the nonce manager.
// GetNonceManager 获取 Nonce 管理器。
func (m *Manager) GetNonceManager() *nonce.NonceManager {
	return m.nonceManager
}

// GetOAuth2Manager retrieves the OAuth2 manager.
// GetOAuth2Manager 获取 OAuth2 管理器。
func (m *Manager) GetOAuth2Manager() *oauth2.OAuth2Server {
	return m.oauth2Manager
}

// ============================================================================
// INTERNAL METHODS - 内部方法
// ============================================================================

// ============================================================================
// Internal Core Methods - 内部核心方法
// ============================================================================

// getSession retrieves session information (internal method).
// getSession 获取会话信息（内部方法）。
func (m *Manager) getSession(ctx context.Context, loginID string, autoCreate ...bool) (*Session, error) {
	sessData, err := m.storage.Get(ctx, m.getSessionKey(loginID))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	if sessData == nil {
		if len(autoCreate) > 0 && autoCreate[0] {
			return &Session{
				AuthType:      m.config.AuthType,
				LoginID:       loginID,
				CreateTime:    time.Now().Unix(),
				TerminalInfos: make([]TerminalInfo, 0),
				Permissions:   make([]string, 0),
				Roles:         make([]string, 0),
			}, nil
		}

		return nil, derror.ErrSessionNotFound
	}

	bytesData, err := utils.ToBytes(sessData)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrTypeConvert, err)
	}

	var sess Session
	err = m.serializer.Decode(bytesData, &sess)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}

	return &sess, nil
}

// getTokenInfo retrieves token information (internal method).
// getTokenInfo 获取 Token 信息（内部方法）。
func (m *Manager) getTokenInfo(ctx context.Context, tokenValue string) (*TokenInfo, error) {
	tokenInfoData, err := m.storage.Get(ctx, m.getTokenKey(tokenValue))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	if tokenInfoData == nil {
		return nil, derror.ErrInvalidToken
	}

	tokenInfoBytes, err := utils.ToBytes(tokenInfoData)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrTypeConvert, err)
	}

	switch string(tokenInfoBytes) {
	case string(TokenStateLogout):
		return nil, derror.ErrInvalidToken
	case string(TokenStateKickOut):
		return nil, derror.ErrTokenKickout
	case string(TokenStateReplaced):
		return nil, derror.ErrTokenReplaced
	}

	var tokenInfo TokenInfo
	err = m.serializer.Decode(tokenInfoBytes, &tokenInfo)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrTypeConvert, err)
	}

	return &tokenInfo, nil
}

// loginGetSession retrieves session for login operation (internal method).
// loginGetSession 获取登录操作的会话信息（内部方法）。
func (m *Manager) loginGetSession(ctx context.Context, loginID string) (*Session, error) {
	sessData, err := m.storage.Get(ctx, m.getSessionKey(loginID))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	if sessData == nil {
		return nil, nil
	}

	bytesData, err := utils.ToBytes(sessData)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrTypeConvert, err)
	}

	var sess Session
	err = m.serializer.Decode(bytesData, &sess)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}

	return &sess, nil
}

// checkLoginInternal performs the core login validation logic (internal method).
// checkLoginInternal 执行登录状态的核心验证逻辑（内部方法）。
func (m *Manager) checkLoginInternal(ctx context.Context, tokenValue string) error {
	// 获取 tokenInfo
	tokenInfo, err := m.getTokenInfo(ctx, tokenValue)
	if err != nil {
		return err
	}

	// 检查最大不活跃时长
	if m.config.ActiveTimeout > 0 {
		timeStampAny, err := m.storage.Get(ctx, m.getActiveKey(tokenValue))
		if err == nil && timeStampAny != nil {
			timeStamp, err := utils.ToInt64(timeStampAny)
			if err != nil {
				_ = m.storage.Delete(ctx, m.getActiveKey(tokenValue))
			} else if time.Now().Unix()-timeStamp > m.config.ActiveTimeout {
				// Token 已超过最大不活跃时长，执行踢出操作
				_ = m.Kickout(ctx, tokenValue)
				return derror.ErrTokenKickout
			}
		}
	}

	// 异步续期
	if m.config.AutoRenew && m.config.Timeout > 0 {
		if ttl, err := m.storage.TTL(ctx, m.getTokenKey(tokenValue)); err == nil && ttl > 0 {
			ttlSeconds := int64(ttl.Seconds())
			if ttlSeconds > 0 &&
				(m.config.RenewMaxRefresh <= 0 || ttlSeconds <= m.config.RenewMaxRefresh) &&
				(m.config.RenewInterval <= 0 || !m.storage.Exists(ctx, m.getRenewKey(tokenValue))) {

				renewFunc := func() {
					m.renewFunc(context.Background(), tokenValue, tokenInfo.LoginID)
				}

				if m.pool != nil {
					_ = m.pool.Submit(renewFunc)
				} else {
					go renewFunc()
				}
			}
		}
	}

	// 异步活跃时长
	if m.config.ActiveTimeout > 0 {
		activeFunc := func() {
			_ = m.storage.Set(ctx, m.getActiveKey(tokenValue), time.Now().Unix(), m.getExpiration())
		}
		if m.pool != nil {
			_ = m.pool.Submit(activeFunc)
		} else {
			go activeFunc()
		}
	}

	return nil
}

// ============================================================================
// Internal Login Logic - 内部登录逻辑
// ============================================================================

// cleanExpiredTerminals removes expired tokens from session (internal method).
// cleanExpiredTerminals 清理会话中已过期的 token（内部方法）。
func (m *Manager) cleanExpiredTerminals(ctx context.Context, sess *Session) error {
	if sess == nil || len(sess.TerminalInfos) == 0 {
		return nil
	}

	var validTerminals []TerminalInfo
	hasExpired := false

	for _, ti := range sess.TerminalInfos {
		// 尝试获取 token 信息，如果成功则说明 token 有效
		_, err := m.getTokenInfo(ctx, ti.Token)
		if err == nil {
			validTerminals = append(validTerminals, ti)
		} else {
			hasExpired = true
		}
	}

	// 如果有过期的 token，更新 session
	if hasExpired {
		sess.TerminalInfos = validTerminals
		if err := m.saveToStorage(ctx, m.getSessionKey(sess.LoginID), *sess); err != nil {
			return err
		}
	}

	return nil
}

// handleConcurrency handles login concurrency strategy (internal method).
// handleConcurrency 处理登录并发策略（内部方法）。
func (m *Manager) handleConcurrency(
	ctx context.Context,
	sess *Session,
	loginID, device string,
) (reuseToken string, handled bool, err error) {
	// 清理已过期的 token
	if err = m.cleanExpiredTerminals(ctx, sess); err != nil {
		return "", false, err
	}

	if !m.config.IsConcurrent {
		// 不允许并发：踢掉旧会话
		if m.config.ConcurrencyScope == config.ConcurrencyScopeAccount {
			_ = m.removeTerminalInfosAndTokens(ctx, sess)
		} else if m.config.ConcurrencyScope == config.ConcurrencyScopeDevice {
			_ = m.removeTerminalInfosAndTokens(ctx, sess, device)
		}
		return "", true, nil
	}

	if m.config.IsShare {
		// 允许共享：尝试复用
		var token string
		if m.config.ConcurrencyScope == config.ConcurrencyScopeAccount {
			token = m.getTokenAndShare(ctx, sess)
		} else if m.config.ConcurrencyScope == config.ConcurrencyScopeDevice {
			token = m.getTokenAndShare(ctx, sess, device)
		}
		if token != "" {
			return token, true, nil
		}
	}

	if m.config.ConcurrencyScope == config.ConcurrencyScopeAccount {
		if m.config.MaxLoginCount > 0 && int64(len(sess.TerminalInfos)) >= m.config.MaxLoginCount {
			if err := m.removeOldestTerminalInfoAndToken(ctx, sess); err != nil {
				return "", false, err
			}
			return "", true, nil
		}
	} else if m.config.ConcurrencyScope == config.ConcurrencyScopeDevice {
		if m.config.MaxLoginCount > 0 && int64(len(sess.getTerminalsByDevice(device))) >= m.config.MaxLoginCount {
			if err := m.removeOldestTerminalInfoAndToken(ctx, sess, device); err != nil {
				return "", false, err
			}
			return "", true, nil
		}
	}

	return "", false, nil
}

// getTokenAndShare retrieves and shares a token (internal method).
// getTokenAndShare 获取 Token 并共享（内部方法）。
func (m *Manager) getTokenAndShare(ctx context.Context, sess *Session, device ...string) string {
	if len(sess.TerminalInfos) == 0 {
		return ""
	}

	// 获取候选的 terminals
	var candidates []TerminalInfo
	if len(device) > 0 {
		// 指定设备：获取该设备的所有 terminals
		candidates = sess.getTerminalsByDevice(device[0])
	} else {
		// 账号级别：获取所有 terminals
		candidates = sess.TerminalInfos
	}

	if len(candidates) == 0 {
		return ""
	}

	// 获取最后一个（最新的）token 并直接复活
	// 注意：如果 token 还在 session 中，说明它没有被踢出或顶替
	// 即使过期了，我们也可以通过重置 TTL 来复活它
	terminalInfo := candidates[len(candidates)-1]

	// 续期 session
	_ = m.storage.Expire(ctx, m.getSessionKey(terminalInfo.LoginID), m.getExpiration())

	// 续期 Token（如果已过期，这会复活它）
	tokenInfo := TokenInfo{
		AuthType:   m.config.AuthType,
		LoginID:    terminalInfo.LoginID,
		Device:     terminalInfo.Device,
		DeviceId:   terminalInfo.DeviceId,
		CreateTime: terminalInfo.CreateTime,
	}
	_ = m.storage.Set(ctx, m.getTokenKey(terminalInfo.Token), tokenInfo, m.getExpiration())

	// 续期或重新设置 metadata
	if m.config.RenewInterval > 0 {
		_ = m.storage.Set(ctx, m.getRenewKey(terminalInfo.Token), time.Now().Unix(), time.Duration(m.config.RenewInterval)*time.Second)
	}
	// 设置最大不活跃时长
	if m.config.ActiveTimeout > 0 {
		_ = m.storage.Set(ctx, m.getActiveKey(terminalInfo.Token), time.Now().Unix(), m.getExpiration())
	}

	return terminalInfo.Token
}

// removeOldestTerminalInfoAndToken removes the oldest terminal and its token (internal method).
// removeOldestTerminalInfoAndToken 移除最旧的终端信息并删除对应的 Token（内部方法）。
func (m *Manager) removeOldestTerminalInfoAndToken(ctx context.Context, sess *Session, device ...string) error {
	if len(device) > 0 {
		terminalInfo, ok := sess.removeOldestTerminal(device...)
		if ok {
			// 保存会话数据
			if err := m.saveToStorage(ctx, m.getSessionKey(sess.LoginID), *sess); err != nil {
				return err
			}
			// 设置token状态为踢出
			if err := m.storage.Set(ctx, m.getTokenKey(terminalInfo.Token), string(TokenStateKickOut), m.getExpiration()); err != nil {
				return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
			}
			// 清理 metadata
			if err := m.storage.Delete(ctx, m.getRenewKey(terminalInfo.Token)); err != nil {
				return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
			}
			if err := m.storage.Delete(ctx, m.getActiveKey(terminalInfo.Token)); err != nil {
				return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
			}
		}
		return nil
	}

	terminalInfo, ok := sess.removeOldestTerminal()
	if ok {
		// 保存会话数据
		if err := m.saveToStorage(ctx, m.getSessionKey(sess.LoginID), *sess); err != nil {
			return err
		}
		// 设置token状态为踢出
		if err := m.storage.Set(ctx, m.getTokenKey(terminalInfo.Token), string(TokenStateKickOut), m.getExpiration()); err != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
		// 清理 metadata
		if err := m.storage.Delete(ctx, m.getRenewKey(terminalInfo.Token)); err != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
		if err := m.storage.Delete(ctx, m.getActiveKey(terminalInfo.Token)); err != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
	}
	return nil
}

// removeTerminalInfosAndTokens removes terminal information and tokens (internal method).
// removeTerminalInfosAndTokens 移除终端信息和 Token（内部方法）。
func (m *Manager) removeTerminalInfosAndTokens(ctx context.Context, sess *Session, device ...string) error {
	if len(device) > 0 {
		// 移除指定设备类型的终端信息
		terminalInfos := sess.removeTerminalByDevice(device[0])

		// 如果 session 中没有剩余终端,删除整个 session
		if len(sess.TerminalInfos) == 0 {
			if err := m.storage.Delete(ctx, m.getSessionKey(sess.LoginID)); err != nil {
				return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
			}
		} else {
			// 否则保存更新后的 session
			if err := m.saveToStorage(ctx, m.getSessionKey(sess.LoginID), *sess); err != nil {
				return err
			}
		}

		// 将所有的Token设置为踢出
		for _, info := range terminalInfos {
			err := m.storage.Set(ctx, m.getTokenKey(info.Token), string(TokenStateKickOut), m.getExpiration())
			if err != nil {
				return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
			}
		}

		// 清理附属 metadata
		tokens := make([]string, len(terminalInfos))
		for i, info := range terminalInfos {
			tokens[i] = info.Token
		}
		if err := m.cleanTokenMetadata(ctx, tokens); err != nil {
			return err
		}

		return nil
	}

	// 获取旧的终端信息
	oldTerminalInfos := sess.TerminalInfos

	// 移除终端信息
	sess.TerminalInfos = make([]TerminalInfo, 0)

	// session 为空,删除整个 session
	if err := m.storage.Delete(ctx, m.getSessionKey(sess.LoginID)); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	// 将所有的Token设置为踢出
	for _, terminalInfo := range oldTerminalInfos {
		err := m.storage.Set(ctx, m.getTokenKey(terminalInfo.Token), string(TokenStateKickOut), m.getExpiration())
		if err != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
	}

	// 清理附属 metadata
	tokens := make([]string, len(oldTerminalInfos))
	for i, info := range oldTerminalInfos {
		tokens[i] = info.Token
	}
	if err := m.cleanTokenMetadata(ctx, tokens); err != nil {
		return err
	}

	return nil
}

// ============================================================================
// Internal Logout Logic - 内部登出逻辑
// ============================================================================

// logoutTerminals performs common logout logic (internal method).
// logoutTerminals 通用登出逻辑：移除终端 + 删除 token + 清理 metadata（内部方法）。
func (m *Manager) logoutTerminals(
	ctx context.Context,
	loginID string,
	removalFunc func(*Session) []TerminalInfo,
) error {
	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		return err
	}
	if sess == nil {
		return nil // session 不存在，登出无害
	}

	removed := removalFunc(sess)
	if len(removed) == 0 {
		return nil
	}

	// 如果 session 中没有剩余终端，删除整个 session
	if len(sess.TerminalInfos) == 0 {
		if err = m.storage.Delete(ctx, m.getSessionKey(loginID)); err != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
	} else {
		// 否则保存更新后的 session
		if err = m.saveToStorage(ctx, m.getSessionKey(loginID), *sess); err != nil {
			return err
		}
	}

	// 提取 token 列表
	tokens := make([]string, len(removed))
	tokenKeys := make([]string, len(removed))
	for i, info := range removed {
		tokens[i] = info.Token
		tokenKeys[i] = m.getTokenKey(info.Token)
	}

	// 删除主 token keys
	if err = m.storage.Delete(ctx, tokenKeys...); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	// 清理附属 metadata
	if err = m.cleanTokenMetadata(ctx, tokens); err != nil {
		return err
	}

	// 触发登出事件
	for _, info := range removed {
		m.triggerEvent(listener.EventLogout, loginID, info.Device, info.DeviceId, info.Token, nil)
	}

	return nil
}

// cleanTokenMetadata cleans token metadata in batch (internal method).
// cleanTokenMetadata 批量清理 token 的附属元数据（续期 key、活跃时间 key）（内部方法）。
func (m *Manager) cleanTokenMetadata(ctx context.Context, tokens []string) error {
	if len(tokens) == 0 {
		return nil
	}

	keys := make([]string, 0, len(tokens)*2)
	for _, token := range tokens {
		keys = append(keys, m.getRenewKey(token), m.getActiveKey(token))
	}

	if err := m.storage.Delete(ctx, keys...); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	return nil
}

// ============================================================================
// Internal Online Status Logic - 内部在线状态逻辑
// ============================================================================

// TerminalRemovalFunc defines how to remove terminals from a session.
// TerminalRemovalFunc 定义如何从 Session 中移除终端。
type TerminalRemovalFunc func(sess *Session) []TerminalInfo

// processTerminals performs common terminal processing logic (internal method).
// processTerminals 通用终端处理逻辑（内部方法）。
func (m *Manager) processTerminals(
	ctx context.Context,
	loginID string,
	removalFunc TerminalRemovalFunc,
	state TokenState,
) error {
	// 加载 Session
	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		return err
	}

	// 执行移除策略
	removedTerminals := removalFunc(sess)

	// 如果有移除项，更新 session
	if len(removedTerminals) > 0 {
		// 如果 session 中没有剩余终端，删除整个 session
		if len(sess.TerminalInfos) == 0 {
			if err = m.storage.Delete(ctx, m.getSessionKey(loginID)); err != nil {
				return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
			}
		} else {
			// 否则保存更新后的 session
			if err = m.saveToStorage(ctx, m.getSessionKey(loginID), *sess); err != nil {
				return err
			}
		}
	}

	// 对每个被移除的 token 执行清理
	for _, info := range removedTerminals {
		token := info.Token

		// 设置 token 状态
		if err = m.storage.Set(ctx, m.getTokenKey(token), string(state), m.getExpiration()); err != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}

		// 删除续期 key
		if err = m.storage.Delete(ctx, m.getRenewKey(token)); err != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}

		// 删除活跃时间 key
		if err = m.storage.Delete(ctx, m.getActiveKey(token)); err != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
	}

	// 触发对应事件
	var event listener.Event
	switch state {
	case TokenStateKickOut:
		event = listener.EventKickout
	case TokenStateReplaced:
		event = listener.EventReplace
	}

	if event != "" {
		for _, info := range removedTerminals {
			m.triggerEvent(event, loginID, info.Device, info.DeviceId, info.Token, nil)
		}
	}

	return nil
}

// ============================================================================
// Internal Disable Logic - 内部封禁逻辑
// ============================================================================

// isDisable checks if an account is disabled (internal method).
// isDisable 检查账号是否被封禁（内部方法）。
func (m *Manager) isDisable(ctx context.Context, loginID string) bool {
	return m.storage.Exists(ctx, m.getDisableKey(loginID))
}

// ============================================================================
// Internal Token Filter - 内部 Token 过滤
// ============================================================================

// filterTokens filters tokens based on checkAlive flag (internal method).
// filterTokens 根据 checkAlive 决定是否验证 token 有效性，并返回 token 列表（内部方法）。
func (m *Manager) filterTokens(ctx context.Context, terminals []TerminalInfo, checkAlive bool) ([]string, error) {
	if len(terminals) == 0 {
		return []string{}, nil
	}

	if !checkAlive {
		// 不检查存活：直接返回所有 token（预分配容量）
		tokens := make([]string, len(terminals))
		for i, ti := range terminals {
			tokens[i] = ti.Token
		}
		return tokens, nil
	}

	// 检查每个 token 是否有效（调用 GetTokenInfo）
	var tokens []string // 无法预知数量，动态 append
	for _, ti := range terminals {
		if _, err := m.GetTokenInfo(ctx, ti.Token); err == nil {
			tokens = append(tokens, ti.Token)
		}
		// 若 token 无效（过期/被踢等），跳过
	}
	return tokens, nil
}

// ============================================================================
// Internal Permission Logic - 内部权限逻辑
// ============================================================================

// matchPermission matches permission with wildcard support (internal method).
// matchPermission 权限匹配（支持通配符）（内部方法）。
func (m *Manager) matchPermission(pattern, permission string) bool {
	// 精确匹配或通配符
	if pattern == PermissionWildcard || pattern == permission {
		return true
	}

	// 支持通配符，例如 user:* 匹配 user:add, user:delete等
	wildcardSuffix := PermissionSeparator + PermissionWildcard
	if strings.HasSuffix(pattern, wildcardSuffix) {
		prefix := strings.TrimSuffix(pattern, PermissionWildcard)
		return strings.HasPrefix(permission, prefix)
	}

	// 支持 user:*:view 这样的模式
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

// hasPermissionInList checks if permission exists in permission list (internal method).
// hasPermissionInList 判断权限是否存在于权限列表中（内部方法）。
func (m *Manager) hasPermissionInList(perms []string, permission string) bool {
	for _, p := range perms {
		if m.matchPermission(p, permission) {
			return true
		}
	}
	return false
}

// ============================================================================
// Internal Renewal Logic - 内部续期逻辑
// ============================================================================

// renewFunc performs token renewal (internal method).
// renewFunc 续期函数（内部方法）。
func (m *Manager) renewFunc(ctx context.Context, tokenValue, loginID string) {
	// 参数为空校验
	if tokenValue == "" || loginID == "" {
		return
	}

	// 续期Token
	_ = m.storage.Expire(ctx, m.getTokenKey(tokenValue), m.getExpiration())

	// 续期Session
	_ = m.storage.Expire(ctx, m.getSessionKey(loginID), m.getExpiration())

	// 设置最小续期间隔标记
	if m.config.RenewInterval > 0 {
		_ = m.storage.Set(ctx, m.getRenewKey(tokenValue), time.Now().Unix(), time.Duration(m.config.RenewInterval)*time.Second)
	}
}

// ============================================================================
// Internal Helper Methods - 内部辅助方法
// ============================================================================

// getTokenKey generates the storage key for a token.
// getTokenKey 获取 Token 存储键。
func (m *Manager) getTokenKey(tokenValue string) string {
	return m.config.KeyPrefix + m.config.AuthType + tokenValue
}

// getSessionKey generates the storage key for a session.
// getSessionKey 获取会话存储键。
func (m *Manager) getSessionKey(loginID string) string {
	return m.config.KeyPrefix + m.config.AuthType + SessionKeyPrefix + loginID
}

// getRenewKey generates the storage key for token renewal tracking.
// getRenewKey 获取 Token 续期追踪键。
func (m *Manager) getRenewKey(tokenValue string) string {
	return m.config.KeyPrefix + m.config.AuthType + RenewKeyPrefix + tokenValue
}

// getActiveKey generates the storage key for token activity tracking.
// getActiveKey 获取 Token 活跃时间追踪键。
func (m *Manager) getActiveKey(tokenValue string) string {
	return m.config.KeyPrefix + m.config.AuthType + ActivePrefix + tokenValue
}

// getDisableKey generates the storage key for account disable status.
// getDisableKey 获取账号禁用状态存储键。
func (m *Manager) getDisableKey(loginID string) string {
	return m.config.KeyPrefix + m.config.AuthType + DisableKeyPrefix + loginID
}

// triggerEvent triggers an event through the event manager.
// triggerEvent 通过事件管理器触发事件。
func (m *Manager) triggerEvent(event listener.Event, loginID, device, deviceId, token string, extra map[string]any) {
	if m.eventManager == nil {
		return
	}

	m.eventManager.Trigger(&listener.EventData{
		Event:     event,
		AuthType:  m.config.AuthType,
		LoginID:   loginID,
		Device:    device,
		DeviceId:  deviceId,
		Token:     token,
		Extra:     extra,
		Timestamp: time.Now().Unix(),
	})
}

// getExpiration calculates token expiration duration from configuration.
// getExpiration 从配置中计算 Token 过期时长。
func (m *Manager) getExpiration() time.Duration {
	if m.config.Timeout > 0 {
		return time.Duration(m.config.Timeout) * time.Second
	}
	return 0
}

// getDeviceAndDeviceId extracts device type and device ID from parameters.
// getDeviceAndDeviceId 获取设备类型和设备 ID。
// 规则：device 和 deviceId 是两个独立的过滤维度，互不影响
func (m *Manager) getDeviceAndDeviceId(deviceAndDeviceId ...string) (string, string) {
	device := ""
	deviceId := ""

	if len(deviceAndDeviceId) > 0 {
		device = strings.TrimSpace(deviceAndDeviceId[0])
	}

	if len(deviceAndDeviceId) > 1 {
		deviceId = strings.TrimSpace(deviceAndDeviceId[1])
	}

	return device, deviceId
}

// saveToStorage serializes and saves data to storage backend.
// saveToStorage 将指定类型的数据序列化并存储到存储后端。
func (m *Manager) saveToStorage(
	ctx context.Context,
	key string,
	value any,
	expiration ...time.Duration,
) error {

	// 序列化为字节
	bytesData, err := m.serializer.Encode(value)
	if err != nil {
		return fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}

	// 构建过期时长
	duration := m.getExpiration()
	if len(expiration) > 0 {
		duration = expiration[0]
	}

	// 存储到后端
	if err = m.storage.Set(ctx, key, bytesData, duration); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	return nil
}
