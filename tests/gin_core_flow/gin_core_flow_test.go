package gin_core_flow_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/derror"
	gincoreapp "github.com/Zany2/dtoken-go/examples/gin_core_app"
)

type apiResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type flowClient struct {
	t      *testing.T
	server *httptest.Server
	token  string
}

func newFlowClient(t *testing.T, cfg gincoreapp.Config) *flowClient {
	t.Helper()

	app, err := gincoreapp.NewApp(cfg)
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	server := httptest.NewServer(app.Router())
	t.Cleanup(func() {
		server.Close()
		app.Close()
	})

	return &flowClient{t: t, server: server}
}

// TestAuthFlow verifies login, missing token rejection, token access, and logout.
// TestAuthFlow 验证鉴权流程：未登录拒绝、登录成功、携带 Token 访问成功、登出后 Token 失效。
func TestAuthFlow(t *testing.T) {
	c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})

	// Step 1: request protected API without token, expect 401.
	// 步骤 1：不带 Token 访问受保护接口，预期 401。
	c.expect("GET", "/api/me", nil, "", http.StatusUnauthorized, derror.CodeNotLogin, nil)

	// Step 2: login and use returned token to access current user info.
	// 步骤 2：登录并携带返回的 Token 访问当前用户信息。
	token := c.login("alice")
	var me struct {
		LoginID string `json:"loginId"`
	}
	c.expect("GET", "/api/me", nil, token, http.StatusOK, derror.CodeSuccess, &me)
	if me.LoginID != "alice" {
		t.Fatalf("loginId = %q, want alice", me.LoginID)
	}

	// Step 3: logout and verify the old token can no longer access protected APIs.
	// 步骤 3：登出后再次使用旧 Token，预期鉴权失败。
	c.expect("POST", "/api/logout", nil, token, http.StatusOK, derror.CodeSuccess, nil)
	c.expect("GET", "/api/me", nil, token, http.StatusUnauthorized, derror.CodeNotLogin, nil)
}

// TestPermissionFlow verifies protected routes before and after granting permission.
// TestPermissionFlow 验证权限流程：无权限被拒绝、授予权限后访问成功。
func TestPermissionFlow(t *testing.T) {
	c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})
	token := c.login("bob")

	// Step 1: access article API without article:read permission, expect 403.
	// 步骤 1：没有 article:read 权限时访问文章接口，预期 403。
	c.expect("GET", "/api/articles", nil, token, http.StatusForbidden, derror.CodePermissionDenied, nil)

	// Step 2: grant article:read permission through a protected management API.
	// 步骤 2：通过受保护的管理接口给当前用户授予 article:read 权限。
	c.expect("POST", "/api/permissions", map[string]any{"value": "article:read"}, token, http.StatusOK, derror.CodeSuccess, nil)

	// Step 3: access article API again, expect success and demo article data.
	// 步骤 3：再次访问文章接口，预期成功并返回示例文章数据。
	var articles []string
	c.expect("GET", "/api/articles", nil, token, http.StatusOK, derror.CodeSuccess, &articles)
	if len(articles) != 2 {
		t.Fatalf("articles = %v, want two demo articles", articles)
	}
}

// TestRoleFlow verifies protected role route before and after granting role.
// TestRoleFlow 验证角色流程：无角色被拒绝、授予角色后访问成功。
func TestRoleFlow(t *testing.T) {
	c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})
	token := c.login("carol")

	// Step 1: access admin API without admin role, expect 403.
	// 步骤 1：没有 admin 角色时访问管理接口，预期 403。
	c.expect("GET", "/api/admin", nil, token, http.StatusForbidden, derror.CodePermissionDenied, nil)

	// Step 2: grant admin role and verify the same API becomes accessible.
	// 步骤 2：授予 admin 角色后，再次访问同一接口应成功。
	c.expect("POST", "/api/roles", map[string]any{"value": "admin"}, token, http.StatusOK, derror.CodeSuccess, nil)
	c.expect("GET", "/api/admin", nil, token, http.StatusOK, derror.CodeSuccess, nil)
}

