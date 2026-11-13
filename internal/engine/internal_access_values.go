package engine

// addPermissions adds permissions with dedupe and returns the added values. addPermissions 向会话添加权限并自动去重，同时返回实际新增值。
func sessionAddPermissions(s *Session, permissions ...string) []string {
	previousCount := len(s.Permissions)
	s.Permissions = addUniqueStrings(s.Permissions, permissions...)
	return s.Permissions[previousCount:]
}

// removePermissions removes permissions and returns the removed values. removePermissions 从会话移除指定权限，同时返回实际移除值。
func sessionRemovePermissions(s *Session, permissions ...string) []string {
	removed := existingAccessValues(s.Permissions, permissions)
	s.Permissions = removeStrings(s.Permissions, removed...)
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
	removed := existingAccessValues(s.Roles, roles)
	s.Roles = removeStrings(s.Roles, removed...)
	return removed
}

// existingAccessValues returns requested values that exist, preserving request order. existingAccessValues 返回已存在的请求值并保留请求顺序。
func existingAccessValues(items, values []string) []string {
	if len(items) == 0 || len(values) == 0 {
		return nil
	}

	existing := make(map[string]struct{}, len(items))
	for _, item := range items {
		existing[item] = struct{}{}
	}

	matched := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := existing[value]; ok {
			matched = append(matched, value)
			delete(existing, value)
		}
	}
	return matched
}
