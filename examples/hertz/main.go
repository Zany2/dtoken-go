// @Author daixk 2025/12/22 15:56:00
package main

import (
	dterror "github.com/Zany2/dtoken-go/errors"
	dt1 "github.com/Zany2/dtoken-go/global"

	"context"
	"errors"
	"github.com/Zany2/dtoken-go"
	"net/http"
	"time"

	hertzdt "github.com/Zany2/dtoken-go/integrations/hertz"
	hertzapp "github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/hlog"
)

// Response defines the example response body Response 定义示例响应结构
type Response struct {
	// Code stores the application response code. Code 保存应用响应码。
	Code int `json:"code"`
	// Message stores the response message. Message 保存响应消息。
	Message string `json:"message"`
	// Data stores the optional response payload. Data 保存可选响应数据。
	Data interface{} `json:"data,omitempty"`
}

// LoginRequest defines the login payload LoginRequest 定义登录请求参数
type LoginRequest struct {
	// Username stores the demo login name. Username 保存示例登录名。
	Username string `json:"username"`
	// Password stores the demo login password. Password 保存示例登录密码。
	Password string `json:"password"`
}

func main() {
	initDToken()
	defer dt1.DeleteAllManager()

	h := server.Default(server.WithHostPorts(":8080"))
	registerRoutes(h)
	h.Spin()
}

// registerRoutes shares the example's middleware and routes with regression tests. registerRoutes 与回归测试共用示例中间件和路由。
func registerRoutes(h *server.Hertz) {
	ctx := context.Background()
	h.Use(hertzdt.RegisterDTokenContextMiddleware(ctx, hertzdt.WithManagerResolver(dt1.GetManager), hertzdt.WithFailFunc(writeAuthError)))
	h.POST("/login", handleLogin)

	auth := h.Group("/")
	auth.Use(hertzdt.AuthMiddleware(ctx, hertzdt.WithFailFunc(writeAuthError)))
	auth.GET("/me", handleMe)
	auth.GET("/admin", hertzdt.RoleMiddleware(ctx, []string{"admin"}, hertzdt.WithFailFunc(writeAuthError)), handleAdmin)
	auth.GET("/articles", hertzdt.PermissionMiddleware(ctx, []string{"article:read"}, hertzdt.WithFailFunc(writeAuthError)), handleArticles)
	auth.POST("/logout", handleLogout)
}

// initDToken initializes integration manager initDToken 初始化集成管理器
func initDToken() {
	builtAuth1, err := dtoken.NewBuilder().
		Timeout(int64((2 * time.Hour).Seconds())).
		RenewMaxRefresh(int64(time.Hour.Seconds())).
		IsPrintBanner(false).
		Build()
	mgr := builtAuth1.Manager()

	if err != nil {
		panic(err)
	}

	dt1.SetManager(mgr)
}

// handleLogin logs in a demo user handleLogin 登录示例用户
func handleLogin(ctx context.Context, c *hertzapp.RequestContext) {
	var req LoginRequest
	if err := c.Bind(&req); err != nil || req.Username == "" || req.Password == "" {
		writeJSON(c, http.StatusBadRequest, dterror.CodeBadRequest, "username and password are required", nil)
		return
	}

	if req.Password != "123456" {
		writeJSON(c, http.StatusUnauthorized, dterror.CodeNotLogin, "invalid username or password", nil)
		return
	}

	token, err := dt1.Login(ctx, req.Username)
	if err != nil {
		writeAuthError(ctx, c, err)
		return
	}

	// Grant every demo user the same role and permission; real apps must load their own access rules. 为每个演示用户授予相同角色和权限；实际应用应加载自身的授权规则。
	if err = dt1.AddRoles(ctx, req.Username, []string{"admin"}); err != nil {
		writeAuthError(ctx, c, err)
		return
	}
	if err = dt1.AddPermissions(ctx, req.Username, []string{"article:read"}); err != nil {
		writeAuthError(ctx, c, err)
		return
	}

	writeJSON(c, http.StatusOK, dterror.CodeSuccess, "ok", map[string]interface{}{"token": token})
}

