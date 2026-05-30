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
)

const (
	addr           = ":9001"
	callbackURL    = "http://localhost:9001/sso/callback"
	clientID       = "demo-client"
	clientSecret   = "demo-secret"
	localCookie    = "demo_client_login"
	localCookieTTL = 2 * time.Hour
)

var localSessions = struct {
	mu     sync.RWMutex
	values map[string]string
}{values: make(map[string]string)}

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

func home(w http.ResponseWriter, _ *http.Request) {
	_, _ = fmt.Fprint(w, "SSO Client\n\nopen: http://localhost:9001/protected\n")
}

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
	sessionID := newLocalSession(result.LoginID)
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
