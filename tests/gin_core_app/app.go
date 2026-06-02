package gin_core_app

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/Zany2/dtoken-go/com/storage/memory"
	redisstorage "github.com/Zany2/dtoken-go/com/storage/redis"
	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/core/oauth2"
	"github.com/Zany2/dtoken-go/dtoken"
	"github.com/gin-gonic/gin"
)

// App owns the Gin router and core auth facade. App 持有 Gin 路由和核心认证门面。
type App struct {
	auth      *dtoken.Auth
	userAuth  *dtoken.Auth
	adminAuth *dtoken.Auth
	router    *gin.Engine
}

// Config controls the demo app auth timings. Config 控制示例应用认证时间配置。
type Config struct {
	// KeyPrefix isolates storage keys for a demo app instance. KeyPrefix 隔离示例应用实例的存储键。
	KeyPrefix string
	// TokenTimeout sets the default token lifetime. TokenTimeout 设置默认 Token 有效期。
	TokenTimeout time.Duration
	// ActiveTimeout sets the inactive timeout in seconds. ActiveTimeout 设置不活跃超时时间，单位秒。
	ActiveTimeout int64
	// RenewInterval sets the minimum interval between renewals in seconds. RenewInterval 设置续期间隔下限，单位秒。
	RenewInterval int64
	// RenewMaxRefresh sets the maximum refresh window in seconds. RenewMaxRefresh 设置最大可刷新窗口，单位秒。
	RenewMaxRefresh int64
	// AutoRenew controls whether login checks renew tokens automatically. AutoRenew 控制登录校验时是否自动续期 Token。
	AutoRenew *bool
	// IsConcurrent controls whether one account can hold multiple tokens. IsConcurrent 控制同账号是否允许多 Token 并存。
	IsConcurrent *bool
	// IsShare controls whether repeated login can share an existing token. IsShare 控制重复登录是否复用已有 Token。
	IsShare *bool
	// MaxLoginCount limits online terminal count. MaxLoginCount 限制在线终端数量。
	MaxLoginCount int64
	// ConcurrencyScope selects the dimension used by concurrency policies. ConcurrencyScope 选择并发登录策略的生效维度。
	ConcurrencyScope config.ConcurrencyScope
	// ReplacedLoginExitMode controls how replaced tokens are marked. ReplacedLoginExitMode 控制被顶下线 Token 的标记方式。
	ReplacedLoginExitMode config.ReplacedLoginExitMode
	// OverflowLogoutMode controls how overflowed tokens are marked. OverflowLogoutMode 控制超限下线 Token 的标记方式。
	OverflowLogoutMode config.LogoutMode
	// TokenStyle selects the core token generation strategy. TokenStyle 选择核心 Token 生成策略。
	TokenStyle adapter.TokenStyle
	// JwtSecretKey sets the secret used by JWT token style. JwtSecretKey 设置 JWT Token 风格使用的密钥。
	JwtSecretKey string
	// RedisURL enables Redis storage when non-empty. RedisURL 非空时启用 Redis 存储。
	RedisURL string
	// UseAccessProvider enables demo access provider overrides. UseAccessProvider 启用示例权限角色提供器覆盖。
	UseAccessProvider bool
}

