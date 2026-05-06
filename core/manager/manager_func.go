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
	"sync"
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

// NewManager creates a new Manager instance with the provided components. NewManager 创建一个新的 Manager 实例,使用提供的组件。
func NewManager(
	cfg *config.Config,
	generator adapter.Generator,
	storage adapter.Storage,
	serializer adapter.Codec,
	logger adapter.Log,
	pool adapter.Pool,
	CustomPermissionListFunc, CustomRoleListFunc func(loginID, authType string) ([]string, error),
	CustomPermissionListExtFunc, CustomRoleListExtFunc func(loginID, device, deviceId, authType string) ([]string, error),
) *Manager {

	// Use default config if cfg is nil cfg 为 nil 时使用默认配置
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	// Create default token generator if generator is nil generator 为 nil 时创建默认 Token 生成器
	if generator == nil {
		generator = dgenerator.NewGenerator(cfg.Timeout, cfg.JwtSecretKey, cfg.TokenStyle)
	}

	// Use memory storage if storage is nil storage 为 nil 时使用内存存储
	if storage == nil {
		storage = memory.NewStorage()
	}

	// Use JSON serializer if serializer is nil serializer 为 nil 时使用 JSON 序列化器
	if serializer == nil {
		serializer = djson.NewJSONSerializer()
	}

	// Use nop logger if logger is nil logger 为 nil 时使用空日志记录器
	if logger == nil {
		logger = nop.NewNopLogger()
	}

	// Create default goroutine pool if AutoRenew is enabled and pool is nil 启用自动续期且 pool 为 nil 时使用默认协程池
	if cfg.AutoRenew && pool == nil {
		pool = ants.NewRenewPoolManagerWithDefaultConfig()
	}

	// Return initialized Manager instance 返回初始化完成的 Manager 实例
	return &Manager{
		config:                      cfg,
		generator:                   generator,
		storage:                     storage,
		serializer:                  serializer,
		logger:                      logger,
		pool:                        pool,
		nonceManager:                nonce.NewNonceManager(cfg.AuthType, cfg.KeyPrefix, storage, nonce.DefaultNonceTTL),
		oauth2Manager:               oauth2.NewOAuth2Server(cfg.AuthType, cfg.KeyPrefix, storage, serializer),
		eventManager:                listener.NewManager(logger),
		CustomPermissionListFunc:    CustomPermissionListFunc,
		CustomRoleListFunc:          CustomRoleListFunc,
		CustomPermissionListExtFunc: CustomPermissionListExtFunc,
		CustomRoleListExtFunc:       CustomRoleListExtFunc,
	}
}

// CloseManager closes the manager and releases all resources. CloseManager 关闭管理器并释放所有资源。
func (m *Manager) CloseManager() {
	// Safely stop goroutine pool and set to nil 安全关闭协程池并置空
	if m.pool != nil {
		m.pool.Stop()
		m.pool = nil
	}

	// Wait for all async events to complete 等待所有异步事件完成
	if m.eventManager != nil {
		m.eventManager.Wait()
	}

	// Close storage if supported 若存储适配器支持 Close 则释放连接资源
	if storageCloser, ok := m.storage.(interface{ Close() error }); ok {
		if err := storageCloser.Close(); err != nil {
			m.logger.Errorf("manager.CloseManager: failed to close storage, error=%v", err)
		}
	}

	// Flush and close logger if it implements LogControl interface 若日志记录器实现了 LogControl 接口则执行 Flush 和 Close
	if logControl, ok := m.logger.(adapter.LogControl); ok {
		logControl.Flush()
		logControl.Close()
	}
}

// Login performs user login and returns a token. Login 执行用户登录并返回 token。
func (m *Manager) Login(ctx context.Context, loginID string, deviceAndDeviceId ...string) (string, error) {
	return m.LoginWithTimeout(ctx, loginID, 0, deviceAndDeviceId...)
}

// LoginWithTimeout performs user login with a custom token timeout and returns a token. LoginWithTimeout 执行用户登录并返回 token，使用指定的过期时间（0 或负数则使用全局配置）。
func (m *Manager) LoginWithTimeout(ctx context.Context, loginID string, timeout time.Duration, deviceAndDeviceId ...string) (string, error) {
	if loginID == "" {
		return "", derror.ErrIDIsEmpty
	}

	unlock := m.lockLoginWrite(loginID)
	defer unlock()

	if m.isDisable(ctx, loginID) {
		return "", derror.ErrAccountDisabled
	}

	device, deviceId := m.getDeviceAndDeviceId(deviceAndDeviceId...)

	// Load existing session 尝试加载现有 session
	sess, err := m.loginGetSession(ctx, loginID)
	if err != nil {
		return "", err
	}

	destroyedSession := false // destroyedSession records whether old terminals removed the whole session destroyedSession 记录旧终端是否清空整个会话

	// Handle concurrency strategy 处理并发策略
	if sess != nil {
		token, handled, sessionDestroyed, handleErr := m.handleConcurrency(ctx, sess, loginID, device)
		if handleErr != nil {
			return "", handleErr
		}
		destroyedSession = sessionDestroyed
		if handled {
			if token != "" {
				return token, nil // 复用 token
			}
		}
	}

	// Generate new token 生成新 token
	token, err := m.generator.Generate(loginID, device, deviceId)
	if err != nil {
		return "", err
	}

	// Record create time 记录创建时间
	createTime := time.Now().Unix()

	createdSession := sess == nil || destroyedSession // createdSession records whether this login creates a new session createdSession 记录本次登录是否创建新会话
	if createdSession {
		sess = &Session{
			AuthType:      m.config.AuthType,
			LoginID:       loginID,
			CreateTime:    createTime,
			TerminalInfos: make([]TerminalInfo, 0),
			Permissions:   make([]string, 0),
			Roles:         make([]string, 0),
		}
	}

	// Increase history terminal count 递增历史终端计数
	sess.HistoryTerminalCount++

	// Append terminal info 添加终端信息
	sess.TerminalInfos = append(sess.TerminalInfos, TerminalInfo{
		Token:      token,
		LoginID:    loginID,
		Device:     device,
		DeviceId:   deviceId,
		CreateTime: createTime,
		Index:      sess.HistoryTerminalCount, // 设置历史登录顺序索引
	})

	// Calculate expiration duration 计算过期时长
	expiration := m.getExpiration()
	if timeout > 0 {
		expiration = timeout
	}

	// Save session without shortening existing TTL 保存 session，避免缩短已有 TTL
	if err = m.saveSessionWithMinTTL(ctx, m.getSessionKey(loginID), *sess, expiration); err != nil {
		return "", err
	}

	// Save token info 保存 token info
	if err = m.saveToStorage(ctx, m.getTokenKey(token), TokenInfo{
		AuthType:   m.config.AuthType,
		LoginID:    loginID,
		Device:     device,
		DeviceId:   deviceId,
		CreateTime: createTime,
		Timeout:    m.timeoutToSeconds(expiration),
	}, expiration); err != nil {
		m.rollbackLogin(ctx, sess, loginID, token, expiration)
		return "", err
	}

	// Initialize token metadata 初始化 token 元数据
	if m.config.RenewInterval > 0 {
		if err = m.storage.Set(ctx, m.getRenewKey(token), time.Now().Unix(), time.Duration(m.config.RenewInterval)*time.Second); err != nil {
			m.rollbackLogin(ctx, sess, loginID, token, expiration)
			return "", fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
	}
	if m.config.ActiveTimeout > 0 {
		if err = m.storage.Set(ctx, m.getActiveKey(token), time.Now().Unix(), expiration); err != nil {
			m.rollbackLogin(ctx, sess, loginID, token, expiration)
			return "", fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
	}

	unlock()
	unlock = func() {}

	if destroyedSession {
		// Trigger session destroy event after lock release 释放账号写锁后触发销毁 Session 事件
		m.triggerEvent(listener.EventDestroySession, loginID, "", "", "", nil)
	}
	if createdSession {
		// Trigger session create event after successful persistence 持久化成功后触发创建 Session 事件
		m.triggerEvent(listener.EventCreateSession, loginID, "", "", "", nil)
	}

	// Trigger login event 触发登录事件
	m.triggerEvent(listener.EventLogin, loginID, device, deviceId, token, nil)

	return token, nil
}

// LoginByToken performs login renewal based on an existing token. LoginByToken 根据 Token 续期登录。
func (m *Manager) LoginByToken(ctx context.Context, tokenValue string) error {
	if tokenValue == "" {
		return derror.ErrInvalidToken
	}

	tokenInfo, err := m.getTokenInfo(ctx, tokenValue)
	if err != nil {
		return err
	}

	unlock := m.lockLoginWrite(tokenInfo.LoginID)
	defer unlock()

	// Reload token after acquiring lock 加锁后重新读取 token，避免并发下复活已失效 token
	tokenInfo, err = m.getTokenInfo(ctx, tokenValue)
	if err != nil {
		return err
	}

	// Check account disable status 检查账号是否被封禁
	if m.isDisable(ctx, tokenInfo.LoginID) {
		return derror.ErrAccountDisabled
	}

	sess, err := m.getSession(ctx, tokenInfo.LoginID)
	if err != nil {
		if errors.Is(err, derror.ErrSessionNotFound) {
			return derror.ErrInvalidToken
		}
		return err
	}

	// Validate token in session terminals 验证 token 是否在 session 的 TerminalInfos 中
	if !sess.hasTerminalToken(tokenValue) {
		return derror.ErrInvalidToken
	}
	if err := m.ensureTerminalTokenAlive(ctx, tokenValue); err != nil {
		return err
	}

	// Renew token and session asynchronously 异步续期 Token 和 Session
	renewFunc := func() {
		bg := context.Background()
		unlock := m.lockLoginWrite(tokenInfo.LoginID)
		defer unlock()

		// Reload token under lock 锁内重新读取 Token，避免续期已失效 Token
		latestTokenInfo, err := m.getTokenInfo(bg, tokenValue)
		if err != nil {
			m.logger.Errorf("manager.LoginByToken: token is no longer valid, token=%s, error=%v", tokenValue, err)
			return
		}

		// Validate token is still attached to session 确认 Token 仍属于当前会话
		latestSession, err := m.getSession(bg, latestTokenInfo.LoginID)
		if err != nil {
			m.logger.Errorf("manager.LoginByToken: failed to reload session, loginID=%s, error=%v", latestTokenInfo.LoginID, err)
			return
		}
		if !latestSession.hasTerminalToken(tokenValue) {
			m.logger.Errorf("manager.LoginByToken: token not found in session, token=%s", tokenValue)
			return
		}

		expiration := m.resolveTokenExpiration(latestTokenInfo)
		sessionKey := m.getSessionKey(latestTokenInfo.LoginID)

		// Renew session without shortening existing TTL 续期 session，避免缩短已有 TTL
		if err := m.saveSessionWithMinTTL(bg, sessionKey, *latestSession, expiration); err != nil {
			m.logger.Errorf("manager.LoginByToken: failed to save session, loginID=%s, error=%v", latestTokenInfo.LoginID, err)
		}
		// Renew token 续期 Token
		if err := m.expireTokenIfLimited(bg, tokenValue, expiration); err != nil {
			m.logger.Errorf("manager.LoginByToken: failed to expire token, token=%s, error=%v", tokenValue, err)
		}

		// Update metadata 更新 metadata
		if m.config.RenewInterval > 0 {
			if err := m.storage.Set(bg, m.getRenewKey(tokenValue), time.Now().Unix(), time.Duration(m.config.RenewInterval)*time.Second); err != nil {
				m.logger.Errorf("manager.LoginByToken: failed to set renew key, token=%s, error=%v", tokenValue, err)
			}
		}
		if m.config.ActiveTimeout > 0 {
			if err := m.storage.Set(bg, m.getActiveKey(tokenValue), time.Now().Unix(), expiration); err != nil {
				m.logger.Errorf("manager.LoginByToken: failed to set active key, token=%s, error=%v", tokenValue, err)
			}
		}

		unlock()
		unlock = func() {}

		// Trigger renew event 触发续期事件
		m.triggerEvent(listener.EventRenew, latestTokenInfo.LoginID, latestTokenInfo.Device, latestTokenInfo.DeviceId, tokenValue, nil)
	}

	m.submitAsync("LoginByToken", renewFunc)

	return nil
}

// Logout logs out a user by token. Logout 根据 Token 登出用户。
func (m *Manager) Logout(ctx context.Context, tokenValue string) error {
	if tokenValue == "" {
		return derror.ErrInvalidToken
	}

	sess, err := m.GetSessionByToken(ctx, tokenValue)
	if err != nil {
		return err
	}

	return m.logoutTerminals(ctx, sess.LoginID, func(sess *Session) []TerminalInfo {
		if info, ok := sess.removeTerminalByToken(tokenValue); ok {
			return []TerminalInfo{info}
		}
		return nil
	})
}

// LogoutByDevice logs out all terminals of a specific device type. LogoutByDevice 根据设备类型登出所有该设备的终端。
func (m *Manager) LogoutByDevice(ctx context.Context, loginID string, device string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}
	device = strings.TrimSpace(device)
	if device == "" {
		return derror.ErrInvalidParam
	}

	return m.logoutTerminals(ctx, loginID, func(sess *Session) []TerminalInfo {
		return sess.removeTerminalByDevice(device)
	})
}

// LogoutByDeviceAndDeviceId logs out a user by device type and device ID. LogoutByDeviceAndDeviceId 根据设备类型和设备ID登出用户。
func (m *Manager) LogoutByDeviceAndDeviceId(ctx context.Context, loginID string, deviceAndDeviceId ...string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	device, deviceId := m.getDeviceAndDeviceId(deviceAndDeviceId...)
	if device == "" || deviceId == "" {
		return derror.ErrInvalidParam
	}
	return m.logoutTerminals(ctx, loginID, func(sess *Session) []TerminalInfo {
		return sess.removeTerminalByDeviceAndDeviceId(device, deviceId)
	})
}

// LogoutByLoginID logs out all terminals for the specified loginID. LogoutByLoginID 登出指定 loginID 的所有终端。
func (m *Manager) LogoutByLoginID(ctx context.Context, loginID string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	return m.logoutTerminals(ctx, loginID, func(sess *Session) []TerminalInfo {
		return sess.removeAllTerminals()
	})
}

// Kickout kicks out a user by token. Kickout 根据 Token 踢人下线。
func (m *Manager) Kickout(ctx context.Context, tokenValue string) error {
	sess, err := m.GetSessionByToken(ctx, tokenValue)
	if err != nil {
		return err
	}

	return m.processTerminals(ctx, sess.LoginID, func(sess *Session) []TerminalInfo {
		if info, ok := sess.removeTerminalByToken(tokenValue); ok {
			return []TerminalInfo{info}
		}
		return nil
	}, TokenStateKickOut)
}

// KickoutByDevice kicks out all terminals of a specific device type. KickoutByDevice 根据设备类型踢人下线（踢掉该设备类型的所有终端）。
func (m *Manager) KickoutByDevice(ctx context.Context, loginID string, device string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}
	device = strings.TrimSpace(device)
	if device == "" {
		return derror.ErrInvalidParam
	}

	return m.processTerminals(ctx, loginID, func(sess *Session) []TerminalInfo {
		return sess.removeTerminalByDevice(device)
	}, TokenStateKickOut)
}

