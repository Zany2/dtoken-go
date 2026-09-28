// @Author daixk 2026/05/29
package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/Zany2/dtoken-go/sso"
	"github.com/gin-gonic/gin"
)

// Example Gin client settings define local endpoints, credentials, and cookie lifetime. Example Gin client settings 定义本地端点、凭证和 Cookie 有效期。
const (
	addr              = "localhost:9101"
	callbackURL       = "http://localhost:9101/sso/callback"
	logoutCallbackURL = "http://localhost:9101/sso/logout-callback"
	clientID          = "gin-demo-client"
	clientSecret      = "gin-demo-secret"
	localCookie       = "gin_demo_client_login"
	localCookieTTL    = 2 * time.Hour
)

// localSession stores an identity and its server-side deadline. localSession 保存身份及服务端截止时间。
type localSession struct {
	loginID   string    // loginID identifies the authenticated user. loginID 标识已认证用户。
	expiresAt time.Time // expiresAt bounds the lifetime even if a Cookie is replayed. expiresAt 限制 Cookie 重放时的会话寿命。
}

// localSessions stores local sessions with server-side expiry. localSessions 保存带服务端过期时间的本地会话。
var localSessions = struct {
	mu     sync.RWMutex
	values map[string]localSession
}{values: make(map[string]localSession)}

// loginStateCookie binds the authorization round trip to this browser for five minutes. loginStateCookie 将授权往返绑定到当前浏览器，有效期五分钟。
var loginStateCookie = sso.CookieOptions{
	Name: "gin_demo_client_state", Path: "/", MaxAge: 5 * time.Minute,
	HTTPOnly: true, SameSite: http.SameSiteLaxMode, SecretKey: rand.Text(),
}

// clientApp configures the Ticket-based SSO client. clientApp 配置基于 Ticket 的 SSO 客户端。
var clientApp = sso.NewClientApp(sso.ClientConfig{
	Mode:              sso.ModeTicket,
	ClientID:          clientID,
	ClientSecret:      clientSecret,
	ServerURL:         "http://localhost:9100",
	RegisterCallback:  true,
	LogoutCallbackURL: logoutCallbackURL,
	CheckSign:         false,
	Endpoints:         sso.DefaultEndpoints(),
	Params:            sso.DefaultParamNames(),
})

func main() {
	gin.SetMode(gin.ReleaseMode)
	log.Printf("Gin SSO client listening on http://%s", addr)
	log.Fatal(newDemoRouter().Run(addr))
}

// newDemoRouter shares the actual routes with regression tests. newDemoRouter 与回归测试共用实际路由。
func newDemoRouter() *gin.Engine {
	r := gin.New()

	// Keep callback Tickets and browser state out of access logs. 避免访问日志记录回调 Ticket 和浏览器状态。
	r.Use(gin.LoggerWithConfig(gin.LoggerConfig{SkipPaths: []string{"/sso/callback"}}), gin.Recovery())
	r.GET("/", home)
	r.GET("/protected", protected)
	r.GET("/sso/callback", callback)
	r.POST("/sso/logout-callback", ginWrap(clientApp.LogoutCallbackHandler(logoutCallback)))
	r.GET("/logout", logout)

	return r
}

// ginWrap adapts a net/http handler to Gin. ginWrap 将 net/http 处理器适配为 Gin 处理器。
func ginWrap(handler http.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		handler(c.Writer, c.Request)
	}
}

// home renders the Gin SSO client landing page. home 渲染 Gin SSO 客户端首页。
func home(c *gin.Context) {
	c.String(http.StatusOK, "Gin SSO Client\n\nopen: http://localhost:9101/protected\n")
}

// protected redirects unauthenticated users to SSO and serves protected content. protected 将未登录用户重定向到 SSO 并返回受保护内容。
func protected(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	loginID, ok := localLoginID(c.Request)
	if !ok {
		state := rand.Text()
		authURL, err := clientApp.AuthURL(callbackURL, url.Values{clientApp.Config().Params.Back: {state}})
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		sso.SetLoginIDCookie(c.Writer, loginStateCookie, state)
		c.Redirect(http.StatusFound, authURL)
		return
	}
	c.String(http.StatusOK, "Protected resource\n\nloginId: %s\n\nlocal logout: http://localhost:9101/logout\ncenter logout: http://localhost:9100/sso/logout\n", loginID)
}

