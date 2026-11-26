package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	dtoken "github.com/Zany2/dtoken-go"
	"github.com/Zany2/dtoken-go/sso"
)

// deploymentTransport routes protocol traffic through real handlers without listening on sockets. deploymentTransport 将协议流量送入实际处理器，无需监听端口。
type deploymentTransport map[string]http.Handler

// RoundTrip dispatches both credential exchange and signed logout delivery. RoundTrip 分发凭证兑换及签名注销通知。
func (d deploymentTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	h := d[r.URL.Host]
	if h == nil {
		return nil, errors.New("test destination unavailable")
	}
	recorder := httptest.NewRecorder()
	h.ServeHTTP(recorder, r)
	return recorder.Result(), nil
}

// deploymentEnvironment supplies two independently signed applications and explicit demo credentials. deploymentEnvironment 提供两个独立签名应用及显式演示凭据。
func deploymentEnvironment(t *testing.T, role, clientID string) map[string]string {
	t.Helper()
	clients := []sso.Client{
		{ClientID: "alpha", ClientSecret: strings.Repeat("a", 32), SecretKey: strings.Repeat("s", 32), RedirectURIs: []string{"http://localhost:9201/sso/callback"}, RequirePKCE: true, Modes: []sso.Mode{sso.ModeTicket}, Scopes: []string{"profile"}},
		{ClientID: "beta", ClientSecret: strings.Repeat("b", 32), SecretKey: strings.Repeat("t", 32), RedirectURIs: []string{"http://localhost:9202/sso/callback"}, RequirePKCE: true, Modes: []sso.Mode{sso.ModeTicket}, Scopes: []string{"profile"}},
	}
	encoded, err := json.Marshal(clients)
	if err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"SSO_DEV": "true", "SSO_NAMESPACE": "test", "SSO_LISTEN": "localhost:9200", "SSO_PUBLIC_URL": "http://localhost:9200", "SSO_CENTER_URL": "http://localhost:9200", "SSO_COOKIE_KEY": strings.Repeat("c", 32), "SSO_CLIENTS_JSON": string(encoded), "SSO_ENABLE_DEMO_LOGIN": "true", "SSO_DEMO_USER": "alice", "SSO_DEMO_PASSWORD": "demo-password-for-tests"}
	if role == "client" {
		client := clients[0]
		port := "9201"
		if clientID == "beta" {
			client, port = clients[1], "9202"
		}
		env["SSO_LISTEN"], env["SSO_PUBLIC_URL"] = "localhost:"+port, "http://localhost:"+port
		env["SSO_CLIENT_ID"], env["SSO_CLIENT_SECRET"], env["SSO_SIGNING_KEY"] = client.ClientID, client.ClientSecret, client.SecretKey
	}
	return env
}

