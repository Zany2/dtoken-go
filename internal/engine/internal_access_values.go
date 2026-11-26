package engine

// addPermissions adds permissions with dedupe and returns the added values. addPermissions 向会话添加权限并自动去重，同时返回实际新增值。
func sessionAddPermissions(s *Session, permissions ...string) []string {
	previousCount := len(s.Permissions)
	s.Permissions = addUniqueStrings(s.Permissions, permissions...)
	return s.Permissions[previousCount:]
}

// removePermissions removes permissions and returns the removed values. removePermissions 从会话移除指定权限，同时返回实际移除值。
func sessionRemovePermissions(s *Session, permissions ...string) []string {
	var removed []string
	s.Permissions, removed = removeAccessValues(s.Permissions, permissions)
	return removed
}

// addRoles adds roles with dedupe and returns the added values. addRoles 向会话添加角色并自动去重，同时返回实际新增值。
func sessionAddRoles(s *Session, roles ...string) []string {
	previousCount := len(s.Roles)
	s.Roles = addUniqueStrings(s.Roles, roles...)
	return s.Roles[previousCount:]
}

// removeRoles removes roles and returns the removed values. removeRoles 从会话移除指定角色，同时返回实际移除值。
func sessionRemoveRoles(s *Session, roles ...string) []string {
	var removed []string
	s.Roles, removed = removeAccessValues(s.Roles, roles)
	return removed
}

// removeAccessValues bounds its lookup set by the smaller input and reports changes in request order. removeAccessValues 按较小输入限制查找集合，并按请求顺序报告变更。
func removeAccessValues(items, values []string) ([]string, []string) {
	if len(items) == 0 || len(values) == 0 {
		return items, nil
	}

	var removed []string
	broad := len(values) >= len(items)
	requested := make(map[string]bool, min(len(items), len(values)))
	if broad {
		// Index stored values so a large request cannot inflate the temporary set. 索引存储值，避免大批次请求扩大临时集合。
		for _, item := range items {
			requested[item] = false
		}
		for _, value := range values {
			if _, exists := requested[value]; value != "" && exists {
				if removed == nil {
					removed = make([]string, 0, len(items))
				}
				removed = append(removed, value)
				delete(requested, value)
			}
		}
		if len(removed) == 0 {
			return items, nil
		}
	} else {
		for _, value := range values {
			if value != "" {
				requested[value] = false
			}
		}
	}
	if !broad && len(requested) == 0 {
		return items, nil
	}

	// Broad requests leave survivors in the set; narrow requests mark matched removals. 大请求在集合中留下保留项，小请求则标记命中的删除项。
	// Allocate only when retaining values after a removal, leaving the original array untouched. 删除后仍需保留值时才分配列表，保持原底层数组不变。
	kept := items
	changed := false
	matchedCount := 0
	for i, item := range items {
		matched, exists := requested[item]
		if exists != broad {
			if !broad && !matched {
				requested[item] = true
				matchedCount++
			}
			if !changed {
				kept = nil
				if i > 0 {
					kept = make([]string, 0, len(items)-1)
					kept = append(kept, items[:i]...)
				}
				changed = true
			}
		} else if changed {
			if kept == nil {
				kept = make([]string, 0, len(items)-i)
			}
			kept = append(kept, item)
		}
	}
	if !changed {
		return items, nil
	}
	if len(kept) == 0 {
		kept = nil
	}

	// Retain first-request order and report each removed value once, even for duplicate stored values. 保留请求首次出现顺序，即使存储值重复也只报告一次。
	if !broad {
		removed = make([]string, 0, matchedCount)
		for _, value := range values {
			if requested[value] {
				removed = append(removed, value)
				delete(requested, value)
			}
		}
	}
	return kept, removed
}
