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
)

// Example client settings define local endpoints, credentials, and cookie lifetime. Example client settings 定义本地端点、凭证和 Cookie 有效期。
const (
	addr           = ":9001"
	callbackURL    = "http://localhost:9001/sso/callback"
	clientID       = "demo-client"
	clientSecret   = "demo-secret"
	localCookie    = "demo_client_login"
	localCookieTTL = 2 * time.Hour
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
	ServerURL:         "http://localhost:9000",
	RegisterCallback:  true,
	LogoutCallbackURL: "http://localhost:9001/sso/logout-callback",
	CheckSign:         false,
	Endpoints:         sso.DefaultEndpoints(),
	Params:            sso.DefaultParamNames(),
})

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", home)
	mux.HandleFunc("/protected", protected)
	mux.HandleFunc("/sso/callback", callback)
	mux.HandleFunc("/sso/logout-callback", clientApp.LogoutCallbackHandler(logoutCallback))
	mux.HandleFunc("/logout", logout)

	log.Printf("SSO client listening on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

// home renders the SSO client landing page. home 渲染 SSO 客户端首页。
func home(w http.ResponseWriter, _ *http.Request) {
	_, _ = fmt.Fprint(w, "SSO Client\n\nopen: http://localhost:9001/protected\n")
}

// protected redirects unauthenticated users to SSO and serves protected content. protected 将未登录用户重定向到 SSO 并返回受保护内容。
func protected(w http.ResponseWriter, r *http.Request) {
	loginID, ok := localLoginID(r)
	if !ok {
		authURL, err := clientApp.AuthURL(callbackURL, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, authURL, http.StatusFound)
		return
	}
	_, _ = fmt.Fprintf(w, "Protected resource\n\nloginId: %s\n\nlogout: http://localhost:9001/logout\n", loginID)
}

// callback exchanges the SSO Ticket and creates a local session. callback 交换 SSO Ticket 并创建本地会话。
func callback(w http.ResponseWriter, r *http.Request) {
	ticket := r.URL.Query().Get("ticket")
	if ticket == "" {
		http.Error(w, "missing ticket", http.StatusBadRequest)
		return
	}
	result, err := clientApp.ExchangeTicket(r.Context(), ticket, callbackURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	if result == nil || result.LoginID == "" {
		http.Error(w, "sso response missing loginId", http.StatusBadGateway)
		return
	}
	sessionID, err := newLocalSession(result.LoginID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     localCookie,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   int(localCookieTTL.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/protected", http.StatusFound)
}

// logout clears the local session and login cookie. logout 清理本地会话和登录 Cookie。
func logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(localCookie); err == nil {
		deleteLocalSession(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     localCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/", http.StatusFound)
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
