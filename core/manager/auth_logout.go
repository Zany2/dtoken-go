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
	// Validate token value 校验 Token 值。
	if tokenValue == "" {
		return derror.ErrInvalidToken
	}

	// Load session by token 根据 Token 加载会话。
	sess, err := m.GetSessionByToken(ctx, tokenValue)
	if err != nil {
		// Treat inactive token errors as idempotent success 已下线 token 视为幂等成功
		if isTokenInactiveError(err) {
			return nil
		}
		return err
	}

	// Remove the matched terminal 移除命中的终端。
	return m.logoutTerminals(ctx, sess.LoginID, func(sess *Session) []TerminalInfo {
		if info, ok := sess.removeTerminalByToken(tokenValue); ok {
			return []TerminalInfo{info}
		}
		return nil
	})
}

// LogoutByDevice logs out all terminals of a specific device type. LogoutByDevice 根据设备类型登出所有该设备的终端。
func (m *Manager) LogoutByDevice(ctx context.Context, loginID string, device string) error {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}
	// Normalize device type 规范化设备类型。
	device = strings.TrimSpace(device)
	// Validate device type 校验设备类型。
	if device == "" {
		return derror.ErrInvalidParam
	}

	// Remove terminals by device type 按设备类型移除终端。
	return m.logoutTerminals(ctx, loginID, func(sess *Session) []TerminalInfo {
		return sess.removeTerminalByDevice(device)
	})
}

// LogoutByDeviceAndDeviceId logs out a user by device type and device ID. LogoutByDeviceAndDeviceId 根据设备类型和设备ID登出用户。
func (m *Manager) LogoutByDeviceAndDeviceId(ctx context.Context, loginID string, deviceAndDeviceId ...string) error {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	// Parse device fields 解析设备字段。
	device, deviceId := m.getDeviceAndDeviceId(deviceAndDeviceId...)
	// Validate device fields 校验设备字段。
	if device == "" || deviceId == "" {
		return derror.ErrInvalidParam
	}
	// Remove terminals by concrete device 按具体设备移除终端。
	return m.logoutTerminals(ctx, loginID, func(sess *Session) []TerminalInfo {
		return sess.removeTerminalByDeviceAndDeviceId(device, deviceId)
	})
}

// LogoutByLoginID logs out all terminals for the specified loginID. LogoutByLoginID 登出指定 loginID 的所有终端。
func (m *Manager) LogoutByLoginID(ctx context.Context, loginID string) error {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	// Remove all terminals 移除全部终端。
	return m.logoutTerminals(ctx, loginID, func(sess *Session) []TerminalInfo {
		return sess.removeAllTerminals()
	})
}

// Kickout kicks out a user by token. Kickout 根据 Token 踢人下线。
func (m *Manager) Kickout(ctx context.Context, tokenValue string) error {
	// Validate token value 校验 Token 值。
	if tokenValue == "" {
		return derror.ErrInvalidToken
	}
	// Load session by token 根据 Token 加载会话。
	sess, err := m.GetSessionByToken(ctx, tokenValue)
	if err != nil {
		// Treat inactive token errors as idempotent success 已下线 token 视为幂等成功
		if isTokenInactiveError(err) {
			return nil
		}
		return err
	}

	// Mark matched terminal as kicked out 将命中终端标记为踢下线。
	return m.processTerminals(ctx, sess.LoginID, func(sess *Session) []TerminalInfo {
		if info, ok := sess.removeTerminalByToken(tokenValue); ok {
			return []TerminalInfo{info}
		}
		return nil
	}, TokenStateKickOut)
}

// KickoutByDevice kicks out all terminals of a specific device type. KickoutByDevice 根据设备类型踢人下线（踢掉该设备类型的所有终端）。
func (m *Manager) KickoutByDevice(ctx context.Context, loginID string, device string) error {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}
	// Normalize device type 规范化设备类型。
	device = strings.TrimSpace(device)
	// Validate device type 校验设备类型。
	if device == "" {
		return derror.ErrInvalidParam
	}

	// Mark device terminals as kicked out 将设备终端标记为踢下线。
	return m.processTerminals(ctx, loginID, func(sess *Session) []TerminalInfo {
		return sess.removeTerminalByDevice(device)
	}, TokenStateKickOut)
}

