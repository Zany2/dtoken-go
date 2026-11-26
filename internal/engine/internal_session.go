// @Author daixk 2025/12/22 15:56:00
package engine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Zany2/dtoken-go/config"
	derror "github.com/Zany2/dtoken-go/errors"
	listener "github.com/Zany2/dtoken-go/event"
	"github.com/Zany2/dtoken-go/internal/utils"
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

	// Decode through a pointer so encoded null cannot become a valid empty session. 通过指针解码，避免编码空值被转换成有效的空会话。
	var sess *Session
	err = m.serializer.Decode(bytesData, &sess)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}
	if sess == nil {
		return nil, derror.ErrInvalidToken
	}

	// Reject records from a different authentication namespace. 拒绝其他认证命名空间的记录。
	if sess.AuthType != "" && sess.AuthType != m.config.AuthType {
		return nil, derror.ErrInvalidToken
	}

	// The requested key remains authoritative for the account ID. 请求的存储键仍是账号 ID 的真值。
	sess.AuthType = m.config.AuthType
	sess.LoginID = loginID

	// Return session 返回会话。
	return sess, nil
}

// getTokenInfo retrieves token information. getTokenInfo 获取 Token 信息。
func (m *Manager) getTokenInfo(ctx context.Context, tokenValue string) (*TokenInfo, error) {
	record, err := m.getTokenRecord(ctx, tokenValue)
	if err != nil {
		return nil, err
	}
	if err = m.loadTokenExtra(ctx, record); err != nil {
		return nil, err
	}
	return &record.TokenInfo, nil
}

// tokenRecord preserves login identity without extending the public token metadata. tokenRecord 保留登录身份，但不扩展公开 Token 元数据。
type tokenRecord struct {
	// TokenInfo keeps existing storage fields flat. TokenInfo 保持已有存储字段平铺。
	TokenInfo `msgpack:",inline"`
	// AccessID distinguishes lifecycles that reuse a token value. AccessID 区分复用同一 Token 值的生命周期。
	AccessID string `json:"accessId,omitempty" msgpack:",omitempty"`
	// TerminalIndex binds the token to its account terminal entry. TerminalIndex 将 Token 绑定到账号终端条目。
	TerminalIndex int64 `json:"terminalIndex,omitempty" msgpack:",omitempty"`
	// Revoked keeps account-disabled credentials invalid after untie. Revoked 保证账号封禁废弃的凭证在解封后仍然无效。
	Revoked bool `json:"revoked,omitempty" msgpack:",omitempty"`
	// HasExtra identifies extension data stored separately from compact credentials. HasExtra 标识与紧凑凭据分开存储的扩展数据。
	HasExtra bool `json:"hasExtra,omitempty" msgpack:",omitempty"`
}

// getTokenRecord loads compact credentials with an explicit namespace and lifecycle identity. getTokenRecord 加载带有明确命名空间和生命周期标识的紧凑凭据。
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

	return m.decodeStoredTokenRecord(tokenInfoData)
}

// decodeStoredTokenRecord interprets an already loaded primary value without accessing storage. decodeStoredTokenRecord 解析已读取的主记录值，不再次访问存储。
func (m *Manager) decodeStoredTokenRecord(tokenInfoData any) (*tokenRecord, error) {
	// Return invalid token when missing Token 不存在时返回无效 Token。
	if tokenInfoData == nil {
		return nil, derror.ErrInvalidToken
	}

	// Redis returns strings; avoid allocating a byte slice for the compact hot path. Redis 返回字符串，紧凑格式热路径避免分配字节切片。
	var data string
	switch value := tokenInfoData.(type) {
	case string:
		data = value
	case []byte:
		data = string(value)
	default:
		encoded, convertErr := utils.ToBytes(value)
		if convertErr != nil {
			return nil, fmt.Errorf("%w: %v", derror.ErrTypeConvert, convertErr)
		}
		data = string(encoded)
	}

	// Detect logical token state 识别 Token 逻辑状态。
	if stateErr := tokenStateError(TokenState(data)); stateErr != nil {
		return nil, stateErr
	}

	// Legacy payload conversion belongs to offline migration, never request validation. 旧载荷转换由离线迁移完成，不进入请求校验路径。
	tokenInfo, err := decodeTokenRecord(data)
	if err != nil {
		return nil, err
	}

	// Runtime records must identify both their namespace and login lifecycle. 运行时记录必须明确认证命名空间和登录生命周期。
	if tokenInfo.AuthType != m.config.AuthType || tokenInfo.AccessID == "" {
		return nil, derror.ErrInvalidToken
	}

	// Return token info 返回 Token 信息。
	return tokenInfo, nil
}

