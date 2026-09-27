package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	beegodt "github.com/Zany2/dtoken-go/integrations/beego"
	web "github.com/beego/beego/v2/server/web"
)

// TestLoginValidationAndAuthorizationSeed verifies login validation and demo authorization data. TestLoginValidationAndAuthorizationSeed 验证登录校验与示例授权数据初始化。
func TestLoginValidationAndAuthorizationSeed(t *testing.T) {
	setupBeegoManager(t)
	router := newBeegoExampleRouter()
	assertBeegoResponse(t, requestBeego(router, http.MethodPost, "/login?username=alice", "", ""), http.StatusBadRequest, beegodt.CodeBadRequest)
	assertBeegoResponse(t, requestBeego(router, http.MethodPost, "/login?username=alice&password=bad", "", ""), http.StatusUnauthorized, beegodt.CodeNotLogin)

	for _, form := range []bool{false, true} {
		path, body := "/login?username=alice&password=123456", ""
		if form {
			path, body = "/login", "username=alice&password=123456"
		}
		login := requestBeego(router, http.MethodPost, path, body, "")
		assertBeegoResponse(t, login, http.StatusOK, beegodt.CodeSuccess)
		var envelope struct {
			Data struct {
				Token string `json:"token"`
			} `json:"data"`
		}
		if err := json.Unmarshal(login.Body.Bytes(), &envelope); err != nil {
			t.Fatal(err)
		}
		if id, err := beegodt.GetLoginID(context.Background(), envelope.Data.Token); err != nil || id != "alice" {
			t.Fatalf("returned token login ID=%q error=%v", id, err)
		}
	}
	if !beegodt.HasRole(context.Background(), "alice", "admin") || !beegodt.HasPermission(context.Background(), "alice", "article:read") {
		t.Fatal("login did not seed demo authorization")
	}
}

// TestProtectedRoutesAndLogout uses native routing to verify failed filters stop protected handlers. TestProtectedRoutesAndLogout 使用真实路由验证过滤器失败后不会执行受保护处理器。
func TestProtectedRoutesAndLogout(t *testing.T) {
	setupBeegoManager(t)
	ctx := context.Background()
	token, err := beegodt.Login(ctx, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if err := beegodt.AddRoles(ctx, "alice", []string{"admin"}); err != nil {
		t.Fatal(err)
	}
	if err := beegodt.AddPermissions(ctx, "alice", []string{"article:read"}); err != nil {
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
				assertBeegoResponse(t, rec, http.StatusOK, beegodt.CodeSuccess)
			}
		}
	}
	for _, path := range []string{"/me", "/admin", "/articles", "/logout"} {
		method := http.MethodGet
		if path == "/logout" {
			method = http.MethodPost
		}
		for _, invalid := range []string{"", "invalid-token"} {
			assertBeegoResponse(t, requestBeego(router, method, path, "", invalid), http.StatusUnauthorized, beegodt.CodeNotLogin)
		}
	}
	bobToken, err := beegodt.Login(ctx, "bob")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/admin", "/articles"} {
		assertBeegoResponse(t, requestBeego(router, http.MethodGet, path, "", bobToken), http.StatusForbidden, beegodt.CodePermissionDenied)
	}
	assertBeegoResponse(t, requestBeego(router, http.MethodPost, "/logout", "", token), http.StatusOK, beegodt.CodeSuccess)
	assertBeegoResponse(t, requestBeego(router, http.MethodGet, "/me", "", token), http.StatusUnauthorized, beegodt.CodeNotLogin)
}

func setupBeegoManager(t *testing.T) {
	t.Helper()
	beegodt.DeleteAllManager()
	t.Cleanup(beegodt.DeleteAllManager)
	initDToken()
}

func newBeegoExampleRouter() *web.ControllerRegister {
	router := web.NewControllerRegister()
	registerRoutes(router)
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
