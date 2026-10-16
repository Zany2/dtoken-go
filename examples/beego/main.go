// @Author daixk 2026/06/08
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	beegodt "github.com/Zany2/dtoken-go/integrations/beego"
	web "github.com/beego/beego/v2/server/web"
	beegocontext "github.com/beego/beego/v2/server/web/context"
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

func main() {
	initDToken()
	defer beegodt.DeleteAllManager()

	registerRoutes(web.BeeApp.Handlers)

	// Let Beego drain requests before Run returns and managers are closed. 让 Beego 在 Run 返回、关闭 Manager 之前等待请求结束。
	web.BConfig.Listen.Graceful = true
	web.Run(":8080")
}

// registerRoutes shares the actual filters and routes with regression tests. registerRoutes 与回归测试共用实际过滤器和路由。
func registerRoutes(r *web.ControllerRegister) {
	ctx := context.Background()
	insertFilter := func(path string, position int, filter web.FilterFunc) {
		if err := r.InsertFilter(path, position, filter, web.WithReturnOnOutput(true)); err != nil {
			panic(err)
		}
	}

	// Validate before Beego's automatic form parser turns malformed input into HTTP 500. 在 Beego 自动表单解析之前校验，避免非法输入被归为 HTTP 500。
	insertFilter("/login", web.BeforeStatic, validateLoginInput)
	insertFilter("/*", web.BeforeRouter, beegodt.RegisterDTokenContextMiddleware(ctx, beegodt.WithFailFunc(writeAuthError)))
	r.Post("/login", handleLogin)

	insertFilter("/me", web.BeforeRouter, beegodt.AuthMiddleware(ctx, beegodt.WithFailFunc(writeAuthError)))
	r.Get("/me", handleMe)

	insertFilter("/admin", web.BeforeRouter, beegodt.AuthMiddleware(ctx, beegodt.WithFailFunc(writeAuthError)))
	insertFilter("/admin", web.BeforeRouter, beegodt.RoleMiddleware(ctx, []string{"admin"}, beegodt.WithFailFunc(writeAuthError)))
	r.Get("/admin", handleAdmin)

	insertFilter("/articles", web.BeforeRouter, beegodt.AuthMiddleware(ctx, beegodt.WithFailFunc(writeAuthError)))
	insertFilter("/articles", web.BeforeRouter, beegodt.PermissionMiddleware(ctx, []string{"article:read"}, beegodt.WithFailFunc(writeAuthError)))
	r.Get("/articles", handleArticles)

	insertFilter("/logout", web.BeforeRouter, beegodt.AuthMiddleware(ctx, beegodt.WithFailFunc(writeAuthError)))
	r.Post("/logout", handleLogout)
}

// initDToken initializes integration manager initDToken 初始化集成管理器
func initDToken() {
	mgr, err := beegodt.NewBuilder().
		Timeout(int64((2 * time.Hour).Seconds())).
		RenewMaxRefresh(int64(time.Hour.Seconds())).
		IsPrintBanner(false).
		Build()
	if err != nil {
		panic(err)
	}

	beegodt.SetManager(mgr)
}

// validateLoginInput retains Query/form support while rejecting partial parse results. validateLoginInput 保留 Query／表单支持，同时拒绝解析失败产生的部分结果。
func validateLoginInput(c *beegocontext.Context) {
	if c.Request.Method != http.MethodPost {
		return
	}

	// Match Beego's body limits before consuming the form. 在读取表单前沿用 Beego 的请求体大小限制。
	bodyLimit := web.BConfig.MaxMemory
	if c.Input.IsUpload() {
		bodyLimit = web.BConfig.MaxUploadSize
	}
	if c.Request.Body != nil {
		c.Request.Body = http.MaxBytesReader(c.ResponseWriter, c.Request.Body, bodyLimit)
	}
	if err := c.Input.ParseFormOrMultiForm(web.BConfig.MaxMemory); err != nil {
		writeJSON(c, http.StatusBadRequest, beegodt.CodeBadRequest, "invalid login input", nil)
	}
}

