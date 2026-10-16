// @Author daixk 2025/12/22 15:56:00
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	kratosdt "github.com/Zany2/dtoken-go/integrations/kratos"
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/middleware"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
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
	defer kratosdt.DeleteAllManager()

	srv := newKratosExampleServer()
	app := kratos.New(
		kratos.Name("dtoken-kratos-example"),
		kratos.Server(srv),
	)

	if err := app.Run(); err != nil {
		panic(err)
	}
}

// newKratosExampleServer shares server configuration and routes with regression tests. newKratosExampleServer 与回归测试共用服务器配置及路由。
func newKratosExampleServer() *khttp.Server {
	srv := khttp.NewServer(
		khttp.Address(":8080"),
		khttp.Middleware(kratosdt.RegisterDTokenContextMiddleware()),
	)

	r := srv.Route("/")
	r.POST("/login", wrapHandler(handleLogin))
	r.GET("/me", wrapHandler(handleMe, kratosdt.AuthMiddleware()))
	r.GET("/admin", wrapHandler(handleAdmin, kratosdt.AuthMiddleware(), kratosdt.RoleMiddleware([]string{"admin"})))
	r.GET("/articles", wrapHandler(handleArticles, kratosdt.AuthMiddleware(), kratosdt.PermissionMiddleware([]string{"article:read"})))
	r.POST("/logout", wrapHandler(handleLogout, kratosdt.AuthMiddleware()))

	return srv
}

// initDToken initializes integration manager initDToken 初始化集成管理器
func initDToken() {
	mgr, err := kratosdt.NewBuilder().
		Timeout(int64((2 * time.Hour).Seconds())).
		RenewMaxRefresh(int64(time.Hour.Seconds())).
		IsPrintBanner(false).
		Build()
	if err != nil {
		panic(err)
	}

	kratosdt.SetManager(mgr)
}

// wrapHandler applies Kratos middleware to a plain handler wrapHandler 为普通处理函数应用 Kratos 中间件
func wrapHandler(handler func(context.Context, khttp.Context) error, mws ...middleware.Middleware) khttp.HandlerFunc {
	return func(httpCtx khttp.Context) error {
		chained := middleware.Chain(mws...)(func(ctx context.Context, _ any) (any, error) {
			return nil, handler(ctx, httpCtx)
		})

		_, err := httpCtx.Middleware(chained)(httpCtx, nil)
		if err != nil {
			return writeAuthError(httpCtx, err)
		}
		return nil
	}
}

// handleLogin logs in a demo user handleLogin 登录示例用户
func handleLogin(ctx context.Context, httpCtx khttp.Context) error {
	var req LoginRequest
	if err := httpCtx.Bind(&req); err != nil || req.Username == "" || req.Password == "" {
		return writeJSON(httpCtx, http.StatusBadRequest, kratosdt.CodeBadRequest, "username and password are required", nil)
	}

	if req.Password != "123456" {
		return writeJSON(httpCtx, http.StatusUnauthorized, kratosdt.CodeNotLogin, "invalid username or password", nil)
	}

	token, err := kratosdt.Login(ctx, req.Username)
	if err != nil {
		return err
	}

	// Grant every demo user the same role and permission; real apps must load their own access rules. 为每个演示用户授予相同角色和权限；实际应用应加载自身的授权规则。
	if err = kratosdt.AddRoles(ctx, req.Username, []string{"admin"}); err != nil {
		return err
	}
	if err = kratosdt.AddPermissions(ctx, req.Username, []string{"article:read"}); err != nil {
		return err
	}

	return writeJSON(httpCtx, http.StatusOK, kratosdt.CodeSuccess, "ok", map[string]interface{}{"token": token})
}

// handleMe returns current login information handleMe 返回当前登录信息
func handleMe(ctx context.Context, httpCtx khttp.Context) error {
	dCtx, ok := kratosdt.GetDTokenContext(ctx)
	if !ok {
		return writeJSON(httpCtx, http.StatusUnauthorized, kratosdt.CodeNotLogin, "not logged in", nil)
	}

	loginID, err := dCtx.Auth().GetLoginID(ctx)
	if err != nil {
		return err
	}

	roles, err := dCtx.Access().GetRoles(ctx)
	if err != nil {
		return err
	}
	permissions, err := dCtx.Access().GetPermissions(ctx)
	if err != nil {
		return err
	}

	return writeJSON(httpCtx, http.StatusOK, kratosdt.CodeSuccess, "ok", map[string]interface{}{
		"loginId":     loginID,
		"roles":       roles,
		"permissions": permissions,
	})
}

// handleAdmin returns admin data handleAdmin 返回管理员数据
func handleAdmin(_ context.Context, httpCtx khttp.Context) error {
	return writeJSON(httpCtx, http.StatusOK, kratosdt.CodeSuccess, "ok", map[string]interface{}{"scope": "admin"})
}

// handleArticles returns protected article data handleArticles 返回受保护的文章数据
func handleArticles(_ context.Context, httpCtx khttp.Context) error {
	return writeJSON(httpCtx, http.StatusOK, kratosdt.CodeSuccess, "ok", []string{"article-a", "article-b"})
}

// handleLogout logs out current token handleLogout 注销当前 Token
func handleLogout(ctx context.Context, httpCtx khttp.Context) error {
	dCtx, ok := kratosdt.GetDTokenContext(ctx)
	if !ok {
		return writeJSON(httpCtx, http.StatusUnauthorized, kratosdt.CodeNotLogin, "not logged in", nil)
	}

	if err := dCtx.Auth().Logout(ctx); err != nil {
		return err
	}

	return writeJSON(httpCtx, http.StatusOK, kratosdt.CodeSuccess, "ok", nil)
}

// writeAuthError classifies both core errors and their Kratos wrappers through the error chain. writeAuthError 沿错误链统一分类核心错误及 Kratos 包装错误。
func writeAuthError(httpCtx khttp.Context, err error) error {
	switch {
	case errors.Is(err, kratosdt.ErrNotLogin), errors.Is(err, kratosdt.ErrInvalidToken),
		errors.Is(err, kratosdt.ErrTokenExpired), errors.Is(err, kratosdt.ErrActiveTimeout),
		errors.Is(err, kratosdt.ErrTokenKickout), errors.Is(err, kratosdt.ErrTokenReplaced):
		return writeJSON(httpCtx, http.StatusUnauthorized, kratosdt.CodeNotLogin, "not logged in or token invalid", nil)
	case errors.Is(err, kratosdt.ErrAccountDisabled), errors.Is(err, kratosdt.ErrDeviceDisabled):
		return writeJSON(httpCtx, http.StatusForbidden, kratosdt.CodeAccountDisabled, "account or device disabled", nil)
	case errors.Is(err, kratosdt.ErrPermissionDenied), errors.Is(err, kratosdt.ErrRoleDenied):
		return writeJSON(httpCtx, http.StatusForbidden, kratosdt.CodePermissionDenied, "permission denied", nil)
	default:
		// Keep server details in logs rather than HTTP responses. 将服务端详情保留在日志中，不在 HTTP 响应中暴露。
		log.Printf("dtoken request failed: %v", err)
		return writeJSON(httpCtx, http.StatusInternalServerError, kratosdt.CodeServerError, "internal server error", nil)
	}
}

// writeJSON writes a unified JSON response writeJSON 写入统一 JSON 响应
func writeJSON(ctx khttp.Context, httpStatus int, code int, message string, data interface{}) error {
	return ctx.JSON(httpStatus, Response{
		Code:    code,
		Message: message,
		Data:    data,
	})
}
