package acceptance

import (
	"html"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
)

// TestLiveDeployment exercises already-running services only after explicit opt-in. TestLiveDeployment 仅显式启用后验收已启动的服务。
func TestLiveDeployment(t *testing.T) {
	if os.Getenv("SSO_ACCEPTANCE_RUN") != "true" {
		t.Skip("set SSO_ACCEPTANCE_RUN=true and explicit targets/credentials for live acceptance")
	}
	readOrigin := func(name string) string {
		t.Helper()
		value, err := acceptanceOrigin(os.Getenv(name))
		if err != nil {
			t.Fatalf("invalid %s: %v", name, err)
		}
		return value
	}
	center, alpha, beta := readOrigin("SSO_ACCEPTANCE_CENTER"), readOrigin("SSO_ACCEPTANCE_ALPHA"), readOrigin("SSO_ACCEPTANCE_BETA")
	if center == alpha || center == beta || alpha == beta {
		t.Fatal("center and applications must have distinct origins")
	}
	username, password := os.Getenv("SSO_ACCEPTANCE_USER"), os.Getenv("SSO_ACCEPTANCE_PASSWORD")
	if username == "" || password == "" {
		t.Fatal("SSO_ACCEPTANCE_USER and SSO_ACCEPTANCE_PASSWORD are required")
	}
	browser, err := newAcceptanceClient(center, alpha, beta)
	if err != nil {
		t.Fatal(err)
	}
	do := func(stage, method, target string, form url.Values, origin string, snapshot []*http.Cookie, want int) *acceptanceResponse {
		t.Helper()
		response, err := browser.request(method, target, form, origin, snapshot)
		if err != nil {
			t.Fatalf("%s: %v", stage, err)
		}
		if response.status != want {
			t.Fatalf("%s: HTTP %d, expected %d", stage, response.status, want)
		}
		if response.header.Get("Cache-Control") != "no-store" {
			t.Fatalf("%s: missing no-store response protection", stage)
		}
		return response
	}
	next := func(stage, from string, response *acceptanceResponse, origin, path string) string {
		t.Helper()
		target, err := browser.redirect(from, response, origin, path)
		if err != nil {
			t.Fatalf("%s: %v", stage, err)
		}
		return target
	}
	sessionNames := make(map[string]string)
	clientIDs := make(map[string]string)
	logoutVerified := false
	capture := func(origin, name string) []*http.Cookie {
		t.Helper()
		cookies := browser.snapshot(origin)
		for _, cookie := range cookies {
			if name != "" && cookie.Name == name && cookie.Value != "" {
				return cookies
			}
		}
		t.Fatal("cannot check replay or revocation without the original named cookie")
		return cookies
	}
	revoked := func(stage, app string, cookies []*http.Cookie) {
		t.Helper()
		response := do(stage, http.MethodGet, app+"/protected", nil, "", cookies, http.StatusFound)
		next(stage, app+"/protected", response, center, "/sso/authorize")
	}
	checkCookies := func(stage, origin string, response *acceptanceResponse) string {
		t.Helper()
		active := 0
		name := ""
		for _, cookie := range response.cookies {
			if cookie.MaxAge < 0 {
				continue
			}
			active++
			name = cookie.Name
			if !cookie.HttpOnly || cookie.Domain != "" || cookie.SameSite != http.SameSiteLaxMode || (strings.HasPrefix(origin, "https://") && !cookie.Secure) {
				t.Fatalf("%s: invalid session/state cookie attributes", stage)
			}
		}
		if active != 1 {
			t.Fatalf("%s: expected one active session/state cookie", stage)
		}
		return name
	}
	// Cleanup uses a new signed logout URL and never replays a previously consumed request. 清理生成新的已签名退出地址，不重放已消费请求。
	t.Cleanup(func() {
		if logoutVerified {
			return
		}
		expected := make([]string, 0, len(clientIDs))
		for _, id := range clientIDs {
			expected = append(expected, id)
		}
		if err := browser.cleanupLogout(alpha, center, expected); err != nil {
			t.Log("cleanup not confirmed; retry global logout from the application after recovery")
		}
	})
	login := func(app string, needsPassword bool) {
		t.Helper()
		start := do("begin login", http.MethodGet, app+"/protected", nil, "", nil, http.StatusFound)
		stateName := checkCookies("browser state", app, start)
		authorize := next("authorize destination", app+"/protected", start, center, "/sso/authorize")
		authorization, _ := url.Parse(authorize)
		ids := authorization.Query()["client"]
		if len(ids) != 1 || ids[0] == "" {
			t.Fatal("authorization must identify exactly one client")
		}
		for other, id := range clientIDs {
			if other != app && id == ids[0] {
				t.Fatal("applications must use distinct client IDs")
			}
		}
		if previous := clientIDs[app]; previous != "" && previous != ids[0] {
			t.Fatal("application changed its client ID during acceptance")
		}
		clientIDs[app] = ids[0]
		response := do("center authorization", http.MethodGet, authorize, nil, "", nil, http.StatusFound)
		if needsPassword {
			page := next("login destination", authorize, response, center, "/login")
			u, _ := url.Parse(page)
			back := u.Query().Get("back")
			resolved, parseErr := url.Parse(center)
			if parseErr != nil {
				t.Fatal("invalid center origin")
			}
			returned, parseErr := resolved.Parse(back)
			if parseErr != nil || returned.String() != authorize {
				t.Fatal("center login did not preserve authorization return target")
			}
			do("login page", http.MethodGet, page, nil, "", nil, http.StatusOK)
			form := url.Values{"username": {username}, "password": {password}, "back": {back}}
			loggedIn := do("center login", http.MethodPost, center+"/login", form, center, nil, http.StatusSeeOther)
			sessionNames[center] = checkCookies("center session", center, loggedIn)
			resume := next("login return", center+"/login", loggedIn, center, "/sso/authorize")
			if resume != authorize {
				t.Fatal("center login changed authorization target")
			}
			response = do("authenticated authorization", http.MethodGet, resume, nil, "", nil, http.StatusFound)
		}
		callback := next("callback destination", authorize, response, app, "/sso/callback")
		state := capture(app, stateName)
		do("oversized callback", http.MethodGet, callback+"&extension="+strings.Repeat("a", 9<<10), nil, "", nil, http.StatusRequestEntityTooLarge)
		completed := do("complete login", http.MethodGet, callback, nil, "", nil, http.StatusSeeOther)
		sessionNames[app] = checkCookies("application session", app, completed)
		protected := next("application return", callback, completed, app, "/protected")
		page := do("protected access", http.MethodGet, protected, nil, "", nil, http.StatusOK)
		if !strings.Contains(html.UnescapeString(string(page.body)), "<p>Signed in as "+username+"</p>") {
			t.Fatal("protected page did not identify the expected test account")
		}
		do("callback replay", http.MethodGet, callback, nil, "", state, http.StatusBadRequest)
	}
	login(alpha, true)
	alphaOld, centerOld := capture(alpha, sessionNames[alpha]), capture(center, sessionNames[center])
	login(beta, false)
	betaOld := capture(beta, sessionNames[beta])
	t.Log("PASS: two application logins, center reuse, cookie attributes, size rejection and callback replay")
	do("logout via GET", http.MethodGet, alpha+"/logout", nil, "", nil, http.StatusMethodNotAllowed)
	do("forged logout origin", http.MethodPost, alpha+"/logout", url.Values{}, "https://untrusted.invalid", nil, http.StatusForbidden)
	do("forged callback", http.MethodPost, beta+"/sso/logout-callback", url.Values{"client": {clientIDs[beta]}, "loginId": {username}}, "", nil, http.StatusUnauthorized)
	do("session after rejected logout", http.MethodGet, alpha+"/protected", nil, "", nil, http.StatusOK)
	local := do("local logout", http.MethodPost, alpha+"/logout", url.Values{}, alpha, nil, http.StatusSeeOther)
	next("local return", alpha+"/logout", local, alpha, "/")
	revoked("old alpha session", alpha, alphaOld)
	do("beta after local logout", http.MethodGet, beta+"/protected", nil, "", betaOld, http.StatusOK)
	login(alpha, false)
	alphaOld = capture(alpha, sessionNames[alpha])
	t.Log("PASS: forged logout rejection, local revocation, other application continuity and center reuse")
	logout := do("global logout start", http.MethodPost, alpha+"/logout-all", url.Values{}, alpha, nil, http.StatusSeeOther)
	target := next("global logout destination", alpha+"/logout-all", logout, center, "/sso/logout")
	complete := do("global logout", http.MethodGet, target, nil, "", nil, http.StatusOK)
	if err := acceptanceLogoutComplete(complete, []string{clientIDs[alpha], clientIDs[beta]}); err != nil {
		t.Fatal(err)
	}
	revoked("revoked alpha", alpha, alphaOld)
	revoked("revoked beta", beta, betaOld)
	start := do("new login after logout", http.MethodGet, beta+"/protected", nil, "", nil, http.StatusFound)
	authorize := next("new authorization", beta+"/protected", start, center, "/sso/authorize")
	loggedOut := do("old center session", http.MethodGet, authorize, nil, "", centerOld, http.StatusFound)
	next("center session revoked", authorize, loggedOut, center, "/login")
	logoutVerified = true
	t.Log("PASS: global logout acknowledgement and revocation of both application sessions and original center session")
}
