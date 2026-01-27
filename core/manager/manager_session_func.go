// @Author daixk 2026/1/22 17:32:00
package manager

// removeTerminalByDevice 根据设备类型删除 TerminalInfos 中的所有匹配项 保持剩余项的原始顺序 并返回被删除的 TerminalInfo 列表
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

// getLatestTerminalByDevice 获取指定 device 下最新的 TerminalInfo（按列表顺序，尾部为最新）
func (s *Session) getLatestTerminalByDevice(device string) (TerminalInfo, bool) {
	for i := len(s.TerminalInfos) - 1; i >= 0; i-- {
		if s.TerminalInfos[i].Device == device {
			return s.TerminalInfos[i], true
		}
	}
	return TerminalInfo{}, false
}

// removeOldestTerminal 移除最老终端（可选按 device 过滤），保持顺序，返回被移除项及是否成功。
func (s *Session) removeOldestTerminal(device ...string) (TerminalInfo, bool) {
	if len(s.TerminalInfos) == 0 {
		return TerminalInfo{}, false
	}

	if len(device) == 0 {
		// 无设备过滤：移除第一个（最老）
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

// removeTerminalByToken 根据 token 值删除 TerminalInfos 中匹配的项（最多一个），保持顺序，返回被删除项及是否成功。
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

// removeTerminalByDeviceAndDeviceId 移除所有精确匹配 device 和 deviceId（支持空字符串）的 TerminalInfo，保持顺序并返回被移除项
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

// getTerminalsByDeviceAndDeviceId 返回所有精确匹配 device 和 deviceId（支持空字符串）的 TerminalInfo。
func (s *Session) getTerminalsByDeviceAndDeviceId(device, deviceId string) []TerminalInfo {
	var matched []TerminalInfo
	for _, ti := range s.TerminalInfos {
		if ti.Device == device && ti.DeviceId == deviceId {
			matched = append(matched, ti)
		}
	}
	return matched
}
