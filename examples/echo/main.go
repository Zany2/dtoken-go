// @Author daixk 2025/12/22 15:56:00
package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"time"

	echodt "github.com/Zany2/dtoken-go/integrations/echo"
	echo4 "github.com/labstack/echo/v4"
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
	defer echodt.DeleteAllManager()

	e := echo4.New()
	registerRoutes(e)

	if err := e.Start(":8080"); err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
}

// registerRoutes shares production middleware and routes with regression tests. registerRoutes 与回归测试共用实际中间件和路由。
func registerRoutes(e *echo4.Echo) {
	ctx := context.Background()
	e.Use(echodt.RegisterDTokenContextMiddleware(ctx, echodt.WithFailFunc(writeAuthError)))
	e.POST("/login", handleLogin)

	auth := e.Group("")
	auth.Use(echodt.AuthMiddleware(ctx, echodt.WithFailFunc(writeAuthError)))
	auth.GET("/me", handleMe)
	auth.GET("/admin", handleAdmin, echodt.RoleMiddleware(ctx, []string{"admin"}, echodt.WithFailFunc(writeAuthError)))
	auth.GET("/articles", handleArticles, echodt.PermissionMiddleware(ctx, []string{"article:read"}, echodt.WithFailFunc(writeAuthError)))
	auth.POST("/logout", handleLogout)
}

// initDToken initializes integration manager initDToken 初始化集成管理器
func initDToken() {
	mgr, err := echodt.NewBuilder().
		Timeout(int64((2 * time.Hour).Seconds())).
		RenewMaxRefresh(int64(time.Hour.Seconds())).
		IsPrintBanner(false).
		Build()
	if err != nil {
		panic(err)
	}

	echodt.SetManager(mgr)
}

// handleLogin logs in a demo user handleLogin 登录示例用户
func handleLogin(c echo4.Context) error {
	var req LoginRequest
	var bindErr error
	mediaType, _, _ := mime.ParseMediaType(c.Request().Header.Get(echo4.HeaderContentType))
	if mediaType == echo4.MIMEApplicationJSON {
		// Validate the entire JSON body before login can change state. 登录改变状态前校验完整 JSON 请求体。
		decoder := json.NewDecoder(c.Request().Body)
		if bindErr = decoder.Decode(&req); bindErr == nil {
			var extra any
			if err := decoder.Decode(&extra); err != io.EOF {
				return writeJSON(c, http.StatusBadRequest, echodt.CodeBadRequest, "request body must contain a single JSON object", nil)
			}
		}
	} else {
		// Preserve Echo's binding behavior for other media types. 保留 Echo 对其他媒体类型的绑定行为。
		bindErr = c.Bind(&req)
	}
	if bindErr != nil || req.Username == "" || req.Password == "" {
		return writeJSON(c, http.StatusBadRequest, echodt.CodeBadRequest, "username and password are required", nil)
	}

	if req.Password != "123456" {
		return writeJSON(c, http.StatusUnauthorized, echodt.CodeNotLogin, "invalid username or password", nil)
	}

	token, err := echodt.Login(c.Request().Context(), req.Username)
	if err != nil {
		return writeAuthError(c, err)
	}

	// Grant every demo user the same role and permission; real apps must load their own access rules. 为每个演示用户授予相同角色和权限；实际应用应加载自身的授权规则。
	if err = echodt.AddRoles(c.Request().Context(), req.Username, []string{"admin"}); err != nil {
		return writeAuthError(c, err)
	}
	if err = echodt.AddPermissions(c.Request().Context(), req.Username, []string{"article:read"}); err != nil {
		return writeAuthError(c, err)
	}

	return writeJSON(c, http.StatusOK, echodt.CodeSuccess, "ok", echo4.Map{"token": token})
}

// handleMe returns current login information handleMe 返回当前登录信息
func handleMe(c echo4.Context) error {
	dCtx, ok := echodt.GetDTokenContext(c)
	if !ok {
		return writeJSON(c, http.StatusUnauthorized, echodt.CodeNotLogin, "not logged in", nil)
	}

	loginID, err := dCtx.Auth().GetLoginID(c.Request().Context())
	if err != nil {
		return writeAuthError(c, err)
	}

	roles, err := dCtx.Access().GetRoles(c.Request().Context())
	if err != nil {
		return writeAuthError(c, err)
	}
	permissions, err := dCtx.Access().GetPermissions(c.Request().Context())
	if err != nil {
		return writeAuthError(c, err)
	}

	return writeJSON(c, http.StatusOK, echodt.CodeSuccess, "ok", echo4.Map{
		"loginId":     loginID,
		"roles":       roles,
		"permissions": permissions,
	})
}

// handleAdmin returns admin data handleAdmin 返回管理员数据
func handleAdmin(c echo4.Context) error {
	return writeJSON(c, http.StatusOK, echodt.CodeSuccess, "ok", echo4.Map{"scope": "admin"})
}

// handleArticles returns protected article data handleArticles 返回受保护的文章数据
func handleArticles(c echo4.Context) error {
	return writeJSON(c, http.StatusOK, echodt.CodeSuccess, "ok", []string{"article-a", "article-b"})
}

// handleLogout logs out current token handleLogout 注销当前 Token
func handleLogout(c echo4.Context) error {
	dCtx, ok := echodt.GetDTokenContext(c)
	if !ok {
		return writeJSON(c, http.StatusUnauthorized, echodt.CodeNotLogin, "not logged in", nil)
	}

	if err := dCtx.Auth().Logout(c.Request().Context()); err != nil {
		return writeAuthError(c, err)
	}

	return writeJSON(c, http.StatusOK, echodt.CodeSuccess, "ok", nil)
}

// writeAuthError distinguishes credential failures from access restrictions and server failures. writeAuthError 区分凭证无效、访问限制及服务端故障。
func writeAuthError(c echo4.Context, err error) error {
	switch {
	case errors.Is(err, echodt.ErrNotLogin), errors.Is(err, echodt.ErrInvalidToken),
		errors.Is(err, echodt.ErrTokenExpired), errors.Is(err, echodt.ErrActiveTimeout),
		errors.Is(err, echodt.ErrTokenKickout), errors.Is(err, echodt.ErrTokenReplaced):
		return writeJSON(c, http.StatusUnauthorized, echodt.CodeNotLogin, err.Error(), nil)
	case errors.Is(err, echodt.ErrAccountDisabled), errors.Is(err, echodt.ErrDeviceDisabled):
		return writeJSON(c, http.StatusForbidden, echodt.CodeAccountDisabled, err.Error(), nil)
	case errors.Is(err, echodt.ErrPermissionDenied), errors.Is(err, echodt.ErrRoleDenied):
		return writeJSON(c, http.StatusForbidden, echodt.CodePermissionDenied, err.Error(), nil)
	default:
		// Keep server details in logs rather than HTTP responses. 将服务端详情保留在日志中，不在 HTTP 响应中暴露。
		c.Logger().Errorf("dtoken request failed: %v", err)
		return writeJSON(c, http.StatusInternalServerError, echodt.CodeServerError, "internal server error", nil)
	}
}

// writeJSON writes a unified JSON response writeJSON 写入统一 JSON 响应
func writeJSON(c echo4.Context, httpStatus int, code int, message string, data interface{}) error {
	return c.JSON(httpStatus, Response{
		Code:    code,
		Message: message,
		Data:    data,
	})
}
