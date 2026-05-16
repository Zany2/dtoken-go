// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"context"
	"errors"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"strings"
)

// GetSession retrieves session information for a login ID. GetSession 获取指定登录 ID 的会话信息。
func (m *Manager) GetSession(ctx context.Context, loginID string) (*Session, error) {
	if loginID == "" {
		return nil, derror.ErrIDIsEmpty
	}
	return m.getSession(ctx, loginID)
}

// GetSessionByToken retrieves session information by token. GetSessionByToken 通过 Token 值获取会话信息。
func (m *Manager) GetSessionByToken(ctx context.Context, tokenValue string) (*Session, error) {
	if tokenValue == "" {
		return nil, derror.ErrInvalidToken
	}

	// Get tokenInfo 获取 tokenInfo
	tokenInfo, err := m.getTokenInfo(ctx, tokenValue)
	if err != nil {
		return nil, err
	}

	sess, err := m.getSession(ctx, tokenInfo.LoginID)
	if err != nil {
		if errors.Is(err, derror.ErrSessionNotFound) {
			return nil, derror.ErrInvalidToken
		}
		return nil, err
	}
	if sess == nil || !sess.hasTerminalToken(tokenValue) {
		return nil, derror.ErrInvalidToken
	}

	return sess, nil
}

// GetTokenValueListByLoginID retrieves all tokens for a login ID. GetTokenValueListByLoginID 获取指定登录 ID 的所有 Token。
func (m *Manager) GetTokenValueListByLoginID(ctx context.Context, loginID string, checkAlive ...bool) ([]string, error) {
	if loginID == "" {
		return nil, derror.ErrIDIsEmpty
	}

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		// Return errors only for real storage failures 仅当存储层真正出错时才返回 error；session 不存在视为 nil
		if !errors.Is(err, derror.ErrSessionNotFound) {
			return nil, err
		}
		return []string{}, nil
	}
	if sess == nil {
		return []string{}, nil
	}

	return m.filterTokens(ctx, sess.TerminalInfos, len(checkAlive) > 0 && checkAlive[0])
}

// GetTokenValueListByDevice retrieves all tokens for a specific device type. GetTokenValueListByDevice 获取指定设备类型的所有 Token。
func (m *Manager) GetTokenValueListByDevice(ctx context.Context, loginID, device string, checkAlive ...bool) ([]string, error) {
	if loginID == "" {
		return []string{}, derror.ErrIDIsEmpty
	}
	device = strings.TrimSpace(device)
	if device == "" {
		return []string{}, derror.ErrInvalidParam
	}

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		if !errors.Is(err, derror.ErrSessionNotFound) {
			return nil, err
		}
		return []string{}, nil
	}
	if sess == nil {
		return []string{}, nil
	}

	matched := sess.getTerminalsByDevice(device)
	return m.filterTokens(ctx, matched, len(checkAlive) > 0 && checkAlive[0])
}

// GetTokenValueListByDeviceAndDeviceId retrieves all tokens for a specific device type and device ID. GetTokenValueListByDeviceAndDeviceId 获取指定设备类型和设备 ID 的所有 Token。
func (m *Manager) GetTokenValueListByDeviceAndDeviceId(ctx context.Context, loginID, device, deviceId string, checkAlive ...bool) ([]string, error) {
	if loginID == "" {
		return []string{}, derror.ErrIDIsEmpty
	}
	device = strings.TrimSpace(device)
	deviceId = strings.TrimSpace(deviceId)
	if device == "" || deviceId == "" {
		return []string{}, derror.ErrInvalidParam
	}

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		if !errors.Is(err, derror.ErrSessionNotFound) {
			return nil, err
		}
		return []string{}, nil
	}
	if sess == nil {
		return []string{}, nil
	}

	matched := sess.getTerminalsByDeviceAndDeviceId(device, deviceId)
	return m.filterTokens(ctx, matched, len(checkAlive) > 0 && checkAlive[0])
}

// GetOnlineTerminalCount retrieves the count of online terminals for a user. GetOnlineTerminalCount 获取用户的在线终端数量。
func (m *Manager) GetOnlineTerminalCount(ctx context.Context, loginID string) (int, error) {
	if loginID == "" {
		return 0, derror.ErrIDIsEmpty
	}

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		if errors.Is(err, derror.ErrSessionNotFound) {
			return 0, nil
		}
		return 0, err
	}
	if sess == nil {
		return 0, nil
	}

	return m.countAliveTokens(ctx, sess.TerminalInfos)
}

