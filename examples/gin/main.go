// @Author daixk 2025/12/22 15:56:00
package main

import (
	"context"
	"encoding/json"
	"errors"
	dterror "github.com/Zany2/dtoken-go/errors"
	dtglobal "github.com/Zany2/dtoken-go/global"
	"io"
	"net/http"
	"time"

	"github.com/Zany2/dtoken-go"
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
	Username string `json:"username"`
	// Password stores the demo login password. Password 保存示例登录密码。
	Password string `json:"password"`
}

// RefreshRequest defines the refresh-token payload RefreshRequest 定义刷新令牌请求参数
type RefreshRequest struct {
	// RefreshToken stores the refresh token to rotate. RefreshToken 保存待轮换的刷新令牌。
	RefreshToken string `json:"refreshToken"`
}

func main() {
	auth := initDToken()
	defer auth.Close()

	r := gin.Default()
	registerRoutes(r, auth)

	if err := r.Run(":8080"); err != nil {
		panic(err)
	}
}

// registerRoutes shares production middleware and routes with regression tests. registerRoutes 与回归测试共用实际中间件和路由。
func registerRoutes(r *gin.Engine, instances ...*dtoken.Auth) {
	ctx := context.Background()
	options := []gindt.AuthOption{gindt.WithFailFunc(writeAuthError), gindt.WithManagerResolver(dtglobal.GetManager)}
	if len(instances) > 0 && instances[0] != nil {
		options = append(options, gindt.WithAuth(instances[0]))
	}
	r.Use(gindt.RegisterDTokenContextMiddleware(ctx, options...))
	r.POST("/login", handleLogin)
	r.POST("/refresh", handleRefresh)

	auth := r.Group("/")
	auth.Use(gindt.AuthMiddleware(ctx, options...))
	auth.GET("/me", handleMe)
	auth.GET("/introspect", handleIntrospect)
	auth.GET("/admin", gindt.RoleMiddleware(ctx, []string{"admin"}, options...), handleAdmin)
	auth.GET("/articles", gindt.PermissionMiddleware(ctx, []string{"article:read"}, options...), handleArticles)
	auth.POST("/logout", handleLogout)
}

// initDToken initializes integration manager initDToken 初始化集成管理器
func initDToken() *dtoken.Auth {
	auth, err := dtoken.NewBuilder().
		Timeout(int64((2 * time.Hour).Seconds())).
		RenewMaxRefresh(int64(time.Hour.Seconds())).
		RefreshTokenTimeout(int64((30 * 24 * time.Hour).Seconds())).
		IsPrintBanner(false).
		Build()
	if err != nil {
		panic(err)
	}

	dtglobal.SetAuth(auth)
	return auth
}

// handleLogin logs in a demo user handleLogin 登录示例用户
func handleLogin(c *gin.Context) {
	var req LoginRequest
	decoder := json.NewDecoder(c.Request.Body)
	if err := decoder.Decode(&req); err != nil || req.Username == "" || req.Password == "" {
		writeJSON(c, http.StatusBadRequest, dterror.CodeBadRequest, "username and password are required", nil)
		return
	}

	// Reject trailing input before issuing credentials. 签发凭证前拒绝多余输入。
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		writeJSON(c, http.StatusBadRequest, dterror.CodeBadRequest, "request body must contain a single JSON object", nil)
		return
	}

	if req.Password != "123456" {
		writeJSON(c, http.StatusUnauthorized, dterror.CodeNotLogin, "invalid username or password", nil)
		return
	}

	mgr, err := gindt.GetManagerByContext(c)
	if err != nil {
		writeAuthError(c, err)
		return
	}
	pair, err := mgr.LoginWithRefreshToken(c.Request.Context(), req.Username, "web", "gin-example")
	if err != nil {
		writeAuthError(c, err)
		return
	}

	// Grant every demo user the same role and permission; real apps must load their own access rules. 为每个演示用户授予相同角色和权限；实际应用应加载自身的授权规则。
	if err = mgr.AddRoles(c.Request.Context(), req.Username, []string{"admin"}); err != nil {
		writeAuthError(c, err)
		return
	}
	if err = mgr.AddPermissions(c.Request.Context(), req.Username, []string{"article:read"}); err != nil {
		writeAuthError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, dterror.CodeSuccess, "ok", pair)
}

