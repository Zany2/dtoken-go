// @Author daixk 2025/12/22 15:56:00
package httpauth

import (
	"context"
	"errors"
	"reflect"
	"strings"

	derror "github.com/Zany2/dtoken-go/errors"
	manager "github.com/Zany2/dtoken-go/internal/engine"
)

// LogicType defines auth check logic type LogicType 定义认证检查逻辑类型。
type LogicType string

const (
	// LogicOr means any permission or role is enough LogicOr 表示任一权限或角色满足即可。
	LogicOr LogicType = "OR"
	// LogicAnd means all permissions or roles are required LogicAnd 表示必须满足全部权限或角色。
	LogicAnd LogicType = "AND"
)

// Request describes one integration auth check Request 描述一次集成层认证检查。
type Request struct {
	// TokenValue is the request token TokenValue 是当前请求携带的 Token。
	TokenValue string
	// CheckLogin indicates whether login state must be checked CheckLogin 表示是否必须校验登录态。
	CheckLogin bool
	// CheckDisable indicates whether account disable state must be checked CheckDisable 表示是否校验账号封禁状态。
	CheckDisable bool
	// Permissions lists required permissions Permissions 表示本次请求需要的权限列表。
	Permissions []string
	// Roles lists required roles Roles 表示本次请求需要的角色列表。
	Roles []string
	// LogicType controls AND/OR checks; empty defaults to OR. LogicType 控制权限和角色的 AND/OR 逻辑，空值默认 OR。
	LogicType LogicType
	// LoginError replaces invalid or inactive token errors, while operational and disable errors are preserved. LoginError 替换无效或非活跃 Token 错误，保留运行故障及封禁错误。
	LoginError error
}

// Result stores useful auth check output Result 保存认证检查后可复用的结果。
type Result struct {
	// LoginID is resolved lazily only when needed LoginID 仅在需要时才解析。
	LoginID string
}

// ManagerResolver selects a manager from an explicitly configured registry; pass global.GetManager to opt into the global registry. ManagerResolver 从显式配置的注册表选择管理器，传入 global.GetManager 可显式启用全局注册表。
type ManagerResolver = func(authType ...string) (*manager.Manager, error)

// ResolveManager validates an explicitly selected manager. ResolveManager 验证显式选择的管理器。
func ResolveManager(explicit *manager.Manager, _ string) (*manager.Manager, error) {
	if explicit == nil || explicit.IsClosed() || explicit.GetConfig() == nil {
		return nil, derror.ErrManagerNotFound
	}
	return explicit, nil
}

// ResolveManagerFromContext inherits request scope or uses an explicit resolver without silently replacing missing or closed request managers. ResolveManagerFromContext 继承请求作用域或使用显式解析器，不静默替换缺失或已关闭的请求管理器。
func ResolveManagerFromContext(authType string, cached any, resolvers ...ManagerResolver) (*manager.Manager, error) {
	var resolver ManagerResolver
	if len(resolvers) > 0 {
		resolver = resolvers[0]
	}
	source, ok := cached.(interface{ GetManager() *manager.Manager })
	if ok {
		value := reflect.ValueOf(source)
		switch value.Kind() {
		case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
			if value.IsNil() {
				ok = false
			}
		}
	}
	if ok {
		if strings.TrimSpace(authType) == "" {
			return ResolveManager(source.GetManager(), authType)
		}
		if resolver == nil {
			if scoped, ok := cached.(interface{ GetManagerResolver() ManagerResolver }); ok {
				resolver = scoped.GetManagerResolver()
			}
		}
		if resolver == nil {
			mgr, err := ResolveManager(source.GetManager(), authType)
			if err != nil {
				return nil, err
			}
			if strings.TrimSuffix(strings.TrimSpace(authType), ":") == strings.TrimSuffix(mgr.GetConfig().AuthType, ":") {
				return mgr, nil
			}
		}
	}
	if resolver == nil {
		return nil, derror.ErrManagerNotFound
	}
	mgr, err := resolver(authType)
	if err != nil {
		return nil, err
	}
	return ResolveManager(mgr, authType)
}

