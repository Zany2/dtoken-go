package main

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/Zany2/dtoken-go"
	dterror "github.com/Zany2/dtoken-go/errors"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestLoginValidationAndAuthorizationSeed verifies login validation and demo authorization data. TestLoginValidationAndAuthorizationSeed 验证登录校验与示例授权数据初始化。
func TestLoginValidationAndAuthorizationSeed(t *testing.T) {
	setupKratosManager(t)
	server, err := newKratosExampleServer(registeredAuth())
	if err != nil {
		t.Fatal(err)
	}

	missing := requestKratos(t, server, http.MethodPost, "/login", `{"username":"alice"}`, "")
	if missing.Code != http.StatusBadRequest {
		t.Fatalf("missing credentials status = %d, want %d", missing.Code, http.StatusBadRequest)
	}
	invalid := requestKratos(t, server, http.MethodPost, "/login", `{"username":"alice","password":"bad"}`, "")
	if invalid.Code != http.StatusUnauthorized {
		t.Fatalf("invalid password status = %d, want %d", invalid.Code, http.StatusUnauthorized)
	}

	login := requestKratos(t, server, http.MethodPost, "/login", `{"username":"alice","password":"123456"}`, "")
	if login.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d", login.Code, http.StatusOK)
	}
	var envelope struct {
		Code int `json:"code"`
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(login.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if envelope.Code != dterror.CodeSuccess || envelope.Data.Token == "" {
		t.Fatalf("login envelope = %+v, want success and token", envelope)
	}
	if id, err := dtoken.GetLoginID(context.Background(), envelope.Data.Token); err != nil || id != "alice" {
		t.Fatalf("returned token login ID=%q error=%v", id, err)
	}
	if !dtoken.HasRole(context.Background(), "alice", "admin") {
		t.Fatal("login did not seed admin role")
	}
	if !dtoken.HasPermission(context.Background(), "alice", "article:read") {
		t.Fatal("login did not seed article permission")
	}
}

// TestProtectedRoutesAndLogout verifies middleware-protected routes and token invalidation after logout. TestProtectedRoutesAndLogout 验证中间件保护路由及注销后的 Token 失效。
func TestProtectedRoutesAndLogout(t *testing.T) {
	setupKratosManager(t)
	ctx := context.Background()
	token, err := dtoken.Login(ctx, dtoken.LoginOptions{LoginID: "alice"})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err = dtoken.AddRoles(ctx, dtoken.RoleOptions{LoginID: "alice", Roles: []string{"admin"}}); err != nil {
		t.Fatalf("AddRoles() error = %v", err)
	}
	if err = dtoken.AddPermissions(ctx, dtoken.PermissionOptions{LoginID: "alice", Permissions: []string{"article:read"}}); err != nil {
		t.Fatalf("AddPermissions() error = %v", err)
	}

	server, err := newKratosExampleServer(registeredAuth())
	if err != nil {
		t.Fatal(err)
	}
	for _, route := range []string{"/me", "/admin", "/articles"} {
		for _, header := range []string{"dtoken", "Authorization"} {
			for _, value := range []string{token, "Bearer " + token} {
				req := httptest.NewRequest(http.MethodGet, route, nil)
				req.Header.Set(header, value)
				response := httptest.NewRecorder()
				server.ServeHTTP(response, req)
				assertKratosResponse(t, response, http.StatusOK, dterror.CodeSuccess)
			}
		}
	}
	for _, route := range []string{"/me", "/admin", "/articles", "/logout"} {
		method := http.MethodGet
		if route == "/logout" {
			method = http.MethodPost
		}
		for _, invalidToken := range []string{"", "invalid-token"} {
			assertKratosResponse(t, requestKratos(t, server, method, route, "", invalidToken), http.StatusUnauthorized, dterror.CodeNotLogin)
		}
	}

	bobToken, err := dtoken.Login(ctx, dtoken.LoginOptions{LoginID: "bob"})
	if err != nil {
		t.Fatalf("Login(bob) error = %v", err)
	}
	for _, route := range []string{"/admin", "/articles"} {
		response := requestKratos(t, server, http.MethodGet, route, "", bobToken)
		assertKratosResponse(t, response, http.StatusForbidden, dterror.CodePermissionDenied)
	}

	logout := requestKratos(t, server, http.MethodPost, "/logout", "", token)
	if logout.Code != http.StatusOK {
		t.Fatalf("logout status = %d, want %d", logout.Code, http.StatusOK)
	}
	afterLogout := requestKratos(t, server, http.MethodGet, "/me", "", token)
	if afterLogout.Code != http.StatusUnauthorized {
		t.Fatalf("request after logout status = %d, want %d", afterLogout.Code, http.StatusUnauthorized)
	}
}

func setupKratosManager(t *testing.T) {
	t.Helper()
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	if _, err := initDToken(); err != nil {
		t.Fatal(err)
	}
}

func requestKratos(t *testing.T, server *khttp.Server, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		mgr, err := dtoken.GetManager()
		if err != nil {
			t.Fatalf("GetManager() error = %v", err)
		}
		req.Header.Set(mgr.GetConfig().TokenName, token)
	}
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, req)
	return recorder
}

// registeredAuth returns the explicitly registered test fixture. registeredAuth 返回显式注册的测试夹具。
func registeredAuth() *dtoken.Auth {
	mgr, err := dtoken.GetManager()
	if err != nil {
		panic(err)
	}
	return dtoken.New(mgr)
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
	srv, err := newKratosExampleServer(auth)
	if err != nil {
		t.Fatal(err)
	}
	response := requestKratos(t, srv, http.MethodPost, "/login", `{"username":"alice","password":"123456"}`, "")
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
