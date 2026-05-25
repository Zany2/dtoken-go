package gin_core_flow_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/config"
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

// TestAuthFlow verifies login, missing token rejection, token access, and logout. TestAuthFlow 验证鉴权流程：未登录拒绝、登录成功、携带 Token 访问成功、登出后 Token 失效。
func TestAuthFlow(t *testing.T) {
	c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})

	// Step 1: request protected API without token, expect 401. 步骤 1：不带 Token 访问受保护接口，预期 401。
	c.expect("GET", "/api/me", nil, "", http.StatusUnauthorized, derror.CodeNotLogin, nil)

	// Step 2: login and use returned token to access current user info. 步骤 2：登录并携带返回的 Token 访问当前用户信息。
	token := c.login("alice")
	var me struct {
		LoginID string `json:"loginId"`
	}
	c.expect("GET", "/api/me", nil, token, http.StatusOK, derror.CodeSuccess, &me)
	if me.LoginID != "alice" {
		t.Fatalf("loginId = %q, want alice", me.LoginID)
	}

	// Step 3: logout and verify the old token can no longer access protected APIs. 步骤 3：登出后再次使用旧 Token，预期鉴权失败。
	c.expect("POST", "/api/logout", nil, token, http.StatusOK, derror.CodeSuccess, nil)
	c.expect("GET", "/api/me", nil, token, http.StatusUnauthorized, derror.CodeNotLogin, nil)
}

// TestTokenMetadataAndStatusFlow verifies token metadata and boolean login status APIs. TestTokenMetadataAndStatusFlow 验证 Token 元信息和布尔登录状态接口。
func TestTokenMetadataAndStatusFlow(t *testing.T) {
	c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})
	token := c.loginWithDevice("token-meta-user", "desktop", "pc-1")

	// Step 1: query login status without and with a valid token. 步骤 1：分别查询无 Token 与有效 Token 的登录状态。
	var status struct {
		IsLogin bool `json:"isLogin"`
	}
	c.expect("GET", "/token/status", nil, "", http.StatusOK, derror.CodeSuccess, &status)
	if status.IsLogin {
		t.Fatal("missing token isLogin = true, want false")
	}
	c.expect("GET", "/token/status", nil, token, http.StatusOK, derror.CodeSuccess, &status)
	if !status.IsLogin {
		t.Fatal("valid token isLogin = false, want true")
	}

	// Step 2: query token metadata from GetTokenInfo/GetDevice/GetDeviceId/GetTokenCreateTime. 步骤 2：通过核心元信息接口读取 Token 绑定信息。
	var info struct {
		LoginID    string `json:"loginId"`
		Device     string `json:"device"`
		DeviceID   string `json:"deviceId"`
		CreateTime int64  `json:"createTime"`
		Timeout    int64  `json:"timeout"`
	}
	c.expect("GET", "/api/token/info", nil, token, http.StatusOK, derror.CodeSuccess, &info)
	if info.LoginID != "token-meta-user" || info.Device != "desktop" || info.DeviceID != "pc-1" {
		t.Fatalf("token info = %+v, want token-meta-user/desktop/pc-1", info)
	}
	if info.CreateTime <= 0 || info.Timeout != 30 {
		t.Fatalf("token timing info = %+v, want createTime > 0 and timeout 30", info)
	}

	// Step 3: custom login timeout overrides default token TTL. 步骤 3：自定义登录过期时间会覆盖默认 TTL。
	var data struct {
		Token string `json:"token"`
	}
	c.expect("POST", "/login/timeout", map[string]any{
		"username": "token-timeout-user",
		"password": "123456",
		"seconds":  5,
		"device":   "web",
		"deviceId": "short-ttl",
	}, "", http.StatusOK, derror.CodeSuccess, &data)
	if data.Token == "" {
		t.Fatal("custom timeout token is empty")
	}
	ttl := c.ttl(data.Token)
	if ttl <= 0 || ttl > 5 {
		t.Fatalf("custom timeout ttl = %d, want 1..5", ttl)
	}

	// Step 4: LoginByToken accepts an existing valid token. 步骤 4：LoginByToken 可以续用一个已有有效 Token。
	c.expect("POST", "/api/token/login-by-token", nil, token, http.StatusOK, derror.CodeSuccess, nil)
}

// TestPermissionFlow verifies protected routes before and after granting permission. TestPermissionFlow 验证权限流程：无权限被拒绝、授予权限后访问成功。
func TestPermissionFlow(t *testing.T) {
	c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})
	token := c.login("bob")

	// Step 1: access article API without article:read permission, expect 403. 步骤 1：没有 article:read 权限时访问文章接口，预期 403。
	c.expect("GET", "/api/articles", nil, token, http.StatusForbidden, derror.CodePermissionDenied, nil)

	// Step 2: grant article:read permission through a protected management API. 步骤 2：通过受保护的管理接口给当前用户授予 article:read 权限。
	c.expect("POST", "/api/permissions", map[string]any{"value": "article:read"}, token, http.StatusOK, derror.CodeSuccess, nil)

	// Step 3: access article API again, expect success and demo article data. 步骤 3：再次访问文章接口，预期成功并返回示例文章数据。
	var articles []string
	c.expect("GET", "/api/articles", nil, token, http.StatusOK, derror.CodeSuccess, &articles)
	if len(articles) != 2 {
		t.Fatalf("articles = %v, want two demo articles", articles)
	}
}

