package main

import (
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	dtoken "github.com/Zany2/dtoken-go"
	"github.com/Zany2/dtoken-go/contract"
	"github.com/Zany2/dtoken-go/sso"
)

// mountCenter binds issuance, exchange, inspection, and logout to the same center Auth. mountCenter 将签发、兑换、检查及注销绑定到同一中心 Auth。
func (a *application) mountCenter(mux *http.ServeMux, storage contract.Storage, replay *sso.ReplayProtection, transport http.RoundTripper) error {
	a.protocol = sso.NewServer(sso.WithStorage(storage), sso.WithKeyPrefix(a.cfg.storagePrefix()+"protocol:"))
	options := sso.DefaultHTTPOptions()
	options.Clients, options.Cookie, options.LoginPageURL = a.cfg.clients, a.cookie, a.cfg.publicURL+"/login"
	options.ReplayProtection = replay
	options.LoginIDResolver = func(r *http.Request) (string, bool) {
		info, err := a.session(r)
		if err != nil || info == nil {
			return "", false
		}
		return info.LoginID, true
	}
	options.AuthorizationExtra = a.authorizationExtra
	options.EnrichExchangeResult = a.enrichExchange
	options.CheckCredentialActive = func(r *http.Request, credential sso.CredentialInfo) (bool, error) {
		info, err := a.centerSession(r, credential.LoginID, credential.Extra)
		return info != nil, err
	}
	options.CenterLogout = func(r *http.Request, loginID string) error { return a.auth.LogoutByLoginID(r.Context(), loginID) }
	options.LogoutCallbackConcurrency, options.LogoutCallbackMaxAttempts = 4, 2
	options.LogoutCallbackTimeout = 2 * time.Second
	options.LogoutHTTPClient = &http.Client{Transport: transport, Timeout: 3 * time.Second}
	// Log only bounded status values; raw errors and signed request URLs remain private. 只记录固定状态值，原始错误与签名请求地址保持私有。
	options.OnHTTPError = func(_ *http.Request, status int, _ error) {
		if status >= http.StatusInternalServerError {
			log.Printf("SSO center HTTP failure status=%d", status)
		}
	}

	options.OnLogoutResult = func(_ *http.Request, result sso.LogoutResult) {
		if result.Cause != nil {
			log.Printf("SSO logout stopped at stage %s", result.Stage)
		}
		if result.Notifications != nil {
			for _, client := range result.Notifications.Clients {
				if client.Status == sso.LogoutCallbackFailed || client.Status == sso.LogoutCallbackCanceled {
					log.Printf("SSO logout client=%q status=%s attempts=%d reason=%s", client.ClientID, client.Status, client.Attempts, client.Reason)
				}
			}
		}
	}
	handler, err := sso.NewHTTPServerChecked(a.protocol, options)
	if err != nil {
		return err
	}
	a.centerHTTP = handler
	handler.Register(mux)
	mux.HandleFunc("/login", a.login)
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprint(w, `<h1>SSO login center</h1><p>Start sign-in or global logout from a registered application.</p>`)
	})
	return nil
}

