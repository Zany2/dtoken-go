package main

import (
	dtoken "github.com/Zany2/dtoken-go"
	dt1 "github.com/Zany2/dtoken-go/contract"
	dterror "github.com/Zany2/dtoken-go/errors"
	dt2 "github.com/Zany2/dtoken-go/event"
	dt3 "github.com/Zany2/dtoken-go/storage/memory"

	"bytes"
	"context"
	"encoding/json"
	"errors"
	dtglobal "github.com/Zany2/dtoken-go/global"
	fiberdt "github.com/Zany2/dtoken-go/integrations/fiber"
	gofiber "github.com/gofiber/fiber/v2"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestInitDTokenConfiguration verifies that the example can initialize with a valid renewal threshold. TestInitDTokenConfiguration 验证示例能使用有效的续期阈值完成初始化。
func TestInitDTokenConfiguration(t *testing.T) {
	setupFiberManager(t)
	mgr, err := dtglobal.GetManager()
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
	if envelope.Code != dterror.CodeSuccess || envelope.Data.Token == "" {
		t.Fatalf("login envelope = %+v, want success and token", envelope)
	}
	if loginID, err := dtglobal.GetLoginID(context.Background(), envelope.Data.Token); err != nil || loginID != "alice" {
		t.Fatalf("returned token login ID = %q, error = %v, want alice", loginID, err)
	}
	if !dtglobal.HasRole(context.Background(), "alice", "admin") {
		t.Fatal("login did not seed admin role")
	}
	if !dtglobal.HasPermission(context.Background(), "alice", "article:read") {
		t.Fatal("login did not seed article permission")
	}
}

func TestProtectedRoutesAndLogout(t *testing.T) {
	setupFiberManager(t)
	ctx := context.Background()
	token, err := dtglobal.Login(ctx, "alice")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err = dtglobal.AddRoles(ctx, "alice", []string{"admin"}); err != nil {
		t.Fatalf("AddRoles() error = %v", err)
	}
	if err = dtglobal.AddPermissions(ctx, "alice", []string{"article:read"}); err != nil {
		t.Fatalf("AddPermissions() error = %v", err)
	}

	app := gofiber.New()
	registerRoutes(app)

	for _, route := range []string{"/me", "/admin", "/articles"} {
		for _, header := range []string{token, "Bearer " + token} {
			response := requestFiber(t, app, http.MethodGet, route, "", header)
			if response.StatusCode != http.StatusOK {
				t.Fatalf("authorized %s status = %d, want %d", route, response.StatusCode, http.StatusOK)
			}
		}
	}
	for _, route := range []string{"/me", "/admin", "/articles", "/logout"} {
		method := http.MethodGet
		if route == "/logout" {
			method = http.MethodPost
		}
		for _, header := range []string{"", "Bearer invalid-token"} {
			unauthorized := requestFiber(t, app, method, route, "", header)
			if unauthorized.StatusCode != http.StatusUnauthorized {
				t.Fatalf("unauthorized %s status = %d, want %d", route, unauthorized.StatusCode, http.StatusUnauthorized)
			}
		}
	}

	bobToken, err := dtglobal.Login(ctx, "bob")
	if err != nil {
		t.Fatalf("Login(bob) error = %v", err)
	}
	for _, route := range []string{"/admin", "/articles"} {
		response := requestFiber(t, app, http.MethodGet, route, "", bobToken)
		if response.StatusCode != http.StatusForbidden {
			t.Fatalf("unauthorized %s status = %d, want %d", route, response.StatusCode, http.StatusForbidden)
		}
	}

	logout := requestFiber(t, app, http.MethodPost, "/logout", "", "Bearer "+token)
	if logout.StatusCode != http.StatusOK {
		t.Fatalf("logout status = %d, want %d", logout.StatusCode, http.StatusOK)
	}
	afterLogout := requestFiber(t, app, http.MethodGet, "/me", "", token)
	if afterLogout.StatusCode != http.StatusUnauthorized {
		t.Fatalf("request after logout status = %d, want %d", afterLogout.StatusCode, http.StatusUnauthorized)
	}
}

// TestLoginPayloads verifies supported JSON/form inputs and rejection before creating login state. TestLoginPayloads 验证 JSON/表单输入及创建登录态前的参数拒绝。
func TestLoginPayloads(t *testing.T) {
	valid := `{"username":"alice","password":"123456"}`
	for _, tc := range []struct {
		name        string
		contentType string
		body        string
		status      int
	}{
		{"json_charset", "application/json; charset=UTF-8", valid + "\n\t", http.StatusOK},
		{"form", "application/x-www-form-urlencoded", "username=alice&password=123456", http.StatusOK},
		{"missing_form_password", "application/x-www-form-urlencoded", "username=alice", http.StatusBadRequest},
		{"empty", "application/json", "", http.StatusBadRequest},
		{"null", "application/json", "null", http.StatusBadRequest},
		{"wrong_type", "application/json", `{"username":123,"password":"123456"}`, http.StatusBadRequest},
		{"second_object", "application/json", valid + `{}`, http.StatusBadRequest},
		{"trailing_garbage", "application/json", valid + `invalid`, http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupFiberManager(t)
			app := gofiber.New()
			app.Post("/login", handleLogin)
			req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", tc.contentType)
			response, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			if response.StatusCode != tc.status {
				t.Fatalf("login status=%d, want %d", response.StatusCode, tc.status)
			}
			if tc.status == http.StatusBadRequest {
				if _, err := dtglobal.GetSession(context.Background(), "alice"); !errors.Is(err, dterror.ErrSessionNotFound) {
					t.Fatalf("invalid payload created a session: error = %v", err)
				}
				return
			}
			var envelope struct {
				Data struct {
					Token string `json:"token"`
				} `json:"data"`
			}
			if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
				t.Fatal(err)
			}
			if loginID, err := dtglobal.GetLoginID(context.Background(), envelope.Data.Token); err != nil || loginID != "alice" {
				t.Fatalf("login ID=%q error=%v, want alice", loginID, err)
			}
		})
	}
}

