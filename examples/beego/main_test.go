package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Zany2/dtoken-go/dtoken"
	beegodt "github.com/Zany2/dtoken-go/integrations/beego"
	beegocontext "github.com/beego/beego/v2/server/web/context"
)

// TestLoginValidationAndAuthorizationSeed verifies login validation and demo authorization data. TestLoginValidationAndAuthorizationSeed 验证登录校验与示例授权数据初始化。
func TestLoginValidationAndAuthorizationSeed(t *testing.T) {
	setupBeegoManager(t)
	ctx := context.Background()

	missing, missingRecorder := newBeegoExampleContext(http.MethodPost, "/login?username=alice")
	handleLogin(missing)
	if missingRecorder.Code != http.StatusBadRequest {
		t.Fatalf("missing credentials status = %d, want %d", missingRecorder.Code, http.StatusBadRequest)
	}

	invalid, invalidRecorder := newBeegoExampleContext(http.MethodPost, "/login?username=alice&password=bad")
	handleLogin(invalid)
	if invalidRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("invalid password status = %d, want %d", invalidRecorder.Code, http.StatusUnauthorized)
	}

	login, loginRecorder := newBeegoExampleContext(http.MethodPost, "/login?username=alice&password=123456")
	handleLogin(login)
	if loginRecorder.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d", loginRecorder.Code, http.StatusOK)
	}
	var envelope struct {
		Code int `json:"code"`
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(loginRecorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if envelope.Code != beegodt.CodeSuccess || envelope.Data.Token == "" {
		t.Fatalf("login envelope = %+v, want success and token", envelope)
	}
	if !dtoken.HasRole(ctx, "alice", "admin") {
		t.Fatal("login did not seed admin role")
	}
	if !dtoken.HasPermission(ctx, "alice", "article:read") {
		t.Fatal("login did not seed article permission")
	}
}

// TestProtectedHandlersAndLogout verifies Beego filter order, protected handlers, and logout invalidation. TestProtectedHandlersAndLogout 验证 Beego 过滤器顺序、受保护处理器及注销失效。
func TestProtectedHandlersAndLogout(t *testing.T) {
	setupBeegoManager(t)
	ctx := context.Background()
	token, err := dtoken.Login(ctx, "alice")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err = dtoken.AddRoles(ctx, "alice", []string{"admin"}); err != nil {
		t.Fatalf("AddRoles() error = %v", err)
	}
	if err = dtoken.AddPermissions(ctx, "alice", []string{"article:read"}); err != nil {
		t.Fatalf("AddPermissions() error = %v", err)
	}

	me, meRecorder := newBeegoExampleContextWithToken(http.MethodGet, "/me", token)
	beegodt.RegisterDTokenContextMiddleware(ctx)(me)
	beegodt.AuthMiddleware(ctx)(me)
	handleMe(me)
	if meRecorder.Code != http.StatusOK {
		t.Fatalf("authorized /me status = %d, want %d", meRecorder.Code, http.StatusOK)
	}

	admin, adminRecorder := newBeegoExampleContextWithToken(http.MethodGet, "/admin", token)
	beegodt.RegisterDTokenContextMiddleware(ctx)(admin)
	beegodt.AuthMiddleware(ctx)(admin)
	beegodt.RoleMiddleware(ctx, []string{"admin"})(admin)
	handleAdmin(admin)
	if adminRecorder.Code != http.StatusOK {
		t.Fatalf("authorized /admin status = %d, want %d", adminRecorder.Code, http.StatusOK)
	}

	articles, articlesRecorder := newBeegoExampleContextWithToken(http.MethodGet, "/articles", token)
	beegodt.RegisterDTokenContextMiddleware(ctx)(articles)
	beegodt.AuthMiddleware(ctx)(articles)
	beegodt.PermissionMiddleware(ctx, []string{"article:read"})(articles)
	handleArticles(articles)
	if articlesRecorder.Code != http.StatusOK {
		t.Fatalf("authorized /articles status = %d, want %d", articlesRecorder.Code, http.StatusOK)
	}

	unauthorized, unauthorizedRecorder := newBeegoExampleContext(http.MethodGet, "/me")
	handleMe(unauthorized)
	if unauthorizedRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized /me status = %d, want %d", unauthorizedRecorder.Code, http.StatusUnauthorized)
	}

	bobToken, err := dtoken.Login(ctx, "bob")
	if err != nil {
		t.Fatalf("Login(bob) error = %v", err)
	}
	adminDenied, adminDeniedRecorder := newBeegoExampleContextWithToken(http.MethodGet, "/admin", bobToken)
	beegodt.RegisterDTokenContextMiddleware(ctx)(adminDenied)
	beegodt.AuthMiddleware(ctx)(adminDenied)
	beegodt.RoleMiddleware(ctx, []string{"admin"})(adminDenied)
	if adminDeniedRecorder.Code != http.StatusForbidden {
		t.Fatalf("unauthorized /admin status = %d, want %d", adminDeniedRecorder.Code, http.StatusForbidden)
	}

	logout, logoutRecorder := newBeegoExampleContextWithToken(http.MethodPost, "/logout", token)
	beegodt.RegisterDTokenContextMiddleware(ctx)(logout)
	beegodt.AuthMiddleware(ctx)(logout)
	handleLogout(logout)
	if logoutRecorder.Code != http.StatusOK {
		t.Fatalf("logout status = %d, want %d", logoutRecorder.Code, http.StatusOK)
	}

	afterLogout, afterLogoutRecorder := newBeegoExampleContextWithToken(http.MethodGet, "/me", token)
	beegodt.RegisterDTokenContextMiddleware(ctx)(afterLogout)
	beegodt.AuthMiddleware(ctx)(afterLogout)
	if afterLogoutRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("request after logout status = %d, want %d", afterLogoutRecorder.Code, http.StatusUnauthorized)
	}
}

func setupBeegoManager(t *testing.T) {
	t.Helper()
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	initDToken()
}

func newBeegoExampleContext(method, path string) (*beegocontext.Context, *httptest.ResponseRecorder) {
	return newBeegoExampleContextWithRequest(httptest.NewRequest(method, path, nil))
}

func newBeegoExampleContextWithToken(method, path, token string) (*beegocontext.Context, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(method, path, nil)
	mgr, _ := dtoken.GetManager()
	req.Header.Set(mgr.GetConfig().TokenName, token)
	return newBeegoExampleContextWithRequest(req)
}

func newBeegoExampleContextWithRequest(req *http.Request) (*beegocontext.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	c := beegocontext.NewContext()
	c.Reset(recorder, req)
	return c, recorder
}
