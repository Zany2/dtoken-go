// @Author daixk 2026/2/2
// Quick Start Example - Comprehensive Test Suite for DToken Framework
// 快速开始示例 - DToken 框架完整测试套件

package main

import (
	"fmt"
	"github.com/Zany2/dtoken-go/com/storage/redis"
	"github.com/Zany2/dtoken-go/core/builder"
	"github.com/Zany2/dtoken-go/dtoken"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

// ============================================================================
// Data Structures - 数据结构
// ============================================================================

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

// LoginByTokenRequest 通过 Token 登录请求
type LoginByTokenRequest struct {
	Token string `json:"token" binding:"required"`
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
	Duration int64  `json:"duration" binding:"required"` // 封禁时长（秒）
	Reason   string `json:"reason"`
}

// DeviceRequest 设备请求
type DeviceRequest struct {
	LoginID  string `json:"loginId" binding:"required"`
	Device   string `json:"device"`
	DeviceId string `json:"deviceId"`
}

// TokenRequest Token 请求
type TokenRequest struct {
	Token string `json:"token" binding:"required"`
}

// ============================================================================
// Helper Functions - 辅助函数
// ============================================================================

// success 成功响应
func success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code: 200,
		Msg:  "success",
		Data: data,
	})
}

// fail 失败响应
func fail(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, Response{
		Code: 500,
		Msg:  msg,
	})
}

// unauthorized 未授权响应
// 返回401未授权响应
func unauthorized(c *gin.Context, msg string) {
	c.JSON(http.StatusUnauthorized, Response{
		Code: 401,
		Msg:  msg,
	})
	c.Abort()
}

// ============================================================================
// Middleware - 中间件
// ============================================================================

// authMiddleware 登录验证中间件
// 验证用户是否已登录，并将token存入上下文
func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从Header中获取token
		token := c.GetHeader("Authorization")
		if token == "" {
			unauthorized(c, "未提供认证token")
			return
		}

		// 验证token是否有效
		if err := dtoken.CheckLogin(c.Request.Context(), token); err != nil {
			unauthorized(c, "token无效或已过期: "+err.Error())
			return
		}

		// 将token存入上下文
		c.Set("token", token)

		// 继续处理请求
		c.Next()
	}
}

// ============================================================================
// Initialization - 初始化
// ============================================================================

// initDToken 初始化 DToken 框架
func initDToken() error {
	// 使用 Redis 存储
	// Redis URL 格式: redis://[username]:[password]@[host]:[port]/[database]
	storage, err := redis.NewStorage("redis://:root@192.168.19.104:6379/0?dial_timeout=3&read_timeout=10s&max_retries=2")
	if err != nil {
		panic("Failed to connect to Redis: " + err.Error())
	}

	// 使用 Builder 构建管理器
	mgr := builder.NewBuilder().
		TokenName("token").    // Token 名称
		Timeout(7200).         // 超时时间 2 小时
		RenewMaxRefresh(1800). // 续期触发阈值 30 分钟
		IsConcurrent(true).    // 允许并发登录
		MaxLoginCount(5).      // 最大并发登录数 5
		IsReadHeader(true).    // 从 Header 读取 Token
		IsLog(true).           // 开启日志
		IsPrintBanner(true).   // 打印启动 Banner
		SetStorage(storage).   // 设置存储适配器
		Build()

	// 设置全局管理器
	dtoken.SetManager(mgr)

	fmt.Println("DToken 框架初始化成功")
	return nil
}

// ============================================================================
// Authentication APIs - 认证相关接口
// ============================================================================

// handleLogin 用户登录
// POST /api/auth/login
func handleLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}

	var token string
	var err error

	// 根据是否提供设备信息选择登录方式
	if req.Device != "" && req.DeviceId != "" {
		token, err = dtoken.Login(c.Request.Context(), req.LoginID, req.Device, req.DeviceId)
	} else if req.Device != "" {
		token, err = dtoken.Login(c.Request.Context(), req.LoginID, req.Device)
	} else {
		token, err = dtoken.Login(c.Request.Context(), req.LoginID)
	}

	if err != nil {
		fail(c, "登录失败: "+err.Error())
		return
	}

	success(c, gin.H{
		"token":   token,
		"loginId": req.LoginID,
	})
}

// handleLoginByToken 通过 Token 续期登录
// POST /api/auth/login-by-token
func handleLoginByToken(c *gin.Context) {
	var req LoginByTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}

	if err := dtoken.LoginByToken(c.Request.Context(), req.Token); err != nil {
		fail(c, "Token 续期失败: "+err.Error())
		return
	}

	success(c, "Token 续期成功")
}

// handleLogout 用户登出
// POST /api/auth/logout
func handleLogout(c *gin.Context) {
	// 从上下文获取token
	token, _ := c.Get("token")
	tokenStr := token.(string)

	if err := dtoken.Logout(c.Request.Context(), tokenStr); err != nil {
		fail(c, "登出失败: "+err.Error())
		return
	}

	success(c, "登出成功")
}