// testConfiguration uses the same configuration loader as the executable. testConfiguration 使用与可执行入口相同的配置加载器。
func testConfiguration(t *testing.T, role, clientID string) deploymentConfig {
	t.Helper()
	env := deploymentEnvironment(t, role, clientID)
	cfg, err := loadConfig(role, func(key string) string { return env[key] })
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

// TestDeploymentConfiguration rejects unsafe deployment inputs without disclosing configured secrets. TestDeploymentConfiguration 拒绝不安全部署输入且不泄露配置密钥。
func TestDeploymentConfiguration(t *testing.T) {
	for _, tc := range []struct{ role, key, value string }{
		{"client", "SSO_DEV", "yes"}, {"client", "SSO_NAMESPACE", "prod:other"}, {"client", "SSO_LISTEN", "0.0.0.0:9201"},
		{"client", "SSO_PUBLIC_URL", "http://other.example"}, {"client", "SSO_CENTER_URL", "https://name:private-secret@example.com"},
		{"client", "SSO_PUBLIC_URL", "http://localhost:9201/prefix"}, {"client", "SSO_COOKIE_KEY", "short-private-secret"},
		{"client", "SSO_SIGNING_KEY", ""}, {"client", "SSO_CLIENT_ID", "other:app"},
		{"center", "SSO_ENABLE_DEMO_LOGIN", "false"}, {"center", "SSO_DEMO_PASSWORD", "short"},
		{"center", "SSO_CLIENTS_JSON", `[{"clientId":"private-secret","unknown":true}]`}, {"center", "SSO_CLIENTS_JSON", `[]`},
	} {
		t.Run(tc.role+tc.key+tc.value, func(t *testing.T) {
			env := deploymentEnvironment(t, tc.role, "alpha")
			env[tc.key] = tc.value
			_, err := loadConfig(tc.role, func(key string) string { return env[key] })
			if err == nil || strings.Contains(err.Error(), "private-secret") {
				t.Fatalf("configuration diagnostic = %v", err)
			}
		})
	}
	env := deploymentEnvironment(t, "client", "alpha")
	env["SSO_DEV"], env["SSO_PUBLIC_URL"], env["SSO_CENTER_URL"] = "false", "https://alpha.example.com", "https://login.example.com"
	if _, err := loadConfig("client", func(key string) string { return env[key] }); err == nil {
		t.Fatal("nondevelopment memory accepted")
	}
	env["SSO_REDIS_URL"] = "redis://localhost:6379/0"
	if _, err := loadConfig("client", func(key string) string { return env[key] }); err != nil {
		t.Fatal(err)
	}
	center, alpha, beta := testConfiguration(t, "center", ""), testConfiguration(t, "client", "alpha"), testConfiguration(t, "client", "beta")
	if center.storagePrefix() == alpha.storagePrefix() || alpha.storagePrefix() == beta.storagePrefix() {
		t.Fatal("storage namespaces collide")
	}
	registryEnv := deploymentEnvironment(t, "center", "")
	var registrations []sso.Client
	if err := json.Unmarshal([]byte(registryEnv["SSO_CLIENTS_JSON"]), &registrations); err != nil {
		t.Fatal(err)
	}
	registrations = append(registrations, registrations[0])
	duplicate, err := json.Marshal(registrations)
	if err != nil {
		t.Fatal(err)
	}
	registryEnv["SSO_CLIENTS_JSON"] = string(duplicate)
	if _, err := loadConfig("center", func(key string) string { return registryEnv[key] }); err == nil {
		t.Fatal("duplicate registration accepted")
	}
}

// requestHandler performs browser requests with explicitly selected cookies and origins. requestHandler 使用显式选择的 Cookie 与来源执行浏览器请求。
func requestHandler(h http.Handler, method, target string, form url.Values, origin string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, target, strings.NewReader(form.Encode()))
	if method == http.MethodPost {
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	for _, cookie := range cookies {
		r.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

// responseCookie finds one session or state cookie without logging its contents. responseCookie 查找会话或状态 Cookie，不记录其内容。
func responseCookie(t *testing.T, rec *httptest.ResponseRecorder, name string) *http.Cookie {
	t.Helper()
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	t.Fatalf("missing cookie %s at status %d", name, rec.Code)
	return nil
}

// loginThroughReplicas completes PKCE login using another client instance for the callback. loginThroughReplicas 使用另一客户端实例完成 PKCE 登录回调。
func loginThroughReplicas(t *testing.T, first, second, center *application, centerCookie *http.Cookie) (*http.Cookie, *http.Cookie) {
	t.Helper()
	start := requestHandler(first.handler, http.MethodGet, first.cfg.publicURL+"/protected", nil, "")
	if start.Code != http.StatusFound {
		t.Fatalf("begin login = %d", start.Code)
	}
	authorizeURL := start.Header().Get("Location")
	stateCookies := start.Result().Cookies()
	var authorized *httptest.ResponseRecorder
	if centerCookie == nil {
		anonymous := requestHandler(center.handler, http.MethodGet, authorizeURL, nil, "")
		loginURL, err := url.Parse(anonymous.Header().Get("Location"))
		if err != nil || anonymous.Code != http.StatusFound || loginURL.Path != "/login" {
			t.Fatal("missing center login redirect")
		}
		form := url.Values{"username": {center.cfg.demoUser}, "password": {center.cfg.demoPassword}, "back": {loginURL.Query().Get("back")}}
		login := requestHandler(center.handler, http.MethodPost, center.cfg.publicURL+"/login", form, center.cfg.publicURL)
		if login.Code != http.StatusSeeOther {
			t.Fatalf("center login = %d", login.Code)
		}
		centerCookie = responseCookie(t, login, center.cookie.Name)
	}
	authorized = requestHandler(center.handler, http.MethodGet, authorizeURL, nil, "", centerCookie)
	if authorized.Code != http.StatusFound {
		t.Fatalf("authorize = %d %s", authorized.Code, authorized.Body.String())
	}
	callbackURL := authorized.Header().Get("Location")
	callback := requestHandler(second.handler, http.MethodGet, callbackURL, nil, "", stateCookies...)
	if callback.Code != http.StatusSeeOther {
		t.Fatalf("callback = %d %s", callback.Code, callback.Body.String())
	}
	cookie := responseCookie(t, callback, second.cookie.Name)
	if cookie.Value == centerCookie.Value || !cookie.HttpOnly {
		t.Fatal("client session was not isolated")
	}
	replay := requestHandler(second.handler, http.MethodGet, callbackURL, nil, "", stateCookies...)
	if replay.Code == http.StatusSeeOther {
		t.Fatal("consumed browser flow replayed")
	}
	return cookie, centerCookie
}

// TestMultiApplicationDeployment covers shared-store replicas, signed callbacks, replay, and global invalidation. TestMultiApplicationDeployment 覆盖共享存储副本、签名回调、防重放及全局失效。
func TestMultiApplicationDeployment(t *testing.T) {
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
	center, centerReplica := create("center", ""), create("center", "")
	alpha, alphaReplica, beta := create("client", "alpha"), create("client", "alpha"), create("client", "beta")
	transport["localhost:9200"], transport["localhost:9201"], transport["localhost:9202"] = centerReplica.handler, alphaReplica.handler, beta.handler
	alphaCookie, centerCookie := loginThroughReplicas(t, alpha, alphaReplica, center, nil)
	betaCookie, _ := loginThroughReplicas(t, beta, beta, centerReplica, centerCookie)
	for _, pair := range []struct {
		app    *application
		cookie *http.Cookie
	}{{alpha, alphaCookie}, {alphaReplica, alphaCookie}, {beta, betaCookie}} {
		rec := requestHandler(pair.app.handler, http.MethodGet, pair.app.cfg.publicURL+"/protected", nil, "", pair.cookie)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "alice") {
			t.Fatalf("shared session status = %d", rec.Code)
		}
	}
	localLogout := requestHandler(alpha.handler, http.MethodPost, alpha.cfg.publicURL+"/logout", nil, alpha.cfg.publicURL, alphaCookie)
	if localLogout.Code != http.StatusSeeOther {
		t.Fatalf("local logout = %d", localLogout.Code)
	}
	if info, err := alphaReplica.session(cookieRequest(alpha.cfg.publicURL, alphaCookie)); err != nil || info != nil {
		t.Fatal("local logout did not reach replica")
	}
	if info, err := beta.session(cookieRequest(beta.cfg.publicURL, betaCookie)); err != nil || info == nil {
		t.Fatal("local logout affected another application")
	}
	alphaCookie, _ = loginThroughReplicas(t, alpha, alphaReplica, centerReplica, centerCookie)

	forged := requestHandler(beta.handler, http.MethodPost, beta.cfg.publicURL+"/sso/logout-callback", url.Values{"loginId": {"alice"}, "client": {"beta"}}, "")
	if forged.Code == http.StatusOK {
		t.Fatal("unsigned callback accepted")
	}
	if info, err := beta.session(cookieRequest(beta.cfg.publicURL, betaCookie)); err != nil || info == nil {
		t.Fatal("forged callback changed session")
	}
	for _, origin := range []string{"", "https://evil.example"} {
		rec := requestHandler(alpha.handler, http.MethodPost, alpha.cfg.publicURL+"/logout-all", nil, origin, alphaCookie)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("logout CSRF = %d", rec.Code)
		}
	}
	logout := requestHandler(alpha.handler, http.MethodPost, alpha.cfg.publicURL+"/logout-all", nil, alpha.cfg.publicURL, alphaCookie)
	if logout.Code != http.StatusSeeOther {
		t.Fatalf("global redirect = %d", logout.Code)
	}

	// Strict delivery preserves the center and unavailable target, while successful targets may already be logged out. 严格通知保留中心及不可达目标，成功通知的目标可能已经退出。
	delete(transport, "localhost:9202")
	failed := requestHandler(centerReplica.handler, http.MethodGet, logout.Header().Get("Location"), nil, "", centerCookie)
	if failed.Code < 400 || !strings.Contains(failed.Body.String(), "notifications") {
		t.Fatalf("strict failure = %d", failed.Code)
	}
	if info, err := center.session(cookieRequest(center.cfg.publicURL, centerCookie)); err != nil || info == nil {
		t.Fatal("strict failure retired center session")
	}
	if info, err := beta.session(cookieRequest(beta.cfg.publicURL, betaCookie)); err != nil || info == nil {
		t.Fatal("unavailable application lost its session")
	}
	transport["localhost:9202"] = beta.handler
	logout = requestHandler(alpha.handler, http.MethodPost, alpha.cfg.publicURL+"/logout-all", nil, alpha.cfg.publicURL)
	if logout.Code != http.StatusSeeOther {
		t.Fatal("fresh retry URL unavailable")
	}
	centerLogout := requestHandler(centerReplica.handler, http.MethodGet, logout.Header().Get("Location"), nil, "", centerCookie)
	if centerLogout.Code != http.StatusOK {
		t.Fatalf("global logout = %d %s", centerLogout.Code, centerLogout.Body.String())
	}
	for _, pair := range []struct {
		app    *application
		cookie *http.Cookie
	}{{alphaReplica, alphaCookie}, {beta, betaCookie}, {center, centerCookie}} {
		if info, err := pair.app.session(cookieRequest(pair.app.cfg.publicURL, pair.cookie)); err != nil || info != nil {
			t.Fatalf("session survived global logout: %v", err)
		}
	}
	replayed := requestHandler(center.handler, http.MethodGet, logout.Header().Get("Location"), nil, "", centerCookie)
	if replayed.Code == http.StatusOK {
		t.Fatal("global logout replay accepted")
	}
}

// cookieRequest creates a request for server-side session inspection. cookieRequest 构建用于服务端会话检查的请求。
func cookieRequest(origin string, cookie *http.Cookie) *http.Request {
	r := httptest.NewRequest(http.MethodGet, origin+"/protected", nil)
	r.AddCookie(cookie)
	return r
}

// readinessStorage injects backend availability without affecting liveness. readinessStorage 注入后端可用性故障，不影响存活检查。
type readinessStorage struct {
	*sso.MemoryStorage
	unavailable bool
	closed      bool
}

// Ping reports the injected backend state. Ping 返回注入的后端状态。
func (s *readinessStorage) Ping(ctx context.Context) error {
	if s.unavailable {
		return errors.New("private-backend-detail")
	}
	return s.MemoryStorage.Ping(ctx)
}

// Close records ownership violations by composed services. Close 记录组装服务违反所有权的关闭调用。
func (s *readinessStorage) Close() error { s.closed = true; return nil }

// TestDeploymentBoundaries covers login CSRF, cookie flags, deadline enforcement, and storage readiness. TestDeploymentBoundaries 覆盖登录跨站保护、Cookie 属性、截止时间及存储就绪状态。
func TestDeploymentBoundaries(t *testing.T) {
	storage := &readinessStorage{MemoryStorage: sso.NewMemoryStorage()}
	cfg := testConfiguration(t, "center", "")
	a, err := newApplication(cfg, storage, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer a.close()
	form := url.Values{"username": {cfg.demoUser}, "password": {cfg.demoPassword}}
	for _, origin := range []string{"", "https://evil.example"} {
		if rec := requestHandler(a.handler, http.MethodPost, cfg.publicURL+"/login", form, origin); rec.Code != http.StatusForbidden {
			t.Fatalf("login CSRF = %d", rec.Code)
		}
	}
	form.Set("password", "wrong")
	if rec := requestHandler(a.handler, http.MethodPost, cfg.publicURL+"/login", form, cfg.publicURL); rec.Code != http.StatusUnauthorized {
		t.Fatalf("login credentials = %d", rec.Code)
	}
	storage.unavailable = true
	ready := requestHandler(a.handler, http.MethodGet, cfg.publicURL+"/health/ready", nil, "")
	if ready.Code != http.StatusServiceUnavailable || strings.Contains(ready.Body.String(), "private") {
		t.Fatal("readiness failure leaked or disappeared")
	}
	if rec := requestHandler(a.handler, http.MethodGet, cfg.publicURL+"/health/live", nil, ""); rec.Code != http.StatusOK {
		t.Fatal("liveness depends on backend")
	}
	storage.unavailable = false
	clientCfg := testConfiguration(t, "client", "alpha")
	clientCfg.publicURL, clientCfg.centerURL = "https://alpha.example.com", "https://login.example.com"
	client, err := newApplication(clientCfg, storage, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer client.close()
	rec := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, clientCfg.publicURL, nil)
	if err := client.startSession(rec, r, "alice", time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	cookie := responseCookie(t, rec, client.cookie.Name)
	if !cookie.Secure || !cookie.HttpOnly || cookie.Domain != "" {
		t.Fatal("HTTPS cookie attributes invalid")
	}
	if err := client.startSession(httptest.NewRecorder(), r, "alice", time.Now().Add(-time.Second)); err == nil {
		t.Fatal("expired deadline accepted")
	}
	// A still-active Auth token cannot extend the original SSO deadline. 仍有效的 Auth Token 不能延长原 SSO 截止时间。
	token, err := client.auth.Login(context.Background(), dtoken.LoginOptions{LoginID: "alice", DeviceID: "expired-proof", Timeout: time.Hour, Extra: map[string]any{"ssoDeadline": time.Now().Add(-time.Second).Format(time.RFC3339Nano)}})
	if err != nil {
		t.Fatal(err)
	}
	expired := httptest.NewRecorder()
	sso.SetLoginIDCookie(expired, client.cookie, token)
	if info, err := client.session(cookieRequest(clientCfg.publicURL, responseCookie(t, expired, client.cookie.Name))); err != nil || info != nil {
		t.Fatal("active Auth token bypassed original deadline")
	}
	client.close()
	if storage.closed {
		t.Fatal("borrowed storage closed by application")
	}
	for _, value := range []string{"https://evil.example", "//evil.example", "/%2fevil.example", "/%5cevil.example", "/%0d%0aevil"} {
		if localBack(value) != "/" {
			t.Fatal("unsafe login return accepted")
		}
	}
}
