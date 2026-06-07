// @Author daixk 2026/06/06
package beego

import (
	"context"

	web "github.com/beego/beego/v2/server/web"
	beegocontext "github.com/beego/beego/v2/server/web/context"
)

// Annotation defines route access config Annotation 定义路由访问配置
type Annotation struct {
	// AuthType specifies auth type AuthType 指定认证类型
	AuthType string `json:"authType"`
	// CheckLogin indicates login check CheckLogin 表示是否检查登录
	CheckLogin bool `json:"checkLogin"`
	// CheckRole lists required roles CheckRole 列出需要校验的角色
	CheckRole []string `json:"checkRole"`
	// CheckPermission lists required permissions CheckPermission 列出需要校验的权限
	CheckPermission []string `json:"checkPermission"`
	// CheckDisable indicates disable check CheckDisable 表示是否检查封禁
	CheckDisable bool `json:"checkDisable"`
	// Ignore bypasses authentication Ignore 表示是否忽略认证
	Ignore bool `json:"ignore"`
	// LogicType sets logic mode LogicType 指定逻辑类型
	LogicType LogicType `json:"logicType"`
}

// RouteAccessHandlerFromAnnotations creates route access handler RouteAccessHandlerFromAnnotations 根据注解创建路由访问处理器
func RouteAccessHandlerFromAnnotations(annotations ...*Annotation) RouteAccessHandler {
	return func(_ context.Context, _ *beegocontext.Context, req *RouteAccessRequest) {
		if len(annotations) == 0 || annotations[0] == nil {
			return
		}

		ann := annotations[0]
		if ann.AuthType != "" {
			req.AuthType = ann.AuthType
		}
		req.CheckDisable = ann.CheckDisable
		if ann.LogicType != "" {
			req.SetLogicType(ann.LogicType)
		}
		if ann.Ignore {
			req.SkipAuth()
			return
		}
		if !ann.CheckLogin && !ann.CheckDisable && len(ann.CheckPermission) == 0 && len(ann.CheckRole) == 0 {
			req.SkipAuth()
			return
		}
		if len(ann.CheckPermission) > 0 {
			req.RequirePermissions(ann.CheckPermission...)
		}
		if len(ann.CheckRole) > 0 {
			req.RequireRoles(ann.CheckRole...)
		}
	}
}

// RegisterAccessFilter registers access filter on default Beego app RegisterAccessFilter 在默认 Beego 应用注册访问过滤器
func RegisterAccessFilter(ctx context.Context, pattern string, opts ...AuthOption) {
	web.InsertFilter(pattern, web.BeforeRouter, AccessMiddleware(ctx, opts...), web.WithReturnOnOutput(true))
}

// RegisterAuthFilter registers auth filter on default Beego app RegisterAuthFilter 在默认 Beego 应用注册认证过滤器
func RegisterAuthFilter(ctx context.Context, pattern string, opts ...AuthOption) {
	web.InsertFilter(pattern, web.BeforeRouter, AuthMiddleware(ctx, opts...), web.WithReturnOnOutput(true))
}

// RegisterPermissionFilter registers permission filter on default Beego app RegisterPermissionFilter 在默认 Beego 应用注册权限过滤器
func RegisterPermissionFilter(ctx context.Context, pattern string, permissions []string, opts ...AuthOption) {
	web.InsertFilter(pattern, web.BeforeRouter, PermissionMiddleware(ctx, permissions, opts...), web.WithReturnOnOutput(true))
}

// RegisterRoleFilter registers role filter on default Beego app RegisterRoleFilter 在默认 Beego 应用注册角色过滤器
func RegisterRoleFilter(ctx context.Context, pattern string, roles []string, opts ...AuthOption) {
	web.InsertFilter(pattern, web.BeforeRouter, RoleMiddleware(ctx, roles, opts...), web.WithReturnOnOutput(true))
}
