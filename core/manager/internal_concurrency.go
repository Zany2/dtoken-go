// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"context"
	"fmt"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"time"
)

// handleConcurrency handles login concurrency strategy. handleConcurrency 处理登录并发策略。
func (m *Manager) handleConcurrency(
	ctx context.Context,
	sess *Session,
	loginID, device, deviceId string,
	policy loginPolicy,
) (concurrencyResult, error) {
	// Clean expired tokens 清理已过期的 token
	if err := m.cleanExpiredTerminals(ctx, sess); err != nil {
		return concurrencyResult{}, err
	}

	// Handle non-concurrent login 处理不允许并发登录。
	if !policy.isConcurrent {
		// Reject new device when configured 配置为拒绝新设备时直接校验。
		if policy.replacedLoginExitMode == config.ReplacedLoginExitModeNewDevice {
			// Reject new login only when an active terminal exists 仅在存在有效终端时拒绝新登录
			// Select terminals by concurrency scope 按并发作用域选择终端。
			var terminals []TerminalInfo
			if m.config.ConcurrencyScope == config.ConcurrencyScopeAccount {
				terminals = sess.TerminalInfos
			} else if m.config.ConcurrencyScope == config.ConcurrencyScopeDevice {
				terminals = sess.getTerminalsByDevice(device)
			}
			// Check active terminal 检查是否存在活跃终端。
			hasActiveTerminal, activeErr := m.hasActiveTerminal(ctx, terminals, sess)
			if activeErr != nil {
				return concurrencyResult{}, activeErr
			}
			// Reject login when active terminal exists 存在活跃终端时拒绝登录。
			if hasActiveTerminal {
				return concurrencyResult{}, derror.ErrLoginLimitExceeded
			}
			// Allow login when no active terminal 无活跃终端时允许继续登录。
			return concurrencyResult{}, nil
		}

		// Replace old sessions when concurrency is disabled 不允许并发：顶掉旧会话
		// Replace terminals by configured scope 按配置作用域顶掉旧终端。
		destroyedSession := false
		var err error
		if m.config.ConcurrencyScope == config.ConcurrencyScopeAccount {
			if destroyedSession, err = m.removeTerminalInfosAndTokens(ctx, sess, config.LogoutModeReplaced); err != nil {
				return concurrencyResult{}, err
			}
		} else if m.config.ConcurrencyScope == config.ConcurrencyScopeDevice {
			if destroyedSession, err = m.removeTerminalInfosAndTokens(ctx, sess, config.LogoutModeReplaced, device); err != nil {
				return concurrencyResult{}, err
			}
		}
		return concurrencyResult{handled: true, destroyedSession: destroyedSession}, nil
	}

	// Try token sharing when enabled 开启共享时尝试复用 Token。
	if policy.isShare {
		// Try token sharing reuse only within the same device dimension. 仅在相同设备维度内尝试复用 Token。
		token, shareErr := m.getTokenAndShare(ctx, sess, device, deviceId)
		if shareErr != nil {
			return concurrencyResult{}, shareErr
		}
		if token != "" {
			return concurrencyResult{reuseToken: token, handled: true}, nil
		}
	}

	// Enforce account-level max login count 执行账号级最大登录数限制。
	if m.config.ConcurrencyScope == config.ConcurrencyScopeAccount {
		removedOverflow := false
		for policy.maxLoginCount > 0 && int64(len(sess.TerminalInfos)) >= policy.maxLoginCount {
			if err := m.removeOldestTerminalInfoAndToken(ctx, sess, policy.overflowLogoutMode); err != nil {
				return concurrencyResult{}, err
			}
			removedOverflow = true
		}
		if removedOverflow {
			return concurrencyResult{handled: true}, nil
		}
	} else if m.config.ConcurrencyScope == config.ConcurrencyScopeDevice {
		// Enforce device-level max login count 执行设备级最大登录数限制。
		removedOverflow := false
		for policy.maxLoginCount > 0 && int64(len(sess.getTerminalsByDevice(device))) >= policy.maxLoginCount {
			if err := m.removeOldestTerminalInfoAndToken(ctx, sess, policy.overflowLogoutMode, device); err != nil {
				return concurrencyResult{}, err
			}
			removedOverflow = true
		}
		if removedOverflow {
			return concurrencyResult{handled: true}, nil
		}
	}

	// No concurrency action needed 无需并发处理。
	return concurrencyResult{}, nil
}

