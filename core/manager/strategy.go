// @Author daixk 2025/12/22 15:56:00
package manager

import "strings"

// Strategy stores replaceable manager algorithms Strategy 存储可替换的管理器算法
type Strategy struct {
	// PermissionMatcher matches owned permission pattern with required permission PermissionMatcher 匹配已有权限模式和所需权限
	PermissionMatcher func(pattern, permission string) bool
	// RoleMatcher matches owned role pattern with required role RoleMatcher 匹配已有角色模式和所需角色
	RoleMatcher func(pattern, role string) bool
	// CreateSession creates a new account session CreateSession 创建新的账号会话
	CreateSession func(authType, loginID string, createTime int64) *Session
}

// DefaultStrategy returns default manager strategy DefaultStrategy 返回默认管理器策略
func DefaultStrategy() *Strategy {
	return &Strategy{
		PermissionMatcher: defaultPermissionMatcher,
		RoleMatcher:       defaultRoleMatcher,
		CreateSession:     defaultCreateSession,
	}
}

// normalize fills missing strategy hooks with default implementations normalize 使用默认实现补齐缺失的策略钩子
func (s *Strategy) normalize() *Strategy {
	if s == nil {
		return DefaultStrategy()
	}
	if s.PermissionMatcher == nil {
		s.PermissionMatcher = defaultPermissionMatcher
	}
	if s.RoleMatcher == nil {
		s.RoleMatcher = defaultRoleMatcher
	}
	if s.CreateSession == nil {
		s.CreateSession = defaultCreateSession
	}
	return s
}

// defaultCreateSession creates the built-in session model defaultCreateSession 创建内置会话模型
func defaultCreateSession(authType, loginID string, createTime int64) *Session {
	return &Session{
		AuthType:      authType,
		LoginID:       loginID,
		CreateTime:    createTime,
		TerminalInfos: make([]TerminalInfo, 0),
		Permissions:   make([]string, 0),
		Roles:         make([]string, 0),
		Data:          make(map[string]any),
	}
}

// defaultPermissionMatcher matches permissions with wildcard support defaultPermissionMatcher 使用通配符支持匹配权限
func defaultPermissionMatcher(pattern, permission string) bool {
	if pattern == PermissionWildcard || pattern == permission {
		return true
	}
	if !strings.Contains(pattern, PermissionWildcard) {
		return false
	}

	separator := PermissionSeparator
	if strings.Contains(pattern, "/") {
		separator = "/"
	}

	patternParts := strings.Split(pattern, separator)
	permParts := strings.Split(permission, separator)
	if len(patternParts) != len(permParts) {
		return false
	}

	for i := range patternParts {
		if patternParts[i] == PermissionWildcard {
			continue
		}
		if patternParts[i] != permParts[i] {
			return false
		}
	}
	return true
}

// defaultRoleMatcher matches roles by exact value defaultRoleMatcher 按精确值匹配角色
func defaultRoleMatcher(pattern, role string) bool {
	return pattern == role
}
