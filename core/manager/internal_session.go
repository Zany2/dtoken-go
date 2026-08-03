// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"context"
	"errors"
	"fmt"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/utils"
	"time"
)

// getSession retrieves session information. getSession 获取会话信息。
func (m *Manager) getSession(ctx context.Context, loginID string) (*Session, error) {
	// Load session data 加载会话数据。
	sessData, err := m.storage.Get(ctx, m.getSessionKey(loginID))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	// Handle missing session 处理会话不存在。
	if sessData == nil {
		// Return session not found 返回会话不存在。
		return nil, derror.ErrSessionNotFound
	}

	// Convert storage value 转换存储值。
	bytesData, err := utils.ToBytes(sessData)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrTypeConvert, err)
	}

	// Decode session data 解码会话数据。
	var sess Session
	err = m.serializer.Decode(bytesData, &sess)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}

	// Return session 返回会话。
	return &sess, nil
}

// getTokenInfo retrieves token information. getTokenInfo 获取 Token 信息。
func (m *Manager) getTokenInfo(ctx context.Context, tokenValue string) (*TokenInfo, error) {
	// Validate token value 校验 Token 值。
	if tokenValue == "" {
		return nil, derror.ErrInvalidToken
	}

	// Load token data 加载 Token 数据。
	tokenInfoData, err := m.storage.Get(ctx, m.getTokenKey(tokenValue))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	// Return invalid token when missing Token 不存在时返回无效 Token。
	if tokenInfoData == nil {
		return nil, derror.ErrInvalidToken
	}

	// Convert token storage value 转换 Token 存储值。
	tokenInfoBytes, err := utils.ToBytes(tokenInfoData)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrTypeConvert, err)
	}

	// Detect logical token state 识别 Token 逻辑状态。
	if stateErr := tokenStateError(TokenState(tokenInfoBytes)); stateErr != nil {
		return nil, stateErr
	}

	// Decode token info 解码 Token 信息。
	var tokenInfo TokenInfo
	err = m.serializer.Decode(tokenInfoBytes, &tokenInfo)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}

	// Return token info 返回 Token 信息。
	return &tokenInfo, nil
}

// checkLoginAndGetContext validates login state and returns loaded context. checkLoginAndGetContext 校验登录态并返回已加载上下文。
func (m *Manager) checkLoginAndGetContext(ctx context.Context, tokenValue string) (*Session, *TokenInfo, error) {
	return m.checkLoginAndGetContextWithOptions(ctx, tokenValue, checkLoginOptions{allowRenew: true})
}

// checkLoginAndGetContextNoRenew validates login state without renew side effects. checkLoginAndGetContextNoRenew 校验登录态但不触发续期副作用。
func (m *Manager) checkLoginAndGetContextNoRenew(ctx context.Context, tokenValue string) (*Session, *TokenInfo, error) {
	return m.checkLoginAndGetContextWithOptions(ctx, tokenValue, checkLoginOptions{})
}

// checkLoginAndGetContextNoRenewLocked validates login state while caller holds login lock. checkLoginAndGetContextNoRenewLocked 在调用方已持有登录锁时校验登录态。
func (m *Manager) checkLoginAndGetContextNoRenewLocked(ctx context.Context, tokenValue string) (*Session, *TokenInfo, error) {
	return m.checkLoginAndGetContextWithOptions(ctx, tokenValue, checkLoginOptions{lockHeld: true})
}

// checkLoginOptions controls login-state validation side effects. checkLoginOptions 控制登录态校验副作用。
type checkLoginOptions struct {
	allowRenew bool // allowRenew enables async renew and active refresh tasks. allowRenew 启用异步续期和活跃刷新任务。
	lockHeld   bool // lockHeld indicates caller already holds the login write lock. lockHeld 表示调用方已持有登录写锁。
}

