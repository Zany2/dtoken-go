package main

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	dtoken "github.com/Zany2/dtoken-go"
	"github.com/Zany2/dtoken-go/contract"
	"github.com/Zany2/dtoken-go/sso"
)

// application composes shared storage with independently namespaced Auth and SSO services. application 将共享存储与独立命名空间的 Auth 和 SSO 服务组装。
type application struct {
	cfg        deploymentConfig
	auth       *dtoken.Auth
	protocol   *sso.Server
	centerHTTP *sso.HTTPServer
	client     *sso.ClientApp
	cookie     sso.CookieOptions
	handler    http.Handler
}

// newApplication borrows storage and transport; the process closes them after draining HTTP requests. newApplication 借用存储和传输，进程在 HTTP 请求结束后负责关闭。
func newApplication(cfg deploymentConfig, storage contract.Storage, transport http.RoundTripper) (*application, error) {
	if storage == nil {
		return nil, errors.New("shared storage is required")
	}
	auth, err := dtoken.NewBuilder().AuthType("browser").KeyPrefix(cfg.storagePrefix() + "auth:").
		Storage(storage).UseManagerOption(dtoken.WithComponentOwnership(dtoken.ComponentOwnership{Logger: true, Pool: true})).
		TimeoutDuration(2 * time.Hour).AutoRenew(false).IsConcurrent(true).IsShare(false).IsLog(false).IsPrintBanner(false).BuildInstance()
	if err != nil {
		return nil, err
	}
	a := &application{cfg: cfg, auth: auth}
	name := "sso_" + cfg.namespace + "_center"
	if cfg.role == "client" {
		name = "sso_" + cfg.namespace + "_client_" + cfg.clientID
	}
	a.cookie = sso.CookieOptions{Name: name, Path: "/", MaxAge: 2 * time.Hour, HTTPOnly: true, Secure: strings.HasPrefix(cfg.publicURL, "https://"), SameSite: http.SameSiteLaxMode, SecretKey: cfg.cookieKey}
	a.cookie.VerificationKey = cfg.cookieVerificationKey
	replay, err := sso.NewReplayProtection(sso.ReplayProtectionOptions{Storage: storage, KeyPrefix: cfg.storagePrefix() + "replay:"})
	if err != nil {
		a.close()
		return nil, err
	}
	mux := http.NewServeMux()
	if cfg.role == "center" {
		err = a.mountCenter(mux, storage, replay, transport)
	} else {
		err = a.mountClient(mux, storage, replay, transport)
	}
	if err != nil {
		a.close()
		return nil, err
	}
	mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, r *http.Request) { _, _ = fmt.Fprintln(w, "live") })
	mux.HandleFunc("GET /health/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()
		if err := storage.Ping(ctx); err != nil {
			http.Error(w, "storage unavailable", http.StatusServiceUnavailable)
			return
		}
		_, _ = fmt.Fprintln(w, "ready")
	})
	a.handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// SSO form redirects cross origins; validate POST Origin in handlers instead of blocking that redirect chain. SSO 表单重定向跨来源，由处理器校验 POST Origin，避免阻断合法跳转链。
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'")
		r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
		mux.ServeHTTP(w, r)
	})
	return a, nil
}

// close releases services while leaving their shared storage under process ownership. close 释放服务，保持共享存储由进程管理。
func (a *application) close() {
	if a.protocol != nil {
		_ = a.protocol.Close()
	}
	if a.auth != nil {
		a.auth.Close()
	}
}

// session reads signed browser state and validates its server-side lifetime without renewal. session 读取签名浏览器状态并检查服务端有效期，不触发续期。
func (a *application) session(r *http.Request) (*dtoken.TokenIntrospection, error) {
	token, ok := sso.LoginIDFromCookie(a.cookie)(r)
	if !ok {
		return nil, nil
	}
	info, err := a.auth.IntrospectToken(r.Context(), token)
	if err != nil {
		return nil, err
	}
	if !info.Active {
		return nil, nil
	}
	if _, ok := sessionDeadline(info); !ok {
		return nil, nil
	}
	return info, nil
}

// sessionDeadline applies the stored absolute limit to both center and application sessions. sessionDeadline 为中心与子应用会话统一执行存储的绝对期限。
func sessionDeadline(info *dtoken.TokenIntrospection) (time.Time, bool) {
	value, _ := info.Extra["ssoDeadline"].(string)
	deadline, err := time.Parse(time.RFC3339Nano, value)
	return deadline, err == nil && time.Now().Before(deadline)
}