// login demonstrates an explicit fixed-account authenticator; replace this credential check with the real identity system. login 演示显式配置的固定账号认证，接入时应替换为实际身份系统。
func (a *application) login(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = loginPage.Execute(w, localBack(r.URL.Query().Get("back")))
	case http.MethodPost:
		// Compare against configured public origin, never untrusted forwarding headers. 与配置的公开来源比较，不信任转发头。
		if len(r.Header.Values("Origin")) != 1 || r.Header.Get("Origin") != a.cfg.publicURL || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
			http.Error(w, "invalid login origin", http.StatusForbidden)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid login form", http.StatusBadRequest)
			return
		}
		actualUser, actualPassword := sha256.Sum256([]byte(r.PostForm.Get("username"))), sha256.Sum256([]byte(r.PostForm.Get("password")))
		expectedUser, expectedPassword := sha256.Sum256([]byte(a.cfg.demoUser)), sha256.Sum256([]byte(a.cfg.demoPassword))
		if subtle.ConstantTimeCompare(actualUser[:], expectedUser[:])&subtle.ConstantTimeCompare(actualPassword[:], expectedPassword[:]) != 1 {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		if err := a.startSession(w, r, a.cfg.demoUser, time.Now().Add(2*time.Hour)); err != nil {
			http.Error(w, "cannot create center session", http.StatusServiceUnavailable)
			return
		}
		http.Redirect(w, r, localBack(r.PostForm.Get("back")), http.StatusSeeOther)
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// localBack permits only local paths and rejects encoded control or separator tricks. localBack 仅允许站内路径，拒绝编码的控制字符和分隔符绕过。
func localBack(value string) string {
	u, err := url.Parse(value)
	if err != nil || u.IsAbs() || u.Host != "" || u.User != nil || !strings.HasPrefix(value, "/") || strings.HasPrefix(u.Path, "//") || strings.Contains(u.Path, "\\") || strings.IndexFunc(u.Path, unicode.IsControl) >= 0 {
		return "/"
	}
	return u.RequestURI()
}

// sessionRef exposes a non-bearer reference to the original center token. sessionRef 对外提供原中心 Token 的非登录凭据引用。
func sessionRef(token string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(token))) }

// authorizationExtra binds issued credentials to a verified center session. authorizationExtra 将签发凭证绑定到已验证的中心会话。
func (a *application) authorizationExtra(r *http.Request, _, loginID string) (map[string]any, error) {
	token, ok := sso.LoginIDFromCookie(a.cookie)(r)
	if !ok {
		return nil, sso.ErrInvalidTicket
	}
	info, err := a.session(r)
	if err != nil {
		return nil, err
	}
	if info == nil || info.LoginID != loginID {
		return nil, sso.ErrInvalidTicket
	}
	return map[string]any{"centerSessionID": sessionRef(token)}, nil
}

// centerSession finds the original active token without trusting a new browser cookie or renewing it. centerSession 查找原始活动 Token，不信任其他浏览器 Cookie 且不续期。
func (a *application) centerSession(r *http.Request, loginID string, extra map[string]any) (*dtoken.TokenIntrospection, error) {
	ref, _ := extra["centerSessionID"].(string)
	if ref == "" {
		return nil, nil
	}
	tokens, err := a.auth.GetTokenValueListByLoginID(r.Context(), loginID, false)
	if err != nil {
		return nil, err
	}
	for _, token := range tokens {
		if sessionRef(token) != ref {
			continue
		}
		info, err := a.auth.IntrospectToken(r.Context(), token)
		if err != nil {
			return nil, err
		}
		if !info.Active || info.LoginID != loginID || info.ExpiresIn <= 0 {
			return nil, nil
		}
		deadline, ok := sessionDeadline(info)
		if !ok {
			return nil, nil
		}
		info.ExpiresIn = min(info.ExpiresIn, int64(time.Until(deadline)/time.Second))
		if info.ExpiresIn <= 0 {
			return nil, nil
		}
		return info, nil
	}
	return nil, nil
}

// enrichExchange restricts client sessions to the remaining lifetime of the issuing center session. enrichExchange 将子应用会话限制在签发中心会话的剩余有效期内。
func (a *application) enrichExchange(r *http.Request, _ string, result *sso.TicketExchangeResult) error {
	info, err := a.centerSession(r, result.LoginID, result.Extra)
	if err != nil {
		return err
	}
	if info == nil {
		return sso.ErrInvalidTicket
	}
	result.RemainTokenTimeout, result.DeviceID = info.ExpiresIn, info.DeviceID
	return nil
}

var loginPage = template.Must(template.New("login").Parse(`<h1>SSO example login</h1><form method="post" action="/login"><input type="hidden" name="back" value="{{.}}"><label>Username <input name="username" autocomplete="username" required></label><label>Password <input type="password" name="password" autocomplete="current-password" required></label><button>Sign in</button></form>`)) // loginPage escapes the authorization return path. loginPage 转义授权回跳路径。
