// @Author daixk 2026/05/29
package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Zany2/dtoken-go/sso"
)

// Example server settings define the listen address and demo client credentials. Example server settings 定义监听地址和示例客户端凭证。
const (
	addr         = ":9000"
	callbackURL  = "http://localhost:9001/sso/callback"
	clientID     = "demo-client"
	clientSecret = "demo-secret"
	cookieSecret = "demo-cookie-signing-secret"
)

var (
	// cookie configures the demo SSO login cookie. cookie 配置示例 SSO 登录 Cookie。
	cookie = sso.CookieOptions{
		Name:      "dtoken_sso_demo",
		Path:      "/",
		MaxAge:    2 * time.Hour,
		HTTPOnly:  true,
		SameSite:  http.SameSiteLaxMode,
		SecretKey: cookieSecret,
	}
	// loginPage stores the parsed login page template. loginPage 保存已解析的登录页模板。
	loginPage = template.Must(template.New("login").Parse(loginHTML))
)

func main() {
	// Register the demo client accepted by the SSO server. 注册 SSO 服务端接受的示例客户端。
	server := sso.NewServer()
	if err := server.RegisterClient(&sso.Client{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Name:         "Demo Client",
		RedirectURIs: []string{callbackURL},
		Modes:        []sso.Mode{sso.ModeTicket},
		Scopes:       []string{"profile", "email"},
	}); err != nil {
		log.Fatal(err)
	}

	// Configure protocol endpoints and cookie-based login resolution. 配置协议端点和基于 Cookie 的登录解析。
	httpSSO := sso.NewHTTPServer(server, sso.HTTPOptions{
		ServerOptions: sso.ServerOptions{
			EnableSLO: true,
			CheckSign: false,
			Endpoints: sso.DefaultEndpoints(),
			Params:    sso.DefaultParamNames(),
		},
		LoginIDResolver: sso.LoginIDFromCookie(cookie),
		LoginPageURL:    "http://localhost:9000/login",
		Cookie:          cookie,
	})

	// Register SSO endpoints and local demo pages. 注册 SSO 端点和本地示例页面。
	mux := http.NewServeMux()
	httpSSO.Register(mux)
	mux.HandleFunc("/", home)
	mux.HandleFunc("/login", login)

	log.Printf("SSO server listening on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

// home renders the SSO server status page. home 渲染 SSO 服务端状态页。
func home(w http.ResponseWriter, r *http.Request) {
	loginID, ok := sso.LoginIDFromCookie(cookie)(r)
	if !ok {
		loginID = "not logged in"
	}
	_, _ = fmt.Fprintf(w, "SSO Server\n\nloginId: %s\n\nlogin: http://localhost:9000/login\n", loginID)
}

// login renders the login page and handles demo login submission. login 渲染登录页并处理示例登录提交。
func login(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Render the page with the requested return location. 使用请求的返回地址渲染页面。
		back := safeBack(r.URL.Query().Get("back"))
		_ = loginPage.Execute(w, map[string]string{"Back": back})
	case http.MethodPost:
		// Persist the demo login ID and return to the original page. 保存示例登录 ID 并返回原页面。
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		loginID := r.FormValue("loginId")
		if loginID == "" {
			loginID = "user-1001"
		}
		sso.SetLoginIDCookie(w, cookie, loginID)
		back := safeBack(r.FormValue("back"))
		http.Redirect(w, r, back, http.StatusFound)
	default:
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
