// @Author daixk 2026/1/22 17:32:00
package manager

import "fmt"

// ============================================================================
// Terminal Management - 终端管理
// ============================================================================

// ----------------------------------------------------------------------------
// Terminal Removal Methods - 终端移除方法
// ----------------------------------------------------------------------------

// removeTerminalByToken removes a terminal from TerminalInfos by token value (at most one match).
// removeTerminalByToken 根据 token 值删除 TerminalInfos 中匹配的项（最多一个）。
func (s *Session) removeTerminalByToken(tokenValue string) (TerminalInfo, bool) {
	if tokenValue == "" {
		return TerminalInfo{}, false
	}

	for i, ti := range s.TerminalInfos {
		if ti.Token == tokenValue {
			removed := ti
			// 拼接 [0:i] + [i+1:]
			s.TerminalInfos = append(s.TerminalInfos[:i], s.TerminalInfos[i+1:]...)
			return removed, true
		}
	}

	return TerminalInfo{}, false
}

// removeTerminalByDevice removes all terminals from TerminalInfos that match the device type.
// removeTerminalByDevice 根据设备类型删除 TerminalInfos 中的所有匹配项。
func (s *Session) removeTerminalByDevice(device string) []TerminalInfo {
	var kept []TerminalInfo    // 保留的项
	var removed []TerminalInfo // 被删除的项

	for _, ti := range s.TerminalInfos {
		if ti.Device == device {
			removed = append(removed, ti)
		} else {
			kept = append(kept, ti)
		}
	}

	fmt.Println(kept)
	fmt.Println(removed)
	s.TerminalInfos = kept
	return removed
}

// removeTerminalByDeviceAndDeviceId removes all terminals that exactly match both device and deviceId.
// removeTerminalByDeviceAndDeviceId 移除所有精确匹配 device 和 deviceId 的 TerminalInfo。
func (s *Session) removeTerminalByDeviceAndDeviceId(device, deviceId string) []TerminalInfo {
	var kept []TerminalInfo
	var removed []TerminalInfo

	for _, ti := range s.TerminalInfos {
		if ti.Device == device && ti.DeviceId == deviceId {
			removed = append(removed, ti)
		} else {
			kept = append(kept, ti)
		}
	}

	s.TerminalInfos = kept
	return removed
}

// removeOldestTerminal removes the oldest terminal (optionally filtered by device).
// removeOldestTerminal 移除最老终端（可选按 device 过滤）。
func (s *Session) removeOldestTerminal(device ...string) (TerminalInfo, bool) {
	if len(s.TerminalInfos) == 0 {
		return TerminalInfo{}, false
	}

	if len(device) == 0 {
		first := s.TerminalInfos[0]
		s.TerminalInfos = s.TerminalInfos[1:]
		return first, true
	}

	// 有设备过滤：查找第一个匹配 device[0] 的项
	targetDevice := device[0]
	for i, ti := range s.TerminalInfos {
		if ti.Device == targetDevice {
			// 找到第一个匹配项，移除它
			removed := ti
			// 保持顺序：拼接 [0:i] + [i+1:]
			s.TerminalInfos = append(s.TerminalInfos[:i], s.TerminalInfos[i+1:]...)
			return removed, true
		}
	}

	// 未找到匹配项
	return TerminalInfo{}, false
}

// ----------------------------------------------------------------------------
// Terminal Query Methods - 终端查询方法
// ----------------------------------------------------------------------------

// getTerminalsByDevice returns all terminals that match the specified device type.
// getTerminalsByDevice 返回所有匹配指定 device 的 TerminalInfo。
func (s *Session) getTerminalsByDevice(device string) []TerminalInfo {
	var matched []TerminalInfo
	for _, ti := range s.TerminalInfos {
		if ti.Device == device {
			matched = append(matched, ti)
		}
	}
	return matched
}