// TestRenewFlow verifies a token can be extended through the HTTP flow.
// TestRenewFlow 验证续期流程：短 TTL 登录、等待 TTL 下降、手动续期后 TTL 变长。
func TestRenewFlow(t *testing.T) {
	c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 2 * time.Second, ActiveTimeout: -1})
	token := c.login("dave")

	// Step 1: read initial TTL from token info API.
	// 步骤 1：通过 Token TTL 接口读取初始有效期。
	before := c.ttl(token)
	if before <= 0 || before > 2 {
		t.Fatalf("initial ttl = %d, want 1..2", before)
	}

	// Step 2: wait for TTL to decrease, proving the token is actually time-bound.
	// 步骤 2：等待 TTL 下降，确认 Token 确实存在过期时间。
	time.Sleep(1100 * time.Millisecond)
	mid := c.ttl(token)
	if mid >= before {
		t.Fatalf("ttl before renew = %d, initial = %d, want decreased", mid, before)
	}

	// Step 3: renew token to five seconds and verify TTL is extended.
	// 步骤 3：把 Token 续期到 5 秒，并验证 TTL 已变长。
	var renewed struct {
		TTL int64 `json:"ttl"`
	}
	c.expect("POST", "/api/token/renew", map[string]any{"seconds": 5}, token, http.StatusOK, derror.CodeSuccess, &renewed)
	if renewed.TTL < 4 || renewed.TTL > 5 {
		t.Fatalf("renewed ttl = %d, want 4..5", renewed.TTL)
	}
}

// TestTokenExpiredFlow verifies expired tokens are rejected by protected APIs.
// TestTokenExpiredFlow 验证 Token 过期流程：短 TTL 登录、等待过期、再次访问受保护接口失败。
func TestTokenExpiredFlow(t *testing.T) {
	c := newFlowClient(t, gincoreapp.Config{TokenTimeout: time.Second, ActiveTimeout: -1})
	token := c.login("expired-user")

	// Step 1: wait longer than the configured token timeout.
	// 步骤 1：等待超过 Token 有效期。
	time.Sleep(2200 * time.Millisecond)

	// Step 2: use expired token to access protected API, expect unauthorized.
	// 步骤 2：使用过期 Token 访问受保护接口，预期未登录。
	c.expect("GET", "/api/me", nil, token, http.StatusUnauthorized, derror.CodeNotLogin, nil)
}

// TestActiveTimeoutFlow verifies inactive tokens are rejected even before absolute TTL expires.
// TestActiveTimeoutFlow 验证活跃超时流程：Token 未到绝对过期时间，但超过不活跃时间后访问失败。
func TestActiveTimeoutFlow(t *testing.T) {
	c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: 1})
	token := c.login("inactive-user")

	// Step 1: wait longer than active timeout while absolute TTL is still valid.
	// 步骤 1：等待超过不活跃超时，但不超过 Token 总 TTL。
	time.Sleep(1200 * time.Millisecond)

	// Step 2: request protected API, expect active-timeout code.
	// 步骤 2：访问受保护接口，预期活跃超时错误码。
	c.expect("GET", "/api/me", nil, token, http.StatusUnauthorized, derror.CodeActiveTimeout, nil)
}

// TestKickoutAndReplaceFlow verifies kicked and replaced tokens are rejected.
// TestKickoutAndReplaceFlow 验证踢下线和顶下线流程：Token 被标记后再次访问失败。
func TestKickoutAndReplaceFlow(t *testing.T) {
	t.Run("kickout", func(t *testing.T) {
		c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})
		token := c.login("kick-user")

		// Step 1: call kickout endpoint for current token.
		// 步骤 1：调用踢下线接口处理当前 Token。
		c.expect("POST", "/api/token/kickout", nil, token, http.StatusOK, derror.CodeSuccess, nil)

		// Step 2: old token should no longer access protected APIs.
		// 步骤 2：旧 Token 再访问受保护接口，预期失败。
		c.expect("GET", "/api/me", nil, token, http.StatusUnauthorized, derror.CodeNotLogin, nil)
	})

	t.Run("replace", func(t *testing.T) {
		c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})
		token := c.login("replace-user")

		// Step 1: call replace endpoint for current token.
		// 步骤 1：调用顶下线接口处理当前 Token。
		c.expect("POST", "/api/token/replace", nil, token, http.StatusOK, derror.CodeSuccess, nil)

		// Step 2: replaced token should no longer access protected APIs.
		// 步骤 2：被顶下线的 Token 再访问受保护接口，预期失败。
		c.expect("GET", "/api/me", nil, token, http.StatusUnauthorized, derror.CodeNotLogin, nil)
	})
}

