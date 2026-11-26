package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Zany2/dtoken-go"
	adapter "github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"testing"
	"time"
)

// TestProductionConfiguration validates the configuration used by main, not a separate test builder. TestProductionConfiguration 校验 main 实际使用的配置，避免独立测试配置掩盖问题。
func TestProductionConfiguration(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	if _, err := initDToken(); err != nil {
		t.Fatal(err)
	}
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
	registerTestRoutes(router)
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
	token, err := dtoken.Login(ctx, dtoken.LoginOptions{LoginID: "reader", Device: "web"})
	if err != nil {
		t.Fatal(err)
	}
	if err := dtoken.DisableDevice(ctx, dtoken.DeviceDisableOptions{LoginID: "reader", Device: "web", Duration: time.Hour}); err != nil {
		t.Fatal(err)
	}
	check(http.MethodGet, "/me", "", token, http.StatusForbidden)
	if err := dtoken.Disable(ctx, dtoken.DisableOptions{LoginID: "blocked", Duration: time.Hour, Reason: ""}); err != nil {
		t.Fatal(err)
	}
	check(http.MethodPost, "/login", `{"username":"blocked","password":"123456"}`, "", http.StatusForbidden)

	dtoken.DeleteAllManager()
	check(http.MethodGet, "/me", "", token, http.StatusInternalServerError)
	baseAuth, err := dtoken.NewBuilder().AutoRenew(false).AsyncEvent(false).IsPrintBanner(false).BuildInstance()
	if err != nil {
		t.Fatal(err)
	}
	base := baseAuth.Manager()
	t.Cleanup(base.CloseManager)
	mgrAuth, err := dtoken.NewBuilder().Storage(&quickStartFailingStorage{Storage: base.GetStorage()}).
		AutoRenew(false).AsyncEvent(false).IsPrintBanner(false).BuildInstance()
	if err != nil {
		t.Fatal(err)
	}
	mgr := mgrAuth.Manager()
	if err := dtoken.SetManager(mgr); err != nil {
		t.Fatalf("register authentication: %v", err)
	}
	router = gin.New()
	registerTestRoutes(router)
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
	registerTestRoutes(router)
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
