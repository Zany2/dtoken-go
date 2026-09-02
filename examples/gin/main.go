// @Author daixk 2025/12/22 15:56:00
package main

import (
	"context"
	"net/http"
	"time"

	gindt "github.com/Zany2/dtoken-go/integrations/gin"
	"github.com/gin-gonic/gin"
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
	Username string `json:"username" binding:"required"`
	// Password stores the demo login password. Password 保存示例登录密码。
	Password string `json:"password" binding:"required"`
}

// RefreshRequest defines the refresh-token payload RefreshRequest 定义刷新令牌请求参数
type RefreshRequest struct {
	// RefreshToken stores the refresh token to rotate. RefreshToken 保存待轮换的刷新令牌。
	RefreshToken string `json:"refreshToken" binding:"required"`
}

func main() {
	ctx := context.Background()
	initDToken()

	r := gin.Default()
	r.Use(gindt.RegisterDTokenContextMiddleware(ctx))
	r.POST("/login", handleLogin)
	r.POST("/refresh", handleRefresh)

	auth := r.Group("/")
	auth.Use(gindt.AuthMiddleware(ctx))
	auth.GET("/me", handleMe)
	auth.GET("/introspect", handleIntrospect)
	auth.GET("/admin", gindt.RoleMiddleware(ctx, []string{"admin"}), handleAdmin)
	auth.GET("/articles", gindt.PermissionMiddleware(ctx, []string{"article:read"}), handleArticles)
	auth.POST("/logout", handleLogout)

	if err := r.Run(":8080"); err != nil {
		panic(err)
	}
}

// initDToken initializes integration manager initDToken 初始化集成管理器
func initDToken() {
	mgr, err := gindt.NewBuilder().
		Timeout(int64((2 * time.Hour).Seconds())).
		RefreshTokenTimeout(int64((30 * 24 * time.Hour).Seconds())).
		IsPrintBanner(false).
		Build()
	if err != nil {
		panic(err)
	}

	gindt.SetManager(mgr)
}

// handleLogin logs in a demo user handleLogin 登录示例用户
func handleLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeJSON(c, http.StatusBadRequest, gindt.CodeBadRequest, "username and password are required", nil)
		return
	}

	if req.Password != "123456" {
		writeJSON(c, http.StatusUnauthorized, gindt.CodeNotLogin, "invalid username or password", nil)
		return
	}

	pair, err := gindt.LoginWithRefreshToken(c.Request.Context(), req.Username, "web", "gin-example")
	if err != nil {
		writeJSON(c, http.StatusInternalServerError, gindt.CodeServerError, err.Error(), nil)
		return
	}

	// Seed demo authorization data 初始化示例权限数据
	if err = gindt.AddRoles(c.Request.Context(), req.Username, []string{"admin"}); err != nil {
		writeJSON(c, http.StatusInternalServerError, gindt.CodeServerError, err.Error(), nil)
		return
	}
	if err = gindt.AddPermissions(c.Request.Context(), req.Username, []string{"article:read"}); err != nil {
		writeJSON(c, http.StatusInternalServerError, gindt.CodeServerError, err.Error(), nil)
		return
	}

	writeJSON(c, http.StatusOK, gindt.CodeSuccess, "ok", pair)
}

// handleRefresh rotates refresh token handleRefresh 轮换刷新令牌
func handleRefresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeJSON(c, http.StatusBadRequest, gindt.CodeBadRequest, "refreshToken is required", nil)
		return
	}

	pair, err := gindt.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		writeJSON(c, http.StatusUnauthorized, gindt.CodeNotLogin, err.Error(), nil)
		return
	}

	writeJSON(c, http.StatusOK, gindt.CodeSuccess, "ok", pair)
}

// handleMe returns current login information handleMe 返回当前登录信息
func handleMe(c *gin.Context) {
	dCtx, ok := gindt.GetDTokenContext(c)
	if !ok {
		writeJSON(c, http.StatusUnauthorized, gindt.CodeNotLogin, "not logged in", nil)
		return
	}

	loginID, err := dCtx.Auth().GetLoginID(c.Request.Context())
	if err != nil {
		writeJSON(c, http.StatusUnauthorized, gindt.CodeNotLogin, err.Error(), nil)
		return
	}

	roles, err := dCtx.Access().GetRoles(c.Request.Context())
	if err != nil {
		writeJSON(c, http.StatusInternalServerError, gindt.CodeServerError, err.Error(), nil)
		return
	}
	permissions, err := dCtx.Access().GetPermissions(c.Request.Context())
	if err != nil {
		writeJSON(c, http.StatusInternalServerError, gindt.CodeServerError, err.Error(), nil)
		return
	}

	writeJSON(c, http.StatusOK, gindt.CodeSuccess, "ok", gin.H{
		"loginId":     loginID,
		"roles":       roles,
		"permissions": permissions,
	})
}

// handleIntrospect returns current token introspection handleIntrospect 返回当前 token 自省结果
func handleIntrospect(c *gin.Context) {
	info, err := gindt.IntrospectTokenByContext(c)
	if err != nil {
		writeJSON(c, http.StatusUnauthorized, gindt.CodeNotLogin, err.Error(), nil)
		return
	}

	writeJSON(c, http.StatusOK, gindt.CodeSuccess, "ok", info)
}

// handleAdmin returns admin data handleAdmin 返回管理员数据
func handleAdmin(c *gin.Context) {
	writeJSON(c, http.StatusOK, gindt.CodeSuccess, "ok", gin.H{"scope": "admin"})
}

// handleArticles returns protected article data handleArticles 返回受保护的文章数据
func handleArticles(c *gin.Context) {
	writeJSON(c, http.StatusOK, gindt.CodeSuccess, "ok", []string{"article-a", "article-b"})
}

// handleLogout logs out current token handleLogout 注销当前 Token
func handleLogout(c *gin.Context) {
	dCtx, ok := gindt.GetDTokenContext(c)
	if !ok {
		writeJSON(c, http.StatusUnauthorized, gindt.CodeNotLogin, "not logged in", nil)
		return
	}

	if err := dCtx.Auth().Logout(c.Request.Context()); err != nil {
		writeJSON(c, http.StatusInternalServerError, gindt.CodeServerError, err.Error(), nil)
		return
	}

	writeJSON(c, http.StatusOK, gindt.CodeSuccess, "ok", nil)
}

// writeJSON writes a unified JSON response writeJSON 写入统一 JSON 响应
func writeJSON(c *gin.Context, httpStatus int, code int, message string, data interface{}) {
	c.JSON(httpStatus, Response{
		Code:    code,
		Message: message,
		Data:    data,
	})
}