// Response is the unified HTTP response body. Response 是统一 HTTP 响应体。
type Response struct {
	// Code stores the business status code. Code 存储业务状态码。
	Code int `json:"code"`
	// Message stores the response message. Message 存储响应消息。
	Message string `json:"message"`
	// Data stores the optional response payload. Data 存储可选响应数据。
	Data any `json:"data,omitempty"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Device   string `json:"device"`
	DeviceID string `json:"deviceId"`
}

type loginTimeoutRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Seconds  int64  `json:"seconds"`
	Device   string `json:"device"`
	DeviceID string `json:"deviceId"`
}

type accessRequest struct {
	Value string `json:"value"`
}

type accessListRequest struct {
	Values []string `json:"values"`
}

type renewRequest struct {
	Seconds int64 `json:"seconds"`
}

type disableRequest struct {
	Reason string `json:"reason"`
}

type serviceLevelDisableRequest struct {
	Level  int    `json:"level"`
	Reason string `json:"reason"`
}

type nonceRequest struct {
	Nonce string `json:"nonce"`
}

type nonceTimeoutRequest struct {
	Seconds int64 `json:"seconds"`
}

type oauthCodeRequest struct {
	ClientID    string   `json:"clientId"`
	UserID      string   `json:"userId"`
	RedirectURI string   `json:"redirectUri"`
	Scopes      []string `json:"scopes"`
}

type oauthTokenRequest struct {
	GrantType    string   `json:"grantType"`
	ClientID     string   `json:"clientId"`
	ClientSecret string   `json:"clientSecret"`
	Code         string   `json:"code"`
	RedirectURI  string   `json:"redirectUri"`
	RefreshToken string   `json:"refreshToken"`
	Username     string   `json:"username"`
	Password     string   `json:"password"`
	Scopes       []string `json:"scopes"`
}

type oauthRevokeRequest struct {
	Token string `json:"token"`
}

type oauthClientRequest struct {
	ClientID     string   `json:"clientId"`
	ClientSecret string   `json:"clientSecret"`
	RedirectURIs []string `json:"redirectUris"`
	GrantTypes   []string `json:"grantTypes"`
	Scopes       []string `json:"scopes"`
}

// NewApp creates a runnable Gin demo app. NewApp 创建可运行的 Gin 示例应用。
func NewApp(cfg Config) (*App, error) {
	gin.SetMode(gin.ReleaseMode)
	storageFactory := newDemoStorageFactory(cfg)

	flowStorage, err := storageFactory()
	if err != nil {
		return nil, err
	}
	flowMgr, err := newDemoManager(cfg, "gin-core-flow", flowStorage)
	if err != nil {
		return nil, err
	}

	userStorage, err := storageFactory()
	if err != nil {
		flowMgr.CloseManager()
		return nil, err
	}
	userMgr, err := newDemoManager(cfg, "user-auth", userStorage)
	if err != nil {
		flowMgr.CloseManager()
		return nil, err
	}

	adminStorage, err := storageFactory()
	if err != nil {
		flowMgr.CloseManager()
		userMgr.CloseManager()
		return nil, err
	}
	adminMgr, err := newDemoManager(cfg, "admin-auth", adminStorage)
	if err != nil {
		flowMgr.CloseManager()
		userMgr.CloseManager()
		return nil, err
	}

	app := &App{
		auth:      dtoken.New(flowMgr),
		userAuth:  dtoken.New(userMgr),
		adminAuth: dtoken.New(adminMgr),
	}
	if err = app.registerDemoOAuth2Client(); err != nil {
		flowMgr.CloseManager()
		userMgr.CloseManager()
		adminMgr.CloseManager()
		return nil, err
	}
	app.router = app.buildRouter()
	return app, nil
}

func newDemoStorageFactory(cfg Config) func() (adapter.Storage, error) {
	if cfg.RedisURL != "" {
		return func() (adapter.Storage, error) {
			return redisstorage.NewStorage(cfg.RedisURL)
		}
	}

	sharedStorage := memory.NewStorage()
	return func() (adapter.Storage, error) {
		return sharedStorage, nil
	}
}

func newDemoManager(cfg Config, authType string, storage adapter.Storage) (*manager.Manager, error) {
	autoRenew := false
	if cfg.AutoRenew != nil {
		autoRenew = *cfg.AutoRenew
	}
	builder := dtoken.NewBuilder().
		AuthType(authType).
		TokenName("Authorization").
		TimeoutDuration(defaultDuration(cfg.TokenTimeout, 30*time.Second)).
		AutoRenew(autoRenew).
		RenewInterval(defaultLimit(cfg.RenewInterval)).
		RenewMaxRefresh(defaultLimit(cfg.RenewMaxRefresh)).
		ActiveTimeout(cfg.ActiveTimeout).
		IsPrintBanner(false).
		IsLog(false).
		AsyncEvent(false).
		EnableNonce().
		EnableOAuth2().
		SetStorage(storage)
	if cfg.KeyPrefix != "" {
		builder.KeyPrefix(cfg.KeyPrefix)
	}
	if cfg.IsConcurrent != nil {
		builder.IsConcurrent(*cfg.IsConcurrent)
	}
	if cfg.IsShare != nil {
		builder.IsShare(*cfg.IsShare)
	}
	if cfg.MaxLoginCount != 0 {
		builder.MaxLoginCount(cfg.MaxLoginCount)
	}
	if cfg.ConcurrencyScope != "" {
		builder.ConcurrencyScope(cfg.ConcurrencyScope)
	}
	if cfg.ReplacedLoginExitMode != "" {
		builder.ReplacedLoginExitMode(cfg.ReplacedLoginExitMode)
	}
	if cfg.OverflowLogoutMode != "" {
		builder.OverflowLogoutMode(cfg.OverflowLogoutMode)
	}
	if cfg.TokenStyle != "" {
		builder.TokenStyle(cfg.TokenStyle)
	}
	if cfg.JwtSecretKey != "" {
		builder.JwtSecretKey(cfg.JwtSecretKey)
	}
	if cfg.UseAccessProvider {
		builder.SetAccessProvider(demoAccessProvider{})
	}
	return builder.Build()
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
	if a == nil {
		return
	}
	for _, auth := range []*dtoken.Auth{a.auth, a.userAuth, a.adminAuth} {
		if auth == nil {
			continue
		}
		if mgr := auth.Manager(); mgr != nil {
			mgr.CloseManager()
		}
	}
}

func (a *App) buildRouter() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/health", func(c *gin.Context) {
		writeOK(c, gin.H{"status": "ok"})
	})
	r.POST("/login", a.handleLogin)
	r.POST("/login/timeout", a.handleLoginWithTimeout)
	r.GET("/nonce", a.handleNonce)
	r.POST("/nonce/timeout", a.handleNonceWithTimeout)
	r.GET("/nonce/status/:nonce", a.handleNonceStatus)
	r.POST("/nonce/verify", a.handleNonceVerify)
	r.GET("/token/status", a.handleTokenStatus)
	r.GET("/operator/disable/account/:loginId", a.handleOperatorAccountDisableInfo)
	r.GET("/operator/disable/service/:loginId/:service", a.handleOperatorServiceDisableInfo)
	r.GET("/operator/disable/device/:loginId/:device", a.handleOperatorDeviceDisableInfo)
	r.GET("/operator/disable/device/:loginId/:device/:deviceId", a.handleOperatorDeviceDisableInfo)
	r.POST("/operator/untie/account/:loginId", a.handleOperatorUntieAccount)
	r.POST("/operator/untie/device/:loginId/:device", a.handleOperatorUntieDevice)
	r.POST("/operator/untie/device/:loginId/:device/:deviceId", a.handleOperatorUntieDevice)

	api := r.Group("/api")
	api.Use(a.authMiddleware())
	api.GET("/me", a.handleMe)
	api.POST("/logout", a.handleLogout)
	api.GET("/token/ttl", a.handleTokenTTL)
	api.GET("/token/info", a.handleTokenInfo)
	api.POST("/token/login-by-token", a.handleLoginByToken)
	api.POST("/token/renew", a.handleTokenRenew)
	api.POST("/token/kickout", a.handleTokenKickout)
	api.POST("/token/replace", a.handleTokenReplace)
	api.POST("/logout/account", a.handleLogoutAccount)
	api.POST("/kickout/account", a.handleKickoutAccount)
	api.POST("/logout/device/:device/:deviceId", a.handleLogoutDevice)
	api.POST("/logout/device/:device", a.handleLogoutDevice)
	api.POST("/kickout/device/:device", a.handleKickoutDevice)
	api.POST("/kickout/device/:device/:deviceId", a.handleKickoutDevice)
	api.POST("/replace/account", a.handleReplaceAccount)
	api.POST("/replace/device/:device", a.handleReplaceDevice)
	api.POST("/replace/device/:device/:deviceId", a.handleReplaceDevice)
	api.GET("/session", a.handleSession)
	api.GET("/terminal", a.handleTerminal)
	api.GET("/session/tokens", a.handleSessionTokens)
	api.GET("/session/terminals", a.handleSessionTerminals)
	api.GET("/session/foreach", a.handleSessionForEach)
	api.GET("/session/search", a.handleSessionSearch)
	api.GET("/access/status", a.handleAccessStatus)
	api.GET("/access/list", a.handleAccessList)
	api.GET("/articles", a.requirePermission("article:read"), func(c *gin.Context) {
		writeOK(c, []string{"article-a", "article-b"})
	})
	api.GET("/article/manage", a.requirePermissionsAnd("article:read", "article:write"), func(c *gin.Context) {
		writeOK(c, gin.H{"scope": "article-manage"})
	})
	api.GET("/content", a.requirePermissionsOr("content:read", "content:write"), func(c *gin.Context) {
		writeOK(c, gin.H{"scope": "content"})
	})
	api.GET("/admin", a.requireRole("admin"), func(c *gin.Context) {
		writeOK(c, gin.H{"scope": "admin"})
	})
	api.GET("/ops", a.requireRolesAnd("admin", "ops"), func(c *gin.Context) {
		writeOK(c, gin.H{"scope": "ops"})
	})
	api.GET("/audit", a.requireRolesOr("auditor", "security"), func(c *gin.Context) {
		writeOK(c, gin.H{"scope": "audit"})
	})
	api.GET("/payment", a.requireService("payment", 1), func(c *gin.Context) {
		writeOK(c, gin.H{"status": "paid"})
	})
	api.POST("/permissions", a.handleAddPermission)
	api.DELETE("/permissions", a.handleRemovePermission)
	api.POST("/permissions/batch", a.handleAddPermissions)
	api.POST("/roles", a.handleAddRole)
	api.DELETE("/roles", a.handleRemoveRole)
	api.POST("/roles/batch", a.handleAddRoles)
	api.POST("/disable/account", a.handleDisableAccount)
	api.POST("/untie/account", a.handleUntieAccount)
	api.POST("/disable/service/:service", a.handleDisableService)
	api.POST("/disable/service/:service/level", a.handleDisableServiceLevel)
	api.GET("/disable/service/:service/level/:level", a.handleServiceLevelStatus)
	api.POST("/untie/service/:service", a.handleUntieService)
	api.POST("/disable/device/:device", a.handleDisableDevice)
	api.POST("/disable/device/:device/:deviceId", a.handleDisableDevice)
	api.POST("/untie/device/:device", a.handleUntieDevice)
	api.POST("/untie/device/:device/:deviceId", a.handleUntieDevice)

	oauth := r.Group("/oauth2")
	oauth.POST("/authorize", a.handleOAuth2Authorize)
	oauth.POST("/token", a.handleOAuth2Token)
	oauth.POST("/revoke", a.handleOAuth2Revoke)
	oauth.GET("/introspect", a.handleOAuth2Introspect)
	oauth.POST("/clients", a.handleOAuth2RegisterClient)
	oauth.GET("/clients/:clientId", a.handleOAuth2GetClient)
	oauth.DELETE("/clients/:clientId", a.handleOAuth2UnregisterClient)

	multi := r.Group("/multi-auth")
	multi.POST("/user/login", a.handleMultiAuthLogin(a.userAuth, "user"))
	multi.POST("/admin/login", a.handleMultiAuthLogin(a.adminAuth, "admin"))

	userAPI := multi.Group("/user")
	userAPI.Use(a.authMiddlewareFor(a.userAuth))
	userAPI.GET("/me", a.handleMultiAuthMe(a.userAuth, "user"))
	userAPI.POST("/permissions", a.handleMultiAuthAddPermission(a.userAuth))
	userAPI.GET("/profile", a.requirePermissionFor(a.userAuth, "profile:read"), func(c *gin.Context) {
		writeOK(c, gin.H{"scope": "user-profile"})
	})

	adminAPI := multi.Group("/admin")
	adminAPI.Use(a.authMiddlewareFor(a.adminAuth))
	adminAPI.GET("/me", a.handleMultiAuthMe(a.adminAuth, "admin"))
	adminAPI.POST("/roles", a.handleMultiAuthAddRole(a.adminAuth))
	adminAPI.GET("/dashboard", a.requireRoleFor(a.adminAuth, "admin"), func(c *gin.Context) {
		writeOK(c, gin.H{"scope": "admin-dashboard"})
	})

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

func (a *App) handleLoginWithTimeout(c *gin.Context) {
	var req loginTimeoutRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Username == "" || req.Password == "" || req.Seconds <= 0 {
		writeError(c, http.StatusBadRequest, derror.CodeBadRequest, "username, password and positive seconds are required")
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
		Timeout:  time.Duration(req.Seconds) * time.Second,
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

func (a *App) handleTokenInfo(c *gin.Context) {
	token := tokenFromContext(c)
	info, err := a.auth.GetTokenInfo(c.Request.Context(), token)
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	device, err := a.auth.Manager().GetDevice(c.Request.Context(), token)
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	deviceID, err := a.auth.Manager().GetDeviceId(c.Request.Context(), token)
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	createTime, err := a.auth.Manager().GetTokenCreateTime(c.Request.Context(), token)
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, gin.H{
		"authType":   info.AuthType,
		"loginId":    info.LoginID,
		"device":     device,
		"deviceId":   deviceID,
		"createTime": createTime,
		"timeout":    info.Timeout,
	})
}

func (a *App) handleTokenStatus(c *gin.Context) {
	token := bearerToken(c.GetHeader("Authorization"))
	if token == "" {
		writeOK(c, gin.H{"isLogin": false})
		return
	}
	writeOK(c, gin.H{"isLogin": a.auth.IsLogin(c.Request.Context(), token)})
}

func (a *App) handleLoginByToken(c *gin.Context) {
	if err := a.auth.LoginByToken(c.Request.Context(), tokenFromContext(c)); err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, nil)
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

func (a *App) handleTokenKickout(c *gin.Context) {
	if err := a.auth.Kickout(c.Request.Context(), dtoken.LogoutOptions{Token: tokenFromContext(c)}); err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, nil)
}

func (a *App) handleTokenReplace(c *gin.Context) {
	if err := a.auth.Replace(c.Request.Context(), dtoken.LogoutOptions{Token: tokenFromContext(c)}); err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, nil)
}

func (a *App) handleLogoutAccount(c *gin.Context) {
	if err := a.auth.Logout(c.Request.Context(), dtoken.LogoutOptions{LoginID: loginIDFromContext(c)}); err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, nil)
}

func (a *App) handleKickoutAccount(c *gin.Context) {
	if err := a.auth.Kickout(c.Request.Context(), dtoken.LogoutOptions{LoginID: loginIDFromContext(c)}); err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, nil)
}

func (a *App) handleLogoutDevice(c *gin.Context) {
	device := c.Param("device")
	deviceID := c.Param("deviceId")
	var err error
	if deviceID != "" {
		err = a.auth.Logout(c.Request.Context(), dtoken.LogoutOptions{
			LoginID:  loginIDFromContext(c),
			Device:   device,
			DeviceID: deviceID,
		})
	} else {
		err = a.auth.Manager().LogoutByDevice(c.Request.Context(), loginIDFromContext(c), device)
	}
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, nil)
}

func (a *App) handleKickoutDevice(c *gin.Context) {
	device := c.Param("device")
	deviceID := c.Param("deviceId")
	var err error
	if deviceID != "" {
		err = a.auth.Kickout(c.Request.Context(), dtoken.LogoutOptions{
			LoginID:  loginIDFromContext(c),
			Device:   device,
			DeviceID: deviceID,
		})
	} else {
		err = a.auth.Manager().KickoutByDevice(c.Request.Context(), loginIDFromContext(c), device)
	}
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, nil)
}

func (a *App) handleReplaceAccount(c *gin.Context) {
	if err := a.auth.Replace(c.Request.Context(), dtoken.LogoutOptions{LoginID: loginIDFromContext(c)}); err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, nil)
}

func (a *App) handleReplaceDevice(c *gin.Context) {
	device := c.Param("device")
	deviceID := c.Param("deviceId")
	var err error
	if deviceID != "" {
		err = a.auth.Replace(c.Request.Context(), dtoken.LogoutOptions{
			LoginID:  loginIDFromContext(c),
			Device:   device,
			DeviceID: deviceID,
		})
	} else {
		err = a.auth.Manager().ReplaceByDevice(c.Request.Context(), loginIDFromContext(c), device)
	}
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, nil)
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

func (a *App) handleTerminal(c *gin.Context) {
	token := tokenFromContext(c)
	info, err := a.auth.GetTerminalInfoByToken(c.Request.Context(), token)
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	mgr := a.auth.Manager()
	onlineCount, err := mgr.GetOnlineTerminalCount(c.Request.Context(), info.LoginID)
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	deviceCount, err := mgr.GetOnlineTerminalCountByDevice(c.Request.Context(), info.LoginID, info.Device)
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	deviceIDCount, err := mgr.GetOnlineTerminalCountByDeviceAndDeviceId(c.Request.Context(), info.LoginID, info.Device, info.DeviceId)
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	latestToken, err := mgr.GetTokenValueByLoginID(c.Request.Context(), info.LoginID, info.Device)
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, gin.H{
		"loginId":         info.LoginID,
		"device":          info.Device,
		"deviceId":        info.DeviceId,
		"index":           info.Index,
		"onlineCount":     onlineCount,
		"deviceCount":     deviceCount,
		"deviceIdCount":   deviceIDCount,
		"latestForDevice": latestToken,
	})
}

func (a *App) handleSessionTokens(c *gin.Context) {
	loginID := loginIDFromContext(c)
	device := c.Query("device")
	deviceID := c.Query("deviceId")
	checkAlive := c.Query("alive") == "true"
	mgr := a.auth.Manager()
	var (
		tokens []string
		err    error
	)
	switch {
	case device != "" && deviceID != "":
		tokens, err = mgr.GetTokenValueListByDeviceAndDeviceId(c.Request.Context(), loginID, device, deviceID, checkAlive)
	case device != "":
		tokens, err = mgr.GetTokenValueListByDevice(c.Request.Context(), loginID, device, checkAlive)
	default:
		tokens, err = mgr.GetTokenValueListByLoginID(c.Request.Context(), loginID, checkAlive)
	}
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, gin.H{"tokens": tokens})
}

func (a *App) handleSessionTerminals(c *gin.Context) {
	loginID := loginIDFromContext(c)
	device := c.Query("device")
	var (
		terminals []manager.TerminalInfo
		err       error
	)
	if device != "" {
		terminals, err = a.auth.Manager().GetTerminalListByLoginID(c.Request.Context(), loginID, device)
	} else {
		terminals, err = a.auth.Manager().GetTerminalListByLoginID(c.Request.Context(), loginID)
	}
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, gin.H{"terminals": terminals})
}

func (a *App) handleSessionForEach(c *gin.Context) {
	loginID := loginIDFromContext(c)
	device := c.Query("device")
	limit := parsePositiveInt(c.Query("limit"))
	visited := make([]string, 0)
	visitor := func(terminal manager.TerminalInfo) bool {
		visited = append(visited, terminal.Token)
		return limit <= 0 || len(visited) < limit
	}
	var err error
	if device != "" {
		err = a.auth.Manager().ForEachTerminalByDevice(c.Request.Context(), loginID, device, visitor)
	} else {
		err = a.auth.Manager().ForEachTerminal(c.Request.Context(), loginID, visitor)
	}
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, gin.H{"visited": visited})
}

func (a *App) handleSessionSearch(c *gin.Context) {
	keyword := c.Query("keyword")
	start := parseIntDefault(c.Query("start"), 0)
	size := parseIntDefault(c.Query("size"), -1)
	tokenValues, err := a.auth.Manager().SearchTokenValue(c.Request.Context(), keyword, start, size)
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	sessionIDs, err := a.auth.Manager().SearchSessionId(c.Request.Context(), keyword, start, size)
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, gin.H{
		"tokens":     tokenValues,
		"sessionIds": sessionIDs,
	})
}

func (a *App) handleAccessStatus(c *gin.Context) {
	token := tokenFromContext(c)
	loginID := loginIDFromContext(c)
	permission := c.Query("permission")
	role := c.Query("role")
	writeOK(c, gin.H{
		"hasPermissionByLoginId": a.auth.Manager().HasPermission(c.Request.Context(), loginID, permission),
		"hasPermissionByToken":   a.auth.Manager().HasPermissionByToken(c.Request.Context(), token, permission),
		"hasRoleByLoginId":       a.auth.Manager().HasRole(c.Request.Context(), loginID, role),
		"hasRoleByToken":         a.auth.Manager().HasRoleByToken(c.Request.Context(), token, role),
	})
}

func (a *App) handleAccessList(c *gin.Context) {
	token := tokenFromContext(c)
	loginID := loginIDFromContext(c)
	permissionsByLoginID, err := a.auth.GetPermissions(c.Request.Context(), loginID)
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	permissionsByToken, err := a.auth.Manager().GetPermissionsByToken(c.Request.Context(), token)
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	rolesByLoginID, err := a.auth.GetRoles(c.Request.Context(), loginID)
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	rolesByToken, err := a.auth.Manager().GetRolesByToken(c.Request.Context(), token)
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, gin.H{
		"permissionsByLoginId": permissionsByLoginID,
		"permissionsByToken":   permissionsByToken,
		"rolesByLoginId":       rolesByLoginID,
		"rolesByToken":         rolesByToken,
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

func (a *App) handleRemovePermission(c *gin.Context) {
	var req accessRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Value == "" {
		writeError(c, http.StatusBadRequest, derror.CodeBadRequest, "permission is required")
		return
	}
	if err := a.auth.RemovePermissions(c.Request.Context(), dtoken.PermissionOptions{
		Token:       tokenFromContext(c),
		Permissions: []string{req.Value},
	}); err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, nil)
}

func (a *App) handleAddPermissions(c *gin.Context) {
	var req accessListRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Values) == 0 {
		writeError(c, http.StatusBadRequest, derror.CodeBadRequest, "permissions are required")
		return
	}
	if err := a.auth.AddPermissions(c.Request.Context(), dtoken.PermissionOptions{
		Token:       tokenFromContext(c),
		Permissions: req.Values,
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

func (a *App) handleRemoveRole(c *gin.Context) {
	var req accessRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Value == "" {
		writeError(c, http.StatusBadRequest, derror.CodeBadRequest, "role is required")
		return
	}
	if err := a.auth.RemoveRoles(c.Request.Context(), dtoken.RoleOptions{
		Token: tokenFromContext(c),
		Roles: []string{req.Value},
	}); err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, nil)
}

func (a *App) handleAddRoles(c *gin.Context) {
	var req accessListRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Values) == 0 {
		writeError(c, http.StatusBadRequest, derror.CodeBadRequest, "roles are required")
		return
	}
	if err := a.auth.AddRoles(c.Request.Context(), dtoken.RoleOptions{
		Token: tokenFromContext(c),
		Roles: req.Values,
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

func (a *App) handleUntieAccount(c *gin.Context) {
	if err := a.auth.Untie(c.Request.Context(), loginIDFromContext(c)); err != nil {
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

func (a *App) handleDisableServiceLevel(c *gin.Context) {
	var req serviceLevelDisableRequest
	_ = c.ShouldBindJSON(&req)
	if req.Level <= 0 {
		writeError(c, http.StatusBadRequest, derror.CodeBadRequest, "level must be positive")
		return
	}
	if err := a.auth.DisableService(c.Request.Context(), dtoken.ServiceDisableOptions{
		LoginID:  loginIDFromContext(c),
		Service:  c.Param("service"),
		Level:    req.Level,
		Duration: time.Minute,
		Reason:   req.Reason,
	}); err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, nil)
}

func (a *App) handleServiceLevelStatus(c *gin.Context) {
	level := parsePositiveInt(c.Param("level"))
	if level <= 0 {
		writeError(c, http.StatusBadRequest, derror.CodeBadRequest, "level must be positive")
		return
	}
	err := a.auth.Manager().CheckDisableServiceLevel(c.Request.Context(), loginIDFromContext(c), c.Param("service"), level)
	writeOK(c, gin.H{
		"disabled": err != nil,
		"level":    level,
	})
}

func (a *App) handleUntieService(c *gin.Context) {
	if err := a.auth.UntieService(c.Request.Context(), loginIDFromContext(c), c.Param("service")); err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, nil)
}

func (a *App) handleDisableDevice(c *gin.Context) {
	var req disableRequest
	_ = c.ShouldBindJSON(&req)
	if err := a.auth.DisableDevice(c.Request.Context(), dtoken.DeviceDisableOptions{
		LoginID:  loginIDFromContext(c),
		Device:   c.Param("device"),
		DeviceID: c.Param("deviceId"),
		Duration: time.Minute,
		Reason:   req.Reason,
	}); err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, nil)
}

func (a *App) handleUntieDevice(c *gin.Context) {
	device := c.Param("device")
	deviceID := c.Param("deviceId")
	var err error
	if deviceID != "" {
		err = a.auth.UntieDeviceAndDeviceId(c.Request.Context(), loginIDFromContext(c), device, deviceID)
	} else {
		err = a.auth.UntieDevice(c.Request.Context(), loginIDFromContext(c), device)
	}
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, nil)
}

func (a *App) handleOperatorUntieAccount(c *gin.Context) {
	if err := a.auth.Untie(c.Request.Context(), c.Param("loginId")); err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, nil)
}

func (a *App) handleOperatorUntieDevice(c *gin.Context) {
	loginID := c.Param("loginId")
	device := c.Param("device")
	deviceID := c.Param("deviceId")
	var err error
	if deviceID != "" {
		err = a.auth.UntieDeviceAndDeviceId(c.Request.Context(), loginID, device, deviceID)
	} else {
		err = a.auth.UntieDevice(c.Request.Context(), loginID, device)
	}
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, nil)
}

func (a *App) handleOperatorAccountDisableInfo(c *gin.Context) {
	loginID := c.Param("loginId")
	info, err := a.auth.Manager().GetDisableInfo(c.Request.Context(), loginID)
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	ttl, err := a.auth.Manager().GetDisableTTL(c.Request.Context(), loginID)
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, gin.H{
		"disabled": true,
		"reason":   info.DisableReason,
		"ttl":      ttl,
	})
}

func (a *App) handleOperatorServiceDisableInfo(c *gin.Context) {
	loginID := c.Param("loginId")
	service := c.Param("service")
	info, err := a.auth.Manager().GetDisableServiceInfo(c.Request.Context(), loginID, service)
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	ttl, err := a.auth.Manager().GetDisableServiceTTL(c.Request.Context(), loginID, service)
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, gin.H{
		"disabled": true,
		"service":  info.Service,
		"level":    info.Level,
		"reason":   info.DisableReason,
		"ttl":      ttl,
	})
}

func (a *App) handleOperatorDeviceDisableInfo(c *gin.Context) {
	loginID := c.Param("loginId")
	device := c.Param("device")
	deviceID := c.Param("deviceId")
	mgr := a.auth.Manager()
	var (
		info *manager.DeviceDisableInfo
		ttl  int64
		err  error
	)
	if deviceID != "" {
		info, err = mgr.GetDisableDeviceAndDeviceIdInfo(c.Request.Context(), loginID, device, deviceID)
		if err == nil {
			ttl, err = mgr.GetDisableDeviceAndDeviceIdTTL(c.Request.Context(), loginID, device, deviceID)
		}
	} else {
		info, err = mgr.GetDisableDeviceInfo(c.Request.Context(), loginID, device)
		if err == nil {
			ttl, err = mgr.GetDisableDeviceTTL(c.Request.Context(), loginID, device)
		}
	}
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, gin.H{
		"disabled": true,
		"device":   info.Device,
		"deviceId": info.DeviceId,
		"reason":   info.DisableReason,
		"ttl":      ttl,
	})
}

func (a *App) handleNonce(c *gin.Context) {
	nonce, err := a.auth.GenerateNonce(c.Request.Context())
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, gin.H{"nonce": nonce})
}

func (a *App) handleNonceWithTimeout(c *gin.Context) {
	var req nonceTimeoutRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Seconds <= 0 {
		writeError(c, http.StatusBadRequest, derror.CodeBadRequest, "seconds must be positive")
		return
	}
	nonce, err := a.auth.GenerateNonceWithTimeout(c.Request.Context(), time.Duration(req.Seconds)*time.Second)
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, gin.H{"nonce": nonce})
}

func (a *App) handleNonceStatus(c *gin.Context) {
	nonceValue := c.Param("nonce")
	ttl, err := a.auth.Manager().GetNonceTTL(c.Request.Context(), nonceValue)
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, gin.H{
		"valid": a.auth.Manager().IsNonceValid(c.Request.Context(), nonceValue),
		"ttl":   ttl,
	})
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

func (a *App) handleOAuth2Authorize(c *gin.Context) {
	var req oauthCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.ClientID == "" || req.UserID == "" || req.RedirectURI == "" {
		writeError(c, http.StatusBadRequest, derror.CodeBadRequest, "clientId, userId and redirectUri are required")
		return
	}
	code, err := a.auth.Manager().GenerateOAuth2AuthorizationCode(c.Request.Context(), req.ClientID, req.UserID, req.RedirectURI, req.Scopes)
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, gin.H{"code": code.Code})
}

func (a *App) handleOAuth2Token(c *gin.Context) {
	var req oauthTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, derror.CodeBadRequest, "invalid oauth2 token request")
		return
	}
	token, err := a.auth.OAuth2Token(c.Request.Context(), &oauth2.TokenRequest{
		GrantType:    oauth2.GrantType(req.GrantType),
		ClientID:     req.ClientID,
		ClientSecret: req.ClientSecret,
		Code:         req.Code,
		RedirectURI:  req.RedirectURI,
		RefreshToken: req.RefreshToken,
		Username:     req.Username,
		Password:     req.Password,
		Scopes:       req.Scopes,
	}, demoOAuth2UserValidator)
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, gin.H{
		"accessToken":  token.Token,
		"tokenType":    token.TokenType,
		"expiresIn":    token.ExpiresIn,
		"refreshToken": token.RefreshToken,
		"scopes":       token.Scopes,
		"userId":       token.UserID,
		"clientId":     token.ClientID,
	})
}

func (a *App) handleOAuth2Revoke(c *gin.Context) {
	var req oauthRevokeRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Token == "" {
		writeError(c, http.StatusBadRequest, derror.CodeBadRequest, "token is required")
		return
	}
	if err := a.auth.Manager().RevokeOAuth2Token(c.Request.Context(), req.Token); err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, nil)
}

func (a *App) handleOAuth2Introspect(c *gin.Context) {
	token := bearerToken(c.GetHeader("Authorization"))
	if token == "" {
		writeError(c, http.StatusUnauthorized, derror.CodeNotLogin, "missing oauth2 token")
		return
	}
	info, err := a.auth.Manager().ValidateOAuth2AccessTokenAndGetInfo(c.Request.Context(), token)
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, gin.H{
		"active":   true,
		"userId":   info.UserID,
		"clientId": info.ClientID,
		"scopes":   info.Scopes,
	})
}

func (a *App) handleOAuth2RegisterClient(c *gin.Context) {
	var req oauthClientRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.ClientID == "" {
		writeError(c, http.StatusBadRequest, derror.CodeBadRequest, "clientId is required")
		return
	}
	grantTypes := make([]oauth2.GrantType, len(req.GrantTypes))
	for i, grantType := range req.GrantTypes {
		grantTypes[i] = oauth2.GrantType(grantType)
	}
	if err := a.auth.RegisterOAuth2Client(&oauth2.Client{
		ClientID:     req.ClientID,
		ClientSecret: req.ClientSecret,
		RedirectURIs: req.RedirectURIs,
		GrantTypes:   grantTypes,
		Scopes:       req.Scopes,
	}); err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, nil)
}

func (a *App) handleOAuth2GetClient(c *gin.Context) {
	client, err := a.auth.Manager().GetOAuth2Client(c.Param("clientId"))
	if err != nil {
		writeDTokenError(c, err)
		return
	}
	grantTypes := make([]string, len(client.GrantTypes))
	for i, grantType := range client.GrantTypes {
		grantTypes[i] = string(grantType)
	}
	writeOK(c, gin.H{
		"clientId":     client.ClientID,
		"clientSecret": client.ClientSecret,
		"redirectUris": client.RedirectURIs,
		"grantTypes":   grantTypes,
		"scopes":       client.Scopes,
	})
}

func (a *App) handleOAuth2UnregisterClient(c *gin.Context) {
	if err := a.auth.Manager().UnregisterOAuth2Client(c.Param("clientId")); err != nil {
		writeDTokenError(c, err)
		return
	}
	writeOK(c, nil)
}

func (a *App) handleMultiAuthLogin(auth *dtoken.Auth, authName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req loginRequest
		if err := c.ShouldBindJSON(&req); err != nil || req.Username == "" || req.Password == "" {
			writeError(c, http.StatusBadRequest, derror.CodeBadRequest, "username and password are required")
			return
		}
		if req.Password != "123456" {
			writeError(c, http.StatusUnauthorized, derror.CodeNotLogin, "invalid username or password")
			return
		}
		token, err := auth.Login(c.Request.Context(), dtoken.LoginOptions{
			LoginID:  req.Username,
			Device:   req.Device,
			DeviceID: req.DeviceID,
		})
		if err != nil {
			writeDTokenError(c, err)
			return
		}
		writeOK(c, gin.H{"auth": authName, "token": token})
	}
}

func (a *App) handleMultiAuthMe(auth *dtoken.Auth, authName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		loginID := loginIDFromContext(c)
		roles, _ := auth.GetRoles(c.Request.Context(), loginID)
		permissions, _ := auth.GetPermissions(c.Request.Context(), loginID)
		writeOK(c, gin.H{
			"auth":        authName,
			"loginId":     loginID,
			"roles":       roles,
			"permissions": permissions,
		})
	}
}

func (a *App) handleMultiAuthAddPermission(auth *dtoken.Auth) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req accessRequest
		if err := c.ShouldBindJSON(&req); err != nil || req.Value == "" {
			writeError(c, http.StatusBadRequest, derror.CodeBadRequest, "permission is required")
			return
		}
		if err := auth.AddPermissions(c.Request.Context(), dtoken.PermissionOptions{
			LoginID:     loginIDFromContext(c),
			Permissions: []string{req.Value},
		}); err != nil {
			writeDTokenError(c, err)
			return
		}
		writeOK(c, nil)
	}
}

func (a *App) handleMultiAuthAddRole(auth *dtoken.Auth) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req accessRequest
		if err := c.ShouldBindJSON(&req); err != nil || req.Value == "" {
			writeError(c, http.StatusBadRequest, derror.CodeBadRequest, "role is required")
			return
		}
		if err := auth.AddRoles(c.Request.Context(), dtoken.RoleOptions{
			LoginID: loginIDFromContext(c),
			Roles:   []string{req.Value},
		}); err != nil {
			writeDTokenError(c, err)
			return
		}
		writeOK(c, nil)
	}
}

func (a *App) authMiddleware() gin.HandlerFunc {
	return a.authMiddlewareFor(a.auth)
}

func (a *App) authMiddlewareFor(auth *dtoken.Auth) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := bearerToken(c.GetHeader("Authorization"))
		if token == "" {
			writeError(c, http.StatusUnauthorized, derror.CodeNotLogin, "missing token")
			c.Abort()
			return
		}
		if err := auth.CheckLogin(c.Request.Context(), token); err != nil {
			writeDTokenError(c, err)
			c.Abort()
			return
		}
		loginID, err := auth.GetLoginID(c.Request.Context(), token)
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
	return a.requirePermissionFor(a.auth, permission)
}

func (a *App) requirePermissionsAnd(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := a.auth.CheckPermissionsAnd(c.Request.Context(), dtoken.PermissionOptions{
			Token:       tokenFromContext(c),
			Permissions: permissions,
		})
		if err != nil {
			writeDTokenError(c, err)
			c.Abort()
			return
		}
		c.Next()
	}
}

func (a *App) requirePermissionsOr(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := a.auth.CheckPermissionsOr(c.Request.Context(), dtoken.PermissionOptions{
			Token:       tokenFromContext(c),
			Permissions: permissions,
		})
		if err != nil {
			writeDTokenError(c, err)
			c.Abort()
			return
		}
		c.Next()
	}
}

func (a *App) requirePermissionFor(auth *dtoken.Auth, permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := auth.CheckPermission(c.Request.Context(), dtoken.PermissionOptions{
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
	return a.requireRoleFor(a.auth, role)
}

func (a *App) requireRolesAnd(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := a.auth.CheckRolesAnd(c.Request.Context(), dtoken.RoleOptions{
			Token: tokenFromContext(c),
			Roles: roles,
		})
		if err != nil {
			writeDTokenError(c, err)
			c.Abort()
			return
		}
		c.Next()
	}
}

func (a *App) requireRolesOr(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := a.auth.CheckRolesOr(c.Request.Context(), dtoken.RoleOptions{
			Token: tokenFromContext(c),
			Roles: roles,
		})
		if err != nil {
			writeDTokenError(c, err)
			c.Abort()
			return
		}
		c.Next()
	}
}

func (a *App) requireRoleFor(auth *dtoken.Auth, role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := auth.CheckRole(c.Request.Context(), dtoken.RoleOptions{
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

func parsePositiveInt(value string) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0
	}
	return parsed
}

func parseIntDefault(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
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
	case errors.Is(err, derror.ErrNotLogin), errors.Is(err, derror.ErrInvalidToken), errors.Is(err, derror.ErrTokenExpired),
		errors.Is(err, derror.ErrTokenKickout), errors.Is(err, derror.ErrTokenReplaced), errors.Is(err, derror.ErrInvalidAccessToken):
		status, code = http.StatusUnauthorized, derror.CodeNotLogin
	case errors.Is(err, derror.ErrActiveTimeout):
		status, code = http.StatusUnauthorized, derror.CodeActiveTimeout
	case errors.Is(err, derror.ErrAccountDisabled), errors.Is(err, derror.ErrDeviceDisabled):
		status, code = http.StatusForbidden, derror.CodeAccountDisabled
	case errors.Is(err, derror.ErrPermissionDenied), errors.Is(err, derror.ErrRoleDenied), errors.Is(err, derror.ErrServiceDisabled):
		status, code = http.StatusForbidden, derror.CodePermissionDenied
	case errors.Is(err, derror.ErrLoginLimitExceeded):
		status, code = http.StatusForbidden, derror.CodeMaxLoginCount
	case errors.Is(err, derror.ErrClientNotFound):
		status, code = http.StatusNotFound, derror.CodeNotFound
	case errors.Is(err, derror.ErrInvalidParam), errors.Is(err, derror.ErrInvalidNonce), errors.Is(err, derror.ErrInvalidClientCredentials),
		errors.Is(err, derror.ErrInvalidGrantType), errors.Is(err, derror.ErrInvalidRedirectURI), errors.Is(err, derror.ErrInvalidScope),
		errors.Is(err, derror.ErrInvalidAuthCode), errors.Is(err, derror.ErrAuthCodeUsed), errors.Is(err, derror.ErrAuthCodeExpired),
		errors.Is(err, derror.ErrInvalidRefreshToken), errors.Is(err, derror.ErrInvalidUserCredentials):
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

func (a *App) registerDemoOAuth2Client() error {
	return a.auth.RegisterOAuth2Client(&oauth2.Client{
		ClientID:     "demo-client",
		ClientSecret: "demo-secret",
		RedirectURIs: []string{
			"https://client.example/callback",
		},
		GrantTypes: []oauth2.GrantType{
			oauth2.GrantTypeAuthorizationCode,
			oauth2.GrantTypeClientCredentials,
			oauth2.GrantTypePassword,
			oauth2.GrantTypeRefreshToken,
		},
		Scopes: []string{"read", "write"},
	})
}

func demoOAuth2UserValidator(username, password string) (string, error) {
	if username == "" || password != "123456" {
		return "", derror.ErrInvalidUserCredentials
	}
	return "user-" + username, nil
}

type demoAccessProvider struct{}

func (demoAccessProvider) Permissions(_ context.Context, subject manager.AccessSubject) ([]string, error) {
	switch subject.LoginID {
	case "provider-user":
		return []string{"provider:read"}, nil
	case "provider-terminal-user":
		if subject.Device == "mobile" {
			return []string{"profile:read"}, nil
		}
		return []string{}, nil
	default:
		return nil, nil
	}
}

func (demoAccessProvider) Roles(_ context.Context, subject manager.AccessSubject) ([]string, error) {
	switch subject.LoginID {
	case "provider-user":
		return []string{"provider-role"}, nil
	case "provider-terminal-user":
		if subject.Device == "mobile" {
			return []string{"admin"}, nil
		}
		return []string{}, nil
	default:
		return nil, nil
	}
}
