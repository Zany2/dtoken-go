package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/sso"
)

// TestBrowserDemoLifecycle checks the reusable integration against application-owned sessions. TestBrowserDemoLifecycle 检查可复用接入与应用管理会话的组合。
func TestBrowserDemoLifecycle(t *testing.T) {
	resetClientSessions(t)
	center := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Error(err)
			http.Error(w, "bad form", 400)
			return
		}
		if r.URL.Path != "/sso/token" || r.Form.Get("redirect") != callbackURL {
			t.Errorf("unexpected exchange: %s %v", r.URL.Path, r.Form)
		}
		if _, err := sso.PKCEChallenge(r.PostForm.Get("code_verifier")); err != nil || r.URL.RawQuery != "" {
			t.Error("browser example must submit a PKCE verifier in the POST body")
		}
		_ = json.NewEncoder(w).Encode(sso.OKResponse(sso.TicketExchangeResult{LoginID: "alice", RemainTokenTimeout: 45}))
	}))
	defer center.Close()
	previous := clientApp
	clientApp = sso.NewClientApp(sso.ClientConfig{ClientID: clientID, ClientSecret: clientSecret, ServerURL: center.URL})
	t.Cleanup(func() { clientApp = previous })
	handler, closeStorage, err := newBrowserDemoHandler()
	if err != nil {
		t.Fatal(err)
	}
	defer closeStorage()
	start := httptest.NewRecorder()
	handler.ServeHTTP(start, httptest.NewRequest("GET", "http://localhost:9001/protected", nil))
	authURL, err := url.Parse(start.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if start.Code != 302 || len(start.Result().Cookies()) != 1 {
		t.Fatalf("start: %d", start.Code)
	}
	if authURL.Query().Get("code_challenge_method") != sso.PKCEMethodS256 || len(authURL.Query().Get("code_challenge")) != 43 || authURL.Query().Has("code_verifier") {
		t.Fatal("browser example must authorize with an S256 challenge only")
	}
	old, err := newLocalSession("old-account")
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("GET", callbackURL+"?"+url.Values{"ticket": {"ticket"}, "back": {authURL.Query().Get("back")}}.Encode(), nil)
	r.AddCookie(start.Result().Cookies()[0])
	r.AddCookie(&http.Cookie{Name: localCookie, Value: old})
	completed := httptest.NewRecorder()
	handler.ServeHTTP(completed, r)
	if completed.Code != 303 || completed.Header().Get("Location") != "/protected" {
		t.Fatalf("callback: %d %s", completed.Code, completed.Body.String())
	}
	var sessionCookie *http.Cookie
	for _, cookie := range completed.Result().Cookies() {
		if cookie.Name == localCookie {
			sessionCookie = cookie
		}
	}
	if sessionCookie == nil {
		t.Fatal("missing application session")
	}
	localSessions.mu.RLock()
	_, oldExists := localSessions.values[old]
	session := localSessions.values[sessionCookie.Value]
	localSessions.mu.RUnlock()
	if oldExists || session.loginID != "alice" || time.Until(session.expiresAt) > 45*time.Second {
		t.Fatalf("session replacement: old=%v new=%+v", oldExists, session)
	}
	logoutRequest := httptest.NewRequest("POST", "http://localhost:9001/logout-all", nil)
	logoutRequest.Header.Set("Origin", "http://localhost:9001")
	logoutRequest.AddCookie(sessionCookie)
	out := httptest.NewRecorder()
	handler.ServeHTTP(out, logoutRequest)
	target, err := url.Parse(out.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if out.Code != 303 || target.Path != "/sso/logout" {
		t.Fatalf("global logout: %d %s", out.Code, out.Body.String())
	}
	if _, ok := localLoginID(logoutRequest); ok {
		t.Fatal("local session survived logout")
	}
}

// TestBrowserDemoFailedReplacementPreservesSession keeps the prior identity when the new deadline is invalid. TestBrowserDemoFailedReplacementPreservesSession 在新截止时间无效时保留原身份。
func TestBrowserDemoFailedReplacementPreservesSession(t *testing.T) {
	resetClientSessions(t)
	old, err := newLocalSession("old-account")
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("GET", callbackURL, nil)
	r.AddCookie(&http.Cookie{Name: localCookie, Value: old})
	w := httptest.NewRecorder()
	err = establishBrowserSession(w, r, &sso.BrowserLoginResult{Identity: sso.TicketExchangeResult{LoginID: "new-account"}, ExpiresAt: time.Now().Add(-time.Second)})
	if err == nil {
		t.Fatal("created an expired session")
	}
	if loginID, ok := localLoginID(r); !ok || loginID != "old-account" || len(w.Result().Cookies()) != 0 {
		t.Fatalf("failed replacement altered prior session: %q %v", loginID, ok)
	}
}
