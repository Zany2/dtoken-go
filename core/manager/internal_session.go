// Author records daixk as original author at 2026/1/22 17:33:00. Author 记录 daixk 为原始作者，创建时间为 2026/1/22 17:33:00。
package manager

import (
	"context"
	"errors"
	"fmt"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/utils"
	"time"
)

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