// getTokenAndShare retrieves and shares a token within one device dimension. getTokenAndShare 在同一设备维度内获取并共享 token。
func (m *Manager) getTokenAndShare(ctx context.Context, sess *Session, device, deviceId string) (string, error) {
	// Return when no terminals exist 没有终端时直接返回。
	if len(sess.TerminalInfos) == 0 {
		return "", nil
	}

	// Get candidate terminals 获取候选 terminals。
	var candidates []TerminalInfo
	switch {
	case device != "" && deviceId != "":
		// Prefer concrete device matches when device ID exists. 存在设备 ID 时优先按具体设备匹配。
		candidates = sess.getTerminalsByDeviceAndDeviceId(device, deviceId)
	case device != "":
		// Fall back to device type matching when no device ID exists. 没有设备 ID 时按设备类型匹配。
		candidates = sess.getTerminalsByDevice(device)
	default:
		// Reuse by account only when caller supplied no device dimension. 调用方未提供设备维度时才按账号复用。
		candidates = sess.TerminalInfos
	}

	if len(candidates) == 0 {
		return "", nil
	}

	// Reuse latest alive token 复用最后一个仍在线的 token
	// Scan candidates from newest to oldest 从新到旧扫描候选终端。
	var terminalInfo TerminalInfo
	var tokenInfo *TokenInfo
	for i := len(candidates) - 1; i >= 0; i-- {
		candidateInfo, err := m.getTokenInfo(ctx, candidates[i].Token)
		if err != nil {
			if isTokenInactiveError(err) {
				continue
			}
			return "", err
		}
		alive, err := m.checkTerminalTokenAliveWithContext(ctx, candidates[i].Token, candidateInfo, sess)
		if err != nil {
			return "", err
		}
		if alive {
			terminalInfo = candidates[i]
			tokenInfo = candidateInfo
			break
		}
	}
	if terminalInfo.Token == "" {
		return "", nil
	}

	// Resolve reused token expiration 解析复用 Token 过期时间。
	expiration := m.resolveTokenExpiration(tokenInfo)
	// Preserve original timeout 保留原始过期秒数。
	tokenTimeout := tokenInfo.Timeout

	// Renew session without shortening existing TTL 续期 session，避免缩短已有 TTL
	if err := m.saveSessionWithMinTTL(ctx, m.getSessionKey(terminalInfo.LoginID), *sess, expiration); err != nil {
		return "", err
	}

	// Renew token by original timeout 按原始有效期续期 Token
	// Rebuild token info 重建 Token 信息。
	updatedTokenInfo := TokenInfo{
		AuthType:      m.config.AuthType,
		LoginID:       terminalInfo.LoginID,
		Device:        terminalInfo.Device,
		DeviceId:      terminalInfo.DeviceId,
		CreateTime:    terminalInfo.CreateTime,
		Timeout:       tokenTimeout,
		ActiveTimeout: tokenInfo.ActiveTimeout,
		Extra:         tokenInfo.Extra,
	}
	// Persist reused token info 持久化复用 Token 信息。
	if err := m.saveToStorage(ctx, m.getTokenKey(terminalInfo.Token), updatedTokenInfo, expiration); err != nil {
		return "", err
	}

	// Renew or reset metadata 续期或重新设置 metadata
	if m.config.RenewInterval > 0 {
		// Refresh renew marker 刷新续期标记。
		if err := m.storage.Set(ctx, m.getRenewKey(terminalInfo.Token), time.Now().Unix(), time.Duration(m.config.RenewInterval)*time.Second); err != nil {
			return "", fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
	}
	// Set active timeout 设置最大不活跃时长
	if m.resolveActiveTimeoutFromSeconds(tokenInfo.ActiveTimeout) > 0 {
		// Refresh active marker 刷新活跃标记。
		if err := m.storage.Set(ctx, m.getActiveKey(terminalInfo.Token), time.Now().Unix(), expiration); err != nil {
			return "", fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
	}

	// Return reused token 返回复用 Token。
	return terminalInfo.Token, nil
}
