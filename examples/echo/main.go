// @Author daixk 2026/2/2 16:40:00
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/Zany2/dtoken-go/com/storage/redis"
	echodt "github.com/Zany2/dtoken-go/integrations/echo"
	echo4 "github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Response defines unified response body Response 定义统一响应体
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// LoginRequest defines login payload LoginRequest 定义登录参数
type LoginRequest struct {
	Username string `json:"username" form:"username"`
	Password string `json:"password" form:"password"`
}

// UserRequest defines username payload UserRequest 定义用户名参数
type UserRequest struct {
	Username string `json:"username" form:"username"`
}

func main() {
	ctx := context.Background()

	// initManager initializes manager initManager 初始化管理器
	initManager(ctx)

	// Create echo instance Create echo instance 创建 Echo 实例
	e := echo4.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(echodt.RegisterDTokenContextMiddleware(ctx))

	// Register public routes Register public routes 注册公开路由
	api := e.Group("/api")
	api.POST("/login", handleLogin)
	api.GET("/public", handlePublic)

	// Register user routes Register user routes 注册用户路由
	user := e.Group("/api/user")
	user.Use(echodt.AuthMiddleware(ctx))
	user.GET("/info", handleUserInfo)
	user.POST("/logout", handleLogout)

	// Register admin routes Register admin routes 注册管理路由
	admin := e.Group("/api/admin")
	admin.Use(echodt.RoleMiddleware(ctx, []string{"admin"}))
	admin.GET("/users", handleAdminUsers)
	admin.POST("/disable", handleDisableUser)
	admin.POST("/enable", handleEnableUser)

	// Register resource routes Register resource routes 注册权限路由
	resource := e.Group("/api/resource")
	resource.Use(echodt.PermissionMiddleware(ctx, []string{"resource:read"}))
	resource.GET("/list", handleResourceList)

	// Register annotation routes Register annotation routes 注册注解路由
	annotation := e.Group("/api/annotation")
	annotation.GET("/profile", echodt.CheckLoginMiddleware(ctx, handleProfile, handleAuthFail))
	annotation.GET("/admin-data", echodt.CheckRoleMiddleware(ctx, []string{"admin"}, handleAdminData, handleAuthFail))
	annotation.GET("/sensitive", echodt.CheckPermissionMiddleware(ctx, []string{"data:read"}, handleSensitiveData, handleAuthFail))
	annotation.GET("/super", echodt.CheckAllMiddleware(ctx, []string{"super-admin"}, []string{"all:access"}, handleSuperData, handleAuthFail))

	// Print routes Print routes 打印路由信息
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
	fmt.Println("  GET  /api/annotation/*")

	if err := e.Start(":8080"); err != nil {
		panic(err)
	}
}

// initManager initializes manager initManager 初始化管理器
func initManager(ctx context.Context) {
	// Create redis storage Create redis storage 创建 Redis 存储
	storage, err := redis.NewStorage("redis://:root@192.168.19.104:6379/0?dial_timeout=3&read_timeout=10s&max_retries=2")
	if err != nil {
		panic("failed to connect redis: " + err.Error())
	}

	// Build manager Build manager 构建管理器
	builder := echodt.NewDefaultBuilder()
	manager := builder.
		SetStorage(storage).
		Timeout(3600).
		ActiveTimeout(1800).
		MaxLoginCount(3).
		Build()

	echodt.SetManager(manager)

	fmt.Println("DToken manager initialized successfully with Echo storage")
	_ = ctx
}

