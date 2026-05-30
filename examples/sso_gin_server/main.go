// @Author daixk 2026/05/29
package main

import (
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/Zany2/dtoken-go/sso"
	"github.com/gin-gonic/gin"
)

const (
	addr         = ":9100"
	callbackURL  = "http://localhost:9101/sso/callback"
	clientID     = "gin-demo-client"
	clientSecret = "gin-demo-secret"
)

var (
	cookie = sso.CookieOptions{
		Name:     "dtoken_sso_gin_demo",
		Path:     "/",
		MaxAge:   2 * time.Hour,
		HTTPOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	loginPage = template.Must(template.New("login").Parse(loginHTML))
)

func main() {
	gin.SetMode(gin.ReleaseMode)

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

func ginWrap(handler http.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		handler(c.Writer, c.Request)
	}
}

func home(c *gin.Context) {
	loginID, ok := sso.LoginIDFromCookie(cookie)(c.Request)
	if !ok {
		loginID = "not logged in"
	}
	c.String(http.StatusOK, "Gin SSO Server\n\nloginId: %s\n\nlogin: http://localhost:9100/login\nlogout: http://localhost:9100/sso/logout?loginId=%s\n", loginID, loginID)
}

func loginPageHandler(c *gin.Context) {
	back := c.Query("back")
	if back == "" {
		back = "/"
	}
	c.Status(http.StatusOK)
	_ = loginPage.Execute(c.Writer, map[string]string{"Back": back})
}

func loginSubmit(c *gin.Context) {
	loginID := c.PostForm("loginId")
	if loginID == "" {
		loginID = "user-1001"
	}
	sso.SetLoginIDCookie(c.Writer, cookie, loginID)
	back := c.PostForm("back")
	if back == "" {
		back = "/"
	}
	c.Redirect(http.StatusFound, back)
}

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
