// Author records daixk as original author at 2026/1/22 17:33:00. Author 记录 daixk 为原始作者，创建时间为 2026/1/22 17:33:00。
package manager

import (
	"context"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"time"
)

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