// checkLoginAndGetContextWithOptions validates login state with optional side effects. checkLoginAndGetContextWithOptions 按选项校验登录态。
func (m *Manager) checkLoginAndGetContextWithOptions(ctx context.Context, tokenValue string, opts checkLoginOptions) (*Session, *TokenInfo, error) {
	// Get tokenInfo 获取 tokenInfo
	tokenInfo, err := m.getTokenInfo(ctx, tokenValue)
	if err != nil {
		return nil, nil, err
	}

	// Check disable status after token lookup 获取 token 后检查封禁状态。
	if err := m.checkLoginDisableState(ctx, tokenInfo.LoginID, tokenInfo.Device, tokenInfo.DeviceID); err != nil {
		return nil, nil, err
	}

	// Load session 加载会话。
	sess, err := m.getSession(ctx, tokenInfo.LoginID)
	if err != nil {
		// Map missing session to invalid token 会话不存在时映射为无效 Token。
		if errors.Is(err, derror.ErrSessionNotFound) {
			return nil, nil, derror.ErrInvalidToken
		}
		return nil, nil, err
	}

	// Validate token attachment 校验 Token 是否属于会话。
	if sess == nil || !sess.hasTerminalToken(tokenValue) {
		return nil, nil, derror.ErrInvalidToken
	}

	// Check max inactive timeout 检查最大不活跃时长
	activeTimeout := m.resolveActiveTimeoutFromSeconds(tokenInfo.ActiveTimeout)
	if activeTimeout > 0 {
		// Load active timestamp 加载活跃时间戳。
		timeStampAny, err := m.storage.Get(ctx, m.getActiveKey(tokenValue))
		if err != nil {
			return nil, nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}

		// Reject missing active marker 拒绝缺失的活跃标记。
		if timeStampAny == nil {
			return nil, nil, derror.ErrInvalidToken
		}

		// Convert active timestamp 转换活跃时间戳。
		timeStamp, err := utils.ToInt64(timeStampAny)
		if err != nil {
			_ = m.storage.Delete(ctx, m.getActiveKey(tokenValue))
			return nil, nil, derror.ErrInvalidToken
		}

		// Handle inactive timeout 处理不活跃超时。
		if time.Now().Unix()-timeStamp > activeTimeout {
			if opts.lockHeld {
				// Mark active timeout without reentering the same login lock. 已持锁时不重复进入同一登录锁。
				if err = m.markActiveTimeoutLocked(ctx, tokenInfo.LoginID, tokenValue, sess); err != nil {
					return nil, nil, err
				}
			} else {
				// Mark inactive timeout separately so later checks keep the exact cause. 单独标记不活跃超时以保留精确原因。
				if err = m.processTerminals(ctx, tokenInfo.LoginID, func(sess *Session) []TerminalInfo {
					if info, ok := sess.removeTerminalByToken(tokenValue); ok {
						return []TerminalInfo{info}
					}
					return nil
				}, TokenStateActiveTimeout); err != nil {
					return nil, nil, err
				}
			}
			return nil, nil, derror.ErrActiveTimeout
		}
	}

	// Renew asynchronously when allowed 允许时异步续。
	if opts.allowRenew && m.config.AutoRenew && m.config.Timeout > 0 {
		// Read token TTL 读取 Token TTL。
		if ttl, err := m.storage.TTL(ctx, m.getTokenKey(tokenValue)); err == nil && ttl > 0 {
			ttlSeconds := int64(ttl.Seconds())

			// Check renew threshold 检查续期阈值。
			if ttlSeconds > 0 &&
				(m.config.RenewMaxRefresh <= 0 || ttlSeconds <= m.config.RenewMaxRefresh) &&
				(m.config.RenewInterval <= 0 || !m.storage.Exists(ctx, m.getRenewKey(tokenValue))) {

				// Build async renew task 构建异步续期任务。
				renewFunc := func() {
					m.renewFunc(context.Background(), tokenValue, tokenInfo.LoginID)
				}

				// Submit async renew task 提交异步续期任务。
				m.submitAsync("checkLoginInternal renew", renewFunc)
			}
		}
	}

	// Update active timeout asynchronously when allowed 允许时异步刷新活跃时。
	if opts.allowRenew && activeTimeout > 0 {
		// Build async active refresh task 构建异步活跃刷新任务。
		activeFunc := func() {
			bg := context.Background()
			unlock := m.lockLoginWrite(tokenInfo.LoginID)
			defer func() { unlock() }()

			// Recheck token attachment before writing metadata 写入元数据前重新确认 Token 仍属于会话。
			latestTokenInfo, err := m.getTokenInfo(bg, tokenValue)
			if err != nil {
				return
			}
			latestSession, err := m.getSession(bg, latestTokenInfo.LoginID)
			if err != nil || !latestSession.hasTerminalToken(tokenValue) {
				return
			}

			// Refresh active marker 刷新活跃标记。
			if err := m.storage.Set(bg, m.getActiveKey(tokenValue), time.Now().Unix(), m.resolveTokenExpiration(latestTokenInfo)); err != nil {
				m.logger.Errorf("manager.checkLoginInternal: failed to set active key, token=%s, error=%v", tokenValue, err)
			}
		}

		// Submit async active refresh task 提交异步活跃刷新任务。
		m.submitAsync("checkLoginInternal active", activeFunc)
	}

	// Return checked context 返回已校验上下文。
	return sess, tokenInfo, nil
}