// TestPermissionMutationAndLogicFlow verifies removal, AND, OR, and wildcard permission checks. TestPermissionMutationAndLogicFlow 验证权限移除、AND/OR 组合校验和通配符权限匹配。
func TestPermissionMutationAndLogicFlow(t *testing.T) {
	c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})
	token := c.login("perm-logic-user")

	// Step 1: grant article:read only; single permission passes but AND check fails. 步骤 1：只授予 article:read；单权限通过，但 AND 组合校验失败。
	c.expect("POST", "/api/permissions", map[string]any{"value": "article:read"}, token, http.StatusOK, derror.CodeSuccess, nil)
	c.expect("GET", "/api/articles", nil, token, http.StatusOK, derror.CodeSuccess, nil)
	c.expect("GET", "/api/article/manage", nil, token, http.StatusForbidden, derror.CodePermissionDenied, nil)

	// Step 2: grant article:write; AND check now succeeds. 步骤 2：再授予 article:write；AND 组合校验成功。
	c.expect("POST", "/api/permissions", map[string]any{"value": "article:write"}, token, http.StatusOK, derror.CodeSuccess, nil)
	c.expect("GET", "/api/article/manage", nil, token, http.StatusOK, derror.CodeSuccess, nil)

	// Step 3: remove article:read; both single and AND checks fail again. 步骤 3：移除 article:read；单权限与 AND 组合校验重新失败。
	c.expect("DELETE", "/api/permissions", map[string]any{"value": "article:read"}, token, http.StatusOK, derror.CodeSuccess, nil)
	c.expect("GET", "/api/articles", nil, token, http.StatusForbidden, derror.CodePermissionDenied, nil)
	c.expect("GET", "/api/article/manage", nil, token, http.StatusForbidden, derror.CodePermissionDenied, nil)

	// Step 4: wildcard permission satisfies OR route requirement. 步骤 4：通配符权限满足 OR 组合路由要求。
	c.expect("POST", "/api/permissions", map[string]any{"value": "content:*"}, token, http.StatusOK, derror.CodeSuccess, nil)
	c.expect("GET", "/api/content", nil, token, http.StatusOK, derror.CodeSuccess, nil)
}

// TestAccessStatusFlow verifies HasPermission and HasRole variants by login ID and token. TestAccessStatusFlow 验证按 loginID 与 Token 判断权限和角色的布尔接口。
func TestAccessStatusFlow(t *testing.T) {
	c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})
	token := c.login("access-status-user")

	c.expect("POST", "/api/permissions", map[string]any{"value": "report:read"}, token, http.StatusOK, derror.CodeSuccess, nil)
	c.expect("POST", "/api/roles", map[string]any{"value": "auditor"}, token, http.StatusOK, derror.CodeSuccess, nil)

	var status struct {
		HasPermissionByLoginID bool `json:"hasPermissionByLoginId"`
		HasPermissionByToken   bool `json:"hasPermissionByToken"`
		HasRoleByLoginID       bool `json:"hasRoleByLoginId"`
		HasRoleByToken         bool `json:"hasRoleByToken"`
	}
	c.expect("GET", "/api/access/status?permission=report:read&role=auditor", nil, token, http.StatusOK, derror.CodeSuccess, &status)
	if !status.HasPermissionByLoginID || !status.HasPermissionByToken || !status.HasRoleByLoginID || !status.HasRoleByToken {
		t.Fatalf("access status = %+v, want all true", status)
	}

	c.expect("GET", "/api/access/status?permission=report:write&role=admin", nil, token, http.StatusOK, derror.CodeSuccess, &status)
	if status.HasPermissionByLoginID || status.HasPermissionByToken || status.HasRoleByLoginID || status.HasRoleByToken {
		t.Fatalf("missing access status = %+v, want all false", status)
	}
}

// TestRoleFlow verifies protected role route before and after granting role. TestRoleFlow 验证角色流程：无角色被拒绝、授予角色后访问成功。
func TestRoleFlow(t *testing.T) {
	c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})
	token := c.login("carol")

	// Step 1: access admin API without admin role, expect 403. 步骤 1：没有 admin 角色时访问管理接口，预期 403。
	c.expect("GET", "/api/admin", nil, token, http.StatusForbidden, derror.CodePermissionDenied, nil)

	// Step 2: grant admin role and verify the same API becomes accessible. 步骤 2：授予 admin 角色后，再次访问同一接口应成功。
	c.expect("POST", "/api/roles", map[string]any{"value": "admin"}, token, http.StatusOK, derror.CodeSuccess, nil)
	c.expect("GET", "/api/admin", nil, token, http.StatusOK, derror.CodeSuccess, nil)
}

// TestRoleMutationAndLogicFlow verifies removal, AND, and OR role checks. TestRoleMutationAndLogicFlow 验证角色移除以及 AND/OR 组合校验。
func TestRoleMutationAndLogicFlow(t *testing.T) {
	c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})
	token := c.login("role-logic-user")

	// Step 1: admin alone passes single-role route but not admin+ops route. 步骤 1：admin 单角色能访问单角色接口，但不能通过 admin+ops 组合校验。
	c.expect("POST", "/api/roles", map[string]any{"value": "admin"}, token, http.StatusOK, derror.CodeSuccess, nil)
	c.expect("GET", "/api/admin", nil, token, http.StatusOK, derror.CodeSuccess, nil)
	c.expect("GET", "/api/ops", nil, token, http.StatusForbidden, derror.CodePermissionDenied, nil)

	// Step 2: grant ops; AND route succeeds. 步骤 2：授予 ops 后，AND 组合角色接口成功。
	c.expect("POST", "/api/roles", map[string]any{"value": "ops"}, token, http.StatusOK, derror.CodeSuccess, nil)
	c.expect("GET", "/api/ops", nil, token, http.StatusOK, derror.CodeSuccess, nil)

	// Step 3: remove admin; single admin and AND route fail. 步骤 3：移除 admin 后，单角色和 AND 组合接口失败。
	c.expect("DELETE", "/api/roles", map[string]any{"value": "admin"}, token, http.StatusOK, derror.CodeSuccess, nil)
	c.expect("GET", "/api/admin", nil, token, http.StatusForbidden, derror.CodePermissionDenied, nil)
	c.expect("GET", "/api/ops", nil, token, http.StatusForbidden, derror.CodePermissionDenied, nil)

	// Step 4: OR role route succeeds when either role exists. 步骤 4：存在任一候选角色时，OR 角色接口成功。
	c.expect("POST", "/api/roles", map[string]any{"value": "security"}, token, http.StatusOK, derror.CodeSuccess, nil)
	c.expect("GET", "/api/audit", nil, token, http.StatusOK, derror.CodeSuccess, nil)
}

