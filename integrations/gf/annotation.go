// @Author daixk 2025/12/28 1:27:00
package gf

import (
	"context"

	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/dtoken"
	"github.com/gogf/gf/v2/net/ghttp"
)

// Annotation represents the annotation structure for authentication and authorization.
// Annotation 表示用于认证和授权的注解结构体。
type Annotation struct {
	// AuthType specifies the authentication type (optional).
	// AuthType 指定认证类型（可选）。
	AuthType string `json:"authType"`
	// CheckLogin indicates whether to check login status.
	// CheckLogin 表示是否检查登录状态。
	CheckLogin bool `json:"checkLogin"`
	// CheckRole specifies roles to check.
	// CheckRole 指定要检查的角色。
	CheckRole []string `json:"checkRole"`
	// CheckPermission specifies permissions to check.
	// CheckPermission 指定要检查的权限。
	CheckPermission []string `json:"checkPermission"`
	// CheckDisable indicates whether to check account disable status.
	// CheckDisable 表示是否检查账号封禁状态。
	CheckDisable bool `json:"checkDisable"`
	// Ignore indicates whether to ignore authentication.
	// Ignore 表示是否忽略认证。
	Ignore bool `json:"ignore"`
	// LogicType specifies the logic type (OR or AND, default: OR).
	// LogicType 指定逻辑类型（OR 或 AND，默认：OR）。
	LogicType LogicType `json:"logicType"`
}

// GetHandler gets handler with annotations.
// GetHandler 获取带注解的处理器。
func GetHandler(ctx context.Context, handler ghttp.HandlerFunc, failFunc func(r *ghttp.Request, err error), annotations ...*Annotation) ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		// Ignore authentication and pass through directly
		// 忽略认证并直接放行
		if len(annotations) > 0 && annotations[0].Ignore {
			if handler != nil {
				handler(r)
			} else {
				r.Middleware.Next()
			}
			return
		}

		// Check if any authentication is needed
		// 检查是否需要任何认证
		ann := &Annotation{}
		if len(annotations) > 0 {
			ann = annotations[0]
		}

		// No authentication required, pass through
		// 无需任何认证，直接放行
		needAuth := ann.CheckLogin || ann.CheckDisable || len(ann.CheckPermission) > 0 || len(ann.CheckRole) > 0
		if !needAuth {
			if handler != nil {
				handler(r)
			} else {
				r.Middleware.Next()
			}
			return
		}

		// Get Manager instance
		// 获取 Manager 实例
		mgr, err := dtoken.GetManager(ann.AuthType)
		if err != nil {
			if failFunc != nil {
				failFunc(r, err)
			} else {
				writeErrorResponse(r, err)
			}
			return
		}

		// Get DTokenContext (reuse cached context)
		// 获取 DTokenContext（复用缓存上下文）
		dCtx := getDContext(r, mgr)
		token := dCtx.GetTokenValue()

		// Check if user is logged in
		// 检查用户是否已登录
		if !dtoken.IsLogin(ctx, token) {
			if failFunc != nil {
				failFunc(r, derror.ErrNotLogin)
			} else {
				writeErrorResponse(r, derror.ErrNotLogin)
			}
			return
		}

		// Get loginID for further checks
		// 获取 loginID 用于后续检查
		var loginID string
		if ann.CheckDisable || len(ann.CheckPermission) > 0 || len(ann.CheckRole) > 0 {
			loginID, err = mgr.GetLoginID(ctx, token)
			if err != nil {
				if failFunc != nil {
					failFunc(r, err)
				} else {
					writeErrorResponse(r, err)
				}
				return
			}
		}

		// Check if account is disabled
		// 检查账号是否被封禁
		if ann.CheckDisable {
			if dtoken.IsDisable(ctx, loginID) {
				if failFunc != nil {
					failFunc(r, derror.ErrAccountDisabled)
				} else {
					writeErrorResponse(r, derror.ErrAccountDisabled)
				}
				return
			}
		}

		// Check permissions
		// 检查权限
		if len(ann.CheckPermission) > 0 {
			var ok bool
			if ann.LogicType == LogicAnd {
				ok = dtoken.HasPermissionsAnd(ctx, loginID, ann.CheckPermission)
			} else {
				ok = dtoken.HasPermissionsOr(ctx, loginID, ann.CheckPermission)
			}
			if !ok {
				if failFunc != nil {
					failFunc(r, derror.ErrPermissionDenied)
				} else {
					writeErrorResponse(r, derror.ErrPermissionDenied)
				}
				return
			}
		}

		// Check roles
		// 检查角色
		if len(ann.CheckRole) > 0 {
			var ok bool
			if ann.LogicType == LogicAnd {
				ok = dtoken.HasRolesAnd(ctx, loginID, ann.CheckRole)
			} else {
				ok = dtoken.HasRolesOr(ctx, loginID, ann.CheckRole)
			}
			if !ok {
				if failFunc != nil {
					failFunc(r, derror.ErrRoleDenied)
				} else {
					writeErrorResponse(r, derror.ErrRoleDenied)
				}
				return
			}
		}

		// All checks passed, execute original handler
		// 所有检查通过，执行原处理器
		if handler != nil {
			handler(r)
		} else {
			r.Middleware.Next()
		}
	}
}

