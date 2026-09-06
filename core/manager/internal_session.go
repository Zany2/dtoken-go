// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/listener"
	"github.com/Zany2/dtoken-go/core/utils"
)

// getSession retrieves session information. getSession 获取会话信息。
func (m *Manager) getSession(ctx context.Context, loginID string) (*Session, error) {
	// Load session data 加载会话数据。
	sessData, err := m.storage.Get(ctx, m.getSessionKey(loginID))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	// Handle missing session 处理会话不存在。
	if sessData == nil {
		// Return session not found 返回会话不存在。
		return nil, derror.ErrSessionNotFound
	}

	// Convert storage value 转换存储值。
	bytesData, err := utils.ToBytes(sessData)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrTypeConvert, err)
	}

	// Decode session data 解码会话数据。
	var sess Session
	err = m.serializer.Decode(bytesData, &sess)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}

	// Treat the manager namespace and requested storage key as canonical identity. 以 Manager 命名空间和请求的存储键作为 Session 身份真值。
	sess.AuthType = m.config.AuthType
	sess.LoginID = loginID

	// Return session 返回会话。
	return &sess, nil
}

// getTokenInfo retrieves token information. getTokenInfo 获取 Token 信息。
func (m *Manager) getTokenInfo(ctx context.Context, tokenValue string) (*TokenInfo, error) {
	record, err := m.getTokenRecord(ctx, tokenValue)
	if err != nil {
		return nil, err
	}
	return &record.TokenInfo, nil
}

// tokenRecord preserves login identity without extending the public token metadata. tokenRecord 保留登录身份，但不扩展公开 Token 元数据。
type tokenRecord struct {
	TokenInfo `msgpack:",inline"` // TokenInfo keeps existing storage fields flat. TokenInfo 保持已有存储字段平铺。
	AccessID  string              `json:"accessId,omitempty" msgpack:",omitempty"` // AccessID distinguishes lifecycles that reuse a token value. AccessID 区分复用同一 Token 值的生命周期。
}

// getTokenRecord loads token metadata together with its optional lifecycle identity. getTokenRecord 加载 Token 元数据及可选的生命周期标识。
func (m *Manager) getTokenRecord(ctx context.Context, tokenValue string) (*tokenRecord, error) {
	// Validate token value 校验 Token 值。
	if tokenValue == "" {
		return nil, derror.ErrInvalidToken
	}

	// Load token data 加载 Token 数据。
	tokenInfoData, err := m.storage.Get(ctx, m.getTokenKey(tokenValue))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	// Return invalid token when missing Token 不存在时返回无效 Token。
	if tokenInfoData == nil {
		return nil, derror.ErrInvalidToken
	}

	// Convert token storage value 转换 Token 存储值。
	tokenInfoBytes, err := utils.ToBytes(tokenInfoData)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrTypeConvert, err)
	}

	// Detect logical token state 识别 Token 逻辑状态。
	if stateErr := tokenStateError(TokenState(tokenInfoBytes)); stateErr != nil {
		return nil, stateErr
	}

	// Decode token info 解码 Token 信息。
	var tokenInfo tokenRecord
	err = m.serializer.Decode(tokenInfoBytes, &tokenInfo)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}

	// Treat the token key namespace as the canonical auth type. 以 Token 存储键命名空间作为认证类型真值。
	tokenInfo.AuthType = m.config.AuthType

	// Return token info 返回 Token 信息。
	return &tokenInfo, nil
}

// checkLoginAndGetContext validates login state and returns loaded context. checkLoginAndGetContext 校验登录态并返回已加载上下文。
func (m *Manager) checkLoginAndGetContext(ctx context.Context, tokenValue string) (*Session, *TokenInfo, error) {
	return m.checkLoginAndGetContextWithOptions(ctx, tokenValue, checkLoginOptions{allowRenew: true})
}