// TestRenewFlow verifies a token can be extended through the HTTP flow. TestRenewFlow 验证续期流程：短 TTL 登录、等待 TTL 下降、手动续期后 TTL 变长。
func TestRenewFlow(t *testing.T) {
	c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 2 * time.Second, ActiveTimeout: -1})
	token := c.login("dave")

	// Step 1: read initial TTL from token info API. 步骤 1：通过 Token TTL 接口读取初始有效期。
	before := c.ttl(token)
	if before <= 0 || before > 2 {
		t.Fatalf("initial ttl = %d, want 1..2", before)
	}

	// Step 2: wait for TTL to decrease, proving the token is actually time-bound. 步骤 2：等待 TTL 下降，确认 Token 确实存在过期时间。
	time.Sleep(1100 * time.Millisecond)
	mid := c.ttl(token)
	if mid >= before {
		t.Fatalf("ttl before renew = %d, initial = %d, want decreased", mid, before)
	}

	// Step 3: renew token to five seconds and verify TTL is extended. 步骤 3：把 Token 续期到 5 秒，并验证 TTL 已变长。
	var renewed struct {
		TTL int64 `json:"ttl"`
	}
	c.expect("POST", "/api/token/renew", map[string]any{"seconds": 5}, token, http.StatusOK, derror.CodeSuccess, &renewed)
	if renewed.TTL < 4 || renewed.TTL > 5 {
		t.Fatalf("renewed ttl = %d, want 4..5", renewed.TTL)
	}
}

// TestTokenExpiredFlow verifies expired tokens are rejected by protected APIs. TestTokenExpiredFlow 验证 Token 过期流程：短 TTL 登录、等待过期、再次访问受保护接口失败。
func TestTokenExpiredFlow(t *testing.T) {
	c := newFlowClient(t, gincoreapp.Config{TokenTimeout: time.Second, ActiveTimeout: -1})
	token := c.login("expired-user")

	// Step 1: wait longer than the configured token timeout. 步骤 1：等待超过 Token 有效期。
	time.Sleep(2200 * time.Millisecond)

	// Step 2: use expired token to access protected API, expect unauthorized. 步骤 2：使用过期 Token 访问受保护接口，预期未登录。
	c.expect("GET", "/api/me", nil, token, http.StatusUnauthorized, derror.CodeNotLogin, nil)
}

// TestActiveTimeoutFlow verifies inactive tokens are rejected even before absolute TTL expires. TestActiveTimeoutFlow 验证活跃超时流程：Token 未到绝对过期时间，但超过不活跃时间后访问失败。
func TestActiveTimeoutFlow(t *testing.T) {
	c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: 1})
	token := c.login("inactive-user")

	// Step 1: wait longer than active timeout while absolute TTL is still valid. 步骤 1：等待超过不活跃超时，但不超过 Token 总 TTL。
	time.Sleep(2200 * time.Millisecond)

	// Step 2: request protected API, expect active-timeout code. 步骤 2：访问受保护接口，预期活跃超时错误码。
	c.expect("GET", "/api/me", nil, token, http.StatusUnauthorized, derror.CodeActiveTimeout, nil)
}

// TestKickoutAndReplaceFlow verifies kicked and replaced tokens are rejected. TestKickoutAndReplaceFlow 验证踢下线和顶下线流程：Token 被标记后再次访问失败。
func TestKickoutAndReplaceFlow(t *testing.T) {
	t.Run("kickout", func(t *testing.T) {
		c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})
		token := c.login("kick-user")

		// Step 1: call kickout endpoint for current token. 步骤 1：调用踢下线接口处理当前 Token。
		c.expect("POST", "/api/token/kickout", nil, token, http.StatusOK, derror.CodeSuccess, nil)

		// Step 2: old token should no longer access protected APIs. 步骤 2：旧 Token 再访问受保护接口，预期失败。
		c.expect("GET", "/api/me", nil, token, http.StatusUnauthorized, derror.CodeNotLogin, nil)
	})

	t.Run("replace", func(t *testing.T) {
		c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})
		token := c.login("replace-user")

		// Step 1: call replace endpoint for current token. 步骤 1：调用顶下线接口处理当前 Token。
		c.expect("POST", "/api/token/replace", nil, token, http.StatusOK, derror.CodeSuccess, nil)

		// Step 2: replaced token should no longer access protected APIs. 步骤 2：被顶下线的 Token 再访问受保护接口，预期失败。
		c.expect("GET", "/api/me", nil, token, http.StatusUnauthorized, derror.CodeNotLogin, nil)
	})
}

// TestSessionFlow verifies session data can be queried with a valid token. TestSessionFlow 验证会话流程：登录后可查询当前 Session 和在线终端数量。
func TestSessionFlow(t *testing.T) {
	c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})
	token := c.login("session-user")

	// Step 1: query session through protected API. 步骤 1：携带 Token 查询当前会话。
	var session struct {
		LoginID       string `json:"loginId"`
		TerminalCount int    `json:"terminalCount"`
	}
	c.expect("GET", "/api/session", nil, token, http.StatusOK, derror.CodeSuccess, &session)

	// Step 2: assert session belongs to login user and has one terminal. 步骤 2：断言会话归属于当前登录用户，并且存在一个在线终端。
	if session.LoginID != "session-user" {
		t.Fatalf("session loginId = %q, want session-user", session.LoginID)
	}
	if session.TerminalCount != 1 {
		t.Fatalf("terminal count = %d, want 1", session.TerminalCount)
	}
}

// TestMultiTerminalSessionFlow verifies multiple logins for the same account are visible in session. TestMultiTerminalSessionFlow 验证多终端会话流程：同账号多设备登录后 Session 终端数正确。
func TestMultiTerminalSessionFlow(t *testing.T) {
	c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})

	// Step 1: login the same account from two different devices. 步骤 1：同一账号从两个不同设备登录。
	webToken := c.loginWithDevice("multi-user", "web", "browser-1")
	mobileToken := c.loginWithDevice("multi-user", "mobile", "phone-1")

	// Step 2: query session with either token and expect two terminals. 步骤 2：使用任一 Token 查询会话，预期在线终端数为 2。
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

