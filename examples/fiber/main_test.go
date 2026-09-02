package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Zany2/dtoken-go/dtoken"
	fiberdt "github.com/Zany2/dtoken-go/integrations/fiber"
	gofiber "github.com/gofiber/fiber/v2"
)

func TestLoginValidationAndAuthorizationSeed(t *testing.T) {
	setupFiberManager(t)
	app := gofiber.New()
	app.Post("/login", handleLogin)

	missing := requestFiber(t, app, http.MethodPost, "/login", `{"username":"alice"}`, "")
	if missing.StatusCode != http.StatusBadRequest {
		t.Fatalf("missing credentials status = %d, want %d", missing.StatusCode, http.StatusBadRequest)
	}
	invalid := requestFiber(t, app, http.MethodPost, "/login", `{"username":"alice","password":"bad"}`, "")
	if invalid.StatusCode != http.StatusUnauthorized {
		t.Fatalf("invalid password status = %d, want %d", invalid.StatusCode, http.StatusUnauthorized)
	}

	login := requestFiber(t, app, http.MethodPost, "/login", `{"username":"alice","password":"123456"}`, "")
	if login.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, want %d", login.StatusCode, http.StatusOK)
	}
	var envelope struct {
		Code int `json:"code"`
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(login.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if envelope.Code != fiberdt.CodeSuccess || envelope.Data.Token == "" {
		t.Fatalf("login envelope = %+v, want success and token", envelope)
	}
	if !dtoken.HasRole(context.Background(), "alice", "admin") {
		t.Fatal("login did not seed admin role")
	}
	if !dtoken.HasPermission(context.Background(), "alice", "article:read") {
		t.Fatal("login did not seed article permission")
	}
	login.Body.Close()
}

func TestProtectedRoutesAndLogout(t *testing.T) {
	setupFiberManager(t)
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

	app := gofiber.New()
	app.Use(fiberdt.RegisterDTokenContextMiddleware(ctx))
	auth := app.Group("")
	auth.Use(fiberdt.AuthMiddleware(ctx))
	auth.Get("/me", handleMe)
	auth.Get("/admin", fiberdt.RoleMiddleware(ctx, []string{"admin"}), handleAdmin)
	auth.Get("/articles", fiberdt.PermissionMiddleware(ctx, []string{"article:read"}), handleArticles)
	auth.Post("/logout", handleLogout)

	for _, route := range []string{"/me", "/admin", "/articles"} {
		response := requestFiber(t, app, http.MethodGet, route, "", token)
		if response.StatusCode != http.StatusOK {
			t.Fatalf("authorized %s status = %d, want %d", route, response.StatusCode, http.StatusOK)
		}
		response.Body.Close()
	}
	unauthorized := requestFiber(t, app, http.MethodGet, "/me", "", "")
	if unauthorized.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthorized /me status = %d, want %d", unauthorized.StatusCode, http.StatusUnauthorized)
	}
	unauthorized.Body.Close()

	bobToken, err := dtoken.Login(ctx, "bob")
	if err != nil {
		t.Fatalf("Login(bob) error = %v", err)
	}
	for _, route := range []string{"/admin", "/articles"} {
		response := requestFiber(t, app, http.MethodGet, route, "", bobToken)
		if response.StatusCode != http.StatusForbidden {
			t.Fatalf("unauthorized %s status = %d, want %d", route, response.StatusCode, http.StatusForbidden)
		}
		response.Body.Close()
	}

	logout := requestFiber(t, app, http.MethodPost, "/logout", "", token)
	if logout.StatusCode != http.StatusOK {
		t.Fatalf("logout status = %d, want %d", logout.StatusCode, http.StatusOK)
	}
	logout.Body.Close()
	afterLogout := requestFiber(t, app, http.MethodGet, "/me", "", token)
	if afterLogout.StatusCode != http.StatusUnauthorized {
		t.Fatalf("request after logout status = %d, want %d", afterLogout.StatusCode, http.StatusUnauthorized)
	}
	afterLogout.Body.Close()
}

func setupFiberManager(t *testing.T) {
	t.Helper()
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	initDToken()
}

func requestFiber(t *testing.T, app *gofiber.App, method, path, body, token string) *http.Response {
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
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	return response
}