// handleLogoutByDevice 根据设备类型登出
// POST /api/auth/logout-by-device
func handleLogoutByDevice(c *gin.Context) {
	var req DeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}

	if err := dtoken.LogoutByDevice(c.Request.Context(), req.LoginID, req.Device); err != nil {
		fail(c, "登出失败: "+err.Error())
		return
	}

	success(c, "登出成功")
}

// handleLogoutByDeviceAndDeviceId 根据设备类型和设备ID登出
// POST /api/auth/logout-by-device-id
func handleLogoutByDeviceAndDeviceId(c *gin.Context) {
	var req DeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}

	if err := dtoken.LogoutByDeviceAndDeviceId(c.Request.Context(), req.LoginID, req.Device, req.DeviceId); err != nil {
		fail(c, "登出失败: "+err.Error())
		return
	}

	success(c, "登出成功")
}

// handleLogoutByLoginID 根据 LoginID 登出所有终端
// POST /api/auth/logout-by-login-id
func handleLogoutByLoginID(c *gin.Context) {
	var req struct {
		LoginID string `json:"loginId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}

	if err := dtoken.LogoutByLoginID(c.Request.Context(), req.LoginID); err != nil {
		fail(c, "登出失败: "+err.Error())
		return
	}

	success(c, "登出成功")
}

// handleIsLogin 检查用户是否登录
// POST /api/auth/is-login
func handleIsLogin(c *gin.Context) {
	// 从上下文获取token
	token, _ := c.Get("token")
	tokenStr := token.(string)

	isLogin := dtoken.IsLogin(c.Request.Context(), tokenStr)

	success(c, gin.H{
		"isLogin": isLogin,
	})
}

// handleCheckLogin 验证登录状态（未登录返回错误）
// POST /api/auth/check-login
func handleCheckLogin(c *gin.Context) {
	// 从上下文获取token
	token, _ := c.Get("token")
	tokenStr := token.(string)

	if err := dtoken.CheckLogin(c.Request.Context(), tokenStr); err != nil {
		fail(c, "未登录: "+err.Error())
		return
	}

	success(c, "已登录")
}

// handleGetLoginID 获取登录ID
// POST /api/auth/get-login-id
func handleGetLoginID(c *gin.Context) {
	// 从上下文获取token
	token, _ := c.Get("token")
	tokenStr := token.(string)

	loginID, err := dtoken.GetLoginID(c.Request.Context(), tokenStr)
	if err != nil {
		fail(c, "获取登录ID失败: "+err.Error())
		return
	}

	success(c, gin.H{
		"loginId": loginID,
	})
}

// handleGetTokenInfo 获取 Token 信息
// POST /api/auth/get-token-info
func handleGetTokenInfo(c *gin.Context) {
	// 从上下文获取token
	token, _ := c.Get("token")
	tokenStr := token.(string)

	tokenInfo, err := dtoken.GetTokenInfo(c.Request.Context(), tokenStr)
	if err != nil {
		fail(c, "获取Token信息失败: "+err.Error())
		return
	}

	success(c, tokenInfo)
}

// handleGetDevice 获取设备类型
// POST /api/auth/get-device
func handleGetDevice(c *gin.Context) {
	// 从上下文获取token
	token, _ := c.Get("token")
	tokenStr := token.(string)

	device, err := dtoken.GetDevice(c.Request.Context(), tokenStr)
	if err != nil {
		fail(c, "获取设备类型失败: "+err.Error())
		return
	}

	success(c, gin.H{
		"device": device,
	})
}

// handleGetDeviceId 获取设备ID
// POST /api/auth/get-device-id
func handleGetDeviceId(c *gin.Context) {
	// 从上下文获取token
	token, _ := c.Get("token")
	tokenStr := token.(string)

	deviceId, err := dtoken.GetDeviceId(c.Request.Context(), tokenStr)
	if err != nil {
		fail(c, "获取设备ID失败: "+err.Error())
		return
	}

	success(c, gin.H{
		"deviceId": deviceId,
	})
}

// handleGetTokenCreateTime 获取 Token 创建时间
// POST /api/auth/get-token-create-time
func handleGetTokenCreateTime(c *gin.Context) {
	// 从上下文获取token
	token, _ := c.Get("token")
	tokenStr := token.(string)

	createTime, err := dtoken.GetTokenCreateTime(c.Request.Context(), tokenStr)
	if err != nil {
		fail(c, "获取Token创建时间失败: "+err.Error())
		return
	}

	success(c, gin.H{
		"createTime": createTime,
	})
}

// handleGetTokenTTL 获取 Token 剩余有效时间
// POST /api/auth/get-token-ttl
func handleGetTokenTTL(c *gin.Context) {
	// 从上下文获取token
	token, _ := c.Get("token")
	tokenStr := token.(string)

	ttl, err := dtoken.GetTokenTTL(c.Request.Context(), tokenStr)
	if err != nil {
		fail(c, "获取Token TTL失败: "+err.Error())
		return
	}

	success(c, gin.H{
		"ttl": ttl,
	})
}

// handleGetOnlineTerminalCount 获取在线终端总数
// GET /api/auth/online-count/:loginId
func handleGetOnlineTerminalCount(c *gin.Context) {
	loginID := c.Param("loginId")
	if loginID == "" {
		fail(c, "loginId 不能为空")
		return
	}

	count, err := dtoken.GetOnlineTerminalCount(c.Request.Context(), loginID)
	if err != nil {
		fail(c, "获取在线终端数失败: "+err.Error())
		return
	}

	success(c, gin.H{
		"count": count,
	})
}

// handleGetOnlineTerminalCountByDevice 获取指定设备的在线终端数
// GET /api/auth/online-count/:loginId/:device
func handleGetOnlineTerminalCountByDevice(c *gin.Context) {
	loginID := c.Param("loginId")
	device := c.Param("device")
	fmt.Println("loginID: %s, device: %s", loginID, device)
	if loginID == "" || device == "" {
		fail(c, "loginId 和 device 不能为空")
		return
	}

	count, err := dtoken.GetOnlineTerminalCountByDevice(c.Request.Context(), loginID, device)
	if err != nil {
		fail(c, "获取在线终端数失败: "+err.Error())
		return
	}

	success(c, gin.H{
		"count": count,
	})
}

// handleGetOnlineTerminalCountByDeviceAndDeviceId 获取指定设备和设备ID的在线终端数
// GET /api/auth/online-count/:loginId/:device/:deviceId
func handleGetOnlineTerminalCountByDeviceAndDeviceId(c *gin.Context) {
	loginID := c.Param("loginId")
	device := c.Param("device")
	deviceId := c.Param("deviceId")
	if loginID == "" || device == "" || deviceId == "" {
		fail(c, "loginId、device 和 deviceId 不能为空")
		return
	}

	count, err := dtoken.GetOnlineTerminalCountByDeviceAndDeviceId(c.Request.Context(), loginID, device, deviceId)
	if err != nil {
		fail(c, "获取在线终端数失败: "+err.Error())
		return
	}

	success(c, gin.H{
		"count": count,
	})
}

// ============================================================================
// Online Status Management APIs - 在线状态管理接口
// ============================================================================

// handleKickout 根据 Token 踢人下线
// POST /api/online/kickout
func handleKickout(c *gin.Context) {
	var req TokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}

	if err := dtoken.Kickout(c.Request.Context(), req.Token); err != nil {
		fail(c, "踢人下线失败: "+err.Error())
		return
	}

	success(c, "踢人下线成功")
}

// handleKickoutByDevice 根据设备类型踢人下线
// POST /api/online/kickout-by-device
func handleKickoutByDevice(c *gin.Context) {
	var req DeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}

	if err := dtoken.KickoutByDevice(c.Request.Context(), req.LoginID, req.Device); err != nil {
		fail(c, "踢人下线失败: "+err.Error())
		return
	}

	success(c, "踢人下线成功")
}

// handleKickoutByDeviceAndDeviceId 根据设备和设备ID踢人下线
// POST /api/online/kickout-by-device-id
func handleKickoutByDeviceAndDeviceId(c *gin.Context) {
	var req DeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}

	if err := dtoken.KickoutByDeviceAndDeviceId(c.Request.Context(), req.LoginID, req.Device, req.DeviceId); err != nil {
		fail(c, "踢人下线失败: "+err.Error())
		return
	}

	success(c, "踢人下线成功")
}

// handleKickoutByLoginID 根据 LoginID 踢出所有终端
// POST /api/online/kickout-by-login-id
func handleKickoutByLoginID(c *gin.Context) {
	var req struct {
		LoginID string `json:"loginId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}

	if err := dtoken.KickoutByLoginID(c.Request.Context(), req.LoginID); err != nil {
		fail(c, "踢人下线失败: "+err.Error())
		return
	}

	success(c, "踢人下线成功")
}

// handleReplace 根据 Token 顶人下线
// POST /api/online/replace
func handleReplace(c *gin.Context) {
	// 从上下文获取token
	token, _ := c.Get("token")
	tokenStr := token.(string)

	if err := dtoken.Replace(c.Request.Context(), tokenStr); err != nil {
		fail(c, "顶人下线失败: "+err.Error())
		return
	}

	success(c, "顶人下线成功")
}

// handleReplaceByDevice 根据设备类型顶人下线
// POST /api/online/replace-by-device
func handleReplaceByDevice(c *gin.Context) {
	var req DeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}

	if err := dtoken.ReplaceByDevice(c.Request.Context(), req.LoginID, req.Device); err != nil {
		fail(c, "顶人下线失败: "+err.Error())
		return
	}

	success(c, "顶人下线成功")
}

