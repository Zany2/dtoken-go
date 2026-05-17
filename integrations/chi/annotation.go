// @Author daixk 2025/12/22 15:56:00
package chi

import (
	"context"
	"net/http"

	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/integrations/internal/authcheck"
	chiRouter "github.com/go-chi/chi/v5"
)

// Annotation defines annotation config Annotation 定义注解配置
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

// GetHandler gets annotation handler GetHandler 获取注解处理器
func GetHandler(handler http.HandlerFunc, annotations ...*Annotation) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if len(annotations) > 0 && annotations[0].Ignore {
			if handler != nil {
				handler(w, r)
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
				handler(w, r)
			}
			return
		}

		mgr, err := authcheck.GetManager(ann.AuthType)
		if err != nil {
			writeErrorResponse(w, err)
			return
		}

		// Get DTokenContext (reuse cached context) 获取 DTokenContext（复用缓存上下文）
		chiCtx := NewChiContext(w, r).(*ChiContext)
		dCtx := getDTokenContext(chiCtx, mgr)
		r = chiCtx.r
		token := dCtx.GetTokenValue()

		_, err = authcheck.Check(r.Context(), mgr, authcheck.Request{
			TokenValue:   token,
			CheckLogin:   true,
			CheckDisable: ann.CheckDisable,
			Permissions:  ann.CheckPermission,
			Roles:        ann.CheckRole,
			LogicType:    ann.LogicType,
			LoginError:   derror.ErrNotLogin,
		})
		if err != nil {
			writeErrorResponse(w, err)
			return
		}

		if handler != nil {
			handler(w, r)
		}
	}
}

// CheckLoginHandler creates login check handler CheckLoginHandler 创建登录检查处理器
func CheckLoginHandler(authType ...string) http.HandlerFunc {
	ann := &Annotation{CheckLogin: true}
	if len(authType) > 0 {
		ann.AuthType = authType[0]
	}
	return GetHandler(nil, ann)
}

// CheckRoleHandler creates role check handler CheckRoleHandler 创建角色检查处理器
func CheckRoleHandler(roles ...string) http.HandlerFunc {
	return GetHandler(nil, &Annotation{CheckRole: roles})
}

// CheckRoleHandlerWithAuthType creates role check handler CheckRoleHandlerWithAuthType 创建带认证类型的角色检查处理器
func CheckRoleHandlerWithAuthType(authType string, roles ...string) http.HandlerFunc {
	return GetHandler(nil, &Annotation{CheckRole: roles, AuthType: authType})
}

// CheckPermissionHandler creates permission check handler CheckPermissionHandler 创建权限检查处理器
func CheckPermissionHandler(perms ...string) http.HandlerFunc {
	return GetHandler(nil, &Annotation{CheckPermission: perms})
}

// CheckPermissionHandlerWithAuthType creates permission check handler CheckPermissionHandlerWithAuthType 创建带认证类型的权限检查处理器
func CheckPermissionHandlerWithAuthType(authType string, perms ...string) http.HandlerFunc {
	return GetHandler(nil, &Annotation{CheckPermission: perms, AuthType: authType})
}

// CheckDisableHandler creates disable check handler CheckDisableHandler 创建封禁检查处理器
func CheckDisableHandler(authType ...string) http.HandlerFunc {
	ann := &Annotation{CheckDisable: true}
	if len(authType) > 0 {
		ann.AuthType = authType[0]
	}
	return GetHandler(nil, ann)
}

// IgnoreHandler creates ignore auth handler IgnoreHandler 创建忽略认证处理器
func IgnoreHandler() http.HandlerFunc {
	return GetHandler(nil, &Annotation{Ignore: true})
}

// CheckLoginAndRoleHandler creates login and role handler CheckLoginAndRoleHandler 创建登录与角色检查处理器
func CheckLoginAndRoleHandler(roles ...string) http.HandlerFunc {
	return GetHandler(nil, &Annotation{CheckLogin: true, CheckRole: roles})
}

