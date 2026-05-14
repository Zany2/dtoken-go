// Author records daixk as original author at 2026/1/22 17:33:00. Author 记录 daixk 为原始作者，创建时间为 2026/1/22 17:33:00。
package manager

import (
	"context"
	"errors"
	"fmt"
	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/listener"
	"time"
)

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
	if ttl == adapter.TTLNotFound {
		ttl, err = m.storage.TTL(ctx, m.getLegacyTokenKey(tokenValue))
		if err != nil {
			return 0, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
	}

	// Normalize TTL sentinel values 统一 TTL 哨兵值语义
	seconds := int64(ttl)
	switch {
	case ttl == adapter.TTLNotFound:
		return -2, nil
	case ttl == adapter.TTLNoExpire:
		return -1, nil
	case seconds > 0:
		return int64(ttl.Seconds()), nil
	default:
		return 0, nil
	}
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
