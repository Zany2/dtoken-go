// @Author daixk 2025/12/22 15:56:00
package manager

// removeTerminalByToken removes terminal by token removeTerminalByToken 根据 token 值移除终端信息
func (s *Session) removeTerminalByToken(tokenValue string) (TerminalInfo, bool) {
	// Validate token value 校验 Token 值。
	if tokenValue == "" {
		return TerminalInfo{}, false
	}

	// Search matched terminal 查找匹配终端。
	for i, ti := range s.TerminalInfos {
		if ti.Token == tokenValue {
			// Record removed terminal 记录被移除终端。
			removed := ti
			// Remove matched terminal 保持顺序移除匹配终端
			s.TerminalInfos = append(s.TerminalInfos[:i], s.TerminalInfos[i+1:]...)
			return removed, true
		}
	}

	// Return not found 返回未找到。
	return TerminalInfo{}, false
}

// removeTerminalByDevice removes terminals by device removeTerminalByDevice 根据设备类型移除全部匹配终端
func (s *Session) removeTerminalByDevice(device string) []TerminalInfo {
	var kept []TerminalInfo    // kept stores remaining terminals kept 存储保留终端
	var removed []TerminalInfo // removed stores removed terminals removed 存储被删除终端

	// Split terminals by device 按设备拆分终端。
	for _, ti := range s.TerminalInfos {
		if ti.Device == device {
			removed = append(removed, ti)
		} else {
			kept = append(kept, ti)
		}
	}

	// Replace terminal list 替换终端列表。
	s.TerminalInfos = kept
	return removed
}

// removeTerminalByDeviceAndDeviceId removes terminals by device and id removeTerminalByDeviceAndDeviceId 根据设备类型和设备 ID 移除终端
func (s *Session) removeTerminalByDeviceAndDeviceId(device, deviceId string) []TerminalInfo {
	// Prepare kept and removed lists 准备保留和移除列表。
	var kept []TerminalInfo
	var removed []TerminalInfo

	// Split terminals by concrete device 按具体设备拆分终端。
	for _, ti := range s.TerminalInfos {
		if ti.Device == device && ti.DeviceId == deviceId {
			removed = append(removed, ti)
		} else {
			kept = append(kept, ti)
		}
	}

	// Replace terminal list 替换终端列表。
	s.TerminalInfos = kept
	return removed
}

// removeOldestTerminal removes oldest terminal removeOldestTerminal 移除最老终端并可按设备过滤
func (s *Session) removeOldestTerminal(device ...string) (TerminalInfo, bool) {
	// Return when session has no terminals 无终端时直接返回。
	if len(s.TerminalInfos) == 0 {
		return TerminalInfo{}, false
	}

	// Remove first terminal when no device filter 无设备过滤时移除第一个终端。
	if len(device) == 0 {
		first := s.TerminalInfos[0]
		s.TerminalInfos = s.TerminalInfos[1:]
		return first, true
	}

	// Find oldest matched terminal 查找最早匹配设备的终端
	// Scan by device filter 按设备过滤扫描。
	targetDevice := device[0]
	for i, ti := range s.TerminalInfos {
		if ti.Device == targetDevice {
			removed := ti
			// Remove matched terminal 保持顺序移除匹配终端
			s.TerminalInfos = append(s.TerminalInfos[:i], s.TerminalInfos[i+1:]...)
			return removed, true
		}
	}

	// Return not found 返回未找到。
	return TerminalInfo{}, false
}

// removeAllTerminals removes all terminals removeAllTerminals 移除全部终端信息
func (s *Session) removeAllTerminals() []TerminalInfo {
	// Copy removed terminals 拷贝被移除终端。
	removed := make([]TerminalInfo, len(s.TerminalInfos))
	copy(removed, s.TerminalInfos)
	// Clear terminal list 清空终端列表。
	s.TerminalInfos = []TerminalInfo{}
	return removed
}

// getTerminalsByDevice gets terminals by device getTerminalsByDevice 返回指定设备的全部终端信息
func (s *Session) getTerminalsByDevice(device string) []TerminalInfo {
	// Collect matched terminals 收集匹配终端。
	var matched []TerminalInfo
	for _, ti := range s.TerminalInfos {
		if ti.Device == device {
			matched = append(matched, ti)
		}
	}
	return matched
}

