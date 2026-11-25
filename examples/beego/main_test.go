package main

import (
	"context"
	"encoding/json"
	"github.com/Zany2/dtoken-go"
	dterror "github.com/Zany2/dtoken-go/errors"
	web "github.com/beego/beego/v2/server/web"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestLoginValidationAndAuthorizationSeed verifies login validation and demo authorization data. TestLoginValidationAndAuthorizationSeed 验证登录校验与示例授权数据初始化。
func TestLoginValidationAndAuthorizationSeed(t *testing.T) {
	setupBeegoManager(t)
	router := newBeegoExampleRouter()
	assertBeegoResponse(t, requestBeego(router, http.MethodPost, "/login?username=alice", "", ""), http.StatusBadRequest, dterror.CodeBadRequest)
	assertBeegoResponse(t, requestBeego(router, http.MethodPost, "/login?username=alice&password=bad", "", ""), http.StatusUnauthorized, dterror.CodeNotLogin)

	for _, form := range []bool{false, true} {
		path, body := "/login?username=alice&password=123456", ""
		if form {
			path, body = "/login", "username=alice&password=123456"
		}
		login := requestBeego(router, http.MethodPost, path, body, "")
		assertBeegoResponse(t, login, http.StatusOK, dterror.CodeSuccess)
		var envelope struct {
			Data struct {
				Token string `json:"token"`
			} `json:"data"`
		}
		if err := json.Unmarshal(login.Body.Bytes(), &envelope); err != nil {
			t.Fatal(err)
		}
		if id, err := dtoken.GetLoginID(context.Background(), envelope.Data.Token); err != nil || id != "alice" {
			t.Fatalf("returned token login ID=%q error=%v", id, err)
		}
	}
	if !dtoken.HasRole(context.Background(), "alice", "admin") || !dtoken.HasPermission(context.Background(), "alice", "article:read") {
		t.Fatal("login did not seed demo authorization")
	}
}

// TestProtectedRoutesAndLogout uses native routing to verify failed filters stop protected handlers. TestProtectedRoutesAndLogout 使用真实路由验证过滤器失败后不会执行受保护处理器。
func TestProtectedRoutesAndLogout(t *testing.T) {
	setupBeegoManager(t)
	ctx := context.Background()
	token, err := dtoken.Login(ctx, dtoken.LoginOptions{LoginID: "alice"})
	if err != nil {
		t.Fatal(err)
	}
	if err := dtoken.AddRoles(ctx, dtoken.RoleOptions{LoginID: "alice", Roles: []string{"admin"}}); err != nil {
		t.Fatal(err)
	}
	if err := dtoken.AddPermissions(ctx, dtoken.PermissionOptions{LoginID: "alice", Permissions: []string{"article:read"}}); err != nil {
		t.Fatal(err)
	}
	router := newBeegoExampleRouter()
	for _, path := range []string{"/me", "/admin", "/articles"} {
		for _, header := range []string{"dtoken", "Authorization"} {
			for _, value := range []string{token, "Bearer " + token} {
				req := httptest.NewRequest(http.MethodGet, path, nil)
				req.Header.Set(header, value)
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)
				assertBeegoResponse(t, rec, http.StatusOK, dterror.CodeSuccess)
			}
		}
	}
	for _, path := range []string{"/me", "/admin", "/articles", "/logout"} {
		method := http.MethodGet
		if path == "/logout" {
			method = http.MethodPost
		}
		for _, invalid := range []string{"", "invalid-token"} {
			assertBeegoResponse(t, requestBeego(router, method, path, "", invalid), http.StatusUnauthorized, dterror.CodeNotLogin)
		}
	}
	bobToken, err := dtoken.Login(ctx, dtoken.LoginOptions{LoginID: "bob"})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/admin", "/articles"} {
		assertBeegoResponse(t, requestBeego(router, http.MethodGet, path, "", bobToken), http.StatusForbidden, dterror.CodePermissionDenied)
	}
	assertBeegoResponse(t, requestBeego(router, http.MethodPost, "/logout", "", token), http.StatusOK, dterror.CodeSuccess)
	assertBeegoResponse(t, requestBeego(router, http.MethodGet, "/me", "", token), http.StatusUnauthorized, dterror.CodeNotLogin)
}

func setupBeegoManager(t *testing.T) {
	t.Helper()
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	if _, err := initDToken(); err != nil {
		t.Fatal(err)
	}
}

func newBeegoExampleRouter() *web.ControllerRegister {
	router := web.NewControllerRegister()
	registerTestRoutes(t, router)
	return router
}

func requestBeego(router *web.ControllerRegister, method, path, body, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if token != "" {
		req.Header.Set("dtoken", token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func assertBeegoResponse(t *testing.T, rec *httptest.ResponseRecorder, status, code int) {
	t.Helper()
	var response Response
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("invalid response: %v, body=%s", err, rec.Body.String())
	}
	if rec.Code != status || response.Code != code || !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("response=%d %s, want status=%d code=%d", rec.Code, rec.Body.String(), status, code)
	}
}

// registeredAuth returns the explicitly registered test fixture. registeredAuth 返回显式注册的测试夹具。
func registeredAuth() *dtoken.Auth {
	mgr, err := dtoken.GetManager()
	if err != nil {
		panic(err)
	}
	return dtoken.New(mgr)
}

// registerTestRoutes binds the test fixture to the production routes. registerTestRoutes 将测试夹具绑定到实际路由。
func registerTestRoutes(t *testing.T, r *web.ControllerRegister) {
	t.Helper()
	if err := registerRoutes(r, registeredAuth()); err != nil {
		t.Fatal(err)
	}
}

// TestInjectedLoginUsesBuiltInstance verifies routes and the registry share the built engine. TestInjectedLoginUsesBuiltInstance 验证路由与注册表共用构建返回的引擎。
func TestInjectedLoginUsesBuiltInstance(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	auth, err := initDToken()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(auth.Close)
	router := web.NewControllerRegister()
	if err := registerRoutes(router, auth); err != nil {
		t.Fatal(err)
	}
	response := requestBeego(router, http.MethodPost, "/login", "username=alice&password=123456", "")
	if response.Code != http.StatusOK {
		t.Fatalf("login status = %d", response.Code)
	}
	ctx := context.Background()
	if _, err := auth.GetSession(ctx, "alice"); err != nil {
		t.Fatalf("injected session: %v", err)
	}
	if !auth.HasRole(ctx, "alice", "admin") || !auth.HasPermission(ctx, "alice", "article:read") {
		t.Fatal("access grants missing from injected instance")
	}
	if mgr, err := dtoken.GetManager(); err != nil || mgr != auth.Manager() {
		t.Fatalf("Build registration differs from injected instance: %v", err)
	}
}
