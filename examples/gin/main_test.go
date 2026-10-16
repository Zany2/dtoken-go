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
	gindt "github.com/Zany2/dtoken-go/integrations/gin"
	"github.com/gin-gonic/gin"
)

func TestInitDTokenConfiguration(t *testing.T) {
	setupGinManager(t)
	mgr, err := gindt.GetManager()
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
	r.Use(gindt.RegisterDTokenContextMiddleware(context.Background()))
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
		Code int                    `json:"code"`
		Data gindt.RefreshTokenPair `json:"data"`
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
	var rotated struct {
		Code int                    `json:"code"`
		Data gindt.RefreshTokenPair `json:"data"`
	}
	if err := json.Unmarshal(refresh.Body.Bytes(), &rotated); err != nil {
		t.Fatalf("decode refresh response: %v", err)
	}
	if rotated.Code != gindt.CodeSuccess || rotated.Data.AccessToken == "" || rotated.Data.RefreshToken == "" ||
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
	registerRoutes(r)

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
				err = gindt.DisableDevice(context.Background(), "alice", "web", time.Hour)
			} else {
				err = gindt.Disable(context.Background(), "alice", time.Hour, "")
			}
			if err != nil {
				t.Fatal(err)
			}
			r := gin.New()
			r.POST("/login", handleLogin)
			rec := requestGin(t, r, http.MethodPost, "/login", `{"username":"alice","password":"123456"}`, "")
			var response Response
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if rec.Code != http.StatusForbidden || response.Code != gindt.CodeAccountDisabled || response.Data != nil {
				t.Fatalf("disabled login status=%d response=%+v", rec.Code, response)
			}
		})
	}
}

func TestRefreshRejectsDisabledDeviceWithoutConsumingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupGinManager(t)
	ctx := context.Background()
	pair, err := gindt.LoginWithRefreshToken(ctx, "alice", "web", "gin-example")
	if err != nil {
		t.Fatal(err)
	}
	if err := gindt.DisableDevice(ctx, "alice", "web", time.Hour); err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	r.POST("/refresh", handleRefresh)
	body := `{"refreshToken":"` + pair.RefreshToken + `"}`
	rec := requestGin(t, r, http.MethodPost, "/refresh", body, "")
	var response Response
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusForbidden || response.Code != gindt.CodeAccountDisabled || response.Data != nil {
		t.Fatalf("disabled refresh status=%d response=%+v", rec.Code, response)
	}
	if err := gindt.UntieDevice(ctx, "alice", "web"); err != nil {
		t.Fatal(err)
	}
	retry := requestGin(t, r, http.MethodPost, "/refresh", body, "")
	if retry.Code != http.StatusOK {
		t.Fatalf("refresh after device recovery status = %d, want %d", retry.Code, http.StatusOK)
	}
}

func TestHandlersReportStorageFailures(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gindt.DeleteAllManager()
	t.Cleanup(gindt.DeleteAllManager)
	mgr, err := gindt.NewBuilder().
		SetStorage(&ginFailingStorage{Storage: gindt.NewMemoryStorage()}).
		AutoRenew(false).AsyncEvent(false).IsPrintBanner(false).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	gindt.SetManager(mgr)

	// Register only request context to exercise each handler's own failure path. 仅注册请求上下文，以覆盖各处理器自身的错误分支。
	r := gin.New()
	r.Use(gindt.RegisterDTokenContextMiddleware(context.Background()))
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
			if rec.Code != http.StatusInternalServerError || response.Code != gindt.CodeServerError ||
				response.Message != "internal server error" || response.Data != nil {
				t.Fatalf("storage failure status=%d response=%+v", rec.Code, response)
			}
		})
	}
}

// ginFailingStorage keeps initialization valid but fails every authentication read. ginFailingStorage 允许正常初始化，但使每次鉴权读取失败。
type ginFailingStorage struct {
	gindt.Storage
}

func (s *ginFailingStorage) Get(context.Context, string) (any, error) {
	return nil, errors.New("private storage connection details")
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