// TestTerminalInspectionFlow verifies terminal metadata and online counts. TestTerminalInspectionFlow 验证终端详情、账号在线数、设备在线数和具体设备在线数。
func TestTerminalInspectionFlow(t *testing.T) {
	c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})

	webToken := c.loginWithDevice("terminal-user", "web", "browser-1")
	mobileToken := c.loginWithDevice("terminal-user", "mobile", "phone-1")

	var webInfo struct {
		LoginID         string `json:"loginId"`
		Device          string `json:"device"`
		DeviceID        string `json:"deviceId"`
		OnlineCount     int    `json:"onlineCount"`
		DeviceCount     int    `json:"deviceCount"`
		DeviceIDCount   int    `json:"deviceIdCount"`
		LatestForDevice string `json:"latestForDevice"`
	}
	c.expect("GET", "/api/terminal", nil, webToken, http.StatusOK, derror.CodeSuccess, &webInfo)
	if webInfo.LoginID != "terminal-user" || webInfo.Device != "web" || webInfo.DeviceID != "browser-1" {
		t.Fatalf("web terminal = %+v, want terminal-user/web/browser-1", webInfo)
	}
	if webInfo.OnlineCount != 2 || webInfo.DeviceCount != 1 || webInfo.DeviceIDCount != 1 {
		t.Fatalf("web counts = %+v, want online=2 device=1 deviceId=1", webInfo)
	}
	if webInfo.LatestForDevice != webToken {
		t.Fatalf("latest web token = %q, want current web token", webInfo.LatestForDevice)
	}

	var mobileInfo struct {
		Device          string `json:"device"`
		DeviceID        string `json:"deviceId"`
		OnlineCount     int    `json:"onlineCount"`
		DeviceCount     int    `json:"deviceCount"`
		DeviceIDCount   int    `json:"deviceIdCount"`
		LatestForDevice string `json:"latestForDevice"`
	}
	c.expect("GET", "/api/terminal", nil, mobileToken, http.StatusOK, derror.CodeSuccess, &mobileInfo)
	if mobileInfo.Device != "mobile" || mobileInfo.DeviceID != "phone-1" {
		t.Fatalf("mobile terminal = %+v, want mobile/phone-1", mobileInfo)
	}
	if mobileInfo.OnlineCount != 2 || mobileInfo.DeviceCount != 1 || mobileInfo.DeviceIDCount != 1 {
		t.Fatalf("mobile counts = %+v, want online=2 device=1 deviceId=1", mobileInfo)
	}
	if mobileInfo.LatestForDevice != mobileToken {
		t.Fatalf("latest mobile token = %q, want current mobile token", mobileInfo.LatestForDevice)
	}
}

// TestSessionQueryFlow verifies token lists, terminal lists, traversal, and search APIs. TestSessionQueryFlow 验证 Token 列表、终端列表、遍历和搜索接口。
func TestSessionQueryFlow(t *testing.T) {
	c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})

	webA := c.loginWithDevice("session-query-user", "web", "browser-a")
	webB := c.loginWithDevice("session-query-user", "web", "browser-b")
	mobileA := c.loginWithDevice("session-query-user", "mobile", "phone-a")

	var tokenList struct {
		Tokens []string `json:"tokens"`
	}
	c.expect("GET", "/api/session/tokens", nil, mobileA, http.StatusOK, derror.CodeSuccess, &tokenList)
	if !sameStringSet(tokenList.Tokens, []string{webA, webB, mobileA}) {
		t.Fatalf("all tokens = %v, want webA/webB/mobileA", tokenList.Tokens)
	}
	c.expect("GET", "/api/session/tokens?device=web", nil, mobileA, http.StatusOK, derror.CodeSuccess, &tokenList)
	if !sameStringSet(tokenList.Tokens, []string{webA, webB}) {
		t.Fatalf("web tokens = %v, want webA/webB", tokenList.Tokens)
	}
	c.expect("GET", "/api/session/tokens?device=web&deviceId=browser-b", nil, mobileA, http.StatusOK, derror.CodeSuccess, &tokenList)
	if !sameStringSet(tokenList.Tokens, []string{webB}) {
		t.Fatalf("web/browser-b tokens = %v, want webB", tokenList.Tokens)
	}

	var terminals struct {
		Terminals []struct {
			Token    string `json:"token"`
			Device   string `json:"device"`
			DeviceID string `json:"deviceId"`
		} `json:"terminals"`
	}
	c.expect("GET", "/api/session/terminals?device=web", nil, mobileA, http.StatusOK, derror.CodeSuccess, &terminals)
	if len(terminals.Terminals) != 2 {
		t.Fatalf("web terminals = %v, want two terminals", terminals.Terminals)
	}

	var visited struct {
		Visited []string `json:"visited"`
	}
	c.expect("GET", "/api/session/foreach?limit=2", nil, mobileA, http.StatusOK, derror.CodeSuccess, &visited)
	if len(visited.Visited) != 2 {
		t.Fatalf("visited = %v, want early stop at 2", visited.Visited)
	}
	c.expect("GET", "/api/session/foreach?device=mobile", nil, mobileA, http.StatusOK, derror.CodeSuccess, &visited)
	if !sameStringSet(visited.Visited, []string{mobileA}) {
		t.Fatalf("mobile visited = %v, want mobile token", visited.Visited)
	}

	var search struct {
		Tokens     []string `json:"tokens"`
		SessionIDs []string `json:"sessionIds"`
	}
	c.expect("GET", "/api/session/search?keyword="+webA+"&start=0&size=-1", nil, mobileA, http.StatusOK, derror.CodeSuccess, &search)
	if !sameStringSet(search.Tokens, []string{webA}) {
		t.Fatalf("token search result = %+v, want webA token", search)
	}
	c.expect("GET", "/api/session/search?keyword=session-query-user&start=0&size=-1", nil, mobileA, http.StatusOK, derror.CodeSuccess, &search)
	if !sameStringSet(search.SessionIDs, []string{"session-query-user"}) {
		t.Fatalf("session search result = %+v, want one session id", search)
	}
	c.expect("GET", "/api/session/search?keyword=session-query-user&start=1&size=1", nil, mobileA, http.StatusOK, derror.CodeSuccess, &search)
	if len(search.SessionIDs) != 0 {
		t.Fatalf("paged session ids = %v, want empty page after first result", search.SessionIDs)
	}
}

