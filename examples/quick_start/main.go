// @Author daixk 2026/2/2
package main

import (
	"context"
	"net/http"
	"time"

	"github.com/Zany2/dtoken-go/com/storage/redis"
	"github.com/Zany2/dtoken-go/core/builder"
	"github.com/Zany2/dtoken-go/dtoken"
	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	LoginID  string `json:"loginId" binding:"required"`
	Device   string `json:"device"`
	DeviceId string `json:"deviceId"`
}

// PermissionRequest 权限请求
type PermissionRequest struct {
	LoginID     string   `json:"loginId" binding:"required"`
	Permissions []string `json:"permissions" binding:"required"`
}

// RoleRequest 角色请求
type RoleRequest struct {
	LoginID string   `json:"loginId" binding:"required"`
	Roles   []string `json:"roles" binding:"required"`
}

// DisableRequest 封禁请求
type DisableRequest struct {
	LoginID  string `json:"loginId" binding:"required"`
	Duration int64  `json:"duration"` // 秒
	Reason   string `json:"reason"`
}

func main() {
	// 初始化 dtoken
	initDToken()

	// 创建 gin 引擎
	r := gin.Default()

	// 注册路由
	registerRoutes(r)

	// 启动服务
	if err := r.Run(":8080"); err != nil {
		panic(err)
	}
}

// initDToken 初始化 dtoken 配置
func initDToken() {
	// 使用 Redis 存储
	// Redis URL 格式: redis://[username]:[password]@[host]:[port]/[database]
	storage, err := redis.NewStorage("redis://:root@192.168.19.104:6379/0?dial_timeout=3&read_timeout=10s&max_retries=2")
	if err != nil {
		panic("Failed to connect to Redis: " + err.Error())
	}

	// 构建 manager（使用链式调用设置配置）
	mgr := builder.NewBuilder().
		Timeout(60).
		IsShare(true).
		IsPrintBanner(true).
		SetStorage(storage).
		Build()

	// 注册 manager
	dtoken.SetManager(mgr)
}

// registerRoutes 注册所有路由
func registerRoutes(r *gin.Engine) {
	// 公开接口
	public := r.Group("/api")
	{
		public.POST("/login", handleLogin)
		public.GET("/test", handleTest)
	}

	// 需要认证的接口
	auth := r.Group("/api")
	auth.Use(authMiddleware())
	{
		// Token 相关
		auth.POST("/logout", handleLogout)
		auth.POST("/logout/device", handleLogoutByDevice)
		auth.GET("/token/info", handleGetTokenInfo)
		auth.GET("/token/list", handleGetTokenList)
		auth.GET("/token/ttl", handleGetTokenTTL)
		auth.GET("/online/count", handleGetOnlineCount)

		// 踢人/顶人
		auth.POST("/kickout", handleKickout)
		auth.POST("/kickout/device", handleKickoutByDevice)
		auth.POST("/replace", handleReplace)
		auth.POST("/replace/device", handleReplaceByDevice)

		// 账号封禁
		auth.POST("/disable", handleDisable)
		auth.POST("/undisable", handleUndisable)
		auth.GET("/disable/info", handleGetDisableInfo)

		// 权限管理
		auth.POST("/permission/add", handleAddPermissions)
		auth.POST("/permission/remove", handleRemovePermissions)
		auth.GET("/permission/check", handleCheckPermission)
		auth.GET("/permission/list", handleGetPermissions)

		// 角色管理
		auth.POST("/role/add", handleAddRoles)
		auth.POST("/role/remove", handleRemoveRoles)
		auth.GET("/role/check", handleCheckRole)
		auth.GET("/role/list", handleGetRoles)

		// Session 管理
		auth.GET("/session", handleGetSession)
	}
}

// authMiddleware 认证中间件
func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 Authorization 请求头获取 token
		token := c.GetHeader("Authorization")
		if token == "" {
			// 如果 Authorization 头为空，尝试从 query 参数获取
			token = c.Query("token")
		}

		if token == "" {
			c.JSON(http.StatusUnauthorized, Response{
				Code: 401,
				Msg:  "未提供 token",
			})
			c.Abort()
			return
		}

		ctx := context.Background()
		if !dtoken.IsLogin(ctx, token) {
			c.JSON(http.StatusUnauthorized, Response{
				Code: 401,
				Msg:  "token 无效或已过期",
			})
			c.Abort()
			return
		}

		// 将 token 存入上下文
		c.Set("token", token)
		c.Next()
	}
}

// ============================================================================
// Handler Functions - 处理函数
// ============================================================================

// handleTest 测试接口
func handleTest(c *gin.Context) {
	c.JSON(http.StatusOK, Response{
		Code: 0,
		Msg:  "success",
		Data: "dtoken quick start example is running",
	})
}

// handleLogin 登录
func handleLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code: 400,
			Msg:  "参数错误: " + err.Error(),
		})
		return
	}

	ctx := context.Background()
	token, err := dtoken.Login(ctx, req.LoginID, req.Device, req.DeviceId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code: 500,
			Msg:  "登录失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code: 0,
		Msg:  "登录成功",
		Data: gin.H{
			"token":   token,
			"loginId": req.LoginID,
		},
	})
}

