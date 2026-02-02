// @Author daixk 2025/12/28
package gin

import (
	"context"

	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/dtoken"
	"github.com/gin-gonic/gin"
)

// Annotation annotation structure
// 注解结构体
type Annotation struct {
	AuthType        string    // Optional: specify auth type 可选：指定认证类型
	CheckLogin      bool      // Check login 检查登录
	CheckRole       []string  // Check roles 检查角色
	CheckPermission []string  // Check permissions 检查权限
	CheckDisable    bool      // Check disable status 检查封禁状态
	Ignore          bool      // Ignore authentication 忽略认证
	LogicType       LogicType // OR or AND logic (default: OR) OR 或 AND 逻辑（默认: OR）
}

// GetHandler gets handler with annotations
// 获取带注解的处理器
func GetHandler(ctx context.Context, handler gin.HandlerFunc, failFunc func(c *gin.Context, err error), annotations ...*Annotation) gin.HandlerFunc {
	return func(c *gin.Context) {
		if len(annotations) > 0 && annotations[0].Ignore {
			if handler != nil {
				handler(c)
			} else {
				c.Next()
			}
			return
		}

		ann := &Annotation{}
		if len(annotations) > 0 {
			ann = annotations[0]
		}

		needAuth := ann.CheckLogin || ann.CheckDisable || len(ann.CheckPermission) > 0 || len(ann.CheckRole) > 0
		if !needAuth {
			if handler != nil {
				handler(c)
			} else {
				c.Next()
			}
			return
		}

		mgr, err := dtoken.GetManager(ann.AuthType)
		if err != nil {
			if failFunc != nil {
				failFunc(c, err)
			} else {
				writeErrorResponse(c, err)
			}
			c.Abort()
			return
		}

		dCtx := getDContext(c, mgr)
		token := dCtx.GetTokenValue()

		// Check if user is logged in
		// 检查用户是否已登录
		if !dtoken.IsLogin(ctx, token) {
			if failFunc != nil {
				failFunc(c, derror.ErrNotLogin)
			} else {
				writeErrorResponse(c, derror.ErrNotLogin)
			}
			c.Abort()
			return
		}

		var loginID string
		if ann.CheckDisable || len(ann.CheckPermission) > 0 || len(ann.CheckRole) > 0 {
			loginID, err = mgr.GetLoginID(ctx, token)
			if err != nil {
				writeErrorResponse(c, err)
				c.Abort()
				return
			}
		}

		if ann.CheckDisable {
			if mgr.IsDisable(ctx, loginID) {
				if failFunc != nil {
					failFunc(c, derror.ErrAccountDisabled)
				} else {
					writeErrorResponse(c, derror.ErrAccountDisabled)
				}
				c.Abort()
				return
			}
		}

		if len(ann.CheckPermission) > 0 {
			var ok bool
			if ann.LogicType == LogicAnd {
				ok = mgr.HasPermissionsAnd(ctx, loginID, ann.CheckPermission)
			} else {
				ok = mgr.HasPermissionsOr(ctx, loginID, ann.CheckPermission)
			}
			if !ok {
				if failFunc != nil {
					failFunc(c, derror.ErrPermissionDenied)
				} else {
					writeErrorResponse(c, derror.ErrPermissionDenied)
				}
				c.Abort()
				return
			}
		}

		if len(ann.CheckRole) > 0 {
			var ok bool
			if ann.LogicType == LogicAnd {
				ok = mgr.HasRolesAnd(ctx, loginID, ann.CheckRole)
			} else {
				ok = mgr.HasRolesOr(ctx, loginID, ann.CheckRole)
			}
			if !ok {
				if failFunc != nil {
					failFunc(c, derror.ErrRoleDenied)
				} else {
					writeErrorResponse(c, derror.ErrRoleDenied)
				}
				c.Abort()
				return
			}
		}

		if handler != nil {
			handler(c)
		} else {
			c.Next()
		}
	}
}

// CheckLoginMiddleware decorator for login checking
// 检查登录装饰器
func CheckLoginMiddleware(
	ctx context.Context,
	handler gin.HandlerFunc,
	failFunc func(c *gin.Context, err error),
	authType ...string,
) gin.HandlerFunc {
	ann := &Annotation{CheckLogin: true}
	if len(authType) > 0 {
		ann.AuthType = authType[0]
	}
	return GetHandler(ctx, handler, failFunc, ann)
}

// CheckRoleMiddleware decorator for role checking
// 检查角色装饰器
func CheckRoleMiddleware(
	ctx context.Context,
	roles []string,
	handler gin.HandlerFunc,
	failFunc func(c *gin.Context, err error),
	authType ...string,
) gin.HandlerFunc {
	ann := &Annotation{CheckRole: roles}
	if len(authType) > 0 {
		ann.AuthType = authType[0]
	}
	return GetHandler(ctx, handler, failFunc, ann)
}

