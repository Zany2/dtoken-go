// @Author daixk 2026/05/29
package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/Zany2/dtoken-go/sso"
	"github.com/gin-gonic/gin"
)

// Example Gin client settings define local endpoints, credentials, and cookie lifetime. Example Gin client settings 定义本地端点、凭证和 Cookie 有效期。
const (
	addr              = ":9101"
	callbackURL       = "http://localhost:9101/sso/callback"
	logoutCallbackURL = "http://localhost:9101/sso/logout-callback"
	clientID          = "gin-demo-client"
	clientSecret      = "gin-demo-secret"
	localCookie       = "gin_demo_client_login"
	localCookieTTL    = 2 * time.Hour
)

// localSessions maps local session IDs to SSO login IDs. localSessions 将本地会话 ID 映射到 SSO 登录 ID。
var localSessions = struct {
	mu     sync.RWMutex
	values map[string]string
}{values: make(map[string]string)}

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

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.GET("/", home)
	r.GET("/protected", protected)
	r.GET("/sso/callback", callback)
	r.POST("/sso/logout-callback", ginWrap(clientApp.LogoutCallbackHandler(logoutCallback)))
	r.GET("/logout", logout)

	log.Printf("Gin SSO client listening on http://localhost%s", addr)
	log.Fatal(r.Run(addr))
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
	loginID, ok := localLoginID(c.Request)
	if !ok {
		authURL, err := clientApp.AuthURL(callbackURL, nil)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		c.Redirect(http.StatusFound, authURL)
		return
	}
	c.String(http.StatusOK, "Protected resource\n\nloginId: %s\n\nlocal logout: http://localhost:9101/logout\ncenter logout: http://localhost:9100/sso/logout?loginId=%s\n", loginID, loginID)
}

// callback exchanges the SSO Ticket and creates a local session. callback 交换 SSO Ticket 并创建本地会话。
func callback(c *gin.Context) {
	ticket := c.Query("ticket")
	if ticket == "" {
		c.String(http.StatusBadRequest, "missing ticket")
		return
	}
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
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     localCookie,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   int(localCookieTTL.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	c.Redirect(http.StatusFound, "/protected")
}

// logout clears the local session and login cookie. logout 清理本地会话和登录 Cookie。
func logout(c *gin.Context) {
	if cookie, err := c.Request.Cookie(localCookie); err == nil {
		deleteLocalSession(cookie.Value)
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     localCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
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
	localSessions.mu.RLock()
	defer localSessions.mu.RUnlock()
	loginID := localSessions.values[cookie.Value]
	return loginID, loginID != ""
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
	localSessions.values[sessionID] = loginID
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
	for sessionID, storedLoginID := range localSessions.values {
		if storedLoginID == loginID {
			delete(localSessions.values, sessionID)
		}
	}
}