// KickoutByDeviceAndDeviceId kicks out a user by device type and device ID. KickoutByDeviceAndDeviceId 根据设备类型和设备ID踢人下线。
func (m *Manager) KickoutByDeviceAndDeviceId(ctx context.Context, loginID string, deviceAndDeviceId ...string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	device, deviceId := m.getDeviceAndDeviceId(deviceAndDeviceId...)
	if device == "" || deviceId == "" {
		return derror.ErrInvalidParam
	}

	return m.processTerminals(ctx, loginID, func(sess *Session) []TerminalInfo {
		return sess.removeTerminalByDeviceAndDeviceId(device, deviceId)
	}, TokenStateKickOut)
}

// KickoutByLoginID kicks out all terminals for the specified loginID. KickoutByLoginID 踢出指定 loginID 的所有终端。
func (m *Manager) KickoutByLoginID(ctx context.Context, loginID string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	return m.processTerminals(ctx, loginID, func(sess *Session) []TerminalInfo {
		return sess.removeAllTerminals()
	}, TokenStateKickOut)
}

// Replace replaces a user session by token. Replace 根据 Token 顶人下线。
func (m *Manager) Replace(ctx context.Context, tokenValue string) error {
	sess, err := m.GetSessionByToken(ctx, tokenValue)
	if err != nil {
		return err
	}

	return m.processTerminals(ctx, sess.LoginID, func(sess *Session) []TerminalInfo {
		if info, ok := sess.removeTerminalByToken(tokenValue); ok {
			return []TerminalInfo{info}
		}
		return nil
	}, TokenStateReplaced)
}

// ReplaceByDevice replaces all terminals of a specific device type. ReplaceByDevice 根据设备类型顶人下线（顶掉该设备类型的所有终端）。
func (m *Manager) ReplaceByDevice(ctx context.Context, loginID string, device string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}
	device = strings.TrimSpace(device)
	if device == "" {
		return derror.ErrInvalidParam
	}

	return m.processTerminals(ctx, loginID, func(sess *Session) []TerminalInfo {
		return sess.removeTerminalByDevice(device)
	}, TokenStateReplaced)
}

// ReplaceByDeviceAndDeviceId replaces a user session by device type and device ID. ReplaceByDeviceAndDeviceId 根据设备类型和设备ID顶人下线。
func (m *Manager) ReplaceByDeviceAndDeviceId(ctx context.Context, loginID string, deviceAndDeviceId ...string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	device, deviceId := m.getDeviceAndDeviceId(deviceAndDeviceId...)
	if device == "" || deviceId == "" {
		return derror.ErrInvalidParam
	}
	return m.processTerminals(ctx, loginID, func(sess *Session) []TerminalInfo {
		return sess.removeTerminalByDeviceAndDeviceId(device, deviceId)
	}, TokenStateReplaced)
}

// ReplaceByLoginID replaces all terminals for the specified loginID. ReplaceByLoginID 顶替指定 loginID 的所有终端。
func (m *Manager) ReplaceByLoginID(ctx context.Context, loginID string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	return m.processTerminals(ctx, loginID, func(sess *Session) []TerminalInfo {
		return sess.removeAllTerminals()
	}, TokenStateReplaced)
}

// IsLogin checks if a user is logged in. IsLogin 检查用户是否登录。
func (m *Manager) IsLogin(ctx context.Context, tokenValue string) bool {
	return m.checkLoginInternal(ctx, tokenValue) == nil
}

// CheckLogin checks if a user is logged in and returns an error if not. CheckLogin 检查用户是否登录，如果未登录则返回错误。
func (m *Manager) CheckLogin(ctx context.Context, tokenValue string) error {
	return m.checkLoginInternal(ctx, tokenValue)
}

// GetLoginID retrieves the login ID from a token. GetLoginID 根据 Token 获取登录 ID。
func (m *Manager) GetLoginID(ctx context.Context, tokenValue string) (string, error) {
	// Get checked token 获取已校验 Token
	_, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		return "", err
	}

	return tokenInfo.LoginID, nil
}

// GetTokenInfo retrieves token information. GetTokenInfo 根据 Token 获取 TokenInfo 信息。
func (m *Manager) GetTokenInfo(ctx context.Context, tokenValue string) (*TokenInfo, error) {
	return m.getTokenInfo(ctx, tokenValue)
}

// GetDevice retrieves the device type for a token. GetDevice 获取 Token 的设备类型。
func (m *Manager) GetDevice(ctx context.Context, tokenValue string) (string, error) {
	_, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		return "", err
	}
	return tokenInfo.Device, nil
}

// GetDeviceId retrieves the device ID for a token. GetDeviceId 获取 Token 的设备 ID。
func (m *Manager) GetDeviceId(ctx context.Context, tokenValue string) (string, error) {
	_, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		return "", err
	}
	return tokenInfo.DeviceId, nil
}

// GetTokenCreateTime retrieves the creation time for a token. GetTokenCreateTime 获取 Token 的创建时间戳。
func (m *Manager) GetTokenCreateTime(ctx context.Context, tokenValue string) (int64, error) {
	_, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		return 0, err
	}
	return tokenInfo.CreateTime, nil
}

// GetTokenTTL retrieves the remaining time-to-live for a token in seconds. GetTokenTTL 获取 Token 的剩余有效时间（秒）。
func (m *Manager) GetTokenTTL(ctx context.Context, tokenValue string) (int64, error) {
	ttl, err := m.storage.TTL(ctx, m.getTokenKey(tokenValue))
	if err != nil {
		return 0, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	if ttl == time.Duration(-2) {
		ttl, err = m.storage.TTL(ctx, m.getLegacyTokenKey(tokenValue))
		if err != nil {
			return 0, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
	}

	// Normalize TTL sentinel values 统一 TTL 哨兵值语义
	seconds := int64(ttl)
	switch {
	case seconds == -2:
		return -2, nil
	case seconds == -1:
		return -1, nil
	case seconds > 0:
		return int64(ttl.Seconds()), nil
	default:
		return 0, nil
	}
}

// Disable disables an account for a specified duration. Disable 封禁账号指定时长。
func (m *Manager) Disable(ctx context.Context, loginID string, duration time.Duration, reason ...string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	unlock := m.lockLoginWrite(loginID)
	defer unlock()

	// Load session before disable 先尝试加载 Session（如果存储出错，在保存封禁信息前就返回，保证原子性）
	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		// Ignore missing session and return other storage errors 如果只是 session 不存在，不算错误；其他存储错误则返回
		if !errors.Is(err, derror.ErrSessionNotFound) {
			return err
		}
		// Continue disable when sess is nil 否则 sess == nil，继续执行封禁操作（幂等）
	}

	// Build and save disable info 构建并保存封禁信息
	disableInfo := DisableInfo{
		DisableTime: time.Now().Unix(),
	}
	if len(reason) > 0 && reason[0] != "" {
		disableInfo.DisableReason = reason[0]
	}

	if err = m.saveToStorage(ctx, m.getDisableKey(loginID), disableInfo, duration); err != nil {
		return err
	}

	// Delete session 删除 Session
	if err = m.storage.Delete(ctx, m.getSessionKey(loginID)); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	// Clean related token data when terminals exist 如果有终端信息，清理所有相关 token 数据
	if sess != nil && len(sess.TerminalInfos) > 0 {
		tokens := make([]string, len(sess.TerminalInfos))
		tokenKeys := make([]string, 0, len(sess.TerminalInfos)*2)
		for i, info := range sess.TerminalInfos {
			tokens[i] = info.Token
			tokenKeys = append(tokenKeys, m.getTokenStorageKeys(info.Token)...)
		}

		// Delete primary token keys 删除主 token keys
		if err = m.storage.Delete(ctx, tokenKeys...); err != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}

		// Clean token metadata 清理附属 metadata（续期、活跃时间）
		if err = m.cleanTokenMetadata(ctx, tokens); err != nil {
			return err
		}
	}

	unlock()
	unlock = func() {}

	if sess != nil {
		// Trigger session destroy event 触发销毁 Session 事件
		m.triggerEvent(listener.EventDestroySession, loginID, "", "", "", nil)
	}

	// Trigger disable event 触发封禁事件
	m.triggerEvent(listener.EventDisable, loginID, "", "", "", map[string]any{
		"reason":   disableInfo.DisableReason,
		"duration": duration.Seconds(),
	})

	return nil
}

// Untie removes the disable status from an account. Untie 解封账号。
func (m *Manager) Untie(ctx context.Context, loginID string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	if err := m.storage.Delete(ctx, m.getDisableKey(loginID)); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	// Trigger untie event 触发解禁事件
	m.triggerEvent(listener.EventUntie, loginID, "", "", "", nil)

	return nil
}

// IsDisable checks if an account is disabled. IsDisable 检查账号是否被封禁。
func (m *Manager) IsDisable(ctx context.Context, loginID string) bool {
	return m.isDisable(ctx, loginID)
}

// GetDisableInfo retrieves disable information for an account. GetDisableInfo 获取账号的封禁信息。
func (m *Manager) GetDisableInfo(ctx context.Context, loginID string) (*DisableInfo, error) {
	disableInfoData, err := m.storage.Get(ctx, m.getDisableKey(loginID))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	// Return explicit error when disable key is missing 如果 key 不存在（用户未被封禁），返回明确的错误
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

// GetDisableTTL retrieves the remaining disable time for an account in seconds. GetDisableTTL 获取账号剩余封禁时间（秒）。 Returns: -2: account is not disabled (未封禁) -1: account is permanently disabled (永久封禁) >0: remaining seconds until unban (剩余封禁秒数)
func (m *Manager) GetDisableTTL(ctx context.Context, loginID string) (int64, error) {
	ttl, err := m.storage.TTL(ctx, m.getDisableKey(loginID))
	if err != nil {
		return 0, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	// Explain TTL semantics 存储适配器返回 time.Duration 类型，直接转换为 int64 即可，标准 Redis TTL 语义：-2 key 不存在，-1 key 无过期时间，>0 剩余秒数
	seconds := int64(ttl)

	switch {
	case seconds == -2:
		return -2, nil // 未封禁
	case seconds == -1:
		return -1, nil // 永久封禁
	case seconds > 0:
		return int64(ttl.Seconds()), nil
	default:
		return 0, nil
	}
}

// DisableService disables a specific service for an account. DisableService 封禁账号的指定服务。
func (m *Manager) DisableService(ctx context.Context, loginID, service string, duration time.Duration, reason ...string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}
	if service == "" {
		return derror.ErrInvalidParam
	}

	info := ServiceDisableInfo{
		Service:     service,
		Level:       0,
		DisableTime: time.Now().Unix(),
	}
	if len(reason) > 0 && reason[0] != "" {
		info.DisableReason = reason[0]
	}

	if err := m.saveToStorage(ctx, m.getDisableServiceKey(loginID, service), info, duration); err != nil {
		return err
	}

	m.triggerEvent(listener.EventDisableService, loginID, "", "", "", map[string]any{
		listener.ExtraKeyService: service,
		"reason":                 info.DisableReason,
		"duration":               duration.Seconds(),
	})

	return nil
}

// DisableServiceLevel disables a specific service for an account with a level. DisableServiceLevel 封禁账号的指定服务并设置封禁等级。
func (m *Manager) DisableServiceLevel(ctx context.Context, loginID, service string, level int, duration time.Duration, reason ...string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}
	if service == "" {
		return derror.ErrInvalidParam
	}

	info := ServiceDisableInfo{
		Service:     service,
		Level:       level,
		DisableTime: time.Now().Unix(),
	}
	if len(reason) > 0 && reason[0] != "" {
		info.DisableReason = reason[0]
	}

	if err := m.saveToStorage(ctx, m.getDisableServiceKey(loginID, service), info, duration); err != nil {
		return err
	}

	m.triggerEvent(listener.EventDisableService, loginID, "", "", "", map[string]any{
		listener.ExtraKeyService: service,
		listener.ExtraKeyLevel:   level,
		"reason":                 info.DisableReason,
		"duration":               duration.Seconds(),
	})

	return nil
}