// KickoutByDeviceAndDeviceId kicks out a user by device type and device ID. KickoutByDeviceAndDeviceId 根据设备类型和设备ID踢人下线。
func (m *Manager) KickoutByDeviceAndDeviceId(ctx context.Context, loginID string, deviceAndDeviceId ...string) error {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	// Parse device fields 解析设备字段。
	device, deviceId := m.getDeviceAndDeviceId(deviceAndDeviceId...)
	// Validate device fields 校验设备字段。
	if device == "" || deviceId == "" {
		return derror.ErrInvalidParam
	}

	// Mark concrete device terminal as kicked out 将具体设备终端标记为踢下线。
	return m.processTerminals(ctx, loginID, func(sess *Session) []TerminalInfo {
		return sess.removeTerminalByDeviceAndDeviceId(device, deviceId)
	}, TokenStateKickOut)
}

// KickoutByLoginID kicks out all terminals for the specified loginID. KickoutByLoginID 踢出指定 loginID 的所有终端。
func (m *Manager) KickoutByLoginID(ctx context.Context, loginID string) error {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	// Mark all terminals as kicked out 将全部终端标记为踢下线。
	return m.processTerminals(ctx, loginID, func(sess *Session) []TerminalInfo {
		return sess.removeAllTerminals()
	}, TokenStateKickOut)
}

// Replace replaces a user session by token. Replace 根据 Token 顶人下线。
func (m *Manager) Replace(ctx context.Context, tokenValue string) error {
	// Validate token value 校验 Token 值。
	if tokenValue == "" {
		return derror.ErrInvalidToken
	}
	// Load session by token 根据 Token 加载会话。
	sess, err := m.GetSessionByToken(ctx, tokenValue)
	if err != nil {
		// Treat inactive token errors as idempotent success 已下线 token 视为幂等成功
		if isTokenInactiveError(err) {
			return nil
		}
		return err
	}

	// Mark matched terminal as replaced 将命中终端标记为顶下线。
	return m.processTerminals(ctx, sess.LoginID, func(sess *Session) []TerminalInfo {
		if info, ok := sess.removeTerminalByToken(tokenValue); ok {
			return []TerminalInfo{info}
		}
		return nil
	}, TokenStateReplaced)
}

// ReplaceByDevice replaces all terminals of a specific device type. ReplaceByDevice 根据设备类型顶人下线（顶掉该设备类型的所有终端）。
func (m *Manager) ReplaceByDevice(ctx context.Context, loginID string, device string) error {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}
	// Normalize device type 规范化设备类型。
	device = strings.TrimSpace(device)
	// Validate device type 校验设备类型。
	if device == "" {
		return derror.ErrInvalidParam
	}

	// Mark device terminals as replaced 将设备终端标记为顶下线。
	return m.processTerminals(ctx, loginID, func(sess *Session) []TerminalInfo {
		return sess.removeTerminalByDevice(device)
	}, TokenStateReplaced)
}

// ReplaceByDeviceAndDeviceId replaces a user session by device type and device ID. ReplaceByDeviceAndDeviceId 根据设备类型和设备ID顶人下线。
func (m *Manager) ReplaceByDeviceAndDeviceId(ctx context.Context, loginID string, deviceAndDeviceId ...string) error {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	// Parse device fields 解析设备字段。
	device, deviceId := m.getDeviceAndDeviceId(deviceAndDeviceId...)
	// Validate device fields 校验设备字段。
	if device == "" || deviceId == "" {
		return derror.ErrInvalidParam
	}
	// Mark concrete device terminal as replaced 将具体设备终端标记为顶下线。
	return m.processTerminals(ctx, loginID, func(sess *Session) []TerminalInfo {
		return sess.removeTerminalByDeviceAndDeviceId(device, deviceId)
	}, TokenStateReplaced)
}

// ReplaceByLoginID replaces all terminals for the specified loginID. ReplaceByLoginID 顶替指定 loginID 的所有终端。
func (m *Manager) ReplaceByLoginID(ctx context.Context, loginID string) error {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	// Mark all terminals as replaced 将全部终端标记为顶下线。
	return m.processTerminals(ctx, loginID, func(sess *Session) []TerminalInfo {
		return sess.removeAllTerminals()
	}, TokenStateReplaced)
}