// GetOnlineTerminalCountByDevice retrieves the count of online terminals for a specific device type. GetOnlineTerminalCountByDevice 获取用户在指定设备类型的在线终端数量。
func (m *Manager) GetOnlineTerminalCountByDevice(ctx context.Context, loginID, device string) (int, error) {
	if loginID == "" {
		return 0, derror.ErrIDIsEmpty
	}
	device = strings.TrimSpace(device)
	if device == "" {
		return 0, derror.ErrInvalidParam
	}

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		if errors.Is(err, derror.ErrSessionNotFound) {
			return 0, nil
		}
		return 0, err
	}
	if sess == nil {
		return 0, nil
	}

	return m.countAliveTokens(ctx, sess.getTerminalsByDevice(device))
}

// GetOnlineTerminalCountByDeviceAndDeviceId retrieves the count of online terminals for a specific device type and device ID. GetOnlineTerminalCountByDeviceAndDeviceId 获取用户在指定设备类型和设备ID的在线终端数量。
func (m *Manager) GetOnlineTerminalCountByDeviceAndDeviceId(ctx context.Context, loginID, device, deviceId string) (int, error) {
	if loginID == "" {
		return 0, derror.ErrIDIsEmpty
	}
	device = strings.TrimSpace(device)
	deviceId = strings.TrimSpace(deviceId)
	if device == "" || deviceId == "" {
		return 0, derror.ErrInvalidParam
	}

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		if errors.Is(err, derror.ErrSessionNotFound) {
			return 0, nil
		}
		return 0, err
	}
	if sess == nil {
		return 0, nil
	}

	return m.countAliveTokens(ctx, sess.getTerminalsByDeviceAndDeviceId(device, deviceId))
}

// GetTerminalListByLoginID retrieves all terminal info for a login ID, optionally filtered by device. GetTerminalListByLoginID 获取指定登录 ID 的所有终端信息列表，可选按设备类型过滤。
func (m *Manager) GetTerminalListByLoginID(ctx context.Context, loginID string, device ...string) ([]TerminalInfo, error) {
	if loginID == "" {
		return nil, derror.ErrIDIsEmpty
	}

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		if errors.Is(err, derror.ErrSessionNotFound) {
			return []TerminalInfo{}, nil
		}
		return nil, err
	}
	if sess == nil {
		return []TerminalInfo{}, nil
	}

	if len(device) > 0 {
		targetDevice := strings.TrimSpace(device[0])
		if targetDevice != "" {
			return sess.getTerminalsByDevice(targetDevice), nil
		}
	}

	// Return copy to avoid external mutation 返回副本，避免外部修改影响内部数据
	result := make([]TerminalInfo, len(sess.TerminalInfos))
	copy(result, sess.TerminalInfos)
	return result, nil
}

// GetTerminalInfoByToken retrieves terminal info for a specific token. GetTerminalInfoByToken 根据 Token 获取终端详情。
func (m *Manager) GetTerminalInfoByToken(ctx context.Context, tokenValue string) (*TerminalInfo, error) {
	if tokenValue == "" {
		return nil, derror.ErrInvalidToken
	}

	sess, _, err := m.getCheckedTokenSession(ctx, tokenValue)
	if err != nil {
		return nil, err
	}

	for _, ti := range sess.TerminalInfos {
		if ti.Token == tokenValue {
			return &ti, nil
		}
	}

	return nil, derror.ErrInvalidToken
}

// GetTokenValueByLoginID retrieves the latest token for a login ID, optionally filtered by device. GetTokenValueByLoginID 获取指定登录 ID 的最新 Token，可选按设备类型过滤。
func (m *Manager) GetTokenValueByLoginID(ctx context.Context, loginID string, device ...string) (string, error) {
	if loginID == "" {
		return "", derror.ErrIDIsEmpty
	}

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		return "", err
	}

	terminals := sess.TerminalInfos
	if len(device) > 0 {
		if targetDevice := strings.TrimSpace(device[0]); targetDevice != "" {
			terminals = sess.getTerminalsByDevice(targetDevice)
		}
	}

	// Walk backward so the newest alive token wins. 反向遍历以返回最新仍有效的 token。
	for i := len(terminals) - 1; i >= 0; i-- {
		alive, err := m.checkTerminalTokenAlive(ctx, terminals[i].Token)
		if err != nil {
			return "", err
		}
		if alive {
			return terminals[i].Token, nil
		}
	}

	return "", derror.ErrInvalidToken
}