// terminalMatchesTokenRecord reports whether a session terminal owns the current token lifecycle. terminalMatchesTokenRecord 判断 Session 终端是否属于当前 Token 生命周期。
func terminalMatchesTokenRecord(sessionLoginID string, terminal TerminalInfo, record *tokenRecord) bool {
	if record == nil || sessionLoginID == "" || terminal.Token == "" ||
		record.LoginID != sessionLoginID || terminal.LoginID != record.LoginID ||
		terminal.Device != record.Device || terminal.DeviceID != record.DeviceID ||
		terminal.CreateTime != record.CreateTime {
		return false
	}

	// New records use the account-local terminal sequence to distinguish same-second token reuse. 新记录使用账号内终端序号区分同一秒内复用 Token 的生命周期。
	if record.TerminalIndex > 0 {
		return terminal.Index == record.TerminalIndex
	}

	// Migrated records may lack a terminal sequence, so retain their original metadata matching. 迁移后的记录可能没有终端序号，因此保留原有元数据匹配。
	return true
}

// tokenRecordsMatchLifecycle compares persisted identities without depending on mutable timeout metadata. tokenRecordsMatchLifecycle 比较持久化生命周期标识，不依赖可变的超时元数据。
func tokenRecordsMatchLifecycle(expected, current *tokenRecord) bool {
	if expected == nil || current == nil || expected.LoginID == "" || expected.LoginID != current.LoginID {
		return false
	}

	return expected.AccessID != "" && expected.AccessID == current.AccessID
}

// tokenRecordStillMatches reloads a token before lifecycle side effects. tokenRecordStillMatches 在产生生命周期副作用前重新加载并核对 Token。
func (m *Manager) tokenRecordStillMatches(ctx context.Context, tokenValue string, expected *tokenRecord) (bool, error) {
	current, err := m.getTokenRecord(ctx, tokenValue)
	if err != nil {
		if isTokenInactiveError(err) {
			return false, nil
		}
		return false, err
	}
	return tokenRecordsMatchLifecycle(expected, current), nil
}

