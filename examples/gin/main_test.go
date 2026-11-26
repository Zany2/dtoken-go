package main

import (
	dtoken "github.com/Zany2/dtoken-go"
	dt2 "github.com/Zany2/dtoken-go/contract"
	dterror "github.com/Zany2/dtoken-go/errors"
	dt1 "github.com/Zany2/dtoken-go/internal/engine"
	dt3 "github.com/Zany2/dtoken-go/storage/memory"

	"bytes"
	"context"
	"encoding/json"
	"errors"
	gindt "github.com/Zany2/dtoken-go/integrations/gin"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestInitDTokenConfiguration(t *testing.T) {
	setupGinManager(t)
	mgr, err := dtoken.GetManager()
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

func TestLoginAndRefreshValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupGinManager(t)

	r := gin.New()
	r.Use(gindt.RegisterDTokenContextMiddleware(context.Background(), gindt.WithManagerResolver(dtoken.GetManager)))
	r.POST("/login", handleLogin)
	r.POST("/refresh", handleRefresh)
	auth := r.Group("/", gindt.AuthMiddleware(context.Background()))
	auth.GET("/me", handleMe)
	auth.POST("/logout", handleLogout)

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
		Code int                  `json:"code"`
		Data dt1.RefreshTokenPair `json:"data"`
	}
	if err := json.Unmarshal(login.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if envelope.Code != dterror.CodeSuccess || envelope.Data.AccessToken == "" || envelope.Data.RefreshToken == "" {
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
	var rotated struct {
		Code int                  `json:"code"`
		Data dt1.RefreshTokenPair `json:"data"`
	}
	if err := json.Unmarshal(refresh.Body.Bytes(), &rotated); err != nil {
		t.Fatalf("decode refresh response: %v", err)
	}
	if rotated.Code != dterror.CodeSuccess || rotated.Data.AccessToken == "" || rotated.Data.RefreshToken == "" ||
		rotated.Data.AccessToken == envelope.Data.AccessToken || rotated.Data.RefreshToken == envelope.Data.RefreshToken {
		t.Fatal("refresh must return a new access token and refresh token")
	}

	// Exercise the documented Bearer flow and reject both old credentials after rotation. 验证文档中的 Bearer 流程，并拒绝轮换后的两个旧凭证。
	oldAccess := requestGin(t, r, http.MethodGet, "/me", "", "Bearer "+envelope.Data.AccessToken)
	if oldAccess.Code != http.StatusUnauthorized {
		t.Fatalf("old access token status = %d, want %d", oldAccess.Code, http.StatusUnauthorized)
	}
	replay := requestGin(t, r, http.MethodPost, "/refresh", `{"refreshToken":"`+envelope.Data.RefreshToken+`"}`, "")
	if replay.Code != http.StatusUnauthorized {
		t.Fatalf("replayed refresh token status = %d, want %d", replay.Code, http.StatusUnauthorized)
	}
	newAccess := requestGin(t, r, http.MethodGet, "/me", "", "Bearer "+rotated.Data.AccessToken)
	if newAccess.Code != http.StatusOK {
		t.Fatalf("new access token status = %d, want %d", newAccess.Code, http.StatusOK)
	}
	var profile struct {
		Data struct {
			LoginID string `json:"loginId"`
		} `json:"data"`
	}
	if err := json.Unmarshal(newAccess.Body.Bytes(), &profile); err != nil {
		t.Fatal(err)
	}
	if profile.Data.LoginID != "alice" {
		t.Fatalf("refreshed login ID = %q, want alice", profile.Data.LoginID)
	}

	logout := requestGin(t, r, http.MethodPost, "/logout", "", "Bearer "+rotated.Data.AccessToken)
	if logout.Code != http.StatusOK {
		t.Fatalf("logout status = %d, want %d", logout.Code, http.StatusOK)
	}
	afterLogout := requestGin(t, r, http.MethodPost, "/refresh", `{"refreshToken":"`+rotated.Data.RefreshToken+`"}`, "")
	if afterLogout.Code != http.StatusUnauthorized {
		t.Fatalf("refresh after logout status = %d, want %d", afterLogout.Code, http.StatusUnauthorized)
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
	pair, err := dtoken.LoginWithRefreshTokenOptions(ctx, dtoken.RefreshTokenOptions{LoginOptions: dtoken.LoginOptions{LoginID: "alice", Device: "web", DeviceID: "browser"}})
	if err != nil {
		t.Fatalf("LoginWithRefreshToken() error = %v", err)
	}
	if err = dtoken.AddRoles(ctx, dtoken.RoleOptions{LoginID: "alice", Roles: []string{"admin"}}); err != nil {
		t.Fatalf("AddRoles() error = %v", err)
	}
	if err = dtoken.AddPermissions(ctx, dtoken.PermissionOptions{LoginID: "alice", Permissions: []string{"article:read"}}); err != nil {
		t.Fatalf("AddPermissions() error = %v", err)
	}
	if mgr.GetConfig().TokenName == "" {
		t.Fatal("configured token name is empty")
	}

	r := gin.New()
	registerTestRoutes(t, r)

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

	bobToken, err := dtoken.Login(ctx, dtoken.LoginOptions{LoginID: "bob"})
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

func TestLoginRejectsDisabledAccountsAndDevices(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, device := range []bool{false, true} {
		name := "account"
		if device {
			name = "device"
		}
		t.Run(name, func(t *testing.T) {
			setupGinManager(t)
			var err error
			if device {
				err = dtoken.DisableDevice(context.Background(), dtoken.DeviceDisableOptions{LoginID: "alice", Device: "web", Duration: time.Hour})
			} else {
				err = dtoken.Disable(context.Background(), dtoken.DisableOptions{LoginID: "alice", Duration: time.Hour, Reason: ""})
			}
			if err != nil {
				t.Fatal(err)
			}
			r := gin.New()
			r.Use(gindt.RegisterDTokenContextMiddleware(context.Background(), gindt.WithManagerResolver(dtoken.GetManager)))
			r.POST("/login", handleLogin)
			rec := requestGin(t, r, http.MethodPost, "/login", `{"username":"alice","password":"123456"}`, "")
			var response Response
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if rec.Code != http.StatusForbidden || response.Code != dterror.CodeAccountDisabled || response.Data != nil {
				t.Fatalf("disabled login status=%d response=%+v", rec.Code, response)
			}
		})
	}
}

func TestRefreshRejectsDisabledDeviceWithoutConsumingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupGinManager(t)
	ctx := context.Background()
	pair, err := dtoken.LoginWithRefreshTokenOptions(ctx, dtoken.RefreshTokenOptions{LoginOptions: dtoken.LoginOptions{LoginID: "alice", Device: "web", DeviceID: "gin-example"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := dtoken.DisableDevice(ctx, dtoken.DeviceDisableOptions{LoginID: "alice", Device: "web", Duration: time.Hour}); err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	r.Use(gindt.RegisterDTokenContextMiddleware(context.Background(), gindt.WithManagerResolver(dtoken.GetManager)))
	r.POST("/refresh", handleRefresh)
	body := `{"refreshToken":"` + pair.RefreshToken + `"}`
	rec := requestGin(t, r, http.MethodPost, "/refresh", body, "")
	var response Response
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusForbidden || response.Code != dterror.CodeAccountDisabled || response.Data != nil {
		t.Fatalf("disabled refresh status=%d response=%+v", rec.Code, response)
	}
	if err := dtoken.UntieDevice(ctx, "alice", "web"); err != nil {
		t.Fatal(err)
	}
	retry := requestGin(t, r, http.MethodPost, "/refresh", body, "")
	if retry.Code != http.StatusOK {
		t.Fatalf("refresh after device recovery status = %d, want %d", retry.Code, http.StatusOK)
	}
}

func TestHandlersReportStorageFailures(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	builtAuth1, err := dtoken.NewBuilder().
		Storage(&ginFailingStorage{Storage: dt3.NewStorage()}).
		AutoRenew(false).AsyncEvent(false).IsPrintBanner(false).
		BuildInstance()
	mgr := builtAuth1.Manager()

	if err != nil {
		t.Fatal(err)
	}
	if err := dtoken.SetManager(mgr); err != nil {
		t.Fatalf("register authentication: %v", err)
	}

	// Register only request context to exercise each handler's own failure path. 仅注册请求上下文，以覆盖各处理器自身的错误分支。
	r := gin.New()
	authOption1, authErr := gindt.WithAuth(builtAuth1)
	if authErr != nil {
		t.Fatal(authErr)
	}
	r.Use(gindt.RegisterDTokenContextMiddleware(context.Background(), authOption1))
	r.POST("/login", handleLogin)
	r.POST("/refresh", handleRefresh)
	r.GET("/me", handleMe)
	r.GET("/introspect", handleIntrospect)
	r.POST("/logout", handleLogout)
	for _, tc := range []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/login", `{"username":"alice","password":"123456"}`},
		{http.MethodPost, "/refresh", `{"refreshToken":"test-refresh-token"}`},
		{http.MethodGet, "/me", ""},
		{http.MethodGet, "/introspect", ""},
		{http.MethodPost, "/logout", ""},
	} {
		t.Run(tc.path, func(t *testing.T) {
			rec := requestGin(t, r, tc.method, tc.path, tc.body, "Bearer test-access-token")
			var response Response
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if rec.Code != http.StatusInternalServerError || response.Code != dterror.CodeServerError ||
				response.Message != "internal server error" || response.Data != nil {
				t.Fatalf("storage failure status=%d response=%+v", rec.Code, response)
			}
		})
	}
}

// ginFailingStorage keeps initialization valid but fails every authentication read. ginFailingStorage 允许正常初始化，但使每次鉴权读取失败。
type ginFailingStorage struct {
	dt2.Storage
}

func (s *ginFailingStorage) Get(context.Context, string) (any, error) {
	return nil, errors.New("private storage connection details")
}

func setupGinManager(t *testing.T) {
	t.Helper()
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	if _, err := initDToken(); err != nil {
		t.Fatal(err)
	}
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

// registeredAuth returns the explicitly registered test fixture. registeredAuth 返回显式注册的测试夹具。
func registeredAuth() *dtoken.Auth {
	mgr, err := dtoken.GetManager()
	if err != nil {
		panic(err)
	}
	return dtoken.New(mgr)
}

// registerTestRoutes binds the test fixture to the production routes. registerTestRoutes 将测试夹具绑定到实际路由。
func registerTestRoutes(t *testing.T, r *gin.Engine) {
	t.Helper()
	if err := registerRoutes(r, registeredAuth()); err != nil {
		t.Fatal(err)
	}
}
