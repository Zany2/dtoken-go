// @Author daixk 2026/2/2 15:30:00
package main

import (
	"context"
	"fmt"
	gindt "github.com/Zany2/dtoken-go/integrations/gin"
	"time"

	"github.com/Zany2/dtoken-go/com/storage/redis"
	"github.com/gin-gonic/gin"
)

func main() {
	ctx := context.Background()

	// Initialize DToken Manager
	// 初始化 DToken 管理器
	initManager(ctx)

	// Create Gin router
	// 创建 Gin 路由器
	r := gin.Default()

	// Register middleware
	// 注册中间件
	r.Use(gindt.RegisterDTokenContextMiddleware(ctx))

	// Public routes (no authentication required)
	// 公开路由（无需认证）
	api := r.Group("/api")
	{
		api.POST("/login", handleLogin)
		api.GET("/public", handlePublic)
	}

	// Protected routes (authentication required)
	// 受保护路由（需要认证）
	user := r.Group("/api/user")
	user.Use(gindt.AuthMiddleware(ctx))
	{
		user.GET("/info", handleUserInfo)
		user.POST("/logout", handleLogout)
	}

	// Admin routes (require admin role)
	// 管理员路由（需要管理员角色）
	admin := r.Group("/api/admin")
	admin.Use(gindt.RoleMiddleware(ctx, []string{"admin"}))
	{
		admin.GET("/users", handleAdminUsers)
		admin.POST("/disable", handleDisableUser)
		admin.POST("/enable", handleEnableUser)
	}

	// Permission-based routes
	// 基于权限的路由
	resource := r.Group("/api/resource")
	resource.Use(gindt.PermissionMiddleware(ctx, []string{"resource:read"}))
	{
		resource.GET("/list", handleResourceList)
	}

	// Annotation-based routes
	// 基于注解的路由
	annotation := r.Group("/api/annotation")
	{
		// Check login only
		// 仅检查登录
		annotation.GET("/profile", gindt.CheckLoginMiddleware(ctx, handleProfile, handleAuthFail))

		// Check role
		// 检查角色
		annotation.GET("/admin-data", gindt.CheckRoleMiddleware(ctx, []string{"admin"}, handleAdminData, handleAuthFail))

		// Check permission
		// 检查权限
		annotation.GET("/sensitive", gindt.CheckPermissionMiddleware(ctx, []string{"data:read"}, handleSensitiveData, handleAuthFail))

		// Check all (login + role + permission)
		// 全面检查（登录 + 角色 + 权限）
		annotation.GET("/super", gindt.CheckAllMiddleware(ctx, []string{"super-admin"}, []string{"all:access"}, handleSuperData, handleAuthFail))
	}

	// Start server
	// 启动服务器
	fmt.Println("Server starting on http://localhost:8080")
	fmt.Println("服务器启动在 http://localhost:8080")
	fmt.Println("\nAvailable endpoints:")
	fmt.Println("可用的接口:")
	fmt.Println("  POST /api/login          - Login (username: admin, password: 123456)")
	fmt.Println("  GET  /api/public         - Public endpoint")
	fmt.Println("  GET  /api/user/info      - Get user info (requires login)")
	fmt.Println("  POST /api/user/logout    - Logout")
	fmt.Println("  GET  /api/admin/users    - Admin users list (requires admin role)")
	fmt.Println("  POST /api/admin/disable  - Disable user (requires admin role)")
	fmt.Println("  POST /api/admin/enable   - Enable user (requires admin role)")
	fmt.Println("  GET  /api/resource/list  - Resource list (requires resource:read permission)")
	fmt.Println("  GET  /api/annotation/*   - Annotation-based routes")

	r.Run(":8080")
}

// initManager initializes the DToken manager
// initManager 初始化 DToken 管理器
func initManager(ctx context.Context) {
	// 使用 Redis 存储
	// Redis URL 格式: redis://[username]:[password]@[host]:[port]/[database]
	storage, err := redis.NewStorage("redis://:root@192.168.19.104:6379/0?dial_timeout=3&read_timeout=10s&max_retries=2")
	if err != nil {
		panic("Failed to connect to Redis: " + err.Error())
	}
	// Create builder and configure
	// 创建构建器并配置
	builder := gindt.NewDefaultBuilder()
	mgr := builder.
		SetStorage(storage).
		Timeout(3600).       // 1 hour
		ActiveTimeout(1800). // 30 minutes
		MaxLoginCount(3).
		Build()

	// Set manager
	// 设置管理器
	gindt.SetManager(mgr)

	fmt.Println("✓ DToken Manager initialized successfully with Gin Redis storage")
	fmt.Println("✓ DToken 管理器初始化成功（使用 Gin Redis 存储）")
}

// ============================================================================
// Handler Functions - 处理器函数
// ============================================================================