// handleReplaceByDeviceAndDeviceId 根据设备和设备ID顶人下线
// POST /api/online/replace-by-device-id
func handleReplaceByDeviceAndDeviceId(c *gin.Context) {
	var req DeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}

	if err := dtoken.ReplaceByDeviceAndDeviceId(c.Request.Context(), req.LoginID, req.Device, req.DeviceId); err != nil {
		fail(c, "顶人下线失败: "+err.Error())
		return
	}

	success(c, "顶人下线成功")
}

// handleReplaceByLoginID 根据 LoginID 顶替所有终端
// POST /api/online/replace-by-login-id
func handleReplaceByLoginID(c *gin.Context) {
	var req struct {
		LoginID string `json:"loginId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}

	if err := dtoken.ReplaceByLoginID(c.Request.Context(), req.LoginID); err != nil {
		fail(c, "顶人下线失败: "+err.Error())
		return
	}

	success(c, "顶人下线成功")
}

// ============================================================================
// Permission Management APIs - 权限管理接口
// ============================================================================

// handleAddPermissions 为用户添加权限
// POST /api/permission/add
func handleAddPermissions(c *gin.Context) {
	var req PermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}

	if err := dtoken.AddPermissions(c.Request.Context(), req.LoginID, req.Permissions); err != nil {
		fail(c, "添加权限失败: "+err.Error())
		return
	}

	success(c, "添加权限成功")
}

// handleAddPermissionsByToken 根据 Token 添加权限
// POST /api/permission/add-by-token
func handleAddPermissionsByToken(c *gin.Context) {
	var req struct {
		Permissions []string `json:"permissions" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}

	// 从上下文获取token
	token, _ := c.Get("token")
	tokenStr := token.(string)

	if err := dtoken.AddPermissionsByToken(c.Request.Context(), tokenStr, req.Permissions); err != nil {
		fail(c, "添加权限失败: "+err.Error())
		return
	}

	success(c, "添加权限成功")
}

