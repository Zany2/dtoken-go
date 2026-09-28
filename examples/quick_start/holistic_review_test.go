package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/defaults"
	"github.com/Zany2/dtoken-go/dtoken"
	"github.com/gin-gonic/gin"
)

// TestProductionConfiguration validates the configuration used by main, not a separate test builder. TestProductionConfiguration 校验 main 实际使用的配置，避免独立测试配置掩盖问题。
func TestProductionConfiguration(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	initDToken()
	mgr, err := dtoken.GetManager()
	if err != nil {
		t.Fatal(err)
	}
	cfg := mgr.GetConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	if cfg.Timeout != 7200 || cfg.RenewMaxRefresh != 3600 || !cfg.AutoRenew || cfg.TokenName != tokenHeader {
		t.Fatalf("unexpected configuration: %+v", cfg)
	}
}

// TestProductionRouteFailures separates login, restriction and backend failures through actual routes. TestProductionRouteFailures 通过实际路由区分登录、封禁与后端故障。
func TestProductionRouteFailures(t *testing.T) {
	setupQuickStartManager(t)
	router := gin.New()
	registerRoutes(router)
	check := func(method, path, body, token string, want int) {
		t.Helper()
		rec := requestQuickStart(t, router, method, path, body, token)
		var response Response
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if rec.Code != want || response.Code != want || response.Data != nil {
			t.Fatalf("response=%d %+v, want %d", rec.Code, response, want)
		}
		if want == http.StatusInternalServerError && response.Message != "internal server error" {
			t.Fatalf("internal details leaked: %+v", response)
		}
	}
	for _, path := range []string{"/me", "/admin", "/articles"} {
		check(http.MethodGet, path, "", "", http.StatusUnauthorized)
	}
	ctx := context.Background()
	token, err := dtoken.Login(ctx, "reader", "web")
	if err != nil {
		t.Fatal(err)
	}
	if err := dtoken.DisableDevice(ctx, "reader", "web", time.Hour); err != nil {
		t.Fatal(err)
	}
	check(http.MethodGet, "/me", "", token, http.StatusForbidden)
	if err := dtoken.Disable(ctx, "blocked", time.Hour, ""); err != nil {
		t.Fatal(err)
	}
	check(http.MethodPost, "/login", `{"username":"blocked","password":"123456"}`, "", http.StatusForbidden)

	dtoken.DeleteAllManager()
	check(http.MethodGet, "/me", "", token, http.StatusInternalServerError)
	base, err := defaults.NewBuilder().AutoRenew(false).AsyncEvent(false).IsPrintBanner(false).Build()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(base.CloseManager)
	mgr, err := defaults.NewBuilder().SetStorage(&quickStartFailingStorage{Storage: base.GetStorage()}).
		AutoRenew(false).AsyncEvent(false).IsPrintBanner(false).Build()
	if err != nil {
		t.Fatal(err)
	}
	dtoken.SetManager(mgr)
	check(http.MethodPost, "/login", `{"username":"alice","password":"123456"}`, "", http.StatusInternalServerError)
	for _, path := range []string{"/me", "/admin", "/articles", "/logout"} {
		method := http.MethodGet
		if path == "/logout" {
			method = http.MethodPost
		}
		check(method, path, "", "some-token", http.StatusInternalServerError)
	}
}

// TestLoginRejectsTrailingJSON ensures malformed input cannot create login state. TestLoginRejectsTrailingJSON 确保非法输入无法创建登录态。
func TestLoginRejectsTrailingJSON(t *testing.T) {
	setupQuickStartManager(t)
	router := gin.New()
	registerRoutes(router)
	valid := `{"username":"alice","password":"123456"}`
	for _, suffix := range []string{" {}", " null", " trailing"} {
		rec := requestQuickStart(t, router, http.MethodPost, "/login", valid+suffix, "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("invalid input accepted: %d %s", rec.Code, rec.Body.String())
		}
	}
	if _, err := dtoken.GetSession(context.Background(), "alice"); !errors.Is(err, derror.ErrSessionNotFound) {
		t.Fatalf("invalid input created a session: %v", err)
	}
	rec := requestQuickStart(t, router, http.MethodPost, "/login", valid+"\n\t", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("valid input rejected: %d %s", rec.Code, rec.Body.String())
	}
}

type quickStartFailingStorage struct{ adapter.Storage }

// Get injects a backend error with private details. Get 注入带有私有详情的后端错误。
func (*quickStartFailingStorage) Get(context.Context, string) (any, error) {
	return nil, errors.New("private storage connection details")
}
