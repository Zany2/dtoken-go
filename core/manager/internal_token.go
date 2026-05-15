// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"context"
	"errors"
	"fmt"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/utils"
	"time"
)

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