// handleMe returns current login information handleMe 返回当前登录信息
func handleMe(ctx context.Context, c *hertzapp.RequestContext) {
	dCtx, ok := hertzdt.GetDTokenContext(c)
	if !ok {
		writeJSON(c, http.StatusUnauthorized, dterror.CodeNotLogin, "not logged in", nil)
		return
	}

	loginID, err := dCtx.Auth().GetLoginID(ctx)
	if err != nil {
		writeAuthError(ctx, c, err)
		return
	}

	roles, err := dCtx.Access().GetRoles(ctx)
	if err != nil {
		writeAuthError(ctx, c, err)
		return
	}
	permissions, err := dCtx.Access().GetPermissions(ctx)
	if err != nil {
		writeAuthError(ctx, c, err)
		return
	}

	writeJSON(c, http.StatusOK, dterror.CodeSuccess, "ok", map[string]interface{}{
		"loginId":     loginID,
		"roles":       roles,
		"permissions": permissions,
	})
}

// handleAdmin returns admin data handleAdmin 返回管理员数据
func handleAdmin(_ context.Context, c *hertzapp.RequestContext) {
	writeJSON(c, http.StatusOK, dterror.CodeSuccess, "ok", map[string]interface{}{"scope": "admin"})
}

// handleArticles returns protected article data handleArticles 返回受保护的文章数据
func handleArticles(_ context.Context, c *hertzapp.RequestContext) {
	writeJSON(c, http.StatusOK, dterror.CodeSuccess, "ok", []string{"article-a", "article-b"})
}

// handleLogout logs out current token handleLogout 注销当前 Token
func handleLogout(ctx context.Context, c *hertzapp.RequestContext) {
	dCtx, ok := hertzdt.GetDTokenContext(c)
	if !ok {
		writeJSON(c, http.StatusUnauthorized, dterror.CodeNotLogin, "not logged in", nil)
		return
	}

	if err := dCtx.Auth().Logout(ctx); err != nil {
		writeAuthError(ctx, c, err)
		return
	}

	writeJSON(c, http.StatusOK, dterror.CodeSuccess, "ok", nil)
}

// writeAuthError distinguishes credential failures, access restrictions and server failures. writeAuthError 区分凭证无效、访问限制及服务端故障。
func writeAuthError(ctx context.Context, c *hertzapp.RequestContext, err error) {
	switch {
	case errors.Is(err, dterror.ErrNotLogin), errors.Is(err, dterror.ErrInvalidToken),
		errors.Is(err, dterror.ErrTokenExpired), errors.Is(err, dterror.ErrActiveTimeout),
		errors.Is(err, dterror.ErrTokenKickout), errors.Is(err, dterror.ErrTokenReplaced):
		writeJSON(c, http.StatusUnauthorized, dterror.CodeNotLogin, err.Error(), nil)
	case errors.Is(err, dterror.ErrAccountDisabled), errors.Is(err, dterror.ErrDeviceDisabled):
		writeJSON(c, http.StatusForbidden, dterror.CodeAccountDisabled, err.Error(), nil)
	case errors.Is(err, dterror.ErrPermissionDenied), errors.Is(err, dterror.ErrRoleDenied):
		writeJSON(c, http.StatusForbidden, dterror.CodePermissionDenied, err.Error(), nil)
	default:
		// Keep server details in logs rather than HTTP responses. 将服务端详情保留在日志中，不在 HTTP 响应中暴露。
		hlog.CtxErrorf(ctx, "dtoken request failed: %v", err)
		writeJSON(c, http.StatusInternalServerError, dterror.CodeServerError, "internal server error", nil)
	}
}

// writeJSON writes a unified JSON response writeJSON 写入统一 JSON 响应
func writeJSON(c *hertzapp.RequestContext, httpStatus int, code int, message string, data interface{}) {
	c.JSON(httpStatus, Response{
		Code:    code,
		Message: message,
		Data:    data,
	})
}
