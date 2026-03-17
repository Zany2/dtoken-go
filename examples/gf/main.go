// @Author daixk 2026/2/2 11:46:00
package main

import (
	"context"
	"fmt"
	gfdt "github.com/Zany2/dtoken-go/integrations/gf"
	"time"

	"github.com/Zany2/dtoken-go/com/storage/redis"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gctx"
)

func main() {
	ctx := gctx.New()

	// Initialize DToken Manager 初始化 DToken 管理器
	initManager(ctx)

	// Create HTTP Server 创建 HTTP 服务器
	s := g.Server()

	// Register middleware 注册中间件
	s.Use(gfdt.RegisterDTokenContextMiddleware(ctx))

	// Public routes (no authentication required) 公开路由（无需认证）
	s.Group("/api", func(group *ghttp.RouterGroup) {
		group.POST("/login", handleLogin)
		group.GET("/public", handlePublic)
	})

	// Protected routes (authentication required) 受保护路由（需要认证）
	s.Group("/api/user", func(group *ghttp.RouterGroup) {
		// Use authentication middleware 使用认证中间件
		group.Middleware(gfdt.AuthMiddleware(ctx))

		group.GET("/info", handleUserInfo)
		group.POST("/logout", handleLogout)
	})

	// Admin routes (require admin role) 管理员路由（需要管理员角色）
	s.Group("/api/admin", func(group *ghttp.RouterGroup) {
		// Use role middleware 使用角色中间件
		group.Middleware(gfdt.RoleMiddleware(ctx, []string{"admin"}))

		group.GET("/users", handleAdminUsers)
		group.POST("/disable", handleDisableUser)
		group.POST("/enable", handleEnableUser)
	})

	// Permission-based routes 基于权限的路由
	s.Group("/api/resource", func(group *ghttp.RouterGroup) {
		// Require specific permissions 需要特定权限
		group.Middleware(gfdt.PermissionMiddleware(ctx, []string{"resource:read"}))

		group.GET("/list", handleResourceList)
	})

	// Annotation-based routes 基于注解的路由
	s.Group("/api/annotation", func(group *ghttp.RouterGroup) {
		// Check login only 仅检查登录
		group.GET("/profile", gfdt.CheckLoginMiddleware(ctx, handleProfile, handleAuthFail))

		// Check role 检查角色
		group.GET("/admin-data", gfdt.CheckRoleMiddleware(ctx, []string{"admin"}, handleAdminData, handleAuthFail))

		// Check permission 检查权限
		group.GET("/sensitive", gfdt.CheckPermissionMiddleware(ctx, []string{"data:read"}, handleSensitiveData, handleAuthFail))

		// Check all (login + role + permission) 全面检查（登录 + 角色 + 权限）
		group.GET("/super", gfdt.CheckAllMiddleware(ctx, []string{"super-admin"}, []string{"all:access"}, handleSuperData, handleAuthFail))
	})

	// Start server 启动服务器
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

	s.SetPort(8080)
	s.Run()
}

// initManager initializes the DToken manager 初始化 DToken 管理器
func initManager(ctx context.Context) {
	// Use Redis Storage And URL Format 使用 Redis 存储与 Redis URL 格式
	storage, err := redis.NewStorage("redis://:root@192.168.19.104:6379/0?dial_timeout=3&read_timeout=10s&max_retries=2")
	if err != nil {
		panic("Failed to connect to Redis: " + err.Error())
	}
	// Create builder and configure 创建构建器并配置
	builder := gfdt.NewDefaultBuilder()
	mgr := builder.
		SetStorage(storage).
		Timeout(3600).       // 1 hour
		ActiveTimeout(1800). // 30 minutes
		MaxLoginCount(3).
		Build()

	// Set manager 设置管理器
	gfdt.SetManager(mgr)

	fmt.Println("✓ DToken Manager initialized successfully with GoFrame Redis storage")
	fmt.Println("✓ DToken 管理器初始化成功（使用 GoFrame Redis 存储）")
}

// -------------------------------------------------- Handler Functions - 处理器函数 --------------------------------------------------