// loadTerminalCleanupRecord returns the checked credential for reuse during the same locked cleanup; nil means an inactive mapping. loadTerminalCleanupRecord 返回可在同次锁内清理中复用的已核对凭据，nil 表示映射已失效。
func (m *Manager) loadTerminalCleanupRecord(ctx context.Context, sessionLoginID string, terminal TerminalInfo) (*tokenRecord, bool, error) {
	if terminal.Token == "" {
		return nil, false, nil
	}
	data, err := m.storage.Get(ctx, m.getTokenKey(terminal.Token))
	if err != nil {
		return nil, false, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	current, err := m.decodeStoredTokenRecord(data)
	if err == nil {
		return current, terminalMatchesTokenRecord(sessionLoginID, terminal, current), nil
	}
	if !isTokenInactiveError(err) {
		return nil, false, err
	}

	// Invalid tokens may be foreign namespace records rather than missing mappings. 无效 Token 可能是其他认证命名空间的记录，而非不存在的映射。
	if errors.Is(err, derror.ErrInvalidToken) && data != nil {
		return nil, false, nil
	}
	allowed, err := m.inactiveTerminalCleanupAllowed(ctx, sessionLoginID, terminal.Token)
	return nil, allowed, err
}

// inactiveTerminalCleanupAllowed checks refresh ownership after the caller has established an inactive primary mapping. inactiveTerminalCleanupAllowed 在调用方已确认主映射失效后核对刷新归属。
func (m *Manager) inactiveTerminalCleanupAllowed(ctx context.Context, sessionLoginID, token string) (bool, error) {
	// An expired token value may now index another account's refresh credential. 已过期的 Token 值可能已索引到其他账号的刷新凭据。
	data, err := m.storage.Get(ctx, m.getTokenRefreshKey(token))
	if err != nil {
		return false, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	if data == nil {
		return true, nil
	}
	refreshBytes, err := utils.ToBytes(data)
	if err != nil {
		return false, fmt.Errorf("%w: %v", derror.ErrTypeConvert, err)
	}
	info, err := m.getRefreshTokenInfo(ctx, string(refreshBytes))
	if err != nil {
		if errors.Is(err, derror.ErrInvalidRefreshToken) {
			return false, nil
		}
		return false, err
	}
	return info.LoginID == sessionLoginID && info.AccessToken == token, nil
}

// checkLoginAndGetTokenInfo validates token identity without loading account data on the normal path. checkLoginAndGetTokenInfo 校验 Token 身份，正常路径不加载账号数据。
func (m *Manager) checkLoginAndGetTokenInfo(ctx context.Context, tokenValue string) (*TokenInfo, error) {
	_, tokenInfo, err := m.checkLoginAndGetContextWithOptions(ctx, tokenValue, checkLoginOptions{allowRenew: true, tokenOnly: true})
	return tokenInfo, err
}

// checkLoginAndGetContextNoRenew validates login state without renew side effects. checkLoginAndGetContextNoRenew 校验登录态但不触发续期副作用。
func (m *Manager) checkLoginAndGetContextNoRenew(ctx context.Context, tokenValue string) (*Session, *TokenInfo, error) {
	return m.checkLoginAndGetContextWithOptions(ctx, tokenValue, checkLoginOptions{})
}

// checkLoginAndGetContextNoRenewLocked validates the captured token lifecycle while its account lock is held. checkLoginAndGetContextNoRenewLocked 在持有账号锁时校验已捕获的 Token 生命周期。
func (m *Manager) checkLoginAndGetContextNoRenewLocked(ctx context.Context, tokenValue string, expectedRecord *tokenRecord) (*Session, *TokenInfo, error) {
	if expectedRecord == nil || expectedRecord.LoginID == "" {
		return nil, nil, derror.ErrInvalidToken
	}
	return m.checkLoginAndGetContextWithOptions(ctx, tokenValue, checkLoginOptions{expectedRecord: expectedRecord, lockHeld: true})
}

// checkLoginOptions controls login-state validation side effects. checkLoginOptions 控制登录态校验副作用。
type checkLoginOptions struct {
	checkDisable bool // checkDisable enforces account and device restrictions only for explicit access checks. checkDisable 仅为显式访问检查执行账号与设备限制。
	allowRenew   bool // allowRenew enables async renew and active refresh tasks. allowRenew 启用异步续期和活跃刷新任务。
	tokenOnly    bool // tokenOnly skips session reads unless an inactive-timeout transition needs cleanup. tokenOnly 跳过会话读取，除非不活跃超时状态变更需要清理。
	// expectedRecord identifies the lifecycle captured before acquiring its account lock. expectedRecord 标识获取账号锁前捕获的生命周期。
	expectedRecord *tokenRecord
	// lockHeld indicates the expected lifecycle's account lock is already held. lockHeld 表示已持有预期生命周期的账号锁。
	lockHeld  bool
	loadExtra bool // loadExtra requests extensions only for complete-detail operations. loadExtra 仅为完整详情操作请求扩展数据。
}

// checkLoginAndGetContextWithOptions validates login state with optional side effects. checkLoginAndGetContextWithOptions 按选项校验登录态。
func (m *Manager) checkLoginAndGetContextWithOptions(ctx context.Context, tokenValue string, opts checkLoginOptions) (*Session, *TokenInfo, error) {
	// Inspect token mapping before loading the account session. 加载账号 Session 前检查 Token 映射。
	record, activeTimeout, activeExpired, err := m.inspectLoginToken(ctx, tokenValue, opts.expectedRecord, opts.checkDisable)
	if err != nil {
		return nil, nil, err
	}
	tokenInfo := &record.TokenInfo

	// Load account data only for callers that need it or for timeout cleanup. 仅在调用方需要账号数据或超时清理时加载会话。
	var sess *Session
	if !opts.tokenOnly || activeExpired {
		sess, err = m.getSession(ctx, tokenInfo.LoginID)
		if err != nil {
			if errors.Is(err, derror.ErrSessionNotFound) {
				return nil, nil, derror.ErrInvalidToken
			}
			return nil, nil, err
		}
		if sess == nil {
			return nil, nil, derror.ErrInvalidToken
		}
	}

	// Handle inactive timeout after the full context has been confirmed. 完整上下文确认后处理不活跃超时。
	if activeExpired {
		if opts.lockHeld {
			// Mark active timeout without reentering the same login lock. 已持锁时不重复进入同一登录锁。
			attached, transitioned, timeoutErr := m.markActiveTimeoutLocked(ctx, tokenInfo.LoginID, tokenValue, sess, record)
			if timeoutErr != nil {
				return nil, nil, timeoutErr
			}
			if !transitioned {
				return nil, nil, derror.ErrActiveTimeout
			}

			// Only a changed session participates in timeout lifecycle events. 只有实际变更的 Session 才参与超时生命周期事件。
			if !attached {
				return nil, tokenInfo, derror.ErrActiveTimeout
			}
			return sess, tokenInfo, derror.ErrActiveTimeout
		}

		// Recheck activity and lifecycle under the account lock before retiring a token observed as expired. 对锁外判定超时的 Token，在账号锁内重新检查活跃状态与生命周期后再下线。
		func() {
			unlock := m.lockLoginWrite(record.LoginID)
			defer unlock()
			recheck := opts
			recheck.expectedRecord, recheck.lockHeld, recheck.allowRenew, recheck.loadExtra = record, true, false, false
			sess, tokenInfo, err = m.checkLoginAndGetContextWithOptions(ctx, tokenValue, recheck)
		}()
		if err != nil {
			// Publish only a completed timeout transition, after releasing the account lock. 释放账号锁后，仅发布已完成的超时状态变更。
			if errors.Is(err, derror.ErrActiveTimeout) && tokenInfo != nil {
				if sess != nil && len(sess.TerminalInfos) == 0 {
					m.triggerEvent(listener.EventDestroySession, tokenInfo.LoginID, "", "", "", nil)
				}
				m.triggerEvent(listener.EventActiveTimeout, tokenInfo.LoginID, tokenInfo.Device, tokenInfo.DeviceID, tokenValue, nil)
			}
			return nil, nil, err
		}

		// A completed activity refresh makes the current request valid again; use the freshly loaded context. 已完成的活跃刷新使当前请求恢复有效，继续使用重新加载的上下文。
		activeTimeout = m.resolveActiveTimeoutFromSeconds(tokenInfo.ActiveTimeout)
	}

	// Keep optional maintenance outside the synchronous validation path. 将可选维护移出同步校验路径。
	if opts.allowRenew {
		m.submitLoginMaintenance(ctx, tokenValue, record, activeTimeout)
	}

	// Return checked context 返回已校验上下文。
	if opts.loadExtra {
		if err = m.loadTokenExtra(ctx, record); err != nil {
			return nil, nil, err
		}
		tokenInfo.Extra = record.Extra
	}
	return sess, tokenInfo, nil
}

// inspectLoginToken checks token validity and, when provided, its captured lifecycle before any cleanup. inspectLoginToken 校验 Token 有效性，并在提供捕获记录时先核对生命周期，再执行清理。
func (m *Manager) inspectLoginToken(ctx context.Context, tokenValue string, expectedRecord *tokenRecord, checkDisable bool) (*tokenRecord, int64, bool, error) {
	// Load token mapping and preserve its logical-state errors. 加载 Token 映射并保留逻辑状态错误。
	record, err := m.getTokenRecord(ctx, tokenValue)
	if err != nil {
		return nil, 0, false, err
	}

	// Token reuse may replace the lifecycle while the caller waits for its account lock. 调用方等待账号锁时，Token 复用可能替换其生命周期。
	if expectedRecord != nil && !tokenRecordsMatchLifecycle(expectedRecord, record) {
		return nil, 0, false, derror.ErrInvalidToken
	}
	return m.inspectLoginTokenRecord(ctx, tokenValue, record, checkDisable)
}

// inspectLoginTokenRecord checks a record already loaded in the current validation scope without fetching it again. inspectLoginTokenRecord 校验当前检查范围内已加载的记录，不再重复读取。
func (m *Manager) inspectLoginTokenRecord(ctx context.Context, tokenValue string, record *tokenRecord, checkDisable bool) (*tokenRecord, int64, bool, error) {
	if record == nil || record.LoginID == "" {
		return nil, 0, false, derror.ErrInvalidToken
	}
	tokenInfo := &record.TokenInfo

	// Explicit access restrictions run before maintenance and preserve their own errors. 显式访问限制在维护前执行，并保留其错误原因。
	if checkDisable {
		if err := m.checkLoginDisableState(ctx, tokenInfo.LoginID, tokenInfo.Device, tokenInfo.DeviceID); err != nil {
			return nil, 0, false, err
		}
	}

	// Retired credentials remain invalid even when restriction checks are omitted. 即使省略限制检查，已废弃凭证也保持无效。
	if record.Revoked {
		return nil, 0, false, derror.ErrInvalidToken
	}

	// Skip active marker lookup when inactive timeout is disabled. 未启用不活跃超时时跳过活跃标记查询。
	activeTimeout := m.resolveActiveTimeoutFromSeconds(tokenInfo.ActiveTimeout)
	if activeTimeout <= 0 {
		return record, activeTimeout, false, nil
	}

	// Load the active marker because it directly determines current validity. 加载直接决定当前有效性的活跃标记。
	pendingActiveAt := m.pendingLoginActiveAt(tokenValue, record)
	activeValue, err := m.storage.Get(ctx, m.getActiveKey(tokenValue))
	if err != nil {
		return nil, 0, false, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	if activeValue == nil {
		return nil, 0, false, derror.ErrInvalidToken
	}

	// Negative timestamps are malformed and can overflow elapsed-time subtraction. 负时间戳属于损坏数据，且可能导致时间差运算溢出。
	activeAt, err := utils.ToInt64(activeValue)
	if err != nil || activeAt < 0 {
		_ = m.storage.Delete(ctx, m.getActiveKey(tokenValue))
		return nil, 0, false, derror.ErrInvalidToken
	}
	activeAt = max(activeAt, pendingActiveAt)
	return record, activeTimeout, time.Now().Unix()-activeAt > activeTimeout, nil
}

// submitLoginMaintenance schedules renewal and active refresh after validation. submitLoginMaintenance 在校验成功后调度续期和活跃刷新。
func (m *Manager) submitLoginMaintenance(ctx context.Context, tokenValue string, record *tokenRecord, activeTimeout int64) {
	if submit := m.prepareLoginMaintenance(ctx, tokenValue, record, activeTimeout); submit != nil {
		submit()
	}
}

// prepareLoginMaintenance reserves maintenance now and returns submission for use after unlocking. prepareLoginMaintenance 立即预留维护任务，返回供解锁后调用的提交函数。
func (m *Manager) prepareLoginMaintenance(ctx context.Context, tokenValue string, record *tokenRecord, activeTimeout int64) func() {
	if record == nil || record.Revoked || tokenValue == "" || record.LoginID == "" {
		return nil
	}

	// Decide admission from configuration only; storage eligibility checks belong to the worker. 仅按配置决定任务接收，存储资格检查交给工作线程。
	checkRenew := m.config.AutoRenew && m.config.Timeout > 0
	if !checkRenew && activeTimeout <= 0 {
		return nil
	}

	// Record request time instead of worker execution time for active-timeout semantics. 记录请求时间而非工作线程执行时间，以保持活跃超时语义准确。
	activeAt := int64(0)
	if activeTimeout > 0 {
		activeAt = time.Now().Unix()
	}

	// Reserve one task before storage checks so concurrent requests do not repeat maintenance reads. 存储检查前登记唯一任务，避免并发请求重复执行维护读取。
	expectedRecord := *record
	generation, reserved := m.beginLoginMaintenance(tokenValue, &expectedRecord, activeAt, false)
	if !reserved {
		return nil
	}

	return func() {
		var fallback func()
		if activeTimeout > 0 {
			fallback = func() {
				defer m.finishLoginMaintenance(tokenValue, generation)
				m.runLoginMaintenance(tokenValue, &expectedRecord, generation, false, false, true)
			}
		}
		accepted := m.submitAsync("check login maintenance", func() {
			defer m.finishLoginMaintenance(tokenValue, generation)
			m.runLoginMaintenance(tokenValue, &expectedRecord, generation, checkRenew, true, activeTimeout > 0)
		}, fallback)
		if !accepted {
			m.finishLoginMaintenance(tokenValue, generation)
		}
	}
}

// runLoginMaintenance renews token timeout and active state using one checked context. runLoginMaintenance 使用一次校验上下文续期 Token 和活跃状态。
func (m *Manager) runLoginMaintenance(tokenValue string, expectedRecord *tokenRecord, generation uint64, renew, recheckRenewDue, refreshActive bool) {
	// Reject a task invalidated before worker execution. 拒绝在线程执行前已失效的任务。
	if expectedRecord == nil || !m.isLoginMaintenanceCurrent(tokenValue, generation) {
		return
	}

	bg := context.Background()
	loginID := expectedRecord.LoginID
	unlock := m.lockLoginWrite(loginID)
	defer func() { unlock() }()

	// Recheck the generation after waiting for account lifecycle writes. 等待账号生命周期写操作后再次校验任务代次。
	if !m.isLoginMaintenanceCurrent(tokenValue, generation) {
		return
	}

	// Check eligibility before loading full records; idle tokens need no further maintenance reads. 加载完整记录前检查续期条件，无需维护的 Token 不再继续读取。
	if renew && recheckRenewDue {
		renew = m.isAutoRenewDue(bg, tokenValue)
	}
	if !renew && !refreshActive {
		return
	}

	// Reload token identity so a stale task cannot maintain a reused token. 重新加载 Token 身份，避免旧任务维护被复用的 Token。
	latestRecord, err := m.getTokenRecord(bg, tokenValue)
	if err != nil || latestRecord.Revoked || !tokenRecordsMatchLifecycle(expectedRecord, latestRecord) {
		return
	}
	latestTokenInfo := &latestRecord.TokenInfo
	if err = m.checkLoginDisableState(bg, loginID, latestTokenInfo.Device, latestTokenInfo.DeviceID); err != nil {
		return
	}

	// Renew timeout after applying the caller's automatic or forced policy. 按调用方的自动或强制策略续期超时时间。
	renewed := false
	if renew {
		renewed = m.renewTokenAndSessionLocked(bg, tokenValue, latestRecord, false)
	}

	// Refresh active state only while the latest token lifecycle still enables it. 仅在最新 Token 生命周期仍启用活跃超时时刷新状态。
	if refreshActive && m.resolveActiveTimeoutFromSeconds(latestTokenInfo.ActiveTimeout) > 0 {
		// Repeat only when another validation advanced activity during the storage write. 仅当存储写入期间有新校验推进活跃时间时再次写入。
		for {
			latestActiveAt, current := m.getLoginMaintenanceActiveAt(tokenValue, generation)
			if !current || latestActiveAt <= 0 {
				break
			}
			if renewed {
				err = m.storage.Set(bg, m.getActiveKey(tokenValue), latestActiveAt, m.resolveTokenExpiration(latestTokenInfo))
			} else {
				// Activity changes the value, not the credential's remaining lifetime. 活跃更新只改变值，不延长凭据的剩余有效期。
				err = m.updateStoredValue(bg, m.getActiveKey(tokenValue), latestActiveAt)
			}
			if err != nil {
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
		// This worker is already tracked; resubmission can deadlock a full pool or lose the event during shutdown. 当前任务已被跟踪，再次提交可能使满载协程池死锁或在关闭期间丢失事件。
		m.triggerEventWithDispatch(listener.EventRenew, loginID, latestTokenInfo.Device, latestTokenInfo.DeviceID, tokenValue, nil, false)
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
	if m.config.RenewMaxRefresh > 0 && ttl > secondsToDuration(m.config.RenewMaxRefresh) {
		return false
	}

	return m.config.RenewInterval <= 0 || !m.storage.Exists(ctx, m.getRenewKey(tokenValue))
}

// markActiveTimeoutLocked marks one matching lifecycle inactive while the login lock is already held. markActiveTimeoutLocked 在已持有登录锁时标记匹配的 Token 生命周期不活跃超时。
func (m *Manager) markActiveTimeoutLocked(ctx context.Context, loginID, tokenValue string, sess *Session, expected *tokenRecord) (bool, bool, error) {
	// Recheck the captured lifecycle because another account lock cannot serialize reuse of the same token value. 重新核对已捕获生命周期，因为其他账号锁无法串行化同值 Token 复用。
	current, err := m.getTokenRecord(ctx, tokenValue)
	if err != nil {
		if isTokenInactiveError(err) {
			return false, false, nil
		}
		return false, false, err
	}
	if !tokenRecordsMatchLifecycle(expected, current) {
		return false, false, nil
	}

	// Expired credentials must also retire when their account session is already absent. 账号 Session 已不存在时，超时凭据仍须下线。
	attached := false
	if sess != nil {
		attached = len(terminalRemovalForTokenRecord(tokenValue, expected)(sess)) > 0
	}
	if err := m.setTokenStateWithRecord(ctx, tokenValue, current, TokenStateActiveTimeout, m.tokenStateExpiration(ctx, tokenValue, current)); err != nil {
		return false, false, err
	}
	if err := m.cleanTokenMetadata(ctx, []string{tokenValue}); err != nil {
		return false, false, err
	}
	if !attached {
		return false, true, nil
	}
	if len(sess.TerminalInfos) == 0 {
		if err := m.storage.Delete(ctx, m.getSessionKey(loginID)); err != nil {
			return false, false, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
		return true, true, nil
	}
	if err := m.saveToStorage(ctx, m.getSessionKey(loginID), *sess); err != nil {
		return false, false, err
	}
	return true, true, nil
}

// checkLoginInternal performs the core login validation logic. checkLoginInternal 执行登录状态的核心验证逻辑。
func (m *Manager) checkLoginInternal(ctx context.Context, tokenValue string) error {
	// Validate the credential without loading unrelated account data. 校验凭据，不加载无关账号数据。
	_, err := m.checkLoginAndGetTokenInfo(ctx, tokenValue)
	return err
}

// pruneExpiredTerminals retires matching inactive entries in memory; the caller persists the final session. pruneExpiredTerminals 下线匹配的失效条目并更新内存列表，由调用方持久化最终会话。
func (m *Manager) pruneExpiredTerminals(ctx context.Context, sess *Session, matches func(TerminalInfo) bool) (bool, []TerminalInfo, error) {
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
		if matches != nil && !matches(ti) {
			validTerminals = append(validTerminals, ti)
			continue
		}
		if ti.Token == "" {
			hasExpired = true
			continue
		}

		// Load token metadata without applying reversible disable rules. 加载 Token 元数据，但不应用可解除的封禁规则。
		data, readErr := m.storage.Get(ctx, m.getTokenKey(ti.Token))
		if readErr != nil {
			return false, activeTimeoutTerminals, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, readErr)
		}
		record, err := m.decodeStoredTokenRecord(data)
		if err != nil {
			if isTokenInactiveError(err) {
				// Natural access expiry must not revoke a longer-lived refresh credential. 访问令牌自然过期不能撤销有效期更长的刷新凭证。
				if data == nil {
					if cleanErr := m.cleanTokenActivityMetadata(ctx, []string{ti.Token}); cleanErr != nil {
						return false, activeTimeoutTerminals, cleanErr
					}
					hasExpired = true
					continue
				}

				// Logout markers share ErrInvalidToken with unidentified values, so recognize only the exact stored marker. 注销标记与无法识别的值共用 ErrInvalidToken，仅识别精确的已存储注销标记。
				logoutMarker := false
				switch value := data.(type) {
				case string:
					logoutMarker = value == string(TokenStateLogout)
				case []byte:
					logoutMarker = string(value) == string(TokenStateLogout)
				}

				// Unidentified values may belong to another namespace; detach only their stale Session entry. 无法识别的值可能属于其他命名空间，仅移除陈旧 Session 条目。
				if logoutMarker || !errors.Is(err, derror.ErrInvalidToken) {
					allowed, checkErr := m.inactiveTerminalCleanupAllowed(ctx, sess.LoginID, ti.Token)
					if checkErr != nil {
						return false, activeTimeoutTerminals, checkErr
					}
					if allowed {
						if cleanErr := m.cleanTokenMetadata(ctx, []string{ti.Token}); cleanErr != nil {
							return false, activeTimeoutTerminals, cleanErr
						}
					}
				}
				hasExpired = true
				continue
			}
			return false, activeTimeoutTerminals, err
		}
		tokenInfo := &record.TokenInfo
		if !terminalMatchesTokenRecord(sess.LoginID, ti, record) {
			// Detach mismatched terminal metadata without mutating the canonical token mapping. 移除错位终端元数据，但不改写作为身份真值的 Token 映射。
			hasExpired = true
			continue
		}

		if record.Revoked {
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
		pendingActiveAt := m.pendingLoginActiveAt(ti.Token, record)
		activeValue, activeErr := m.storage.Get(ctx, m.getActiveKey(ti.Token))
		if activeErr != nil {
			return false, activeTimeoutTerminals, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, activeErr)
		}
		var activeAt int64
		var convertErr error
		if activeValue != nil {
			activeAt, convertErr = utils.ToInt64(activeValue)
		}
		if activeValue == nil || convertErr != nil || activeAt < 0 {
			if deleteErr := m.applyLogoutModeToToken(ctx, ti.Token, record, config.LogoutModeLogout); deleteErr != nil {
				return false, activeTimeoutTerminals, deleteErr
			}
			if cleanErr := m.cleanTokenMetadata(ctx, []string{ti.Token}); cleanErr != nil {
				return false, activeTimeoutTerminals, cleanErr
			}
			hasExpired = true
			continue
		}

		activeAt = max(activeAt, pendingActiveAt)
		if time.Now().Unix()-activeAt > activeTimeout {
			if stateErr := m.setTokenStateWithRecord(ctx, ti.Token, record, TokenStateActiveTimeout, m.tokenStateExpiration(ctx, ti.Token, record)); stateErr != nil {
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

	// Leave persistence to the login boundary so pruning and insertion share one write. 持久化交给登录边界，使整理与新增终端共用一次写入。
	if hasExpired {
		sess.TerminalInfos = validTerminals
		return len(validTerminals) == 0, activeTimeoutTerminals, nil
	}

	return false, activeTimeoutTerminals, nil
}