// markActiveTimeoutLocked marks one token inactive while login lock is already held. markActiveTimeoutLocked 在已持有登录锁时标记 Token 不活跃超时。
func (m *Manager) markActiveTimeoutLocked(ctx context.Context, loginID, tokenValue string, sess *Session) error {
	if sess == nil {
		return nil
	}
	if _, ok := sess.removeTerminalByToken(tokenValue); !ok {
		return nil
	}
	if err := m.setTokenState(ctx, tokenValue, TokenStateActiveTimeout, m.tokenStateExpiration(ctx, tokenValue)); err != nil {
		return err
	}
	if err := m.cleanTokenMetadata(ctx, []string{tokenValue}); err != nil {
		return err
	}
	if len(sess.TerminalInfos) == 0 {
		if err := m.storage.Delete(ctx, m.getSessionKey(loginID)); err != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
		return nil
	}
	return m.saveToStorage(ctx, m.getSessionKey(loginID), *sess)
}

// checkLoginInternal performs the core login validation logic. checkLoginInternal 执行登录状态的核心验证逻辑。
func (m *Manager) checkLoginInternal(ctx context.Context, tokenValue string) error {
	// Validate and discard loaded context 校验并丢弃已加载上下文。
	_, _, err := m.checkLoginAndGetContext(ctx, tokenValue)
	return err
}

// cleanExpiredTerminals removes expired tokens from session. cleanExpiredTerminals 清理会话中已过期的 token。
func (m *Manager) cleanExpiredTerminals(ctx context.Context, sess *Session) (bool, error) {
	// Skip empty session 跳过空会话。
	if sess == nil || len(sess.TerminalInfos) == 0 {
		return false, nil
	}

	// Prepare valid terminal list 准备有效终端列表。
	var validTerminals []TerminalInfo
	hasExpired := false

	// Check each terminal 逐个检查终端。
	for _, ti := range sess.TerminalInfos {
		// Check token by full alive rules 按完整存活规则检查 token
		alive, err := m.checkTerminalTokenAliveWithContext(ctx, ti.Token, nil, sess)
		if err != nil {
			return false, err
		}
		if alive {
			validTerminals = append(validTerminals, ti)
			continue
		}

		// Remove invalid terminal 移除无效终端
		hasExpired = true
	}

	// Update session when expired tokens exist 存在过期 Token 时更新 Session
	if hasExpired {
		// Replace terminal list 替换终端列表。
		sess.TerminalInfos = validTerminals

		// Delete session when all terminals expired 所有终端均已过期时删除整个 session
		if len(validTerminals) == 0 {
			if err := m.storage.Delete(ctx, m.getSessionKey(sess.LoginID)); err != nil {
				return false, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
			}
			return true, nil
		} else {
			if err := m.saveToStorage(ctx, m.getSessionKey(sess.LoginID), *sess); err != nil {
				return false, err
			}
		}
	}

	return false, nil
}