// UntieService removes the disable status of a specific service for an account. UntieService 解封账号的指定服务。
func (m *Manager) UntieService(ctx context.Context, loginID, service string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}
	if service == "" {
		return derror.ErrInvalidParam
	}

	if err := m.storage.Delete(ctx, m.getDisableServiceKey(loginID, service)); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	m.triggerEvent(listener.EventUntieService, loginID, "", "", "", map[string]any{
		listener.ExtraKeyService: service,
	})

	return nil
}

// IsDisableService checks if a specific service is disabled for an account. IsDisableService 检查账号的指定服务是否被封禁。
func (m *Manager) IsDisableService(ctx context.Context, loginID, service string) bool {
	if loginID == "" || service == "" {
		return false
	}
	return m.storage.Exists(ctx, m.getDisableServiceKey(loginID, service))
}

// IsDisableServiceLevel checks if a specific service is disabled at or above the given level. IsDisableServiceLevel 检查账号的指定服务是否达到指定封禁等级。
func (m *Manager) IsDisableServiceLevel(ctx context.Context, loginID, service string, level int) bool {
	info, err := m.GetDisableServiceInfo(ctx, loginID, service)
	if err != nil {
		return false
	}
	return info.Level >= level
}

// CheckDisableService checks if any of the specified services are disabled, returns error if disabled. CheckDisableService 校验账号的指定服务是否被封禁，被封禁则返回 error。
func (m *Manager) CheckDisableService(ctx context.Context, loginID string, services ...string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}
	for _, service := range services {
		if m.IsDisableService(ctx, loginID, service) {
			return fmt.Errorf("%w: service=%s", derror.ErrServiceDisabled, service)
		}
	}
	return nil
}

// CheckDisableServiceLevel checks if a service is disabled at or above the given level, returns error if so. CheckDisableServiceLevel 校验账号的指定服务是否达到指定封禁等级，达到则返回 error。
func (m *Manager) CheckDisableServiceLevel(ctx context.Context, loginID, service string, level int) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}
	if m.IsDisableServiceLevel(ctx, loginID, service, level) {
		return fmt.Errorf("%w: service=%s, level=%d", derror.ErrServiceDisabled, service, level)
	}
	return nil
}

// GetDisableServiceInfo retrieves the disable info for a specific service. GetDisableServiceInfo 获取账号指定服务的封禁信息。
func (m *Manager) GetDisableServiceInfo(ctx context.Context, loginID, service string) (*ServiceDisableInfo, error) {
	if loginID == "" {
		return nil, derror.ErrIDIsEmpty
	}
	if service == "" {
		return nil, derror.ErrInvalidParam
	}

	data, err := m.storage.Get(ctx, m.getDisableServiceKey(loginID, service))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	if data == nil {
		return nil, derror.ErrServiceNotDisabled
	}

	bytesData, err := utils.ToBytes(data)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrTypeConvert, err)
	}

	var info ServiceDisableInfo
	if err = m.serializer.Decode(bytesData, &info); err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}

	return &info, nil
}

// GetDisableServiceTTL retrieves the remaining disable time for a specific service in seconds. GetDisableServiceTTL 获取账号指定服务的剩余封禁时间（秒）。
func (m *Manager) GetDisableServiceTTL(ctx context.Context, loginID, service string) (int64, error) {
	ttl, err := m.storage.TTL(ctx, m.getDisableServiceKey(loginID, service))
	if err != nil {
		return 0, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	seconds := int64(ttl)
	switch {
	case seconds == -2:
		return -2, nil
	case seconds == -1:
		return -1, nil
	default:
		return int64(ttl.Seconds()), nil
	}
}

// GetSession retrieves session information for a login ID. GetSession 获取指定登录 ID 的会话信息。
func (m *Manager) GetSession(ctx context.Context, loginID string) (*Session, error) {
	if loginID == "" {
		return nil, derror.ErrIDIsEmpty
	}
	return m.getSession(ctx, loginID)
}

// GetSessionByToken retrieves session information by token. GetSessionByToken 通过 Token 值获取会话信息。
func (m *Manager) GetSessionByToken(ctx context.Context, tokenValue string) (*Session, error) {
	// Get tokenInfo 获取 tokenInfo
	tokenInfo, err := m.getTokenInfo(ctx, tokenValue)
	if err != nil {
		return nil, err
	}

	sess, err := m.getSession(ctx, tokenInfo.LoginID)
	if err != nil {
		if errors.Is(err, derror.ErrSessionNotFound) {
			return nil, derror.ErrInvalidToken
		}
		return nil, err
	}
	if sess == nil || !sess.hasTerminalToken(tokenValue) {
		return nil, derror.ErrInvalidToken
	}

	return sess, nil
}

// GetTokenValueListByLoginID retrieves all tokens for a login ID. GetTokenValueListByLoginID 获取指定登录 ID 的所有 Token。
func (m *Manager) GetTokenValueListByLoginID(ctx context.Context, loginID string, checkAlive ...bool) ([]string, error) {
	if loginID == "" {
		return nil, derror.ErrIDIsEmpty
	}

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		// Return errors only for real storage failures 仅当存储层真正出错时才返回 error；session 不存在视为 nil
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

// GetTokenValueListByDevice retrieves all tokens for a specific device type. GetTokenValueListByDevice 获取指定设备类型的所有 Token。
func (m *Manager) GetTokenValueListByDevice(ctx context.Context, loginID, device string, checkAlive ...bool) ([]string, error) {
	if loginID == "" {
		return []string{}, derror.ErrIDIsEmpty
	}
	device = strings.TrimSpace(device)
	if device == "" {
		return []string{}, derror.ErrInvalidParam
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

// GetTokenValueListByDeviceAndDeviceId retrieves all tokens for a specific device type and device ID. GetTokenValueListByDeviceAndDeviceId 获取指定设备类型和设备 ID 的所有 Token。
func (m *Manager) GetTokenValueListByDeviceAndDeviceId(ctx context.Context, loginID, device, deviceId string, checkAlive ...bool) ([]string, error) {
	if loginID == "" {
		return []string{}, derror.ErrIDIsEmpty
	}
	device = strings.TrimSpace(device)
	deviceId = strings.TrimSpace(deviceId)
	if device == "" || deviceId == "" {
		return []string{}, derror.ErrInvalidParam
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

	matched := sess.getTerminalsByDeviceAndDeviceId(device, deviceId)
	return m.filterTokens(ctx, matched, len(checkAlive) > 0 && checkAlive[0])
}

// GetOnlineTerminalCount retrieves the count of online terminals for a user. GetOnlineTerminalCount 获取用户的在线终端数量。
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

// GetOnlineTerminalCountByDevice retrieves the count of online terminals for a specific device type. GetOnlineTerminalCountByDevice 获取用户在指定设备类型的在线终端数量。
func (m *Manager) GetOnlineTerminalCountByDevice(ctx context.Context, loginID, device string) (int, error) {
	tokens, err := m.GetTokenValueListByDevice(ctx, loginID, device, true)
	if err != nil {
		return 0, err
	}
	return len(tokens), nil
}

// GetOnlineTerminalCountByDeviceAndDeviceId retrieves the count of online terminals for a specific device type and device ID. GetOnlineTerminalCountByDeviceAndDeviceId 获取用户在指定设备类型和设备ID的在线终端数量。
func (m *Manager) GetOnlineTerminalCountByDeviceAndDeviceId(ctx context.Context, loginID, device, deviceId string) (int, error) {
	tokens, err := m.GetTokenValueListByDeviceAndDeviceId(ctx, loginID, device, deviceId, true)
	if err != nil {
		return 0, err
	}
	return len(tokens), nil
}

// GetTerminalListByLoginID retrieves all terminal info for a login ID, optionally filtered by device. GetTerminalListByLoginID 获取指定登录 ID 的所有终端信息列表，可选按设备类型过滤。
func (m *Manager) GetTerminalListByLoginID(ctx context.Context, loginID string, device ...string) ([]TerminalInfo, error) {
	if loginID == "" {
		return nil, derror.ErrIDIsEmpty
	}

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		if errors.Is(err, derror.ErrSessionNotFound) {
			return []TerminalInfo{}, nil
		}
		return nil, err
	}
	if sess == nil {
		return []TerminalInfo{}, nil
	}

	if len(device) > 0 && device[0] != "" {
		return sess.getTerminalsByDevice(device[0]), nil
	}

	// Return copy to avoid external mutation 返回副本，避免外部修改影响内部数据
	result := make([]TerminalInfo, len(sess.TerminalInfos))
	copy(result, sess.TerminalInfos)
	return result, nil
}

// GetTerminalInfoByToken retrieves terminal info for a specific token. GetTerminalInfoByToken 根据 Token 获取终端详情。
func (m *Manager) GetTerminalInfoByToken(ctx context.Context, tokenValue string) (*TerminalInfo, error) {
	if tokenValue == "" {
		return nil, derror.ErrInvalidToken
	}

	sess, _, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		return nil, err
	}

	for _, ti := range sess.TerminalInfos {
		if ti.Token == tokenValue {
			return &ti, nil
		}
	}

	return nil, derror.ErrInvalidToken
}

// GetTokenValueByLoginID retrieves the latest token for a login ID, optionally filtered by device. GetTokenValueByLoginID 获取指定登录 ID 的最新 Token，可选按设备类型过滤。
func (m *Manager) GetTokenValueByLoginID(ctx context.Context, loginID string, device ...string) (string, error) {
	if loginID == "" {
		return "", derror.ErrIDIsEmpty
	}

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		return "", err
	}

	if len(device) > 0 && device[0] != "" {
		if ti, ok := sess.getLatestTerminalByDevice(device[0]); ok {
			return ti.Token, nil
		}
		return "", derror.ErrInvalidToken
	}

	// Return latest token 返回最后一个（最新的）
	if len(sess.TerminalInfos) == 0 {
		return "", derror.ErrInvalidToken
	}
	return sess.TerminalInfos[len(sess.TerminalInfos)-1].Token, nil
}

// SearchTokenValue searches token values by keyword with pagination. SearchTokenValue 根据关键词搜索 Token 值，支持分页。 keyword: 搜索关键词（模糊匹配），start: 起始索引，size: 返回数量（-1 返回全部）
func (m *Manager) SearchTokenValue(ctx context.Context, keyword string, start, size int) ([]string, error) {
	prefix := m.config.KeyPrefix + m.config.AuthType + config.TokenKeyPrefix
	pattern := prefix + "*" + keyword + "*"
	return m.searchValues(ctx, pattern, prefix, start, size)
}

// SearchSessionId searches session IDs by keyword with pagination. SearchSessionId 根据关键词搜索 Session ID，支持分页。 keyword: 搜索关键词（模糊匹配），start: 起始索引，size: 返回数量（-1 返回全部）
func (m *Manager) SearchSessionId(ctx context.Context, keyword string, start, size int) ([]string, error) {
	prefix := m.config.KeyPrefix + m.config.AuthType + SessionKeyPrefix
	pattern := prefix + "*" + keyword + "*"
	return m.searchValues(ctx, pattern, prefix, start, size)
}

// TerminalVisitor is a callback function for terminal traversal. TerminalVisitor 终端遍历回调函数。 Return false to stop traversal. 返回 false 停止遍历。
type TerminalVisitor func(terminal TerminalInfo) bool

// ForEachTerminal iterates over all terminals for a login ID and calls the visitor function. ForEachTerminal 遍历指定登录 ID 的所有终端，对每个终端调用回调函数。
func (m *Manager) ForEachTerminal(ctx context.Context, loginID string, visitor TerminalVisitor) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}
	if visitor == nil {
		return derror.ErrInvalidParam
	}

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		if errors.Is(err, derror.ErrSessionNotFound) {
			return nil
		}
		return err
	}

	for _, ti := range sess.TerminalInfos {
		if !visitor(ti) {
			break
		}
	}
	return nil
}

// ForEachTerminalByDevice iterates over terminals filtered by device type. ForEachTerminalByDevice 遍历指定设备类型的终端。
func (m *Manager) ForEachTerminalByDevice(ctx context.Context, loginID, device string, visitor TerminalVisitor) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}
	device = strings.TrimSpace(device)
	if device == "" {
		return derror.ErrInvalidParam
	}
	if visitor == nil {
		return derror.ErrInvalidParam
	}

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		if errors.Is(err, derror.ErrSessionNotFound) {
			return nil
		}
		return err
	}

	for _, ti := range sess.TerminalInfos {
		if ti.Device == device {
			if !visitor(ti) {
				break
			}
		}
	}
	return nil
}