// handleRefresh rotates refresh token handleRefresh 轮换刷新令牌
func handleRefresh(c *gin.Context) {
	var req RefreshRequest
	decoder := json.NewDecoder(c.Request.Body)
	if err := decoder.Decode(&req); err != nil || req.RefreshToken == "" {
		writeJSON(c, http.StatusBadRequest, dterror.CodeBadRequest, "refreshToken is required", nil)
		return
	}

	// Validate the full body before consuming a refresh token. 消耗刷新令牌前校验完整请求体。
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		writeJSON(c, http.StatusBadRequest, dterror.CodeBadRequest, "request body must contain a single JSON object", nil)
		return
	}

	pair, err := gindt.RefreshTokenByContext(c, req.RefreshToken)
	if err != nil {
		writeAuthError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, dterror.CodeSuccess, "ok", pair)
}

// handleMe returns current login information handleMe 返回当前登录信息
func handleMe(c *gin.Context) {
	dCtx, ok := gindt.GetDTokenContext(c)
	if !ok {
		writeJSON(c, http.StatusUnauthorized, dterror.CodeNotLogin, "not logged in", nil)
		return
	}

	loginID, err := dCtx.Auth().GetLoginID(c.Request.Context())
	if err != nil {
		writeAuthError(c, err)
		return
	}

	roles, err := dCtx.Access().GetRoles(c.Request.Context())
	if err != nil {
		writeAuthError(c, err)
		return
	}
	permissions, err := dCtx.Access().GetPermissions(c.Request.Context())
	if err != nil {
		writeAuthError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, dterror.CodeSuccess, "ok", gin.H{
		"loginId":     loginID,
		"roles":       roles,
		"permissions": permissions,
	})
}

// handleIntrospect returns current token introspection handleIntrospect 返回当前 token 自省结果
func handleIntrospect(c *gin.Context) {
	info, err := gindt.IntrospectTokenByContext(c)
	if err != nil {
		writeAuthError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, dterror.CodeSuccess, "ok", info)
}

// handleAdmin returns admin data handleAdmin 返回管理员数据
func handleAdmin(c *gin.Context) {
	writeJSON(c, http.StatusOK, dterror.CodeSuccess, "ok", gin.H{"scope": "admin"})
}

// handleArticles returns protected article data handleArticles 返回受保护的文章数据
func handleArticles(c *gin.Context) {
	writeJSON(c, http.StatusOK, dterror.CodeSuccess, "ok", []string{"article-a", "article-b"})
}

// handleLogout logs out current token handleLogout 注销当前 Token
func handleLogout(c *gin.Context) {
	dCtx, ok := gindt.GetDTokenContext(c)
	if !ok {
		writeJSON(c, http.StatusUnauthorized, dterror.CodeNotLogin, "not logged in", nil)
		return
	}

	if err := dCtx.Auth().Logout(c.Request.Context()); err != nil {
		writeAuthError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, dterror.CodeSuccess, "ok", nil)
}

// writeAuthError distinguishes credential failures from access restrictions and server failures. writeAuthError 区分凭证无效、访问限制及服务端故障。
func writeAuthError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, dterror.ErrNotLogin), errors.Is(err, dterror.ErrInvalidToken),
		errors.Is(err, dterror.ErrInvalidRefreshToken), errors.Is(err, dterror.ErrTokenExpired),
		errors.Is(err, dterror.ErrActiveTimeout), errors.Is(err, dterror.ErrTokenKickout),
		errors.Is(err, dterror.ErrTokenReplaced):
		writeJSON(c, http.StatusUnauthorized, dterror.CodeNotLogin, err.Error(), nil)
	case errors.Is(err, dterror.ErrAccountDisabled), errors.Is(err, dterror.ErrDeviceDisabled):
		writeJSON(c, http.StatusForbidden, dterror.CodeAccountDisabled, err.Error(), nil)
	case errors.Is(err, dterror.ErrPermissionDenied), errors.Is(err, dterror.ErrRoleDenied):
		writeJSON(c, http.StatusForbidden, dterror.CodePermissionDenied, err.Error(), nil)
	default:
		// Record server details for Gin's logger without exposing them in HTTP responses. 保留服务端详情供 Gin 日志记录，不在 HTTP 响应中暴露。
		c.Error(err)
		writeJSON(c, http.StatusInternalServerError, dterror.CodeServerError, "internal server error", nil)
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