// TestSessionFlow verifies session data can be queried with a valid token.
// TestSessionFlow 验证会话流程：登录后可查询当前 Session 和在线终端数量。
func TestSessionFlow(t *testing.T) {
	c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})
	token := c.login("session-user")

	// Step 1: query session through protected API.
	// 步骤 1：携带 Token 查询当前会话。
	var session struct {
		LoginID       string `json:"loginId"`
		TerminalCount int    `json:"terminalCount"`
	}
	c.expect("GET", "/api/session", nil, token, http.StatusOK, derror.CodeSuccess, &session)

	// Step 2: assert session belongs to login user and has one terminal.
	// 步骤 2：断言会话归属于当前登录用户，并且存在一个在线终端。
	if session.LoginID != "session-user" {
		t.Fatalf("session loginId = %q, want session-user", session.LoginID)
	}
	if session.TerminalCount != 1 {
		t.Fatalf("terminal count = %d, want 1", session.TerminalCount)
	}
}

// TestMultiTerminalSessionFlow verifies multiple logins for the same account are visible in session.
// TestMultiTerminalSessionFlow 验证多终端会话流程：同账号多设备登录后 Session 终端数正确。
func TestMultiTerminalSessionFlow(t *testing.T) {
	c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})

	// Step 1: login the same account from two different devices.
	// 步骤 1：同一账号从两个不同设备登录。
	webToken := c.loginWithDevice("multi-user", "web", "browser-1")
	mobileToken := c.loginWithDevice("multi-user", "mobile", "phone-1")

	// Step 2: query session with either token and expect two terminals.
	// 步骤 2：使用任一 Token 查询会话，预期在线终端数为 2。
	var session struct {
		LoginID       string `json:"loginId"`
		TerminalCount int    `json:"terminalCount"`
	}
	c.expect("GET", "/api/session", nil, webToken, http.StatusOK, derror.CodeSuccess, &session)
	if session.LoginID != "multi-user" || session.TerminalCount != 2 {
		t.Fatalf("web session = %+v, want loginId multi-user and terminalCount 2", session)
	}
	c.expect("GET", "/api/session", nil, mobileToken, http.StatusOK, derror.CodeSuccess, &session)
	if session.LoginID != "multi-user" || session.TerminalCount != 2 {
		t.Fatalf("mobile session = %+v, want loginId multi-user and terminalCount 2", session)
	}
}

