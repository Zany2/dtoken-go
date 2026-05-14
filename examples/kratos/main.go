package main

import (
	"context"
	"net/http"
	"time"

	kratosdt "github.com/Zany2/dtoken-go/integrations/kratos"
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/middleware"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
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
	initDToken()

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

	app := kratos.New(
		kratos.Name("dtoken-kratos-example"),
		kratos.Server(srv),
	)

	if err := app.Run(); err != nil {
		panic(err)
	}
}

// initDToken initializes integration manager initDToken 初始化集成管理器
func initDToken() {
	mgr, err := kratosdt.NewBuilder().
		Timeout(int64((2 * time.Hour).Seconds())).
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
		return err
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
		return writeJSON(httpCtx, http.StatusInternalServerError, kratosdt.CodeServerError, err.Error(), nil)
	}

	// Seed demo authorization data 初始化示例权限数据
	_ = kratosdt.AddRoles(ctx, req.Username, []string{"admin"})
	_ = kratosdt.AddPermissions(ctx, req.Username, []string{"article:read"})

	return writeJSON(httpCtx, http.StatusOK, kratosdt.CodeSuccess, "ok", map[string]interface{}{"token": token})
}

// handleMe returns current login information handleMe 返回当前登录信息
func handleMe(ctx context.Context, httpCtx khttp.Context) error {
	dCtx, ok := kratosdt.GetDTokenContext(ctx)
	if !ok {
		return writeJSON(httpCtx, http.StatusUnauthorized, kratosdt.CodeNotLogin, "not logged in", nil)
	}

	loginID, err := dCtx.GetLoginID(ctx)
	if err != nil {
		return writeJSON(httpCtx, http.StatusUnauthorized, kratosdt.CodeNotLogin, err.Error(), nil)
	}

	roles, _ := dCtx.GetRoles(ctx)
	permissions, _ := dCtx.GetPermissions(ctx)

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

	if err := dCtx.Logout(ctx); err != nil {
		return writeJSON(httpCtx, http.StatusInternalServerError, kratosdt.CodeServerError, err.Error(), nil)
	}

	return writeJSON(httpCtx, http.StatusOK, kratosdt.CodeSuccess, "ok", nil)
}

// writeJSON writes a unified JSON response writeJSON 写入统一 JSON 响应
func writeJSON(ctx khttp.Context, httpStatus int, code int, message string, data interface{}) error {
	return ctx.JSON(httpStatus, Response{
		Code:    code,
		Message: message,
		Data:    data,
	})
}
