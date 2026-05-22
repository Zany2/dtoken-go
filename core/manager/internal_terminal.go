// @Author daixk 2025/12/22 15:56:00
package manager

// terminalMatchFunc matches terminal info. terminalMatchFunc 匹配终端信息。
type terminalMatchFunc func(TerminalInfo) bool

// removeTerminals removes matched terminals. removeTerminals 移除匹配的终端信息。
func (s *Session) removeTerminals(match terminalMatchFunc) []TerminalInfo {
	var kept []TerminalInfo    // kept stores remaining terminals. kept 存储保留终端。
	var removed []TerminalInfo // removed stores removed terminals. removed 存储被移除终端。

	// Split terminals by matcher. 按匹配器拆分终端。
	for _, ti := range s.TerminalInfos {
		if match != nil && match(ti) {
			removed = append(removed, ti)
			continue
		}
		kept = append(kept, ti)
	}

	// Replace terminal list. 替换终端列表。
	s.TerminalInfos = kept
	return removed
}

// filterTerminals returns matched terminals. filterTerminals 返回匹配的终端信息。
func (s *Session) filterTerminals(match terminalMatchFunc) []TerminalInfo {
	var matched []TerminalInfo
	// Collect matched terminals. 收集匹配终端。
	for _, ti := range s.TerminalInfos {
		if match != nil && match(ti) {
			matched = append(matched, ti)
		}
	}
	return matched
}

// addUniqueStrings appends non-empty strings with dedupe. addUniqueStrings 追加非空字符串并去重。
func addUniqueStrings(items []string, values ...string) []string {
	// Return original list when no values provided. 没有新值时返回原列表。
	if len(values) == 0 {
		return items
	}

	// Build existing set. 构建已有值集合。
	existing := make(map[string]struct{}, len(items))
	for _, item := range items {
		existing[item] = struct{}{}
	}

	// Append new unique values. 追加新的唯一值。
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, exists := existing[value]; !exists {
			existing[value] = struct{}{}
			items = append(items, value)
		}
	}
	return items
}

// removeStrings removes matched non-empty strings. removeStrings 移除匹配的非空字符串。
func removeStrings(items []string, values ...string) []string {
	// Return original list when no removal needed. 无需移除时返回原列表。
	if len(values) == 0 || len(items) == 0 {
		return items
	}

	// Build remove set. 构建待移除值集合。
	toRemove := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value != "" {
			toRemove[value] = struct{}{}
		}
	}

	// Keep unmatched values. 过滤保留未删除值。
	var kept []string
	for _, item := range items {
		if _, shouldRemove := toRemove[item]; !shouldRemove {
			kept = append(kept, item)
		}
	}
	return kept
}

// removeTerminalByToken removes terminal by token. removeTerminalByToken 根据 token 值移除终端信息。
func (s *Session) removeTerminalByToken(tokenValue string) (TerminalInfo, bool) {
	// Validate token value. 校验 Token 值。
	if tokenValue == "" {
		return TerminalInfo{}, false
	}

	// Search matched terminal. 查找匹配终端。
	for i, ti := range s.TerminalInfos {
		if ti.Token == tokenValue {
			removed := ti
			// Remove matched terminal while preserving order. 保持顺序移除匹配终端。
			s.TerminalInfos = append(s.TerminalInfos[:i], s.TerminalInfos[i+1:]...)
			return removed, true
		}
	}

	// Return not found. 返回未找到。
	return TerminalInfo{}, false
}

// removeTerminalByDevice removes terminals by device. removeTerminalByDevice 根据设备类型移除全部匹配终端。
func (s *Session) removeTerminalByDevice(device string) []TerminalInfo {
	return s.removeTerminals(func(ti TerminalInfo) bool {
		return ti.Device == device
	})
}

// removeTerminalByDeviceAndDeviceId removes terminals by device and id. removeTerminalByDeviceAndDeviceId 根据设备类型和设备 ID 移除终端。
func (s *Session) removeTerminalByDeviceAndDeviceId(device, deviceId string) []TerminalInfo {
	return s.removeTerminals(func(ti TerminalInfo) bool {
		return ti.Device == device && ti.DeviceId == deviceId
	})
}

