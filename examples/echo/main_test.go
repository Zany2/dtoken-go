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
	echodt "github.com/Zany2/dtoken-go/integrations/echo"
	echo4 "github.com/labstack/echo/v4"
)

// TestInitDTokenConfiguration verifies that the example can initialize with a valid renewal threshold. TestInitDTokenConfiguration 验证示例能使用有效的续期阈值完成初始化。
func TestInitDTokenConfiguration(t *testing.T) {
	setupEchoManager(t)
	mgr, err := echodt.GetManager()
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

	success := requestEcho(t, e, http.MethodPost, "/login", `{"username":"alice","password":"123456"}`+"\n\t", "")
	if success.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d", success.Code, http.StatusOK)
	}
	var envelope struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(success.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if envelope.Code != echodt.CodeSuccess || envelope.Message != "ok" {
		t.Fatalf("login response = %+v, want success response", envelope)
	}
	if envelope.Data.Token == "" {
		t.Fatalf("login response data = %#v, want token", envelope.Data)
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
	registerRoutes(e)

	for _, route := range []string{"/me", "/admin", "/articles"} {
		for _, header := range []string{token, "Bearer " + token} {
			response := requestEcho(t, e, http.MethodGet, route, "", header)
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
			unauthorized := requestEcho(t, e, method, route, "", header)
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
		response := requestEcho(t, e, http.MethodGet, route, "", bobToken)
		if response.Code != http.StatusForbidden {
			t.Fatalf("unauthorized %s status = %d, want %d", route, response.Code, http.StatusForbidden)
		}
	}

	logout := requestEcho(t, e, http.MethodPost, "/logout", "", "Bearer "+token)
	if logout.Code != http.StatusOK {
		t.Fatalf("logout status = %d, want %d", logout.Code, http.StatusOK)
	}
	afterLogout := requestEcho(t, e, http.MethodGet, "/me", "", token)
	if afterLogout.Code != http.StatusUnauthorized {
		t.Fatalf("request after logout status = %d, want %d", afterLogout.Code, http.StatusUnauthorized)
	}
}

// TestLoginRejectsInvalidJSON verifies malformed or multiple JSON values cannot create a login. TestLoginRejectsInvalidJSON 验证格式错误或多个 JSON 值不能创建登录态。
func TestLoginRejectsInvalidJSON(t *testing.T) {
	setupEchoManager(t)
	e := echo4.New()
	e.POST("/login", handleLogin)
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
			rec := requestEcho(t, e, http.MethodPost, "/login", tc.body, "")
			var response Response
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if rec.Code != http.StatusBadRequest || response.Code != echodt.CodeBadRequest || response.Data != nil {
				t.Fatalf("invalid JSON status=%d response=%+v", rec.Code, response)
			}
		})
	}
	if _, err := dtoken.GetSession(context.Background(), "alice"); !errors.Is(err, echodt.ErrSessionNotFound) {
		t.Fatalf("invalid JSON created a session: error = %v", err)
	}
}

// TestLoginBindingCompatibility preserves JSON content-type parameters and Echo's XML binding. TestLoginBindingCompatibility 保留带参数的 JSON 媒体类型及 Echo 的 XML 绑定行为。
func TestLoginBindingCompatibility(t *testing.T) {
	setupEchoManager(t)
	e := echo4.New()
	e.POST("/login", handleLogin)
	for _, tc := range []struct {
		contentType string
		body        string
	}{
		{"application/json; charset=UTF-8", `{"username":"alice","password":"123456"}`},
		{"application/xml", `<LoginRequest><Username>alice</Username><Password>123456</Password></LoginRequest>`},
	} {
		t.Run(tc.contentType, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", tc.contentType)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			var response struct {
				Data struct {
					Token string `json:"token"`
				} `json:"data"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			loginID, err := dtoken.GetLoginID(context.Background(), response.Data.Token)
			if rec.Code != http.StatusOK || err != nil || loginID != "alice" {
				t.Fatalf("login status=%d loginID=%q error=%v", rec.Code, loginID, err)
			}
		})
	}
}

// TestLoginRejectsDisabledAccount verifies an account restriction returns forbidden without issuing a token. TestLoginRejectsDisabledAccount 验证账号封禁返回禁止访问且不签发 Token。
func TestLoginRejectsDisabledAccount(t *testing.T) {
	setupEchoManager(t)
	if err := echodt.Disable(context.Background(), "alice", time.Hour, ""); err != nil {
		t.Fatal(err)
	}
	e := echo4.New()
	e.POST("/login", handleLogin)
	rec := requestEcho(t, e, http.MethodPost, "/login", `{"username":"alice","password":"123456"}`, "")
	var response Response
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusForbidden || response.Code != echodt.CodeAccountDisabled || response.Data != nil {
		t.Fatalf("disabled login status=%d response=%+v", rec.Code, response)
	}
}

// TestHandlersReportStorageFailures verifies backend errors remain server errors and expose no internal details. TestHandlersReportStorageFailures 验证后端故障返回服务端错误且不暴露内部详情。
func TestHandlersReportStorageFailures(t *testing.T) {
	echodt.DeleteAllManager()
	t.Cleanup(echodt.DeleteAllManager)
	mgr, err := echodt.NewBuilder().
		SetStorage(&echoFailingStorage{Storage: echodt.NewMemoryStorage()}).
		AutoRenew(false).AsyncEvent(false).IsPrintBanner(false).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	echodt.SetManager(mgr)

	// Register only request context to exercise each handler's own failure path. 仅注册请求上下文，以覆盖各处理器自身的错误分支。
	e := echo4.New()
	e.Use(echodt.RegisterDTokenContextMiddleware(context.Background()))
	e.POST("/login", handleLogin)
	e.GET("/me", handleMe)
	e.POST("/logout", handleLogout)
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
			rec := requestEcho(t, e, tc.method, tc.path, tc.body, "Bearer test-token")
			var response Response
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if rec.Code != http.StatusInternalServerError || response.Code != echodt.CodeServerError ||
				response.Message != "internal server error" || response.Data != nil {
				t.Fatalf("storage failure status=%d response=%+v", rec.Code, response)
			}
		})
	}
}

// echoFailingStorage keeps initialization valid but fails every authentication read. echoFailingStorage 允许正常初始化，但使每次鉴权读取失败。
type echoFailingStorage struct {
	echodt.Storage
}

func (s *echoFailingStorage) Get(context.Context, string) (any, error) {
	return nil, errors.New("private storage connection details")
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
		request.Header.Set("Authorization", token)
	}
	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, request)
	return recorder
}
