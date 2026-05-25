package gin_core_app

import (
	"errors"
	"net/http"
	"time"

	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/dtoken"
	"github.com/gin-gonic/gin"
)

// App owns the Gin router and core auth facade. App 持有 Gin 路由和核心认证门面。
type App struct {
	auth   *dtoken.Auth
	router *gin.Engine
}

// Config controls the demo app auth timings. Config 控制示例应用认证时间配置。
type Config struct {
	TokenTimeout    time.Duration
	ActiveTimeout   int64
	RenewInterval   int64
	RenewMaxRefresh int64
}

// Response is the unified HTTP response body. Response 是统一 HTTP 响应体。
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Device   string `json:"device"`
	DeviceID string `json:"deviceId"`
}

type accessRequest struct {
	Value string `json:"value"`
}

type renewRequest struct {
	Seconds int64 `json:"seconds"`
}

type disableRequest struct {
	Reason string `json:"reason"`
}

type nonceRequest struct {
	Nonce string `json:"nonce"`
}

// NewApp creates a runnable Gin demo app. NewApp 创建可运行的 Gin 示例应用。
func NewApp(cfg Config) (*App, error) {
	gin.SetMode(gin.ReleaseMode)
	mgr, err := dtoken.NewBuilder().
		AuthType("gin-core-flow").
		TokenName("Authorization").
		TimeoutDuration(defaultDuration(cfg.TokenTimeout, 30*time.Second)).
		AutoRenew(false).
		RenewInterval(defaultLimit(cfg.RenewInterval)).
		RenewMaxRefresh(defaultLimit(cfg.RenewMaxRefresh)).
		ActiveTimeout(cfg.ActiveTimeout).
		IsPrintBanner(false).
		IsLog(false).
		AsyncEvent(false).
		Build()
	if err != nil {
		return nil, err
	}

	app := &App{auth: dtoken.New(mgr)}
	app.router = app.buildRouter()
	return app, nil
}

// MustNewApp creates an app or panics. MustNewApp 创建应用，失败时 panic。
func MustNewApp(cfg Config) *App {
	app, err := NewApp(cfg)
	if err != nil {
		panic(err)
	}
	return app
}

// Router returns the Gin engine for tests or servers. Router 返回 Gin 引擎用于测试或服务启动。
func (a *App) Router() http.Handler {
	return a.router
}

// Engine returns the concrete Gin engine. Engine 返回具体的 Gin 引擎。
func (a *App) Engine() *gin.Engine {
	return a.router
}

// Close releases auth manager resources. Close 释放认证管理器资源。
func (a *App) Close() {
	if a == nil || a.auth == nil {
		return
	}
	if mgr := a.auth.Manager(); mgr != nil {
		mgr.CloseManager()
	}
}

func (a *App) buildRouter() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/health", func(c *gin.Context) {
		writeOK(c, gin.H{"status": "ok"})
	})
	r.POST("/login", a.handleLogin)
	r.GET("/nonce", a.handleNonce)
	r.POST("/nonce/verify", a.handleNonceVerify)

	api := r.Group("/api")
	api.Use(a.authMiddleware())
	api.GET("/me", a.handleMe)
	api.POST("/logout", a.handleLogout)
	api.GET("/token/ttl", a.handleTokenTTL)
	api.POST("/token/renew", a.handleTokenRenew)
	api.GET("/session", a.handleSession)
	api.GET("/articles", a.requirePermission("article:read"), func(c *gin.Context) {
		writeOK(c, []string{"article-a", "article-b"})
	})
	api.GET("/admin", a.requireRole("admin"), func(c *gin.Context) {
		writeOK(c, gin.H{"scope": "admin"})
	})
	api.GET("/payment", a.requireService("payment", 1), func(c *gin.Context) {
		writeOK(c, gin.H{"status": "paid"})
	})
	api.POST("/permissions", a.handleAddPermission)
	api.POST("/roles", a.handleAddRole)
	api.POST("/disable/account", a.handleDisableAccount)
	api.POST("/disable/service/:service", a.handleDisableService)

	return r
}

