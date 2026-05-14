package main

import (
	"context"
	"net/http"
	"time"

	fiberdt "github.com/Zany2/dtoken-go/integrations/fiber"
	gofiber "github.com/gofiber/fiber/v2"
)

// Response defines the example response body Response 定义示例响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// LoginRequest defines the login payload LoginRequest 定义登录请求参数
type LoginRequest struct {
	Username string `json:"username" form:"username"`
	Password string `json:"password" form:"password"`
}

func main() {
	ctx := context.Background()
	initDToken()

	app := gofiber.New()
	app.Use(fiberdt.RegisterDTokenContextMiddleware(ctx))
	app.Post("/login", handleLogin)

	auth := app.Group("")
	auth.Use(fiberdt.AuthMiddleware(ctx))
	auth.Get("/me", handleMe)
	auth.Get("/admin", fiberdt.RoleMiddleware(ctx, []string{"admin"}), handleAdmin)
	auth.Get("/articles", fiberdt.PermissionMiddleware(ctx, []string{"article:read"}), handleArticles)
	auth.Post("/logout", handleLogout)

	_ = app.Listen(":8080")
}

// initDToken initializes integration manager initDToken 初始化集成管理器
func initDToken() {
	mgr, err := fiberdt.NewBuilder().
		Timeout(int64((2 * time.Hour).Seconds())).
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

	token, err := fiberdt.Login(c.UserContext(), req.Username)
	if err != nil {
		return writeJSON(c, http.StatusInternalServerError, fiberdt.CodeServerError, err.Error(), nil)
	}

	// Seed demo authorization data 初始化示例权限数据
	_ = fiberdt.AddRoles(c.UserContext(), req.Username, []string{"admin"})
	_ = fiberdt.AddPermissions(c.UserContext(), req.Username, []string{"article:read"})

	return writeJSON(c, http.StatusOK, fiberdt.CodeSuccess, "ok", gofiber.Map{"token": token})
}

// handleMe returns current login information handleMe 返回当前登录信息
func handleMe(c *gofiber.Ctx) error {
	dCtx, ok := fiberdt.GetDTokenContext(c)
	if !ok {
		return writeJSON(c, http.StatusUnauthorized, fiberdt.CodeNotLogin, "not logged in", nil)
	}

	loginID, err := dCtx.GetLoginID(c.UserContext())
	if err != nil {
		return writeJSON(c, http.StatusUnauthorized, fiberdt.CodeNotLogin, err.Error(), nil)
	}

	roles, _ := dCtx.GetRoles(c.UserContext())
	permissions, _ := dCtx.GetPermissions(c.UserContext())

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

	if err := dCtx.Logout(c.UserContext()); err != nil {
		return writeJSON(c, http.StatusInternalServerError, fiberdt.CodeServerError, err.Error(), nil)
	}

	return writeJSON(c, http.StatusOK, fiberdt.CodeSuccess, "ok", nil)
}

// writeJSON writes a unified JSON response writeJSON 写入统一 JSON 响应
func writeJSON(c *gofiber.Ctx, httpStatus int, code int, message string, data interface{}) error {
	return c.Status(httpStatus).JSON(Response{
		Code:    code,
		Message: message,
		Data:    data,
	})
}
