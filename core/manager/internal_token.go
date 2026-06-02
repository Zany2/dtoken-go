// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"context"
	"errors"
	"fmt"
	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/utils"
	"time"
)

// tokenStateError maps stored token state to public errors. tokenStateError 将已存 token 状态映射为公开错误。
func tokenStateError(state TokenState) error {
	// Map state to error 按状态映射错误。
	switch state {
	case TokenStateLogout:
		return derror.ErrInvalidToken
	case TokenStateKickOut:
		return derror.ErrTokenKickout
	case TokenStateReplaced:
		return derror.ErrTokenReplaced
	case TokenStateActiveTimeout:
		return derror.ErrActiveTimeout
	default:
		return nil
	}
}

// resolveActiveTimeoutFromSeconds resolves token active timeout. resolveActiveTimeoutFromSeconds 解析 Token 活跃超时秒数。
func (m *Manager) resolveActiveTimeoutFromSeconds(activeTimeout int64) int64 {
	if activeTimeout > 0 {
		return activeTimeout
	}
	if activeTimeout == config.NoLimit {
		return 0
	}
	return m.config.ActiveTimeout
}

// activeTimeoutToSeconds stores explicit active timeout override. activeTimeoutToSeconds 存储显式活跃超时覆盖值。
func (m *Manager) activeTimeoutToSeconds(activeTimeout time.Duration) int64 {
	if activeTimeout > 0 {
		return m.timeoutToSeconds(activeTimeout)
	}
	if activeTimeout < 0 {
		return config.NoLimit
	}
	return 0
}

// setTokenState marks token logical state. setTokenState 标记 Token 逻辑状态。
func (m *Manager) setTokenState(ctx context.Context, tokenValue string, state TokenState, expiration time.Duration) error {
	// Save logical token state 保存 Token 逻辑状态。
	if err := m.storage.Set(ctx, m.getTokenKey(tokenValue), string(state), expiration); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	// Remove legacy token key 删除历史 Token 键。
	if err := m.storage.Delete(ctx, m.getLegacyTokenKey(tokenValue)); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	// Return state save success 返回状态保存成功。
	return nil
}

// tokenStateExpiration resolves token state TTL. tokenStateExpiration 计算 Token 状态 TTL。
func (m *Manager) tokenStateExpiration(ctx context.Context, tokenValue string) time.Duration {
	// Prefer current token TTL 优先使用当前 Token TTL。
	if ttl, err := m.storage.TTL(ctx, m.getTokenKey(tokenValue)); err == nil {
		switch {
		case ttl == adapter.TTLNoExpire:
			return 0
		case ttl > 0:
			return ttl
		}
	}

	// Fallback to token info 回退到 Token 信息。
	tokenInfo, err := m.getTokenInfo(ctx, tokenValue)
	if err != nil {
		return m.getExpiration()
	}
	// Resolve token expiration 解析 Token 过期时间。
	return m.resolveTokenExpiration(tokenInfo)
}

// applyLogoutModeToToken applies logout mode to token mapping. applyLogoutModeToToken 按下线模式处理 Token 映射。
func (m *Manager) applyLogoutModeToToken(ctx context.Context, tokenValue string, mode config.LogoutMode) error {
	// Apply mode by logout type 按下线模式处理。
	switch mode {
	case config.LogoutModeLogout:
		// Delete mapping for normal logout 普通登出直接删除映射。
		if err := m.storage.Delete(ctx, m.getTokenStorageKeys(tokenValue)...); err != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
	case config.LogoutModeKickout:
		// Mark token as kicked out 标记 Token 为踢下线。
		return m.setTokenState(ctx, tokenValue, TokenStateKickOut, m.tokenStateExpiration(ctx, tokenValue))
	case config.LogoutModeReplaced:
		// Mark token as replaced 标记 Token 为顶下线。
		return m.setTokenState(ctx, tokenValue, TokenStateReplaced, m.tokenStateExpiration(ctx, tokenValue))
	default:
		return derror.ErrInvalidParam
	}
	// Return mode application success 返回模式处理成功。
	return nil
}

// getCheckedTokenSession gets token session after full login validation. getCheckedTokenSession 完整校验登录态后获取 Token 对应 Session。
func (m *Manager) getCheckedTokenSession(ctx context.Context, tokenValue string) (*Session, *TokenInfo, error) {
	// Validate login state 校验登录状态。
	if err := m.checkLoginInternal(ctx, tokenValue); err != nil {
		return nil, nil, err
	}

	// Load token info 加载 Token 信息。
	tokenInfo, err := m.getTokenInfo(ctx, tokenValue)
	if err != nil {
		return nil, nil, err
	}
	// Load session by token 根据 Token 加载会话。
	sess, err := m.GetSessionByToken(ctx, tokenValue)
	if err != nil {
		return nil, nil, err
	}
	// Return checked context 返回已校验上下文。
	return sess, tokenInfo, nil
}

