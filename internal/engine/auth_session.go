// @Author daixk 2025/12/22 15:56:00
package engine

import (
	"context"
	"errors"
	"strings"

	"github.com/Zany2/dtoken-go/config"
	derror "github.com/Zany2/dtoken-go/errors"
)

// GetSession retrieves session information for a login ID. GetSession 获取指定登录 ID 的会话信息。
func (m *Manager) GetSession(ctx context.Context, loginID string) (*Session, error) {
	if m.IsClosed() {
		return nil, derror.ErrManagerNotFound
	}

	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return nil, derror.ErrIDIsEmpty
	}

	// Load session 加载会话。
	return m.getSession(ctx, loginID)
}

// GetSessionByToken retrieves session information by token. GetSessionByToken 通过 Token 值获取会话信息。
func (m *Manager) GetSessionByToken(ctx context.Context, tokenValue string) (*Session, error) {
	if m.IsClosed() {
		return nil, derror.ErrManagerNotFound
	}

	// Validate token value 校验 Token 值。
	if tokenValue == "" {
		return nil, derror.ErrInvalidToken
	}

	// Validate login context without renew side effects 校验登录上下文但不触发续期副作用。
	sess, _, err := m.checkLoginAndGetContextNoRenew(ctx, tokenValue)
	if err != nil {
		return nil, err
	}

	// Return session 返回会话。
	return sess, nil
}

// GetTokenValueListByLoginID retrieves all tokens for a login ID. GetTokenValueListByLoginID 获取指定登录 ID 的所有 Token。
func (m *Manager) GetTokenValueListByLoginID(ctx context.Context, loginID string, checkAlive ...bool) ([]string, error) {
	if m.IsClosed() {
		return nil, derror.ErrManagerNotFound
	}

	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return nil, derror.ErrIDIsEmpty
	}
	if len(checkAlive) > 1 {
		return nil, derror.ErrInvalidParam
	}

	// Load session 加载会话。
	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		// Ignore missing sessions and propagate other storage errors. 忽略不存在的会话，仅返回其他存储错误。
		if !errors.Is(err, derror.ErrSessionNotFound) {
			return nil, err
		}

		// Return empty list 返回空列表。
		return []string{}, nil
	}

	// Return empty list when session missing 会话为空时返回空列表。
	if sess == nil {
		return []string{}, nil
	}

	// Filter terminal tokens 过滤终端 Token。
	return m.filterTokens(ctx, sess, len(checkAlive) > 0 && checkAlive[0], "", "")
}

// GetTokenValueListByDevice retrieves all tokens for a specific device type. GetTokenValueListByDevice 获取指定设备类型的所有 Token。
func (m *Manager) GetTokenValueListByDevice(ctx context.Context, loginID, device string, checkAlive ...bool) ([]string, error) {
	if m.IsClosed() {
		return nil, derror.ErrManagerNotFound
	}

	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return []string{}, derror.ErrIDIsEmpty
	}
	if len(checkAlive) > 1 {
		return nil, derror.ErrInvalidParam
	}

	// Normalize device type 规范化设备类型。
	device = strings.TrimSpace(device)

	// Validate device type 校验设备类型。
	if device == "" {
		return []string{}, derror.ErrInvalidParam
	}

	// Load session 加载会话。
	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		// Ignore missing session 忽略不存在的会话。
		if !errors.Is(err, derror.ErrSessionNotFound) {
			return nil, err
		}

		// Return empty list 返回空列表。
		return []string{}, nil
	}

	// Return empty list when session missing 会话为空时返回空列表。
	if sess == nil {
		return []string{}, nil
	}

	// Filter during traversal without copying matching terminal records. 遍历时直接筛选，不复制匹配的终端记录。
	return m.filterTokens(ctx, sess, len(checkAlive) > 0 && checkAlive[0], device, "")
}

