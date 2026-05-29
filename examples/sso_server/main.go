// @Author daixk 2026/05/29
package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/Zany2/dtoken-go/sso"
)

const (
	addr         = ":9000"
	callbackURL  = "http://localhost:9001/sso/callback"
	clientID     = "demo-client"
	clientSecret = "demo-secret"
)

var (
	cookie = sso.CookieOptions{
		Name:     "dtoken_sso_demo",
		Path:     "/",
		MaxAge:   2 * time.Hour,
		HTTPOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	loginPage = template.Must(template.New("login").Parse(loginHTML))
)

func main() {
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

	mux := http.NewServeMux()
	httpSSO.Register(mux)
	mux.HandleFunc("/", home)
	mux.HandleFunc("/login", login)

	log.Printf("SSO server listening on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func home(w http.ResponseWriter, r *http.Request) {
	loginID, ok := sso.LoginIDFromCookie(cookie)(r)
	if !ok {
		loginID = "not logged in"
	}
	_, _ = fmt.Fprintf(w, "SSO Server\n\nloginId: %s\n\nlogin: http://localhost:9000/login\n", loginID)
}

func login(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		back := r.URL.Query().Get("back")
		if back == "" {
			back = "/"
		}
		_ = loginPage.Execute(w, map[string]string{"Back": back})
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		loginID := r.FormValue("loginId")
		if loginID == "" {
			loginID = "user-1001"
		}
		sso.SetLoginIDCookie(w, cookie, loginID)
		back := r.FormValue("back")
		if back == "" {
			back = "/"
		}
		http.Redirect(w, r, back, http.StatusFound)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

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
