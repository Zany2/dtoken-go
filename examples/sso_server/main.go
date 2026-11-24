// @Author daixk 2026/05/29
package main

import (
	"crypto/rand"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	dtoken "github.com/Zany2/dtoken-go"
	"github.com/Zany2/dtoken-go/sso"
)

// Example server settings define the listen address and demo client credentials. Example server settings 定义监听地址和示例客户端凭证。
const (
	addr         = "localhost:9000"
	callbackURL  = "http://localhost:9001/sso/callback"
	clientID     = "demo-client"
	clientSecret = "demo-secret"
)

var (
	// cookie configures the demo SSO login cookie. cookie 配置示例 SSO 登录 Cookie。
	cookie = sso.CookieOptions{
		Name:     "dtoken_sso_demo",
		Path:     "/",
		MaxAge:   2 * time.Hour,
		HTTPOnly: true,
		SameSite: http.SameSiteLaxMode,

		// A process-local key prevents reuse of publicly known demo signatures. 进程独立密钥避免复用公开示例签名。
		SecretKey: rand.Text(),
	}
	loginPage = template.Must(template.New("login").Parse(loginHTML)) // loginPage stores the parsed login page template. loginPage 保存已解析的登录页模板。
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run owns the demo server so returning errors still release its storage. run 持有示例服务端，确保错误返回时仍释放存储。
func run() error {
	auth, err := newCenterAuth()
	if err != nil {
		return err
	}
	defer auth.Close()
	server := sso.NewServer()
	defer server.Close()
	handler, err := newDemoHandler(server, auth)
	if err != nil {
		return err
	}
	log.Printf("SSO server listening on http://%s", addr)
	return http.ListenAndServe(addr, handler)
}

// newDemoHandler shares the actual demo routes with regression tests. newDemoHandler 与回归测试共用实际示例路由。
func newDemoHandler(server *sso.Server, auth *dtoken.Auth) (http.Handler, error) {
	center := &demoCenter{auth: auth}

	// Register the demo client accepted by the SSO server. 注册 SSO 服务端接受的示例客户端。
	if err := server.RegisterClientChecked(&sso.Client{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Name:         "Demo Client",
		RedirectURIs: []string{callbackURL},
		Modes:        []sso.Mode{sso.ModeTicket},
		Scopes:       []string{"profile", "email"},
	}); err != nil {
		return nil, err
	}

	// Bind protocol issuance, exchange, and logout to the same center Auth. 将协议签发、换票和注销绑定到同一中心 Auth。
	httpSSO, err := sso.NewHTTPServerChecked(server, sso.HTTPOptions{
		ServerOptions: sso.ServerOptions{
			EnableSLO:                 true,
			LogoutCallbackConcurrency: 4,
			LogoutCallbackMaxAttempts: 1,
			CheckSign:                 false,
			Endpoints:                 sso.DefaultEndpoints(),
			Params:                    sso.DefaultParamNames(),
		},
		LoginIDResolver:       center.loginID,
		AuthorizationExtra:    center.authorizationExtra,
		EnrichExchangeResult:  center.enrichExchange,
		CheckCredentialActive: center.checkCredentialActive,
		CenterLogout:          center.logout,
		OnLogoutResult:        observeLogoutResult,
		LoginPageURL:          "http://localhost:9000/login",
		Cookie:                cookie,
	})
	if err != nil {
		return nil, err
	}

	// Register SSO endpoints and local demo pages. 注册 SSO 端点和本地示例页面。
	mux := http.NewServeMux()
	httpSSO.Register(mux)
	mux.HandleFunc("/", center.home)

	// Protect browser login submissions without restricting server-to-server SSO POSTs. 保护浏览器登录提交，不限制服务间的 SSO POST 请求。
	mux.Handle("/login", http.NewCrossOriginProtection().Handler(http.HandlerFunc(center.login)))
	return mux, nil
}

// observeLogoutResult logs safe callback diagnostics for both strict and best-effort policies. observeLogoutResult 为严格与尽力策略记录安全的回调诊断信息。
func observeLogoutResult(_ *http.Request, result sso.LogoutResult) {
	if result.Cause != nil {
		log.Printf("SSO logout stopped at stage %s", result.Stage)
	}
	if result.Notifications != nil {
		for _, target := range result.Notifications.Clients {
			if target.Status == sso.LogoutCallbackFailed || target.Status == sso.LogoutCallbackCanceled {
				log.Printf("SSO logout client=%q status=%s attempts=%d reason=%s", target.ClientID, target.Status, target.Attempts, target.Reason)
			}
		}
	}
}

// home renders the SSO server status page. home 渲染 SSO 服务端状态页。
func (c *demoCenter) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	loginID, ok := c.loginID(r)
	if !ok {
		loginID = "not logged in"
	}
	_, _ = fmt.Fprintf(w, "SSO Server\n\nloginId: %s\n\nlogin: http://localhost:9000/login\nlogout: http://localhost:9000/sso/logout\n", loginID)
}

// login renders the login page and handles demo login submission. login 渲染登录页并处理示例登录提交。
func (c *demoCenter) login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	switch r.Method {
	case http.MethodGet:
		// Render the page with the requested return location. 使用请求的返回地址渲染页面。
		back := safeBack(r.URL.Query().Get("back"))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = loginPage.Execute(w, map[string]string{"Back": back})
	case http.MethodPost:
		// Persist the demo login ID and return to the original page. 保存示例登录 ID 并返回原页面。
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		loginID := r.PostForm.Get("loginId")
		if loginID == "" {
			loginID = "user-1001"
		}
		if err := c.startSession(w, r, loginID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		back := safeBack(r.PostForm.Get("back"))
		http.Redirect(w, r, back, http.StatusFound)
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// safeBack accepts only local paths to prevent open redirects. safeBack 仅接受站内路径以防止开放重定向。
func safeBack(raw string) string {
	if raw == "" {
		return "/"
	}
	target, err := url.Parse(raw)
	if err != nil || target.IsAbs() || target.Host != "" || target.User != nil || strings.Contains(target.Path, "\\") || !strings.HasPrefix(target.Path, "/") || strings.HasPrefix(target.Path, "//") {
		return "/"
	}
	return target.RequestURI()
}

// loginHTML defines the demo login page template. loginHTML 定义示例登录页模板。
const loginHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>DToken-Go SSO Server</title>
</head>
<body>
  <h1>SSO Login Center</h1>
  <form method="post" action="/login">
    <input type="hidden" name="back" value="{{.Back}}">
    <label>Login ID <input name="loginId" value="user-1001"></label>
    <button type="submit">Login</button>
  </form>
</body>
</html>`
