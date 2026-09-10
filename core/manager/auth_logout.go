// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/listener"
)

// Logout logs out a user by token. Logout 根据 Token 登出用户。
func (m *Manager) Logout(ctx context.Context, tokenValue string) error {
	// Validate token value 校验 Token 值。
	if tokenValue == "" {
		return derror.ErrInvalidToken
	}

	// Capture the original lifecycle before validation so a reused token cannot be logged out later. 校验前捕获原生命周期，避免后续误登出已复用的 Token。
	expected, err := m.getTokenRecord(ctx, tokenValue)
	if err != nil {
		if isTokenInactiveError(err) {
			return nil
		}
		return err
	}
	checkBinding := tokenRecordBindingCheck(m, ctx, tokenValue, expected)
	removeToken := terminalRemovalForTokenRecord(tokenValue, expected)

	// Keep the regular validation path so inactive token states retain their existing semantics. 保留常规校验路径，确保非活跃 Token 状态维持既有语义。
	_, _, err = m.checkLoginAndGetContextNoRenew(ctx, tokenValue)
	if err == nil {
		return m.logoutTerminalsIf(ctx, expected.LoginID, checkBinding, removeToken, terminalInfoFromTokenRecord(tokenValue, expected))
	}
	if isTokenInactiveError(err) {
		return nil
	}
	if !errors.Is(err, derror.ErrAccountDisabled) && !errors.Is(err, derror.ErrDeviceDisabled) {
		return err
	}

	if expected.LoginID == "" {
		return nil
	}

	// Remove the matched terminal, or clean the detached token when account disable already removed its session. 移除命中的终端；账号封禁已删除 Session 时清理脱离会话的 Token。
	return m.logoutTerminalsIf(ctx, expected.LoginID, checkBinding, removeToken, terminalInfoFromTokenRecord(tokenValue, expected))
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

// LogoutByDeviceAndDeviceID logs out a user by device type and device ID. LogoutByDeviceAndDeviceID 根据设备类型和设备ID登出用户。
// Exactly two device arguments are required. 必须提供设备类型和设备 ID 两个参数。
func (m *Manager) LogoutByDeviceAndDeviceID(ctx context.Context, loginID string, deviceAndDeviceID ...string) error {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	// Reject ambiguous device arguments before any terminal mutation. 修改终端前拒绝含糊的设备参数。
	if len(deviceAndDeviceID) != 2 {
		return derror.ErrInvalidParam
	}

	// Parse device fields 解析设备字段。
	device, deviceID := m.getDeviceAndDeviceID(deviceAndDeviceID...)

	// Validate device fields 校验设备字段。
	if device == "" || deviceID == "" {
		return derror.ErrInvalidParam
	}

	// Remove terminals by concrete device 按具体设备移除终端。
	return m.logoutTerminals(ctx, loginID, func(sess *Session) []TerminalInfo {
		return sess.removeTerminalByDeviceAndDeviceID(device, deviceID)
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

	// Capture the original lifecycle before validation and account-lock acquisition. 校验及获取账号锁前捕获原生命周期。
	expected, err := m.getTokenRecord(ctx, tokenValue)
	if err != nil {
		if isTokenInactiveError(err) {
			return nil
		}
		return err
	}

	// Validate token and retain metadata for detached-terminal fallback. 校验 Token，并保留元数据用于终端记录缺失时回退。
	_, _, err = m.checkLoginAndGetContextNoRenew(ctx, tokenValue)
	if err != nil {
		// Treat inactive token errors as idempotent success 已下线 token 视为幂等成功
		if isTokenInactiveError(err) {
			return nil
		}
		return err
	}

	// Mark matched terminal as kicked out 将命中终端标记为踢下线。
	return m.processTerminalsIf(ctx, expected.LoginID, tokenRecordBindingCheck(m, ctx, tokenValue, expected),
		terminalRemovalForTokenRecord(tokenValue, expected), TokenStateKickOut, terminalInfoFromTokenRecord(tokenValue, expected))
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

// KickoutByDeviceAndDeviceID kicks out a user by device type and device ID. KickoutByDeviceAndDeviceID 根据设备类型和设备ID踢人下线。
// Exactly two device arguments are required. 必须提供设备类型和设备 ID 两个参数。
func (m *Manager) KickoutByDeviceAndDeviceID(ctx context.Context, loginID string, deviceAndDeviceID ...string) error {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	// Reject ambiguous device arguments before any terminal mutation. 修改终端前拒绝含糊的设备参数。
	if len(deviceAndDeviceID) != 2 {
		return derror.ErrInvalidParam
	}

	// Parse device fields 解析设备字段。
	device, deviceID := m.getDeviceAndDeviceID(deviceAndDeviceID...)

	// Validate device fields 校验设备字段。
	if device == "" || deviceID == "" {
		return derror.ErrInvalidParam
	}

	// Mark concrete device terminal as kicked out 将具体设备终端标记为踢下线。
	return m.processTerminals(ctx, loginID, func(sess *Session) []TerminalInfo {
		return sess.removeTerminalByDeviceAndDeviceID(device, deviceID)
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

	// Capture the original lifecycle before validation and account-lock acquisition. 校验及获取账号锁前捕获原生命周期。
	expected, err := m.getTokenRecord(ctx, tokenValue)
	if err != nil {
		if isTokenInactiveError(err) {
			return nil
		}
		return err
	}

	// Validate token and retain metadata for detached-terminal fallback. 校验 Token，并保留元数据用于终端记录缺失时回退。
	_, _, err = m.checkLoginAndGetContextNoRenew(ctx, tokenValue)
	if err != nil {
		// Treat inactive token errors as idempotent success 已下线 token 视为幂等成功
		if isTokenInactiveError(err) {
			return nil
		}
		return err
	}

	// Mark matched terminal as replaced 将命中终端标记为顶下线。
	return m.processTerminalsIf(ctx, expected.LoginID, tokenRecordBindingCheck(m, ctx, tokenValue, expected),
		terminalRemovalForTokenRecord(tokenValue, expected), TokenStateReplaced, terminalInfoFromTokenRecord(tokenValue, expected))
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

// ReplaceByDeviceAndDeviceID replaces a user session by device type and device ID. ReplaceByDeviceAndDeviceID 根据设备类型和设备ID顶人下线。
// Exactly two device arguments are required. 必须提供设备类型和设备 ID 两个参数。
func (m *Manager) ReplaceByDeviceAndDeviceID(ctx context.Context, loginID string, deviceAndDeviceID ...string) error {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	// Reject ambiguous device arguments before any terminal mutation. 修改终端前拒绝含糊的设备参数。
	if len(deviceAndDeviceID) != 2 {
		return derror.ErrInvalidParam
	}

	// Parse device fields 解析设备字段。
	device, deviceID := m.getDeviceAndDeviceID(deviceAndDeviceID...)

	// Validate device fields 校验设备字段。
	if device == "" || deviceID == "" {
		return derror.ErrInvalidParam
	}

	// Mark concrete device terminal as replaced 将具体设备终端标记为顶下线。
	return m.processTerminals(ctx, loginID, func(sess *Session) []TerminalInfo {
		return sess.removeTerminalByDeviceAndDeviceID(device, deviceID)
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
func (m *Manager) removeOldestTerminalInfoAndToken(ctx context.Context, sess *Session, mode config.LogoutMode, device ...string) (TerminalInfo, bool, error) {
	// Remove oldest terminal 移除最旧终端。
	terminalInfo, ok := sess.removeOldestTerminal(device...)
	if ok {
		// Skip a token value that now belongs to another lifecycle while still retiring the stale terminal. Token 值已属于其他生命周期时仅移除陈旧终端。
		cleanupAllowed, err := m.terminalTokenCleanupAllowed(ctx, sess.LoginID, terminalInfo)
		if err != nil {
			return TerminalInfo{}, false, err
		}
		if cleanupAllowed {
			// Apply overflow mode 应用超限处理模式
			if err = m.applyLogoutModeToToken(ctx, terminalInfo.Token, mode); err != nil {
				return TerminalInfo{}, false, err
			}

			// Clean metadata 清理 metadata
			if err = m.cleanTokenMetadata(ctx, []string{terminalInfo.Token}); err != nil {
				return TerminalInfo{}, false, err
			}
		}

		// Save session data 保存会话数据
		if err := m.saveToStorage(ctx, m.getSessionKey(sess.LoginID), *sess); err != nil {
			return TerminalInfo{}, false, err
		}
		if !cleanupAllowed {
			return TerminalInfo{}, true, nil
		}
		return terminalInfo, true, nil
	}
	return TerminalInfo{}, false, nil
}

// removeTerminalInfosAndTokens removes terminal information and tokens. removeTerminalInfosAndTokens 移除终端信息和 Token。
func (m *Manager) removeTerminalInfosAndTokens(ctx context.Context, sess *Session, mode config.LogoutMode, device ...string) (bool, []TerminalInfo, error) {
	// Prepare removed terminals 准备被移除终端列表。
	var terminalInfos []TerminalInfo
	if len(device) > 0 {
		// Remove terminals for specified device 移除指定设备类型的终端信。
		terminalInfos = sess.removeTerminalByDevice(device[0])
	} else {
		// Remove all terminals 移除所有终端信。
		terminalInfos = sess.removeAllTerminals()
	}
	if len(terminalInfos) == 0 {
		return false, nil, nil
	}

	// Apply mode only to terminal entries that still own their token lifecycle. 仅处理仍属于对应 Token 生命周期的终端条目。
	processedTerminals := make([]TerminalInfo, 0, len(terminalInfos))
	for _, terminalInfo := range terminalInfos {
		cleanupAllowed, err := m.terminalTokenCleanupAllowed(ctx, sess.LoginID, terminalInfo)
		if err != nil {
			return false, nil, err
		}
		if !cleanupAllowed {
			continue
		}
		if err = m.applyLogoutModeToToken(ctx, terminalInfo.Token, mode); err != nil {
			return false, nil, err
		}
		processedTerminals = append(processedTerminals, terminalInfo)
	}

	// Clean token metadata 清理附属 metadata
	// Collect removed tokens 收集被移除 Token。
	tokens := make([]string, len(processedTerminals))
	for i, info := range processedTerminals {
		tokens[i] = info.Token
	}
	if err := m.cleanTokenMetadata(ctx, tokens); err != nil {
		return false, nil, err
	}

	// Preserve the account session during replacement so the incoming login keeps account-level data. 顶替登录时保留账号 Session，使新登录继续持有账号级数据。
	destroyedSession := false
	if len(sess.TerminalInfos) == 0 && mode != config.LogoutModeReplaced {
		if err := m.storage.Delete(ctx, m.getSessionKey(sess.LoginID)); err != nil {
			return false, nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
		destroyedSession = true
	} else {
		// Save updated session otherwise 否则保存更新后的 session
		if err := m.saveToStorage(ctx, m.getSessionKey(sess.LoginID), *sess); err != nil {
			return false, nil, err
		}
	}

	return destroyedSession, processedTerminals, nil
}

// logoutTerminals performs common logout logic. logoutTerminals 通用登出逻辑：移除终。+ 删除 token + 清理 metadata。
func (m *Manager) logoutTerminals(
	ctx context.Context,
	loginID string,
	removalFunc func(*Session) []TerminalInfo,
	detachedTerminals ...TerminalInfo,
) error {
	return m.logoutTerminalsIf(ctx, loginID, nil, removalFunc, detachedTerminals...)
}

// logoutTerminalsIf checks an optional binding under the same account lock as cleanup. logoutTerminalsIf 在与清理相同的账号锁内检查可选绑定。
func (m *Manager) logoutTerminalsIf(
	ctx context.Context,
	loginID string,
	checkBinding func() (bool, error),
	removalFunc func(*Session) []TerminalInfo,
	detachedTerminals ...TerminalInfo,
) error {
	// Lock account writes 锁定账号写操作。
	unlock := m.lockLoginWrite(loginID)

	// Release lock on function exit 函数退出时释放锁。
	defer func() { unlock() }()

	// Reject stale ownership before either session removal or detached-token fallback. 移除 Session 终端或回退清理脱离终端的 Token 前拒绝陈旧归属。
	if checkBinding != nil {
		matched, err := checkBinding()
		if err != nil || !matched {
			return err
		}
	}

	// Load session 加载会话。
	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		// Keep processing detached tokens when the account session is already gone. 账号 Session 已不存在时继续处理脱离会话的 Token。
		if !errors.Is(err, derror.ErrSessionNotFound) {
			return err
		}
		sess = nil
	}

	// Apply terminal removal strategy when the account session still exists. 账号 Session 仍存在时执行终端移除策略。
	var removedCandidates []TerminalInfo
	sessionChanged := false
	if sess != nil {
		removedCandidates = removalFunc(sess)
		sessionChanged = len(removedCandidates) > 0
	}

	// Exclude stale entries whose token value now represents another active lifecycle. 排除 Token 值已代表其他活跃生命周期的陈旧条目。
	removed := make([]TerminalInfo, 0, len(removedCandidates))
	for _, terminal := range removedCandidates {
		cleanupAllowed, checkErr := m.terminalTokenCleanupAllowed(ctx, loginID, terminal)
		if checkErr != nil {
			return checkErr
		}
		if cleanupAllowed {
			removed = append(removed, terminal)
		}
	}

	// Fall back to detached token metadata when no terminal can be removed. 无法移除终端时回退到脱离会话的 Token 元数据。
	if len(removed) == 0 {
		removed, err = m.resolveDetachedTerminals(ctx, loginID, detachedTerminals)
		if err != nil {
			return err
		}
	}

	// Return only when neither token cleanup nor stale-session cleanup is needed. 无需清理 Token 或陈旧 Session 时才直接返回。
	if len(removed) == 0 && !sessionChanged {
		return nil
	}

	// Extract token list 提取 token 列表
	tokens := make([]string, len(removed))
	tokenKeys := make([]string, 0, len(removed))
	for i, info := range removed {
		tokens[i] = info.Token
		tokenKeys = append(tokenKeys, m.getTokenKey(info.Token))
	}

	// Delete token keys 删除 Token 键。
	if err = m.storage.Delete(ctx, tokenKeys...); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	// Clean token metadata 清理附属 metadata
	if err = m.cleanTokenMetadata(ctx, tokens); err != nil {
		return err
	}

	destroySession := false

	// Persist the account session only when the removal strategy changed it. 仅在移除策略修改账号 Session 时持久化。
	if sessionChanged {
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

	// Trigger logout event 触发登出事件
	for _, info := range removed {
		m.triggerEvent(listener.EventLogout, loginID, info.Device, info.DeviceID, info.Token, nil)
	}

	return nil
}

// cleanTokenMetadata cleans token metadata in batch. cleanTokenMetadata 批量清理 Token 的附属元数据，包括续期键和活跃时间键。
func (m *Manager) cleanTokenMetadata(ctx context.Context, tokens []string) error {
	// Return when token list is empty Token 列表为空时直接返回。
	if len(tokens) == 0 {
		return nil
	}

	// Build metadata keys 构建元数据键。
	keys := make([]string, 0, len(tokens)*2)
	for _, token := range tokens {
		if token == "" {
			continue
		}
		// Invalidate queued maintenance before removing token lifecycle metadata. 删除 Token 生命周期元数据前使排队中的维护任务失效。
		m.cancelLoginMaintenance(token)
		keys = append(keys, m.getRenewKey(token), m.getActiveKey(token))
	}

	// Delete metadata keys 删除元数据键。
	if len(keys) > 0 {
		if err := m.storage.Delete(ctx, keys...); err != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
	}

	for _, token := range tokens {
		if token == "" {
			continue
		}
		if err := m.cleanRefreshTokenByAccessToken(ctx, token); err != nil {
			return err
		}
	}

	// Return cleanup success 返回清理成功。
	return nil
}

// TerminalRemovalFunc defines how to remove terminals from a session. TerminalRemovalFunc 定义如何从 Session 中移除终端。
type TerminalRemovalFunc func(sess *Session) []TerminalInfo

// terminalInfoFromTokenRecord builds terminal metadata when the account session has no matching terminal. terminalInfoFromTokenRecord 在账号 Session 缺少对应终端时根据 Token 记录构建终端元数据。
func terminalInfoFromTokenRecord(tokenValue string, record *tokenRecord) TerminalInfo {
	if record == nil {
		return TerminalInfo{Token: tokenValue}
	}
	return TerminalInfo{
		Token:      tokenValue,
		LoginID:    record.LoginID,
		Device:     record.Device,
		DeviceID:   record.DeviceID,
		CreateTime: record.CreateTime,
		Index:      record.TerminalIndex,
	}
}

// terminalRemovalForTokenRecord removes only the terminal bound to an expected lifecycle. terminalRemovalForTokenRecord 仅移除绑定到预期生命周期的终端。
func terminalRemovalForTokenRecord(tokenValue string, record *tokenRecord) TerminalRemovalFunc {
	return func(sess *Session) []TerminalInfo {
		return sess.removeTerminals(func(terminal TerminalInfo) bool {
			return terminal.Token == tokenValue && terminalMatchesTokenRecord(sess.LoginID, terminal, record)
		})
	}
}

// tokenRecordBindingCheck creates a lock-held lifecycle recheck. tokenRecordBindingCheck 创建在锁内执行的生命周期复核函数。
func tokenRecordBindingCheck(m *Manager, ctx context.Context, tokenValue string, expected *tokenRecord) func() (bool, error) {
	return func() (bool, error) {
		return m.tokenRecordStillMatches(ctx, tokenValue, expected)
	}
}

// resolveDetachedTerminals rechecks detached token mappings under the account lock. resolveDetachedTerminals 在账号锁内重新确认脱离终端列表的 Token 映射。
func (m *Manager) resolveDetachedTerminals(ctx context.Context, loginID string, terminals []TerminalInfo) ([]TerminalInfo, error) {
	resolved := make([]TerminalInfo, 0, len(terminals))
	for _, terminal := range terminals {
		if terminal.Token == "" {
			continue
		}

		record, err := m.getTokenRecord(ctx, terminal.Token)
		if err != nil {
			if isTokenInactiveError(err) {
				continue
			}
			return nil, err
		}
		if record.LoginID != loginID {
			continue
		}
		if (terminal.Index > 0 || terminal.CreateTime > 0) && !terminalMatchesTokenRecord(loginID, terminal, record) {
			continue
		}
		resolved = append(resolved, terminalInfoFromTokenRecord(terminal.Token, record))
	}
	return resolved, nil
}

// cloneSessionForAliveCheck copies session slices used by alive checks. cloneSessionForAliveCheck 拷贝存活校验依赖的会话切片。
func cloneSessionForAliveCheck(sess *Session) Session {
	if sess == nil {
		return Session{}
	}
	clone := *sess
	clone.TerminalInfos = append([]TerminalInfo(nil), sess.TerminalInfos...)
	clone.Permissions = append([]string(nil), sess.Permissions...)
	clone.Roles = append([]string(nil), sess.Roles...)
	return clone
}

// processTerminals performs common terminal processing logic. processTerminals 通用终端处理逻辑。
func (m *Manager) processTerminals(
	ctx context.Context,
	loginID string,
	removalFunc TerminalRemovalFunc,
	state TokenState,
	detachedTerminals ...TerminalInfo,
) error {
	return m.processTerminalsIf(ctx, loginID, nil, removalFunc, state, detachedTerminals...)
}

// processTerminalsIf rechecks an optional lifecycle under the account lock before changing token state. processTerminalsIf 在账号锁内复核可选生命周期后再变更 Token 状态。
func (m *Manager) processTerminalsIf(
	ctx context.Context,
	loginID string,
	checkBinding func() (bool, error),
	removalFunc TerminalRemovalFunc,
	state TokenState,
	detachedTerminals ...TerminalInfo,
) error {
	// Lock account writes 锁定账号写操作。
	unlock := m.lockLoginWrite(loginID)

	// Release lock on function exit 函数退出时释放锁。
	defer func() { unlock() }()

	// Reject a stale direct request before either session or token lifecycle mutation. Session 或 Token 生命周期变更前拒绝陈旧的直接请求。
	if checkBinding != nil {
		matched, err := checkBinding()
		if err != nil || !matched {
			return err
		}
	}

	// Load session 加载 Session
	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		// Keep processing an explicitly supplied token when the account session disappeared concurrently. 账号 Session 并发消失时继续处理显式传入的 Token。
		if !errors.Is(err, derror.ErrSessionNotFound) {
			return err
		}
		sess = nil
	}

	// Apply the removal strategy only when the account session still exists. 仅在账号 Session 仍存在时执行终端移除策略。
	var originalSession *Session
	var removedCandidates []TerminalInfo
	sessionChanged := false
	if sess != nil {
		cloned := cloneSessionForAliveCheck(sess)
		originalSession = &cloned
		removedCandidates = removalFunc(sess)
		sessionChanged = len(removedCandidates) > 0
	}

	// Exclude stale entries whose token value now represents another active lifecycle. 排除 Token 值已代表其他活跃生命周期的陈旧条目。
	removedTerminals := make([]TerminalInfo, 0, len(removedCandidates))
	for _, terminal := range removedCandidates {
		cleanupAllowed, checkErr := m.terminalTokenCleanupAllowed(ctx, loginID, terminal)
		if checkErr != nil {
			return checkErr
		}
		if cleanupAllowed {
			removedTerminals = append(removedTerminals, terminal)
		}
	}

	// Recheck the token mapping when no terminal record can be removed. 无法移除终端记录时重新确认 Token 映射。
	usedDetachedFallback := false
	if len(removedTerminals) == 0 {
		removedTerminals, err = m.resolveDetachedTerminals(ctx, loginID, detachedTerminals)
		if err != nil {
			return err
		}
		usedDetachedFallback = len(removedTerminals) > 0
	}

	// Clean each removed token 对每个被移除的 token 执行清理
	transitionedTerminals := make([]TerminalInfo, 0, len(removedTerminals))
	for _, info := range removedTerminals {
		// Read removed token 读取被移除 Token。
		token := info.Token

		// Invalidate queued maintenance before changing the token lifecycle. 修改 Token 生命周期前使排队维护任务失效。
		m.cancelLoginMaintenance(token)

		// Active-timeout processing must persist its exact cause; other states apply to structurally alive tokens even while temporarily disabled. 不活跃超时必须保留精确原因；其他状态作用于结构性有效的 Token，即使其正处于临时封禁状态。
		shouldSetState := state == TokenStateActiveTimeout || usedDetachedFallback
		if !shouldSetState {
			alive, aliveErr := m.checkTerminalTokenStructurallyAliveWithContext(ctx, token, nil, originalSession)
			if aliveErr != nil {
				return aliveErr
			}
			shouldSetState = alive
		}
		if shouldSetState {
			// Set token state 设置 token 状。
			if err = m.setTokenState(ctx, token, state, m.tokenStateExpiration(ctx, token)); err != nil {
				return err
			}
			transitionedTerminals = append(transitionedTerminals, info)
		}

		// Delete renew key 删除续期 key
		if err = m.storage.Delete(ctx, m.getRenewKey(token)); err != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}

		// Delete active key 删除活跃时间 key
		if err = m.storage.Delete(ctx, m.getActiveKey(token)); err != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}

		// Clean linked refresh token 清理关联的刷新令。
		if err = m.cleanRefreshTokenByAccessToken(ctx, token); err != nil {
			return err
		}
	}

	// Track whether session is destroyed 跟踪会话是否被销毁。
	destroySession := false

	// Update session when terminals are removed 存在移除项时更新 Session
	if sessionChanged {
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
	case TokenStateKickOut:
		event = listener.EventKickout
	case TokenStateActiveTimeout:
		event = listener.EventActiveTimeout
	case TokenStateReplaced:
		event = listener.EventReplace
	}

	if event != "" {
		// Trigger events only for tokens that entered the requested state. 仅为实际进入目标状态的 Token 触发事件。
		for _, info := range transitionedTerminals {
			m.triggerEvent(event, loginID, info.Device, info.DeviceID, info.Token, nil)
		}
	}

	return nil
}
