package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Zany2/dtoken-go/dtoken"
	echodt "github.com/Zany2/dtoken-go/integrations/echo"
	echo4 "github.com/labstack/echo/v4"
)

func TestLoginValidationAndAuthorizationSeed(t *testing.T) {
	setupEchoManager(t)

	e := echo4.New()
	e.POST("/login", handleLogin)

	missing := requestEcho(t, e, http.MethodPost, "/login", `{"username":"alice"}`, "")
	if missing.Code != http.StatusBadRequest {
		t.Fatalf("missing credentials status = %d, want %d", missing.Code, http.StatusBadRequest)
	}
	invalid := requestEcho(t, e, http.MethodPost, "/login", `{"username":"alice","password":"bad"}`, "")
	if invalid.Code != http.StatusUnauthorized {
		t.Fatalf("invalid password status = %d, want %d", invalid.Code, http.StatusUnauthorized)
	}

	success := requestEcho(t, e, http.MethodPost, "/login", `{"username":"alice","password":"123456"}`, "")
	if success.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d", success.Code, http.StatusOK)
	}
	var envelope Response
	if err := json.Unmarshal(success.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if envelope.Code != echodt.CodeSuccess || envelope.Message != "ok" {
		t.Fatalf("login response = %+v, want success response", envelope)
	}
	data, ok := envelope.Data.(map[string]interface{})
	if !ok || data["token"] == "" {
		t.Fatalf("login response data = %#v, want token", envelope.Data)
	}
	if !dtoken.HasRole(context.Background(), "alice", "admin") {
		t.Fatal("login did not seed admin role")
	}
	if !dtoken.HasPermission(context.Background(), "alice", "article:read") {
		t.Fatal("login did not seed article permission")
	}
}

func TestProtectedRoutesAndLogout(t *testing.T) {
	setupEchoManager(t)
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

	e := echo4.New()
	e.Use(echodt.RegisterDTokenContextMiddleware(ctx))
	e.POST("/login", handleLogin)
	auth := e.Group("")
	auth.Use(echodt.AuthMiddleware(ctx))
	auth.GET("/me", handleMe)
	auth.GET("/admin", handleAdmin, echodt.RoleMiddleware(ctx, []string{"admin"}))
	auth.GET("/articles", handleArticles, echodt.PermissionMiddleware(ctx, []string{"article:read"}))
	auth.POST("/logout", handleLogout)

	for _, route := range []string{"/me", "/admin", "/articles"} {
		response := requestEcho(t, e, http.MethodGet, route, "", token)
		if response.Code != http.StatusOK {
			t.Fatalf("authorized %s status = %d, want %d", route, response.Code, http.StatusOK)
		}
	}
	unauthorized := requestEcho(t, e, http.MethodGet, "/me", "", "")
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized /me status = %d, want %d", unauthorized.Code, http.StatusUnauthorized)
	}

	bobToken, err := dtoken.Login(ctx, "bob")
	if err != nil {
		t.Fatalf("Login(bob) error = %v", err)
	}
	for _, route := range []string{"/admin", "/articles"} {
		response := requestEcho(t, e, http.MethodGet, route, "", bobToken)
		if response.Code != http.StatusForbidden {
			t.Fatalf("unauthorized %s status = %d, want %d", route, response.Code, http.StatusForbidden)
		}
	}

	logout := requestEcho(t, e, http.MethodPost, "/logout", "", token)
	if logout.Code != http.StatusOK {
		t.Fatalf("logout status = %d, want %d", logout.Code, http.StatusOK)
	}
	afterLogout := requestEcho(t, e, http.MethodGet, "/me", "", token)
	if afterLogout.Code != http.StatusUnauthorized {
		t.Fatalf("request after logout status = %d, want %d", afterLogout.Code, http.StatusUnauthorized)
	}
}

func setupEchoManager(t *testing.T) {
	t.Helper()
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	initDToken()
}

func requestEcho(t *testing.T, e *echo4.Echo, method, path, body, token string) *httptest.ResponseRecorder {
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
	e.ServeHTTP(recorder, request)
	return recorder
}
