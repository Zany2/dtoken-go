package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Zany2/dtoken-go/sso"
)

// newBrowserDemoHandler composes reusable login handlers with this application's existing sessions. newBrowserDemoHandler 将可复用登录处理器与应用已有会话组合。
func newBrowserDemoHandler() (http.Handler, func(), error) {
	storage := sso.NewMemoryStorage()
	browser, err := sso.NewBrowserClient(clientApp, sso.BrowserClientOptions{
		PKCE:    true,
		Storage: storage, CallbackURL: callbackURL,
		StateCookie: loginStateCookie, SessionMaxAge: localCookieTTL,
	})
	// In-memory storage owns no background resources; retain the cleanup hook for other adapters. 内存存储没有后台资源，保留清理钩子供其他适配器使用。
	closeStorage := func() {}
	if err != nil {
		closeStorage()
		return nil, nil, err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		_, _ = fmt.Fprint(w, "Browser SSO example: open /protected\n")
	})
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := browser.BeginLogin(w, r, "/protected", nil); err != nil {
			http.Error(w, "cannot start login", http.StatusInternalServerError)
		}
	})
	mux.HandleFunc("/protected", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if _, ok := localLoginID(r); !ok {
			if err := browser.BeginLogin(w, r, "/protected", nil); err != nil {
				http.Error(w, "cannot start login", http.StatusInternalServerError)
			}
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprint(w, `<p>Signed in</p><form method="post" action="/logout"><button>Local logout</button></form><form method="post" action="/logout-all"><button>Global logout</button></form>`)
	})
	mux.HandleFunc("/sso/callback", browser.CallbackHandler(establishBrowserSession))
	mux.HandleFunc("/logout", browser.LogoutHandler(clearBrowserSession, false, "/"))
	mux.HandleFunc("/logout-all", browser.LogoutHandler(clearBrowserSession, true, "/"))
	mux.HandleFunc("/sso/logout-callback", clientApp.LogoutCallbackHandler(logoutCallback))
	return mux, closeStorage, nil
}

// establishBrowserSession creates the replacement before retiring the previous browser session. establishBrowserSession 创建替代会话后才撤销浏览器旧会话。
func establishBrowserSession(w http.ResponseWriter, r *http.Request, result *sso.BrowserLoginResult) error {
	sessionID, err := newLocalSessionUntil(result.Identity.LoginID, result.ExpiresAt)
	if err != nil {
		return err
	}
	if previous, err := r.Cookie(localCookie); err == nil {
		deleteLocalSession(previous.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name: localCookie, Value: sessionID, Path: "/",
		Expires: result.ExpiresAt, MaxAge: max(1, int(time.Until(result.ExpiresAt)/time.Second)),
		HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
	return nil
}

// clearBrowserSession revokes server state before clearing the local browser cookie. clearBrowserSession 先撤销服务端状态再清除本地浏览器 Cookie。
func clearBrowserSession(w http.ResponseWriter, r *http.Request) error {
	if cookie, err := r.Cookie(localCookie); err == nil {
		deleteLocalSession(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name: localCookie, Path: "/", MaxAge: -1, Expires: time.Unix(1, 0),
		HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
	return nil
}