// handleLogin handles user login 处理用户登录
func handleLogin(r *ghttp.Request) {
	username := r.Get("username").String()
	password := r.Get("password").String()

	// Simple authentication (in production, verify against database) 简单认证（生产环境中应该验证数据库）
	if username == "" || password == "" {
		r.Response.WriteJson(g.Map{
			"code":    gfdt.CodeBadRequest,
			"message": "Username and password are required",
			"data":    nil,
		})
		return
	}

	// Simulate user authentication 模拟用户认证
	if username != "admin" || password != "123456" {
		r.Response.WriteJson(g.Map{
			"code":    gfdt.CodeNotLogin,
			"message": "Invalid username or password",
			"data":    nil,
		})
		return
	}

	// Login and get token 登录并获取 token
	token, err := gfdt.Login(r.Context(), username)
	if err != nil {
		r.Response.WriteJson(g.Map{
			"code":    gfdt.CodeServerError,
			"message": fmt.Sprintf("Login failed: %v", err),
			"data":    nil,
		})
		return
	}

	// Add roles and permissions for admin user 为管理员用户添加角色和权限
	//_ = gfdt.AddRoles(r.Context(), username, []string{"admin", "super-admin"})
	//_ = gfdt.AddPermissions(r.Context(), username, []string{"resource:read", "resource:write", "data:read", "all:access"})

	r.Response.WriteJson(g.Map{
		"code":    gfdt.CodeSuccess,
		"message": "Login successful",
		"data": g.Map{
			"token":    token,
			"username": username,
		},
	})
}

// handlePublic handles public endpoint 处理公开接口
func handlePublic(r *ghttp.Request) {
	r.Response.WriteJson(g.Map{
		"code":    gfdt.CodeSuccess,
		"message": "This is a public endpoint",
		"data":    "Anyone can access this",
	})
}

// handleUserInfo handles user info request 处理用户信息请求
func handleUserInfo(r *ghttp.Request) {
	// Get login ID from context 从上下文获取登录 ID
	loginID, err := gfdt.GetLoginIDByCtx(r.Context())
	if err != nil {
		r.Response.WriteJson(g.Map{
			"code":    gfdt.CodeNotLogin,
			"message": fmt.Sprintf("Failed to get login ID: %v", err),
			"data":    nil,
		})
		return
	}

	// Get token info 获取 token 信息
	tokenInfo, err := gfdt.GetTokenInfoByCtx(r.Context())
	if err != nil {
		r.Response.WriteJson(g.Map{
			"code":    gfdt.CodeServerError,
			"message": fmt.Sprintf("Failed to get token info: %v", err),
			"data":    nil,
		})
		return
	}

	// Get token value from context 从上下文获取 token 值
	dCtx, ok := gfdt.GetDTokenContextByCtx(r.Context())
	if !ok {
		r.Response.WriteJson(g.Map{
			"code":    gfdt.CodeServerError,
			"message": "Failed to get DToken context",
			"data":    nil,
		})
		return
	}
	tokenValue := dCtx.GetTokenValue()

	// Get roles and permissions 获取角色和权限
	roles, _ := gfdt.GetRoles(r.Context(), loginID)
	permissions, _ := gfdt.GetPermissions(r.Context(), loginID)

	r.Response.WriteJson(g.Map{
		"code":    gfdt.CodeSuccess,
		"message": "User info retrieved successfully",
		"data": g.Map{
			"loginID":     loginID,
			"tokenValue":  tokenValue,
			"device":      tokenInfo.Device,
			"createTime":  time.Unix(tokenInfo.CreateTime, 0).Format("2006-01-02 15:04:05"),
			"roles":       roles,
			"permissions": permissions,
		},
	})
}

// handleLogout handles user logout 处理用户登出
func handleLogout(r *ghttp.Request) {
	// Get token value from context 从上下文获取 token 值
	dCtx, ok := gfdt.GetDTokenContextByCtx(r.Context())
	if !ok {
		r.Response.WriteJson(g.Map{
			"code":    gfdt.CodeNotLogin,
			"message": "Not logged in",
			"data":    nil,
		})
		return
	}
	tokenValue := dCtx.GetTokenValue()

	err := gfdt.Logout(r.Context(), tokenValue)
	if err != nil {
		r.Response.WriteJson(g.Map{
			"code":    gfdt.CodeServerError,
			"message": fmt.Sprintf("Logout failed: %v", err),
			"data":    nil,
		})
		return
	}

	r.Response.WriteJson(g.Map{
		"code":    gfdt.CodeSuccess,
		"message": "Logout successful",
		"data":    nil,
	})
}

// handleAdminUsers handles admin users list 处理管理员用户列表
func handleAdminUsers(r *ghttp.Request) {
	r.Response.WriteJson(g.Map{
		"code":    gfdt.CodeSuccess,
		"message": "Admin users list",
		"data": []g.Map{
			{"id": 1, "username": "admin", "role": "admin"},
			{"id": 2, "username": "user1", "role": "user"},
		},
	})
}

