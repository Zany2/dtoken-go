package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
	auth := r.Group("/")
	auth.Use(authMiddleware(ctx))
	auth.GET("/me", handleMe)
	auth.GET("/admin", roleMiddleware(ctx, "admin"), handleAdmin)
	auth.GET("/articles", permissionMiddleware(ctx, "article:read"), handleArticles)
	auth.POST("/logout", handleLogout)

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

func setupQuickStartManager(t *testing.T) {
	t.Helper()
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)

	mgr, err := defaults.NewBuilder().
		TokenName(tokenHeader).
		AutoRenew(false).
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
