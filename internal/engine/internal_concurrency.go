// @Author daixk 2025/12/22 15:56:00
package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/Zany2/dtoken-go/config"
	derror "github.com/Zany2/dtoken-go/errors"
)

// handleConcurrency handles login concurrency strategy. handleConcurrency 处理登录并发策略。
func (m *Manager) handleConcurrency(
	ctx context.Context,
	sess *Session,
	loginID, device, deviceID string,
	policy loginPolicy,
) (concurrencyResult, error) {
	var result concurrencyResult

	// Handle non-concurrent login 处理不允许并发登录。
	if !policy.isConcurrent {
		// Reject new device when configured 配置为拒绝新设备时直接校验。
		if policy.replacedLoginExitMode == config.ReplacedLoginExitModeNewDevice {
			// Select terminals by concurrency scope and reject login only if an active terminal exists. 按并发作用域选择终端，仅在存在活跃终端时拒绝新登录。
			var terminals []TerminalInfo
			if m.config.ConcurrencyScope == config.ConcurrencyScopeAccount {
				terminals = sess.TerminalInfos
			} else if m.config.ConcurrencyScope == config.ConcurrencyScopeDevice {
				terminals = sessionGetTerminalsByDevice(sess, device)
			}

			// Check active terminal 检查是否存在活跃终端。
			hasActiveTerminal, activeErr := m.hasActiveTerminal(ctx, terminals, sess)
			if activeErr != nil {
				return result, activeErr
			}

			// Reject login when active terminal exists 存在活跃终端时拒绝登录。
			if hasActiveTerminal {
				return result, derror.ErrLoginLimitExceeded
			}

			// Allow login when no active terminal 无活跃终端时允许继续登录。
			return result, nil
		}

		// Replace existing terminals within the configured scope when concurrency is disabled. 禁止并发登录时，按配置作用域顶替已有终端。
		var err error
		result, err = m.cleanConcurrencyScope(ctx, sess, device)
		if err != nil {
			return result, err
		}
		if m.config.ConcurrencyScope == config.ConcurrencyScopeAccount {
			destroyedByReplace, removed, err := m.removeTerminalInfosAndTokens(ctx, sess, config.LogoutModeReplaced)
			addTerminalLifecycleEvents(&result, removed, TokenStateReplaced)
			if err != nil {
				return result, err
			} else {
				result.destroyedSession = result.destroyedSession || destroyedByReplace
			}
		} else if m.config.ConcurrencyScope == config.ConcurrencyScopeDevice {
			destroyedByReplace, removed, err := m.removeTerminalInfosAndTokens(ctx, sess, config.LogoutModeReplaced, device)
			addTerminalLifecycleEvents(&result, removed, TokenStateReplaced)
			if err != nil {
				return result, err
			} else {
				result.destroyedSession = result.destroyedSession || destroyedByReplace
			}
		}
		result.handled = true
		return result, nil
	}

	// Try token sharing when enabled 开启共享时尝试复用 Token。
	if policy.isShare {
		// Try token sharing reuse only within the same device dimension. 仅在相同设备维度内尝试复用 Token。
		token, shareErr := m.getTokenAndShare(ctx, sess, device, deviceID)
		if shareErr != nil {
			return result, shareErr
		}
		if token != "" {
			limitResult, limitErr := m.enforceSharedLoginLimit(ctx, sess, token, policy)
			result.terminalEvents = append(result.terminalEvents, limitResult.terminalEvents...)
			if limitErr != nil {
				return result, limitErr
			}
			result.reuseToken = token
			result.handled = true
			return result, nil
		}
	}

	// Verify live slots only when the recorded count can reach the limit. 仅在记录数量可能触达上限时核实有效名额。
	if policy.maxLoginCount <= 0 || m.concurrencyTerminalCount(sess, device) < policy.maxLoginCount {
		return result, nil
	}
	var err error
	result, err = m.cleanConcurrencyScope(ctx, sess, device)
	if err != nil {
		return result, err
	}

	// Enforce account-level max login count 执行账号级最大登录数限制。
	if m.config.ConcurrencyScope == config.ConcurrencyScopeAccount {
		removedOverflow := false
		for policy.maxLoginCount > 0 && int64(len(sess.TerminalInfos)) >= policy.maxLoginCount {
			terminal, removed, err := m.removeOldestTerminalInfoAndToken(ctx, sess, policy.overflowLogoutMode)
			if err != nil {
				return result, err
			}
			if !removed {
				return result, derror.ErrLoginLimitExceeded
			}
			addTerminalLifecycleEvents(&result, []TerminalInfo{terminal}, tokenStateFromLogoutMode(policy.overflowLogoutMode))
			removedOverflow = true
		}
		if removedOverflow {
			result.handled = true
			return result, nil
		}
	} else if m.config.ConcurrencyScope == config.ConcurrencyScopeDevice {
		// Enforce device-level max login count 执行设备级最大登录数限制。
		removedOverflow := false
		for policy.maxLoginCount > 0 && m.concurrencyTerminalCount(sess, device) >= policy.maxLoginCount {
			terminal, removed, err := m.removeOldestTerminalInfoAndToken(ctx, sess, policy.overflowLogoutMode, device)
			if err != nil {
				return result, err
			}
			if !removed {
				return result, derror.ErrLoginLimitExceeded
			}
			addTerminalLifecycleEvents(&result, []TerminalInfo{terminal}, tokenStateFromLogoutMode(policy.overflowLogoutMode))
			removedOverflow = true
		}
		if removedOverflow {
			result.handled = true
			return result, nil
		}
	}

	// No concurrency action needed 无需并发处理。
	return result, nil
}