// GetTokenValueListByDeviceAndDeviceID retrieves all tokens for a specific device type and device ID. GetTokenValueListByDeviceAndDeviceID 获取指定设备类型和设备 ID 的所有 Token。
func (m *Manager) GetTokenValueListByDeviceAndDeviceID(ctx context.Context, loginID, device, deviceID string, checkAlive ...bool) ([]string, error) {
	if m.IsClosed() {
		return nil, derror.ErrManagerNotFound
	}

	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return []string{}, derror.ErrIDIsEmpty
	}
	if len(checkAlive) > 1 {
		return nil, derror.ErrInvalidParam
	}

	// Normalize device fields 规范化设备字段。
	device = strings.TrimSpace(device)
	deviceID = strings.TrimSpace(deviceID)

	// Validate device fields 校验设备字段。
	if device == "" || deviceID == "" {
		return []string{}, derror.ErrInvalidParam
	}

	// Load session 加载会话。
	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		// Ignore missing session 忽略不存在的会话。
		if !errors.Is(err, derror.ErrSessionNotFound) {
			return nil, err
		}

		// Return empty list 返回空列表。
		return []string{}, nil
	}

	// Return empty list when session missing 会话为空时返回空列表。
	if sess == nil {
		return []string{}, nil
	}

	// Filter during traversal without copying matching terminal records. 遍历时直接筛选，不复制匹配的终端记录。
	return m.filterTokens(ctx, sess, len(checkAlive) > 0 && checkAlive[0], device, deviceID)
}

// GetOnlineTerminalCount retrieves the count of online terminals for a user. GetOnlineTerminalCount 获取用户的在线终端数量。
func (m *Manager) GetOnlineTerminalCount(ctx context.Context, loginID string) (int, error) {
	if m.IsClosed() {
		return 0, derror.ErrManagerNotFound
	}

	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return 0, derror.ErrIDIsEmpty
	}

	// Load session 加载会话。
	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		// Treat missing session as zero count 会话不存在视为 0。
		if errors.Is(err, derror.ErrSessionNotFound) {
			return 0, nil
		}
		return 0, err
	}

	// Return zero when session missing 会话为空时返回 0。
	if sess == nil {
		return 0, nil
	}

	// Count alive tokens 统计存活 Token。
	return m.countAliveTokens(ctx, sess, "", "")
}

// GetOnlineTerminalCountByDevice retrieves the count of online terminals for a specific device type. GetOnlineTerminalCountByDevice 获取用户在指定设备类型的在线终端数量。
func (m *Manager) GetOnlineTerminalCountByDevice(ctx context.Context, loginID, device string) (int, error) {
	if m.IsClosed() {
		return 0, derror.ErrManagerNotFound
	}

	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return 0, derror.ErrIDIsEmpty
	}

	// Normalize device type 规范化设备类型。
	device = strings.TrimSpace(device)

	// Validate device type 校验设备类型。
	if device == "" {
		return 0, derror.ErrInvalidParam
	}

	// Load session 加载会话。
	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		// Treat missing session as zero count 会话不存在视为 0。
		if errors.Is(err, derror.ErrSessionNotFound) {
			return 0, nil
		}
		return 0, err
	}

	// Return zero when session missing 会话为空时返回 0。
	if sess == nil {
		return 0, nil
	}

	// Count alive tokens by device 按设备统计存活 Token。
	return m.countAliveTokens(ctx, sess, device, "")
}

