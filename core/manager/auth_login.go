// @Author daixk 2025/12/22 15:56:00
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
	// Delegate to default timeout login 委托默认过期时间登录。
	device, deviceId := m.getDeviceAndDeviceId(deviceAndDeviceId...)
	return m.LoginWithOptions(ctx, LoginOptions{
		LoginID:  loginID,
		Device:   device,
		DeviceID: deviceId,
	})
}

// LoginWithTimeout performs user login with a custom token timeout and returns a token. LoginWithTimeout 执行用户登录并返回 token，使用指定的过期时间（0 或负数则使用全局配置）。
func (m *Manager) LoginWithTimeout(ctx context.Context, loginID string, timeout time.Duration, deviceAndDeviceId ...string) (string, error) {
	device, deviceId := m.getDeviceAndDeviceId(deviceAndDeviceId...)
	return m.LoginWithOptions(ctx, LoginOptions{
		LoginID:  loginID,
		Device:   device,
		DeviceID: deviceId,
		Timeout:  timeout,
	})
}

// LoginWithOptions performs user login with per-call options. LoginWithOptions 使用单次登录选项执行登录。
func (m *Manager) LoginWithOptions(ctx context.Context, opts LoginOptions) (string, error) {
	// Validate login ID 校验登录 ID。
	if opts.LoginID == "" {
		return "", derror.ErrIDIsEmpty
	}

	// Lock account writes 锁定账号写操作。
	unlock := m.lockLoginWrite(opts.LoginID)
	// Release lock on function exit 函数退出时释放锁。
	defer func() { unlock() }()

	// Reject disabled account 拒绝已封禁账号。
	if m.isDisable(ctx, opts.LoginID) {
		return "", derror.ErrAccountDisabled
	}

	// Parse device fields 解析设备字段。
	device, deviceId := opts.Device, opts.DeviceID
	// Reject disabled device 拒绝已封禁设备。
	if m.isDisableDeviceMatch(ctx, opts.LoginID, device, deviceId) {
		return "", derror.ErrDeviceDisabled
	}

	// Load existing session 尝试加载现有 session
	sess, err := m.getSession(ctx, opts.LoginID)
	if errors.Is(err, derror.ErrSessionNotFound) {
		sess = nil
		err = nil
	}
	if err != nil {
		return "", err
	}

	destroyedSession := false // destroyedSession records whether old terminals removed the whole session destroyedSession 记录旧终端是否清空整个会话

	// Handle concurrency strategy 处理并发策略
	if sess != nil {
		token, handled, sessionDestroyed, handleErr := m.handleConcurrency(ctx, sess, opts.LoginID, device, deviceId, m.resolveLoginPolicy(opts))
		if handleErr != nil {
			return "", handleErr
		}
		// Record session destroy result 记录会话销毁结果。
		destroyedSession = sessionDestroyed
		if handled {
			// Return shared token when reused 复用时直接返回共享 Token。
			if token != "" {
				// Release lock before events 触发事件前释放锁。
				unlock()
				unlock = func() {}

				// Trigger shared login event 触发共享 Token 登录事件。
				m.triggerEvent(listener.EventLogin, opts.LoginID, device, deviceId, token, map[string]any{
					listener.ExtraKeyShared: true,
				})
				return token, nil // 复用 token
			}
		}
	}

	// Generate new token 生成新 token
	token := opts.Token
	if token == "" {
		token, err = m.generator.Generate(opts.LoginID, device, deviceId)
		if err != nil {
			return "", err
		}
	}

	// Record create time 记录创建时间
	createTime := time.Now().Unix()

	createdSession := sess == nil || destroyedSession // createdSession records whether this login creates a new session createdSession 记录本次登录是否创建新会话
	if createdSession {
		// Initialize new session 初始化新会话。
		sess = m.strategy.normalize().CreateSession(m.config.AuthType, opts.LoginID, createTime)
	}

	// Increase history terminal count 递增历史终端计数
	sess.HistoryTerminalCount++

	// Append terminal info 添加终端信息
	sess.TerminalInfos = append(sess.TerminalInfos, TerminalInfo{
		Token:      token,
		LoginID:    opts.LoginID,
		Device:     device,
		DeviceId:   deviceId,
		CreateTime: createTime,
		Extra:      opts.TerminalExtra,
		Index:      sess.HistoryTerminalCount, // 设置历史登录顺序索引
	})

	// Calculate expiration duration 计算过期时长
	expiration := m.getExpiration()
	// Override expiration when specified 指定时覆盖过期时间。
	if opts.Timeout > 0 {
		expiration = opts.Timeout
	}

	// Save session without shortening existing TTL 保存 session，避免缩短已有 TTL
	if err = m.saveSessionWithMinTTL(ctx, m.getSessionKey(opts.LoginID), *sess, expiration); err != nil {
		return "", err
	}

	// Save token info 保存 token info
	if err = m.saveToStorage(ctx, m.getTokenKey(token), TokenInfo{
		AuthType:      m.config.AuthType,
		LoginID:       opts.LoginID,
		Device:        device,
		DeviceId:      deviceId,
		CreateTime:    createTime,
		Timeout:       m.timeoutToSeconds(expiration),
		ActiveTimeout: m.activeTimeoutToSeconds(opts.ActiveTimeout),
		Extra:         opts.Extra,
	}, expiration); err != nil {
		m.rollbackLogin(ctx, sess, opts.LoginID, token, expiration)
		return "", err
	}

	// Initialize token metadata 初始化 token 元数据
	if m.config.RenewInterval > 0 {
		// Initialize renew marker 初始化续期标记。
		if err = m.storage.Set(ctx, m.getRenewKey(token), time.Now().Unix(), time.Duration(m.config.RenewInterval)*time.Second); err != nil {
			m.rollbackLogin(ctx, sess, opts.LoginID, token, expiration)
			return "", fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
	}
	if m.resolveActiveTimeoutFromSeconds(m.activeTimeoutToSeconds(opts.ActiveTimeout)) > 0 {
		// Initialize active marker 初始化活跃标记。
		if err = m.storage.Set(ctx, m.getActiveKey(token), time.Now().Unix(), expiration); err != nil {
			m.rollbackLogin(ctx, sess, opts.LoginID, token, expiration)
			return "", fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
	}

	// Release lock before events 触发事件前释放锁。
	unlock()
	unlock = func() {}

	if destroyedSession {
		// Trigger session destroy event after lock release 释放账号写锁后触发销毁 Session 事件
		m.triggerEvent(listener.EventDestroySession, opts.LoginID, "", "", "", nil)
	}
	if createdSession {
		// Trigger session create event after successful persistence 持久化成功后触发创建 Session 事件
		m.triggerEvent(listener.EventCreateSession, opts.LoginID, "", "", "", nil)
	}

	// Trigger login event 触发登录事件
	m.triggerEvent(listener.EventLogin, opts.LoginID, device, deviceId, token, nil)

	return token, nil
}

// LoginByToken performs login renewal based on an existing token. LoginByToken 根据 Token 续期登录。
func (m *Manager) LoginByToken(ctx context.Context, tokenValue string) error {
	// Validate token value 校验 Token 值。
	if tokenValue == "" {
		return derror.ErrInvalidToken
	}

	// Load token info 加载 Token 信息。
	tokenInfo, err := m.getTokenInfo(ctx, tokenValue)
	if err != nil {
		return err
	}

	// Lock account writes 锁定账号写操作。
	unlock := m.lockLoginWrite(tokenInfo.LoginID)
	// Release lock on function exit 函数退出时释放锁。
	defer func() { unlock() }()

	// Reload token after acquiring lock 加锁后重新读取 token，避免并发下复活已失效 token
	tokenInfo, err = m.getTokenInfo(ctx, tokenValue)
	if err != nil {
		return err
	}

	// Check account disable status 检查账号是否被封禁
	if m.isDisable(ctx, tokenInfo.LoginID) {
		return derror.ErrAccountDisabled
	}
	// Check device disable status 检查设备封禁状态。
	if m.isDisableDeviceMatch(ctx, tokenInfo.LoginID, tokenInfo.Device, tokenInfo.DeviceId) {
		return derror.ErrDeviceDisabled
	}

	// Load session 加载会话。
	sess, err := m.getSession(ctx, tokenInfo.LoginID)
	if err != nil {
		// Treat missing session as invalid token 会话不存在时视为无效 Token。
		if errors.Is(err, derror.ErrSessionNotFound) {
			return derror.ErrInvalidToken
		}
		return err
	}

	// Validate token in session terminals 验证 token 是否在 session 的 TerminalInfos 中
	if !sess.hasTerminalToken(tokenValue) {
		return derror.ErrInvalidToken
	}
	// Ensure token is still alive 确认 Token 仍然有效。
	if err := m.ensureTerminalTokenAlive(ctx, tokenValue); err != nil {
		return err
	}

	// Renew token and session asynchronously 异步续期 Token 和 Session
	renewFunc := func() {
		// Use background context for async renewal 异步续期使用后台上下文。
		bg := context.Background()
		// Lock account writes in async task 异步任务中锁定账号写操作。
		unlock := m.lockLoginWrite(tokenInfo.LoginID)
		// Release async lock on exit 异步任务退出时释放锁。
		defer func() { unlock() }()

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

		// Resolve token expiration 解析 Token 过期时长。
		expiration := m.resolveTokenExpiration(latestTokenInfo)
		// Build session key 构建会话键。
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
			// Refresh renew marker 刷新续期标记。
			if err := m.storage.Set(bg, m.getRenewKey(tokenValue), time.Now().Unix(), time.Duration(m.config.RenewInterval)*time.Second); err != nil {
				m.logger.Errorf("manager.LoginByToken: failed to set renew key, token=%s, error=%v", tokenValue, err)
			}
		}
		if m.config.ActiveTimeout > 0 {
			// Refresh active marker 刷新活跃标记。
			if err := m.storage.Set(bg, m.getActiveKey(tokenValue), time.Now().Unix(), expiration); err != nil {
				m.logger.Errorf("manager.LoginByToken: failed to set active key, token=%s, error=%v", tokenValue, err)
			}
		}

		// Release lock before event 触发事件前释放锁。
		unlock()
		unlock = func() {}

		// Trigger renew event 触发续期事件
		m.triggerEvent(listener.EventRenew, latestTokenInfo.LoginID, latestTokenInfo.Device, latestTokenInfo.DeviceId, tokenValue, nil)
	}

	// Submit async renewal 提交异步续期。
	m.submitAsync("LoginByToken", renewFunc)

	return nil
}

// IsLogin checks if a user is logged in. IsLogin 检查用户是否登录。
func (m *Manager) IsLogin(ctx context.Context, tokenValue string) bool {
	// Check login state 检查登录状态。
	return m.checkLoginInternal(ctx, tokenValue) == nil
}

// CheckLogin checks if a user is logged in and returns an error if not. CheckLogin 检查用户是否登录，如果未登录则返回错误。
func (m *Manager) CheckLogin(ctx context.Context, tokenValue string) error {
	// Check login state 检查登录状态。
	return m.checkLoginInternal(ctx, tokenValue)
}

// GetLoginID retrieves the login ID from a token. GetLoginID 根据 Token 获取登录 ID。
func (m *Manager) GetLoginID(ctx context.Context, tokenValue string) (string, error) {
	// Get checked token 获取已校验 Token
	_, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		return "", err
	}

	// Return login ID 返回登录 ID。
	return tokenInfo.LoginID, nil
}

