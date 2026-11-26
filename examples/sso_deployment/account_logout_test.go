package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/sso"
)

// TestDeploymentAccountLogout exercises real Auth sessions, signed callbacks, replica visibility and administrative retry without a browser session. TestDeploymentAccountLogout 验证实际 Auth 会话、签名回调、跨副本可见性及无浏览器会话的管理重试。
func TestDeploymentAccountLogout(t *testing.T) {
	storage := sso.NewMemoryStorage()
	transport := deploymentTransport{}
	create := func(role, id string) *application {
		a, err := newApplication(testConfiguration(t, role, id), storage, transport)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(a.close)
		return a
	}
	center, replica := create("center", ""), create("center", "")
	alpha, beta := create("client", "alpha"), create("client", "beta")
	transport["localhost:9200"], transport["localhost:9201"], transport["localhost:9202"] = replica.handler, alpha.handler, beta.handler
	alphaCookie, centerCookie := loginThroughReplicas(t, alpha, alpha, center, nil)
	betaCookie, _ := loginThroughReplicas(t, beta, beta, replica, centerCookie)
	otherCookies := make(map[*application]*http.Cookie)
	for _, a := range []*application{center, alpha, beta} {
		w := httptest.NewRecorder()
		if err := a.startSession(w, httptest.NewRequest("POST", a.cfg.publicURL+"/login", nil), "bob", time.Now().Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
		otherCookies[a] = responseCookie(t, w, a.cookie.Name)
	}
	challenge, err := sso.PKCEChallenge(sso.NewPKCEVerifier())
	if err != nil {
		t.Fatal(err)
	}
	pending, err := center.protocol.GenerateTicketWithOptions(context.Background(), "alpha", "alice", "http://localhost:9201/sso/callback", nil, nil, sso.OneTimeCredentialOptions{CodeChallenge: challenge, CodeChallengeMethod: sso.PKCEMethodS256})
	if err != nil {
		t.Fatal(err)
	}

	delete(transport, "localhost:9202")
	request := httptest.NewRequest("POST", "/trusted-job", nil)
	result, err := replica.centerHTTP.LogoutAccount(request, "alice")
	if err == nil || result.Stage != "notifications" || !result.CenterRevoked || !result.CredentialsRevoked || result.Complete || result.BindingsCleared {
		t.Fatalf("partial administrative logout = %+v, %v", result, err)
	}
	for _, pair := range []struct {
		app    *application
		cookie *http.Cookie
		active bool
	}{
		{center, centerCookie, false}, {replica, centerCookie, false}, {alpha, alphaCookie, false}, {beta, betaCookie, true},
	} {
		info, err := pair.app.session(cookieRequest(pair.app.cfg.publicURL, pair.cookie))
		if err != nil || (info != nil) != pair.active {
			t.Fatalf("partial logout role=%s client=%s active=%v err=%v", pair.app.cfg.role, pair.app.cfg.clientID, info != nil, err)
		}
	}
	if _, err := center.protocol.ValidateTicket(context.Background(), pending.Ticket); !errors.Is(err, sso.ErrInvalidTicket) {
		t.Fatalf("pending credential survived = %v", err)
	}
	if bindings, err := center.protocol.GetClientSessions(context.Background(), "alice"); err != nil || len(bindings) != 2 {
		t.Fatalf("retry bindings = %v, %v", bindings, err)
	}

	// Recovery uses trusted identity, because the original center cookie is already invalid. 恢复使用可信身份，因为原中心 Cookie 已失效。
	transport["localhost:9202"] = beta.handler
	result, err = center.centerHTTP.LogoutAccount(request, "alice")
	if err != nil || !result.Complete || !result.BindingsCleared || result.Notifications == nil || !result.Notifications.Complete {
		t.Fatalf("administrative retry = %+v, %v", result, err)
	}
	if info, err := beta.session(cookieRequest(beta.cfg.publicURL, betaCookie)); err != nil || info != nil {
		t.Fatal("recovered application retained its session")
	}
	for a, cookie := range otherCookies {
		info, err := a.session(cookieRequest(a.cfg.publicURL, cookie))
		if err != nil || info == nil || info.LoginID != "bob" {
			t.Fatal("administrative logout affected another account")
		}
	}
	if bindings, err := replica.protocol.GetClientSessions(context.Background(), "alice"); err != nil || len(bindings) != 0 {
		t.Fatalf("replica cleanup = %v, %v", bindings, err)
	}
}