// RenewTimeout manually renews the timeout of a token. RenewTimeout 手动续期指定 Token 的过期时间。
func (m *Manager) RenewTimeout(ctx context.Context, tokenValue string, timeout time.Duration) error {
	if tokenValue == "" {
		return derror.ErrInvalidToken
	}

	tokenInfo, err := m.getTokenInfo(ctx, tokenValue)
	if err != nil {
		return err
	}

	unlock := m.lockLoginWrite(tokenInfo.LoginID)
	defer unlock()

	// Reload token after acquiring lock 加锁后重新读取 token，避免并发续期失效 token
	tokenInfo, err = m.getTokenInfo(ctx, tokenValue)
	if err != nil {
		return err
	}

	sess, err := m.getSession(ctx, tokenInfo.LoginID)
	if err != nil {
		if errors.Is(err, derror.ErrSessionNotFound) {
			return derror.ErrInvalidToken
		}
		return err
	}
	if sess == nil || !sess.hasTerminalToken(tokenValue) {
		return derror.ErrInvalidToken
	}
	if err = m.ensureTerminalTokenAlive(ctx, tokenValue); err != nil {
		return err
	}

	expiration := timeout
	if expiration <= 0 {
		expiration = 0
	}
	tokenInfo.Timeout = m.timeoutToSeconds(expiration)

	// Persist token with the new timeout 保存 Token 并记录新的有效期
	if err = m.saveToStorage(ctx, m.getTokenKey(tokenValue), *tokenInfo, expiration); err != nil {
		return err
	}

	// Renew session without shortening existing TTL 续期 Session，避免缩短已有 TTL
	if err = m.saveSessionWithMinTTL(ctx, m.getSessionKey(tokenInfo.LoginID), *sess, expiration); err != nil {
		return err
	}

	if m.config.ActiveTimeout > 0 {
		activeValue, activeErr := m.storage.Get(ctx, m.getActiveKey(tokenValue))
		if activeErr != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, activeErr)
		}
		if activeValue == nil {
			activeValue = time.Now().Unix()
		}
		if err = m.storage.Set(ctx, m.getActiveKey(tokenValue), activeValue, expiration); err != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
	}

	unlock()
	unlock = func() {}

	// Trigger renew event 触发续期事件
	m.triggerEvent(listener.EventRenew, tokenInfo.LoginID, tokenInfo.Device, tokenInfo.DeviceId, tokenValue, map[string]any{
		"timeout": timeout.Seconds(),
	})

	return nil
}

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
	err = m.saveToStorage(ctx, m.getSessionKey(loginID), *sess)
	if err != nil {
		return err
	}

	return nil
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

	// Add permissions 添加权限
	sess.addPermissions(permissions...)
	// Save session 保存 Session
	err = m.saveToStorage(ctx, m.getSessionKey(sess.LoginID), *sess)
	if err != nil {
		return err
	}

	return nil
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
	err = m.saveToStorage(ctx, m.getSessionKey(loginID), *sess)
	if err != nil {
		return err
	}

	return nil
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

	// Remove permissions 删除权限
	sess.removePermissions(permissions...)
	// Save session 保存 Session
	err = m.saveToStorage(ctx, m.getSessionKey(sess.LoginID), *sess)
	if err != nil {
		return err
	}

	return nil
}

// GetPermissions retrieves the permission list for a user. GetPermissions 获取用户的权限列表。
func (m *Manager) GetPermissions(ctx context.Context, loginID string) ([]string, error) {
	if loginID == "" {
		return nil, derror.ErrIDIsEmpty
	}

	// Use custom permission list function 使用自定义权限列表获取函数
	if m.CustomPermissionListFunc != nil {
		return m.CustomPermissionListFunc(loginID, m.config.AuthType)
	}

	// Get session 获取 Session
	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		return nil, err
	}

	return sess.Permissions, nil
}

// GetPermissionsByToken retrieves the permission list by token. GetPermissionsByToken 根据 Token 获取权限列表。
func (m *Manager) GetPermissionsByToken(ctx context.Context, tokenValue string) ([]string, error) {
	// Get checked session and token 获取已校验的 Session 和 TokenInfo
	sess, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		return nil, err
	}

	permissions := sess.Permissions
	if m.CustomPermissionListExtFunc != nil {
		customPerms, err := m.CustomPermissionListExtFunc(sess.LoginID, tokenInfo.Device, tokenInfo.DeviceId, m.config.AuthType)
		if err == nil && customPerms != nil {
			permissions = customPerms
		}
	} else if m.CustomPermissionListFunc != nil {
		customPerms, err := m.CustomPermissionListFunc(sess.LoginID, m.config.AuthType)
		if err == nil && customPerms != nil {
			permissions = customPerms
		}
	}

	return permissions, nil
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

	// Get permissions with Func then Session priority 获取权限列表（两级优先级：Func > Session）
	permissions := sess.Permissions
	if m.CustomPermissionListFunc != nil {
		customPerms, err := m.CustomPermissionListFunc(loginID, m.config.AuthType)
		if err == nil && customPerms != nil {
			permissions = customPerms
		}
	}

	hasPermission := false
	for _, p := range permissions {
		if m.matchPermission(p, permission) {
			hasPermission = true
			break
		}
	}

	// Trigger permission check event 触发权限检查事件
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
	// Get device and deviceId 获取 device/deviceId
	device, deviceId := tokenInfo.Device, tokenInfo.DeviceId

	// Get permissions with Ext Func Session priority 获取权限列表（三级优先级：Ext > Func > Session）
	permissions := sess.Permissions
	if m.CustomPermissionListExtFunc != nil {
		customPerms, err := m.CustomPermissionListExtFunc(sess.LoginID, device, deviceId, m.config.AuthType)
		if err == nil && customPerms != nil {
			permissions = customPerms
		}
	} else if m.CustomPermissionListFunc != nil {
		customPerms, err := m.CustomPermissionListFunc(sess.LoginID, m.config.AuthType)
		if err == nil && customPerms != nil {
			permissions = customPerms
		}
	}

	hasPermission := false
	for _, p := range permissions {
		if m.matchPermission(p, permission) {
			hasPermission = true
			break
		}
	}

	// Trigger permission check event 触发权限检查事件
	m.triggerEvent(listener.EventPermissionCheck, sess.LoginID, device, deviceId, tokenValue, map[string]any{
		listener.ExtraKeyPermission: permission,
		listener.ExtraKeyResult:     hasPermission,
	})

	return hasPermission
}

// HasPermissionsAnd checks if a user has all specified permissions (AND logic). HasPermissionsAnd 检查用户是否拥有所有指定权限（AND 逻辑）。
func (m *Manager) HasPermissionsAnd(ctx context.Context, loginID string, permissions []string) bool {
	if loginID == "" {
		return false
	}

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		m.logger.Errorf("manager.HasPermissionsAnd: failed to get session, loginID=%s, error=%v", loginID, err)
		return false
	}

	// Get permissions with Func then Session priority 获取权限列表（两级优先级：Func > Session）
	permList := sess.Permissions
	if m.CustomPermissionListFunc != nil {
		customPerms, err := m.CustomPermissionListFunc(loginID, m.config.AuthType)
		if err == nil && customPerms != nil {
			permList = customPerms
		}
	}

	// Check each required permission 校验每一个必需权限
	hasAll := true
	for _, need := range permissions {
		if !m.hasPermissionInList(permList, need) {
			hasAll = false
			break
		}
	}

	// Trigger permission check event 触发权限检查事件
	m.triggerEvent(listener.EventPermissionCheck, loginID, "", "", "", map[string]any{
		listener.ExtraKeyPermissions: permissions,
		listener.ExtraKeyLogic:       listener.LogicAnd,
		listener.ExtraKeyResult:      hasAll,
	})

	return hasAll
}

// HasPermissionsAndByToken checks if a user has all specified permissions by token (AND logic). HasPermissionsAndByToken 根据 Token 检查用户是否拥有所有指定权限（AND 逻辑）。
func (m *Manager) HasPermissionsAndByToken(ctx context.Context, tokenValue string, permissions []string) bool {
	sess, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		m.logger.Errorf("manager.HasPermissionsAndByToken: failed to get token info, token=%s, error=%v", tokenValue, err)
		return false
	}
	// Get device and deviceId 获取 device/deviceId
	device, deviceId := tokenInfo.Device, tokenInfo.DeviceId

	// Get permissions with Ext Func Session priority 获取权限列表（三级优先级：Ext > Func > Session）
	permList := sess.Permissions
	if m.CustomPermissionListExtFunc != nil {
		customPerms, err := m.CustomPermissionListExtFunc(sess.LoginID, device, deviceId, m.config.AuthType)
		if err == nil && customPerms != nil {
			permList = customPerms
		}
	} else if m.CustomPermissionListFunc != nil {
		customPerms, err := m.CustomPermissionListFunc(sess.LoginID, m.config.AuthType)
		if err == nil && customPerms != nil {
			permList = customPerms
		}
	}

	// Check each required permission 校验每一个必需权限
	hasAll := true
	for _, need := range permissions {
		if !m.hasPermissionInList(permList, need) {
			hasAll = false
			break
		}
	}

	// Trigger permission check event 触发权限检查事件
	m.triggerEvent(listener.EventPermissionCheck, sess.LoginID, device, deviceId, tokenValue, map[string]any{
		listener.ExtraKeyPermissions: permissions,
		listener.ExtraKeyLogic:       listener.LogicAnd,
		listener.ExtraKeyResult:      hasAll,
	})

	return hasAll
}

// HasPermissionsOr checks if a user has any of the specified permissions (OR logic). HasPermissionsOr 检查用户是否拥有任一指定权限（OR 逻辑）。
func (m *Manager) HasPermissionsOr(ctx context.Context, loginID string, permissions []string) bool {
	if loginID == "" {
		return false
	}

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		m.logger.Errorf("manager.HasPermissionsOr: failed to get session, loginID=%s, error=%v", loginID, err)
		return false
	}
	if len(permissions) == 0 {
		return true
	}

	// Get permissions with Func then Session priority 获取权限列表（两级优先级：Func > Session）
	permList := sess.Permissions
	if m.CustomPermissionListFunc != nil {
		customPerms, err := m.CustomPermissionListFunc(loginID, m.config.AuthType)
		if err == nil && customPerms != nil {
			permList = customPerms
		}
	}

	// Pass on any matching permission 任一权限匹配即通过
	hasAny := false
	for _, need := range permissions {
		if m.hasPermissionInList(permList, need) {
			hasAny = true
			break
		}
	}

	// Trigger permission check event 触发权限检查事件
	m.triggerEvent(listener.EventPermissionCheck, loginID, "", "", "", map[string]any{
		listener.ExtraKeyPermissions: permissions,
		listener.ExtraKeyLogic:       listener.LogicOr,
		listener.ExtraKeyResult:      hasAny,
	})

	return hasAny
}

// HasPermissionsOrByToken checks if a user has any of the specified permissions by token (OR logic). HasPermissionsOrByToken 根据 Token 检查用户是否拥有任一指定权限（OR 逻辑）。
func (m *Manager) HasPermissionsOrByToken(ctx context.Context, tokenValue string, permissions []string) bool {
	sess, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		m.logger.Errorf("manager.HasPermissionsOrByToken: failed to get token info, token=%s, error=%v", tokenValue, err)
		return false
	}
	if len(permissions) == 0 {
		return true
	}
	// Get device and deviceId 获取 device/deviceId
	device, deviceId := tokenInfo.Device, tokenInfo.DeviceId

	// Get permissions with Ext Func Session priority 获取权限列表（三级优先级：Ext > Func > Session）
	permList := sess.Permissions
	if m.CustomPermissionListExtFunc != nil {
		customPerms, err := m.CustomPermissionListExtFunc(sess.LoginID, device, deviceId, m.config.AuthType)
		if err == nil && customPerms != nil {
			permList = customPerms
		}
	} else if m.CustomPermissionListFunc != nil {
		customPerms, err := m.CustomPermissionListFunc(sess.LoginID, m.config.AuthType)
		if err == nil && customPerms != nil {
			permList = customPerms
		}
	}

	// Pass on any matching permission 任一权限匹配即通过
	hasAny := false
	for _, need := range permissions {
		if m.hasPermissionInList(permList, need) {
			hasAny = true
			break
		}
	}

	// Trigger permission check event 触发权限检查事件
	m.triggerEvent(listener.EventPermissionCheck, sess.LoginID, device, deviceId, tokenValue, map[string]any{
		listener.ExtraKeyPermissions: permissions,
		listener.ExtraKeyLogic:       listener.LogicOr,
		listener.ExtraKeyResult:      hasAny,
	})

	return hasAny
}

// AddRoles adds roles to a user. AddRoles 为用户添加角色。
func (m *Manager) AddRoles(ctx context.Context, loginID string, roles []string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	unlock := m.lockLoginWrite(loginID)
	defer unlock()

	// Get session 获取 Session
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

// AddRolesByToken adds roles to a user by token. AddRolesByToken 根据 Token 为用户添加角色。
func (m *Manager) AddRolesByToken(ctx context.Context, tokenValue string, roles []string) error {
	_, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		return err
	}

	unlock := m.lockLoginWrite(tokenInfo.LoginID)
	defer unlock()

	if err = m.ensureTerminalTokenAlive(ctx, tokenValue); err != nil {
		return err
	}
	// Get session 获取 Session
	sess, err := m.GetSessionByToken(ctx, tokenValue)
	if err != nil {
		return err
	}

	// Add roles 添加角色
	sess.addRoles(roles...)
	// Save session 保存 Session
	err = m.saveToStorage(ctx, m.getSessionKey(sess.LoginID), *sess)
	if err != nil {
		return err
	}

	return nil
}

// RemoveRoles removes roles from a user. RemoveRoles 删除用户的指定角色。
func (m *Manager) RemoveRoles(ctx context.Context, loginID string, roles []string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	unlock := m.lockLoginWrite(loginID)
	defer unlock()

	// Get session 获取 Session
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

// RemoveRolesByToken removes roles from a user by token. RemoveRolesByToken 根据 Token 删除用户的指定角色。
func (m *Manager) RemoveRolesByToken(ctx context.Context, tokenValue string, roles []string) error {
	_, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		return err
	}

	unlock := m.lockLoginWrite(tokenInfo.LoginID)
	defer unlock()

	if err = m.ensureTerminalTokenAlive(ctx, tokenValue); err != nil {
		return err
	}
	// Get session 获取 Session
	sess, err := m.GetSessionByToken(ctx, tokenValue)
	if err != nil {
		return err
	}

	// Remove roles 删除角色
	sess.removeRoles(roles...)
	// Save session 保存 Session
	err = m.saveToStorage(ctx, m.getSessionKey(sess.LoginID), *sess)
	if err != nil {
		return err
	}

	return nil
}