// GetOnlineTerminalCountByDeviceAndDeviceID retrieves the count of online terminals for a specific device type and device ID. GetOnlineTerminalCountByDeviceAndDeviceID 获取用户在指定设备类型和设备ID的在线终端数量。
func (m *Manager) GetOnlineTerminalCountByDeviceAndDeviceID(ctx context.Context, loginID, device, deviceID string) (int, error) {
	if m.IsClosed() {
		return 0, derror.ErrManagerNotFound
	}

	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return 0, derror.ErrIDIsEmpty
	}

	// Normalize device fields 规范化设备字段。
	device = strings.TrimSpace(device)
	deviceID = strings.TrimSpace(deviceID)

	// Validate device fields 校验设备字段。
	if device == "" || deviceID == "" {
		return 0, derror.ErrInvalidParam
	}

	// Load session 加载会话。
	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		// Treat missing session as zero count 会话不存在视为 0。
		if errors.Is(err, derror.ErrSessionNotFound) {
			return 0, nil
		}
		return 0, err
	}

	// Return zero when session missing 会话为空时返回 0。
	if sess == nil {
		return 0, nil
	}

	// Count alive tokens by concrete device 按具体设备统计存活 Token。
	return m.countAliveTokens(ctx, sess, device, deviceID)
}

// GetTerminalListByLoginID retrieves all terminal info for a login ID, optionally filtered by device. GetTerminalListByLoginID 获取指定登录 ID 的所有终端信息列表，可选按设备类型过滤。
func (m *Manager) GetTerminalListByLoginID(ctx context.Context, loginID string, device ...string) ([]TerminalInfo, error) {
	if m.IsClosed() {
		return nil, derror.ErrManagerNotFound
	}

	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return nil, derror.ErrIDIsEmpty
	}
	if len(device) > 1 {
		return nil, derror.ErrInvalidParam
	}

	// Validate the optional filter before reading session state. 读取会话状态前校验可选过滤条件。
	var targetDevice string
	if len(device) > 0 {
		targetDevice = strings.TrimSpace(device[0])
		if targetDevice == "" {
			return nil, derror.ErrInvalidParam
		}
	}

	// Load session 加载会话。
	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		// Treat missing session as empty list 会话不存在视为空列表。
		if errors.Is(err, derror.ErrSessionNotFound) {
			return []TerminalInfo{}, nil
		}
		return nil, err
	}

	// Return empty list when session missing 会话为空时返回空列表。
	if sess == nil {
		return []TerminalInfo{}, nil
	}

	// Filter by optional device 按可选设备过滤。
	if len(device) > 0 {
		return sessionGetTerminalsByDevice(sess, targetDevice), nil
	}

	// Return a copy of the terminal list to prevent external mutation. 返回终端列表副本，避免外部修改影响内部数据。
	result := make([]TerminalInfo, len(sess.TerminalInfos))
	copy(result, sess.TerminalInfos)
	return result, nil
}

// GetTerminalInfoByToken retrieves terminal info for a specific token. GetTerminalInfoByToken 根据 Token 获取终端详情。
func (m *Manager) GetTerminalInfoByToken(ctx context.Context, tokenValue string) (*TerminalInfo, error) {
	if m.IsClosed() {
		return nil, derror.ErrManagerNotFound
	}

	// Validate token value 校验 Token 值。
	if tokenValue == "" {
		return nil, derror.ErrInvalidToken
	}

	// Capture lifecycle identity before validating the account context. 校验账号上下文前捕获生命周期身份。
	record, err := m.getTokenRecord(ctx, tokenValue)
	if err != nil {
		return nil, err
	}
	sess, tokenInfo, err := m.checkLoginAndGetContextWithOptions(ctx, tokenValue, checkLoginOptions{expectedRecord: record})
	if err != nil {
		return nil, err
	}

	// Search terminal info 查找终端信息。
	for _, ti := range sess.TerminalInfos {
		if ti.Token == tokenValue && terminalMatchesTokenRecord(sess.LoginID, ti, record) {
			// Only successful terminal queries count as activity. 仅成功的终端查询触发活跃维护。
			m.submitLoginMaintenance(ctx, tokenValue, record, m.resolveActiveTimeoutFromSeconds(tokenInfo.ActiveTimeout))
			return &ti, nil
		}
	}

	return nil, derror.ErrInvalidToken
}

