// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"context"
	"errors"
	"fmt"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/listener"
	"strings"
)

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