// startSession bounds local access by the original deadline even when storage writes are slow. startSession 即使存储写入缓慢，也按原始截止时间限制本地访问。
func (a *application) startSession(w http.ResponseWriter, r *http.Request, loginID string, deadline time.Time) error {
	ttl := time.Until(deadline)
	if ttl < time.Second {
		return sso.ErrRequestExpired
	}
	token, err := a.auth.Login(r.Context(), dtoken.LoginOptions{LoginID: loginID, Device: "browser", DeviceID: rand.Text(), Timeout: ttl, Extra: map[string]any{"ssoDeadline": deadline.UTC().Format(time.RFC3339Nano)}})
	if err != nil {
		return err
	}
	if !time.Now().Before(deadline) {
		return sso.ErrRequestExpired
	}
	if previous, ok := sso.LoginIDFromCookie(a.cookie)(r); ok {
		if err := a.auth.LogoutByToken(r.Context(), previous); err != nil {
			return err
		}
	}
	if !time.Now().Before(deadline) {
		return sso.ErrRequestExpired
	}
	cookie := a.cookie
	cookie.MaxAge = time.Until(deadline)
	sso.SetLoginIDCookie(w, cookie, token)
	return nil
}

// clearSession revokes the current local token before clearing its cookie. clearSession 撤销当前本地 Token 后再清理 Cookie。
func (a *application) clearSession(w http.ResponseWriter, r *http.Request) error {
	if token, ok := sso.LoginIDFromCookie(a.cookie)(r); ok {
		if err := a.auth.LogoutByToken(r.Context(), token); err != nil {
			return err
		}
	}
	sso.ClearLoginIDCookie(w, a.cookie)
	return nil
}

// mountClient wires PKCE, replay protection, local Auth sessions, and authenticated logout callbacks. mountClient 接入 PKCE、防重放、本地 Auth 会话及已认证注销回调。
func (a *application) mountClient(mux *http.ServeMux, storage contract.Storage, replay *sso.ReplayProtection, transport http.RoundTripper) error {
	config := sso.DefaultClientConfig()
	config.ClientID, config.ClientSecret, config.SecretKey = a.cfg.clientID, a.cfg.clientSecret, a.cfg.signingKey
	config.VerificationKey = a.cfg.verificationKey
	config.ServerURL, config.RegisterCallback, config.LogoutCallbackURL = a.cfg.centerURL, true, a.cfg.publicURL+"/sso/logout-callback"
	config.ReplayProtection = replay
	config.HTTPClient = &http.Client{Transport: transport, Timeout: 10 * time.Second}
	app, err := sso.NewClientAppChecked(config)
	if err != nil {
		return err
	}
	a.client = app
	stateCookie := a.cookie
	stateCookie.Name += "_state"
	stateCookie.MaxAge = 5 * time.Minute
	browser, err := sso.NewBrowserClient(app, sso.BrowserClientOptions{PKCE: true, Storage: storage, KeyPrefix: a.cfg.storagePrefix() + "flows:", CallbackURL: a.cfg.publicURL + "/sso/callback", StateCookie: stateCookie, SessionMaxAge: 2 * time.Hour})
	if err != nil {
		return err
	}
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprint(w, `<a href="/protected">Open protected page</a>`)
	})
	mux.HandleFunc("GET /protected", func(w http.ResponseWriter, r *http.Request) {
		info, err := a.session(r)
		if err != nil {
			http.Error(w, "session storage unavailable", http.StatusServiceUnavailable)
			return
		}
		if info == nil {
			if err := browser.BeginLogin(w, r, "/protected", nil); err != nil {
				http.Error(w, "cannot start login", http.StatusServiceUnavailable)
			}
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = protectedPage.Execute(w, map[string]string{"Client": a.cfg.clientID, "LoginID": info.LoginID})
	})
	mux.HandleFunc("GET /sso/callback", browser.CallbackHandler(func(w http.ResponseWriter, r *http.Request, result *sso.BrowserLoginResult) error {
		return a.startSession(w, r, result.Identity.LoginID, result.ExpiresAt)
	}))
	mux.HandleFunc("/logout", browser.LogoutHandler(a.clearSession, false, "/"))
	mux.HandleFunc("/logout-all", browser.LogoutHandler(a.clearSession, true, "/"))
	mux.HandleFunc("/sso/logout-callback", app.LogoutCallbackHandler(func(r *http.Request, callback sso.LogoutCallback) error {
		return a.auth.LogoutByLoginID(r.Context(), callback.LoginID)
	}))
	return nil
}

var protectedPage = template.Must(template.New("protected").Parse(`<h1>{{.Client}}</h1><p>Signed in as {{.LoginID}}</p><form method="post" action="/logout"><button>Local logout</button></form><form method="post" action="/logout-all"><button>Global logout</button></form>`)) // protectedPage escapes identity and application names. protectedPage 转义身份与应用名称。