// removeOldestTerminalInfoAndToken removes the oldest terminal and its token. removeOldestTerminalInfoAndToken 移除最旧的终端信息并按模式处理 Token。
func (m *Manager) removeOldestTerminalInfoAndToken(ctx context.Context, sess *Session, mode config.LogoutMode, device ...string) error {
	// Remove oldest terminal 移除最旧终端。
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

// removeTerminalInfosAndTokens removes terminal information and tokens. removeTerminalInfosAndTokens 移除终端信息和 Token。
func (m *Manager) removeTerminalInfosAndTokens(ctx context.Context, sess *Session, mode config.LogoutMode, device ...string) (bool, error) {
	// Prepare removed terminals 准备被移除终端列表。
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
	// Collect removed tokens 收集被移除 Token。
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

// logoutTerminals performs common logout logic. logoutTerminals 通用登出逻辑：移除终端 + 删除 token + 清理 metadata。
func (m *Manager) logoutTerminals(
	ctx context.Context,
	loginID string,
	removalFunc func(*Session) []TerminalInfo,
) error {
	// Lock account writes 锁定账号写操作。
	unlock := m.lockLoginWrite(loginID)
	// Release lock on function exit 函数退出时释放锁。
	defer func() { unlock() }()

	// Load session 加载会话。
	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		// Ignore missing session 忽略不存在的会话。
		if errors.Is(err, derror.ErrSessionNotFound) {
			return nil
		}
		return err
	}
	// Treat nil session as no-op 空会话视为无操作。
	if sess == nil {
		return nil // session 不存在，登出无害
	}

	// Apply terminal removal strategy 执行终端移除策略。
	removed := removalFunc(sess)
	// Return when nothing removed 没有移除项时直接返回。
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

	// Release lock before events 触发事件前释放锁。
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

// cleanTokenMetadata cleans token metadata in batch. cleanTokenMetadata 批量清理 token 的附属元数据（续期 key、活跃时间 key）。
func (m *Manager) cleanTokenMetadata(ctx context.Context, tokens []string) error {
	// Return when token list is empty Token 列表为空时直接返回。
	if len(tokens) == 0 {
		return nil
	}

	// Build metadata keys 构建元数据键。
	keys := make([]string, 0, len(tokens)*2)
	for _, token := range tokens {
		keys = append(keys, m.getRenewKey(token), m.getActiveKey(token))
	}

	// Delete metadata keys 删除元数据键。
	if err := m.storage.Delete(ctx, keys...); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	for _, token := range tokens {
		if err := m.cleanRefreshTokenByAccessToken(ctx, token); err != nil {
			return err
		}
	}

	// Return cleanup success 返回清理成功。
	return nil
}

// TerminalRemovalFunc defines how to remove terminals from a session. TerminalRemovalFunc 定义如何从 Session 中移除终端。
type TerminalRemovalFunc func(sess *Session) []TerminalInfo

// processTerminals performs common terminal processing logic. processTerminals 通用终端处理逻辑。
func (m *Manager) processTerminals(
	ctx context.Context,
	loginID string,
	removalFunc TerminalRemovalFunc,
	state TokenState,
) error {
	// Lock account writes 锁定账号写操作。
	unlock := m.lockLoginWrite(loginID)
	// Release lock on function exit 函数退出时释放锁。
	defer func() { unlock() }()

	// Load session 加载 Session
	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		// Ignore missing session 忽略不存在的会话。
		if errors.Is(err, derror.ErrSessionNotFound) {
			return nil
		}
		return err
	}

	// Apply removal strategy 执行移除策略
	removedTerminals := removalFunc(sess)

	// Clean each removed token 对每个被移除的 token 执行清理
	for _, info := range removedTerminals {
		// Read removed token 读取被移除 Token。
		token := info.Token

		// Set token state 设置 token 状态
		if err = m.setTokenState(ctx, token, state, m.tokenStateExpiration(ctx, token)); err != nil {
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

		// Clean linked refresh token 清理关联的刷新令牌
		if err = m.cleanRefreshTokenByAccessToken(ctx, token); err != nil {
			return err
		}
	}

	// Track whether session is destroyed 跟踪会话是否被销毁。
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

	// Release lock before events 触发事件前释放锁。
	unlock()
	unlock = func() {}

	if destroySession {
		// Trigger session destroy event 触发销毁 Session 事件
		m.triggerEvent(listener.EventDestroySession, loginID, "", "", "", nil)
	}

	// Trigger matched event 触发对应事件
	// Resolve event by token state 根据 Token 状态解析事件。
	var event listener.Event
	switch state {
	case TokenStateKickOut, TokenStateActiveTimeout:
		event = listener.EventKickout
	case TokenStateReplaced:
		event = listener.EventReplace
	}

	if event != "" {
		// Trigger event for each removed terminal 为每个被移除终端触发事件。
		for _, info := range removedTerminals {
			m.triggerEvent(event, loginID, info.Device, info.DeviceId, info.Token, nil)
		}
	}

	return nil
}
