// @Author daixk 2025/12/22 15:56:00
package main

import (
	"context"
	"net/http"
	"time"

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
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// LoginRequest defines the login payload LoginRequest 定义登录请求参数
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func main() {
	ctx := context.Background()
	initDToken()

	r := gin.Default()
	r.POST("/login", handleLogin)

	auth := r.Group("/")
	auth.Use(authMiddleware(ctx))
	auth.GET("/me", handleMe)
	auth.GET("/admin", roleMiddleware(ctx, "admin"), handleAdmin)
	auth.GET("/articles", permissionMiddleware(ctx, "article:read"), handleArticles)
	auth.POST("/logout", handleLogout)

	_ = r.Run(":8080")
}

// initDToken initializes DToken with bundled memory storage initDToken 使用内置内存存储初始化 DToken
func initDToken() {
	mgr, err := defaults.NewBuilder().
		TokenName(tokenHeader).
		Timeout(int64((2 * time.Hour).Seconds())).
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
	if err := c.ShouldBindJSON(&req); err != nil {
		writeJSON(c, http.StatusBadRequest, 400, "username and password are required", nil)
		return
	}

	if req.Password != "123456" {
		writeJSON(c, http.StatusUnauthorized, 401, "invalid username or password", nil)
		return
	}

	token, err := dtoken.Login(c.Request.Context(), req.Username)
	if err != nil {
		writeJSON(c, http.StatusInternalServerError, 500, err.Error(), nil)
		return
	}

	// Seed demo authorization data 初始化示例权限数据
	_ = dtoken.AddRoles(c.Request.Context(), req.Username, []string{"admin"})
	_ = dtoken.AddPermissions(c.Request.Context(), req.Username, []string{"article:read"})

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
		writeJSON(c, http.StatusUnauthorized, 401, err.Error(), nil)
		return
	}

	roles, _ := dtoken.GetRoles(c.Request.Context(), loginID)
	permissions, _ := dtoken.GetPermissions(c.Request.Context(), loginID)

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
		writeJSON(c, http.StatusInternalServerError, 500, err.Error(), nil)
		return
	}

	writeJSON(c, http.StatusOK, 0, "ok", nil)
}

// authMiddleware checks login state authMiddleware 校验登录状态
func authMiddleware(ctx context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := dtoken.CheckLogin(ctx, currentToken(c)); err != nil {
			writeJSON(c, http.StatusUnauthorized, 401, err.Error(), nil)
			c.Abort()
			return
		}

		c.Next()
	}
}

// roleMiddleware checks current user role roleMiddleware 校验当前用户角色
func roleMiddleware(ctx context.Context, role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		loginID, err := dtoken.GetLoginID(ctx, currentToken(c))
		if err != nil || !dtoken.HasRole(ctx, loginID, role) {
			writeJSON(c, http.StatusForbidden, 403, "role denied", nil)
			c.Abort()
			return
		}

		c.Next()
	}
}

// permissionMiddleware checks current user permission permissionMiddleware 校验当前用户权限
func permissionMiddleware(ctx context.Context, permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		loginID, err := dtoken.GetLoginID(ctx, currentToken(c))
		if err != nil || !dtoken.HasPermission(ctx, loginID, permission) {
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

// writeJSON writes a unified JSON response writeJSON 写入统一 JSON 响应
func writeJSON(c *gin.Context, httpStatus int, code int, message string, data interface{}) {
	c.JSON(httpStatus, Response{
		Code:    code,
		Message: message,
		Data:    data,
	})
}