// handleRemovePermissions 删除用户权限
// POST /api/permission/remove
func handleRemovePermissions(c *gin.Context) {
	var req PermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}

	if err := dtoken.RemovePermissions(c.Request.Context(), req.LoginID, req.Permissions); err != nil {
		fail(c, "删除权限失败: "+err.Error())
		return
	}

	success(c, "删除权限成功")
}

// handleRemovePermissionsByToken 根据 Token 删除权限
// POST /api/permission/remove-by-token
func handleRemovePermissionsByToken(c *gin.Context) {
	var req struct {
		Permissions []string `json:"permissions" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}

	// 从上下文获取token
	token, _ := c.Get("token")
	tokenStr := token.(string)

	if err := dtoken.RemovePermissionsByToken(c.Request.Context(), tokenStr, req.Permissions); err != nil {
		fail(c, "删除权限失败: "+err.Error())
		return
	}

	success(c, "删除权限成功")
}

// handleGetPermissions 获取用户权限列表
// GET /api/permission/list/:loginId
func handleGetPermissions(c *gin.Context) {
	loginID := c.Param("loginId")
	if loginID == "" {
		fail(c, "loginId 不能为空")
		return
	}

	permissions, err := dtoken.GetPermissions(c.Request.Context(), loginID)
	if err != nil {
		fail(c, "获取权限列表失败: "+err.Error())
		return
	}

	success(c, gin.H{
		"permissions": permissions,
	})
}

// handleGetPermissionsByToken 根据 Token 获取权限列表
// POST /api/permission/list-by-token
func handleGetPermissionsByToken(c *gin.Context) {
	// 从上下文获取token
	token, _ := c.Get("token")
	tokenStr := token.(string)

	permissions, err := dtoken.GetPermissionsByToken(c.Request.Context(), tokenStr)
	if err != nil {
		fail(c, "获取权限列表失败: "+err.Error())
		return
	}

	success(c, gin.H{
		"permissions": permissions,
	})
}

// handleHasPermission 检查是否拥有指定权限
// POST /api/permission/has
func handleHasPermission(c *gin.Context) {
	var req struct {
		LoginID    string `json:"loginId" binding:"required"`
		Permission string `json:"permission" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}

	has := dtoken.HasPermission(c.Request.Context(), req.LoginID, req.Permission)

	success(c, gin.H{
		"hasPermission": has,
	})
}

// handleHasPermissionByToken 根据 Token 检查权限
// POST /api/permission/has-by-token
func handleHasPermissionByToken(c *gin.Context) {
	var req struct {
		Permission string `json:"permission" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}

	// 从上下文获取token
	token, _ := c.Get("token")
	tokenStr := token.(string)

	has := dtoken.HasPermissionByToken(c.Request.Context(), tokenStr, req.Permission)

	success(c, gin.H{
		"hasPermission": has,
	})
}

// handleHasPermissionsAnd 检查是否拥有所有权限（AND逻辑）
// POST /api/permission/has-and
func handleHasPermissionsAnd(c *gin.Context) {
	var req PermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}

	has := dtoken.HasPermissionsAnd(c.Request.Context(), req.LoginID, req.Permissions)

	success(c, gin.H{
		"hasAllPermissions": has,
	})
}

// handleHasPermissionsAndByToken 根据 Token 检查所有权限
// POST /api/permission/has-and-by-token
func handleHasPermissionsAndByToken(c *gin.Context) {
	var req struct {
		Permissions []string `json:"permissions" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}

	// 从上下文获取token
	token, _ := c.Get("token")
	tokenStr := token.(string)

	has := dtoken.HasPermissionsAndByToken(c.Request.Context(), tokenStr, req.Permissions)

	success(c, gin.H{
		"hasAllPermissions": has,
	})
}