// getTerminalsByDeviceAndDeviceId returns all terminals that exactly match both device and deviceId.
// getTerminalsByDeviceAndDeviceId 返回所有精确匹配 device 和 deviceId 的 TerminalInfo。
func (s *Session) getTerminalsByDeviceAndDeviceId(device, deviceId string) []TerminalInfo {
	var matched []TerminalInfo
	for _, ti := range s.TerminalInfos {
		if ti.Device == device && ti.DeviceId == deviceId {
			matched = append(matched, ti)
		}
	}
	return matched
}

// getLatestTerminalByDevice retrieves the latest terminal for the specified device.
// getLatestTerminalByDevice 获取指定 device 下最新的 TerminalInfo。
func (s *Session) getLatestTerminalByDevice(device string) (TerminalInfo, bool) {
	for i := len(s.TerminalInfos) - 1; i >= 0; i-- {
		if s.TerminalInfos[i].Device == device {
			return s.TerminalInfos[i], true
		}
	}
	return TerminalInfo{}, false
}

// ============================================================================
// Permission Management - 权限管理
// ============================================================================

// addPermissions adds a set of permissions to the session with automatic deduplication.
// addPermissions 向会话中添加一组权限（自动去重）。
func (s *Session) addPermissions(permissions ...string) {
	if len(permissions) == 0 {
		return
	}

	// 构建现有权限的集合（用于去重）
	existing := make(map[string]struct{}, len(s.Permissions))
	for _, p := range s.Permissions {
		existing[p] = struct{}{}
	}

	// 添加新权限（跳过已存在的）
	for _, p := range permissions {
		if p == "" {
			continue // 跳过空权限
		}
		if _, exists := existing[p]; !exists {
			existing[p] = struct{}{}
			s.Permissions = append(s.Permissions, p)
		}
	}
}

// removePermissions removes a set of permissions from the session.
// removePermissions 从会话中移除一组权限。
func (s *Session) removePermissions(permissions ...string) {
	if len(permissions) == 0 || len(s.Permissions) == 0 {
		return
	}

	// 构建要删除的权限集合（去重 + 忽略空）
	toRemove := make(map[string]struct{}, len(permissions))
	for _, p := range permissions {
		if p != "" {
			toRemove[p] = struct{}{}
		}
	}

	// 过滤保留不在 toRemove 中的权限
	var kept []string
	for _, p := range s.Permissions {
		if _, shouldRemove := toRemove[p]; !shouldRemove {
			kept = append(kept, p)
		}
	}

	s.Permissions = kept
}

// ============================================================================
// Role Management - 角色管理
// ============================================================================

// addRoles adds a set of roles to the session with automatic deduplication.
// addRoles 向会话中添加一组角色（自动去重）。
func (s *Session) addRoles(roles ...string) {
	if len(roles) == 0 {
		return
	}

	// 构建现有角色的集合（用于去重）
	existing := make(map[string]struct{}, len(s.Roles))
	for _, r := range s.Roles {
		existing[r] = struct{}{}
	}

	// 添加新角色（跳过已存在或空的）
	for _, r := range roles {
		if r == "" {
			continue // 跳过空角色
		}
		if _, exists := existing[r]; !exists {
			existing[r] = struct{}{}
			s.Roles = append(s.Roles, r)
		}
	}
}

// removeRoles removes a set of roles from the session.
// removeRoles 从会话中移除一组角色。
func (s *Session) removeRoles(roles ...string) {
	if len(roles) == 0 || len(s.Roles) == 0 {
		return
	}

	// 构建要删除的角色集合（去重 + 忽略空）
	toRemove := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		if r != "" {
			toRemove[r] = struct{}{}
		}
	}

	// 过滤保留不在 toRemove 中的角色
	var kept []string
	for _, r := range s.Roles {
		if _, shouldRemove := toRemove[r]; !shouldRemove {
			kept = append(kept, r)
		}
	}

	s.Roles = kept
}