// CheckLoginMiddleware provides a decorator for login checking.
// CheckLoginMiddleware 提供检查登录的装饰器。
func CheckLoginMiddleware(
	ctx context.Context,
	handler ghttp.HandlerFunc,
	failFunc func(r *ghttp.Request, err error),
	authType ...string,
) ghttp.HandlerFunc {
	ann := &Annotation{CheckLogin: true}
	if len(authType) > 0 {
		ann.AuthType = authType[0]
	}
	return GetHandler(ctx, handler, failFunc, ann)
}

// CheckRoleMiddleware provides a decorator for role checking.
// CheckRoleMiddleware 提供检查角色的装饰器。
func CheckRoleMiddleware(
	ctx context.Context,
	roles []string,
	handler ghttp.HandlerFunc,
	failFunc func(r *ghttp.Request, err error),
	authType ...string,
) ghttp.HandlerFunc {
	ann := &Annotation{CheckRole: roles}
	if len(authType) > 0 {
		ann.AuthType = authType[0]
	}
	return GetHandler(ctx, handler, failFunc, ann)
}

// CheckPermissionMiddleware provides a decorator for permission checking.
// CheckPermissionMiddleware 提供检查权限的装饰器。
func CheckPermissionMiddleware(
	ctx context.Context,
	perms []string,
	handler ghttp.HandlerFunc,
	failFunc func(r *ghttp.Request, err error),
	authType ...string,
) ghttp.HandlerFunc {
	ann := &Annotation{CheckPermission: perms}
	if len(authType) > 0 {
		ann.AuthType = authType[0]
	}
	return GetHandler(ctx, handler, failFunc, ann)
}

// CheckDisableMiddleware provides a decorator for checking if account is disabled.
// CheckDisableMiddleware 提供检查账号是否被封禁的装饰器。
func CheckDisableMiddleware(
	ctx context.Context,
	handler ghttp.HandlerFunc,
	failFunc func(r *ghttp.Request, err error),
	authType ...string,
) ghttp.HandlerFunc {
	ann := &Annotation{CheckDisable: true}
	if len(authType) > 0 {
		ann.AuthType = authType[0]
	}
	return GetHandler(ctx, handler, failFunc, ann)
}

// IgnoreMiddleware provides a decorator to ignore authentication.
// IgnoreMiddleware 提供忽略认证的装饰器。
func IgnoreMiddleware(
	ctx context.Context,
	handler ghttp.HandlerFunc,
	failFunc func(r *ghttp.Request, err error),
) ghttp.HandlerFunc {
	ann := &Annotation{Ignore: true}
	return GetHandler(ctx, handler, failFunc, ann)
}

