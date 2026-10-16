// @Author daixk 2025/12/22 15:56:00
package authcheck

import (
	"context"
	"errors"
	"reflect"

	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/dtoken"
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

// GetManager resolves manager by auth type GetManager 根据 authType 获取对应的 Manager。
func GetManager(authType string) (*manager.Manager, error) {
	return dtoken.GetManager(authType)
}

// ResolveManager prefers an explicitly injected manager and falls back to the global registry. ResolveManager 优先使用显式注入的 Manager，否则回退到全局注册表。
func ResolveManager(explicit *manager.Manager, authType string) (*manager.Manager, error) {
	if explicit != nil {
		if explicit.IsClosed() {
			return nil, derror.ErrManagerNotFound
		}
		return explicit, nil
	}
	return GetManager(authType)
}

// ResolveManagerFromContext prefers a request-scoped manager when auth type is implicit. ResolveManagerFromContext 在未指定认证类型时优先使用请求级 Manager。
func ResolveManagerFromContext(authType string, cached any) (*manager.Manager, error) {
	if authType == "" {
		if source, ok := cached.(interface{ GetManager() *manager.Manager }); ok {
			// A missing framework context can reach this interface as a typed nil. 框架中缺失的上下文可能以带类型的空值传入接口。
			value := reflect.ValueOf(source)
			switch value.Kind() {
			case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
				if value.IsNil() {
					return GetManager(authType)
				}
			}

			// An existing scope without a manager is invalid, not a global fallback. 已有请求作用域缺少 Manager 时应拒绝，避免切换到全局认证空间。
			mgr := source.GetManager()
			if mgr == nil {
				return nil, derror.ErrManagerNotFound
			}
			return ResolveManager(mgr, authType)
		}
	}
	return GetManager(authType)
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

	// Preserve failure causes instead of collapsing storage faults and restrictions into login failures. 保留失败原因，避免把存储故障和访问限制压成登录失效。
	if req.CheckLogin {
		if err := mgr.CheckLogin(ctx, req.TokenValue); err != nil {
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
	}

	// Resolve loginID only once because disable/annotation checks share it loginID 只解析一次，供封禁和注解类权限校验复用。
	ensureLoginID := func() (string, error) {
		if result.LoginID != "" {
			return result.LoginID, nil
		}

		loginID, err := mgr.GetLoginID(ctx, req.TokenValue)
		if err != nil {
			return "", err
		}
		result.LoginID = loginID
		return loginID, nil
	}

	if req.CheckDisable {
		loginID, err := ensureLoginID()
		if err != nil {
			return nil, err
		}

		// Preserve storage errors when checking disable state. 校验封禁状态时保留存储错误。
		if err := mgr.CheckDisable(ctx, loginID); err != nil {
			return nil, err
		}
	}

	if len(req.Permissions) > 0 {
		ok, err := checkPermissions(ctx, mgr, req, ensureLoginID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, derror.ErrPermissionDenied
		}
	}

	if len(req.Roles) > 0 {
		ok, err := checkRoles(ctx, mgr, req, ensureLoginID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, derror.ErrRoleDenied
		}
	}

	return result, nil
}

// checkPermissions checks permissions by loginID or token checkPermissions 按 loginID 或 Token 校验权限。
func checkPermissions(
	ctx context.Context,
	mgr *manager.Manager,
	req Request,
	ensureLoginID func() (string, error),
) (bool, error) {
	// Annotation checks already require loginID; simple middleware keeps old ByToken behavior 注解场景已需要 loginID；普通中间件保留原 ByToken 行为。
	if req.CheckLogin || req.CheckDisable {
		loginID, err := ensureLoginID()
		if err != nil {
			return false, err
		}
		if req.LogicType == LogicAnd {
			return mgr.HasPermissionsAnd(ctx, loginID, req.Permissions), nil
		}
		return mgr.HasPermissionsOr(ctx, loginID, req.Permissions), nil
	}

	if req.LogicType == LogicAnd {
		return mgr.HasPermissionsAndByToken(ctx, req.TokenValue, req.Permissions), nil
	}
	return mgr.HasPermissionsOrByToken(ctx, req.TokenValue, req.Permissions), nil
}

// checkRoles checks roles by loginID or token checkRoles 按 loginID 或 Token 校验角色。
func checkRoles(
	ctx context.Context,
	mgr *manager.Manager,
	req Request,
	ensureLoginID func() (string, error),
) (bool, error) {
	// Annotation checks already require loginID; simple middleware keeps old ByToken behavior 注解场景已需要 loginID；普通中间件保留原 ByToken 行为。
	if req.CheckLogin || req.CheckDisable {
		loginID, err := ensureLoginID()
		if err != nil {
			return false, err
		}
		if req.LogicType == LogicAnd {
			return mgr.HasRolesAnd(ctx, loginID, req.Roles), nil
		}
		return mgr.HasRolesOr(ctx, loginID, req.Roles), nil
	}

	if req.LogicType == LogicAnd {
		return mgr.HasRolesAndByToken(ctx, req.TokenValue, req.Roles), nil
	}
	return mgr.HasRolesOrByToken(ctx, req.TokenValue, req.Roles), nil
}