// handleLogin handles user login
// handleLogin 处理用户登录
func handleLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username" form:"username"`
		Password string `json:"password" form:"password"`
	}

	if err := c.ShouldBind(&req); err != nil {
		c.JSON(200, gin.H{
			"code":    gindt.CodeBadRequest,
			"message": "Invalid request",
			"data":    nil,
		})
		return
	}

	// Simple authentication (in production, verify against database)
	// 简单认证（生产环境中应该验证数据库）
	if req.Username == "" || req.Password == "" {
		c.JSON(200, gin.H{
			"code":    gindt.CodeBadRequest,
			"message": "Username and password are required",
			"data":    nil,
		})
		return
	}

	// Simulate user authentication
	// 模拟用户认证
	if req.Username != "admin" || req.Password != "123456" {
		c.JSON(200, gin.H{
			"code":    gindt.CodeNotLogin,
			"message": "Invalid username or password",
			"data":    nil,
		})
		return
	}

	// Login and get token
	// 登录并获取 token
	token, err := gindt.Login(c.Request.Context(), req.Username)
	if err != nil {
		c.JSON(200, gin.H{
			"code":    gindt.CodeServerError,
			"message": fmt.Sprintf("Login failed: %v", err),
			"data":    nil,
		})
		return
	}

	// Add roles and permissions for admin user
	// 为管理员用户添加角色和权限
	_ = gindt.AddRoles(c.Request.Context(), req.Username, []string{"admin", "super-admin"})
	_ = gindt.AddPermissions(c.Request.Context(), req.Username, []string{"resource:read", "resource:write", "data:read", "all:access"})

	c.JSON(200, gin.H{
		"code":    gindt.CodeSuccess,
		"message": "Login successful",
		"data": gin.H{
			"token":    token,
			"username": req.Username,
		},
	})
}

// handlePublic handles public endpoint
// handlePublic 处理公开接口
func handlePublic(c *gin.Context) {
	c.JSON(200, gin.H{
		"code":    gindt.CodeSuccess,
		"message": "This is a public endpoint",
		"data":    "Anyone can access this",
	})
}

// handleUserInfo handles user info request
// handleUserInfo 处理用户信息请求
func handleUserInfo(c *gin.Context) {
	// Get DToken context
	// 获取 DToken 上下文
	dCtx, ok := gindt.GetDTokenContext(c)
	if !ok {
		c.JSON(200, gin.H{
			"code":    gindt.CodeServerError,
			"message": "Failed to get DToken context",
			"data":    nil,
		})
		return
	}

	// Use convenience methods - 使用便捷方法
	loginID, err := dCtx.GetLoginID(c.Request.Context())
	if err != nil {
		c.JSON(200, gin.H{
			"code":    gindt.CodeNotLogin,
			"message": fmt.Sprintf("Failed to get login ID: %v", err),
			"data":    nil,
		})
		return
	}

	// Get token info
	// 获取 token 信息
	tokenInfo, err := dCtx.GetTokenInfo(c.Request.Context())
	if err != nil {
		c.JSON(200, gin.H{
			"code":    gindt.CodeServerError,
			"message": fmt.Sprintf("Failed to get token info: %v", err),
			"data":    nil,
		})
		return
	}

	// Get roles and permissions using convenience methods
	// 使用便捷方法获取角色和权限
	roles, _ := dCtx.GetRoles(c.Request.Context())
	permissions, _ := dCtx.GetPermissions(c.Request.Context())

	c.JSON(200, gin.H{
		"code":    gindt.CodeSuccess,
		"message": "User info retrieved successfully",
		"data": gin.H{
			"loginID":     loginID,
			"tokenValue":  dCtx.GetTokenValue(),
			"device":      tokenInfo.Device,
			"createTime":  time.Unix(tokenInfo.CreateTime, 0).Format("2006-01-02 15:04:05"),
			"roles":       roles,
			"permissions": permissions,
		},
	})
}