// SearchTokenValue searches token values by keyword with pagination. SearchTokenValue 根据关键词搜索 Token 值，支持分页。 keyword: 搜索关键词（模糊匹配），start: 起始索引，size: 返回数量（-1 返回全部）
func (m *Manager) SearchTokenValue(ctx context.Context, keyword string, start, size int) ([]string, error) {
	prefix := m.config.KeyPrefix + m.config.AuthType + config.TokenKeyPrefix
	pattern := prefix + "*" + keyword + "*"
	return m.searchValues(ctx, pattern, prefix, start, size)
}

// SearchSessionId searches session IDs by keyword with pagination. SearchSessionId 根据关键词搜索 Session ID，支持分页。 keyword: 搜索关键词（模糊匹配），start: 起始索引，size: 返回数量（-1 返回全部）
func (m *Manager) SearchSessionId(ctx context.Context, keyword string, start, size int) ([]string, error) {
	prefix := m.config.KeyPrefix + m.config.AuthType + SessionKeyPrefix
	pattern := prefix + "*" + keyword + "*"
	return m.searchValues(ctx, pattern, prefix, start, size)
}

// TerminalVisitor is a callback function for terminal traversal. TerminalVisitor 终端遍历回调函数。 Return false to stop traversal. 返回 false 停止遍历。
type TerminalVisitor func(terminal TerminalInfo) bool

// ForEachTerminal iterates over all terminals for a login ID and calls the visitor function. ForEachTerminal 遍历指定登录 ID 的所有终端，对每个终端调用回调函数。
func (m *Manager) ForEachTerminal(ctx context.Context, loginID string, visitor TerminalVisitor) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}
	if visitor == nil {
		return derror.ErrInvalidParam
	}

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		if errors.Is(err, derror.ErrSessionNotFound) {
			return nil
		}
		return err
	}

	for _, ti := range sess.TerminalInfos {
		if !visitor(ti) {
			break
		}
	}
	return nil
}

// ForEachTerminalByDevice iterates over terminals filtered by device type. ForEachTerminalByDevice 遍历指定设备类型的终端。
func (m *Manager) ForEachTerminalByDevice(ctx context.Context, loginID, device string, visitor TerminalVisitor) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}
	device = strings.TrimSpace(device)
	if device == "" {
		return derror.ErrInvalidParam
	}
	if visitor == nil {
		return derror.ErrInvalidParam
	}

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		if errors.Is(err, derror.ErrSessionNotFound) {
			return nil
		}
		return err
	}

	for _, ti := range sess.TerminalInfos {
		if ti.Device == device {
			if !visitor(ti) {
				break
			}
		}
	}
	return nil
}

// filterTokens filters tokens based on checkAlive flag (internal method). filterTokens 根据 checkAlive 决定是否验证 token 有效性，并返回 token 列表（内部方法）。
func (m *Manager) filterTokens(ctx context.Context, terminals []TerminalInfo, checkAlive bool) ([]string, error) {
	if len(terminals) == 0 {
		return []string{}, nil
	}

	if !checkAlive {
		// Return all tokens without alive check 不检查存活：直接返回所有 token（预分配容量）
		tokens := make([]string, len(terminals))
		for i, ti := range terminals {
			tokens[i] = ti.Token
		}
		return tokens, nil
	}

	// Check each token by full alive rules 按完整存活规则检查每个 token
	tokens := make([]string, 0, len(terminals))
	for _, ti := range terminals {
		alive, err := m.checkTerminalTokenAlive(ctx, ti.Token)
		if err != nil {
			return nil, err
		}
		if alive {
			tokens = append(tokens, ti.Token)
		}
		// Skip invalid tokens 若 token 无效（过期/被踢等），跳过
	}
	return tokens, nil
}

// countAliveTokens counts alive tokens without collecting token values. countAliveTokens 不收集 token 直接统计存活数量。
func (m *Manager) countAliveTokens(ctx context.Context, terminals []TerminalInfo) (int, error) {
	if len(terminals) == 0 {
		return 0, nil
	}

	count := 0
	for _, ti := range terminals {
		alive, err := m.checkTerminalTokenAlive(ctx, ti.Token)
		if err != nil {
			return 0, err
		}
		if alive {
			count++
		}
	}
	return count, nil
}
