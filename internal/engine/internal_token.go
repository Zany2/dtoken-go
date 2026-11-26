// @Author daixk 2025/12/22 15:56:00
package engine

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Zany2/dtoken-go/config"
	adapter "github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
	"github.com/Zany2/dtoken-go/internal/utils"
)

// terminalDisableStateKey identifies one device or concrete-device disable scope. terminalDisableStateKey 标识一个设备或具体设备封禁作用域。
type terminalDisableStateKey struct {
	loginID  string // loginID stores the account identifier. loginID 存储账号标识。
	device   string // device stores the device type. device 存储设备类型。
	deviceID string // deviceID stores the concrete device identifier. deviceID 存储具体设备标识。
}

// terminalAliveCheckCache reuses disable checks within one aggregate query. terminalAliveCheckCache 在一次聚合查询中复用封禁检查结果。
type terminalAliveCheckCache struct {
	accountStates map[string]error                  // accountStates caches account disable results. accountStates 缓存账号封禁结果。
	deviceStates  map[terminalDisableStateKey]error // deviceStates caches device disable results. deviceStates 缓存设备封禁结果。
}

// tokenStateError maps stored token state to public errors. tokenStateError 将已存储 token 状态映射为公开错误。
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

// setTokenStateWithRecord reuses the extension owner checked by the caller under the account lock. setTokenStateWithRecord 复用调用方在账号锁内核对的扩展数据归属。
func (m *Manager) setTokenStateWithRecord(ctx context.Context, tokenValue string, record *tokenRecord, state TokenState, expiration time.Duration) error {
	// Save logical token state 保存 Token 逻辑状态。
	if err := m.storage.Set(ctx, m.getTokenKey(tokenValue), string(state), expiration); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	if record != nil && record.HasExtra {
		if err := m.storage.Delete(ctx, m.getTokenExtraKey(record.AccessID)); err != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
	}

	// Return state save success 返回状态保存成功。
	return nil
}

// tokenStateExpiration prefers current TTL and optionally reuses a checked record; an explicit nil means the mapping is inactive. tokenStateExpiration 优先使用当前 TTL，可复用已核对记录，显式 nil 表示映射已失效。
func (m *Manager) tokenStateExpiration(ctx context.Context, tokenValue string, knownRecord ...*tokenRecord) time.Duration {
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
	var record *tokenRecord
	if len(knownRecord) > 0 {
		record = knownRecord[0]
	} else {
		var err error
		record, err = m.getTokenRecord(ctx, tokenValue)
		if err != nil {
			return m.getExpiration()
		}
	}
	if record == nil {
		return m.getExpiration()
	}

	// Resolve token expiration 解析 Token 过期时间。
	return m.resolveTokenExpiration(&record.TokenInfo)
}

// applyLogoutModeToToken applies a logout mode using ownership checked under the account lock. applyLogoutModeToToken 复用账号锁内已核对的归属，按下线模式处理 Token 映射。
func (m *Manager) applyLogoutModeToToken(ctx context.Context, tokenValue string, record *tokenRecord, mode config.LogoutMode) error {
	// Apply mode by logout type 按下线模式处理。
	switch mode {
	case config.LogoutModeLogout:
		// Delete mapping for normal logout 普通登出直接删除映射。
		keys := []string{m.getTokenKey(tokenValue)}
		if record != nil && record.HasExtra {
			keys = append(keys, m.getTokenExtraKey(record.AccessID))
		}
		if err := m.storage.Delete(ctx, keys...); err != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
	case config.LogoutModeKickout:
		// Mark token as kicked out 标记 Token 为踢下线。
		return m.setTokenStateWithRecord(ctx, tokenValue, record, TokenStateKickOut, m.tokenStateExpiration(ctx, tokenValue, record))
	case config.LogoutModeReplaced:
		// Mark token as replaced 标记 Token 为顶下线。
		return m.setTokenStateWithRecord(ctx, tokenValue, record, TokenStateReplaced, m.tokenStateExpiration(ctx, tokenValue, record))
	default:
		return derror.ErrInvalidParam
	}

	// Return mode application success 返回模式处理成功。
	return nil
}