// TestDisableFlow verifies account and service disable behavior through routes.
// TestDisableFlow 验证封禁流程：账号封禁阻止旧 Token 和新登录，服务封禁只阻止指定服务。
func TestDisableFlow(t *testing.T) {
	t.Run("account", func(t *testing.T) {
		c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})
		token := c.login("erin")

		// Step 1: disable current account through protected API.
		// 步骤 1：通过受保护接口封禁当前账号。
		c.expect("POST", "/api/disable/account", map[string]any{"reason": "risk"}, token, http.StatusOK, derror.CodeSuccess, nil)

		// Step 2: old token should be rejected as account disabled.
		// 步骤 2：旧 Token 再访问接口，预期账号封禁错误。
		c.expect("GET", "/api/me", nil, token, http.StatusForbidden, derror.CodeAccountDisabled, nil)

		// Step 3: same account should not be able to login again while disabled.
		// 步骤 3：同账号在封禁期间重新登录，预期被拒绝。
		c.expect("POST", "/login", map[string]any{"username": "erin", "password": "123456"}, "", http.StatusForbidden, derror.CodeAccountDisabled, nil)
	})

	t.Run("service", func(t *testing.T) {
		c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})
		token := c.login("frank")

		// Step 1: payment service is available before service disable.
		// 步骤 1：服务封禁前，支付接口可正常访问。
		c.expect("GET", "/api/payment", nil, token, http.StatusOK, derror.CodeSuccess, nil)

		// Step 2: disable only the payment service for current account.
		// 步骤 2：只封禁当前账号的 payment 服务。
		c.expect("POST", "/api/disable/service/payment", map[string]any{"reason": "risk"}, token, http.StatusOK, derror.CodeSuccess, nil)

		// Step 3: payment API is rejected, while login state itself is still valid.
		// 步骤 3：支付接口被拒绝，但不是整账号下线。
		c.expect("GET", "/api/payment", nil, token, http.StatusForbidden, derror.CodePermissionDenied, nil)
	})
}

// TestDeviceDisableFlow verifies device disable only blocks matching device dimensions.
// TestDeviceDisableFlow 验证设备封禁流程：被封禁设备无法登录，其他设备仍可登录。
func TestDeviceDisableFlow(t *testing.T) {
	c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})
	adminToken := c.loginWithDevice("device-user", "web", "browser-1")

	// Step 1: disable the web device type for current account.
	// 步骤 1：封禁当前账号的 web 设备类型。
	c.expect("POST", "/api/disable/device/web", map[string]any{"reason": "risk"}, adminToken, http.StatusOK, derror.CodeSuccess, nil)

	// Step 2: web login should be rejected.
	// 步骤 2：web 设备再次登录，预期被拒绝。
	c.expect("POST", "/login", map[string]any{
		"username": "device-user",
		"password": "123456",
		"device":   "web",
		"deviceId": "browser-2",
	}, "", http.StatusForbidden, derror.CodeAccountDisabled, nil)

	// Step 3: mobile login should still be accepted.
	// 步骤 3：mobile 设备登录不受 web 设备封禁影响。
	mobileToken := c.loginWithDevice("device-user", "mobile", "phone-1")
	c.expect("GET", "/api/me", nil, mobileToken, http.StatusOK, derror.CodeSuccess, nil)
}

// TestNonceFlow verifies nonce generation and one-time consumption.
// TestNonceFlow 验证 nonce 流程：生成 nonce、首次校验成功、重复使用失败。
func TestNonceFlow(t *testing.T) {
	c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})

	// Step 1: generate a nonce from public API.
	// 步骤 1：通过公开接口生成 nonce。
	var generated struct {
		Nonce string `json:"nonce"`
	}
	c.expect("GET", "/nonce", nil, "", http.StatusOK, derror.CodeSuccess, &generated)
	if generated.Nonce == "" {
		t.Fatal("nonce is empty")
	}

	// Step 2: verify the nonce once, then verify the same nonce again.
	// 步骤 2：第一次校验成功，第二次重复使用应失败。
	body := map[string]any{"nonce": generated.Nonce}
	c.expect("POST", "/nonce/verify", body, "", http.StatusOK, derror.CodeSuccess, nil)
	c.expect("POST", "/nonce/verify", body, "", http.StatusBadRequest, derror.CodeBadRequest, nil)
}

