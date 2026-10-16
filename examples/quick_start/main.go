// @Author daixk 2025/12/22 15:56:00
package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/defaults"
	"github.com/Zany2/dtoken-go/dtoken"
	"github.com/gin-gonic/gin"
)

const (
	// tokenHeader stores the header used by this example tokenHeader 保存本示例读取的 Token 请求头
	tokenHeader = "Authorization"
)

// Response defines the example response body Response 定义示例响应结构
type Response struct {
	// Code stores the application response code. Code 保存应用响应码。
	Code int `json:"code"`
	// Message stores the response message. Message 保存响应消息。
	Message string `json:"message"`
	// Data stores the optional response payload. Data 保存可选响应数据。
	Data interface{} `json:"data,omitempty"`
}

// LoginRequest defines the login payload LoginRequest 定义登录请求参数
type LoginRequest struct {
	// Username stores the demo login name. Username 保存示例登录名。
	Username string `json:"username"`
	// Password stores the demo login password. Password 保存示例登录密码。
	Password string `json:"password"`
}

func main() {
	initDToken()
	defer dtoken.DeleteAllManager()

	r := gin.Default()
	registerRoutes(r)

	if err := r.Run(":8080"); err != nil {
		panic(err)
	}
}

// registerRoutes shares production routes with regression tests. registerRoutes 与回归测试共用实际路由。
func registerRoutes(r *gin.Engine) {
	r.POST("/login", handleLogin)

	auth := r.Group("/")
	auth.Use(authMiddleware())
	auth.GET("/me", handleMe)
	auth.GET("/admin", roleMiddleware("admin"), handleAdmin)
	auth.GET("/articles", permissionMiddleware("article:read"), handleArticles)
	auth.POST("/logout", handleLogout)
}

// initDToken initializes DToken with bundled memory storage initDToken 使用内置内存存储初始化 DToken
func initDToken() {
	mgr, err := defaults.NewBuilder().
		TokenName(tokenHeader).
		Timeout(int64((2 * time.Hour).Seconds())).
		RenewMaxRefresh(int64(time.Hour.Seconds())).
		IsPrintBanner(false).
		Build()
	if err != nil {
		panic(err)
	}

	dtoken.SetManager(mgr)
}

// handleLogin logs in a demo user handleLogin 登录示例用户
func handleLogin(c *gin.Context) {
	var req LoginRequest
	decoder := json.NewDecoder(c.Request.Body)
	if err := decoder.Decode(&req); err != nil || req.Username == "" || req.Password == "" {
		writeJSON(c, http.StatusBadRequest, 400, "username and password are required", nil)
		return
	}

	// Reject trailing input before creating login state. 创建登录态前拒绝多余输入。
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		writeJSON(c, http.StatusBadRequest, 400, "request body must contain a single JSON object", nil)
		return
	}

	if req.Password != "123456" {
		writeJSON(c, http.StatusUnauthorized, 401, "invalid username or password", nil)
		return
	}

	token, err := dtoken.Login(c.Request.Context(), req.Username)
	if err != nil {
		writeAuthError(c, err)
		return
	}

	// Grant every demo user the same access rules. 为所有示例用户授予相同权限，仅用于演示鉴权流程。
	if err = dtoken.AddRoles(c.Request.Context(), req.Username, []string{"admin"}); err != nil {
		writeAuthError(c, err)
		return
	}
	if err = dtoken.AddPermissions(c.Request.Context(), req.Username, []string{"article:read"}); err != nil {
		writeAuthError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, 0, "ok", gin.H{
		"token":       token,
		"tokenHeader": tokenHeader,
	})
}

// handleMe returns current login information handleMe 返回当前登录信息
func handleMe(c *gin.Context) {
	token := currentToken(c)
	loginID, err := dtoken.GetLoginID(c.Request.Context(), token)
	if err != nil {
		writeAuthError(c, err)
		return
	}

	roles, err := dtoken.GetRoles(c.Request.Context(), loginID)
	if err != nil {
		writeAuthError(c, err)
		return
	}
	permissions, err := dtoken.GetPermissions(c.Request.Context(), loginID)
	if err != nil {
		writeAuthError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, 0, "ok", gin.H{
		"loginId":     loginID,
		"roles":       roles,
		"permissions": permissions,
	})
}

// handleAdmin returns admin data handleAdmin 返回管理员数据
func handleAdmin(c *gin.Context) {
	writeJSON(c, http.StatusOK, 0, "ok", gin.H{"scope": "admin"})
}

// handleArticles returns protected article data handleArticles 返回受保护的文章数据
func handleArticles(c *gin.Context) {
	writeJSON(c, http.StatusOK, 0, "ok", []string{"article-a", "article-b"})
}

// handleLogout logs out current token handleLogout 注销当前 Token
func handleLogout(c *gin.Context) {
	if err := dtoken.Logout(c.Request.Context(), currentToken(c)); err != nil {
		writeAuthError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, 0, "ok", nil)
}

// authMiddleware checks login state authMiddleware 校验登录状态
func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := dtoken.CheckLogin(c.Request.Context(), currentToken(c)); err != nil {
			writeAuthError(c, err)
			c.Abort()
			return
		}

		c.Next()
	}
}

// roleMiddleware checks current user role roleMiddleware 校验当前用户角色
func roleMiddleware(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		loginID, err := dtoken.GetLoginID(ctx, currentToken(c))
		if err != nil {
			writeAuthError(c, err)
			c.Abort()
			return
		}

		if !dtoken.HasRole(ctx, loginID, role) {
			writeJSON(c, http.StatusForbidden, 403, "role denied", nil)
			c.Abort()
			return
		}

		c.Next()
	}
}

// permissionMiddleware checks current user permission permissionMiddleware 校验当前用户权限
func permissionMiddleware(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		loginID, err := dtoken.GetLoginID(ctx, currentToken(c))
		if err != nil {
			writeAuthError(c, err)
			c.Abort()
			return
		}

		if !dtoken.HasPermission(ctx, loginID, permission) {
			writeJSON(c, http.StatusForbidden, 403, "permission denied", nil)
			c.Abort()
			return
		}

		c.Next()
	}
}

// currentToken reads token from request header currentToken 从请求头读取 Token
func currentToken(c *gin.Context) string {
	return c.GetHeader(tokenHeader)
}

// writeAuthError preserves failure categories without exposing internal details. writeAuthError 保留错误分类，不暴露内部详情。
func writeAuthError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, derror.ErrNotLogin), errors.Is(err, derror.ErrInvalidToken),
		errors.Is(err, derror.ErrTokenExpired), errors.Is(err, derror.ErrActiveTimeout),
		errors.Is(err, derror.ErrTokenKickout), errors.Is(err, derror.ErrTokenReplaced):
		writeJSON(c, http.StatusUnauthorized, 401, "not logged in or token invalid", nil)
	case errors.Is(err, derror.ErrAccountDisabled), errors.Is(err, derror.ErrDeviceDisabled):
		writeJSON(c, http.StatusForbidden, 403, "account or device disabled", nil)
	case errors.Is(err, derror.ErrPermissionDenied), errors.Is(err, derror.ErrRoleDenied):
		writeJSON(c, http.StatusForbidden, 403, "permission denied", nil)
	default:
		c.Error(err)
		writeJSON(c, http.StatusInternalServerError, 500, "internal server error", nil)
	}
}

// writeJSON writes a unified JSON response writeJSON 写入统一 JSON 响应
func writeJSON(c *gin.Context, httpStatus int, code int, message string, data interface{}) {
	c.JSON(httpStatus, Response{
		Code:    code,
		Message: message,
		Data:    data,
	})
}