// handleLogin logs in a demo user handleLogin 登录示例用户
func handleLogin(c *beegocontext.Context) {
	username := c.Input.Query("username")
	password := c.Input.Query("password")
	if username == "" || password == "" {
		writeJSON(c, http.StatusBadRequest, beegodt.CodeBadRequest, "username and password are required", nil)
		return
	}

	if password != "123456" {
		writeJSON(c, http.StatusUnauthorized, beegodt.CodeNotLogin, "invalid username or password", nil)
		return
	}

	token, err := beegodt.Login(c.Request.Context(), username)
	if err != nil {
		writeAuthError(c, err)
		return
	}

	// Grant demo access to every user; real applications must load their own access rules. 为每个演示用户授予权限；实际应用应加载自身的授权规则。
	if err = beegodt.AddRoles(c.Request.Context(), username, []string{"admin"}); err != nil {
		writeAuthError(c, err)
		return
	}
	if err = beegodt.AddPermissions(c.Request.Context(), username, []string{"article:read"}); err != nil {
		writeAuthError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, beegodt.CodeSuccess, "ok", map[string]interface{}{"token": token})
}

// handleMe returns current login information handleMe 返回当前登录信息
func handleMe(c *beegocontext.Context) {
	dCtx, ok := beegodt.GetDTokenContext(c)
	if !ok {
		writeJSON(c, http.StatusUnauthorized, beegodt.CodeNotLogin, "not logged in", nil)
		return
	}

	loginID, err := dCtx.Auth().GetLoginID(c.Request.Context())
	if err != nil {
		writeAuthError(c, err)
		return
	}

	roles, err := dCtx.Access().GetRoles(c.Request.Context())
	if err != nil {
		writeAuthError(c, err)
		return
	}
	permissions, err := dCtx.Access().GetPermissions(c.Request.Context())
	if err != nil {
		writeAuthError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, beegodt.CodeSuccess, "ok", map[string]interface{}{
		"loginId":     loginID,
		"roles":       roles,
		"permissions": permissions,
	})
}

// handleAdmin returns admin data handleAdmin 返回管理员数据
func handleAdmin(c *beegocontext.Context) {
	writeJSON(c, http.StatusOK, beegodt.CodeSuccess, "ok", map[string]interface{}{"scope": "admin"})
}

// handleArticles returns protected article data handleArticles 返回受保护的文章数据
func handleArticles(c *beegocontext.Context) {
	writeJSON(c, http.StatusOK, beegodt.CodeSuccess, "ok", []string{"article-a", "article-b"})
}

// handleLogout logs out current token handleLogout 注销当前 Token
func handleLogout(c *beegocontext.Context) {
	dCtx, ok := beegodt.GetDTokenContext(c)
	if !ok {
		writeJSON(c, http.StatusUnauthorized, beegodt.CodeNotLogin, "not logged in", nil)
		return
	}

	if err := dCtx.Auth().Logout(c.Request.Context()); err != nil {
		writeAuthError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, beegodt.CodeSuccess, "ok", nil)
}

// writeAuthError uses one response policy for filters and handlers. writeAuthError 为过滤器与处理器使用统一的错误响应规则。
func writeAuthError(c *beegocontext.Context, err error) {
	switch {
	case errors.Is(err, beegodt.ErrNotLogin), errors.Is(err, beegodt.ErrInvalidToken),
		errors.Is(err, beegodt.ErrTokenExpired), errors.Is(err, beegodt.ErrActiveTimeout),
		errors.Is(err, beegodt.ErrTokenKickout), errors.Is(err, beegodt.ErrTokenReplaced):
		writeJSON(c, http.StatusUnauthorized, beegodt.CodeNotLogin, "not logged in or token invalid", nil)
	case errors.Is(err, beegodt.ErrAccountDisabled), errors.Is(err, beegodt.ErrDeviceDisabled):
		writeJSON(c, http.StatusForbidden, beegodt.CodeAccountDisabled, "account or device disabled", nil)
	case errors.Is(err, beegodt.ErrPermissionDenied), errors.Is(err, beegodt.ErrRoleDenied):
		writeJSON(c, http.StatusForbidden, beegodt.CodePermissionDenied, "permission denied", nil)
	default:
		// Keep internal details in server logs. 将内部详情保留在服务端日志中。
		log.Printf("dtoken request failed: %v", err)
		writeJSON(c, http.StatusInternalServerError, beegodt.CodeServerError, "internal server error", nil)
	}
}

// writeJSON writes a unified JSON response writeJSON 写入统一 JSON 响应
func writeJSON(c *beegocontext.Context, httpStatus int, code int, message string, data interface{}) {
	c.Output.SetStatus(httpStatus)
	_ = c.Output.JSON(Response{
		Code:    code,
		Message: message,
		Data:    data,
	}, false, false)
}