// GetTokenValueByLoginID retrieves the latest token for a login ID, optionally filtered by device. GetTokenValueByLoginID 获取指定登录 ID 的最新 Token，可选按设备类型过滤。
func (m *Manager) GetTokenValueByLoginID(ctx context.Context, loginID string, device ...string) (string, error) {
	if m.IsClosed() {
		return "", derror.ErrManagerNotFound
	}

	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return "", derror.ErrIDIsEmpty
	}
	if len(device) > 1 {
		return "", derror.ErrInvalidParam
	}

	// Validate the optional filter before reading session state. 读取会话状态前校验可选过滤条件。
	var targetDevice string
	if len(device) > 0 {
		targetDevice = strings.TrimSpace(device[0])
		if targetDevice == "" {
			return "", derror.ErrInvalidParam
		}
	}

	// Load session 加载会话。
	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		// Treat missing session as no alive token 会话不存在视为无有效 Token
		if errors.Is(err, derror.ErrSessionNotFound) {
			return "", derror.ErrInvalidToken
		}
		return "", err
	}

	// Filter while walking backward so a match stops without collecting older terminals. 反向遍历时直接筛选，命中即停止，不收集较旧终端。
	cache := &terminalAliveCheckCache{}
	for i := len(sess.TerminalInfos) - 1; i >= 0; i-- {
		terminal := &sess.TerminalInfos[i]
		if targetDevice != "" && terminal.Device != targetDevice {
			continue
		}
		alive, err := m.checkTerminalEntryAlive(ctx, *terminal, sess, cache)
		if err != nil {
			return "", err
		}
		if alive {
			return terminal.Token, nil
		}
	}

	return "", derror.ErrInvalidToken
}

// SearchTokenValue searches token values by keyword with pagination. SearchTokenValue 按关键词分页搜索 Token 值，size 为 -1 时返回全部。
func (m *Manager) SearchTokenValue(ctx context.Context, keyword string, start, size int) ([]string, error) {
	if m.IsClosed() {
		return nil, derror.ErrManagerNotFound
	}

	// Treat both the namespace and keyword as literals in the scan pattern. 扫描模式中的命名空间和关键词均按字面量处理。
	prefix := m.storageNamespace() + config.TokenKeyPrefix
	pattern := escapeSearchKeyword(prefix) + "*" + escapeSearchKeyword(keyword) + "*"
	return m.searchValues(ctx, pattern, prefix, start, size)
}

// SearchSessionId searches session IDs by keyword with pagination. SearchSessionId 按关键词分页搜索 Session ID，size 为 -1 时返回全部。
func (m *Manager) SearchSessionId(ctx context.Context, keyword string, start, size int) ([]string, error) {
	if m.IsClosed() {
		return nil, derror.ErrManagerNotFound
	}

	// Treat both the namespace and keyword as literals in the scan pattern. 扫描模式中的命名空间和关键词均按字面量处理。
	prefix := m.storageNamespace() + SessionKeyPrefix
	pattern := escapeSearchKeyword(prefix) + "*" + escapeSearchKeyword(keyword) + "*"
	return m.searchValues(ctx, pattern, prefix, start, size)
}

// escapeSearchKeyword escapes literal text embedded in a storage scan pattern. escapeSearchKeyword 转义嵌入存储扫描模式的字面量文本。
func escapeSearchKeyword(keyword string) string {
	keyword = strings.ReplaceAll(keyword, "\\", "\\\\")
	keyword = strings.ReplaceAll(keyword, "*", "\\*")
	keyword = strings.ReplaceAll(keyword, "?", "\\?")

	// Redis also recognizes character classes in scan patterns. Redis 扫描模式还支持字符集合。
	keyword = strings.ReplaceAll(keyword, "[", "\\[")
	keyword = strings.ReplaceAll(keyword, "]", "\\]")
	return keyword
}

// TerminalVisitor visits terminals and stops when the callback returns false. TerminalVisitor 遍历终端，回调返回 false 时停止。
type TerminalVisitor func(terminal TerminalInfo) bool