// CheckLoginAndPermissionHandler creates login and permission handler CheckLoginAndPermissionHandler 创建登录与权限检查处理器
func CheckLoginAndPermissionHandler(perms ...string) http.HandlerFunc {
	return GetHandler(nil, &Annotation{CheckLogin: true, CheckPermission: perms})
}

// CheckAllHandler creates combined auth handler CheckAllHandler 创建组合认证检查处理器
func CheckAllHandler(roles []string, perms []string) http.HandlerFunc {
	return GetHandler(nil, &Annotation{
		CheckLogin:      true,
		CheckRole:       roles,
		CheckPermission: perms,
		CheckDisable:    true,
	})
}

// AuthGroup creates auth route group AuthGroup 创建认证路由组
func AuthGroup(group chiRouter.Router, opts ...AuthOption) chiRouter.Router {
	group.Use(AuthMiddleware(opts...))
	return group
}

// RoleGroup creates role route group RoleGroup 创建角色路由组
func RoleGroup(group chiRouter.Router, roles []string, opts ...AuthOption) chiRouter.Router {
	group.Use(AuthMiddleware(opts...))
	group.Use(RoleMiddleware(roles, opts...))
	return group
}

// PermissionGroup creates permission route group PermissionGroup 创建权限路由组
func PermissionGroup(group chiRouter.Router, perms []string, opts ...AuthOption) chiRouter.Router {
	group.Use(AuthMiddleware(opts...))
	group.Use(PermissionMiddleware(perms, opts...))
	return group
}

// RoleAndPermissionGroup creates role and permission route group RoleAndPermissionGroup 创建角色与权限路由组
func RoleAndPermissionGroup(group chiRouter.Router, roles []string, perms []string, opts ...AuthOption) chiRouter.Router {
	group.Use(AuthMiddleware(opts...))
	group.Use(RoleMiddleware(roles, opts...))
	group.Use(PermissionMiddleware(perms, opts...))
	return group
}

// GetLoginIDFromRequest gets login ID from request GetLoginIDFromRequest 从请求获取登录 ID
func GetLoginIDFromRequest(w http.ResponseWriter, r *http.Request, authType ...string) (string, error) {
	mgr, err := authcheck.GetManager(firstAuthType(authType...))
	if err != nil {
		return "", err
	}

	chiCtx := NewChiContext(w, r).(*ChiContext)
	dCtx := getDTokenContext(chiCtx, mgr)
	return dCtx.GetLoginID(chiCtx.r.Context())
}

// IsLoginFromRequest checks login state from request IsLoginFromRequest 从请求检查登录状态
func IsLoginFromRequest(w http.ResponseWriter, r *http.Request, authType ...string) bool {
	mgr, err := authcheck.GetManager(firstAuthType(authType...))
	if err != nil {
		return false
	}

	chiCtx := NewChiContext(w, r).(*ChiContext)
	dCtx := getDTokenContext(chiCtx, mgr)
	_, err = authcheck.Check(chiCtx.r.Context(), mgr, authcheck.Request{
		TokenValue: dCtx.GetTokenValue(),
		CheckLogin: true,
	})
	return err == nil
}

// GetTokenFromRequest gets token from request GetTokenFromRequest 从请求获取 Token
func GetTokenFromRequest(w http.ResponseWriter, r *http.Request, authType ...string) string {
	mgr, err := authcheck.GetManager(firstAuthType(authType...))
	if err != nil {
		return ""
	}

	chiCtx := NewChiContext(w, r).(*ChiContext)
	dCtx := getDTokenContext(chiCtx, mgr)
	return dCtx.GetTokenValue()
}

// WithContext returns request context WithContext 返回请求上下文
func WithContext(r *http.Request, authType ...string) context.Context {
	return r.Context()
}

// firstAuthType returns the optional auth type firstAuthType 返回可选认证类型
func firstAuthType(authType ...string) string {
	if len(authType) == 0 {
		return ""
	}
	return authType[0]
}
