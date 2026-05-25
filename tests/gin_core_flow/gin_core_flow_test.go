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

func (c *flowClient) login(username string) string {
	c.t.Helper()

	var data struct {
		Token string `json:"token"`
	}
	c.expect("POST", "/login", map[string]any{
		"username": username,
		"password": "123456",
		"device":   "web",
		"deviceId": username + "-browser",
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