// GetRoles retrieves the role list for a user. GetRoles 获取用户的角色列表。
func (m *Manager) GetRoles(ctx context.Context, loginID string) ([]string, error) {
	if loginID == "" {
		return nil, derror.ErrIDIsEmpty
	}

	// Use custom role list function 使用自定义角色列表获取函数
	if m.CustomRoleListFunc != nil {
		return m.CustomRoleListFunc(loginID, m.config.AuthType)
	}

	// Get session 获取 Session
	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		return nil, err
	}

	return sess.Roles, nil
}

// GetRolesByToken retrieves the role list by token. GetRolesByToken 根据 Token 获取角色列表。
func (m *Manager) GetRolesByToken(ctx context.Context, tokenValue string) ([]string, error) {
	// Get checked session and token 获取已校验的 Session 和 TokenInfo
	sess, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		return nil, err
	}

	roles := sess.Roles
	if m.CustomRoleListExtFunc != nil {
		customRoles, err := m.CustomRoleListExtFunc(sess.LoginID, tokenInfo.Device, tokenInfo.DeviceId, m.config.AuthType)
		if err == nil && customRoles != nil {
			roles = customRoles
		}
	} else if m.CustomRoleListFunc != nil {
		customRoles, err := m.CustomRoleListFunc(sess.LoginID, m.config.AuthType)
		if err == nil && customRoles != nil {
			roles = customRoles
		}
	}

	return roles, nil
}

// HasRole checks if a user has a specific role. HasRole 检查用户是否拥有指定角色。
func (m *Manager) HasRole(ctx context.Context, loginID string, role string) bool {
	if loginID == "" {
		return false
	}

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		m.logger.Errorf("manager.HasRole: failed to get session, loginID=%s, error=%v", loginID, err)
		return false
	}

	// Get roles with Func then Session priority 获取角色列表（两级优先级：Func > Session）
	roles := sess.Roles
	if m.CustomRoleListFunc != nil {
		customRoles, err := m.CustomRoleListFunc(loginID, m.config.AuthType)
		if err == nil && customRoles != nil {
			roles = customRoles
		}
	}

	hasRole := false
	for _, r := range roles {
		if r == role {
			hasRole = true
			break
		}
	}

	// Trigger role check event 触发角色检查事件
	m.triggerEvent(listener.EventRoleCheck, loginID, "", "", "", map[string]any{
		listener.ExtraKeyRole:   role,
		listener.ExtraKeyResult: hasRole,
	})

	return hasRole
}

// HasRoleByToken checks if a user has a specific role by token. HasRoleByToken 根据 Token 检查用户是否拥有指定角色。
func (m *Manager) HasRoleByToken(ctx context.Context, tokenValue string, role string) bool {
	sess, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		m.logger.Errorf("manager.HasRoleByToken: failed to get token info, token=%s, error=%v", tokenValue, err)
		return false
	}
	// Get device and deviceId 获取 device/deviceId
	device, deviceId := tokenInfo.Device, tokenInfo.DeviceId

	// Get roles with Ext Func Session priority 获取角色列表（三级优先级：Ext > Func > Session）
	roles := sess.Roles
	if m.CustomRoleListExtFunc != nil {
		customRoles, err := m.CustomRoleListExtFunc(sess.LoginID, device, deviceId, m.config.AuthType)
		if err == nil && customRoles != nil {
			roles = customRoles
		}
	} else if m.CustomRoleListFunc != nil {
		customRoles, err := m.CustomRoleListFunc(sess.LoginID, m.config.AuthType)
		if err == nil && customRoles != nil {
			roles = customRoles
		}
	}

	hasRole := false
	for _, r := range roles {
		if r == role {
			hasRole = true
			break
		}
	}

	// Trigger role check event 触发角色检查事件
	m.triggerEvent(listener.EventRoleCheck, sess.LoginID, device, deviceId, tokenValue, map[string]any{
		listener.ExtraKeyRole:   role,
		listener.ExtraKeyResult: hasRole,
	})

	return hasRole
}

// HasRolesAnd checks if a user has all specified roles (AND logic). HasRolesAnd 检查用户是否拥有所有指定角色（AND 逻辑）。
func (m *Manager) HasRolesAnd(ctx context.Context, loginID string, roles []string) bool {
	if loginID == "" {
		return false
	}

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		m.logger.Errorf("manager.HasRolesAnd: failed to get session, loginID=%s, error=%v", loginID, err)
		return false
	}

	// Get roles with Func then Session priority 获取角色列表（两级优先级：Func > Session）
	roleList := sess.Roles
	if m.CustomRoleListFunc != nil {
		customRoles, err := m.CustomRoleListFunc(loginID, m.config.AuthType)
		if err == nil && customRoles != nil {
			roleList = customRoles
		}
	}

	// Check each required role 校验每一个必需角色
	hasAll := true
	for _, need := range roles {
		found := false
		for _, r := range roleList {
			if r == need {
				found = true
				break
			}
		}
		if !found {
			hasAll = false
			break
		}
	}

	// Trigger role check event 触发角色检查事件
	m.triggerEvent(listener.EventRoleCheck, loginID, "", "", "", map[string]any{
		listener.ExtraKeyRoles:  roles,
		listener.ExtraKeyLogic:  listener.LogicAnd,
		listener.ExtraKeyResult: hasAll,
	})

	return hasAll
}

// HasRolesAndByToken checks if a user has all specified roles by token (AND logic). HasRolesAndByToken 根据 Token 检查用户是否拥有所有指定角色（AND 逻辑）。
func (m *Manager) HasRolesAndByToken(ctx context.Context, tokenValue string, roles []string) bool {
	sess, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		m.logger.Errorf("manager.HasRolesAndByToken: failed to get token info, token=%s, error=%v", tokenValue, err)
		return false
	}
	// Get device and deviceId 获取 device/deviceId
	device, deviceId := tokenInfo.Device, tokenInfo.DeviceId

	// Get roles with Ext Func Session priority 获取角色列表（三级优先级：Ext > Func > Session）
	roleList := sess.Roles
	if m.CustomRoleListExtFunc != nil {
		customRoles, err := m.CustomRoleListExtFunc(sess.LoginID, device, deviceId, m.config.AuthType)
		if err == nil && customRoles != nil {
			roleList = customRoles
		}
	} else if m.CustomRoleListFunc != nil {
		customRoles, err := m.CustomRoleListFunc(sess.LoginID, m.config.AuthType)
		if err == nil && customRoles != nil {
			roleList = customRoles
		}
	}

	// Check each required role 校验每一个必需角色
	hasAll := true
	for _, need := range roles {
		found := false
		for _, r := range roleList {
			if r == need {
				found = true
				break
			}
		}
		if !found {
			hasAll = false
			break
		}
	}

	// Trigger role check event 触发角色检查事件
	m.triggerEvent(listener.EventRoleCheck, sess.LoginID, device, deviceId, tokenValue, map[string]any{
		listener.ExtraKeyRoles:  roles,
		listener.ExtraKeyLogic:  listener.LogicAnd,
		listener.ExtraKeyResult: hasAll,
	})

	return hasAll
}

// HasRolesOr checks if a user has any of the specified roles (OR logic). HasRolesOr 检查用户是否拥有任一指定角色（OR 逻辑）。
func (m *Manager) HasRolesOr(ctx context.Context, loginID string, roles []string) bool {
	if loginID == "" {
		return false
	}

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		m.logger.Errorf("manager.HasRolesOr: failed to get session, loginID=%s, error=%v", loginID, err)
		return false
	}
	if len(roles) == 0 {
		return true
	}

	// Get roles with Func then Session priority 获取角色列表（两级优先级：Func > Session）
	roleList := sess.Roles
	if m.CustomRoleListFunc != nil {
		customRoles, err := m.CustomRoleListFunc(loginID, m.config.AuthType)
		if err == nil && customRoles != nil {
			roleList = customRoles
		}
	}

	// Pass on any matching role 任一角色匹配即通过
	hasAny := false
	for _, need := range roles {
		for _, r := range roleList {
			if r == need {
				hasAny = true
				break
			}
		}
		if hasAny {
			break
		}
	}

	// Trigger role check event 触发角色检查事件
	m.triggerEvent(listener.EventRoleCheck, loginID, "", "", "", map[string]any{
		listener.ExtraKeyRoles:  roles,
		listener.ExtraKeyLogic:  listener.LogicOr,
		listener.ExtraKeyResult: hasAny,
	})

	return hasAny
}

// HasRolesOrByToken checks if a user has any of the specified roles by token (OR logic). HasRolesOrByToken 根据 Token 检查用户是否拥有任一指定角色（OR 逻辑）。
func (m *Manager) HasRolesOrByToken(ctx context.Context, tokenValue string, roles []string) bool {
	sess, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		m.logger.Errorf("manager.HasRolesOrByToken: failed to get token info, token=%s, error=%v", tokenValue, err)
		return false
	}
	if len(roles) == 0 {
		return true
	}
	// Get device and deviceId 获取 device/deviceId
	device, deviceId := tokenInfo.Device, tokenInfo.DeviceId

	// Get roles with Ext Func Session priority 获取角色列表（三级优先级：Ext > Func > Session）
	roleList := sess.Roles
	if m.CustomRoleListExtFunc != nil {
		customRoles, err := m.CustomRoleListExtFunc(sess.LoginID, device, deviceId, m.config.AuthType)
		if err == nil && customRoles != nil {
			roleList = customRoles
		}
	} else if m.CustomRoleListFunc != nil {
		customRoles, err := m.CustomRoleListFunc(sess.LoginID, m.config.AuthType)
		if err == nil && customRoles != nil {
			roleList = customRoles
		}
	}

	// Pass on any matching role 任一角色匹配即通过
	hasAny := false
	for _, need := range roles {
		for _, r := range roleList {
			if r == need {
				hasAny = true
				break
			}
		}
		if hasAny {
			break
		}
	}

	// Trigger role check event 触发角色检查事件
	m.triggerEvent(listener.EventRoleCheck, sess.LoginID, device, deviceId, tokenValue, map[string]any{
		listener.ExtraKeyRoles:  roles,
		listener.ExtraKeyLogic:  listener.LogicOr,
		listener.ExtraKeyResult: hasAny,
	})

	return hasAny
}

// CheckPermission checks if a user has a specific permission, returns error if not. CheckPermission 校验用户是否拥有指定权限，无权限返回 error。
func (m *Manager) CheckPermission(ctx context.Context, loginID string, permission string) error {
	if !m.HasPermission(ctx, loginID, permission) {
		return fmt.Errorf("%w: %s", derror.ErrPermissionDenied, permission)
	}
	return nil
}

// CheckPermissionAnd checks if a user has all specified permissions, returns error if not. CheckPermissionAnd 校验用户是否拥有所有指定权限，缺少任一权限返回 error。
func (m *Manager) CheckPermissionAnd(ctx context.Context, loginID string, permissions []string) error {
	if !m.HasPermissionsAnd(ctx, loginID, permissions) {
		return derror.ErrPermissionDenied
	}
	return nil
}

// CheckPermissionOr checks if a user has any of the specified permissions, returns error if none. CheckPermissionOr 校验用户是否拥有任一指定权限，全部缺少返回 error。
func (m *Manager) CheckPermissionOr(ctx context.Context, loginID string, permissions []string) error {
	if !m.HasPermissionsOr(ctx, loginID, permissions) {
		return derror.ErrPermissionDenied
	}
	return nil
}

// CheckRole checks if a user has a specific role, returns error if not. CheckRole 校验用户是否拥有指定角色，无角色返回 error。
func (m *Manager) CheckRole(ctx context.Context, loginID string, role string) error {
	if !m.HasRole(ctx, loginID, role) {
		return fmt.Errorf("%w: %s", derror.ErrRoleDenied, role)
	}
	return nil
}

// CheckRoleAnd checks if a user has all specified roles, returns error if not. CheckRoleAnd 校验用户是否拥有所有指定角色，缺少任一角色返回 error。
func (m *Manager) CheckRoleAnd(ctx context.Context, loginID string, roles []string) error {
	if !m.HasRolesAnd(ctx, loginID, roles) {
		return derror.ErrRoleDenied
	}
	return nil
}

// CheckRoleOr checks if a user has any of the specified roles, returns error if none. CheckRoleOr 校验用户是否拥有任一指定角色，全部缺少返回 error。
func (m *Manager) CheckRoleOr(ctx context.Context, loginID string, roles []string) error {
	if !m.HasRolesOr(ctx, loginID, roles) {
		return derror.ErrRoleDenied
	}
	return nil
}

// CheckDisable checks if an account is disabled, returns error if disabled. CheckDisable 校验账号是否被封禁，被封禁返回 error。
func (m *Manager) CheckDisable(ctx context.Context, loginID string) error {
	if m.IsDisable(ctx, loginID) {
		return derror.ErrAccountDisabled
	}
	return nil
}

// GetConfig retrieves the manager configuration. GetConfig 获取管理器配置。
func (m *Manager) GetConfig() *config.Config {
	return m.config
}

// GetGenerator retrieves the token generator. GetGenerator 获取 Token 生成器。
func (m *Manager) GetGenerator() adapter.Generator {
	return m.generator
}

// GetStorage retrieves the storage adapter. GetStorage 获取存储适配器。
func (m *Manager) GetStorage() adapter.Storage {
	return m.storage
}