// handleLogout 登出
func handleLogout(c *gin.Context) {
	token := c.GetString("token")
	ctx := context.Background()

	if err := dtoken.Logout(ctx, token); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code: 500,
			Msg:  "登出失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code: 0,
		Msg:  "登出成功",
	})
}

// handleLogoutByDevice 根据设备类型登出
func handleLogoutByDevice(c *gin.Context) {
	loginID := c.Query("loginId")
	device := c.Query("device")

	if loginID == "" || device == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code: 400,
			Msg:  "loginId 和 device 不能为空",
		})
		return
	}

	ctx := context.Background()
	if err := dtoken.LogoutByDevice(ctx, loginID, device); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code: 500,
			Msg:  "登出失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code: 0,
		Msg:  "登出成功",
	})
}

// handleGetTokenInfo 获取 Token 信息
func handleGetTokenInfo(c *gin.Context) {
	token := c.GetString("token")
	ctx := context.Background()

	tokenInfo, err := dtoken.GetTokenInfo(ctx, token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code: 500,
			Msg:  "获取失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code: 0,
		Msg:  "success",
		Data: tokenInfo,
	})
}

// handleGetTokenList 获取 Token 列表
func handleGetTokenList(c *gin.Context) {
	loginID := c.Query("loginId")
	checkAlive := c.Query("checkAlive") == "true"

	if loginID == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code: 400,
			Msg:  "loginId 不能为空",
		})
		return
	}

	ctx := context.Background()
	tokens, err := dtoken.GetTokenValueListByLoginID(ctx, loginID, checkAlive)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code: 500,
			Msg:  "获取失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code: 0,
		Msg:  "success",
		Data: gin.H{
			"tokens": tokens,
			"count":  len(tokens),
		},
	})
}

// handleGetTokenTTL 获取 Token 剩余时间
func handleGetTokenTTL(c *gin.Context) {
	token := c.GetString("token")
	ctx := context.Background()

	ttl, err := dtoken.GetTokenTTL(ctx, token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code: 500,
			Msg:  "获取失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code: 0,
		Msg:  "success",
		Data: gin.H{
			"ttl": ttl,
		},
	})
}

// handleGetOnlineCount 获取在线终端数量
func handleGetOnlineCount(c *gin.Context) {
	loginID := c.Query("loginId")
	device := c.Query("device")

	if loginID == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code: 400,
			Msg:  "loginId 不能为空",
		})
		return
	}

	ctx := context.Background()
	var count int
	var err error

	if device != "" {
		count, err = dtoken.GetOnlineTerminalCountByDevice(ctx, loginID, device)
	} else {
		count, err = dtoken.GetOnlineTerminalCount(ctx, loginID)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code: 500,
			Msg:  "获取失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code: 0,
		Msg:  "success",
		Data: gin.H{
			"count": count,
		},
	})
}

// handleKickout 踢人下线
func handleKickout(c *gin.Context) {
	targetToken := c.Query("targetToken")
	if targetToken == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code: 400,
			Msg:  "targetToken 不能为空",
		})
		return
	}

	ctx := context.Background()
	if err := dtoken.Kickout(ctx, targetToken); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code: 500,
			Msg:  "踢人失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code: 0,
		Msg:  "踢人成功",
	})
}

// handleKickoutByDevice 根据设备类型踢人下线
func handleKickoutByDevice(c *gin.Context) {
	loginID := c.Query("loginId")
	device := c.Query("device")

	if loginID == "" || device == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code: 400,
			Msg:  "loginId 和 device 不能为空",
		})
		return
	}

	ctx := context.Background()
	if err := dtoken.KickoutByDevice(ctx, loginID, device); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code: 500,
			Msg:  "踢人失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code: 0,
		Msg:  "踢人成功",
	})
}

// handleReplace 顶人下线
func handleReplace(c *gin.Context) {
	targetToken := c.Query("targetToken")
	if targetToken == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code: 400,
			Msg:  "targetToken 不能为空",
		})
		return
	}

	ctx := context.Background()
	if err := dtoken.Replace(ctx, targetToken); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code: 500,
			Msg:  "顶人失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code: 0,
		Msg:  "顶人成功",
	})
}

// handleReplaceByDevice 根据设备类型顶人下线
func handleReplaceByDevice(c *gin.Context) {
	loginID := c.Query("loginId")
	device := c.Query("device")

	if loginID == "" || device == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code: 400,
			Msg:  "loginId 和 device 不能为空",
		})
		return
	}

	ctx := context.Background()
	if err := dtoken.ReplaceByDevice(ctx, loginID, device); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code: 500,
			Msg:  "顶人失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code: 0,
		Msg:  "顶人成功",
	})
}