// handleLogin handles login handleLogin 处理登录
func handleLogin(c echo4.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(200, Response{
			Code:    echodt.CodeBadRequest,
			Message: "Invalid request",
			Data:    nil,
		})
	}

	if req.Username == "" || req.Password == "" {
		return c.JSON(200, Response{
			Code:    echodt.CodeBadRequest,
			Message: "Username and password are required",
			Data:    nil,
		})
	}

	if req.Username != "admin" || req.Password != "123456" {
		return c.JSON(200, Response{
			Code:    echodt.CodeNotLogin,
			Message: "Invalid username or password",
			Data:    nil,
		})
	}

	// Seed auth data after login Seed auth data after login 登录后写入示例权限数据
	token, err := echodt.Login(c.Request().Context(), req.Username)
	if err != nil {
		return c.JSON(200, Response{
			Code:    echodt.CodeServerError,
			Message: fmt.Sprintf("Login failed: %v", err),
			Data:    nil,
		})
	}
	_ = echodt.AddRoles(c.Request().Context(), req.Username, []string{"admin", "super-admin"})
	_ = echodt.AddPermissions(c.Request().Context(), req.Username, []string{"resource:read", "resource:write", "data:read", "all:access"})

	return c.JSON(200, Response{
		Code:    echodt.CodeSuccess,
		Message: "Login successful",
		Data: map[string]interface{}{
			"token":    token,
			"username": req.Username,
		},
	})
}

// handlePublic handles public endpoint handlePublic 处理公开接口
func handlePublic(c echo4.Context) error {
	return c.JSON(200, Response{
		Code:    echodt.CodeSuccess,
		Message: "This is a public endpoint",
		Data:    "Anyone can access this",
	})
}

// handleUserInfo handles user info handleUserInfo 处理用户信息
func handleUserInfo(c echo4.Context) error {
	dCtx, ok := echodt.GetDTokenContext(c)
	if !ok {
		return c.JSON(200, Response{
			Code:    echodt.CodeServerError,
			Message: "Failed to get DToken context",
			Data:    nil,
		})
	}

	loginID, err := dCtx.GetLoginID(c.Request().Context())
	if err != nil {
		return c.JSON(200, Response{
			Code:    echodt.CodeNotLogin,
			Message: fmt.Sprintf("Failed to get login ID: %v", err),
			Data:    nil,
		})
	}

	tokenInfo, err := dCtx.GetTokenInfo(c.Request().Context())
	if err != nil {
		return c.JSON(200, Response{
			Code:    echodt.CodeServerError,
			Message: fmt.Sprintf("Failed to get token info: %v", err),
			Data:    nil,
		})
	}

	roles, _ := dCtx.GetRoles(c.Request().Context())
	permissions, _ := dCtx.GetPermissions(c.Request().Context())

	return c.JSON(200, Response{
		Code:    echodt.CodeSuccess,
		Message: "User info retrieved successfully",
		Data: map[string]interface{}{
			"loginID":     loginID,
			"tokenValue":  dCtx.GetTokenValue(),
			"device":      tokenInfo.Device,
			"createTime":  time.Unix(tokenInfo.CreateTime, 0).Format("2006-01-02 15:04:05"),
			"roles":       roles,
			"permissions": permissions,
		},
	})
}

// handleLogout handles logout handleLogout 处理退出登录
func handleLogout(c echo4.Context) error {
	dCtx, ok := echodt.GetDTokenContext(c)
	if !ok {
		return c.JSON(200, Response{
			Code:    echodt.CodeNotLogin,
			Message: "Not logged in",
			Data:    nil,
		})
	}

	if err := dCtx.Logout(c.Request().Context()); err != nil {
		return c.JSON(200, Response{
			Code:    echodt.CodeServerError,
			Message: fmt.Sprintf("Logout failed: %v", err),
			Data:    nil,
		})
	}

	return c.JSON(200, Response{
		Code:    echodt.CodeSuccess,
		Message: "Logout successful",
		Data:    nil,
	})
}

// handleAdminUsers handles admin users handleAdminUsers 处理管理用户列表
func handleAdminUsers(c echo4.Context) error {
	return c.JSON(200, Response{
		Code:    echodt.CodeSuccess,
		Message: "Admin users list",
		Data: []map[string]interface{}{
			{"id": 1, "username": "admin", "role": "admin"},
			{"id": 2, "username": "user1", "role": "user"},
		},
	})
}