// ensureTerminalTokenAlive ensures token is still alive without renew side effects. ensureTerminalTokenAlive 无续期副作用地确认 Token 仍有效。
func (m *Manager) ensureTerminalTokenAlive(ctx context.Context, tokenValue string) error {
	// Check token alive state 检查 Token 存活状态。
	alive, err := m.checkTerminalTokenAlive(ctx, tokenValue)
	if err != nil {
		return err
	}
	// Reject dead token 拒绝无效 Token。
	if !alive {
		return derror.ErrInvalidToken
	}
	// Return alive success 返回存活校验成功。
	return nil
}

// hasActiveTerminal reports whether any terminal is still alive. hasActiveTerminal 判断是否存在仍有效的终端。
func (m *Manager) hasActiveTerminal(ctx context.Context, terminals []TerminalInfo) (bool, error) {
	// Check each terminal 逐个检查终端。
	for _, terminal := range terminals {
		alive, err := m.checkTerminalTokenAlive(ctx, terminal.Token)
		if err != nil {
			return false, err
		}
		if alive {
			return true, nil
		}
	}
	// Return no active terminal 返回无活跃终端。
	return false, nil
}

// isTerminalTokenAlive checks token validity without renew side effects. isTerminalTokenAlive 无续期副作用地检查 Token 是否有效。
func (m *Manager) isTerminalTokenAlive(ctx context.Context, tokenValue string) bool {
	// Check token alive state 检查 Token 存活状态。
	alive, err := m.checkTerminalTokenAlive(ctx, tokenValue)
	// Return boolean result 返回布尔结果。
	return err == nil && alive
}

// checkTerminalTokenAlive checks token validity without renew side effects. checkTerminalTokenAlive 无续期副作用地检查 Token 是否有效。
func (m *Manager) checkTerminalTokenAlive(ctx context.Context, tokenValue string) (bool, error) {
	// Load token info 加载 Token 信息。
	tokenInfo, err := m.getTokenInfo(ctx, tokenValue)
	if err != nil {
		// Treat known token states as not alive 已知 Token 状态视为不存活。
		if errors.Is(err, derror.ErrInvalidToken) ||
			errors.Is(err, derror.ErrTokenExpired) ||
			errors.Is(err, derror.ErrActiveTimeout) ||
			errors.Is(err, derror.ErrTokenKickout) ||
			errors.Is(err, derror.ErrTokenReplaced) {
			return false, nil
		}
		return false, err
	}
	// Reject empty or disabled account 拒绝空账号或封禁账号。
	if tokenInfo.LoginID == "" || m.isDisable(ctx, tokenInfo.LoginID) {
		return false, nil
	}
	// Reject disabled device 拒绝已封禁设备。
	if m.isDisableDeviceMatch(ctx, tokenInfo.LoginID, tokenInfo.Device, tokenInfo.DeviceId) {
		return false, nil
	}

	// Load session 加载会话。
	sess, err := m.getSession(ctx, tokenInfo.LoginID)
	if err != nil || sess == nil || !sess.hasTerminalToken(tokenValue) {
		// Treat missing session as not alive 会话不存在视为不存活。
		if errors.Is(err, derror.ErrSessionNotFound) {
			return false, nil
		}
		return false, err
	}

	// Skip active timeout when disabled 未启用活跃超时时直接通过。
	activeTimeout := m.resolveActiveTimeoutFromSeconds(tokenInfo.ActiveTimeout)
	if activeTimeout <= 0 {
		return true, nil
	}

	// Load active timestamp 加载活跃时间戳。
	timeStampAny, err := m.storage.Get(ctx, m.getActiveKey(tokenValue))
	if err != nil || timeStampAny == nil {
		// Return storage errors 返回存储错误。
		if err != nil {
			return false, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
		return false, nil
	}
	// Convert active timestamp 转换活跃时间戳。
	timeStamp, err := utils.ToInt64(timeStampAny)
	if err != nil {
		return false, nil
	}
	// Compare active timeout 比较活跃超时。
	return time.Now().Unix()-timeStamp <= activeTimeout, nil
}