// handleDisableUser handles disabling a user 处理封禁用户
func handleDisableUser(r *ghttp.Request) {
	targetUser := r.Get("username").String()
	if targetUser == "" {
		r.Response.WriteJson(g.Map{
			"code":    gfdt.CodeBadRequest,
			"message": "Username is required",
			"data":    nil,
		})
		return
	}

	// Disable user for 1 hour 封禁用户 1 小时
	err := gfdt.Disable(r.Context(), targetUser, time.Hour, "Violated terms of service")
	if err != nil {
		r.Response.WriteJson(g.Map{
			"code":    gfdt.CodeServerError,
			"message": fmt.Sprintf("Failed to disable user: %v", err),
			"data":    nil,
		})
		return
	}

	r.Response.WriteJson(g.Map{
		"code":    gfdt.CodeSuccess,
		"message": fmt.Sprintf("User %s has been disabled for 1 hour", targetUser),
		"data":    nil,
	})
}

// handleEnableUser handles enabling a user 处理解封用户
func handleEnableUser(r *ghttp.Request) {
	targetUser := r.Get("username").String()
	if targetUser == "" {
		r.Response.WriteJson(g.Map{
			"code":    gfdt.CodeBadRequest,
			"message": "Username is required",
			"data":    nil,
		})
		return
	}

	err := gfdt.Untie(r.Context(), targetUser)
	if err != nil {
		r.Response.WriteJson(g.Map{
			"code":    gfdt.CodeServerError,
			"message": fmt.Sprintf("Failed to enable user: %v", err),
			"data":    nil,
		})
		return
	}

	r.Response.WriteJson(g.Map{
		"code":    gfdt.CodeSuccess,
		"message": fmt.Sprintf("User %s has been enabled", targetUser),
		"data":    nil,
	})
}

// handleResourceList handles resource list 处理资源列表
func handleResourceList(r *ghttp.Request) {
	r.Response.WriteJson(g.Map{
		"code":    gfdt.CodeSuccess,
		"message": "Resource list",
		"data": []g.Map{
			{"id": 1, "name": "Resource 1", "type": "document"},
			{"id": 2, "name": "Resource 2", "type": "image"},
		},
	})
}

// handleProfile handles profile request 处理个人资料请求
func handleProfile(r *ghttp.Request) {
	loginID, _ := gfdt.GetLoginIDByCtx(r.Context())
	r.Response.WriteJson(g.Map{
		"code":    gfdt.CodeSuccess,
		"message": "Profile data",
		"data": g.Map{
			"username": loginID,
			"email":    loginID + "@example.com",
		},
	})
}

// handleAdminData handles admin data request 处理管理员数据请求
func handleAdminData(r *ghttp.Request) {
	r.Response.WriteJson(g.Map{
		"code":    gfdt.CodeSuccess,
		"message": "Admin data",
		"data":    "This is admin-only data",
	})
}

// handleSensitiveData handles sensitive data request 处理敏感数据请求
func handleSensitiveData(r *ghttp.Request) {
	r.Response.WriteJson(g.Map{
		"code":    gfdt.CodeSuccess,
		"message": "Sensitive data",
		"data":    "This is sensitive data requiring data:read permission",
	})
}

// handleSuperData handles super admin data request 处理超级管理员数据请求
func handleSuperData(r *ghttp.Request) {
	r.Response.WriteJson(g.Map{
		"code":    gfdt.CodeSuccess,
		"message": "Super admin data",
		"data":    "This requires super-admin role and all:access permission",
	})
}

// handleAuthFail handles authentication failure 处理认证失败
func handleAuthFail(r *ghttp.Request, err error) {
	var code int
	var message string

	switch err {
	case gfdt.ErrNotLogin:
		code = gfdt.CodeNotLogin
		message = "Not logged in"
	case gfdt.ErrPermissionDenied:
		code = gfdt.CodePermissionDenied
		message = "Permission denied"
	case gfdt.ErrRoleDenied:
		code = gfdt.CodePermissionDenied
		message = "Role denied"
	case gfdt.ErrAccountDisabled:
		code = gfdt.CodeAccountDisabled
		message = "Account disabled"
	default:
		code = gfdt.CodeServerError
		message = err.Error()
	}

	r.Response.WriteJson(g.Map{
		"code":    code,
		"message": message,
		"data":    nil,
	})
}
