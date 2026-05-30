// @Author daixk 2025/12/22 15:56:00
package authcheck

import (
	"context"

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
	// LogicType controls AND/OR checks for permissions and roles LogicType 控制权限和角色的 AND/OR 逻辑。
	LogicType LogicType
	// LoginError is returned when IsLogin is false LoginError 是登录态校验失败时返回的错误。
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

	if req.LoginError == nil {
		req.LoginError = derror.ErrNotLogin
	}

	// Check login first when caller requires explicit login 需要显式登录时先检查登录态。
	if req.CheckLogin && !mgr.IsLogin(ctx, req.TokenValue) {
		return nil, req.LoginError
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

		// Check account disable state after loginID is resolved 获取 loginID 后校验账号封禁状态。
		if mgr.IsDisable(ctx, loginID) {
			return nil, derror.ErrAccountDisabled
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