// callback exchanges the SSO Ticket and creates a local session. callback 交换 SSO Ticket 并创建本地会话。
func callback(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	ticket := c.Query("ticket")
	if ticket == "" {
		c.String(http.StatusBadRequest, "missing ticket")
		return
	}

	// Verify browser binding before consuming the Ticket or changing local identity. 消费 Ticket 或变更本地身份前，校验浏览器绑定。
	state, ok := sso.LoginIDFromCookie(loginStateCookie)(c.Request)
	returnedState := c.Query(clientApp.Config().Params.Back)
	if !ok || returnedState == "" || subtle.ConstantTimeCompare([]byte(state), []byte(returnedState)) != 1 {
		c.String(http.StatusBadRequest, "invalid login state; restart from /protected")
		return
	}
	sso.ClearLoginIDCookie(c.Writer, loginStateCookie)
	result, err := clientApp.ExchangeTicket(c.Request.Context(), ticket, callbackURL)
	if err != nil {
		c.String(http.StatusBadGateway, err.Error())
		return
	}
	if result == nil || result.LoginID == "" {
		c.String(http.StatusBadGateway, "sso response missing loginId")
		return
	}
	sessionID, err := newLocalSession(result.LoginID)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	// Retire the previous browser session only after the new identity is established. 新身份建立后再删除浏览器原有会话。
	if previous, err := c.Request.Cookie(localCookie); err == nil {
		deleteLocalSession(previous.Value)
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     localCookie,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   int(localCookieTTL.Seconds()),
		Expires:  time.Now().Add(localCookieTTL),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	c.Redirect(http.StatusFound, "/protected")
}

// logout clears the local session and login cookie. logout 清理本地会话和登录 Cookie。
func logout(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	if cookie, err := c.Request.Cookie(localCookie); err == nil {
		deleteLocalSession(cookie.Value)
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     localCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	c.Redirect(http.StatusFound, "/")
}

// logoutCallback removes local sessions after single logout. logoutCallback 在单点登出后删除本地会话。
func logoutCallback(_ *http.Request, callback sso.LogoutCallback) error {
	deleteLocalSessionsByLoginID(callback.LoginID)
	return nil
}

// localLoginID resolves the SSO login ID from a local session cookie. localLoginID 根据本地会话 Cookie 解析 SSO 登录 ID。
func localLoginID(r *http.Request) (string, bool) {
	if r == nil {
		return "", false
	}
	cookie, err := r.Cookie(localCookie)
	if err != nil || cookie.Value == "" {
		return "", false
	}
	localSessions.mu.Lock()
	defer localSessions.mu.Unlock()
	session, ok := localSessions.values[cookie.Value]
	if !ok || !time.Now().Before(session.expiresAt) {
		delete(localSessions.values, cookie.Value)
		return "", false
	}
	return session.loginID, session.loginID != ""
}

// newLocalSession creates a local session for an SSO login ID. newLocalSession 为 SSO 登录 ID 创建本地会话。
func newLocalSession(loginID string) (string, error) {
	if loginID == "" {
		return "", fmt.Errorf("login id is empty")
	}
	var randomBytes [32]byte
	if _, err := rand.Read(randomBytes[:]); err != nil {
		return "", err
	}
	sessionID := hex.EncodeToString(randomBytes[:])
	localSessions.mu.Lock()
	defer localSessions.mu.Unlock()

	// Reclaim expired records during login without a background goroutine. 登录时回收过期记录，无需后台协程。
	now := time.Now()
	for id, session := range localSessions.values {
		if !now.Before(session.expiresAt) {
			delete(localSessions.values, id)
		}
	}
	localSessions.values[sessionID] = localSession{loginID: loginID, expiresAt: now.Add(localCookieTTL)}
	return sessionID, nil
}

// deleteLocalSession removes one local session. deleteLocalSession 删除一个本地会话。
func deleteLocalSession(sessionID string) {
	localSessions.mu.Lock()
	defer localSessions.mu.Unlock()
	delete(localSessions.values, sessionID)
}

// deleteLocalSessionsByLoginID removes all local sessions for a login ID. deleteLocalSessionsByLoginID 删除指定登录 ID 的全部本地会话。
func deleteLocalSessionsByLoginID(loginID string) {
	localSessions.mu.Lock()
	defer localSessions.mu.Unlock()
	for sessionID, session := range localSessions.values {
		if session.loginID == loginID {
			delete(localSessions.values, sessionID)
		}
	}
}