// ForEachTerminal iterates over all terminals for a login ID and calls the visitor function. ForEachTerminal 遍历指定登录 ID 的所有终端，对每个终端调用回调函数。
func (m *Manager) ForEachTerminal(ctx context.Context, loginID string, visitor TerminalVisitor) error {
	if m.IsClosed() {
		return derror.ErrManagerNotFound
	}

	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	// Validate visitor 校验回调函数。
	if visitor == nil {
		return derror.ErrInvalidParam
	}

	// Load session 加载会话。
	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		// Ignore missing session 忽略不存在的会话。
		if errors.Is(err, derror.ErrSessionNotFound) {
			return nil
		}
		return err
	}

	// Visit each terminal 遍历每个终端。
	for _, ti := range sess.TerminalInfos {
		if !visitor(ti) {
			break
		}
	}
	return nil
}

// ForEachTerminalByDevice iterates over terminals filtered by device type. ForEachTerminalByDevice 遍历指定设备类型的终端。
func (m *Manager) ForEachTerminalByDevice(ctx context.Context, loginID, device string, visitor TerminalVisitor) error {
	if m.IsClosed() {
		return derror.ErrManagerNotFound
	}

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

	// Validate visitor 校验回调函数。
	if visitor == nil {
		return derror.ErrInvalidParam
	}

	// Load session 加载会话。
	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		// Ignore missing session 忽略不存在的会话。
		if errors.Is(err, derror.ErrSessionNotFound) {
			return nil
		}
		return err
	}

	// Visit matched terminals 遍历匹配终端。
	for _, ti := range sess.TerminalInfos {
		if ti.Device == device {
			if !visitor(ti) {
				break
			}
		}
	}
	return nil
}

// filterTokens collects scoped tokens directly; empty private filters mean all devices. filterTokens 直接收集作用域内 Token，私有过滤参数为空表示不限设备。
func (m *Manager) filterTokens(ctx context.Context, sess *Session, checkAlive bool, device, deviceID string) ([]string, error) {
	// Keep the unfiltered snapshot path as a direct copy of token strings. 未筛选的快照路径直接复制 Token 字符串。
	if !checkAlive && device == "" && deviceID == "" {
		tokens := make([]string, len(sess.TerminalInfos))
		for i := range sess.TerminalInfos {
			tokens[i] = sess.TerminalInfos[i].Token
		}
		return tokens, nil
	}

	// Avoid allocating for unrelated terminals in a narrow query. 窄范围查询不为无关终端预分配结果空间。
	tokens := []string{}
	if device == "" {
		tokens = make([]string, 0, len(sess.TerminalInfos))
	}
	cache := &terminalAliveCheckCache{}
	for i := range sess.TerminalInfos {
		ti := &sess.TerminalInfos[i]
		if device != "" && ti.Device != device || deviceID != "" && ti.DeviceID != deviceID {
			continue
		}
		if checkAlive {
			alive, err := m.checkTerminalEntryAlive(ctx, *ti, sess, cache)
			if err != nil {
				return nil, err
			}
			if !alive {
				continue
			}
		}
		tokens = append(tokens, ti.Token)
	}
	return tokens, nil
}

// countAliveTokens counts scoped live entries without collecting terminals or token values. countAliveTokens 直接统计作用域内存活条目，不收集终端或 Token 列表。
func (m *Manager) countAliveTokens(ctx context.Context, sess *Session, device, deviceID string) (int, error) {
	count := 0
	cache := &terminalAliveCheckCache{}
	for i := range sess.TerminalInfos {
		ti := &sess.TerminalInfos[i]
		if device != "" && ti.Device != device || deviceID != "" && ti.DeviceID != deviceID {
			continue
		}
		alive, err := m.checkTerminalEntryAlive(ctx, *ti, sess, cache)
		if err != nil {
			return 0, err
		}
		if alive {
			count++
		}
	}
	return count, nil
}
