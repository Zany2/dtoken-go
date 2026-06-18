// @Author daixk 2025/12/22 15:56:00
package main

import (
	"context"
	"net/http"
	"time"

	hertzdt "github.com/Zany2/dtoken-go/integrations/hertz"
	hertzapp "github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
)

// Response defines the example response body Response 定义示例响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// LoginRequest defines the login payload LoginRequest 定义登录请求参数
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func main() {
	ctx := context.Background()
	initDToken()

	h := server.Default(server.WithHostPorts(":8080"))
	h.Use(hertzdt.RegisterDTokenContextMiddleware(ctx))
	h.POST("/login", handleLogin)

	auth := h.Group("/")
	auth.Use(hertzdt.AuthMiddleware(ctx))
	auth.GET("/me", handleMe)
	auth.GET("/admin", hertzdt.RoleMiddleware(ctx, []string{"admin"}), handleAdmin)
	auth.GET("/articles", hertzdt.PermissionMiddleware(ctx, []string{"article:read"}), handleArticles)
	auth.POST("/logout", handleLogout)

	h.Spin()
}

// initDToken initializes integration manager initDToken 初始化集成管理器
func initDToken() {
	mgr, err := hertzdt.NewBuilder().
		Timeout(int64((2 * time.Hour).Seconds())).
		IsPrintBanner(false).
		Build()
	if err != nil {
		panic(err)
	}

	hertzdt.SetManager(mgr)
}

// handleLogin logs in a demo user handleLogin 登录示例用户
func handleLogin(ctx context.Context, c *hertzapp.RequestContext) {
	var req LoginRequest
	if err := c.Bind(&req); err != nil || req.Username == "" || req.Password == "" {
		writeJSON(c, http.StatusBadRequest, hertzdt.CodeBadRequest, "username and password are required", nil)
		return
	}

	if req.Password != "123456" {
		writeJSON(c, http.StatusUnauthorized, hertzdt.CodeNotLogin, "invalid username or password", nil)
		return
	}

	token, err := hertzdt.Login(ctx, req.Username)
	if err != nil {
		writeJSON(c, http.StatusInternalServerError, hertzdt.CodeServerError, err.Error(), nil)
		return
	}

	// Seed demo authorization data 初始化示例权限数据
	_ = hertzdt.AddRoles(ctx, req.Username, []string{"admin"})
	_ = hertzdt.AddPermissions(ctx, req.Username, []string{"article:read"})

	writeJSON(c, http.StatusOK, hertzdt.CodeSuccess, "ok", map[string]interface{}{"token": token})
}

// handleMe returns current login information handleMe 返回当前登录信息
func handleMe(ctx context.Context, c *hertzapp.RequestContext) {
	dCtx, ok := hertzdt.GetDTokenContext(c)
	if !ok {
		writeJSON(c, http.StatusUnauthorized, hertzdt.CodeNotLogin, "not logged in", nil)
		return
	}

	loginID, err := dCtx.Auth().GetLoginID(ctx)
	if err != nil {
		writeJSON(c, http.StatusUnauthorized, hertzdt.CodeNotLogin, err.Error(), nil)
		return
	}

	roles, _ := dCtx.Access().GetRoles(ctx)
	permissions, _ := dCtx.Access().GetPermissions(ctx)

	writeJSON(c, http.StatusOK, hertzdt.CodeSuccess, "ok", map[string]interface{}{
		"loginId":     loginID,
		"roles":       roles,
		"permissions": permissions,
	})
}

// handleAdmin returns admin data handleAdmin 返回管理员数据
func handleAdmin(_ context.Context, c *hertzapp.RequestContext) {
	writeJSON(c, http.StatusOK, hertzdt.CodeSuccess, "ok", map[string]interface{}{"scope": "admin"})
}

// handleArticles returns protected article data handleArticles 返回受保护的文章数据
func handleArticles(_ context.Context, c *hertzapp.RequestContext) {
	writeJSON(c, http.StatusOK, hertzdt.CodeSuccess, "ok", []string{"article-a", "article-b"})
}

// handleLogout logs out current token handleLogout 注销当前 Token
func handleLogout(ctx context.Context, c *hertzapp.RequestContext) {
	dCtx, ok := hertzdt.GetDTokenContext(c)
	if !ok {
		writeJSON(c, http.StatusUnauthorized, hertzdt.CodeNotLogin, "not logged in", nil)
		return
	}

	if err := dCtx.Auth().Logout(ctx); err != nil {
		writeJSON(c, http.StatusInternalServerError, hertzdt.CodeServerError, err.Error(), nil)
		return
	}

	writeJSON(c, http.StatusOK, hertzdt.CodeSuccess, "ok", nil)
}

// writeJSON writes a unified JSON response writeJSON 写入统一 JSON 响应
func writeJSON(c *hertzapp.RequestContext, httpStatus int, code int, message string, data interface{}) {
	c.JSON(httpStatus, Response{
		Code:    code,
		Message: message,
		Data:    data,
	})
}
