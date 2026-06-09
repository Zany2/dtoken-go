// @Author daixk 2026/06/08
package main

import (
	"context"
	"net/http"
	"time"

	beegodt "github.com/Zany2/dtoken-go/integrations/beego"
	web "github.com/beego/beego/v2/server/web"
	beegocontext "github.com/beego/beego/v2/server/web/context"
)

// Response defines the example response body Response 定义示例响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func main() {
	ctx := context.Background()
	initDToken()

	web.InsertFilter("/*", web.BeforeRouter, beegodt.RegisterDTokenContextMiddleware(ctx))
	web.Post("/login", handleLogin)

	web.InsertFilter("/me", web.BeforeRouter, beegodt.AuthMiddleware(ctx))
	web.Get("/me", handleMe)

	web.InsertFilter("/admin", web.BeforeRouter, beegodt.AuthMiddleware(ctx))
	web.InsertFilter("/admin", web.BeforeRouter, beegodt.RoleMiddleware(ctx, []string{"admin"}))
	web.Get("/admin", handleAdmin)

	web.InsertFilter("/articles", web.BeforeRouter, beegodt.AuthMiddleware(ctx))
	web.InsertFilter("/articles", web.BeforeRouter, beegodt.PermissionMiddleware(ctx, []string{"article:read"}))
	web.Get("/articles", handleArticles)

	web.InsertFilter("/logout", web.BeforeRouter, beegodt.AuthMiddleware(ctx))
	web.Post("/logout", handleLogout)

	web.Run(":8080")
}

// initDToken initializes integration manager initDToken 初始化集成管理器
func initDToken() {
	mgr, err := beegodt.NewBuilder().
		Timeout(int64((2 * time.Hour).Seconds())).
		IsPrintBanner(false).
		Build()
	if err != nil {
		panic(err)
	}

	beegodt.SetManager(mgr)
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
		writeJSON(c, http.StatusInternalServerError, beegodt.CodeServerError, err.Error(), nil)
		return
	}

	// Seed demo authorization data 初始化示例权限数据
	_ = beegodt.AddRoles(c.Request.Context(), username, []string{"admin"})
	_ = beegodt.AddPermissions(c.Request.Context(), username, []string{"article:read"})

	writeJSON(c, http.StatusOK, beegodt.CodeSuccess, "ok", map[string]interface{}{"token": token})
}

// handleMe returns current login information handleMe 返回当前登录信息
func handleMe(c *beegocontext.Context) {
	dCtx, ok := beegodt.GetDTokenContext(c)
	if !ok {
		writeJSON(c, http.StatusUnauthorized, beegodt.CodeNotLogin, "not logged in", nil)
		return
	}

	loginID, err := dCtx.GetLoginID(c.Request.Context())
	if err != nil {
		writeJSON(c, http.StatusUnauthorized, beegodt.CodeNotLogin, err.Error(), nil)
		return
	}

	roles, _ := dCtx.GetRoles(c.Request.Context())
	permissions, _ := dCtx.GetPermissions(c.Request.Context())

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

	if err := dCtx.Logout(c.Request.Context()); err != nil {
		writeJSON(c, http.StatusInternalServerError, beegodt.CodeServerError, err.Error(), nil)
		return
	}

	writeJSON(c, http.StatusOK, beegodt.CodeSuccess, "ok", nil)
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