// handleHasPermissionsOr 检查是否拥有任一权限（OR逻辑）
// POST /api/permission/has-or
func handleHasPermissionsOr(c *gin.Context) {
	var req PermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}

	has := dtoken.HasPermissionsOr(c.Request.Context(), req.LoginID, req.Permissions)

	success(c, gin.H{
		"hasAnyPermission": has,
	})
}

// handleHasPermissionsOrByToken 根据 Token 检查任一权限
// POST /api/permission/has-or-by-token
func handleHasPermissionsOrByToken(c *gin.Context) {
	var req struct {
		Permissions []string `json:"permissions" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}

	// 从上下文获取token
	token, _ := c.Get("token")
	tokenStr := token.(string)

	has := dtoken.HasPermissionsOrByToken(c.Request.Context(), tokenStr, req.Permissions)

	success(c, gin.H{
		"hasAnyPermission": has,
	})
}

// ============================================================================
// Role Management APIs - 角色管理接口
// ============================================================================

// handleAddRoles 为用户添加角色
// POST /api/role/add
func handleAddRoles(c *gin.Context) {
	var req RoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}

	if err := dtoken.AddRoles(c.Request.Context(), req.LoginID, req.Roles); err != nil {
		fail(c, "添加角色失败: "+err.Error())
		return
	}

	success(c, "添加角色成功")
}

// handleAddRolesByToken 根据 Token 添加角色
// POST /api/role/add-by-token
func handleAddRolesByToken(c *gin.Context) {
	var req struct {
		Roles []string `json:"roles" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}

	// 从上下文获取token
	token, _ := c.Get("token")
	tokenStr := token.(string)

	if err := dtoken.AddRolesByToken(c.Request.Context(), tokenStr, req.Roles); err != nil {
		fail(c, "添加角色失败: "+err.Error())
		return
	}

	success(c, "添加角色成功")
}

// handleRemoveRoles 删除用户角色
// POST /api/role/remove
func handleRemoveRoles(c *gin.Context) {
	var req RoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}

	if err := dtoken.RemoveRoles(c.Request.Context(), req.LoginID, req.Roles); err != nil {
		fail(c, "删除角色失败: "+err.Error())
		return
	}

	success(c, "删除角色成功")
}

// handleRemoveRolesByToken 根据 Token 删除角色
// POST /api/role/remove-by-token
func handleRemoveRolesByToken(c *gin.Context) {
	var req struct {
		Roles []string `json:"roles" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}

	// 从上下文获取token
	token, _ := c.Get("token")
	tokenStr := token.(string)

	if err := dtoken.RemoveRolesByToken(c.Request.Context(), tokenStr, req.Roles); err != nil {
		fail(c, "删除角色失败: "+err.Error())
		return
	}

	success(c, "删除角色成功")
}

// handleGetRoles 获取用户角色列表
// GET /api/role/list/:loginId
func handleGetRoles(c *gin.Context) {
	loginID := c.Param("loginId")
	if loginID == "" {
		fail(c, "loginId 不能为空")
		return
	}

	roles, err := dtoken.GetRoles(c.Request.Context(), loginID)
	if err != nil {
		fail(c, "获取角色列表失败: "+err.Error())
		return
	}

	success(c, gin.H{
		"roles": roles,
	})
}

// handleGetRolesByToken 根据 Token 获取角色列表
// POST /api/role/list-by-token
func handleGetRolesByToken(c *gin.Context) {
	// 从上下文获取token
	token, _ := c.Get("token")
	tokenStr := token.(string)

	roles, err := dtoken.GetRolesByToken(c.Request.Context(), tokenStr)
	if err != nil {
		fail(c, "获取角色列表失败: "+err.Error())
		return
	}

	success(c, gin.H{
		"roles": roles,
	})
}

// handleHasRole 检查是否拥有指定角色
// POST /api/role/has
func handleHasRole(c *gin.Context) {
	var req struct {
		LoginID string `json:"loginId" binding:"required"`
		Role    string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}

	has := dtoken.HasRole(c.Request.Context(), req.LoginID, req.Role)

	success(c, gin.H{
		"hasRole": has,
	})
}

// handleHasRoleByToken 根据 Token 检查角色
// POST /api/role/has-by-token
func handleHasRoleByToken(c *gin.Context) {
	var req struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}

	// 从上下文获取token
	token, _ := c.Get("token")
	tokenStr := token.(string)

	has := dtoken.HasRoleByToken(c.Request.Context(), tokenStr, req.Role)

	success(c, gin.H{
		"hasRole": has,
	})
}

// handleHasRolesAnd 检查是否拥有所有角色（AND逻辑）
// POST /api/role/has-and
func handleHasRolesAnd(c *gin.Context) {
	var req RoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}

	has := dtoken.HasRolesAnd(c.Request.Context(), req.LoginID, req.Roles)

	success(c, gin.H{
		"hasAllRoles": has,
	})
}

// handleHasRolesAndByToken 根据 Token 检查所有角色
// POST /api/role/has-and-by-token
func handleHasRolesAndByToken(c *gin.Context) {
	var req struct {
		Roles []string `json:"roles" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}

	// 从上下文获取token
	token, _ := c.Get("token")
	tokenStr := token.(string)

	has := dtoken.HasRolesAndByToken(c.Request.Context(), tokenStr, req.Roles)

	success(c, gin.H{
		"hasAllRoles": has,
	})
}

