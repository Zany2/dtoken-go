package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Zany2/dtoken-go/dtoken"
	gindt "github.com/Zany2/dtoken-go/integrations/gin"
	"github.com/gin-gonic/gin"
)

func TestLoginAndRefreshValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupGinManager(t)

	r := gin.New()
	r.POST("/login", handleLogin)
	r.POST("/refresh", handleRefresh)

	missing := requestGin(t, r, http.MethodPost, "/login", `{"username":"alice"}`, "")
	if missing.Code != http.StatusBadRequest {
		t.Fatalf("missing credentials status = %d, want %d", missing.Code, http.StatusBadRequest)
	}
	invalid := requestGin(t, r, http.MethodPost, "/login", `{"username":"alice","password":"bad"}`, "")
	if invalid.Code != http.StatusUnauthorized {
		t.Fatalf("invalid password status = %d, want %d", invalid.Code, http.StatusUnauthorized)
	}

	login := requestGin(t, r, http.MethodPost, "/login", `{"username":"alice","password":"123456"}`, "")
	if login.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d", login.Code, http.StatusOK)
	}
	var envelope struct {
		Code int `json:"code"`
		Data struct {
			AccessToken  string `json:"accessToken"`
			RefreshToken string `json:"refreshToken"`
		} `json:"data"`
	}
	if err := json.Unmarshal(login.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if envelope.Code != gindt.CodeSuccess || envelope.Data.AccessToken == "" || envelope.Data.RefreshToken == "" {
		t.Fatalf("login envelope = %+v, want success and both tokens", envelope)
	}
	if !dtoken.HasRole(context.Background(), "alice", "admin") {
		t.Fatal("login did not seed admin role")
	}
	if !dtoken.HasPermission(context.Background(), "alice", "article:read") {
		t.Fatal("login did not seed article permission")
	}

	badRefresh := requestGin(t, r, http.MethodPost, "/refresh", `{}`, "")
	if badRefresh.Code != http.StatusBadRequest {
		t.Fatalf("missing refresh token status = %d, want %d", badRefresh.Code, http.StatusBadRequest)
	}
	refresh := requestGin(t, r, http.MethodPost, "/refresh", `{"refreshToken":"`+envelope.Data.RefreshToken+`"}`, "")
	if refresh.Code != http.StatusOK {
		t.Fatalf("refresh status = %d, want %d", refresh.Code, http.StatusOK)
	}
}

func TestProtectedRoutesIntrospectionAndLogout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupGinManager(t)
	ctx := context.Background()
	mgr, err := dtoken.GetManager()
	if err != nil {
		t.Fatalf("GetManager() error = %v", err)
	}
	pair, err := dtoken.LoginWithRefreshToken(ctx, "alice", "web", "browser")
	if err != nil {
		t.Fatalf("LoginWithRefreshToken() error = %v", err)
	}
	if err = dtoken.AddRoles(ctx, "alice", []string{"admin"}); err != nil {
		t.Fatalf("AddRoles() error = %v", err)
	}
	if err = dtoken.AddPermissions(ctx, "alice", []string{"article:read"}); err != nil {
		t.Fatalf("AddPermissions() error = %v", err)
	}
	if mgr.GetConfig().TokenName == "" {
		t.Fatal("configured token name is empty")
	}

	r := gin.New()
	r.Use(gindt.RegisterDTokenContextMiddleware(ctx))
	auth := r.Group("/")
	auth.Use(gindt.AuthMiddleware(ctx))
	auth.GET("/me", handleMe)
	auth.GET("/introspect", handleIntrospect)
	auth.GET("/admin", gindt.RoleMiddleware(ctx, []string{"admin"}), handleAdmin)
	auth.GET("/articles", gindt.PermissionMiddleware(ctx, []string{"article:read"}), handleArticles)
	auth.POST("/logout", handleLogout)

	for _, route := range []string{"/me", "/introspect", "/admin", "/articles"} {
		response := requestGin(t, r, http.MethodGet, route, "", pair.AccessToken)
		if response.Code != http.StatusOK {
			t.Fatalf("authorized %s status = %d, want %d", route, response.Code, http.StatusOK)
		}
	}
	unauthorized := requestGin(t, r, http.MethodGet, "/me", "", "")
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized /me status = %d, want %d", unauthorized.Code, http.StatusUnauthorized)
	}

	bobToken, err := dtoken.Login(ctx, "bob")
	if err != nil {
		t.Fatalf("Login(bob) error = %v", err)
	}
	for _, route := range []string{"/admin", "/articles"} {
		response := requestGin(t, r, http.MethodGet, route, "", bobToken)
		if response.Code != http.StatusForbidden {
			t.Fatalf("unauthorized %s status = %d, want %d", route, response.Code, http.StatusForbidden)
		}
	}

	logout := requestGin(t, r, http.MethodPost, "/logout", "", pair.AccessToken)
	if logout.Code != http.StatusOK {
		t.Fatalf("logout status = %d, want %d", logout.Code, http.StatusOK)
	}
	afterLogout := requestGin(t, r, http.MethodGet, "/me", "", pair.AccessToken)
	if afterLogout.Code != http.StatusUnauthorized {
		t.Fatalf("request after logout status = %d, want %d", afterLogout.Code, http.StatusUnauthorized)
	}
}

func setupGinManager(t *testing.T) {
	t.Helper()
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	initDToken()
}

func requestGin(t *testing.T, router *gin.Engine, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		mgr, err := dtoken.GetManager()
		if err != nil {
			t.Fatalf("GetManager() error = %v", err)
		}
		request.Header.Set(mgr.GetConfig().TokenName, token)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}