// GetSerializer retrieves the serializer adapter. GetSerializer 获取序列化器适配器。
func (m *Manager) GetSerializer() adapter.Codec {
	return m.serializer
}

// GetLogger retrieves the logger adapter. GetLogger 获取日志适配器。
func (m *Manager) GetLogger() adapter.Log {
	return m.logger
}

// GetPool retrieves the goroutine pool. GetPool 获取协程池。
func (m *Manager) GetPool() adapter.Pool {
	return m.pool
}

// GetCustomPermissionListFunc retrieves the custom permission list function. GetCustomPermissionListFunc 获取自定义权限列表获取函数。
func (m *Manager) GetCustomPermissionListFunc() func(loginID, authType string) ([]string, error) {
	return m.CustomPermissionListFunc
}

// GetCustomRoleListFunc retrieves the custom role list function. GetCustomRoleListFunc 获取自定义角色列表获取函数。
func (m *Manager) GetCustomRoleListFunc() func(loginID, authType string) ([]string, error) {
	return m.CustomRoleListFunc
}

// GetNonceManager retrieves the nonce manager. GetNonceManager 获取 Nonce 管理器。
func (m *Manager) GetNonceManager() *nonce.NonceManager {
	return m.nonceManager
}

// GetOAuth2Manager retrieves the OAuth2 manager. GetOAuth2Manager 获取 OAuth2 管理器。
func (m *Manager) GetOAuth2Manager() *oauth2.OAuth2Server {
	return m.oauth2Manager
}

// lockLoginWrite locks write operations for one login ID lockLoginWrite 锁定指定账号的写操作
func (m *Manager) lockLoginWrite(loginID string) func() {
	if loginID == "" {
		return func() {}
	}

	value, _ := m.loginLocks.LoadOrStore(loginID, &sync.Mutex{})
	lock := value.(*sync.Mutex)
	lock.Lock()
	return lock.Unlock
}

// submitAsync submits async work with goroutine fallback submitAsync 提交异步任务并在池不可用时回退到 goroutine
func (m *Manager) submitAsync(name string, task func()) {
	if m.pool == nil {
		go task()
		return
	}

	if err := m.pool.Submit(task); err != nil {
		m.logger.Errorf("manager.submitAsync: failed to submit async task, task=%s, error=%v", name, err)
		go task()
	}
}

// expireIfLimited renews key only when duration is limited expireIfLimited 仅在有限过期时间下续期 key
func (m *Manager) expireIfLimited(ctx context.Context, key string, expiration time.Duration) error {
	if expiration <= 0 {
		return nil
	}
	return m.storage.Expire(ctx, key, expiration)
}

// expireTokenIfLimited renews current or legacy token key. expireTokenIfLimited 续期当前或历史 Token 键。
func (m *Manager) expireTokenIfLimited(ctx context.Context, tokenValue string, expiration time.Duration) error {
	if expiration <= 0 {
		return nil
	}
	for _, key := range m.getTokenStorageKeys(tokenValue) {
		if !m.storage.Exists(ctx, key) {
			continue
		}
		return m.expireIfLimited(ctx, key, expiration)
	}
	return nil
}

// rollbackLogin removes data written by a failed login rollbackLogin 回滚失败登录已写入的数据
func (m *Manager) rollbackLogin(ctx context.Context, sess *Session, loginID, token string, expiration time.Duration) {
	if sess != nil {
		if _, ok := sess.removeTerminalByToken(token); ok {
			if len(sess.TerminalInfos) == 0 {
				if err := m.storage.Delete(ctx, m.getSessionKey(loginID)); err != nil {
					m.logger.Errorf("manager.rollbackLogin: failed to delete empty session, loginID=%s, token=%s, error=%v", loginID, token, err)
				}
			} else {
				if err := m.saveSessionWithMinTTL(ctx, m.getSessionKey(loginID), *sess, expiration); err != nil {
					m.logger.Errorf("manager.rollbackLogin: failed to save session, loginID=%s, token=%s, error=%v", loginID, token, err)
				}
			}
		}
	}
	if err := m.storage.Delete(ctx, append(m.getTokenStorageKeys(token), m.getRenewKey(token), m.getActiveKey(token))...); err != nil {
		m.logger.Errorf("manager.rollbackLogin: failed to delete token data, loginID=%s, token=%s, error=%v", loginID, token, err)
	}
}

