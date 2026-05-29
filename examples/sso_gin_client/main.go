// @Author daixk 2026/05/29
package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/Zany2/dtoken-go/sso"
	"github.com/gin-gonic/gin"
)

const (
	addr              = ":9101"
	callbackURL       = "http://localhost:9101/sso/callback"
	logoutCallbackURL = "http://localhost:9101/sso/logout-callback"
	clientID          = "gin-demo-client"
	clientSecret      = "gin-demo-secret"
	localCookie       = "gin_demo_client_login"
	localCookieTTL    = 2 * time.Hour
)

var localSessions = struct {
	mu     sync.RWMutex
	values map[string]string
}{values: make(map[string]string)}

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

func ginWrap(handler http.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		handler(c.Writer, c.Request)
	}
}

func home(c *gin.Context) {
	c.String(http.StatusOK, "Gin SSO Client\n\nopen: http://localhost:9101/protected\n")
}

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
	sessionID := newLocalSession(result.LoginID)
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

func logoutCallback(_ *http.Request, callback sso.LogoutCallback) error {
	deleteLocalSessionsByLoginID(callback.LoginID)
	return nil
}

func localLoginID(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(localCookie)
	if err != nil || cookie.Value == "" {
		return "", false
	}
	localSessions.mu.RLock()
	defer localSessions.mu.RUnlock()
	loginID := localSessions.values[cookie.Value]
	return loginID, loginID != ""
}

func newLocalSession(loginID string) string {
	sessionID := fmt.Sprintf("%d-%d", time.Now().UnixNano(), rand.Int63())
	localSessions.mu.Lock()
	defer localSessions.mu.Unlock()
	localSessions.values[sessionID] = loginID
	return sessionID
}

func deleteLocalSession(sessionID string) {
	localSessions.mu.Lock()
	defer localSessions.mu.Unlock()
	delete(localSessions.values, sessionID)
}

func deleteLocalSessionsByLoginID(loginID string) {
	localSessions.mu.Lock()
	defer localSessions.mu.Unlock()
	for sessionID, storedLoginID := range localSessions.values {
		if storedLoginID == loginID {
			delete(localSessions.values, sessionID)
		}
	}
}
