// @Author daixk 2025/12/22 15:56:00
package main

import (
	"context"
	"net/http"
	"time"

	echodt "github.com/Zany2/dtoken-go/integrations/echo"
	echo4 "github.com/labstack/echo/v4"
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

	e := echo4.New()
	e.Use(echodt.RegisterDTokenContextMiddleware(ctx))
	e.POST("/login", handleLogin)

	auth := e.Group("")
	auth.Use(echodt.AuthMiddleware(ctx))
	auth.GET("/me", handleMe)
	auth.GET("/admin", handleAdmin, echodt.RoleMiddleware(ctx, []string{"admin"}))
	auth.GET("/articles", handleArticles, echodt.PermissionMiddleware(ctx, []string{"article:read"}))
	auth.POST("/logout", handleLogout)

	e.Logger.Fatal(e.Start(":8080"))
}

// initDToken initializes integration manager initDToken 初始化集成管理器
func initDToken() {
	mgr, err := echodt.NewBuilder().
		Timeout(int64((2 * time.Hour).Seconds())).
		IsPrintBanner(false).
		Build()
	if err != nil {
		panic(err)
	}

	echodt.SetManager(mgr)
}

// handleLogin logs in a demo user handleLogin 登录示例用户
func handleLogin(c echo4.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil || req.Username == "" || req.Password == "" {
		return writeJSON(c, http.StatusBadRequest, echodt.CodeBadRequest, "username and password are required", nil)
	}

	if req.Password != "123456" {
		return writeJSON(c, http.StatusUnauthorized, echodt.CodeNotLogin, "invalid username or password", nil)
	}

	token, err := echodt.Login(c.Request().Context(), req.Username)
	if err != nil {
		return writeJSON(c, http.StatusInternalServerError, echodt.CodeServerError, err.Error(), nil)
	}

	// Seed demo authorization data 初始化示例权限数据
	_ = echodt.AddRoles(c.Request().Context(), req.Username, []string{"admin"})
	_ = echodt.AddPermissions(c.Request().Context(), req.Username, []string{"article:read"})

	return writeJSON(c, http.StatusOK, echodt.CodeSuccess, "ok", echo4.Map{"token": token})
}

// handleMe returns current login information handleMe 返回当前登录信息
func handleMe(c echo4.Context) error {
	dCtx, ok := echodt.GetDTokenContext(c)
	if !ok {
		return writeJSON(c, http.StatusUnauthorized, echodt.CodeNotLogin, "not logged in", nil)
	}

	loginID, err := dCtx.GetLoginID(c.Request().Context())
	if err != nil {
		return writeJSON(c, http.StatusUnauthorized, echodt.CodeNotLogin, err.Error(), nil)
	}

	roles, _ := dCtx.GetRoles(c.Request().Context())
	permissions, _ := dCtx.GetPermissions(c.Request().Context())

	return writeJSON(c, http.StatusOK, echodt.CodeSuccess, "ok", echo4.Map{
		"loginId":     loginID,
		"roles":       roles,
		"permissions": permissions,
	})
}

// handleAdmin returns admin data handleAdmin 返回管理员数据
func handleAdmin(c echo4.Context) error {
	return writeJSON(c, http.StatusOK, echodt.CodeSuccess, "ok", echo4.Map{"scope": "admin"})
}

// handleArticles returns protected article data handleArticles 返回受保护的文章数据
func handleArticles(c echo4.Context) error {
	return writeJSON(c, http.StatusOK, echodt.CodeSuccess, "ok", []string{"article-a", "article-b"})
}

// handleLogout logs out current token handleLogout 注销当前 Token
func handleLogout(c echo4.Context) error {
	dCtx, ok := echodt.GetDTokenContext(c)
	if !ok {
		return writeJSON(c, http.StatusUnauthorized, echodt.CodeNotLogin, "not logged in", nil)
	}

	if err := dCtx.Logout(c.Request().Context()); err != nil {
		return writeJSON(c, http.StatusInternalServerError, echodt.CodeServerError, err.Error(), nil)
	}

	return writeJSON(c, http.StatusOK, echodt.CodeSuccess, "ok", nil)
}

// writeJSON writes a unified JSON response writeJSON 写入统一 JSON 响应
func writeJSON(c echo4.Context, httpStatus int, code int, message string, data interface{}) error {
	return c.JSON(httpStatus, Response{
		Code:    code,
		Message: message,
		Data:    data,
	})
}
