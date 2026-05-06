// @Author daixk 2026/2/2 17:20:00
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/Zany2/dtoken-go/com/storage/redis"
	kratosdt "github.com/Zany2/dtoken-go/integrations/kratos"
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/middleware"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

// Response defines response body Response 定义响应体
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// LoginRequest defines login body LoginRequest 定义登录参数
type LoginRequest struct {
	Username string `json:"username" form:"username"`
	Password string `json:"password" form:"password"`
}

// UserRequest defines user body UserRequest 定义用户参数
type UserRequest struct {
	Username string `json:"username" form:"username"`
}

func main() {
	ctx := context.Background()

	// initManager initializes manager initManager 初始化管理器
	initManager(ctx)

	// Create server Create server 创建服务
	httpSrv := khttp.NewServer(
		khttp.Address(":8080"),
		khttp.Middleware(kratosdt.RegisterDTokenContextMiddleware()),
	)

	// Register public routes Register public routes 注册公开路由
	api := httpSrv.Route("/api")
	api.POST("/login", wrapHandler(handleLogin))
	api.GET("/public", wrapHandler(handlePublic))

	// Register user routes Register user routes 注册用户路由
	user := api.Group("/user")
	user.GET("/info", wrapHandler(handleUserInfo, kratosdt.AuthMiddleware()))
	user.POST("/logout", wrapHandler(handleLogout, kratosdt.AuthMiddleware()))

	// Register admin routes Register admin routes 注册管理路由
	admin := api.Group("/admin")
	admin.GET("/users", wrapHandler(handleAdminUsers, kratosdt.RoleMiddleware([]string{"admin"})))
	admin.POST("/disable", wrapHandler(handleDisableUser, kratosdt.RoleMiddleware([]string{"admin"})))
	admin.POST("/enable", wrapHandler(handleEnableUser, kratosdt.RoleMiddleware([]string{"admin"})))

	// Register permission routes Register permission routes 注册权限路由
	resource := api.Group("/resource")
	resource.GET("/list", wrapHandler(handleResourceList, kratosdt.PermissionMiddleware([]string{"resource:read"})))

	// Register annotation routes Register annotation routes 注册注解路由
	annotation := api.Group("/annotation")
	annotation.GET("/profile", wrapHandler(handleProfile, kratosdt.CheckLoginMiddleware(nil)))
	annotation.GET("/admin-data", wrapHandler(handleAdminData, kratosdt.CheckRoleMiddleware([]string{"admin"}, nil)))
	annotation.GET("/sensitive", wrapHandler(handleSensitiveData, kratosdt.CheckPermissionMiddleware([]string{"data:read"}, nil)))
	annotation.GET("/super", wrapHandler(handleSuperData, kratosdt.CheckAllMiddleware([]string{"super-admin"}, []string{"all:access"}, nil)))

	app := kratos.New(
		kratos.Name("dtoken-kratos-example"),
		kratos.Server(httpSrv),
	)

	// Print routes Print routes 打印路由
	fmt.Println("Server starting on http://localhost:8080")
	fmt.Println("Available endpoints:")
	fmt.Println("  POST /api/login")
	fmt.Println("  GET  /api/public")
	fmt.Println("  GET  /api/user/info")
	fmt.Println("  POST /api/user/logout")
	fmt.Println("  GET  /api/admin/users")
	fmt.Println("  POST /api/admin/disable")
	fmt.Println("  POST /api/admin/enable")
	fmt.Println("  GET  /api/resource/list")
	fmt.Println("  GET  /api/annotation/profile")
	fmt.Println("  GET  /api/annotation/admin-data")
	fmt.Println("  GET  /api/annotation/sensitive")
	fmt.Println("  GET  /api/annotation/super")

	if err := app.Run(); err != nil {
		panic(err)
	}
}

// initManager initializes manager initManager 初始化管理器
func initManager(ctx context.Context) {
	// Create storage Create storage 创建存储
	storage, err := redis.NewStorage("redis://:root@192.168.19.104:6379/0?dial_timeout=3&read_timeout=10s&max_retries=2")
	if err != nil {
		panic("failed to connect redis: " + err.Error())
	}

	// Build manager Build manager 构建管理器
	builder := kratosdt.NewDefaultBuilder()
	manager := builder.
		SetStorage(storage).
		Timeout(3600).
		ActiveTimeout(1800).
		MaxLoginCount(3).
		Build()

	kratosdt.SetManager(manager)

	fmt.Println("DToken manager initialized successfully with Kratos storage")
	_ = ctx
}