// NeedAuth reports whether request needs auth checks NeedAuth 判断请求是否需要执行认证检查。
func NeedAuth(req Request) bool {
	return req.CheckLogin || req.CheckDisable || len(req.Permissions) > 0 || len(req.Roles) > 0
}

// Check performs common integration auth checks Check 执行集成层公共认证检查。
func Check(ctx context.Context, mgr *manager.Manager, req Request) (*Result, error) {
	result := &Result{}

	if !NeedAuth(req) {
		return result, nil
	}

	if mgr == nil || mgr.IsClosed() {
		return nil, derror.ErrManagerNotFound
	}

	// Reject misspelled access logic instead of silently weakening it to OR. 拒绝错误的权限逻辑配置，避免静默放宽为 OR。
	if len(req.Permissions) > 0 || len(req.Roles) > 0 {
		switch req.LogicType {
		case "", LogicOr, LogicAnd:
		default:
			return nil, derror.ErrInvalidParam
		}
	}

	if req.LoginError == nil {
		req.LoginError = derror.ErrNotLogin
	}

	// Reuse the fully validated identity in combined checks. 组合检查复用完整登录校验取得的身份。
	if req.CheckLogin {
		loginID, err := mgr.GetLoginID(ctx, req.TokenValue)
		if err != nil {
			switch {
			case errors.Is(err, derror.ErrNotLogin), errors.Is(err, derror.ErrInvalidToken),
				errors.Is(err, derror.ErrTokenExpired), errors.Is(err, derror.ErrActiveTimeout),
				errors.Is(err, derror.ErrTokenKickout), errors.Is(err, derror.ErrTokenReplaced),
				errors.Is(err, derror.ErrSessionNotFound):
				return nil, req.LoginError
			default:
				return nil, err
			}
		}

		// Keep identity output lazy for login-only requests. 仅检查登录时保持延迟身份输出的约定。
		if req.CheckDisable || len(req.Permissions) > 0 || len(req.Roles) > 0 {
			result.LoginID = loginID
		}
	}

	// Resolve identity for standalone disable checks. 独立封禁检查按需解析身份。
	if req.CheckDisable {
		if result.LoginID == "" {
			loginID, err := mgr.GetLoginID(ctx, req.TokenValue)
			if err != nil {
				return nil, err
			}
			result.LoginID = loginID
		}

		// Preserve storage errors when checking disable state. 校验封禁状态时保留存储错误。
		if err := mgr.CheckDisable(ctx, result.LoginID); err != nil {
			return nil, err
		}
	}

	// Token checks retain device scope and operational errors. Token 校验保留设备维度和运行故障。
	if len(req.Permissions) > 0 {
		if err := checkPermissions(ctx, mgr, req); err != nil {
			return nil, err
		}
	}
	if len(req.Roles) > 0 {
		if err := checkRoles(ctx, mgr, req); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// checkPermissions checks the complete token subject and propagates failures. checkPermissions 校验完整 Token 主体并传递故障。
func checkPermissions(
	ctx context.Context,
	mgr *manager.Manager,
	req Request,
) error {
	if req.LogicType == LogicAnd {
		return mgr.CheckPermissionAndByToken(ctx, req.TokenValue, req.Permissions)
	}
	return mgr.CheckPermissionOrByToken(ctx, req.TokenValue, req.Permissions)
}

// checkRoles checks the complete token subject and propagates failures. checkRoles 校验完整 Token 主体并传递故障。
func checkRoles(
	ctx context.Context,
	mgr *manager.Manager,
	req Request,
) error {
	if req.LogicType == LogicAnd {
		return mgr.CheckRoleAndByToken(ctx, req.TokenValue, req.Roles)
	}
	return mgr.CheckRoleOrByToken(ctx, req.TokenValue, req.Roles)
}