// getTerminalsByDeviceAndDeviceId gets terminals by device and id getTerminalsByDeviceAndDeviceId 返回精确匹配设备和设备 ID 的终端信息
func (s *Session) getTerminalsByDeviceAndDeviceId(device, deviceId string) []TerminalInfo {
	// Collect matched terminals 收集匹配终端。
	var matched []TerminalInfo
	for _, ti := range s.TerminalInfos {
		if ti.Device == device && ti.DeviceId == deviceId {
			matched = append(matched, ti)
		}
	}
	return matched
}

// getLatestTerminalByDevice gets latest terminal by device getLatestTerminalByDevice 获取指定设备下最新的终端信息
func (s *Session) getLatestTerminalByDevice(device string) (TerminalInfo, bool) {
	// Scan from newest to oldest 从新到旧扫描。
	for i := len(s.TerminalInfos) - 1; i >= 0; i-- {
		if s.TerminalInfos[i].Device == device {
			return s.TerminalInfos[i], true
		}
	}
	// Return not found 返回未找到。
	return TerminalInfo{}, false
}

// hasTerminalToken checks whether token exists in session hasTerminalToken 检查会话中是否存在指定 Token
func (s *Session) hasTerminalToken(tokenValue string) bool {
	// Validate token value 校验 Token 值。
	if tokenValue == "" {
		return false
	}

	// Search token in terminals 在终端列表中查找 Token。
	for _, ti := range s.TerminalInfos {
		if ti.Token == tokenValue {
			return true
		}
	}
	return false
}

// addPermissions adds permissions with dedupe addPermissions 向会话添加权限并自动去重
func (s *Session) addPermissions(permissions ...string) {
	// Return when no permissions provided 没有权限时直接返回。
	if len(permissions) == 0 {
		return
	}

	// Build existing set 构建现有权限集合
	existing := make(map[string]struct{}, len(s.Permissions))
	for _, p := range s.Permissions {
		existing[p] = struct{}{}
	}

	// Append new permissions 追加新的权限项
	for _, p := range permissions {
		if p == "" {
			continue // Skip empty permission 跳过空权限
		}
		if _, exists := existing[p]; !exists {
			existing[p] = struct{}{}
			s.Permissions = append(s.Permissions, p)
		}
	}
}

// removePermissions removes permissions removePermissions 从会话移除指定权限
func (s *Session) removePermissions(permissions ...string) {
	// Return when no removal needed 无需移除时直接返回。
	if len(permissions) == 0 || len(s.Permissions) == 0 {
		return
	}

	// Build remove set 构建待删除权限集合
	toRemove := make(map[string]struct{}, len(permissions))
	for _, p := range permissions {
		if p != "" {
			toRemove[p] = struct{}{}
		}
	}

	// Keep unmatched permissions 过滤保留未删除权限
	var kept []string
	for _, p := range s.Permissions {
		if _, shouldRemove := toRemove[p]; !shouldRemove {
			kept = append(kept, p)
		}
	}

	// Replace permissions 替换权限列表。
	s.Permissions = kept
}

// addRoles adds roles with dedupe addRoles 向会话添加角色并自动去重
func (s *Session) addRoles(roles ...string) {
	// Return when no roles provided 没有角色时直接返回。
	if len(roles) == 0 {
		return
	}

	// Build existing set 构建现有角色集合
	existing := make(map[string]struct{}, len(s.Roles))
	for _, r := range s.Roles {
		existing[r] = struct{}{}
	}

	// Append new roles 追加新的角色项
	for _, r := range roles {
		if r == "" {
			continue // Skip empty role 跳过空角色
		}
		if _, exists := existing[r]; !exists {
			existing[r] = struct{}{}
			s.Roles = append(s.Roles, r)
		}
	}
}

// removeRoles removes roles removeRoles 从会话移除指定角色
func (s *Session) removeRoles(roles ...string) {
	// Return when no removal needed 无需移除时直接返回。
	if len(roles) == 0 || len(s.Roles) == 0 {
		return
	}

	// Build remove set 构建待删除角色集合
	toRemove := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		if r != "" {
			toRemove[r] = struct{}{}
		}
	}

	// Keep unmatched roles 过滤保留未删除角色
	var kept []string
	for _, r := range s.Roles {
		if _, shouldRemove := toRemove[r]; !shouldRemove {
			kept = append(kept, r)
		}
	}

	// Replace roles 替换角色列表。
	s.Roles = kept
}
