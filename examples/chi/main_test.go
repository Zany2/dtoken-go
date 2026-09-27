package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/dtoken"
	chidt "github.com/Zany2/dtoken-go/integrations/chi"
	"github.com/go-chi/chi/v5"
)

// TestInitDTokenConfiguration verifies that the example can initialize with a valid renewal threshold. TestInitDTokenConfiguration 验证示例能使用有效的续期阈值完成初始化。
func TestInitDTokenConfiguration(t *testing.T) {
	setupChiManager(t)
	mgr, err := chidt.GetManager()
	if err != nil {
		t.Fatal(err)
	}
	cfg := mgr.GetConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("example configuration is invalid: %v", err)
	}
	if !cfg.AutoRenew || cfg.Timeout != int64((2*time.Hour).Seconds()) ||
		cfg.RenewMaxRefresh <= 0 || cfg.RenewMaxRefresh >= cfg.Timeout {
		t.Fatalf("unexpected renewal configuration: %+v", cfg)
	}
}

// TestLoginValidationAndAuthorizationSeed verifies login validation and demo authorization data. TestLoginValidationAndAuthorizationSeed 验证登录校验与示例授权数据初始化。
func TestLoginValidationAndAuthorizationSeed(t *testing.T) {
	setupChiManager(t)
	router := chi.NewRouter()
	router.Post("/login", handleLogin)

	missing := requestChi(t, router, http.MethodPost, "/login", `{"username":"alice"}`, "")
	if missing.Code != http.StatusBadRequest {
		t.Fatalf("missing credentials status = %d, want %d", missing.Code, http.StatusBadRequest)
	}
	invalid := requestChi(t, router, http.MethodPost, "/login", `{"username":"alice","password":"bad"}`, "")
	if invalid.Code != http.StatusUnauthorized {
		t.Fatalf("invalid password status = %d, want %d", invalid.Code, http.StatusUnauthorized)
	}

	login := requestChi(t, router, http.MethodPost, "/login", `{"username":"alice","password":"123456"}`+"\n\t", "")
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
	if envelope.Code != chidt.CodeSuccess || envelope.Data.Token == "" {
		t.Fatalf("login envelope = %+v, want success and token", envelope)
	}
	if loginID, err := dtoken.GetLoginID(context.Background(), envelope.Data.Token); err != nil || loginID != "alice" {
		t.Fatalf("returned token login ID = %q, error = %v, want alice", loginID, err)
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
	setupChiManager(t)
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

	router := chi.NewRouter()
	registerRoutes(router)

	for _, route := range []string{"/me", "/admin", "/articles"} {
		for _, header := range []string{token, "Bearer " + token} {
			response := requestChi(t, router, http.MethodGet, route, "", header)
			if response.Code != http.StatusOK {
				t.Fatalf("authorized %s status = %d, want %d", route, response.Code, http.StatusOK)
			}
		}
	}
	for _, route := range []string{"/me", "/admin", "/articles", "/logout"} {
		method := http.MethodGet
		if route == "/logout" {
			method = http.MethodPost
		}
		for _, header := range []string{"", "Bearer invalid-token"} {
			unauthorized := requestChi(t, router, method, route, "", header)
			if unauthorized.Code != http.StatusUnauthorized {
				t.Fatalf("unauthorized %s status = %d, want %d", route, unauthorized.Code, http.StatusUnauthorized)
			}
		}
	}

	bobToken, err := dtoken.Login(ctx, "bob")
	if err != nil {
		t.Fatalf("Login(bob) error = %v", err)
	}
	for _, route := range []string{"/admin", "/articles"} {
		response := requestChi(t, router, http.MethodGet, route, "", bobToken)
		if response.Code != http.StatusForbidden {
			t.Fatalf("unauthorized %s status = %d, want %d", route, response.Code, http.StatusForbidden)
		}
	}

	logout := requestChi(t, router, http.MethodPost, "/logout", "", "Bearer "+token)
	if logout.Code != http.StatusOK {
		t.Fatalf("logout status = %d, want %d", logout.Code, http.StatusOK)
	}
	afterLogout := requestChi(t, router, http.MethodGet, "/me", "", token)
	if afterLogout.Code != http.StatusUnauthorized {
		t.Fatalf("request after logout status = %d, want %d", afterLogout.Code, http.StatusUnauthorized)
	}
}

// TestLoginRejectsInvalidJSON verifies malformed or multiple JSON values cannot create a login. TestLoginRejectsInvalidJSON 验证格式错误或多个 JSON 值不能创建登录态。
func TestLoginRejectsInvalidJSON(t *testing.T) {
	setupChiManager(t)
	router := chi.NewRouter()
	router.Post("/login", handleLogin)
	valid := `{"username":"alice","password":"123456"}`
	for _, tc := range []struct {
		name string
		body string
	}{
		{"empty", ""},
		{"null", "null"},
		{"array", "[]"},
		{"wrong_type", `{"username":123,"password":"123456"}`},
		{"malformed", `{"username":"alice",`},
		{"second_object", valid + `{}`},
		{"trailing_null", valid + ` null`},
		{"trailing_garbage", valid + `invalid`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := requestChi(t, router, http.MethodPost, "/login", tc.body, "")
			var response Response
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if rec.Code != http.StatusBadRequest || response.Code != chidt.CodeBadRequest || response.Data != nil {
				t.Fatalf("invalid JSON status=%d response=%+v", rec.Code, response)
			}
		})
	}
	if _, err := dtoken.GetSession(context.Background(), "alice"); !errors.Is(err, chidt.ErrSessionNotFound) {
		t.Fatalf("invalid JSON created a session: error = %v", err)
	}
}