// wrapHandler wraps handler with middleware wrapHandler 使用中间件包装处理函数
func wrapHandler(handler func(context.Context, khttp.Context) error, mws ...middleware.Middleware) khttp.HandlerFunc {
	return func(httpCtx khttp.Context) error {
		// Chain middlewares Chain middlewares 串联中间件
		chained := middleware.Chain(mws...)(func(execCtx context.Context, req any) (any, error) {
			return nil, handler(execCtx, httpCtx)
		})

		// Reuse context pipeline Reuse context pipeline 复用上下文链路
		h := httpCtx.Middleware(chained)
		_, err := h(httpCtx, nil)
		return err
	}
}

// writeJSON writes response body writeJSON 写入响应体
func writeJSON(ctx khttp.Context, code int, message string, data interface{}) error {
	return ctx.JSON(200, Response{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

// getDTokenContext gets dtoken context getDTokenContext 获取 DToken 上下文
func getDTokenContext(ctx context.Context) (*kratosdt.DTokenContext, error) {
	dCtx, ok := kratosdt.GetDTokenContextByCtx(ctx)
	if !ok {
		return nil, kratosdt.ErrNotLogin
	}
	return dCtx, nil
}

// handleLogin handles login handleLogin 处理登录
func handleLogin(ctx context.Context, httpCtx khttp.Context) error {
	var req LoginRequest
	if err := httpCtx.Bind(&req); err != nil {
		return writeJSON(httpCtx, kratosdt.CodeBadRequest, "Invalid request", nil)
	}

	if req.Username == "" || req.Password == "" {
		return writeJSON(httpCtx, kratosdt.CodeBadRequest, "Username and password are required", nil)
	}

	if req.Username != "admin" || req.Password != "123456" {
		return writeJSON(httpCtx, kratosdt.CodeNotLogin, "Invalid username or password", nil)
	}

	token, err := kratosdt.Login(ctx, req.Username)
	if err != nil {
		return writeJSON(httpCtx, kratosdt.CodeServerError, fmt.Sprintf("Login failed: %v", err), nil)
	}

	// Seed auth data Seed auth data 初始化权限数据
	_ = kratosdt.AddRoles(ctx, req.Username, []string{"admin", "super-admin"})
	_ = kratosdt.AddPermissions(ctx, req.Username, []string{"resource:read", "resource:write", "data:read", "all:access"})

	return writeJSON(httpCtx, kratosdt.CodeSuccess, "Login successful", map[string]interface{}{
		"token":    token,
		"username": req.Username,
	})
}

// handlePublic handles public api handlePublic 处理公开接口
func handlePublic(ctx context.Context, httpCtx khttp.Context) error {
	_ = ctx
	return writeJSON(httpCtx, kratosdt.CodeSuccess, "This is a public endpoint", "Anyone can access this")
}

// handleUserInfo handles user info handleUserInfo 处理用户信息
func handleUserInfo(ctx context.Context, httpCtx khttp.Context) error {
	dCtx, err := getDTokenContext(ctx)
	if err != nil {
		return writeJSON(httpCtx, kratosdt.CodeNotLogin, "Not logged in", nil)
	}

	loginID, err := dCtx.GetLoginID(ctx)
	if err != nil {
		return writeJSON(httpCtx, kratosdt.CodeNotLogin, fmt.Sprintf("Failed to get login ID: %v", err), nil)
	}

	tokenInfo, err := dCtx.GetTokenInfo(ctx)
	if err != nil {
		return writeJSON(httpCtx, kratosdt.CodeServerError, fmt.Sprintf("Failed to get token info: %v", err), nil)
	}

	roles, _ := dCtx.GetRoles(ctx)
	permissions, _ := dCtx.GetPermissions(ctx)

	return writeJSON(httpCtx, kratosdt.CodeSuccess, "User info retrieved successfully", map[string]interface{}{
		"loginID":     loginID,
		"tokenValue":  dCtx.GetTokenValue(),
		"device":      tokenInfo.Device,
		"createTime":  time.Unix(tokenInfo.CreateTime, 0).Format("2006-01-02 15:04:05"),
		"roles":       roles,
		"permissions": permissions,
	})
}

// handleLogout handles logout handleLogout 处理退出登录
func handleLogout(ctx context.Context, httpCtx khttp.Context) error {
	dCtx, err := getDTokenContext(ctx)
	if err != nil {
		return writeJSON(httpCtx, kratosdt.CodeNotLogin, "Not logged in", nil)
	}

	if err = dCtx.Logout(ctx); err != nil {
		return writeJSON(httpCtx, kratosdt.CodeServerError, fmt.Sprintf("Logout failed: %v", err), nil)
	}

	return writeJSON(httpCtx, kratosdt.CodeSuccess, "Logout successful", nil)
}

// handleAdminUsers handles admin users handleAdminUsers 处理管理员列表
func handleAdminUsers(ctx context.Context, httpCtx khttp.Context) error {
	dCtx, err := getDTokenContext(ctx)
	if err != nil {
		return writeJSON(httpCtx, kratosdt.CodeNotLogin, "Not logged in", nil)
	}

	loginID, _ := dCtx.GetLoginID(ctx)
	return writeJSON(httpCtx, kratosdt.CodeSuccess, "Admin users list", []map[string]interface{}{
		{"id": 1, "username": "admin", "role": "admin"},
		{"id": 2, "username": "user1", "role": "user"},
		{"id": 3, "username": loginID, "role": "current-admin"},
	})
}

// handleDisableUser handles disable user handleDisableUser 处理封禁用户
func handleDisableUser(ctx context.Context, httpCtx khttp.Context) error {
	var req UserRequest
	if err := httpCtx.Bind(&req); err != nil || req.Username == "" {
		return writeJSON(httpCtx, kratosdt.CodeBadRequest, "Username is required", nil)
	}

	if err := kratosdt.Disable(ctx, req.Username, time.Hour, "Violated terms of service"); err != nil {
		return writeJSON(httpCtx, kratosdt.CodeServerError, fmt.Sprintf("Failed to disable user: %v", err), nil)
	}

	return writeJSON(httpCtx, kratosdt.CodeSuccess, fmt.Sprintf("User %s has been disabled for 1 hour", req.Username), nil)
}

// handleEnableUser handles enable user handleEnableUser 处理解封用户
func handleEnableUser(ctx context.Context, httpCtx khttp.Context) error {
	var req UserRequest
	if err := httpCtx.Bind(&req); err != nil || req.Username == "" {
		return writeJSON(httpCtx, kratosdt.CodeBadRequest, "Username is required", nil)
	}

	if err := kratosdt.Untie(ctx, req.Username); err != nil {
		return writeJSON(httpCtx, kratosdt.CodeServerError, fmt.Sprintf("Failed to enable user: %v", err), nil)
	}

	return writeJSON(httpCtx, kratosdt.CodeSuccess, fmt.Sprintf("User %s has been enabled", req.Username), nil)
}

// handleResourceList handles resource list handleResourceList 处理资源列表
func handleResourceList(ctx context.Context, httpCtx khttp.Context) error {
	dCtx, err := getDTokenContext(ctx)
	if err != nil {
		return writeJSON(httpCtx, kratosdt.CodeNotLogin, "Not logged in", nil)
	}

	loginID, _ := dCtx.GetLoginID(ctx)
	return writeJSON(httpCtx, kratosdt.CodeSuccess, "Resource list", []map[string]interface{}{
		{"id": 1, "name": "resource-a", "owner": loginID},
		{"id": 2, "name": "resource-b", "owner": "system"},
	})
}

// handleProfile handles profile api handleProfile 处理个人信息
func handleProfile(ctx context.Context, httpCtx khttp.Context) error {
	dCtx, err := getDTokenContext(ctx)
	if err != nil {
		return writeJSON(httpCtx, kratosdt.CodeNotLogin, "Not logged in", nil)
	}

	loginID, _ := dCtx.GetLoginID(ctx)
	return writeJSON(httpCtx, kratosdt.CodeSuccess, "Profile data", map[string]interface{}{
		"loginID": loginID,
		"token":   dCtx.GetTokenValue(),
	})
}

// handleAdminData handles admin data handleAdminData 处理管理员数据
func handleAdminData(ctx context.Context, httpCtx khttp.Context) error {
	_ = ctx
	return writeJSON(httpCtx, kratosdt.CodeSuccess, "Admin data", map[string]interface{}{
		"dashboard": "kratos-admin",
		"status":    "ok",
	})
}

// handleSensitiveData handles sensitive data handleSensitiveData 处理敏感数据
func handleSensitiveData(ctx context.Context, httpCtx khttp.Context) error {
	_ = ctx
	return writeJSON(httpCtx, kratosdt.CodeSuccess, "Sensitive data", map[string]interface{}{
		"scope":   "data:read",
		"records": 3,
	})
}

// handleSuperData handles super data handleSuperData 处理超级权限数据
func handleSuperData(ctx context.Context, httpCtx khttp.Context) error {
	_ = ctx
	return writeJSON(httpCtx, kratosdt.CodeSuccess, "Super admin data", map[string]interface{}{
		"system": "kratos",
		"level":  "super-admin",
	})
}