// getSession retrieves session information (internal method). getSession 获取会话信息（内部方法）。
func (m *Manager) getSession(ctx context.Context, loginID string, autoCreate ...bool) (*Session, error) {
	sessData, err := m.storage.Get(ctx, m.getSessionKey(loginID))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	if sessData == nil {
		if len(autoCreate) > 0 && autoCreate[0] {
			newSession := &Session{
				AuthType:      m.config.AuthType,
				LoginID:       loginID,
				CreateTime:    time.Now().Unix(),
				TerminalInfos: make([]TerminalInfo, 0),
				Permissions:   make([]string, 0),
				Roles:         make([]string, 0),
			}
			return newSession, nil
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

// getTokenInfo retrieves token information (internal method). getTokenInfo 获取 Token 信息（内部方法）。
func (m *Manager) getTokenInfo(ctx context.Context, tokenValue string) (*TokenInfo, error) {
	tokenInfoData, err := m.storage.Get(ctx, m.getTokenKey(tokenValue))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	if tokenInfoData == nil {
		tokenInfoData, err = m.storage.Get(ctx, m.getLegacyTokenKey(tokenValue))
		if err != nil {
			return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
		if tokenInfoData == nil {
			return nil, derror.ErrInvalidToken
		}
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

// loginGetSession retrieves session for login operation (internal method). loginGetSession 获取登录操作的会话信息（内部方法）。
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

// checkLoginInternal performs the core login validation logic (internal method). checkLoginInternal 执行登录状态的核心验证逻辑（内部方法）。
func (m *Manager) checkLoginInternal(ctx context.Context, tokenValue string) error {
	// Get tokenInfo 获取 tokenInfo
	tokenInfo, err := m.getTokenInfo(ctx, tokenValue)
	if err != nil {
		return err
	}

	// Check disable status after token lookup 获取 token 后检查封禁状态
	if m.isDisable(ctx, tokenInfo.LoginID) {
		return derror.ErrAccountDisabled
	}

	sess, err := m.getSession(ctx, tokenInfo.LoginID)
	if err != nil {
		if errors.Is(err, derror.ErrSessionNotFound) {
			return derror.ErrInvalidToken
		}
		return err
	}
	if sess == nil || !sess.hasTerminalToken(tokenValue) {
		return derror.ErrInvalidToken
	}

	// Check max inactive timeout 检查最大不活跃时长
	if m.config.ActiveTimeout > 0 {
		timeStampAny, err := m.storage.Get(ctx, m.getActiveKey(tokenValue))
		if err != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
		if timeStampAny == nil {
			return derror.ErrInvalidToken
		}
		timeStamp, err := utils.ToInt64(timeStampAny)
		if err != nil {
			_ = m.storage.Delete(ctx, m.getActiveKey(tokenValue))
			return derror.ErrInvalidToken
		}
		if time.Now().Unix()-timeStamp > m.config.ActiveTimeout {
			// Kick out token when inactive timeout exceeded Token 已超过最大不活跃时长，执行踢出操作
			_ = m.Kickout(ctx, tokenValue)
			return derror.ErrTokenKickout
		}
	}

	// Renew asynchronously 异步续期
	if m.config.AutoRenew && m.config.Timeout > 0 {
		if ttl, err := m.storage.TTL(ctx, m.getTokenKey(tokenValue)); err == nil && ttl > 0 {
			ttlSeconds := int64(ttl.Seconds())
			if ttlSeconds > 0 &&
				(m.config.RenewMaxRefresh <= 0 || ttlSeconds <= m.config.RenewMaxRefresh) &&
				(m.config.RenewInterval <= 0 || !m.storage.Exists(ctx, m.getRenewKey(tokenValue))) {

				renewFunc := func() {
					m.renewFunc(context.Background(), tokenValue, tokenInfo.LoginID)
				}

				m.submitAsync("checkLoginInternal renew", renewFunc)
			}
		}
	}

	// Update active timeout asynchronously 异步活跃时长
	if m.config.ActiveTimeout > 0 {
		activeFunc := func() {
			bg := context.Background()
			unlock := m.lockLoginWrite(tokenInfo.LoginID)
			defer unlock()

			// Recheck token attachment before writing metadata 写入元数据前重新确认 Token 仍属于会话
			latestTokenInfo, err := m.getTokenInfo(bg, tokenValue)
			if err != nil {
				return
			}
			latestSession, err := m.getSession(bg, latestTokenInfo.LoginID)
			if err != nil || !latestSession.hasTerminalToken(tokenValue) {
				return
			}

			if err := m.storage.Set(bg, m.getActiveKey(tokenValue), time.Now().Unix(), m.resolveTokenExpiration(latestTokenInfo)); err != nil {
				m.logger.Errorf("manager.checkLoginInternal: failed to set active key, token=%s, error=%v", tokenValue, err)
			}
		}
		m.submitAsync("checkLoginInternal active", activeFunc)
	}

	return nil
}

// cleanExpiredTerminals removes expired tokens from session (internal method). cleanExpiredTerminals 清理会话中已过期的 token（内部方法）。
func (m *Manager) cleanExpiredTerminals(ctx context.Context, sess *Session) error {
	if sess == nil || len(sess.TerminalInfos) == 0 {
		return nil
	}

	var validTerminals []TerminalInfo
	hasExpired := false

	for _, ti := range sess.TerminalInfos {
		// Check token by full alive rules 按完整存活规则检查 token
		alive, err := m.checkTerminalTokenAlive(ctx, ti.Token)
		if err != nil {
			return err
		}
		if alive {
			validTerminals = append(validTerminals, ti)
			continue
		}

		// Remove invalid terminal 移除无效终端
		hasExpired = true
	}

	// Update session when expired tokens exist 如果有过期的 token，更新 session
	if hasExpired {
		sess.TerminalInfos = validTerminals
		if err := m.saveToStorage(ctx, m.getSessionKey(sess.LoginID), *sess); err != nil {
			return err
		}
	}

	return nil
}

// handleConcurrency handles login concurrency strategy (internal method). handleConcurrency 处理登录并发策略（内部方法）。
func (m *Manager) handleConcurrency(
	ctx context.Context,
	sess *Session,
	loginID, device string,
) (reuseToken string, handled bool, destroyedSession bool, err error) {
	// Clean expired tokens 清理已过期的 token
	if err = m.cleanExpiredTerminals(ctx, sess); err != nil {
		return "", false, false, err
	}

	if !m.config.IsConcurrent {
		if m.config.ReplacedLoginExitMode == config.ReplacedLoginExitModeNewDevice {
			// Reject new login only when an active terminal exists 仅在存在有效终端时拒绝新登录
			var terminals []TerminalInfo
			if m.config.ConcurrencyScope == config.ConcurrencyScopeAccount {
				terminals = sess.TerminalInfos
			} else if m.config.ConcurrencyScope == config.ConcurrencyScopeDevice {
				terminals = sess.getTerminalsByDevice(device)
			}
			hasActiveTerminal, activeErr := m.hasActiveTerminal(ctx, terminals)
			if activeErr != nil {
				return "", false, false, activeErr
			}
			if hasActiveTerminal {
				return "", false, false, derror.ErrLoginLimitExceeded
			}
			return "", false, false, nil
		}

		// Replace old sessions when concurrency is disabled 不允许并发：顶掉旧会话
		if m.config.ConcurrencyScope == config.ConcurrencyScopeAccount {
			if destroyedSession, err = m.removeTerminalInfosAndTokens(ctx, sess, config.LogoutModeReplaced); err != nil {
				return "", false, false, err
			}
		} else if m.config.ConcurrencyScope == config.ConcurrencyScopeDevice {
			if destroyedSession, err = m.removeTerminalInfosAndTokens(ctx, sess, config.LogoutModeReplaced, device); err != nil {
				return "", false, false, err
			}
		}
		return "", true, destroyedSession, nil
	}

	if m.config.IsShare {
		// Try token sharing reuse 允许共享：尝试复用
		var token string
		var shareErr error
		if m.config.ConcurrencyScope == config.ConcurrencyScopeAccount {
			token, shareErr = m.getTokenAndShare(ctx, sess)
		} else if m.config.ConcurrencyScope == config.ConcurrencyScopeDevice {
			token, shareErr = m.getTokenAndShare(ctx, sess, device)
		}
		if shareErr != nil {
			return "", false, false, shareErr
		}
		if token != "" {
			return token, true, false, nil
		}
	}

	if m.config.ConcurrencyScope == config.ConcurrencyScopeAccount {
		removedOverflow := false
		for m.config.MaxLoginCount > 0 && int64(len(sess.TerminalInfos)) >= m.config.MaxLoginCount {
			if err := m.removeOldestTerminalInfoAndToken(ctx, sess, m.config.OverflowLogoutMode); err != nil {
				return "", false, false, err
			}
			removedOverflow = true
		}
		if removedOverflow {
			return "", true, false, nil
		}
	} else if m.config.ConcurrencyScope == config.ConcurrencyScopeDevice {
		removedOverflow := false
		for m.config.MaxLoginCount > 0 && int64(len(sess.getTerminalsByDevice(device))) >= m.config.MaxLoginCount {
			if err := m.removeOldestTerminalInfoAndToken(ctx, sess, m.config.OverflowLogoutMode, device); err != nil {
				return "", false, false, err
			}
			removedOverflow = true
		}
		if removedOverflow {
			return "", true, false, nil
		}
	}

	return "", false, false, nil
}

// getTokenAndShare retrieves and shares a token 获取并共享 token
func (m *Manager) getTokenAndShare(ctx context.Context, sess *Session, device ...string) (string, error) {
	if len(sess.TerminalInfos) == 0 {
		return "", nil
	}

	// Get candidate terminals 获取候选的 terminals
	var candidates []TerminalInfo
	if len(device) > 0 {
		// Get terminals for specified device 指定设备：获取该设备的所有 terminals
		candidates = sess.getTerminalsByDevice(device[0])
	} else {
		// Get all terminals for account scope 账号级别：获取所有 terminals
		candidates = sess.TerminalInfos
	}

	if len(candidates) == 0 {
		return "", nil
	}

	// Reuse latest alive token 复用最后一个仍在线的 token
	var terminalInfo TerminalInfo
	for i := len(candidates) - 1; i >= 0; i-- {
		alive, err := m.checkTerminalTokenAlive(ctx, candidates[i].Token)
		if err != nil {
			return "", err
		}
		if alive {
			terminalInfo = candidates[i]
			break
		}
	}
	if terminalInfo.Token == "" {
		return "", nil
	}

	tokenInfo, err := m.getTokenInfo(ctx, terminalInfo.Token)
	if err != nil {
		return "", err
	}
	expiration := m.resolveTokenExpiration(tokenInfo)
	tokenTimeout := tokenInfo.Timeout

	// Renew session without shortening existing TTL 续期 session，避免缩短已有 TTL
	if err := m.saveSessionWithMinTTL(ctx, m.getSessionKey(terminalInfo.LoginID), *sess, expiration); err != nil {
		m.logger.Errorf("manager.getTokenAndShare: failed to save session, loginID=%s, error=%v", terminalInfo.LoginID, err)
	}

	// Renew token by original timeout 按原始有效期续期 Token
	updatedTokenInfo := TokenInfo{
		AuthType:   m.config.AuthType,
		LoginID:    terminalInfo.LoginID,
		Device:     terminalInfo.Device,
		DeviceId:   terminalInfo.DeviceId,
		CreateTime: terminalInfo.CreateTime,
		Timeout:    tokenTimeout,
	}
	if err := m.saveToStorage(ctx, m.getTokenKey(terminalInfo.Token), updatedTokenInfo, expiration); err != nil {
		return "", err
	}

	// Renew or reset metadata 续期或重新设置 metadata
	if m.config.RenewInterval > 0 {
		if err := m.storage.Set(ctx, m.getRenewKey(terminalInfo.Token), time.Now().Unix(), time.Duration(m.config.RenewInterval)*time.Second); err != nil {
			m.logger.Errorf("manager.getTokenAndShare: failed to set renew key, token=%s, error=%v", terminalInfo.Token, err)
		}
	}
	// Set active timeout 设置最大不活跃时长
	if m.config.ActiveTimeout > 0 {
		if err := m.storage.Set(ctx, m.getActiveKey(terminalInfo.Token), time.Now().Unix(), expiration); err != nil {
			m.logger.Errorf("manager.getTokenAndShare: failed to set active key, token=%s, error=%v", terminalInfo.Token, err)
		}
	}

	return terminalInfo.Token, nil
}

// removeOldestTerminalInfoAndToken removes the oldest terminal and its token (internal method). removeOldestTerminalInfoAndToken 移除最旧的终端信息并按模式处理 Token（内部方法）。
func (m *Manager) removeOldestTerminalInfoAndToken(ctx context.Context, sess *Session, mode config.LogoutMode, device ...string) error {
	terminalInfo, ok := sess.removeOldestTerminal(device...)
	if ok {
		// Apply overflow mode 应用超限处理模式
		if err := m.applyLogoutModeToToken(ctx, terminalInfo.Token, mode); err != nil {
			return err
		}
		// Clean metadata 清理 metadata
		if err := m.cleanTokenMetadata(ctx, []string{terminalInfo.Token}); err != nil {
			return err
		}
		// Save session data 保存会话数据
		if err := m.saveToStorage(ctx, m.getSessionKey(sess.LoginID), *sess); err != nil {
			return err
		}
	}
	return nil
}

// removeTerminalInfosAndTokens removes terminal information and tokens (internal method). removeTerminalInfosAndTokens 移除终端信息和 Token（内部方法）。
func (m *Manager) removeTerminalInfosAndTokens(ctx context.Context, sess *Session, mode config.LogoutMode, device ...string) (bool, error) {
	var terminalInfos []TerminalInfo
	if len(device) > 0 {
		// Remove terminals for specified device 移除指定设备类型的终端信息
		terminalInfos = sess.removeTerminalByDevice(device[0])
	} else {
		// Remove all terminals 移除所有终端信息
		terminalInfos = sess.removeAllTerminals()
	}

	// Apply mode to all removed tokens 按模式处理所有被移除 Token
	for _, terminalInfo := range terminalInfos {
		if err := m.applyLogoutModeToToken(ctx, terminalInfo.Token, mode); err != nil {
			return false, err
		}
	}
	// Clean token metadata 清理附属 metadata
	tokens := make([]string, len(terminalInfos))
	for i, info := range terminalInfos {
		tokens[i] = info.Token
	}
	if err := m.cleanTokenMetadata(ctx, tokens); err != nil {
		return false, err
	}

	// Delete session when no terminals remain 如果 session 中没有剩余终端，删除整个 session
	destroyedSession := false
	if len(sess.TerminalInfos) == 0 {
		if err := m.storage.Delete(ctx, m.getSessionKey(sess.LoginID)); err != nil {
			return false, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
		destroyedSession = true
	} else {
		// Save updated session otherwise 否则保存更新后的 session
		if err := m.saveToStorage(ctx, m.getSessionKey(sess.LoginID), *sess); err != nil {
			return false, err
		}
	}

	return destroyedSession, nil
}

// logoutTerminals performs common logout logic (internal method). logoutTerminals 通用登出逻辑：移除终端 + 删除 token + 清理 metadata（内部方法）。
func (m *Manager) logoutTerminals(
	ctx context.Context,
	loginID string,
	removalFunc func(*Session) []TerminalInfo,
) error {
	unlock := m.lockLoginWrite(loginID)
	defer unlock()

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		if errors.Is(err, derror.ErrSessionNotFound) {
			return nil
		}
		return err
	}
	if sess == nil {
		return nil // session 不存在，登出无害
	}

	removed := removalFunc(sess)
	if len(removed) == 0 {
		return nil
	}

	// Extract token list 提取 token 列表
	tokens := make([]string, len(removed))
	tokenKeys := make([]string, 0, len(removed)*2)
	for i, info := range removed {
		tokens[i] = info.Token
		tokenKeys = append(tokenKeys, m.getTokenStorageKeys(info.Token)...)
	}

	// Delete primary token keys 删除主 token keys
	if err = m.storage.Delete(ctx, tokenKeys...); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	// Clean token metadata 清理附属 metadata
	if err = m.cleanTokenMetadata(ctx, tokens); err != nil {
		return err
	}

	destroySession := false

	// Delete session when no terminals remain 如果 session 中没有剩余终端，删除整个 session
	if len(sess.TerminalInfos) == 0 {
		if err = m.storage.Delete(ctx, m.getSessionKey(loginID)); err != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
		destroySession = true
	} else {
		// Save updated session otherwise 否则保存更新后的 session
		if err = m.saveToStorage(ctx, m.getSessionKey(loginID), *sess); err != nil {
			return err
		}
	}

	unlock()
	unlock = func() {}

	if destroySession {
		// Trigger session destroy event 触发销毁 Session 事件
		m.triggerEvent(listener.EventDestroySession, loginID, "", "", "", nil)
	}

	// Trigger logout event 触发登出事件
	for _, info := range removed {
		m.triggerEvent(listener.EventLogout, loginID, info.Device, info.DeviceId, info.Token, nil)
	}

	return nil
}

// cleanTokenMetadata cleans token metadata in batch (internal method). cleanTokenMetadata 批量清理 token 的附属元数据（续期 key、活跃时间 key）（内部方法）。
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

// TerminalRemovalFunc defines how to remove terminals from a session. TerminalRemovalFunc 定义如何从 Session 中移除终端。
type TerminalRemovalFunc func(sess *Session) []TerminalInfo

// processTerminals performs common terminal processing logic (internal method). processTerminals 通用终端处理逻辑（内部方法）。
func (m *Manager) processTerminals(
	ctx context.Context,
	loginID string,
	removalFunc TerminalRemovalFunc,
	state TokenState,
) error {
	unlock := m.lockLoginWrite(loginID)
	defer unlock()

	// Load session 加载 Session
	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		if errors.Is(err, derror.ErrSessionNotFound) {
			return nil
		}
		return err
	}

	// Apply removal strategy 执行移除策略
	removedTerminals := removalFunc(sess)

	// Clean each removed token 对每个被移除的 token 执行清理
	for _, info := range removedTerminals {
		token := info.Token

		// Set token state 设置 token 状态
		if err = m.setTokenState(ctx, token, state); err != nil {
			return err
		}

		// Delete renew key 删除续期 key
		if err = m.storage.Delete(ctx, m.getRenewKey(token)); err != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}

		// Delete active key 删除活跃时间 key
		if err = m.storage.Delete(ctx, m.getActiveKey(token)); err != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
	}

	destroySession := false

	// Update session when terminals are removed 如果有移除项，更新 session
	if len(removedTerminals) > 0 {
		// Delete session when no terminals remain 如果 session 中没有剩余终端，删除整个 session
		if len(sess.TerminalInfos) == 0 {
			if err = m.storage.Delete(ctx, m.getSessionKey(loginID)); err != nil {
				return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
			}
			destroySession = true
		} else {
			// Save updated session otherwise 否则保存更新后的 session
			if err = m.saveToStorage(ctx, m.getSessionKey(loginID), *sess); err != nil {
				return err
			}
		}
	}

	unlock()
	unlock = func() {}

	if destroySession {
		// Trigger session destroy event 触发销毁 Session 事件
		m.triggerEvent(listener.EventDestroySession, loginID, "", "", "", nil)
	}

	// Trigger matched event 触发对应事件
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

// isDisable checks if an account is disabled (internal method). isDisable 检查账号是否被封禁（内部方法）。
func (m *Manager) isDisable(ctx context.Context, loginID string) bool {
	return m.storage.Exists(ctx, m.getDisableKey(loginID))
}

// filterTokens filters tokens based on checkAlive flag (internal method). filterTokens 根据 checkAlive 决定是否验证 token 有效性，并返回 token 列表（内部方法）。
func (m *Manager) filterTokens(ctx context.Context, terminals []TerminalInfo, checkAlive bool) ([]string, error) {
	if len(terminals) == 0 {
		return []string{}, nil
	}

	if !checkAlive {
		// Return all tokens without alive check 不检查存活：直接返回所有 token（预分配容量）
		tokens := make([]string, len(terminals))
		for i, ti := range terminals {
			tokens[i] = ti.Token
		}
		return tokens, nil
	}

	// Check each token by full alive rules 按完整存活规则检查每个 token
	var tokens []string // 无法预知数量，动态 append
	for _, ti := range terminals {
		alive, err := m.checkTerminalTokenAlive(ctx, ti.Token)
		if err != nil {
			return nil, err
		}
		if alive {
			tokens = append(tokens, ti.Token)
		}
		// Skip invalid tokens 若 token 无效（过期/被踢等），跳过
	}
	return tokens, nil
}

// matchPermission matches permission with wildcard support (internal method). matchPermission 权限匹配（支持通配符）（内部方法）。
func (m *Manager) matchPermission(pattern, permission string) bool {
	// Wildcard matches all permissions 全通配符匹配所有权限
	if pattern == PermissionWildcard {
		return true
	}

	// Exact match 精确匹配
	if pattern == permission {
		return true
	}

	// Return false when pattern has no wildcard 如果 pattern 不包含通配符，则不匹配
	if !strings.Contains(pattern, PermissionWildcard) {
		return false
	}

	// Auto detect separator from pattern 自动检测分隔符：优先使用 pattern 中的分隔符
	separator := PermissionSeparator // 默认使用 ":"
	if strings.Contains(pattern, "/") {
		separator = "/" // 如果包含 "/"，则使用 URL 路径格式
	}

	// Match wildcard by segments 通配符匹配：按段分割并逐段比较
	patternParts := strings.Split(pattern, separator)
	permParts := strings.Split(permission, separator)

	// Require equal segment count 段数必须一致（避免意外越权）
	if len(patternParts) != len(permParts) {
		return false
	}

	// Match each segment 逐段匹配
	for i := range patternParts {
		// Match wildcard segment 如果 pattern 的当前段是通配符，则该段匹配
		if patternParts[i] == PermissionWildcard {
			continue
		}
		// Require exact segment match 如果 pattern 的当前段不是通配符，则必须精确匹配
		if patternParts[i] != permParts[i] {
			return false
		}
	}

	return true
}

// hasPermissionInList checks if permission exists in permission list (internal method). hasPermissionInList 判断权限是否存在于权限列表中（内部方法）。
func (m *Manager) hasPermissionInList(perms []string, permission string) bool {
	for _, p := range perms {
		if m.matchPermission(p, permission) {
			return true
		}
	}
	return false
}

// renewFunc performs token renewal (internal method). renewFunc 续期函数（内部方法）。
func (m *Manager) renewFunc(ctx context.Context, tokenValue, loginID string) {
	// Validate empty parameters 参数为空校验
	if tokenValue == "" || loginID == "" {
		return
	}

	unlock := m.lockLoginWrite(loginID)
	defer unlock()

	// Recheck token attachment before renewal 续期前重新确认 Token 仍属于会话
	tokenInfo, err := m.getTokenInfo(ctx, tokenValue)
	if err != nil {
		m.logger.Errorf("manager.renewFunc: token is no longer valid, token=%s, error=%v", tokenValue, err)
		return
	}
	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		m.logger.Errorf("manager.renewFunc: failed to get session, loginID=%s, error=%v", loginID, err)
		return
	}
	if !sess.hasTerminalToken(tokenValue) {
		m.logger.Errorf("manager.renewFunc: token not found in session, token=%s", tokenValue)
		return
	}

	// Renew token with its original timeout 使用 Token 原始有效期续期
	expiration := m.resolveTokenExpiration(tokenInfo)
	if err := m.expireTokenIfLimited(ctx, tokenValue, expiration); err != nil {
		m.logger.Errorf("manager.renewFunc: failed to expire token, token=%s, error=%v", tokenValue, err)
	}

	// Renew session without shortening existing TTL 续期 Session，避免缩短已有 TTL
	if err := m.saveSessionWithMinTTL(ctx, m.getSessionKey(loginID), *sess, expiration); err != nil {
		m.logger.Errorf("manager.renewFunc: failed to save session, loginID=%s, error=%v", loginID, err)
	}

	// Set renew interval marker 设置最小续期间隔标记
	if m.config.RenewInterval > 0 {
		if err := m.storage.Set(ctx, m.getRenewKey(tokenValue), time.Now().Unix(), time.Duration(m.config.RenewInterval)*time.Second); err != nil {
			m.logger.Errorf("manager.renewFunc: failed to set renew key, token=%s, error=%v", tokenValue, err)
		}
	}

	unlock()
	unlock = func() {}

	// Trigger renew event 触发续期事件
	m.triggerEvent(listener.EventRenew, loginID, tokenInfo.Device, tokenInfo.DeviceId, tokenValue, nil)
}

// getTokenKey generates the storage key for a token. getTokenKey 获取 Token 存储键。
func (m *Manager) getTokenKey(tokenValue string) string {
	return m.config.KeyPrefix + m.config.AuthType + config.TokenKeyPrefix + tokenValue
}

// getLegacyTokenKey generates legacy token key before token namespace was added. getLegacyTokenKey 获取历史版本 Token 存储键。
func (m *Manager) getLegacyTokenKey(tokenValue string) string {
	return m.config.KeyPrefix + m.config.AuthType + tokenValue
}

// getTokenStorageKeys returns all token storage keys for cleanup. getTokenStorageKeys 返回 Token 清理需要覆盖的全部存储键。
func (m *Manager) getTokenStorageKeys(tokenValue string) []string {
	return []string{m.getTokenKey(tokenValue), m.getLegacyTokenKey(tokenValue)}
}

// setTokenState marks token logical state and removes legacy mapping. setTokenState 标记 Token 逻辑状态并清理历史映射。
func (m *Manager) setTokenState(ctx context.Context, tokenValue string, state TokenState) error {
	if err := m.storage.Set(ctx, m.getTokenKey(tokenValue), string(state), m.getExpiration()); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	if err := m.storage.Delete(ctx, m.getLegacyTokenKey(tokenValue)); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	return nil
}

// applyLogoutModeToToken applies logout mode to token mapping. applyLogoutModeToToken 按下线模式处理 Token 映射。
func (m *Manager) applyLogoutModeToToken(ctx context.Context, tokenValue string, mode config.LogoutMode) error {
	switch mode {
	case config.LogoutModeLogout:
		// Delete mapping for normal logout 普通登出直接删除映射
		if err := m.storage.Delete(ctx, m.getTokenStorageKeys(tokenValue)...); err != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
	case config.LogoutModeKickout:
		return m.setTokenState(ctx, tokenValue, TokenStateKickOut)
	case config.LogoutModeReplaced:
		return m.setTokenState(ctx, tokenValue, TokenStateReplaced)
	default:
		return derror.ErrInvalidParam
	}
	return nil
}

// getCheckedTokenSession gets token session after full login validation. getCheckedTokenSession 完整校验登录态后获取 Token 对应 Session。
func (m *Manager) getCheckedTokenSession(ctx context.Context, tokenValue string) (*Session, *TokenInfo, error) {
	if err := m.checkLoginInternal(ctx, tokenValue); err != nil {
		return nil, nil, err
	}

	tokenInfo, err := m.getTokenInfo(ctx, tokenValue)
	if err != nil {
		return nil, nil, err
	}
	sess, err := m.GetSessionByToken(ctx, tokenValue)
	if err != nil {
		return nil, nil, err
	}
	return sess, tokenInfo, nil
}

// ensureTerminalTokenAlive ensures token is still alive without renew side effects. ensureTerminalTokenAlive 无续期副作用地确认 Token 仍有效。
func (m *Manager) ensureTerminalTokenAlive(ctx context.Context, tokenValue string) error {
	alive, err := m.checkTerminalTokenAlive(ctx, tokenValue)
	if err != nil {
		return err
	}
	if !alive {
		return derror.ErrInvalidToken
	}
	return nil
}

// hasActiveTerminal reports whether any terminal is still alive. hasActiveTerminal 判断是否存在仍有效的终端。
func (m *Manager) hasActiveTerminal(ctx context.Context, terminals []TerminalInfo) (bool, error) {
	for _, terminal := range terminals {
		alive, err := m.checkTerminalTokenAlive(ctx, terminal.Token)
		if err != nil {
			return false, err
		}
		if alive {
			return true, nil
		}
	}
	return false, nil
}

// isTerminalTokenAlive checks token validity without renew side effects. isTerminalTokenAlive 无续期副作用地检查 Token 是否有效。
func (m *Manager) isTerminalTokenAlive(ctx context.Context, tokenValue string) bool {
	alive, err := m.checkTerminalTokenAlive(ctx, tokenValue)
	return err == nil && alive
}

// checkTerminalTokenAlive checks token validity without renew side effects. checkTerminalTokenAlive 无续期副作用地检查 Token 是否有效。
func (m *Manager) checkTerminalTokenAlive(ctx context.Context, tokenValue string) (bool, error) {
	tokenInfo, err := m.getTokenInfo(ctx, tokenValue)
	if err != nil {
		if errors.Is(err, derror.ErrInvalidToken) ||
			errors.Is(err, derror.ErrTokenExpired) ||
			errors.Is(err, derror.ErrTokenKickout) ||
			errors.Is(err, derror.ErrTokenReplaced) {
			return false, nil
		}
		return false, err
	}
	if tokenInfo.LoginID == "" || m.isDisable(ctx, tokenInfo.LoginID) {
		return false, nil
	}

	sess, err := m.getSession(ctx, tokenInfo.LoginID)
	if err != nil || sess == nil || !sess.hasTerminalToken(tokenValue) {
		if errors.Is(err, derror.ErrSessionNotFound) {
			return false, nil
		}
		return false, err
	}

	if m.config.ActiveTimeout <= 0 {
		return true, nil
	}

	timeStampAny, err := m.storage.Get(ctx, m.getActiveKey(tokenValue))
	if err != nil || timeStampAny == nil {
		if err != nil {
			return false, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
		return false, nil
	}
	timeStamp, err := utils.ToInt64(timeStampAny)
	if err != nil {
		return false, nil
	}
	return time.Now().Unix()-timeStamp <= m.config.ActiveTimeout, nil
}

// getSessionKey generates the storage key for a session. getSessionKey 获取会话存储键。
func (m *Manager) getSessionKey(loginID string) string {
	return m.config.KeyPrefix + m.config.AuthType + SessionKeyPrefix + loginID
}

// getRenewKey generates the storage key for token renewal tracking. getRenewKey 获取 Token 续期追踪键。
func (m *Manager) getRenewKey(tokenValue string) string {
	return m.config.KeyPrefix + m.config.AuthType + RenewKeyPrefix + tokenValue
}

// getActiveKey generates the storage key for token activity tracking. getActiveKey 获取 Token 活跃时间追踪键。
func (m *Manager) getActiveKey(tokenValue string) string {
	return m.config.KeyPrefix + m.config.AuthType + ActivePrefix + tokenValue
}

// getDisableKey generates the storage key for account disable status. getDisableKey 获取账号禁用状态存储键。
func (m *Manager) getDisableKey(loginID string) string {
	return m.config.KeyPrefix + m.config.AuthType + DisableKeyPrefix + loginID
}

// getDisableServiceKey generates the storage key for service disable status. getDisableServiceKey 获取账号分类禁用状态存储键。
func (m *Manager) getDisableServiceKey(loginID, service string) string {
	return m.config.KeyPrefix + m.config.AuthType + DisableServiceKeyPrefix + loginID + ":" + service
}

// triggerEvent triggers an event through the event manager. triggerEvent 通过事件管理器触发事件（根据配置决定同步或异步）。
func (m *Manager) triggerEvent(event listener.Event, loginID, device, deviceId, token string, extra map[string]any) {
	if m.eventManager == nil {
		return
	}

	eventData := &listener.EventData{
		Event:     event,
		AuthType:  m.config.AuthType,
		LoginID:   loginID,
		Device:    device,
		DeviceId:  deviceId,
		Token:     token,
		Extra:     extra,
		Timestamp: time.Now().Unix(),
	}

	// Choose sync or async by config 根据配置决定同步或异步触发
	if m.config.AsyncEvent {
		// Trigger asynchronously 异步触发
		eventFunc := func() {
			m.eventManager.Trigger(eventData)
		}

		m.submitAsync("triggerEvent", eventFunc)
	} else {
		// Trigger synchronously 同步触发
		m.eventManager.Trigger(eventData)
	}
}

// getExpiration calculates token expiration duration from configuration. getExpiration 从配置中计算 Token 过期时长。
func (m *Manager) getExpiration() time.Duration {
	if m.config.Timeout > 0 {
		return time.Duration(m.config.Timeout) * time.Second
	}
	return 0
}

// timeoutToSeconds converts duration to storage seconds timeoutToSeconds 将时长转换为存储层秒数
func (m *Manager) timeoutToSeconds(timeout time.Duration) int64 {
	if timeout <= 0 {
		return config.NoLimit
	}

	seconds := int64(timeout / time.Second)
	if timeout%time.Second != 0 {
		seconds++
	}
	if seconds <= 0 {
		return 1
	}
	return seconds
}

// resolveTokenExpiration resolves token expiration from token info resolveTokenExpiration 根据 token info 解析实际过期时长
func (m *Manager) resolveTokenExpiration(tokenInfo *TokenInfo) time.Duration {
	if tokenInfo != nil {
		switch {
		case tokenInfo.Timeout == config.NoLimit:
			return 0
		case tokenInfo.Timeout > 0:
			return time.Duration(tokenInfo.Timeout) * time.Second
		}
	}
	return m.getExpiration()
}

// saveSessionWithMinTTL saves session while keeping the longer existing TTL saveSessionWithMinTTL 保存 session，并保留更长的现有 TTL
func (m *Manager) saveSessionWithMinTTL(
	ctx context.Context,
	key string,
	value any,
	expiration time.Duration,
) error {
	finalExpiration := expiration
	if expiration > 0 {
		currentTTL, err := m.storage.TTL(ctx, key)
		if err == nil {
			switch {
			case currentTTL == -1:
				finalExpiration = 0
			case currentTTL > expiration:
				finalExpiration = currentTTL
			}
		}
	}

	return m.saveToStorage(ctx, key, value, finalExpiration)
}

// getDeviceAndDeviceId extracts device type and device ID from parameters. getDeviceAndDeviceId 获取设备类型和设备 ID。 规则：device 和 deviceId 是两个独立的过滤维度，互不影响
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

// saveToStorage serializes and saves data to storage backend. saveToStorage 将指定类型的数据序列化并存储到存储后端。
func (m *Manager) saveToStorage(
	ctx context.Context,
	key string,
	value any,
	expiration ...time.Duration,
) error {

	// Serialize to bytes 序列化为字节
	bytesData, err := m.serializer.Encode(value)
	if err != nil {
		return fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}

	// Build expiration duration 构建过期时长
	duration := m.getExpiration()
	if len(expiration) > 0 {
		duration = expiration[0]
	} else {
		currentTTL, ttlErr := m.storage.TTL(ctx, key)
		if ttlErr == nil {
			switch {
			case currentTTL == -1:
				duration = 0
			case currentTTL > 0:
				duration = currentTTL
			}
		}
	}

	// Persist to storage 存储到后端
	if err = m.storage.Set(ctx, key, bytesData, duration); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	return nil
}

// searchKeys searches storage keys by pattern with pagination (internal method). searchKeys 根据模式搜索存储键并分页（内部方法）。
func (m *Manager) searchKeys(ctx context.Context, pattern string, start, size int) ([]string, error) {
	keys, err := m.storage.Keys(ctx, pattern)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	total := len(keys)
	if start < 0 {
		start = 0
	}
	if start >= total {
		return []string{}, nil
	}

	// Return all when size == -1 size == -1 表示返回全部
	end := total
	if size >= 0 {
		end = start + size
		if end > total {
			end = total
		}
	}

	return keys[start:end], nil
}

// searchValues searches keys and strips storage prefix. searchValues 搜索存储键并裁剪为业务值。
func (m *Manager) searchValues(ctx context.Context, pattern, prefix string, start, size int) ([]string, error) {
	keys, err := m.searchKeys(ctx, pattern, start, size)
	if err != nil {
		return nil, err
	}
	values := make([]string, len(keys))
	for i, key := range keys {
		values[i] = strings.TrimPrefix(key, prefix)
	}
	return values, nil
}