// TestTerminalOperationFlow verifies device-specific logout, device kickout, and account replace. TestTerminalOperationFlow 验证按具体设备注销、按设备类型踢下线和按账号顶下线。
func TestTerminalOperationFlow(t *testing.T) {
	t.Run("logout-device", func(t *testing.T) {
		c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})
		webToken := c.loginWithDevice("terminal-op-user", "web", "browser-1")
		mobileToken := c.loginWithDevice("terminal-op-user", "mobile", "phone-1")

		c.expect("POST", "/api/logout/device/web/browser-1", nil, mobileToken, http.StatusOK, derror.CodeSuccess, nil)
		c.expect("GET", "/api/me", nil, webToken, http.StatusUnauthorized, derror.CodeNotLogin, nil)
		c.expect("GET", "/api/me", nil, mobileToken, http.StatusOK, derror.CodeSuccess, nil)
	})

	t.Run("kickout-device", func(t *testing.T) {
		c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})
		webToken := c.loginWithDevice("terminal-kick-user", "web", "browser-1")
		mobileToken := c.loginWithDevice("terminal-kick-user", "mobile", "phone-1")

		c.expect("POST", "/api/kickout/device/web", nil, mobileToken, http.StatusOK, derror.CodeSuccess, nil)
		c.expect("GET", "/api/me", nil, webToken, http.StatusUnauthorized, derror.CodeNotLogin, nil)
		c.expect("GET", "/api/me", nil, mobileToken, http.StatusOK, derror.CodeSuccess, nil)
	})

	t.Run("replace-account", func(t *testing.T) {
		c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})
		webToken := c.loginWithDevice("terminal-replace-user", "web", "browser-1")
		mobileToken := c.loginWithDevice("terminal-replace-user", "mobile", "phone-1")

		c.expect("POST", "/api/replace/account", nil, mobileToken, http.StatusOK, derror.CodeSuccess, nil)
		c.expect("GET", "/api/me", nil, webToken, http.StatusUnauthorized, derror.CodeNotLogin, nil)
		c.expect("GET", "/api/me", nil, mobileToken, http.StatusUnauthorized, derror.CodeNotLogin, nil)
	})
}

// TestConcurrencyPolicyFlow verifies shared token, overflow, and non-concurrent policies. TestConcurrencyPolicyFlow 验证共享 Token、超限下线和非并发登录策略。
func TestConcurrencyPolicyFlow(t *testing.T) {
	t.Run("share-same-device-token", func(t *testing.T) {
		share := true
		c := newFlowClient(t, gincoreapp.Config{
			TokenTimeout:  30 * time.Second,
			ActiveTimeout: -1,
			IsShare:       &share,
		})
		first := c.loginWithDevice("share-user", "web", "browser-1")
		second := c.loginWithDevice("share-user", "web", "browser-1")
		if first != second {
			t.Fatalf("shared token second = %q, want first token %q", second, first)
		}
		var session struct {
			TerminalCount int `json:"terminalCount"`
		}
		c.expect("GET", "/api/session", nil, first, http.StatusOK, derror.CodeSuccess, &session)
		if session.TerminalCount != 1 {
			t.Fatalf("shared session terminal count = %d, want 1", session.TerminalCount)
		}
	})

	t.Run("account-overflow-kicks-oldest", func(t *testing.T) {
		share := false
		c := newFlowClient(t, gincoreapp.Config{
			TokenTimeout:       30 * time.Second,
			ActiveTimeout:      -1,
			IsShare:            &share,
			MaxLoginCount:      2,
			OverflowLogoutMode: config.LogoutModeKickout,
		})
		first := c.loginWithDevice("overflow-user", "web", "a")
		second := c.loginWithDevice("overflow-user", "mobile", "b")
		third := c.loginWithDevice("overflow-user", "desktop", "c")

		c.expect("GET", "/api/me", nil, first, http.StatusUnauthorized, derror.CodeNotLogin, nil)
		c.expect("GET", "/api/me", nil, second, http.StatusOK, derror.CodeSuccess, nil)
		c.expect("GET", "/api/me", nil, third, http.StatusOK, derror.CodeSuccess, nil)
	})

	t.Run("device-scope-overflow-keeps-other-devices", func(t *testing.T) {
		share := false
		c := newFlowClient(t, gincoreapp.Config{
			TokenTimeout:       30 * time.Second,
			ActiveTimeout:      -1,
			IsShare:            &share,
			MaxLoginCount:      2,
			ConcurrencyScope:   config.ConcurrencyScopeDevice,
			OverflowLogoutMode: config.LogoutModeKickout,
		})
		webA := c.loginWithDevice("device-overflow-user", "web", "a")
		webB := c.loginWithDevice("device-overflow-user", "web", "b")
		mobile := c.loginWithDevice("device-overflow-user", "mobile", "a")
		webC := c.loginWithDevice("device-overflow-user", "web", "c")

		c.expect("GET", "/api/me", nil, webA, http.StatusUnauthorized, derror.CodeNotLogin, nil)
		c.expect("GET", "/api/me", nil, webB, http.StatusOK, derror.CodeSuccess, nil)
		c.expect("GET", "/api/me", nil, webC, http.StatusOK, derror.CodeSuccess, nil)
		c.expect("GET", "/api/me", nil, mobile, http.StatusOK, derror.CodeSuccess, nil)
	})

	t.Run("non-concurrent-replaces-old-account", func(t *testing.T) {
		concurrent := false
		c := newFlowClient(t, gincoreapp.Config{
			TokenTimeout:          30 * time.Second,
			ActiveTimeout:         -1,
			IsConcurrent:          &concurrent,
			ConcurrencyScope:      config.ConcurrencyScopeAccount,
			ReplacedLoginExitMode: config.ReplacedLoginExitModeOldDevice,
		})
		first := c.loginWithDevice("nonconcurrent-user", "web", "a")
		second := c.loginWithDevice("nonconcurrent-user", "mobile", "b")
		if first == second {
			t.Fatal("non-concurrent login reused token, want replacement")
		}
		c.expect("GET", "/api/me", nil, first, http.StatusUnauthorized, derror.CodeNotLogin, nil)
		c.expect("GET", "/api/me", nil, second, http.StatusOK, derror.CodeSuccess, nil)
	})

	t.Run("non-concurrent-new-device-rejects-login", func(t *testing.T) {
		concurrent := false
		c := newFlowClient(t, gincoreapp.Config{
			TokenTimeout:          30 * time.Second,
			ActiveTimeout:         -1,
			IsConcurrent:          &concurrent,
			ReplacedLoginExitMode: config.ReplacedLoginExitModeNewDevice,
		})
		token := c.loginWithDevice("reject-new-device-user", "web", "a")
		c.expect("POST", "/login", map[string]any{
			"username": "reject-new-device-user",
			"password": "123456",
			"device":   "mobile",
			"deviceId": "b",
		}, "", http.StatusForbidden, derror.CodeMaxLoginCount, nil)
		c.expect("GET", "/api/me", nil, token, http.StatusOK, derror.CodeSuccess, nil)
	})
}