// CheckPermissionMiddleware decorator for permission checking
// 检查权限装饰器
func CheckPermissionMiddleware(
	ctx context.Context,
	perms []string,
	handler gin.HandlerFunc,
	failFunc func(c *gin.Context, err error),
	authType ...string,
) gin.HandlerFunc {
	ann := &Annotation{CheckPermission: perms}
	if len(authType) > 0 {
		ann.AuthType = authType[0]
	}
	return GetHandler(ctx, handler, failFunc, ann)
}

// CheckDisableMiddleware decorator for checking if account is disabled
// 检查是否被封禁装饰器
func CheckDisableMiddleware(
	ctx context.Context,
	handler gin.HandlerFunc,
	failFunc func(c *gin.Context, err error),
	authType ...string,
) gin.HandlerFunc {
	ann := &Annotation{CheckDisable: true}
	if len(authType) > 0 {
		ann.AuthType = authType[0]
	}
	return GetHandler(ctx, handler, failFunc, ann)
}

// IgnoreMiddleware decorator to ignore authentication
// 忽略认证装饰器
func IgnoreMiddleware(
	ctx context.Context,
	handler gin.HandlerFunc,
	failFunc func(c *gin.Context, err error),
) gin.HandlerFunc {
	ann := &Annotation{Ignore: true}
	return GetHandler(ctx, handler, failFunc, ann)
}

// ============ Combined Middleware ============
// ============ 组合中间件 ============

// CheckLoginAndRoleMiddleware checks login and role
// 检查登录和角色
func CheckLoginAndRoleMiddleware(
	ctx context.Context,
	roles []string,
	handler gin.HandlerFunc,
	failFunc func(c *gin.Context, err error),
	authType ...string,
) gin.HandlerFunc {
	ann := &Annotation{CheckLogin: true, CheckRole: roles}
	if len(authType) > 0 {
		ann.AuthType = authType[0]
	}
	return GetHandler(ctx, handler, failFunc, ann)
}

// CheckLoginAndPermissionMiddleware checks login and permission
// 检查登录和权限
func CheckLoginAndPermissionMiddleware(
	ctx context.Context,
	perms []string,
	handler gin.HandlerFunc,
	failFunc func(c *gin.Context, err error),
	authType ...string,
) gin.HandlerFunc {
	ann := &Annotation{CheckLogin: true, CheckPermission: perms}
	if len(authType) > 0 {
		ann.AuthType = authType[0]
	}
	return GetHandler(ctx, handler, failFunc, ann)
}

// CheckAllMiddleware checks login, role, permission and disable status
// 全面检查
func CheckAllMiddleware(
	ctx context.Context,
	roles []string,
	perms []string,
	handler gin.HandlerFunc,
	failFunc func(c *gin.Context, err error),
	authType ...string,
) gin.HandlerFunc {
	ann := &Annotation{CheckLogin: true, CheckRole: roles, CheckPermission: perms}
	if len(authType) > 0 {
		ann.AuthType = authType[0]
	}
	return GetHandler(ctx, handler, failFunc, ann)
}

// ============ Route Group Helper ============
// ============ 路由组辅助函数 ============

// AuthGroup creates a route group with authentication
// 创建带认证的路由组
func AuthGroup(
	ctx context.Context,
	group *gin.RouterGroup,
	handler gin.HandlerFunc,
	failFunc func(c *gin.Context, err error),
	authType ...string,
) *gin.RouterGroup {
	group.Use(CheckLoginMiddleware(ctx, handler, failFunc, authType...))
	return group
}

// RoleGroup creates a route group with role checking
// 创建带角色检查的路由组
func RoleGroup(
	ctx context.Context,
	group *gin.RouterGroup,
	roles []string,
	handler gin.HandlerFunc,
	failFunc func(c *gin.Context, err error),
	authType ...string,
) *gin.RouterGroup {
	group.Use(CheckLoginAndRoleMiddleware(ctx, roles, handler, failFunc, authType...))
	return group
}

// PermissionGroup creates a route group with permission checking
// 创建带权限检查的路由组
func PermissionGroup(
	ctx context.Context,
	group *gin.RouterGroup,
	perms []string,
	handler gin.HandlerFunc,
	failFunc func(c *gin.Context, err error),
	authType ...string,
) *gin.RouterGroup {
	group.Use(CheckLoginAndPermissionMiddleware(ctx, perms, handler, failFunc, authType...))
	return group
}

// RoleAndPermissionGroup creates a route group with role and permission checking
// 创建带角色和权限检查的路由组
func RoleAndPermissionGroup(
	ctx context.Context,
	group *gin.RouterGroup,
	roles []string,
	perms []string,
	handler gin.HandlerFunc,
	failFunc func(c *gin.Context, err error),
	authType ...string,
) *gin.RouterGroup {
	group.Use(CheckAllMiddleware(ctx, roles, perms, handler, failFunc, authType...))
	return group
}