// checkLoginAndGetTokenInfo validates login state without loading account session. checkLoginAndGetTokenInfo 校验登录态但不加载账号 Session。
func (m *Manager) checkLoginAndGetTokenInfo(ctx context.Context, tokenValue string) (*TokenInfo, error) {
	// Inspect token mapping and the states that directly determine validity. 检查直接决定有效性的 Token 映射及状态。
	tokenInfo, activeTimeout, activeExpired, err := m.inspectLoginToken(ctx, tokenValue)
	if err != nil {
		return nil, err
	}

	// Require only the session key because account disable intentionally detaches all tokens. 仅确认 Session 键存在，因为账号封禁会主动解绑全部 Token。
	if !m.storage.Exists(ctx, m.getSessionKey(tokenInfo.LoginID)) {
		return nil, derror.ErrInvalidToken
	}

	// Persist active-timeout state only on the exceptional path. 仅在不活跃超时的异常路径落盘状态。
	if activeExpired {
		if err = m.processTerminals(ctx, tokenInfo.LoginID, func(sess *Session) []TerminalInfo {
			if info, ok := sess.removeTerminalByToken(tokenValue); ok {
				return []TerminalInfo{info}
			}
			return nil
		}, TokenStateActiveTimeout, terminalInfoFromTokenInfo(tokenValue, tokenInfo)); err != nil {
			return nil, err
		}
		return nil, derror.ErrActiveTimeout
	}

	// Keep renewal and active refresh outside the synchronous validation path. 将续期和活跃刷新移出同步校验路径。
	m.submitLoginMaintenance(ctx, tokenValue, tokenInfo, activeTimeout)
	return tokenInfo, nil
}

// checkLoginAndGetContextNoRenew validates login state without renew side effects. checkLoginAndGetContextNoRenew 校验登录态但不触发续期副作用。
func (m *Manager) checkLoginAndGetContextNoRenew(ctx context.Context, tokenValue string) (*Session, *TokenInfo, error) {
	return m.checkLoginAndGetContextWithOptions(ctx, tokenValue, checkLoginOptions{})
}

// checkLoginAndGetContextNoRenewLocked validates login state while caller holds login lock. checkLoginAndGetContextNoRenewLocked 在调用方已持有登录锁时校验登录态。
func (m *Manager) checkLoginAndGetContextNoRenewLocked(ctx context.Context, tokenValue string) (*Session, *TokenInfo, error) {
	return m.checkLoginAndGetContextWithOptions(ctx, tokenValue, checkLoginOptions{lockHeld: true})
}

// checkLoginOptions controls login-state validation side effects. checkLoginOptions 控制登录态校验副作用。
type checkLoginOptions struct {
	allowRenew bool // allowRenew enables async renew and active refresh tasks. allowRenew 启用异步续期和活跃刷新任务。
	lockHeld   bool // lockHeld indicates caller already holds the login write lock. lockHeld 表示调用方已持有登录写锁。
}

// checkLoginAndGetContextWithOptions validates login state with optional side effects. checkLoginAndGetContextWithOptions 按选项校验登录态。
func (m *Manager) checkLoginAndGetContextWithOptions(ctx context.Context, tokenValue string, opts checkLoginOptions) (*Session, *TokenInfo, error) {
	// Inspect token mapping before loading the account session. 加载账号 Session 前检查 Token 映射。
	tokenInfo, activeTimeout, activeExpired, err := m.inspectLoginToken(ctx, tokenValue)
	if err != nil {
		return nil, nil, err
	}

	// Load session 加载会话。
	sess, err := m.getSession(ctx, tokenInfo.LoginID)
	if err != nil {
		// Map missing session to invalid token 会话不存在时映射为无效 Token。
		if errors.Is(err, derror.ErrSessionNotFound) {
			return nil, nil, derror.ErrInvalidToken
		}
		return nil, nil, err
	}

	// Require a valid account session; token mapping remains the login-state source. 要求账号 Session 有效，登录态仍以 Token 映射为准。
	if sess == nil {
		return nil, nil, derror.ErrInvalidToken
	}

	// Handle inactive timeout after the full context has been confirmed. 完整上下文确认后处理不活跃超时。
	if activeExpired {
		if opts.lockHeld {
			// Mark active timeout without reentering the same login lock. 已持锁时不重复进入同一登录锁。
			attached := sess.hasTerminalToken(tokenValue)
			if err = m.markActiveTimeoutLocked(ctx, tokenInfo.LoginID, tokenValue, sess); err != nil {
				return nil, nil, err
			}
			// Only a changed session participates in timeout lifecycle events. 只有实际变更的 Session 才参与超时生命周期事件。
			if !attached {
				return nil, tokenInfo, derror.ErrActiveTimeout
			}
			return sess, tokenInfo, derror.ErrActiveTimeout
		}

		// Mark inactive timeout separately so later checks keep the exact cause. 单独标记不活跃超时以保留精确原因。
		if err = m.processTerminals(ctx, tokenInfo.LoginID, func(sess *Session) []TerminalInfo {
			if info, ok := sess.removeTerminalByToken(tokenValue); ok {
				return []TerminalInfo{info}
			}
			return nil
		}, TokenStateActiveTimeout, terminalInfoFromTokenInfo(tokenValue, tokenInfo)); err != nil {
			return nil, nil, err
		}
		return nil, nil, derror.ErrActiveTimeout
	}

	// Keep optional maintenance outside the synchronous validation path. 将可选维护移出同步校验路径。
	if opts.allowRenew {
		m.submitLoginMaintenance(ctx, tokenValue, tokenInfo, activeTimeout)
	}

	// Return checked context 返回已校验上下文。
	return sess, tokenInfo, nil
}