// handleDisableUser handles disable handleDisableUser 处理封禁用户
func handleDisableUser(c echo4.Context) error {
	var req UserRequest
	if err := c.Bind(&req); err != nil || req.Username == "" {
		return c.JSON(200, Response{
			Code:    echodt.CodeBadRequest,
			Message: "Username is required",
			Data:    nil,
		})
	}

	if err := echodt.Disable(c.Request().Context(), req.Username, time.Hour, "Violated terms of service"); err != nil {
		return c.JSON(200, Response{
			Code:    echodt.CodeServerError,
			Message: fmt.Sprintf("Failed to disable user: %v", err),
			Data:    nil,
		})
	}

	return c.JSON(200, Response{
		Code:    echodt.CodeSuccess,
		Message: fmt.Sprintf("User %s has been disabled for 1 hour", req.Username),
		Data:    nil,
	})
}

// handleEnableUser handles enable handleEnableUser 处理解除封禁
func handleEnableUser(c echo4.Context) error {
	var req UserRequest
	if err := c.Bind(&req); err != nil || req.Username == "" {
		return c.JSON(200, Response{
			Code:    echodt.CodeBadRequest,
			Message: "Username is required",
			Data:    nil,
		})
	}

	if err := echodt.Untie(c.Request().Context(), req.Username); err != nil {
		return c.JSON(200, Response{
			Code:    echodt.CodeServerError,
			Message: fmt.Sprintf("Failed to enable user: %v", err),
			Data:    nil,
		})
	}

	return c.JSON(200, Response{
		Code:    echodt.CodeSuccess,
		Message: fmt.Sprintf("User %s has been enabled", req.Username),
		Data:    nil,
	})
}

// handleResourceList handles resource list handleResourceList 处理资源列表
func handleResourceList(c echo4.Context) error {
	return c.JSON(200, Response{
		Code:    echodt.CodeSuccess,
		Message: "Resource list",
		Data: []map[string]interface{}{
			{"id": 1, "name": "Resource 1", "type": "document"},
			{"id": 2, "name": "Resource 2", "type": "image"},
		},
	})
}

// handleProfile handles profile handleProfile 处理个人资料
func handleProfile(c echo4.Context) error {
	dCtx, _ := echodt.GetDTokenContext(c)
	loginID, _ := dCtx.GetLoginID(c.Request().Context())

	return c.JSON(200, Response{
		Code:    echodt.CodeSuccess,
		Message: "Profile data",
		Data: map[string]interface{}{
			"username": loginID,
			"email":    loginID + "@example.com",
		},
	})
}

// handleAdminData handles admin data handleAdminData 处理管理数据
func handleAdminData(c echo4.Context) error {
	return c.JSON(200, Response{
		Code:    echodt.CodeSuccess,
		Message: "Admin data",
		Data:    "This is admin-only data",
	})
}

// handleSensitiveData handles sensitive data handleSensitiveData 处理敏感数据
func handleSensitiveData(c echo4.Context) error {
	return c.JSON(200, Response{
		Code:    echodt.CodeSuccess,
		Message: "Sensitive data",
		Data:    "This is sensitive data requiring data:read permission",
	})
}

// handleSuperData handles super data handleSuperData 处理超级管理数据
func handleSuperData(c echo4.Context) error {
	return c.JSON(200, Response{
		Code:    echodt.CodeSuccess,
		Message: "Super admin data",
		Data:    "This requires super-admin role and all:access permission",
	})
}

// handleAuthFail handles auth failure handleAuthFail 处理鉴权失败
func handleAuthFail(c echo4.Context, err error) error {
	var code int
	var message string

	switch err {
	case echodt.ErrNotLogin:
		code = echodt.CodeNotLogin
		message = "Not logged in"
	case echodt.ErrPermissionDenied:
		code = echodt.CodePermissionDenied
		message = "Permission denied"
	case echodt.ErrRoleDenied:
		code = echodt.CodePermissionDenied
		message = "Role denied"
	case echodt.ErrAccountDisabled:
		code = echodt.CodeAccountDisabled
		message = "Account disabled"
	default:
		code = echodt.CodeServerError
		message = err.Error()
	}

	return c.JSON(200, Response{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}