// handleLogout handles user logout
// handleLogout 处理用户登出
func handleLogout(c *gin.Context) {
	// Get DToken context
	// 获取 DToken 上下文
	dCtx, ok := gindt.GetDTokenContext(c)
	if !ok {
		c.JSON(200, gin.H{
			"code":    gindt.CodeNotLogin,
			"message": "Not logged in",
			"data":    nil,
		})
		return
	}

	// Use convenience method - 使用便捷方法
	err := dCtx.Logout(c.Request.Context())
	if err != nil {
		c.JSON(200, gin.H{
			"code":    gindt.CodeServerError,
			"message": fmt.Sprintf("Logout failed: %v", err),
			"data":    nil,
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    gindt.CodeSuccess,
		"message": "Logout successful",
		"data":    nil,
	})
}

// handleAdminUsers handles admin users list
// handleAdminUsers 处理管理员用户列表
func handleAdminUsers(c *gin.Context) {
	c.JSON(200, gin.H{
		"code":    gindt.CodeSuccess,
		"message": "Admin users list",
		"data": []gin.H{
			{"id": 1, "username": "admin", "role": "admin"},
			{"id": 2, "username": "user1", "role": "user"},
		},
	})
}

// handleDisableUser handles disabling a user
// handleDisableUser 处理封禁用户
func handleDisableUser(c *gin.Context) {
	var req struct {
		Username string `json:"username" form:"username"`
	}

	if err := c.ShouldBind(&req); err != nil || req.Username == "" {
		c.JSON(200, gin.H{
			"code":    gindt.CodeBadRequest,
			"message": "Username is required",
			"data":    nil,
		})
		return
	}

	// Disable user for 1 hour
	// 封禁用户 1 小时
	err := gindt.Disable(c.Request.Context(), req.Username, time.Hour, "Violated terms of service")
	if err != nil {
		c.JSON(200, gin.H{
			"code":    gindt.CodeServerError,
			"message": fmt.Sprintf("Failed to disable user: %v", err),
			"data":    nil,
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    gindt.CodeSuccess,
		"message": fmt.Sprintf("User %s has been disabled for 1 hour", req.Username),
		"data":    nil,
	})
}

// handleEnableUser handles enabling a user
// handleEnableUser 处理解封用户
func handleEnableUser(c *gin.Context) {
	var req struct {
		Username string `json:"username" form:"username"`
	}

	if err := c.ShouldBind(&req); err != nil || req.Username == "" {
		c.JSON(200, gin.H{
			"code":    gindt.CodeBadRequest,
			"message": "Username is required",
			"data":    nil,
		})
		return
	}

	err := gindt.Untie(c.Request.Context(), req.Username)
	if err != nil {
		c.JSON(200, gin.H{
			"code":    gindt.CodeServerError,
			"message": fmt.Sprintf("Failed to enable user: %v", err),
			"data":    nil,
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    gindt.CodeSuccess,
		"message": fmt.Sprintf("User %s has been enabled", req.Username),
		"data":    nil,
	})
}

// handleResourceList handles resource list
// handleResourceList 处理资源列表
func handleResourceList(c *gin.Context) {
	c.JSON(200, gin.H{
		"code":    gindt.CodeSuccess,
		"message": "Resource list",
		"data": []gin.H{
			{"id": 1, "name": "Resource 1", "type": "document"},
			{"id": 2, "name": "Resource 2", "type": "image"},
		},
	})
}

// handleProfile handles profile request
// handleProfile 处理个人资料请求
func handleProfile(c *gin.Context) {
	dCtx, _ := gindt.GetDTokenContext(c)
	// Use convenience method - 使用便捷方法
	loginID, _ := dCtx.GetLoginID(c.Request.Context())

	c.JSON(200, gin.H{
		"code":    gindt.CodeSuccess,
		"message": "Profile data",
		"data": gin.H{
			"username": loginID,
			"email":    loginID + "@example.com",
		},
	})
}

// handleAdminData handles admin data request
// handleAdminData 处理管理员数据请求
func handleAdminData(c *gin.Context) {
	c.JSON(200, gin.H{
		"code":    gindt.CodeSuccess,
		"message": "Admin data",
		"data":    "This is admin-only data",
	})
}

// handleSensitiveData handles sensitive data request
// handleSensitiveData 处理敏感数据请求
func handleSensitiveData(c *gin.Context) {
	c.JSON(200, gin.H{
		"code":    gindt.CodeSuccess,
		"message": "Sensitive data",
		"data":    "This is sensitive data requiring data:read permission",
	})
}

// handleSuperData handles super admin data request
// handleSuperData 处理超级管理员数据请求
func handleSuperData(c *gin.Context) {
	c.JSON(200, gin.H{
		"code":    gindt.CodeSuccess,
		"message": "Super admin data",
		"data":    "This requires super-admin role and all:access permission",
	})
}

// handleAuthFail handles authentication failure
// handleAuthFail 处理认证失败
func handleAuthFail(c *gin.Context, err error) {
	var code int
	var message string

	switch err {
	case gindt.ErrNotLogin:
		code = gindt.CodeNotLogin
		message = "Not logged in"
	case gindt.ErrPermissionDenied:
		code = gindt.CodePermissionDenied
		message = "Permission denied"
	case gindt.ErrRoleDenied:
		code = gindt.CodePermissionDenied
		message = "Role denied"
	case gindt.ErrAccountDisabled:
		code = gindt.CodeAccountDisabled
		message = "Account disabled"
	default:
		code = gindt.CodeServerError
		message = err.Error()
	}

	c.JSON(200, gin.H{
		"code":    code,
		"message": message,
		"data":    nil,
	})
}