// TestLoginRejectsDisabledAccount verifies an account restriction returns forbidden without issuing a token. TestLoginRejectsDisabledAccount 验证账号封禁返回禁止访问且不签发 Token。
func TestLoginRejectsDisabledAccount(t *testing.T) {
	setupChiManager(t)
	if err := chidt.Disable(context.Background(), "alice", time.Hour); err != nil {
		t.Fatal(err)
	}
	router := chi.NewRouter()
	router.Post("/login", handleLogin)
	rec := requestChi(t, router, http.MethodPost, "/login", `{"username":"alice","password":"123456"}`, "")
	var response Response
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusForbidden || response.Code != chidt.CodeAccountDisabled || response.Data != nil {
		t.Fatalf("disabled login status=%d response=%+v", rec.Code, response)
	}
}

// TestHandlersReportStorageFailures verifies backend errors remain server errors and expose no internal details. TestHandlersReportStorageFailures 验证后端故障返回服务端错误且不暴露内部详情。
func TestHandlersReportStorageFailures(t *testing.T) {
	chidt.DeleteAllManager()
	t.Cleanup(chidt.DeleteAllManager)
	mgr, err := chidt.NewBuilder().
		SetStorage(&chiFailingStorage{Storage: chidt.NewMemoryStorage()}).
		AutoRenew(false).AsyncEvent(false).IsPrintBanner(false).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	chidt.SetManager(mgr)

	// Register only request context to exercise each handler's own failure path. 仅注册请求上下文，以覆盖各处理器自身的错误分支。
	router := chi.NewRouter()
	router.Use(chidt.RegisterDTokenContextMiddleware())
	router.Post("/login", handleLogin)
	router.Get("/me", handleMe)
	router.Post("/logout", handleLogout)
	for _, tc := range []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/login", `{"username":"alice","password":"123456"}`},
		{http.MethodGet, "/me", ""},
		{http.MethodPost, "/logout", ""},
	} {
		t.Run(tc.path, func(t *testing.T) {
			rec := requestChi(t, router, tc.method, tc.path, tc.body, "Bearer test-token")
			var response Response
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if rec.Code != http.StatusInternalServerError || response.Code != chidt.CodeServerError ||
				response.Message != "internal server error" || response.Data != nil {
				t.Fatalf("storage failure status=%d response=%+v", rec.Code, response)
			}
		})
	}
}

// chiFailingStorage keeps initialization valid but fails every authentication read. chiFailingStorage 允许正常初始化，但使每次鉴权读取失败。
type chiFailingStorage struct {
	chidt.Storage
}

func (s *chiFailingStorage) Get(context.Context, string) (any, error) {
	return nil, errors.New("private storage connection details")
}

func setupChiManager(t *testing.T) {
	t.Helper()
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	initDToken()
}

func requestChi(t *testing.T, router http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}