// TestOAuth2AuthorizationCodeFlow verifies code exchange, introspection, refresh, and revoke.
// TestOAuth2AuthorizationCodeFlow 验证 OAuth2 授权码流程：生成 code、换 token、查询 token、刷新、撤销。
func TestOAuth2AuthorizationCodeFlow(t *testing.T) {
	c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})

	// Step 1: generate an authorization code for the demo client.
	// 步骤 1：为示例客户端生成授权码。
	var codeData struct {
		Code string `json:"code"`
	}
	c.expect("POST", "/oauth2/authorize", map[string]any{
		"clientId":    "demo-client",
		"userId":      "oauth-user",
		"redirectUri": "https://client.example/callback",
		"scopes":      []string{"read"},
	}, "", http.StatusOK, derror.CodeSuccess, &codeData)
	if codeData.Code == "" {
		t.Fatal("oauth2 code is empty")
	}

	// Step 2: exchange authorization code for access and refresh tokens.
	// 步骤 2：使用授权码换取访问令牌和刷新令牌。
	token := c.oauth2Token(map[string]any{
		"grantType":    "authorization_code",
		"clientId":     "demo-client",
		"clientSecret": "demo-secret",
		"code":         codeData.Code,
		"redirectUri":  "https://client.example/callback",
	})
	if token.UserID != "oauth-user" || token.AccessToken == "" || token.RefreshToken == "" {
		t.Fatalf("oauth2 token = %+v, want populated token for oauth-user", token)
	}

	// Step 3: authorization code is single-use.
	// 步骤 3：授权码只能使用一次，重复换取应失败。
	c.expect("POST", "/oauth2/token", map[string]any{
		"grantType":    "authorization_code",
		"clientId":     "demo-client",
		"clientSecret": "demo-secret",
		"code":         codeData.Code,
		"redirectUri":  "https://client.example/callback",
	}, "", http.StatusBadRequest, derror.CodeBadRequest, nil)

	// Step 4: introspect access token, expect active token info.
	// 步骤 4：查询访问令牌信息，预期处于有效状态。
	var info struct {
		Active   bool   `json:"active"`
		UserID   string `json:"userId"`
		ClientID string `json:"clientId"`
	}
	c.expect("GET", "/oauth2/introspect", nil, token.AccessToken, http.StatusOK, derror.CodeSuccess, &info)
	if !info.Active || info.UserID != "oauth-user" || info.ClientID != "demo-client" {
		t.Fatalf("oauth2 introspection = %+v, want active oauth-user/demo-client", info)
	}

	// Step 5: refresh token rotates old access and refresh tokens.
	// 步骤 5：使用刷新令牌换新令牌，旧访问令牌应失效。
	refreshed := c.oauth2Token(map[string]any{
		"grantType":    "refresh_token",
		"clientId":     "demo-client",
		"clientSecret": "demo-secret",
		"refreshToken": token.RefreshToken,
	})
	if refreshed.AccessToken == token.AccessToken || refreshed.RefreshToken == token.RefreshToken {
		t.Fatalf("refreshed token = %+v, want rotated token values", refreshed)
	}
	c.expect("GET", "/oauth2/introspect", nil, token.AccessToken, http.StatusUnauthorized, derror.CodeNotLogin, nil)

	// Step 6: revoke refreshed token and verify it is no longer active.
	// 步骤 6：撤销刷新后的访问令牌，并验证它已失效。
	c.expect("POST", "/oauth2/revoke", map[string]any{"token": refreshed.AccessToken}, "", http.StatusOK, derror.CodeSuccess, nil)
	c.expect("GET", "/oauth2/introspect", nil, refreshed.AccessToken, http.StatusUnauthorized, derror.CodeNotLogin, nil)
}

// TestOAuth2PasswordAndClientCredentialsFlow verifies additional OAuth2 grant types.
// TestOAuth2PasswordAndClientCredentialsFlow 验证 OAuth2 密码模式和客户端凭证模式。
func TestOAuth2PasswordAndClientCredentialsFlow(t *testing.T) {
	c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})

	// Step 1: use password grant with demo user credentials.
	// 步骤 1：使用密码模式和示例用户凭证换取令牌。
	passwordToken := c.oauth2Token(map[string]any{
		"grantType":    "password",
		"clientId":     "demo-client",
		"clientSecret": "demo-secret",
		"username":     "alice",
		"password":     "123456",
		"scopes":       []string{"write"},
	})
	if passwordToken.UserID != "user-alice" {
		t.Fatalf("password token userId = %q, want user-alice", passwordToken.UserID)
	}

	// Step 2: use client credentials grant for machine-to-machine access.
	// 步骤 2：使用客户端凭证模式换取机器访问令牌。
	clientToken := c.oauth2Token(map[string]any{
		"grantType":    "client_credentials",
		"clientId":     "demo-client",
		"clientSecret": "demo-secret",
		"scopes":       []string{"read"},
	})
	if clientToken.UserID != "demo-client" {
		t.Fatalf("client credentials userId = %q, want demo-client", clientToken.UserID)
	}

	// Step 3: invalid client secret should be rejected.
	// 步骤 3：错误客户端密钥应被拒绝。
	c.expect("POST", "/oauth2/token", map[string]any{
		"grantType":    "client_credentials",
		"clientId":     "demo-client",
		"clientSecret": "wrong-secret",
	}, "", http.StatusBadRequest, derror.CodeBadRequest, nil)
}

