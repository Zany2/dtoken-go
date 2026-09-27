package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/defaults"
	"github.com/Zany2/dtoken-go/dtoken"
	"github.com/gin-gonic/gin"
)

func TestLoginValidationAndAuthorizationSeed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupQuickStartManager(t)

	r := gin.New()
	r.POST("/login", handleLogin)

	missing := requestQuickStart(t, r, http.MethodPost, "/login", `{"username":"alice"}`, "")
	if missing.Code != http.StatusBadRequest {
		t.Fatalf("missing credentials status = %d, want %d", missing.Code, http.StatusBadRequest)
	}

	invalid := requestQuickStart(t, r, http.MethodPost, "/login", `{"username":"alice","password":"bad"}`, "")
	if invalid.Code != http.StatusUnauthorized {
		t.Fatalf("invalid password status = %d, want %d", invalid.Code, http.StatusUnauthorized)
	}

	success := requestQuickStart(t, r, http.MethodPost, "/login", `{"username":"alice","password":"123456"}`, "")
	if success.Code != http.StatusOK {
		t.Fatalf("successful login status = %d, want %d", success.Code, http.StatusOK)
	}
	var response Response
	if err := json.Unmarshal(success.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if response.Code != 0 || response.Message != "ok" {
		t.Fatalf("login response = %+v, want success response", response)
	}
	data, ok := response.Data.(map[string]interface{})
	if !ok || data["tokenHeader"] != tokenHeader {
		t.Fatalf("login data = %#v, want token header metadata", response.Data)
	}
	token, ok := data["token"].(string)
	if !ok || token == "" {
		t.Fatal("login response did not contain a token")
	}
	if loginID, err := dtoken.GetLoginID(context.Background(), token); err != nil || loginID != "alice" {
		t.Fatalf("returned token loginID = %q, error = %v", loginID, err)
	}
	if !dtoken.HasRole(context.Background(), "alice", "admin") {
		t.Fatal("login did not seed admin role")
	}
	if !dtoken.HasPermission(context.Background(), "alice", "article:read") {
		t.Fatal("login did not seed article permission")
	}
}

func TestProtectedRoutesAndLogout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupQuickStartManager(t)
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

	r := gin.New()
	registerRoutes(r)

	for _, route := range []string{"/me", "/admin", "/articles"} {
		response := requestQuickStart(t, r, http.MethodGet, route, "", token)
		if response.Code != http.StatusOK {
			t.Fatalf("authorized %s status = %d, want %d", route, response.Code, http.StatusOK)
		}
	}
	unauthorized := requestQuickStart(t, r, http.MethodGet, "/me", "", "")
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized /me status = %d, want %d", unauthorized.Code, http.StatusUnauthorized)
	}

	logout := requestQuickStart(t, r, http.MethodPost, "/logout", "", token)
	if logout.Code != http.StatusOK {
		t.Fatalf("logout status = %d, want %d", logout.Code, http.StatusOK)
	}
	afterLogout := requestQuickStart(t, r, http.MethodGet, "/me", "", token)
	if afterLogout.Code != http.StatusUnauthorized {
		t.Fatalf("request after logout status = %d, want %d", afterLogout.Code, http.StatusUnauthorized)
	}
}

// TestMiddlewareUsesRequestContext verifies cancellation and deadlines reach all custom auth middleware. TestMiddlewareUsesRequestContext 验证请求取消及截止时间传入全部自定义鉴权中间件。
func TestMiddlewareUsesRequestContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupQuickStartManager(t)
	ctx := context.Background()
	token, err := dtoken.Login(ctx, "context-user")
	if err != nil {
		t.Fatal(err)
	}
	if err := dtoken.AddRoles(ctx, "context-user", []string{"admin"}); err != nil {
		t.Fatal(err)
	}
	if err := dtoken.AddPermissions(ctx, "context-user", []string{"article:read"}); err != nil {
		t.Fatal(err)
	}
	for _, check := range []struct {
		name    string
		handler gin.HandlerFunc
	}{
		{"login", authMiddleware()},
		{"role", roleMiddleware("admin")},
		{"permission", permissionMiddleware("article:read")},
	} {
		for _, state := range []string{"active", "canceled", "deadline"} {
			t.Run(check.name+"/"+state, func(t *testing.T) {
				deadline := time.Now().Add(time.Minute)
				if state == "deadline" {
					deadline = time.Unix(0, 0)
				}
				requestCtx, cancel := context.WithDeadline(context.Background(), deadline)
				t.Cleanup(cancel)
				if state == "canceled" {
					cancel()
				}
				r := gin.New()
				called := false
				r.GET("/", check.handler, func(c *gin.Context) {
					called = true
					c.Status(http.StatusNoContent)
				})
				request := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(requestCtx)
				request.Header.Set(tokenHeader, token)
				rec := httptest.NewRecorder()
				r.ServeHTTP(rec, request)
				wantStatus := http.StatusInternalServerError
				if state == "active" {
					wantStatus = http.StatusNoContent
				}
				if called != (state == "active") || rec.Code != wantStatus {
					t.Fatalf("called=%v status=%d want=%d", called, rec.Code, wantStatus)
				}
			})
		}
	}
}

// TestAccessDenialStopsHandlers distinguishes authentication failure from missing access rights. TestAccessDenialStopsHandlers 区分登录失败和权限不足，并验证业务处理器不会执行。
func TestAccessDenialStopsHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupQuickStartManager(t)
	token, err := dtoken.Login(context.Background(), "unprivileged-user")
	if err != nil {
		t.Fatal(err)
	}
	for _, check := range []struct {
		name    string
		handler gin.HandlerFunc
	}{
		{"role", roleMiddleware("admin")},
		{"permission", permissionMiddleware("article:read")},
	} {
		for _, tc := range []struct {
			name   string
			token  string
			status int
		}{
			{"missing", "", http.StatusUnauthorized},
			{"invalid", "invalid-token", http.StatusUnauthorized},
			{"insufficient_access", token, http.StatusForbidden},
		} {
			t.Run(check.name+"/"+tc.name, func(t *testing.T) {
				r := gin.New()
				called := false
				r.GET("/", check.handler, func(*gin.Context) { called = true })
				rec := requestQuickStart(t, r, http.MethodGet, "/", "", tc.token)
				var response Response
				if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				if called || rec.Code != tc.status || response.Code != tc.status {
					t.Fatalf("called=%v status=%d response=%+v", called, rec.Code, response)
				}
			})
		}
	}
}

func setupQuickStartManager(t *testing.T) {
	t.Helper()
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)

	mgr, err := defaults.NewBuilder().
		TokenName(tokenHeader).
		AutoRenew(false).
		AsyncEvent(false).
		IsLog(false).
		IsPrintBanner(false).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	dtoken.SetManager(mgr)
}

func requestQuickStart(t *testing.T, router *gin.Engine, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set(tokenHeader, token)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}
