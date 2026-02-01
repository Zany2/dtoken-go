// @Author daixk 2026/1/22 17:32:00
package manager

// ============================================================================
// Terminal Management - 终端管理
// ============================================================================

// ----------------------------------------------------------------------------
// Terminal Removal Methods - 终端移除方法
// ----------------------------------------------------------------------------

// removeTerminalByToken removes a terminal from TerminalInfos by token value (at most one match).
// It maintains the original order and returns the removed terminal and success status.
// removeTerminalByToken 根据 token 值删除 TerminalInfos 中匹配的项（最多一个）。
// 保持顺序，返回被删除项及是否成功。
//
// Parameters:
//   - tokenValue: The token value to match
//
// Returns:
//   - TerminalInfo: The removed terminal info
//   - bool: true if a terminal was removed, false otherwise
//
// 参数:
//   - tokenValue: 要匹配的 token 值
//
// 返回:
//   - TerminalInfo: 被删除的终端信息
//   - bool: 如果删除成功返回 true，否则返回 false
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
// It maintains the original order of remaining items and returns the list of removed terminals.
// removeTerminalByDevice 根据设备类型删除 TerminalInfos 中的所有匹配项。
// 保持剩余项的原始顺序，并返回被删除的 TerminalInfo 列表。
//
// Parameters:
//   - device: The device type to match
//
// Returns:
//   - []TerminalInfo: List of removed terminal infos
//
// 参数:
//   - device: 要匹配的设备类型
//
// 返回:
//   - []TerminalInfo: 被删除的终端信息列表
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

	s.TerminalInfos = kept
	return removed
}

// removeTerminalByDeviceAndDeviceId removes all terminals that exactly match both device and deviceId (supports empty strings).
// It maintains the original order and returns the list of removed terminals.
// removeTerminalByDeviceAndDeviceId 移除所有精确匹配 device 和 deviceId（支持空字符串）的 TerminalInfo。
// 保持顺序并返回被移除项。
//
// Parameters:
//   - device: The device type to match
//   - deviceId: The device ID to match (can be empty string)
//
// Returns:
//   - []TerminalInfo: List of removed terminal infos
//
// 参数:
//   - device: 要匹配的设备类型
//   - deviceId: 要匹配的设备 ID（可以是空字符串）
//
// 返回:
//   - []TerminalInfo: 被删除的终端信息列表
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
// It maintains the order and returns the removed terminal and success status.
// removeOldestTerminal 移除最老终端（可选按 device 过滤）。
// 保持顺序，返回被移除项及是否成功。
//
// Parameters:
//   - device: Optional device type filter (variadic parameter)
//
// Returns:
//   - TerminalInfo: The removed terminal info
//   - bool: true if a terminal was removed, false otherwise
//
// 参数:
//   - device: 可选的设备类型过滤器（可变参数）
//
// 返回:
//   - TerminalInfo: 被删除的终端信息
//   - bool: 如果删除成功返回 true，否则返回 false
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

// getTerminalsByDevice returns all terminals that match the specified device type (ignores deviceId).
// getTerminalsByDevice 返回所有匹配指定 device 的 TerminalInfo（不考虑 deviceId）。
//
// Parameters:
//   - device: The device type to match
//
// Returns:
//   - []TerminalInfo: List of matching terminal infos
//
// 参数:
//   - device: 要匹配的设备类型
//
// 返回:
//   - []TerminalInfo: 匹配的终端信息列表
func (s *Session) getTerminalsByDevice(device string) []TerminalInfo {
	var matched []TerminalInfo
	for _, ti := range s.TerminalInfos {
		if ti.Device == device {
			matched = append(matched, ti)
		}
	}
	return matched
}

// getTerminalsByDeviceAndDeviceId returns all terminals that exactly match both device and deviceId (supports empty strings).
// getTerminalsByDeviceAndDeviceId 返回所有精确匹配 device 和 deviceId（支持空字符串）的 TerminalInfo。
//
// Parameters:
//   - device: The device type to match
//   - deviceId: The device ID to match (can be empty string)
//
// Returns:
//   - []TerminalInfo: List of matching terminal infos
//
// 参数:
//   - device: 要匹配的设备类型
//   - deviceId: 要匹配的设备 ID（可以是空字符串）
//
// 返回:
//   - []TerminalInfo: 匹配的终端信息列表
func (s *Session) getTerminalsByDeviceAndDeviceId(device, deviceId string) []TerminalInfo {
	var matched []TerminalInfo
	for _, ti := range s.TerminalInfos {
		if ti.Device == device && ti.DeviceId == deviceId {
			matched = append(matched, ti)
		}
	}
	return matched
}

// getLatestTerminalByDevice retrieves the latest terminal for the specified device (by list order, tail is newest).
// getLatestTerminalByDevice 获取指定 device 下最新的 TerminalInfo（按列表顺序，尾部为最新）。
//
// Parameters:
//   - device: The device type to match
//
// Returns:
//   - TerminalInfo: The latest terminal info for the device
//   - bool: true if found, false otherwise
//
// 参数:
//   - device: 要匹配的设备类型
//
// 返回:
//   - TerminalInfo: 该设备的最新终端信息
//   - bool: 如果找到返回 true，否则返回 false
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
// Empty permissions are skipped.
// addPermissions 向会话中添加一组权限（自动去重）。
// 空权限会被跳过。
//
// Parameters:
//   - permissions: Variable number of permission strings to add
//
// 参数:
//   - permissions: 要添加的权限字符串（可变参数）
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
// Non-existent or empty permissions are automatically ignored.
// removePermissions 从会话中移除一组权限。
// 自动忽略不存在或空的权限。
//
// Parameters:
//   - permissions: Variable number of permission strings to remove
//
// 参数:
//   - permissions: 要移除的权限字符串（可变参数）
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
// Empty roles are skipped.
// addRoles 向会话中添加一组角色（自动去重）。
// 空角色会被跳过。
//
// Parameters:
//   - roles: Variable number of role strings to add
//
// 参数:
//   - roles: 要添加的角色字符串（可变参数）
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
// Non-existent or empty roles are automatically ignored.
// removeRoles 从会话中移除一组角色。
// 自动忽略不存在或空的角色。
//
// Parameters:
//   - roles: Variable number of role strings to remove
//
// 参数:
//   - roles: 要移除的角色字符串（可变参数）
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