// enforceSharedLoginLimit retains the shared terminal and retires the oldest other terminals above the limit. enforceSharedLoginLimit 保留共享终端并按旧到新淘汰超额的其他终端。
func (m *Manager) enforceSharedLoginLimit(ctx context.Context, sess *Session, token string, policy loginPolicy) (concurrencyResult, error) {
	var result concurrencyResult
	var shared *TerminalInfo
	for i := range sess.TerminalInfos {
		if sess.TerminalInfos[i].Token == token {
			shared = &sess.TerminalInfos[i]
			break
		}
	}
	if shared == nil {
		return result, derror.ErrInvalidToken
	}
	if policy.maxLoginCount <= 0 {
		return result, nil
	}
	device := shared.Device
	if m.concurrencyTerminalCount(sess, device) <= policy.maxLoginCount {
		return result, nil
	}
	var err error
	result, err = m.cleanConcurrencyScope(ctx, sess, device)
	if err != nil {
		return result, err
	}
	if !sessionHasTerminalToken(sess, token) {
		return result, derror.ErrInvalidToken
	}

	// Snapshot the actual shared device scope before removals mutate the session slice. 移除操作修改会话切片前，按共享终端的实际设备类型保存作用域快照。
	terminals := sessionFilterTerminals(sess, func(terminal TerminalInfo) bool {
		return m.config.ConcurrencyScope == config.ConcurrencyScopeAccount || terminal.Device == device
	})
	count := int64(len(terminals))
	for _, terminal := range terminals {
		// Sharing adds no terminal and must keep the token returned by this login alive. 共享不新增终端，且必须保留本次登录返回的 Token。
		if count <= policy.maxLoginCount {
			break
		}
		if terminal.Token == token {
			continue
		}
		retired, err := m.retireConcurrencyTerminal(ctx, sess.LoginID, terminal, policy.overflowLogoutMode)
		if err != nil {
			return result, err
		}
		sessionRemoveTerminalByToken(sess, terminal.Token)
		addTerminalLifecycleEvents(&result, []TerminalInfo{retired}, tokenStateFromLogoutMode(policy.overflowLogoutMode))
		count--
	}
	return result, nil
}

// concurrencyTerminalCount counts recorded slots without probing token storage. concurrencyTerminalCount 统计作用域内的记录名额，不探测 Token 存储。
func (m *Manager) concurrencyTerminalCount(sess *Session, device string) int64 {
	if m.config.ConcurrencyScope == config.ConcurrencyScopeAccount {
		return int64(len(sess.TerminalInfos))
	}
	var count int64
	for _, terminal := range sess.TerminalInfos {
		if terminal.Device == device {
			count++
		}
	}
	return count
}