// TestFormLoginOwnsUsername verifies retained event data survives reuse of Fiber's form buffer. TestFormLoginOwnsUsername 验证保留的事件数据不受 Fiber 表单缓冲区复用影响。
func TestFormLoginOwnsUsername(t *testing.T) {
	dtglobal.DeleteAllManager()
	t.Cleanup(dtglobal.DeleteAllManager)
	builtAuth1, err := dtoken.NewBuilder().AutoRenew(false).AsyncEvent(false).IsPrintBanner(false).Build()
	mgr := builtAuth1.Manager()
	if err != nil {
		t.Fatal(err)
	}
	dtglobal.SetManager(mgr)
	var eventLoginID string
	mgr.GetEventManager().RegisterFuncWithConfig(dt2.EventLogin, func(data *dt2.EventData) {
		eventLoginID = data.LoginID
	}, dt2.ListenerConfig{Async: false})

	app := gofiber.New()
	mutated := 0
	app.Post("/login", func(c *gofiber.Ctx) error {
		err := handleLogin(c)

		// Overwrite the backing bytes deterministically after login completes. 登录完成后确定性地覆盖底层字节，模拟缓冲区复用。
		mutated = copy(c.Context().PostArgs().Peek("username"), []byte("other"))
		return err
	})
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString("username=alice&password=123456"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || mutated != len("other") || eventLoginID != "alice" {
		t.Fatalf("status=%d mutated=%d retained login ID=%q, want alice", response.StatusCode, mutated, eventLoginID)
	}
}

// TestLoginRejectsDisabledAccount verifies an account restriction returns forbidden without issuing a token. TestLoginRejectsDisabledAccount 验证账号封禁返回禁止访问且不签发 Token。
func TestLoginRejectsDisabledAccount(t *testing.T) {
	setupFiberManager(t)
	if err := dtglobal.Disable(context.Background(), "alice", time.Hour, ""); err != nil {
		t.Fatal(err)
	}
	app := gofiber.New()
	app.Post("/login", handleLogin)
	response := requestFiber(t, app, http.MethodPost, "/login", `{"username":"alice","password":"123456"}`, "")
	var envelope Response
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusForbidden || envelope.Code != dterror.CodeAccountDisabled || envelope.Data != nil {
		t.Fatalf("disabled login status=%d response=%+v", response.StatusCode, envelope)
	}
}

// TestHandlersReportStorageFailures verifies backend errors remain server errors and expose no internal details. TestHandlersReportStorageFailures 验证后端故障返回服务端错误且不暴露内部详情。
func TestHandlersReportStorageFailures(t *testing.T) {
	dtglobal.DeleteAllManager()
	t.Cleanup(dtglobal.DeleteAllManager)
	builtAuth2, err := dtoken.NewBuilder().
		Storage(&fiberFailingStorage{Storage: dt3.NewStorage()}).
		AutoRenew(false).AsyncEvent(false).IsPrintBanner(false).
		Build()
	mgr := builtAuth2.Manager()

	if err != nil {
		t.Fatal(err)
	}
	dtglobal.SetManager(mgr)

	// Register only request context to exercise each handler's own failure path. 仅注册请求上下文，以覆盖各处理器自身的错误分支。
	app := gofiber.New()
	app.Use(fiberdt.RegisterDTokenContextMiddleware(context.Background(), fiberdt.WithAuth(builtAuth2)))
	app.Post("/login", handleLogin)
	app.Get("/me", handleMe)
	app.Post("/logout", handleLogout)
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
			response := requestFiber(t, app, tc.method, tc.path, tc.body, "Bearer test-token")
			var envelope Response
			if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != http.StatusInternalServerError || envelope.Code != dterror.CodeServerError ||
				envelope.Message != "internal server error" || envelope.Data != nil {
				t.Fatalf("storage failure status=%d response=%+v", response.StatusCode, envelope)
			}
		})
	}
}

// fiberFailingStorage keeps initialization valid but fails every authentication read. fiberFailingStorage 允许正常初始化，但使每次鉴权读取失败。
type fiberFailingStorage struct {
	dt1.Storage
}

func (s *fiberFailingStorage) Get(context.Context, string) (any, error) {
	return nil, errors.New("private storage connection details")
}

func setupFiberManager(t *testing.T) {
	t.Helper()
	dtglobal.DeleteAllManager()
	t.Cleanup(dtglobal.DeleteAllManager)
	initDToken()
}

func requestFiber(t *testing.T, app *gofiber.App, method, path, body, token string) *http.Response {
	t.Helper()
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", token)
	}
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	return response
}