// TestDisableFlow verifies account and service disable behavior through routes. TestDisableFlow 验证封禁流程：账号封禁阻止旧 Token 和新登录，服务封禁只阻止指定服务。
func TestDisableFlow(t *testing.T) {
	t.Run("account", func(t *testing.T) {
		c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})
		token := c.login("erin")

		// Step 1: disable current account through protected API. 步骤 1：通过受保护接口封禁当前账号。
		c.expect("POST", "/api/disable/account", map[string]any{"reason": "risk"}, token, http.StatusOK, derror.CodeSuccess, nil)

		var info struct {
			Disabled bool   `json:"disabled"`
			Reason   string `json:"reason"`
			TTL      int64  `json:"ttl"`
		}
		c.expect("GET", "/operator/disable/account/erin", nil, "", http.StatusOK, derror.CodeSuccess, &info)
		if !info.Disabled || info.Reason != "risk" || info.TTL <= 0 || info.TTL > 60 {
			t.Fatalf("account disable info = %+v, want disabled risk with ttl 1..60", info)
		}

		// Step 2: old token should be rejected as account disabled. 步骤 2：旧 Token 再访问接口，预期账号封禁错误。
		c.expect("GET", "/api/me", nil, token, http.StatusForbidden, derror.CodeAccountDisabled, nil)

		// Step 3: same account should not be able to login again while disabled. 步骤 3：同账号在封禁期间重新登录，预期被拒绝。
		c.expect("POST", "/login", map[string]any{"username": "erin", "password": "123456"}, "", http.StatusForbidden, derror.CodeAccountDisabled, nil)
	})

	t.Run("service", func(t *testing.T) {
		c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})
		token := c.login("frank")

		// Step 1: payment service is available before service disable. 步骤 1：服务封禁前，支付接口可正常访问。
		c.expect("GET", "/api/payment", nil, token, http.StatusOK, derror.CodeSuccess, nil)

		// Step 2: disable only the payment service for current account. 步骤 2：只封禁当前账号的 payment 服务。
		c.expect("POST", "/api/disable/service/payment", map[string]any{"reason": "risk"}, token, http.StatusOK, derror.CodeSuccess, nil)

		var info struct {
			Disabled bool   `json:"disabled"`
			Service  string `json:"service"`
			Level    int    `json:"level"`
			Reason   string `json:"reason"`
			TTL      int64  `json:"ttl"`
		}
		c.expect("GET", "/operator/disable/service/frank/payment", nil, "", http.StatusOK, derror.CodeSuccess, &info)
		if !info.Disabled || info.Service != "payment" || info.Level != 1 || info.Reason != "risk" || info.TTL <= 0 || info.TTL > 60 {
			t.Fatalf("service disable info = %+v, want payment level 1 risk with ttl 1..60", info)
		}

		// Step 3: payment API is rejected, while login state itself is still valid. 步骤 3：支付接口被拒绝，但不是整账号下线。
		c.expect("GET", "/api/payment", nil, token, http.StatusForbidden, derror.CodePermissionDenied, nil)
	})
}

// TestUntieFlow verifies account, service, and device disable states can be removed. TestUntieFlow 验证账号、服务和设备封禁状态可以被解除。
func TestUntieFlow(t *testing.T) {
	t.Run("account", func(t *testing.T) {
		c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})
		token := c.login("untie-account-user")

		c.expect("POST", "/api/disable/account", map[string]any{"reason": "risk"}, token, http.StatusOK, derror.CodeSuccess, nil)
		c.expect("POST", "/login", map[string]any{"username": "untie-account-user", "password": "123456"}, "", http.StatusForbidden, derror.CodeAccountDisabled, nil)

		c.expect("POST", "/operator/untie/account/untie-account-user", nil, "", http.StatusOK, derror.CodeSuccess, nil)
		newToken := c.login("untie-account-user")
		c.expect("GET", "/api/me", nil, newToken, http.StatusOK, derror.CodeSuccess, nil)
	})

	t.Run("service", func(t *testing.T) {
		c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})
		token := c.login("untie-service-user")

		c.expect("POST", "/api/disable/service/payment", map[string]any{"reason": "risk"}, token, http.StatusOK, derror.CodeSuccess, nil)
		c.expect("GET", "/api/payment", nil, token, http.StatusForbidden, derror.CodePermissionDenied, nil)

		c.expect("POST", "/api/untie/service/payment", nil, token, http.StatusOK, derror.CodeSuccess, nil)
		c.expect("GET", "/api/payment", nil, token, http.StatusOK, derror.CodeSuccess, nil)
	})

	t.Run("device", func(t *testing.T) {
		c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})
		token := c.loginWithDevice("untie-device-user", "web", "browser-1")

		c.expect("POST", "/api/disable/device/web", map[string]any{"reason": "risk"}, token, http.StatusOK, derror.CodeSuccess, nil)
		c.expect("POST", "/login", map[string]any{
			"username": "untie-device-user",
			"password": "123456",
			"device":   "web",
			"deviceId": "browser-2",
		}, "", http.StatusForbidden, derror.CodeAccountDisabled, nil)

		c.expect("POST", "/operator/untie/device/untie-device-user/web", nil, "", http.StatusOK, derror.CodeSuccess, nil)
		newToken := c.loginWithDevice("untie-device-user", "web", "browser-2")
		c.expect("GET", "/api/me", nil, newToken, http.StatusOK, derror.CodeSuccess, nil)
	})
}