// ============================================================================
// Combined Middleware - 组合中间件
// ============================================================================

// CheckLoginAndRoleMiddleware checks both login and role.
// CheckLoginAndRoleMiddleware 检查登录和角色。
func CheckLoginAndRoleMiddleware(
	ctx context.Context,
	roles []string,
	handler ghttp.HandlerFunc,
	failFunc func(r *ghttp.Request, err error),
	authType ...string,
) ghttp.HandlerFunc {
	ann := &Annotation{CheckLogin: true, CheckRole: roles}
	if len(authType) > 0 {
		ann.AuthType = authType[0]
	}
	return GetHandler(ctx, handler, failFunc, ann)
}

// CheckLoginAndPermissionMiddleware checks both login and permission.
// CheckLoginAndPermissionMiddleware 检查登录和权限。
func CheckLoginAndPermissionMiddleware(
	ctx context.Context,
	perms []string,
	handler ghttp.HandlerFunc,
	failFunc func(r *ghttp.Request, err error),
	authType ...string,
) ghttp.HandlerFunc {
	ann := &Annotation{CheckLogin: true, CheckPermission: perms}
	if len(authType) > 0 {
		ann.AuthType = authType[0]
	}
	return GetHandler(ctx, handler, failFunc, ann)
}

// CheckAllMiddleware checks login, role, permission and disable status.
// CheckAllMiddleware 全面检查登录、角色、权限和封禁状态。
func CheckAllMiddleware(
	ctx context.Context,
	roles []string,
	perms []string,
	handler ghttp.HandlerFunc,
	failFunc func(r *ghttp.Request, err error),
	authType ...string,
) ghttp.HandlerFunc {
	ann := &Annotation{CheckLogin: true, CheckRole: roles, CheckPermission: perms}
	if len(authType) > 0 {
		ann.AuthType = authType[0]
	}
	return GetHandler(ctx, handler, failFunc, ann)
}

// ============================================================================
// Route Group Helper - 路由组辅助函数
// ============================================================================

// AuthGroup creates a route group with authentication.
// AuthGroup 创建带认证的路由组。
func AuthGroup(
	ctx context.Context,
	group *ghttp.RouterGroup,
	handler ghttp.HandlerFunc,
	failFunc func(r *ghttp.Request, err error),
	authType ...string,
) *ghttp.RouterGroup {
	group.Middleware(CheckLoginMiddleware(ctx, handler, failFunc, authType...))
	return group
}

// RoleGroup creates a route group with role checking.
// RoleGroup 创建带角色检查的路由组。
func RoleGroup(
	ctx context.Context,
	group *ghttp.RouterGroup,
	roles []string,
	handler ghttp.HandlerFunc,
	failFunc func(r *ghttp.Request, err error),
	authType ...string,
) *ghttp.RouterGroup {
	group.Middleware(CheckLoginAndRoleMiddleware(ctx, roles, handler, failFunc, authType...))
	return group
}

// PermissionGroup creates a route group with permission checking.
// PermissionGroup 创建带权限检查的路由组。
func PermissionGroup(
	ctx context.Context,
	group *ghttp.RouterGroup,
	perms []string,
	handler ghttp.HandlerFunc,
	failFunc func(r *ghttp.Request, err error),
	authType ...string,
) *ghttp.RouterGroup {
	group.Middleware(CheckLoginAndPermissionMiddleware(ctx, perms, handler, failFunc, authType...))
	return group
}

// RoleAndPermissionGroup creates a route group with role and permission checking.
// RoleAndPermissionGroup 创建带角色和权限检查的路由组。
func RoleAndPermissionGroup(
	ctx context.Context,
	group *ghttp.RouterGroup,
	roles []string,
	perms []string,
	handler ghttp.HandlerFunc,
	failFunc func(r *ghttp.Request, err error),
	authType ...string,
) *ghttp.RouterGroup {
	group.Middleware(CheckAllMiddleware(ctx, roles, perms, handler, failFunc, authType...))
	return group
}