// hasActiveTerminal reports whether a terminal still occupies a login slot, including temporarily disabled devices. hasActiveTerminal 判断是否有终端仍占用登录名额，包括临时封禁设备。
func (m *Manager) hasActiveTerminal(ctx context.Context, terminals []TerminalInfo, sess *Session) (bool, error) {
	// Check each terminal 逐个检查终端。
	for i := len(terminals) - 1; i >= 0; i-- {
		terminal := &terminals[i]
		alive, err := m.checkTerminalTokenAliveWithOptions(ctx, terminal.Token, nil, sess, nil, false, terminal)
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

// checkTerminalTokenAlive checks token validity without renew side effects. checkTerminalTokenAlive 无续期副作用地检查 Token 是否有效。
func (m *Manager) checkTerminalTokenAlive(ctx context.Context, tokenValue string) (bool, error) {
	// Check token with loaded context fallback 使用可复用上下文检查 Token。
	return m.checkTerminalTokenAliveWithContext(ctx, tokenValue, nil, nil)
}

// checkTerminalTokenAliveWithContext checks token validity with optional loaded data. checkTerminalTokenAliveWithContext 使用可选已加载数据检查 Token 是否有效。
func (m *Manager) checkTerminalTokenAliveWithContext(ctx context.Context, tokenValue string, record *tokenRecord, sess *Session) (bool, error) {
	return m.checkTerminalTokenAliveWithCache(ctx, tokenValue, record, sess, nil)
}

// checkTerminalTokenStructurallyAliveWithContext checks token validity without applying reversible disable rules. checkTerminalTokenStructurallyAliveWithContext 检查 Token 结构性有效性，但不应用可逆封禁规则。
func (m *Manager) checkTerminalTokenStructurallyAliveWithContext(ctx context.Context, tokenValue string, record *tokenRecord, sess *Session) (bool, error) {
	return m.checkTerminalTokenAliveWithOptions(ctx, tokenValue, record, sess, nil, false, nil)
}

// checkTerminalEntryAlive validates a queried terminal against its current token lifecycle. checkTerminalEntryAlive 将被查询终端与当前 Token 生命周期核对后判断存活。
func (m *Manager) checkTerminalEntryAlive(ctx context.Context, terminal TerminalInfo, sess *Session, cache *terminalAliveCheckCache) (bool, error) {
	return m.checkTerminalTokenAliveWithOptions(ctx, terminal.Token, nil, sess, cache, true, &terminal)
}

// checkTerminalTokenAliveWithCache checks token validity and reuses disable-state lookups. checkTerminalTokenAliveWithCache 检查 Token 有效性并复用封禁状态查询。
func (m *Manager) checkTerminalTokenAliveWithCache(ctx context.Context, tokenValue string, record *tokenRecord, sess *Session, cache *terminalAliveCheckCache) (bool, error) {
	return m.checkTerminalTokenAliveWithOptions(ctx, tokenValue, record, sess, cache, true, nil)
}

// checkTerminalTokenAliveWithOptions checks token validity with configurable disable-state enforcement. checkTerminalTokenAliveWithOptions 按可配置的封禁状态规则检查 Token 有效性。
func (m *Manager) checkTerminalTokenAliveWithOptions(ctx context.Context, tokenValue string, record *tokenRecord, sess *Session, cache *terminalAliveCheckCache, checkDisableState bool, terminal *TerminalInfo) (bool, error) {
	var err error

	// Load token info when caller has not provided it 调用方未提供时加载 Token 信息。
	if record == nil {
		record, err = m.getTokenRecord(ctx, tokenValue)
	}
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

	// Retired credentials remain invalid even if an old session snapshot is available. 即使仍有旧会话快照，已废弃凭证也不能恢复有效。
	if record.Revoked {
		return false, nil
	}
	tokenInfo := &record.TokenInfo

	// Reject empty account 拒绝空账号。
	if tokenInfo.LoginID == "" {
		return false, nil
	}

	// A supplied entry avoids rescanning the same session list during aggregate queries. 已提供终端条目时避免聚合查询重复扫描同一会话列表。
	if terminal != nil {
		if sess == nil || terminal.Token != tokenValue || !terminalMatchesTokenRecord(sess.LoginID, *terminal, record) {
			return false, nil
		}
	}

	// Reject disabled account or device for ordinary alive checks. 普通存活检查拒绝已封禁账号或设备。
	if checkDisableState {
		if err := m.checkLoginDisableStateCached(ctx, tokenInfo.LoginID, tokenInfo.Device, tokenInfo.DeviceID, cache); err != nil {
			if errors.Is(err, derror.ErrAccountDisabled) || errors.Is(err, derror.ErrDeviceDisabled) {
				return false, nil
			}
			return false, err
		}
	}

	// Reject foreign token when caller provided a session. 传入会话时拒绝不属于该会话的 Token。
	if sess != nil && sess.LoginID != tokenInfo.LoginID {
		return false, nil
	}

	// Load session 加载会话。
	if sess == nil {
		sess, err = m.getSession(ctx, tokenInfo.LoginID)
	}
	if err != nil || sess == nil || (terminal == nil && !sessionHasTerminalToken(sess, tokenValue)) {
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
	pendingActiveAt := m.pendingLoginActiveAt(tokenValue, record)
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
	if err != nil || timeStamp < 0 {
		return false, nil
	}

	// Compare active timeout 比较活跃超时。
	timeStamp = max(timeStamp, pendingActiveAt)
	return time.Now().Unix()-timeStamp <= activeTimeout, nil
}

// checkLoginDisableStateCached checks disable state with an optional aggregate-query cache. checkLoginDisableStateCached 使用可选的聚合查询缓存检查封禁状态。
func (m *Manager) checkLoginDisableStateCached(ctx context.Context, loginID, device, deviceID string, cache *terminalAliveCheckCache) error {
	if cache == nil {
		return m.checkLoginDisableState(ctx, loginID, device, deviceID)
	}

	// Check one account marker at most once during the aggregate query. 一次聚合查询中最多检查一次账号封禁标记。
	if cache.accountStates == nil {
		cache.accountStates = make(map[string]error)
	}
	accountErr, checked := cache.accountStates[loginID]
	if !checked {
		accountErr = m.CheckDisable(ctx, loginID)
		cache.accountStates[loginID] = accountErr
	}
	if accountErr != nil {
		return accountErr
	}

	// Check each normalized device-type scope at most once. 每个规范化的设备类型作用域最多检查一次。
	device = strings.TrimSpace(device)
	deviceID = strings.TrimSpace(deviceID)
	if device == "" {
		return nil
	}
	if cache.deviceStates == nil {
		cache.deviceStates = make(map[terminalDisableStateKey]error)
	}
	deviceKey := terminalDisableStateKey{loginID: loginID, device: device}
	deviceErr, checked := cache.deviceStates[deviceKey]
	if !checked {
		deviceErr = m.CheckDisableDevice(ctx, loginID, device)
		cache.deviceStates[deviceKey] = deviceErr
	}
	if deviceErr != nil || deviceID == "" {
		return deviceErr
	}

	// Check each concrete device scope at most once after its device type passes. 设备类型检查通过后，每个具体设备作用域最多检查一次。
	deviceIDKey := terminalDisableStateKey{loginID: loginID, device: device, deviceID: deviceID}
	deviceIDErr, checked := cache.deviceStates[deviceIDKey]
	if !checked {
		_, deviceIDErr = m.GetDisableDeviceAndDeviceIDInfo(ctx, loginID, device, deviceID)
		if deviceIDErr == nil {
			deviceIDErr = derror.ErrDeviceDisabled
		} else if errors.Is(deviceIDErr, derror.ErrDeviceNotDisabled) {
			deviceIDErr = nil
		}
		cache.deviceStates[deviceIDKey] = deviceIDErr
	}
	return deviceIDErr
}