// handleDisable 封禁账号
func handleDisable(c *gin.Context) {
	var req DisableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code: 400,
			Msg:  "参数错误: " + err.Error(),
		})
		return
	}

	ctx := context.Background()
	duration := time.Duration(req.Duration) * time.Second
	err := dtoken.Disable(ctx, req.LoginID, duration, req.Reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code: 500,
			Msg:  "封禁失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code: 0,
		Msg:  "封禁成功",
	})
}

// handleUndisable 解封账号
func handleUndisable(c *gin.Context) {
	loginID := c.Query("loginId")
	if loginID == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code: 400,
			Msg:  "loginId 不能为空",
		})
		return
	}

	ctx := context.Background()
	if err := dtoken.Untie(ctx, loginID); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code: 500,
			Msg:  "解封失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code: 0,
		Msg:  "解封成功",
	})
}

// handleGetDisableInfo 获取封禁信息
func handleGetDisableInfo(c *gin.Context) {
	loginID := c.Query("loginId")
	if loginID == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code: 400,
			Msg:  "loginId 不能为空",
		})
		return
	}

	ctx := context.Background()
	info, err := dtoken.GetDisableInfo(ctx, loginID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code: 500,
			Msg:  "获取失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code: 0,
		Msg:  "success",
		Data: info,
	})
}

// handleAddPermissions 添加权限
func handleAddPermissions(c *gin.Context) {
	var req PermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code: 400,
			Msg:  "参数错误: " + err.Error(),
		})
		return
	}

	ctx := context.Background()
	if err := dtoken.AddPermissions(ctx, req.LoginID, req.Permissions); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code: 500,
			Msg:  "添加失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code: 0,
		Msg:  "添加成功",
	})
}

// handleRemovePermissions 移除权限
func handleRemovePermissions(c *gin.Context) {
	var req PermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code: 400,
			Msg:  "参数错误: " + err.Error(),
		})
		return
	}

	ctx := context.Background()
	if err := dtoken.RemovePermissions(ctx, req.LoginID, req.Permissions); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code: 500,
			Msg:  "移除失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code: 0,
		Msg:  "移除成功",
	})
}

// handleCheckPermission 检查权限
func handleCheckPermission(c *gin.Context) {
	token := c.GetString("token")
	permission := c.Query("permission")

	if permission == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code: 400,
			Msg:  "permission 不能为空",
		})
		return
	}

	ctx := context.Background()
	has := dtoken.HasPermissionByToken(ctx, token, permission)

	c.JSON(http.StatusOK, Response{
		Code: 0,
		Msg:  "success",
		Data: gin.H{
			"hasPermission": has,
		},
	})
}

// handleGetPermissions 获取权限列表
func handleGetPermissions(c *gin.Context) {
	loginID := c.Query("loginId")
	if loginID == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code: 400,
			Msg:  "loginId 不能为空",
		})
		return
	}

	ctx := context.Background()
	permissions, err := dtoken.GetPermissions(ctx, loginID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code: 500,
			Msg:  "获取失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code: 0,
		Msg:  "success",
		Data: gin.H{
			"permissions": permissions,
		},
	})
}

// handleAddRoles 添加角色
func handleAddRoles(c *gin.Context) {
	var req RoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code: 400,
			Msg:  "参数错误: " + err.Error(),
		})
		return
	}

	ctx := context.Background()
	if err := dtoken.AddRoles(ctx, req.LoginID, req.Roles); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code: 500,
			Msg:  "添加失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code: 0,
		Msg:  "添加成功",
	})
}

// handleRemoveRoles 移除角色
func handleRemoveRoles(c *gin.Context) {
	var req RoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code: 400,
			Msg:  "参数错误: " + err.Error(),
		})
		return
	}

	ctx := context.Background()
	if err := dtoken.RemoveRoles(ctx, req.LoginID, req.Roles); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code: 500,
			Msg:  "移除失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code: 0,
		Msg:  "移除成功",
	})
}

// handleCheckRole 检查角色
func handleCheckRole(c *gin.Context) {
	token := c.GetString("token")
	role := c.Query("role")

	if role == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code: 400,
			Msg:  "role 不能为空",
		})
		return
	}

	ctx := context.Background()
	has := dtoken.HasRoleByToken(ctx, token, role)

	c.JSON(http.StatusOK, Response{
		Code: 0,
		Msg:  "success",
		Data: gin.H{
			"hasRole": has,
		},
	})
}

// handleGetRoles 获取角色列表
func handleGetRoles(c *gin.Context) {
	loginID := c.Query("loginId")
	if loginID == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code: 400,
			Msg:  "loginId 不能为空",
		})
		return
	}

	ctx := context.Background()
	roles, err := dtoken.GetRoles(ctx, loginID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code: 500,
			Msg:  "获取失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code: 0,
		Msg:  "success",
		Data: gin.H{
			"roles": roles,
		},
	})
}

// handleGetSession 获取 Session 信息
func handleGetSession(c *gin.Context) {
	token := c.GetString("token")
	ctx := context.Background()

	session, err := dtoken.GetSessionByToken(ctx, token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code: 500,
			Msg:  "获取失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code: 0,
		Msg:  "success",
		Data: session,
	})
}