// inspectLoginToken checks only states that directly determine token validity. inspectLoginToken 仅检查直接决定 Token 有效性的状态。
func (m *Manager) inspectLoginToken(ctx context.Context, tokenValue string) (*TokenInfo, int64, bool, error) {
	// Load token mapping and preserve its logical-state errors. 加载 Token 映射并保留逻辑状态错误。
	tokenInfo, err := m.getTokenInfo(ctx, tokenValue)
	if err != nil {
		return nil, 0, false, err
	}
	if tokenInfo.LoginID == "" {
		return nil, 0, false, derror.ErrInvalidToken
	}

	// Keep reversible disable rules synchronous because they gate the current request. 可解除封禁会限制当前请求，因此保持同步检查。
	if err = m.checkLoginDisableState(ctx, tokenInfo.LoginID, tokenInfo.Device, tokenInfo.DeviceID); err != nil {
		return nil, 0, false, err
	}

	// Skip active marker lookup when inactive timeout is disabled. 未启用不活跃超时时跳过活跃标记查询。
	activeTimeout := m.resolveActiveTimeoutFromSeconds(tokenInfo.ActiveTimeout)
	if activeTimeout <= 0 {
		return tokenInfo, activeTimeout, false, nil
	}

	// Load the active marker because it directly determines current validity. 加载直接决定当前有效性的活跃标记。
	activeValue, err := m.storage.Get(ctx, m.getActiveKey(tokenValue))
	if err != nil {
		return nil, 0, false, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	if activeValue == nil {
		return nil, 0, false, derror.ErrInvalidToken
	}

	activeAt, err := utils.ToInt64(activeValue)
	if err != nil {
		_ = m.storage.Delete(ctx, m.getActiveKey(tokenValue))
		return nil, 0, false, derror.ErrInvalidToken
	}
	return tokenInfo, activeTimeout, time.Now().Unix()-activeAt > activeTimeout, nil
}

// submitLoginMaintenance schedules renewal and active refresh after validation. submitLoginMaintenance 在校验成功后调度续期和活跃刷新。
func (m *Manager) submitLoginMaintenance(ctx context.Context, tokenValue string, tokenInfo *TokenInfo, activeTimeout int64) {
	if submit := m.prepareLoginMaintenance(ctx, tokenValue, tokenInfo, activeTimeout); submit != nil {
		submit()
	}
}

// prepareLoginMaintenance reserves maintenance now and returns submission for use after unlocking. prepareLoginMaintenance 立即预留维护任务，返回供解锁后调用的提交函数。
func (m *Manager) prepareLoginMaintenance(ctx context.Context, tokenValue string, tokenInfo *TokenInfo, activeTimeout int64) func() {
	if tokenInfo == nil || tokenValue == "" || tokenInfo.LoginID == "" {
		return nil
	}

	// Skip task coordination when no maintenance feature is enabled. 未启用任何维护功能时跳过任务协调。
	if !m.config.AutoRenew && activeTimeout <= 0 {
		return nil
	}

	// Record request time instead of worker execution time for active-timeout semantics. 记录请求时间而非工作线程执行时间，以保持活跃超时语义准确。
	activeAt := int64(0)
	if activeTimeout > 0 {
		activeAt = time.Now().Unix()
	}

	// Reserve one task before storage checks so concurrent requests do not repeat maintenance reads. 存储检查前登记唯一任务，避免并发请求重复执行维护读取。
	generation, reserved := m.beginLoginMaintenance(tokenValue, activeAt, false)
	if !reserved {
		return nil
	}

	// Active maintenance already needs a worker; otherwise submit only when timeout renewal is due. 活跃维护本身需要工作线程；否则仅在 Token 超时续期到期时提交。
	checkRenew := m.config.AutoRenew
	if activeTimeout <= 0 {
		checkRenew = checkRenew && m.isAutoRenewDue(ctx, tokenValue)
	}
	if !checkRenew && activeTimeout <= 0 {
		m.finishLoginMaintenance(tokenValue, generation)
		return nil
	}

	loginID := tokenInfo.LoginID
	createTime := tokenInfo.CreateTime
	return func() {
		accepted := m.submitAsync("check login maintenance", func() {
			defer m.finishLoginMaintenance(tokenValue, generation)
			m.runLoginMaintenance(tokenValue, loginID, createTime, generation, checkRenew, true, activeTimeout > 0)
		})
		if !accepted {
			m.finishLoginMaintenance(tokenValue, generation)
		}
	}
}

// runLoginMaintenance renews token timeout and active state using one checked context. runLoginMaintenance 使用一次校验上下文续期 Token 和活跃状态。
func (m *Manager) runLoginMaintenance(tokenValue, loginID string, createTime int64, generation uint64, renew, recheckRenewDue, refreshActive bool) {
	// Reject a task invalidated before worker execution. 拒绝在线程执行前已失效的任务。
	if !m.isLoginMaintenanceCurrent(tokenValue, generation) {
		return
	}

	bg := context.Background()
	unlock := m.lockLoginWrite(loginID)
	defer func() { unlock() }()

	// Recheck the generation after waiting for account lifecycle writes. 等待账号生命周期写操作后再次校验任务代次。
	if !m.isLoginMaintenanceCurrent(tokenValue, generation) {
		return
	}

	// Reload token identity so a stale task cannot maintain a reused token. 重新加载 Token 身份，避免旧任务维护被复用的 Token。
	latestTokenInfo, err := m.getTokenInfo(bg, tokenValue)
	if err != nil || latestTokenInfo.LoginID != loginID || latestTokenInfo.CreateTime != createTime {
		return
	}
	if err = m.checkLoginDisableState(bg, loginID, latestTokenInfo.Device, latestTokenInfo.DeviceID); err != nil {
		return
	}

	// Load the account session once for both maintenance operations. 为两类维护操作只加载一次账号 Session。
	latestSession, err := m.getSession(bg, loginID)
	if err != nil || latestSession == nil {
		return
	}

	// Renew timeout after applying the caller's automatic or forced policy. 按调用方的自动或强制策略续期超时时间。
	renewed := false
	if renew {
		renewed = m.renewTokenAndSessionLocked(bg, tokenValue, latestTokenInfo, latestSession, recheckRenewDue)
	}

	// Refresh active state only while the latest token lifecycle still enables it. 仅在最新 Token 生命周期仍启用活跃超时时刷新状态。
	if refreshActive && m.resolveActiveTimeoutFromSeconds(latestTokenInfo.ActiveTimeout) > 0 {
		// Repeat only when another validation advanced activity during the storage write. 仅当存储写入期间有新校验推进活跃时间时再次写入。
		for {
			latestActiveAt, current := m.getLoginMaintenanceActiveAt(tokenValue, generation)
			if !current || latestActiveAt <= 0 {
				break
			}
			if err = m.storage.Set(bg, m.getActiveKey(tokenValue), latestActiveAt, m.resolveTokenExpiration(latestTokenInfo)); err != nil {
				m.logger.Errorf("manager.runLoginMaintenance: failed to set active key, token=%s, error=%v", tokenValue, err)
				break
			}
			if m.finishLoginMaintenanceActiveWrite(tokenValue, generation, latestActiveAt) {
				break
			}
		}
	}

	// Release the account lock before publishing the renewal event. 发布续期事件前释放账号锁。
	unlock()
	unlock = func() {}
	if renewed {
		m.triggerEvent(listener.EventRenew, loginID, latestTokenInfo.Device, latestTokenInfo.DeviceID, tokenValue, nil)
	}
}

// isAutoRenewDue reports whether a token currently meets auto-renew conditions. isAutoRenewDue 判断 Token 当前是否满足自动续期条件。
func (m *Manager) isAutoRenewDue(ctx context.Context, tokenValue string) bool {
	if !m.config.AutoRenew || m.config.Timeout <= 0 {
		return false
	}

	// Require a positive limited TTL before applying threshold and interval rules. 仅对剩余时间为正的有限期 Token 应用阈值与间隔规则。
	ttl, err := m.storage.TTL(ctx, m.getTokenKey(tokenValue))
	if err != nil || ttl <= 0 {
		return false
	}
	ttlSeconds := int64(ttl.Seconds())
	if ttlSeconds <= 0 || (m.config.RenewMaxRefresh > 0 && ttlSeconds > m.config.RenewMaxRefresh) {
		return false
	}

	return m.config.RenewInterval <= 0 || !m.storage.Exists(ctx, m.getRenewKey(tokenValue))
}

// markActiveTimeoutLocked marks one token inactive while login lock is already held. markActiveTimeoutLocked 在已持有登录锁时标记 Token 不活跃超时。
func (m *Manager) markActiveTimeoutLocked(ctx context.Context, loginID, tokenValue string, sess *Session) error {
	if sess == nil {
		return nil
	}
	_, attached := sess.removeTerminalByToken(tokenValue)
	if err := m.setTokenState(ctx, tokenValue, TokenStateActiveTimeout, m.tokenStateExpiration(ctx, tokenValue)); err != nil {
		return err
	}
	if err := m.cleanTokenMetadata(ctx, []string{tokenValue}); err != nil {
		return err
	}
	if !attached {
		return nil
	}
	if len(sess.TerminalInfos) == 0 {
		if err := m.storage.Delete(ctx, m.getSessionKey(loginID)); err != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
		return nil
	}
	return m.saveToStorage(ctx, m.getSessionKey(loginID), *sess)
}

// checkLoginInternal performs the core login validation logic. checkLoginInternal 执行登录状态的核心验证逻辑。
func (m *Manager) checkLoginInternal(ctx context.Context, tokenValue string) error {
	// Validate the token mapping without loading account session. 校验 Token 映射但不加载账号 Session。
	_, err := m.checkLoginAndGetTokenInfo(ctx, tokenValue)
	return err
}

// cleanExpiredTerminals removes structurally inactive tokens without treating temporary disable state as expiration. cleanExpiredTerminals 清理结构性失效 Token，但不把临时封禁状态视为过期。
func (m *Manager) cleanExpiredTerminals(ctx context.Context, sess *Session) (bool, []TerminalInfo, error) {
	// Skip empty session 跳过空会话。
	if sess == nil || len(sess.TerminalInfos) == 0 {
		return false, nil, nil
	}

	// Prepare valid terminal list 准备有效终端列表。
	var validTerminals []TerminalInfo
	var activeTimeoutTerminals []TerminalInfo
	hasExpired := false

	// Check each terminal 逐个检查终端。
	for _, ti := range sess.TerminalInfos {
		// Load token metadata without applying reversible disable rules. 加载 Token 元数据，但不应用可解除的封禁规则。
		tokenInfo, err := m.getTokenInfo(ctx, ti.Token)
		if err != nil {
			if isTokenInactiveError(err) {
				// Clean metadata for inactive tokens while preserving any logical token state. 清理失效 Token 的元数据，同时保留其逻辑状态。
				if cleanErr := m.cleanTokenMetadata(ctx, []string{ti.Token}); cleanErr != nil {
					return false, activeTimeoutTerminals, cleanErr
				}
				hasExpired = true
				continue
			}
			return false, activeTimeoutTerminals, err
		}
		if tokenInfo.LoginID == "" {
			// Drop malformed token records that cannot be associated with an account. 删除无法关联账号的畸形 Token 记录。
			if deleteErr := m.storage.Delete(ctx, m.getTokenKey(ti.Token)); deleteErr != nil {
				return false, activeTimeoutTerminals, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, deleteErr)
			}
			if cleanErr := m.cleanTokenMetadata(ctx, []string{ti.Token}); cleanErr != nil {
				return false, activeTimeoutTerminals, cleanErr
			}
			hasExpired = true
			continue
		}
		if tokenInfo.LoginID != sess.LoginID {
			hasExpired = true
			continue
		}
		if ti.LoginID != tokenInfo.LoginID ||
			ti.Device != tokenInfo.Device ||
			ti.DeviceID != tokenInfo.DeviceID ||
			ti.CreateTime != tokenInfo.CreateTime {
			// Detach mismatched terminal metadata without mutating the canonical token mapping. 移除错位终端元数据，但不改写作为身份真值的 Token 映射。
			hasExpired = true
			continue
		}

		// Keep tokens that do not use active timeout. 保留未启用活跃超时的 Token。
		activeTimeout := m.resolveActiveTimeoutFromSeconds(tokenInfo.ActiveTimeout)
		if activeTimeout <= 0 {
			validTerminals = append(validTerminals, ti)
			continue
		}

		// Load and validate active timestamp. 加载并校验活跃时间戳。
		activeValue, activeErr := m.storage.Get(ctx, m.getActiveKey(ti.Token))
		if activeErr != nil {
			return false, activeTimeoutTerminals, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, activeErr)
		}
		if activeValue == nil {
			if deleteErr := m.storage.Delete(ctx, m.getTokenKey(ti.Token)); deleteErr != nil {
				return false, activeTimeoutTerminals, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, deleteErr)
			}
			if cleanErr := m.cleanTokenMetadata(ctx, []string{ti.Token}); cleanErr != nil {
				return false, activeTimeoutTerminals, cleanErr
			}
			hasExpired = true
			continue
		}

		activeAt, convertErr := utils.ToInt64(activeValue)
		if convertErr != nil {
			if deleteErr := m.storage.Delete(ctx, m.getTokenKey(ti.Token)); deleteErr != nil {
				return false, activeTimeoutTerminals, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, deleteErr)
			}
			if cleanErr := m.cleanTokenMetadata(ctx, []string{ti.Token}); cleanErr != nil {
				return false, activeTimeoutTerminals, cleanErr
			}
			hasExpired = true
			continue
		}

		if time.Now().Unix()-activeAt > activeTimeout {
			if stateErr := m.setTokenState(ctx, ti.Token, TokenStateActiveTimeout, m.tokenStateExpiration(ctx, ti.Token)); stateErr != nil {
				return false, activeTimeoutTerminals, stateErr
			}
			if cleanErr := m.cleanTokenMetadata(ctx, []string{ti.Token}); cleanErr != nil {
				return false, activeTimeoutTerminals, cleanErr
			}
			activeTimeoutTerminals = append(activeTimeoutTerminals, ti)
			hasExpired = true
			continue
		}

		validTerminals = append(validTerminals, ti)
	}

	// Update session when expired tokens exist 存在过期 Token 时更新 Session
	if hasExpired {
		// Replace terminal list 替换终端列表。
		sess.TerminalInfos = validTerminals

		// Delete session when all terminals expired 所有终端均已过期时删除整个 session
		if len(validTerminals) == 0 {
			if err := m.storage.Delete(ctx, m.getSessionKey(sess.LoginID)); err != nil {
				return false, activeTimeoutTerminals, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
			}
			return true, activeTimeoutTerminals, nil
		} else {
			if err := m.saveToStorage(ctx, m.getSessionKey(sess.LoginID), *sess); err != nil {
				return false, activeTimeoutTerminals, err
			}
		}
	}

	return false, activeTimeoutTerminals, nil
}
