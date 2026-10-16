// @Author daixk 2025/12/22 15:56:00
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	fiberdt "github.com/Zany2/dtoken-go/integrations/fiber"
	gofiber "github.com/gofiber/fiber/v2"
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
	Username string `json:"username" form:"username"`
	// Password stores the demo login password. Password 保存示例登录密码。
	Password string `json:"password" form:"password"`
}

func main() {
	initDToken()
	defer fiberdt.DeleteAllManager()

	app := gofiber.New()
	registerRoutes(app)

	if err := app.Listen(":8080"); err != nil {
		panic(err)
	}
}

// registerRoutes shares production middleware and routes with regression tests. registerRoutes 与回归测试共用实际中间件和路由。
func registerRoutes(app *gofiber.App) {
	ctx := context.Background()
	fail := func(c *gofiber.Ctx, err error) { _ = writeAuthError(c, err) }
	app.Use(fiberdt.RegisterDTokenContextMiddleware(ctx, fiberdt.WithFailFunc(fail)))
	app.Post("/login", handleLogin)

	auth := app.Group("")
	auth.Use(fiberdt.AuthMiddleware(ctx, fiberdt.WithFailFunc(fail)))
	auth.Get("/me", handleMe)
	auth.Get("/admin", fiberdt.RoleMiddleware(ctx, []string{"admin"}, fiberdt.WithFailFunc(fail)), handleAdmin)
	auth.Get("/articles", fiberdt.PermissionMiddleware(ctx, []string{"article:read"}, fiberdt.WithFailFunc(fail)), handleArticles)
	auth.Post("/logout", handleLogout)
}

// initDToken initializes integration manager initDToken 初始化集成管理器
func initDToken() {
	mgr, err := fiberdt.NewBuilder().
		Timeout(int64((2 * time.Hour).Seconds())).
		RenewMaxRefresh(int64(time.Hour.Seconds())).
		IsPrintBanner(false).
		Build()
	if err != nil {
		panic(err)
	}

	fiberdt.SetManager(mgr)
}

// handleLogin logs in a demo user handleLogin 登录示例用户
func handleLogin(c *gofiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil || req.Username == "" || req.Password == "" {
		return writeJSON(c, http.StatusBadRequest, fiberdt.CodeBadRequest, "username and password are required", nil)
	}

	if req.Password != "123456" {
		return writeJSON(c, http.StatusUnauthorized, fiberdt.CodeNotLogin, "invalid username or password", nil)
	}

	// Form strings may share Fiber's request buffer; login events can outlive the request. 表单字符串可能引用 Fiber 请求缓冲区，而登录事件可能在请求结束后执行。
	req.Username = strings.Clone(req.Username)

	token, err := fiberdt.Login(c.UserContext(), req.Username)
	if err != nil {
		return writeAuthError(c, err)
	}

	// Grant every demo user the same role and permission; real apps must load their own access rules. 为每个演示用户授予相同角色和权限；实际应用应加载自身的授权规则。
	if err = fiberdt.AddRoles(c.UserContext(), req.Username, []string{"admin"}); err != nil {
		return writeAuthError(c, err)
	}
	if err = fiberdt.AddPermissions(c.UserContext(), req.Username, []string{"article:read"}); err != nil {
		return writeAuthError(c, err)
	}

	return writeJSON(c, http.StatusOK, fiberdt.CodeSuccess, "ok", gofiber.Map{"token": token})
}

// handleMe returns current login information handleMe 返回当前登录信息
func handleMe(c *gofiber.Ctx) error {
	dCtx, ok := fiberdt.GetDTokenContext(c)
	if !ok {
		return writeJSON(c, http.StatusUnauthorized, fiberdt.CodeNotLogin, "not logged in", nil)
	}

	loginID, err := dCtx.Auth().GetLoginID(c.UserContext())
	if err != nil {
		return writeAuthError(c, err)
	}

	roles, err := dCtx.Access().GetRoles(c.UserContext())
	if err != nil {
		return writeAuthError(c, err)
	}
	permissions, err := dCtx.Access().GetPermissions(c.UserContext())
	if err != nil {
		return writeAuthError(c, err)
	}

	return writeJSON(c, http.StatusOK, fiberdt.CodeSuccess, "ok", gofiber.Map{
		"loginId":     loginID,
		"roles":       roles,
		"permissions": permissions,
	})
}

// handleAdmin returns admin data handleAdmin 返回管理员数据
func handleAdmin(c *gofiber.Ctx) error {
	return writeJSON(c, http.StatusOK, fiberdt.CodeSuccess, "ok", gofiber.Map{"scope": "admin"})
}

// handleArticles returns protected article data handleArticles 返回受保护的文章数据
func handleArticles(c *gofiber.Ctx) error {
	return writeJSON(c, http.StatusOK, fiberdt.CodeSuccess, "ok", []string{"article-a", "article-b"})
}

// handleLogout logs out current token handleLogout 注销当前 Token
func handleLogout(c *gofiber.Ctx) error {
	dCtx, ok := fiberdt.GetDTokenContext(c)
	if !ok {
		return writeJSON(c, http.StatusUnauthorized, fiberdt.CodeNotLogin, "not logged in", nil)
	}

	if err := dCtx.Auth().Logout(c.UserContext()); err != nil {
		return writeAuthError(c, err)
	}

	return writeJSON(c, http.StatusOK, fiberdt.CodeSuccess, "ok", nil)
}

// writeAuthError distinguishes credential failures from access restrictions and server failures. writeAuthError 区分凭证无效、访问限制及服务端故障。
func writeAuthError(c *gofiber.Ctx, err error) error {
	switch {
	case errors.Is(err, fiberdt.ErrNotLogin), errors.Is(err, fiberdt.ErrInvalidToken),
		errors.Is(err, fiberdt.ErrTokenExpired), errors.Is(err, fiberdt.ErrActiveTimeout),
		errors.Is(err, fiberdt.ErrTokenKickout), errors.Is(err, fiberdt.ErrTokenReplaced):
		return writeJSON(c, http.StatusUnauthorized, fiberdt.CodeNotLogin, err.Error(), nil)
	case errors.Is(err, fiberdt.ErrAccountDisabled), errors.Is(err, fiberdt.ErrDeviceDisabled):
		return writeJSON(c, http.StatusForbidden, fiberdt.CodeAccountDisabled, err.Error(), nil)
	case errors.Is(err, fiberdt.ErrPermissionDenied), errors.Is(err, fiberdt.ErrRoleDenied):
		return writeJSON(c, http.StatusForbidden, fiberdt.CodePermissionDenied, err.Error(), nil)
	default:
		// Keep server details in logs rather than HTTP responses. 将服务端详情保留在日志中，不在 HTTP 响应中暴露。
		log.Printf("dtoken request failed: %v", err)
		return writeJSON(c, http.StatusInternalServerError, fiberdt.CodeServerError, "internal server error", nil)
	}
}

// writeJSON writes a unified JSON response writeJSON 写入统一 JSON 响应
func writeJSON(c *gofiber.Ctx, httpStatus int, code int, message string, data interface{}) error {
	return c.Status(httpStatus).JSON(Response{
		Code:    code,
		Message: message,
		Data:    data,
	})
}