// handleHasRolesOr 检查是否拥有任一角色（OR逻辑）
// POST /api/role/has-or
func handleHasRolesOr(c *gin.Context) {
	var req RoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}

	has := dtoken.HasRolesOr(c.Request.Context(), req.LoginID, req.Roles)

	success(c, gin.H{
		"hasAnyRole": has,
	})
}

// handleHasRolesOrByToken 根据 Token 检查任一角色
// POST /api/role/has-or-by-token
func handleHasRolesOrByToken(c *gin.Context) {
	var req struct {
		Roles []string `json:"roles" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}

	// 从上下文获取token
	token, _ := c.Get("token")
	tokenStr := token.(string)

	has := dtoken.HasRolesOrByToken(c.Request.Context(), tokenStr, req.Roles)

	success(c, gin.H{
		"hasAnyRole": has,
	})
}

// ============================================================================
// Session Management APIs - Session 管理接口
// ============================================================================

// handleGetSession 获取指定登录ID的会话
// GET /api/session/:loginId
func handleGetSession(c *gin.Context) {
	loginID := c.Param("loginId")
	if loginID == "" {
		fail(c, "loginId 不能为空")
		return
	}

	session, err := dtoken.GetSession(c.Request.Context(), loginID)
	if err != nil {
		fail(c, "获取会话失败: "+err.Error())
		return
	}

	success(c, session)
}

// handleGetSessionByToken 通过 Token 获取会话
// POST /api/session/by-token
func handleGetSessionByToken(c *gin.Context) {
	// 从上下文获取token
	token, _ := c.Get("token")
	tokenStr := token.(string)

	session, err := dtoken.GetSessionByToken(c.Request.Context(), tokenStr)
	if err != nil {
		fail(c, "获取会话失败: "+err.Error())
		return
	}

	success(c, session)
}

// handleGetTokenValueListByLoginID 获取指定登录ID的所有Token
// GET /api/session/tokens/:loginId
func handleGetTokenValueListByLoginID(c *gin.Context) {
	loginID := c.Param("loginId")
	if loginID == "" {
		fail(c, "loginId 不能为空")
		return
	}

	checkAlive := c.DefaultQuery("checkAlive", "true") == "true"

	tokens, err := dtoken.GetTokenValueListByLoginID(c.Request.Context(), loginID, checkAlive)
	if err != nil {
		fail(c, "获取Token列表失败: "+err.Error())
		return
	}

	success(c, gin.H{
		"tokens": tokens,
	})
}

// handleGetTokenValueListByDevice 获取指定设备类型的所有Token
// GET /api/session/tokens/:loginId/:device
func handleGetTokenValueListByDevice(c *gin.Context) {
	loginID := c.Param("loginId")
	device := c.Param("device")
	if loginID == "" || device == "" {
		fail(c, "loginId 和 device 不能为空")
		return
	}

	checkAlive := c.DefaultQuery("checkAlive", "true") == "true"

	tokens, err := dtoken.GetTokenValueListByDevice(c.Request.Context(), loginID, device, checkAlive)
	if err != nil {
		fail(c, "获取Token列表失败: "+err.Error())
		return
	}

	success(c, gin.H{
		"tokens": tokens,
	})
}

// ============================================================================
// Account Disable Management APIs - 账号封禁管理接口
// ============================================================================

// handleDisable 封禁账号指定时长
// POST /api/disable/ban
func handleDisable(c *gin.Context) {
	var req DisableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}

	if err := dtoken.Disable(c.Request.Context(), req.LoginID, time.Duration(req.Duration)*time.Second, req.Reason); err != nil {
		fail(c, "封禁账号失败: "+err.Error())
		return
	}

	success(c, "封禁账号成功")
}

// handleUntie 解封账号
// POST /api/disable/unban
func handleUntie(c *gin.Context) {
	var req struct {
		LoginID string `json:"loginId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}

	if err := dtoken.Untie(c.Request.Context(), req.LoginID); err != nil {
		fail(c, "解封账号失败: "+err.Error())
		return
	}

	success(c, "解封账号成功")
}

// handleIsDisable 检查账号是否被封禁
// GET /api/disable/is-disabled/:loginId
func handleIsDisable(c *gin.Context) {
	loginID := c.Param("loginId")
	if loginID == "" {
		fail(c, "loginId 不能为空")
		return
	}

	isDisabled := dtoken.IsDisable(c.Request.Context(), loginID)

	success(c, gin.H{
		"isDisabled": isDisabled,
	})
}

// handleGetDisableInfo 获取账号封禁信息
// GET /api/disable/info/:loginId
func handleGetDisableInfo(c *gin.Context) {
	loginID := c.Param("loginId")
	if loginID == "" {
		fail(c, "loginId 不能为空")
		return
	}

	info, err := dtoken.GetDisableInfo(c.Request.Context(), loginID)
	if err != nil {
		fail(c, "获取封禁信息失败: "+err.Error())
		return
	}

	success(c, info)
}

// handleGetDisableTTL 获取账号剩余封禁时间
// GET /api/disable/ttl/:loginId
func handleGetDisableTTL(c *gin.Context) {
	loginID := c.Param("loginId")
	if loginID == "" {
		fail(c, "loginId 不能为空")
		return
	}

	ttl, err := dtoken.GetDisableTTL(c.Request.Context(), loginID)
	if err != nil {
		fail(c, "获取封禁TTL失败: "+err.Error())
		return
	}

	success(c, gin.H{
		"ttl": ttl,
	})
}

// ============================================================================
// Router Setup - 路由注册
// ============================================================================

// setupRoutes 注册所有路由
func setupRoutes(r *gin.Engine) {
	// 根路径
	r.GET("/", func(c *gin.Context) {
		success(c, gin.H{
			"message": "DToken Quick Start API",
			"version": "1.0.0",
		})
	})

	api := r.Group("/api")

	// ========== 认证相关接口 ==========
	auth := api.Group("/auth")
	{
		// 不需要验证的接口
		auth.POST("/login", handleLogin)                 // 用户登录
		auth.POST("/login-by-token", handleLoginByToken) // Token 续期登录

		// 需要验证的接口
		auth.POST("/logout", authMiddleware(), handleLogout)                                                                    // 用户登出
		auth.POST("/logout-by-device", authMiddleware(), handleLogoutByDevice)                                                  // 根据设备类型登出
		auth.POST("/logout-by-device-id", authMiddleware(), handleLogoutByDeviceAndDeviceId)                                    // 根据设备和设备ID登出
		auth.POST("/logout-by-login-id", authMiddleware(), handleLogoutByLoginID)                                               // 根据LoginID登出所有终端
		auth.POST("/is-login", authMiddleware(), handleIsLogin)                                                                 // 检查是否登录
		auth.POST("/check-login", authMiddleware(), handleCheckLogin)                                                           // 验证登录状态
		auth.POST("/get-login-id", authMiddleware(), handleGetLoginID)                                                          // 获取登录ID
		auth.POST("/get-token-info", authMiddleware(), handleGetTokenInfo)                                                      // 获取Token信息
		auth.POST("/get-device", authMiddleware(), handleGetDevice)                                                             // 获取设备类型
		auth.POST("/get-device-id", authMiddleware(), handleGetDeviceId)                                                        // 获取设备ID
		auth.POST("/get-token-create-time", authMiddleware(), handleGetTokenCreateTime)                                         // 获取Token创建时间
		auth.POST("/get-token-ttl", authMiddleware(), handleGetTokenTTL)                                                        // 获取Token TTL
		auth.GET("/online-count/:loginId", authMiddleware(), handleGetOnlineTerminalCount)                                      // 获取在线终端总数
		auth.GET("/online-count/:loginId/:device", authMiddleware(), handleGetOnlineTerminalCountByDevice)                      // 获取指定设备在线终端数
		auth.GET("/online-count/:loginId/:device/:deviceId", authMiddleware(), handleGetOnlineTerminalCountByDeviceAndDeviceId) // 获取指定设备和设备ID在线终端数
	}

	// ========== 在线状态管理接口 ==========
	online := api.Group("/online", authMiddleware())
	{
		online.POST("/kickout", handleKickout)                                 // 根据Token踢人下线
		online.POST("/kickout-by-device", handleKickoutByDevice)               // 根据设备类型踢人下线
		online.POST("/kickout-by-device-id", handleKickoutByDeviceAndDeviceId) // 根据设备和设备ID踢人下线
		online.POST("/kickout-by-login-id", handleKickoutByLoginID)            // 根据LoginID踢出所有终端
		online.POST("/replace", handleReplace)                                 // 根据Token顶人下线
		online.POST("/replace-by-device", handleReplaceByDevice)               // 根据设备类型顶人下线
		online.POST("/replace-by-device-id", handleReplaceByDeviceAndDeviceId) // 根据设备和设备ID顶人下线
		online.POST("/replace-by-login-id", handleReplaceByLoginID)            // 根据LoginID顶替所有终端
	}

	// ========== 权限管理接口 ==========
	permission := api.Group("/permission", authMiddleware())
	{
		permission.POST("/add", handleAddPermissions)                        // 添加权限
		permission.POST("/add-by-token", handleAddPermissionsByToken)        // 根据Token添加权限
		permission.POST("/remove", handleRemovePermissions)                  // 删除权限
		permission.POST("/remove-by-token", handleRemovePermissionsByToken)  // 根据Token删除权限
		permission.GET("/list/:loginId", handleGetPermissions)               // 获取权限列表
		permission.POST("/list-by-token", handleGetPermissionsByToken)       // 根据Token获取权限列表
		permission.POST("/has", handleHasPermission)                         // 检查是否拥有指定权限
		permission.POST("/has-by-token", handleHasPermissionByToken)         // 根据Token检查权限
		permission.POST("/has-and", handleHasPermissionsAnd)                 // 检查是否拥有所有权限（AND）
		permission.POST("/has-and-by-token", handleHasPermissionsAndByToken) // 根据Token检查所有权限（AND）
		permission.POST("/has-or", handleHasPermissionsOr)                   // 检查是否拥有任一权限（OR）
		permission.POST("/has-or-by-token", handleHasPermissionsOrByToken)   // 根据Token检查任一权限（OR）
	}

	// ========== 角色管理接口 ==========
	role := api.Group("/role", authMiddleware())
	{
		role.POST("/add", handleAddRoles)                        // 添加角色
		role.POST("/add-by-token", handleAddRolesByToken)        // 根据Token添加角色
		role.POST("/remove", handleRemoveRoles)                  // 删除角色
		role.POST("/remove-by-token", handleRemoveRolesByToken)  // 根据Token删除角色
		role.GET("/list/:loginId", handleGetRoles)               // 获取角色列表
		role.POST("/list-by-token", handleGetRolesByToken)       // 根据Token获取角色列表
		role.POST("/has", handleHasRole)                         // 检查是否拥有指定角色
		role.POST("/has-by-token", handleHasRoleByToken)         // 根据Token检查角色
		role.POST("/has-and", handleHasRolesAnd)                 // 检查是否���有所有角色（AND）
		role.POST("/has-and-by-token", handleHasRolesAndByToken) // 根据Token检查所有角色（AND）
		role.POST("/has-or", handleHasRolesOr)                   // 检查是否拥有任一角色（OR）
		role.POST("/has-or-by-token", handleHasRolesOrByToken)   // 根据Token检查任一角色（OR）
	}

	// ========== Session 管理接口 ==========
	session := api.Group("/session", authMiddleware())
	{
		session.GET("/:loginId", handleGetSession)                               // 获取会话
		session.POST("/by-token", handleGetSessionByToken)                       // 根据Token获取会话
		session.GET("/tokens/:loginId", handleGetTokenValueListByLoginID)        // 获取所有Token
		session.GET("/tokens/:loginId/:device", handleGetTokenValueListByDevice) // 获取指定设备的所有Token
	}

	// ========== 账号封禁管理接口 ==========
	disable := api.Group("/disable", authMiddleware())
	{
		disable.POST("/ban", handleDisable)                   // 封禁账号
		disable.POST("/unban", handleUntie)                   // 解封账号
		disable.GET("/is-disabled/:loginId", handleIsDisable) // 检查是否被封禁
		disable.GET("/info/:loginId", handleGetDisableInfo)   // 获取封禁信息
		disable.GET("/ttl/:loginId", handleGetDisableTTL)     // 获取封禁TTL
	}

	fmt.Println("所有路由注册完成")
}

// ============================================================================
// Main Function - 主函数
// ============================================================================

func main() {
	fmt.Println("========================================")
	fmt.Println("DToken Quick Start - 完整测试示例")
	fmt.Println("========================================")

	// 初始化 DToken 框架
	if err := initDToken(); err != nil {
		fmt.Printf("初始化失败: %v\n", err)
		return
	}

	// 创建 Gin 引擎
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// 注册所有路由
	setupRoutes(r)

	// 启动服务器
	port := ":8080"
	fmt.Printf("\n服务器启动成功，监听端口: %s\n", port)
	fmt.Println("========================================")
	fmt.Println("API 接口分类:")
	fmt.Println("  - 认证接口: http://localhost:8080/api/auth/*")
	fmt.Println("  - 在线管理: http://localhost:8080/api/online/*")
	fmt.Println("  - 权限管理: http://localhost:8080/api/permission/*")
	fmt.Println("  - 角色管理: http://localhost:8080/api/role/*")
	fmt.Println("  - Session:  http://localhost:8080/api/session/*")
	fmt.Println("  - 账号封禁: http://localhost:8080/api/disable/*")
	fmt.Println("========================================")

	if err := r.Run(port); err != nil {
		fmt.Printf("服务器启动失败: %v\n", err)
	}
}
