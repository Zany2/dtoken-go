package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Zany2/dtoken-go/sso"
)

// TestDeploymentCookieRotation covers rolling session/state verification and account revocation with shared storage. TestDeploymentCookieRotation 使用共享存储覆盖滚动会话／状态验签及账号撤销。
func TestDeploymentCookieRotation(t *testing.T) {
	storage := sso.NewMemoryStorage()
	transport := deploymentTransport{}
	makeApp := func(role, primary, companion string) *application {
		t.Helper()
		env := deploymentEnvironment(t, role, "alpha")
		env["SSO_COOKIE_KEY"], env["SSO_COOKIE_VERIFICATION_KEY"] = primary, companion
		cfg, err := loadConfig(role, func(key string) string { return env[key] })
		if err != nil {
			t.Fatal(err)
		}
		app, err := newApplication(cfg, storage, transport)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(app.close)
		return app
	}
	centerOldKey, centerNewKey := strings.Repeat("c", 32), strings.Repeat("d", 32)
	appOldKey, appNewKey := strings.Repeat("a", 32), strings.Repeat("b", 32)
	centerOld := makeApp("center", centerOldKey, "")
	alphaOld := makeApp("client", appOldKey, "")
	transport["localhost:9200"], transport["localhost:9201"] = centerOld.handler, alphaOld.handler
	oldSession, centerSession := loginThroughReplicas(t, alphaOld, alphaOld, centerOld, nil)
	centerSwitched := makeApp("center", centerNewKey, centerOldKey)
	alphaSwitched := makeApp("client", appNewKey, appOldKey)
	alphaPrepared := makeApp("client", appOldKey, appNewKey)
	transport["localhost:9200"], transport["localhost:9201"] = centerSwitched.handler, alphaSwitched.handler
	assertSession := func(app *application, cookie *http.Cookie, active bool) {
		t.Helper()
		info, err := app.session(cookieRequest(app.cfg.publicURL, cookie))
		if err != nil || (info != nil) != active {
			t.Fatalf("session active=%v want=%v err=%v", info != nil, active, err)
		}
	}
	assertSession(centerSwitched, centerSession, true)
	assertSession(alphaSwitched, oldSession, true)

	// Complete an old-key state on the switched replica while reusing the original center session. 在切换副本上完成旧密钥状态，同时复用原中心会话。
	newSession, _ := loginThroughReplicas(t, alphaOld, alphaSwitched, centerSwitched, centerSession)
	assertSession(alphaPrepared, newSession, true)
	assertSession(alphaOld, newSession, false)
	alphaRetired := makeApp("client", appNewKey, "")
	centerRetired := makeApp("center", centerNewKey, "")
	assertSession(alphaRetired, oldSession, false)
	assertSession(alphaRetired, newSession, true)
	assertSession(centerRetired, centerSession, false)

	// A valid old-key signature cannot override a revoked server-side session. 有效旧密钥签名不能覆盖已撤销的服务端会话。
	oldToken, ok := sso.LoginIDFromCookie(alphaOld.cookie)(cookieRequest(alphaOld.cfg.publicURL, oldSession))
	if !ok {
		t.Fatal("old session cookie could not be read")
	}
	if err := alphaSwitched.auth.LogoutByToken(cookieRequest(alphaSwitched.cfg.publicURL, oldSession).Context(), oldToken); err != nil {
		t.Fatal(err)
	}
	assertSession(alphaSwitched, oldSession, false)
	result, err := centerSwitched.centerHTTP.LogoutAccount(httptest.NewRequest(http.MethodPost, "/trusted-logout", nil), "alice")
	if err != nil || !result.Complete {
		t.Fatalf("account logout after rotation: %v", err)
	}
	assertSession(alphaSwitched, newSession, false)
	assertSession(alphaPrepared, newSession, false)
	assertSession(centerSwitched, centerSession, false)
}

// TestDeploymentCookieRotationConfiguration validates both roles without leaking configured key values. TestDeploymentCookieRotationConfiguration 校验两种角色且不泄露密钥配置值。
func TestDeploymentCookieRotationConfiguration(t *testing.T) {
	for _, role := range []string{"center", "client"} {
		for _, key := range []string{"", strings.Repeat("n", 32), "private-short-key"} {
			env := deploymentEnvironment(t, role, "alpha")
			env["SSO_COOKIE_VERIFICATION_KEY"] = key
			cfg, err := loadConfig(role, func(name string) string { return env[name] })
			if key == "private-short-key" {
				if err == nil || strings.Contains(err.Error(), key) {
					t.Fatal("invalid cookie companion accepted or leaked")
				}
			} else if err != nil || cfg.cookieVerificationKey != key {
				t.Fatalf("cookie rotation configuration: %v", err)
			}
		}
	}
}