func (c *flowClient) login(username string) string {
	c.t.Helper()
	return c.loginWithDevice(username, "web", username+"-browser")
}

func (c *flowClient) loginWithDevice(username, device, deviceID string) string {
	c.t.Helper()

	var data struct {
		Token string `json:"token"`
	}
	c.expect("POST", "/login", map[string]any{
		"username": username,
		"password": "123456",
		"device":   device,
		"deviceId": deviceID,
	}, "", http.StatusOK, derror.CodeSuccess, &data)
	if data.Token == "" {
		c.t.Fatal("login token is empty")
	}
	c.token = data.Token
	return data.Token
}

func (c *flowClient) ttl(token string) int64 {
	c.t.Helper()

	var data struct {
		TTL int64 `json:"ttl"`
	}
	c.expect("GET", "/api/token/ttl", nil, token, http.StatusOK, derror.CodeSuccess, &data)
	return data.TTL
}

type oauth2TokenData struct {
	AccessToken  string   `json:"accessToken"`
	TokenType    string   `json:"tokenType"`
	ExpiresIn    int64    `json:"expiresIn"`
	RefreshToken string   `json:"refreshToken"`
	Scopes       []string `json:"scopes"`
	UserID       string   `json:"userId"`
	ClientID     string   `json:"clientId"`
}

func (c *flowClient) oauth2Token(body map[string]any) oauth2TokenData {
	c.t.Helper()

	var data oauth2TokenData
	c.expect("POST", "/oauth2/token", body, "", http.StatusOK, derror.CodeSuccess, &data)
	return data
}

func (c *flowClient) expect(method, path string, body any, token string, wantStatus, wantCode int, data any) {
	c.t.Helper()

	resp := c.do(method, path, body, token)
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		c.t.Fatalf("%s %s read body error = %v", method, path, err)
	}
	if resp.StatusCode != wantStatus {
		c.t.Fatalf("%s %s status = %d, want %d, body=%s", method, path, resp.StatusCode, wantStatus, raw)
	}

	var decoded apiResponse
	if err = json.Unmarshal(raw, &decoded); err != nil {
		c.t.Fatalf("%s %s decode response error = %v, body=%s", method, path, err, raw)
	}
	if decoded.Code != wantCode {
		c.t.Fatalf("%s %s code = %d, want %d, body=%s", method, path, decoded.Code, wantCode, raw)
	}
	if data != nil && len(decoded.Data) > 0 && string(decoded.Data) != "null" {
		if err = json.Unmarshal(decoded.Data, data); err != nil {
			c.t.Fatalf("%s %s decode data error = %v, data=%s", method, path, err, decoded.Data)
		}
	}
}

func (c *flowClient) do(method, path string, body any, token string) *http.Response {
	c.t.Helper()

	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			c.t.Fatalf("marshal request body error = %v", err)
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequest(method, c.server.URL+path, reader)
	if err != nil {
		c.t.Fatalf("NewRequest(%s %s) error = %v", method, path, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.server.Client().Do(req)
	if err != nil {
		c.t.Fatalf("%s %s request error = %v", method, path, err)
	}
	if resp == nil {
		c.t.Fatalf("%s %s response is nil", method, path)
	}
	return resp
}