// TestDeviceDisableFlow verifies device disable only blocks matching device dimensions. TestDeviceDisableFlow 验证设备封禁流程：被封禁设备无法登录，其他设备仍可登录。
func TestDeviceDisableFlow(t *testing.T) {
	c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})
	adminToken := c.loginWithDevice("device-user", "web", "browser-1")

	// Step 1: disable the web device type for current account. 步骤 1：封禁当前账号的 web 设备类型。
	c.expect("POST", "/api/disable/device/web", map[string]any{"reason": "risk"}, adminToken, http.StatusOK, derror.CodeSuccess, nil)

	var info struct {
		Disabled bool   `json:"disabled"`
		Device   string `json:"device"`
		Reason   string `json:"reason"`
		TTL      int64  `json:"ttl"`
	}
	c.expect("GET", "/operator/disable/device/device-user/web", nil, "", http.StatusOK, derror.CodeSuccess, &info)
	if !info.Disabled || info.Device != "web" || info.Reason != "risk" || info.TTL <= 0 || info.TTL > 60 {
		t.Fatalf("device disable info = %+v, want web risk with ttl 1..60", info)
	}

	// Step 2: web login should be rejected. 步骤 2：web 设备再次登录，预期被拒绝。
	c.expect("POST", "/login", map[string]any{
		"username": "device-user",
		"password": "123456",
		"device":   "web",
		"deviceId": "browser-2",
	}, "", http.StatusForbidden, derror.CodeAccountDisabled, nil)

	// Step 3: mobile login should still be accepted. 步骤 3：mobile 设备登录不受 web 设备封禁影响。
	mobileToken := c.loginWithDevice("device-user", "mobile", "phone-1")
	c.expect("GET", "/api/me", nil, mobileToken, http.StatusOK, derror.CodeSuccess, nil)
}

// TestConcreteDeviceDisableFlow verifies only the exact device ID is blocked. TestConcreteDeviceDisableFlow 验证只封禁命中的具体设备 ID。
func TestConcreteDeviceDisableFlow(t *testing.T) {
	c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})
	token := c.loginWithDevice("concrete-device-user", "web", "browser-1")

	// Step 1: disable only web/browser-1 for current account. 步骤 1：只封禁当前账号的 web/browser-1 具体设备。
	c.expect("POST", "/api/disable/device/web/browser-1", map[string]any{"reason": "risk"}, token, http.StatusOK, derror.CodeSuccess, nil)

	var info struct {
		Disabled bool   `json:"disabled"`
		Device   string `json:"device"`
		DeviceID string `json:"deviceId"`
		Reason   string `json:"reason"`
		TTL      int64  `json:"ttl"`
	}
	c.expect("GET", "/operator/disable/device/concrete-device-user/web/browser-1", nil, "", http.StatusOK, derror.CodeSuccess, &info)
	if !info.Disabled || info.Device != "web" || info.DeviceID != "browser-1" || info.Reason != "risk" || info.TTL <= 0 || info.TTL > 60 {
		t.Fatalf("concrete device disable info = %+v, want web/browser-1 risk with ttl 1..60", info)
	}

	// Step 2: same concrete device is rejected. 步骤 2：同一个具体设备再次登录会被拒绝。
	c.expect("POST", "/login", map[string]any{
		"username": "concrete-device-user",
		"password": "123456",
		"device":   "web",
		"deviceId": "browser-1",
	}, "", http.StatusForbidden, derror.CodeAccountDisabled, nil)

	// Step 3: same device type with a different device ID is allowed. 步骤 3：相同设备类型但不同设备 ID 不受影响。
	otherWebToken := c.loginWithDevice("concrete-device-user", "web", "browser-2")
	c.expect("GET", "/api/me", nil, otherWebToken, http.StatusOK, derror.CodeSuccess, nil)

	// Step 4: untie the concrete device and verify the original device can login again. 步骤 4：解除具体设备封禁后，原设备可以重新登录。
	c.expect("POST", "/operator/untie/device/concrete-device-user/web/browser-1", nil, "", http.StatusOK, derror.CodeSuccess, nil)
	originalDeviceToken := c.loginWithDevice("concrete-device-user", "web", "browser-1")
	c.expect("GET", "/api/me", nil, originalDeviceToken, http.StatusOK, derror.CodeSuccess, nil)
}

// TestNonceFlow verifies nonce generation and one-time consumption. TestNonceFlow 验证 nonce 流程：生成 nonce、首次校验成功、重复使用失败。
func TestNonceFlow(t *testing.T) {
	c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})

	// Step 1: generate a nonce from public API. 步骤 1：通过公开接口生成 nonce。
	var generated struct {
		Nonce string `json:"nonce"`
	}
	c.expect("GET", "/nonce", nil, "", http.StatusOK, derror.CodeSuccess, &generated)
	if generated.Nonce == "" {
		t.Fatal("nonce is empty")
	}

	// Step 2: status check proves the nonce is valid without consuming it. 步骤 2：状态查询证明 nonce 有效，但不会消费它。
	var status struct {
		Valid bool  `json:"valid"`
		TTL   int64 `json:"ttl"`
	}
	c.expect("GET", "/nonce/status/"+generated.Nonce, nil, "", http.StatusOK, derror.CodeSuccess, &status)
	if !status.Valid || status.TTL <= 0 {
		t.Fatalf("nonce status = %+v, want valid with positive ttl", status)
	}

	// Step 3: verify the nonce once, then verify the same nonce again. 步骤 3：第一次校验成功，第二次重复使用应失败。
	body := map[string]any{"nonce": generated.Nonce}
	c.expect("POST", "/nonce/verify", body, "", http.StatusOK, derror.CodeSuccess, nil)
	c.expect("POST", "/nonce/verify", body, "", http.StatusBadRequest, derror.CodeBadRequest, nil)
	c.expect("GET", "/nonce/status/"+generated.Nonce, nil, "", http.StatusOK, derror.CodeSuccess, &status)
	if status.Valid {
		t.Fatalf("nonce status after consume = %+v, want invalid", status)
	}
}