// cleanConcurrencyScope prunes only terminals that can affect this login policy. cleanConcurrencyScope 仅整理影响本次登录策略的终端。
func (m *Manager) cleanConcurrencyScope(ctx context.Context, sess *Session, device string) (concurrencyResult, error) {
	destroyed, expired, err := m.pruneExpiredTerminals(ctx, sess, func(terminal TerminalInfo) bool {
		return m.config.ConcurrencyScope == config.ConcurrencyScopeAccount || terminal.Device == device
	})
	result := concurrencyResult{destroyedSession: destroyed}
	addTerminalLifecycleEvents(&result, expired, TokenStateActiveTimeout)
	return result, err
}

// addTerminalLifecycleEvents appends terminal state changes to a concurrency result. addTerminalLifecycleEvents 向并发结果追加终端状态事件。
func addTerminalLifecycleEvents(result *concurrencyResult, terminals []TerminalInfo, state TokenState) {
	if result == nil || state == "" {
		return
	}
	for _, terminal := range terminals {
		if terminal.Token == "" {
			continue
		}
		result.terminalEvents = append(result.terminalEvents, terminalLifecycleEvent{terminal: terminal, state: state})
	}
}

// tokenStateFromLogoutMode maps overflow handling mode to terminal state. tokenStateFromLogoutMode 将超限处理模式映射为终端状态。
func tokenStateFromLogoutMode(mode config.LogoutMode) TokenState {
	switch mode {
	case config.LogoutModeLogout:
		return TokenStateLogout
	case config.LogoutModeKickout:
		return TokenStateKickOut
	case config.LogoutModeReplaced:
		return TokenStateReplaced
	default:
		return ""
	}
}

// getTokenAndShare retrieves and shares a token within one device dimension. getTokenAndShare 在同一设备维度内获取并共享 token。
func (m *Manager) getTokenAndShare(ctx context.Context, sess *Session, device, deviceID string) (string, error) {
	// Return when no terminals exist 没有终端时直接返回。
	if sess == nil || len(sess.TerminalInfos) == 0 {
		return "", nil
	}

	var terminalInfo TerminalInfo
	var tokenInfo *TokenInfo
	var record *tokenRecord
	// Filter newest entries in place and stop at the first usable candidate. 就地筛选最新条目，找到可用候选即停止。
	for i := len(sess.TerminalInfos) - 1; i >= 0; i-- {
		candidate := sess.TerminalInfos[i]
		if device != "" && candidate.Device != device || deviceID != "" && candidate.DeviceID != deviceID {
			continue
		}
		candidateRecord, err := m.getTokenRecord(ctx, candidate.Token)
		if err != nil {
			if isTokenInactiveError(err) {
				continue
			}
			return "", err
		}
		candidateInfo := &candidateRecord.TokenInfo

		// Require the session terminal and token mapping to describe the same lifecycle. 要求 Session 终端与 Token 映射描述同一生命周期。
		if candidateRecord.Revoked || !terminalMatchesTokenRecord(sess.LoginID, candidate, candidateRecord) {
			continue
		}

		alive, err := m.checkTerminalTokenAliveWithOptions(ctx, candidate.Token, candidateRecord, sess, nil, true, &candidate)
		if err != nil {
			return "", err
		}
		if alive {
			terminalInfo = candidate
			tokenInfo = candidateInfo
			record = candidateRecord
			break
		}
	}
	if terminalInfo.Token == "" {
		return "", nil
	}

	// Resolve reused token expiration 解析复用 Token 过期时间。
	expiration := m.resolveTokenExpiration(tokenInfo)

	// Renew session without shortening existing TTL 续期 session，避免缩短已有 TTL
	if err := m.renewSessionTTL(ctx, m.getSessionKey(tokenInfo.LoginID), expiration); err != nil {
		return "", err
	}

	// Renew the original token mapping without rebuilding identity from session metadata. 续期原 Token 映射，不使用 Session 元数据重建身份。
	if err := m.saveTokenRecord(ctx, terminalInfo.Token, record, expiration); err != nil {
		return "", err
	}

	// Shared login supersedes activity captured by older queued validation tasks. 共享登录取代旧校验任务捕获的活跃时间，避免后续回写倒退。
	m.cancelLoginMaintenance(terminalInfo.Token)

	// Renew or reset metadata 续期或重置元数据
	if m.config.RenewInterval > 0 {
		// Refresh renew marker 刷新续期标记。
		if err := m.setRenewMarker(ctx, terminalInfo.Token, expiration); err != nil {
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