func (a *App) handleLogin(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Username == "" || req.Password == "" {
		writeError(c, http.StatusBadRequest, derror.CodeBadRequest, "username and password are required")
		return
	}
	if req.Password != "123456" {
		writeError(c, http.StatusUnauthorized, derror.CodeNotLogin, "invalid username or password")
		return
	}

	token, err := a.auth.Login(c.Request.Context(), dtoken.LoginOptions{
		LoginID:  req.Username,
		Device:   req.Device,
		DeviceID: req.DeviceID,
	})
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, gin.H{"token": token})
}

func (a *App) handleMe(c *gin.Context) {
	token := tokenFromContext(c)
	loginID, err := a.auth.GetLoginID(c.Request.Context(), token)
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	roles, _ := a.auth.GetRoles(c.Request.Context(), loginID)
	permissions, _ := a.auth.GetPermissions(c.Request.Context(), loginID)
	writeOK(c, gin.H{
		"loginId":     loginID,
		"roles":       roles,
		"permissions": permissions,
	})
}

func (a *App) handleLogout(c *gin.Context) {
	if err := a.auth.LogoutByToken(c.Request.Context(), tokenFromContext(c)); err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, nil)
}

func (a *App) handleTokenTTL(c *gin.Context) {
	ttl, err := a.auth.GetTokenTTL(c.Request.Context(), tokenFromContext(c))
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, gin.H{"ttl": ttl})
}

func (a *App) handleTokenRenew(c *gin.Context) {
	var req renewRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Seconds <= 0 {
		writeError(c, http.StatusBadRequest, derror.CodeBadRequest, "seconds must be positive")
		return
	}
	if err := a.auth.RenewTimeout(c.Request.Context(), tokenFromContext(c), time.Duration(req.Seconds)*time.Second); err != nil {
		writeDTokenError(c, err)
		return
	}
	a.handleTokenTTL(c)
}

func (a *App) handleSession(c *gin.Context) {
	sess, err := a.auth.GetSessionByToken(c.Request.Context(), tokenFromContext(c))
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, gin.H{
		"loginId":       sess.LoginID,
		"terminalCount": len(sess.TerminalInfos),
	})
}

func (a *App) handleAddPermission(c *gin.Context) {
	var req accessRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Value == "" {
		writeError(c, http.StatusBadRequest, derror.CodeBadRequest, "permission is required")
		return
	}
	loginID := loginIDFromContext(c)
	if err := a.auth.AddPermissions(c.Request.Context(), dtoken.PermissionOptions{
		LoginID:     loginID,
		Permissions: []string{req.Value},
	}); err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, nil)
}

func (a *App) handleAddRole(c *gin.Context) {
	var req accessRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Value == "" {
		writeError(c, http.StatusBadRequest, derror.CodeBadRequest, "role is required")
		return
	}
	loginID := loginIDFromContext(c)
	if err := a.auth.AddRoles(c.Request.Context(), dtoken.RoleOptions{
		LoginID: loginID,
		Roles:   []string{req.Value},
	}); err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, nil)
}

func (a *App) handleDisableAccount(c *gin.Context) {
	var req disableRequest
	_ = c.ShouldBindJSON(&req)
	if err := a.auth.Disable(c.Request.Context(), dtoken.DisableOptions{
		LoginID:  loginIDFromContext(c),
		Duration: time.Minute,
		Reason:   req.Reason,
	}); err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, nil)
}

func (a *App) handleDisableService(c *gin.Context) {
	var req disableRequest
	_ = c.ShouldBindJSON(&req)
	if err := a.auth.DisableService(c.Request.Context(), dtoken.ServiceDisableOptions{
		LoginID:  loginIDFromContext(c),
		Service:  c.Param("service"),
		Level:    1,
		Duration: time.Minute,
		Reason:   req.Reason,
	}); err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, nil)
}

func (a *App) handleNonce(c *gin.Context) {
	nonce, err := a.auth.GenerateNonce(c.Request.Context())
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, gin.H{"nonce": nonce})
}

