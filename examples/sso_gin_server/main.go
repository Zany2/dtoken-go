// @Author daixk 2026/05/29
package main

import (
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Zany2/dtoken-go/sso"
	"github.com/gin-gonic/gin"
)

// Example Gin server settings define the listen address and demo client credentials. Example Gin server settings 定义监听地址和示例客户端凭证。
const (
	addr         = ":9100"
	callbackURL  = "http://localhost:9101/sso/callback"
	clientID     = "gin-demo-client"
	clientSecret = "gin-demo-secret"
)

var (
	// cookie configures the demo SSO login cookie. cookie 配置示例 SSO 登录 Cookie。
	cookie = sso.CookieOptions{
		Name:     "dtoken_sso_gin_demo",
		Path:     "/",
		MaxAge:   2 * time.Hour,
		HTTPOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	// loginPage stores the parsed login page template. loginPage 保存已解析的登录页模板。
	loginPage = template.Must(template.New("login").Parse(loginHTML))
)

func main() {
	gin.SetMode(gin.ReleaseMode)

	// Register the demo client accepted by the SSO server. 注册 SSO 服务端接受的示例客户端。
	server := sso.NewServer()
	if err := server.RegisterClient(&sso.Client{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Name:         "Gin Demo Client",
		RedirectURIs: []string{callbackURL},
		Modes:        []sso.Mode{sso.ModeTicket},
		Scopes:       []string{"profile", "email"},
	}); err != nil {
		log.Fatal(err)
	}

	// Configure protocol endpoints and cookie-based login resolution. 配置协议端点和基于 Cookie 的登录解析。
	httpSSO := sso.NewHTTPServer(server, sso.HTTPOptions{
		ServerOptions: sso.ServerOptions{
			EnableSLO:                true,
			LogoutCallbackTimeout:    3 * time.Second,
			LogoutCallbackBestEffort: true,
			CheckSign:                false,
			Endpoints:                sso.DefaultEndpoints(),
			Params:                   sso.DefaultParamNames(),
		},
		LoginIDResolver: sso.LoginIDFromCookie(cookie),
		LoginPageURL:    "http://localhost:9100/login",
		Cookie:          cookie,
	})

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	registerSSORoutes(r, httpSSO)
	r.GET("/", home)
	r.GET("/login", loginPageHandler)
	r.POST("/login", loginSubmit)

	log.Printf("Gin SSO server listening on http://localhost%s", addr)
	log.Fatal(r.Run(addr))
}

// registerSSORoutes maps standard SSO endpoints to Gin handlers. registerSSORoutes 将标准 SSO 端点映射为 Gin 处理器。
func registerSSORoutes(r *gin.Engine, httpSSO *sso.HTTPServer) {
	endpoints := sso.DefaultEndpoints()
	r.GET(endpoints.Authorize, ginWrap(httpSSO.HandleAuthorize))
	r.GET(endpoints.Token, ginWrap(httpSSO.HandleToken))
	r.POST(endpoints.Token, ginWrap(httpSSO.HandleToken))
	r.GET(endpoints.Introspect, ginWrap(httpSSO.HandleIntrospect))
	r.POST(endpoints.Introspect, ginWrap(httpSSO.HandleIntrospect))
	r.GET(endpoints.UserInfo, ginWrap(httpSSO.HandleUserInfo))
	r.POST(endpoints.UserInfo, ginWrap(httpSSO.HandleUserInfo))
	r.GET(endpoints.Revoke, ginWrap(httpSSO.HandleRevoke))
	r.POST(endpoints.Revoke, ginWrap(httpSSO.HandleRevoke))
	r.GET(endpoints.Logout, ginWrap(httpSSO.HandleLogout))
	r.POST(endpoints.Logout, ginWrap(httpSSO.HandleLogout))
}

// ginWrap adapts a net/http handler to Gin. ginWrap 将 net/http 处理器适配为 Gin 处理器。
func ginWrap(handler http.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		handler(c.Writer, c.Request)
	}
}

// home renders the Gin SSO server status page. home 渲染 Gin SSO 服务端状态页。
func home(c *gin.Context) {
	loginID, ok := sso.LoginIDFromCookie(cookie)(c.Request)
	if !ok {
		loginID = "not logged in"
	}
	c.String(http.StatusOK, "Gin SSO Server\n\nloginId: %s\n\nlogin: http://localhost:9100/login\nlogout: http://localhost:9100/sso/logout?loginId=%s\n", loginID, loginID)
}

// loginPageHandler renders the demo login page. loginPageHandler 渲染示例登录页。
func loginPageHandler(c *gin.Context) {
	back := safeBack(c.Query("back"))
	c.Status(http.StatusOK)
	_ = loginPage.Execute(c.Writer, map[string]string{"Back": back})
}

// loginSubmit stores the demo login cookie and redirects back. loginSubmit 保存示例登录 Cookie 并重定向返回。
func loginSubmit(c *gin.Context) {
	loginID := c.PostForm("loginId")
	if loginID == "" {
		loginID = "user-1001"
	}
	sso.SetLoginIDCookie(c.Writer, cookie, loginID)
	back := safeBack(c.PostForm("back"))
	c.Redirect(http.StatusFound, back)
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
  <title>DToken-Go Gin SSO Server</title>
</head>
<body>
  <h1>Gin SSO Login Center</h1>
  <form method="post" action="/login">
    <input type="hidden" name="back" value="{{.Back}}">
    <label>Login ID <input name="loginId" value="user-1001"></label>
    <button type="submit">Login</button>
  </form>
  <p>{{printf "%s" "After login, the browser returns to the client app with a Ticket."}}</p>
</body>
</html>`