// GetTokenInfo retrieves token information. GetTokenInfo 根据 Token 获取 TokenInfo 信息。
func (m *Manager) GetTokenInfo(ctx context.Context, tokenValue string) (*TokenInfo, error) {
	// Load token info 加载 Token 信息。
	return m.getTokenInfo(ctx, tokenValue)
}

// GetDevice retrieves the device type for a token. GetDevice 获取 Token 的设备类型。
func (m *Manager) GetDevice(ctx context.Context, tokenValue string) (string, error) {
	// Validate token and load info 校验 Token 并加载信息。
	_, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		return "", err
	}
	// Return device type 返回设备类型。
	return tokenInfo.Device, nil
}

// GetDeviceId retrieves the device ID for a token. GetDeviceId 获取 Token 的设备 ID。
func (m *Manager) GetDeviceId(ctx context.Context, tokenValue string) (string, error) {
	// Validate token and load info 校验 Token 并加载信息。
	_, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		return "", err
	}
	// Return device ID 返回设备 ID。
	return tokenInfo.DeviceId, nil
}

// GetTokenCreateTime retrieves the creation time for a token. GetTokenCreateTime 获取 Token 的创建时间戳。
func (m *Manager) GetTokenCreateTime(ctx context.Context, tokenValue string) (int64, error) {
	// Validate token and load info 校验 Token 并加载信息。
	_, tokenInfo, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		return 0, err
	}
	// Return token create time 返回 Token 创建时间。
	return tokenInfo.CreateTime, nil
}