func (a *App) handleNonceVerify(c *gin.Context) {
	var req nonceRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Nonce == "" {
		writeError(c, http.StatusBadRequest, derror.CodeBadRequest, "nonce is required")
		return
	}
	if err := a.auth.VerifyAndConsumeNonce(c.Request.Context(), req.Nonce); err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, nil)
}

func (a *App) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := bearerToken(c.GetHeader("Authorization"))
		if token == "" {
			writeError(c, http.StatusUnauthorized, derror.CodeNotLogin, "missing token")
			c.Abort()
			return
		}
		if err := a.auth.CheckLogin(c.Request.Context(), token); err != nil {
			writeDTokenError(c, err)
			c.Abort()
			return
		}
		loginID, err := a.auth.GetLoginID(c.Request.Context(), token)
		if err != nil {
			writeDTokenError(c, err)
			c.Abort()
			return
		}
		c.Set("token", token)
		c.Set("loginID", loginID)
		c.Next()
	}
}

func (a *App) requirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := a.auth.CheckPermission(c.Request.Context(), dtoken.PermissionOptions{
			Token:      tokenFromContext(c),
			Permission: permission,
		})
		if err != nil {
			writeDTokenError(c, err)
			c.Abort()
			return
		}
		c.Next()
	}
}

func (a *App) requireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := a.auth.CheckRole(c.Request.Context(), dtoken.RoleOptions{
			Token: tokenFromContext(c),
			Role:  role,
		})
		if err != nil {
			writeDTokenError(c, err)
			c.Abort()
			return
		}
		c.Next()
	}
}

func (a *App) requireService(service string, level int) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr := a.auth.Manager()
		if err := mgr.CheckDisableServiceLevel(c.Request.Context(), loginIDFromContext(c), service, level); err != nil {
			writeDTokenError(c, err)
			c.Abort()
			return
		}
		c.Next()
	}
}

func tokenFromContext(c *gin.Context) string {
	token, _ := c.Get("token")
	value, _ := token.(string)
	return value
}

func loginIDFromContext(c *gin.Context) string {
	loginID, _ := c.Get("loginID")
	value, _ := loginID.(string)
	return value
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if len(header) > len(prefix) && header[:len(prefix)] == prefix {
		return header[len(prefix):]
	}
	return header
}

func defaultDuration(value, fallback time.Duration) time.Duration {
	if value > 0 {
		return value
	}
	return fallback
}

func defaultLimit(value int64) int64 {
	if value != 0 {
		return value
	}
	return -1
}

func writeOK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{Code: derror.CodeSuccess, Message: "ok", Data: data})
}

func writeError(c *gin.Context, status, code int, message string) {
	c.JSON(status, Response{Code: code, Message: message})
}

func writeDTokenError(c *gin.Context, err error) {
	status, code := http.StatusInternalServerError, derror.CodeServerError
	switch {
	case errors.Is(err, derror.ErrNotLogin), errors.Is(err, derror.ErrInvalidToken), errors.Is(err, derror.ErrTokenExpired):
		status, code = http.StatusUnauthorized, derror.CodeNotLogin
	case errors.Is(err, derror.ErrActiveTimeout):
		status, code = http.StatusUnauthorized, derror.CodeActiveTimeout
	case errors.Is(err, derror.ErrAccountDisabled):
		status, code = http.StatusForbidden, derror.CodeAccountDisabled
	case errors.Is(err, derror.ErrPermissionDenied), errors.Is(err, derror.ErrRoleDenied), errors.Is(err, derror.ErrServiceDisabled):
		status, code = http.StatusForbidden, derror.CodePermissionDenied
	case errors.Is(err, derror.ErrInvalidParam), errors.Is(err, derror.ErrInvalidNonce):
		status, code = http.StatusBadRequest, derror.CodeBadRequest
	}
	c.JSON(status, Response{Code: code, Message: err.Error()})
}

// Manager exposes the underlying manager for advanced demo checks. Manager 暴露底层管理器用于高级示例检查。
func (a *App) Manager() *manager.Manager {
	if a == nil || a.auth == nil {
		return nil
	}
	return a.auth.Manager()
}