// removeOldestTerminal removes oldest terminal. removeOldestTerminal 移除最老终端并可按设备过滤。
func (s *Session) removeOldestTerminal(device ...string) (TerminalInfo, bool) {
	// Return when session has no terminals. 无终端时直接返回。
	if len(s.TerminalInfos) == 0 {
		return TerminalInfo{}, false
	}

	// Remove first terminal when no device filter. 无设备过滤时移除第一个终端。
	if len(device) == 0 {
		first := s.TerminalInfos[0]
		s.TerminalInfos = s.TerminalInfos[1:]
		return first, true
	}

	// Scan by device filter. 按设备过滤扫描。
	targetDevice := device[0]
	for i, ti := range s.TerminalInfos {
		if ti.Device == targetDevice {
			removed := ti
			// Remove matched terminal while preserving order. 保持顺序移除匹配终端。
			s.TerminalInfos = append(s.TerminalInfos[:i], s.TerminalInfos[i+1:]...)
			return removed, true
		}
	}

	// Return not found. 返回未找到。
	return TerminalInfo{}, false
}

// removeAllTerminals removes all terminals. removeAllTerminals 移除全部终端信息。
func (s *Session) removeAllTerminals() []TerminalInfo {
	// Copy removed terminals. 拷贝被移除终端。
	removed := make([]TerminalInfo, len(s.TerminalInfos))
	copy(removed, s.TerminalInfos)
	// Clear terminal list. 清空终端列表。
	s.TerminalInfos = []TerminalInfo{}
	return removed
}

// getTerminalsByDevice gets terminals by device. getTerminalsByDevice 返回指定设备的全部终端信息。
func (s *Session) getTerminalsByDevice(device string) []TerminalInfo {
	return s.filterTerminals(func(ti TerminalInfo) bool {
		return ti.Device == device
	})
}

// getTerminalsByDeviceAndDeviceId gets terminals by device and id. getTerminalsByDeviceAndDeviceId 返回精确匹配设备和设备 ID 的终端信息。
func (s *Session) getTerminalsByDeviceAndDeviceId(device, deviceId string) []TerminalInfo {
	return s.filterTerminals(func(ti TerminalInfo) bool {
		return ti.Device == device && ti.DeviceId == deviceId
	})
}

// getLatestTerminalByDevice gets latest terminal by device. getLatestTerminalByDevice 获取指定设备下最新的终端信息。
func (s *Session) getLatestTerminalByDevice(device string) (TerminalInfo, bool) {
	// Scan from newest to oldest. 从新到旧扫描。
	for i := len(s.TerminalInfos) - 1; i >= 0; i-- {
		if s.TerminalInfos[i].Device == device {
			return s.TerminalInfos[i], true
		}
	}
	// Return not found. 返回未找到。
	return TerminalInfo{}, false
}

// hasTerminalToken checks whether token exists in session. hasTerminalToken 检查会话中是否存在指定 Token。
func (s *Session) hasTerminalToken(tokenValue string) bool {
	// Validate token value. 校验 Token 值。
	if tokenValue == "" {
		return false
	}

	// Search token in terminals. 在终端列表中查找 Token。
	for _, ti := range s.TerminalInfos {
		if ti.Token == tokenValue {
			return true
		}
	}
	return false
}

// addPermissions adds permissions with dedupe. addPermissions 向会话添加权限并自动去重。
func (s *Session) addPermissions(permissions ...string) {
	s.Permissions = addUniqueStrings(s.Permissions, permissions...)
}

// removePermissions removes permissions. removePermissions 从会话移除指定权限。
func (s *Session) removePermissions(permissions ...string) {
	s.Permissions = removeStrings(s.Permissions, permissions...)
}

// addRoles adds roles with dedupe. addRoles 向会话添加角色并自动去重。
func (s *Session) addRoles(roles ...string) {
	s.Roles = addUniqueStrings(s.Roles, roles...)
}

// removeRoles removes roles. removeRoles 从会话移除指定角色。
func (s *Session) removeRoles(roles ...string) {
	s.Roles = removeStrings(s.Roles, roles...)
}