// GetTokenTTL retrieves the remaining time-to-live for a token in seconds. GetTokenTTL 获取 Token 的剩余有效时间（秒）。
func (m *Manager) GetTokenTTL(ctx context.Context, tokenValue string) (int64, error) {
	// Validate token value 校验 Token 值。
	if tokenValue == "" {
		return 0, derror.ErrInvalidToken
	}

	// Load current token TTL 加载当前 Token TTL。
	ttl, err := m.storage.TTL(ctx, m.getTokenKey(tokenValue))
	if err != nil {
		return 0, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	// Fallback to legacy token key 回退到历史 Token 键。
	if ttl == adapter.TTLNotFound {
		ttl, err = m.storage.TTL(ctx, m.getLegacyTokenKey(tokenValue))
		if err != nil {
			return 0, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
	}

	// Normalize TTL sentinel values 统一 TTL 哨兵值语义
	switch {
	case ttl == adapter.TTLNotFound:
		return normalizeTTLSeconds(ttl), nil
	case ttl == adapter.TTLNoExpire:
		return normalizeTTLSeconds(ttl), nil
	case ttl > 0:
		return normalizeTTLSeconds(ttl), nil
	default:
		return 0, nil
	}
}

// normalizeTTLSeconds normalizes storage ttl sentinels to seconds. normalizeTTLSeconds 将存储 TTL 哨兵值归一化为秒数。
func normalizeTTLSeconds(ttl time.Duration) int64 {
	switch {
	case ttl == adapter.TTLNotFound:
		return -2
	case ttl == adapter.TTLNoExpire:
		return -1
	case ttl > 0:
		return int64(ttl.Seconds())
	default:
		return 0
	}
}

// RenewTimeout manually renews the timeout of a token. RenewTimeout 手动续期指定 Token 的过期时间。
func (m *Manager) RenewTimeout(ctx context.Context, tokenValue string, timeout time.Duration) error {
	// Validate token value 校验 Token 值。
	if tokenValue == "" {
		return derror.ErrInvalidToken
	}

	// Load token info 加载 Token 信息。
	tokenInfo, err := m.getTokenInfo(ctx, tokenValue)
	if err != nil {
		return err
	}

	// Lock account writes 锁定账号写操作。
	unlock := m.lockLoginWrite(tokenInfo.LoginID)
	// Release lock on function exit 函数退出时释放锁。
	defer func() { unlock() }()

	// Reload token after acquiring lock 加锁后重新读取 token，避免并发续期失效 token
	tokenInfo, err = m.getTokenInfo(ctx, tokenValue)
	if err != nil {
		return err
	}

	// Load session 加载会话。
	sess, err := m.getSession(ctx, tokenInfo.LoginID)
	if err != nil {
		// Treat missing session as invalid token 会话不存在时视为无效 Token。
		if errors.Is(err, derror.ErrSessionNotFound) {
			return derror.ErrInvalidToken
		}
		return err
	}
	// Validate token attachment 校验 Token 是否属于会话。
	if sess == nil || !sess.hasTerminalToken(tokenValue) {
		return derror.ErrInvalidToken
	}
	// Ensure token is still alive 确认 Token 仍然有效。
	if err = m.ensureTerminalTokenAlive(ctx, tokenValue); err != nil {
		return err
	}

	// Normalize renewal expiration 规范化续期时长。
	expiration := timeout
	if expiration <= 0 {
		expiration = 0
	}
	// Record timeout seconds 记录过期秒数。
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
		// Load current active marker 加载当前活跃标记。
		activeValue, activeErr := m.storage.Get(ctx, m.getActiveKey(tokenValue))
		if activeErr != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, activeErr)
		}
		// Initialize active marker if missing 缺失时初始化活跃标记。
		if activeValue == nil {
			activeValue = time.Now().Unix()
		}
		// Persist active marker TTL 持久化活跃标记 TTL。
		if err = m.storage.Set(ctx, m.getActiveKey(tokenValue), activeValue, expiration); err != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
	}

	// Release lock before event 触发事件前释放锁。
	unlock()
	unlock = func() {}

	// Trigger renew event 触发续期事件
	m.triggerEvent(listener.EventRenew, tokenInfo.LoginID, tokenInfo.Device, tokenInfo.DeviceId, tokenValue, map[string]any{
		"timeout": timeout.Seconds(),
	})

	return nil
}

// renewFunc performs token renewal. renewFunc 续期函数。
func (m *Manager) renewFunc(ctx context.Context, tokenValue, loginID string) {
	// Validate empty parameters 参数为空校验
	if tokenValue == "" || loginID == "" {
		return
	}

	// Lock account writes 锁定账号写操作。
	unlock := m.lockLoginWrite(loginID)
	// Release lock on function exit 函数退出时释放锁。
	defer func() { unlock() }()

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
	// Validate token attachment 校验 Token 是否属于会话。
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
		// Refresh renew marker 刷新续期标记。
		if err := m.storage.Set(ctx, m.getRenewKey(tokenValue), time.Now().Unix(), time.Duration(m.config.RenewInterval)*time.Second); err != nil {
			m.logger.Errorf("manager.renewFunc: failed to set renew key, token=%s, error=%v", tokenValue, err)
		}
	}

	// Release lock before event 触发事件前释放锁。
	unlock()
	unlock = func() {}

	// Trigger renew event 触发续期事件
	m.triggerEvent(listener.EventRenew, loginID, tokenInfo.Device, tokenInfo.DeviceId, tokenValue, nil)
}