// TestOAuth2AuthorizationCodeFlow verifies code exchange, introspection, refresh, and revoke. TestOAuth2AuthorizationCodeFlow 验证 OAuth2 授权码流程：生成 code、换 token、查询 token、刷新、撤销。
func TestOAuth2AuthorizationCodeFlow(t *testing.T) {
	c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})

	// Step 1: generate an authorization code for the demo client. 步骤 1：为示例客户端生成授权码。
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

	// Step 2: exchange authorization code for access and refresh tokens. 步骤 2：使用授权码换取访问令牌和刷新令牌。
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

	// Step 3: authorization code is single-use. 步骤 3：授权码只能使用一次，重复换取应失败。
	c.expect("POST", "/oauth2/token", map[string]any{
		"grantType":    "authorization_code",
		"clientId":     "demo-client",
		"clientSecret": "demo-secret",
		"code":         codeData.Code,
		"redirectUri":  "https://client.example/callback",
	}, "", http.StatusBadRequest, derror.CodeBadRequest, nil)

	// Step 4: introspect access token, expect active token info. 步骤 4：查询访问令牌信息，预期处于有效状态。
	var info struct {
		Active   bool   `json:"active"`
		UserID   string `json:"userId"`
		ClientID string `json:"clientId"`
	}
	c.expect("GET", "/oauth2/introspect", nil, token.AccessToken, http.StatusOK, derror.CodeSuccess, &info)
	if !info.Active || info.UserID != "oauth-user" || info.ClientID != "demo-client" {
		t.Fatalf("oauth2 introspection = %+v, want active oauth-user/demo-client", info)
	}

	// Step 5: refresh token rotates old access and refresh tokens. 步骤 5：使用刷新令牌换新令牌，旧访问令牌应失效。
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

	// Step 6: revoke refreshed token and verify it is no longer active. 步骤 6：撤销刷新后的访问令牌，并验证它已失效。
	c.expect("POST", "/oauth2/revoke", map[string]any{"token": refreshed.AccessToken}, "", http.StatusOK, derror.CodeSuccess, nil)
	c.expect("GET", "/oauth2/introspect", nil, refreshed.AccessToken, http.StatusUnauthorized, derror.CodeNotLogin, nil)
}

// TestOAuth2PasswordAndClientCredentialsFlow verifies additional OAuth2 grant types. TestOAuth2PasswordAndClientCredentialsFlow 验证 OAuth2 密码模式和客户端凭证模式。
func TestOAuth2PasswordAndClientCredentialsFlow(t *testing.T) {
	c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})

	// Step 1: use password grant with demo user credentials. 步骤 1：使用密码模式和示例用户凭证换取令牌。
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

	// Step 2: use client credentials grant for machine-to-machine access. 步骤 2：使用客户端凭证模式换取机器访问令牌。
	clientToken := c.oauth2Token(map[string]any{
		"grantType":    "client_credentials",
		"clientId":     "demo-client",
		"clientSecret": "demo-secret",
		"scopes":       []string{"read"},
	})
	if clientToken.UserID != "demo-client" {
		t.Fatalf("client credentials userId = %q, want demo-client", clientToken.UserID)
	}

	// Step 3: invalid client secret should be rejected. 步骤 3：错误客户端密钥应被拒绝。
	c.expect("POST", "/oauth2/token", map[string]any{
		"grantType":    "client_credentials",
		"clientId":     "demo-client",
		"clientSecret": "wrong-secret",
	}, "", http.StatusBadRequest, derror.CodeBadRequest, nil)
}

// TestMultiAuthIsolationFlow verifies independent auth systems do not share tokens or access data. TestMultiAuthIsolationFlow 验证多认证体系隔离：不同 AuthType 的 Token、权限、角色和 Session 互不串用。
func TestMultiAuthIsolationFlow(t *testing.T) {
	c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})

	// Step 1: login same loginID into user-auth and admin-auth systems. 步骤 1：同一个 loginID 分别登录 user-auth 和 admin-auth 两个认证体系。
	userToken := c.multiAuthLogin("/multi-auth/user/login", "same-id", "web", "user-browser")
	adminToken := c.multiAuthLogin("/multi-auth/admin/login", "same-id", "web", "admin-browser")
	if userToken == adminToken {
		t.Fatal("user and admin tokens are equal, want isolated token values")
	}

	// Step 2: tokens are accepted only by their own auth system. 步骤 2：Token 只能访问所属认证体系的接口，不能跨体系使用。
	c.expect("GET", "/multi-auth/user/me", nil, userToken, http.StatusOK, derror.CodeSuccess, nil)
	c.expect("GET", "/multi-auth/admin/me", nil, adminToken, http.StatusOK, derror.CodeSuccess, nil)
	c.expect("GET", "/multi-auth/admin/me", nil, userToken, http.StatusUnauthorized, derror.CodeNotLogin, nil)
	c.expect("GET", "/multi-auth/user/me", nil, adminToken, http.StatusUnauthorized, derror.CodeNotLogin, nil)

	// Step 3: user-auth permission does not grant admin-auth role. 步骤 3：user-auth 的权限不会影响 admin-auth 的角色校验。
	c.expect("POST", "/multi-auth/user/permissions", map[string]any{"value": "profile:read"}, userToken, http.StatusOK, derror.CodeSuccess, nil)
	c.expect("GET", "/multi-auth/user/profile", nil, userToken, http.StatusOK, derror.CodeSuccess, nil)
	c.expect("GET", "/multi-auth/admin/dashboard", nil, adminToken, http.StatusForbidden, derror.CodePermissionDenied, nil)

	// Step 4: admin-auth role does not grant user-auth token access. 步骤 4：admin-auth 的角色只在 admin-auth 内生效，不会让 adminToken 访问 user-auth。
	c.expect("POST", "/multi-auth/admin/roles", map[string]any{"value": "admin"}, adminToken, http.StatusOK, derror.CodeSuccess, nil)
	c.expect("GET", "/multi-auth/admin/dashboard", nil, adminToken, http.StatusOK, derror.CodeSuccess, nil)
	c.expect("GET", "/multi-auth/user/profile", nil, adminToken, http.StatusUnauthorized, derror.CodeNotLogin, nil)
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

func (c *flowClient) multiAuthLogin(path, username, device, deviceID string) string {
	c.t.Helper()

	var data struct {
		Auth  string `json:"auth"`
		Token string `json:"token"`
	}
	c.expect("POST", path, map[string]any{
		"username": username,
		"password": "123456",
		"device":   device,
		"deviceId": deviceID,
	}, "", http.StatusOK, derror.CodeSuccess, &data)
	if data.Token == "" {
		c.t.Fatalf("%s token is empty", path)
	}
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

func sameStringSet(got, want []string) bool {
	got = append([]string(nil), got...)
	want = append([]string(nil), want...)
	slices.Sort(got)
	slices.Sort(want)
	return slices.Equal(got, want)
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
