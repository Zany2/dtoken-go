package main

import (
	dtoken "github.com/Zany2/dtoken-go"
	dt1 "github.com/Zany2/dtoken-go/contract"
	dterror "github.com/Zany2/dtoken-go/errors"
	dt2 "github.com/Zany2/dtoken-go/storage/memory"

	"context"
	"encoding/json"
	"errors"
	"fmt"
	dtglobal "github.com/Zany2/dtoken-go/global"
	hertzdt "github.com/Zany2/dtoken-go/integrations/hertz"
	hertzapp "github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestInitDTokenConfiguration verifies that automatic renewal fits the token timeout. TestInitDTokenConfiguration 验证自动续期配置与 Token 有效期兼容。
func TestInitDTokenConfiguration(t *testing.T) {
	setupHertzManager(t)
	mgr, err := dtglobal.GetManager()
	if err != nil {
		t.Fatal(err)
	}
	cfg := mgr.GetConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("invalid example configuration: %v", err)
	}
	if !cfg.AutoRenew || cfg.Timeout != 7200 || cfg.RenewMaxRefresh != 3600 {
		t.Fatalf("unexpected renewal configuration: %+v", cfg)
	}
}

func TestHandleLoginValidation(t *testing.T) {
	setupHertzManager(t)
	h := server.Default()
	registerRoutes(h)

	for _, body := range []string{
		"", " \n\t ", "null", "[]", `{"username":"alice"}`,
		`{"username":"alice",`, `{"username":123,"password":"123456"}`,
		`{"username":"alice","password":123456}`,
		`{"username":"alice","password":"123456"} {}`,
		`{"username":"alice","password":"123456"} null`,
		`{"username":"alice","password":"123456"} trailing`,
	} {
		t.Run(body, func(t *testing.T) {
			assertHertzResponse(t, requestHertz(h, http.MethodPost, "/login", body, ""), http.StatusBadRequest, dterror.CodeBadRequest)
		})
	}
	assertHertzResponse(t, requestHertz(h, http.MethodPost, "/login", `{"username":"alice","password":"bad"}`, ""), http.StatusUnauthorized, dterror.CodeNotLogin)
	assertHertzResponse(t, requestHertz(h, http.MethodPost, "/login?username=alice&password=123456", `{}`, ""), http.StatusBadRequest, dterror.CodeBadRequest)
}

// TestProtectedRoutesAndLogout covers the actual login, authentication and authorization chain. TestProtectedRoutesAndLogout 覆盖真实登录、认证和权限校验链。
func TestProtectedRoutesAndLogout(t *testing.T) {
	setupHertzManager(t)
	ctx := context.Background()
	h := server.Default()
	registerRoutes(h)
	login := requestHertz(h, http.MethodPost, "/login", `{"username":"alice","password":"123456"}`+"\n", "")
	data := assertHertzResponse(t, login, http.StatusOK, dterror.CodeSuccess)
	var result struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	token := result.Token
	if id, err := dtglobal.GetLoginID(ctx, token); err != nil || id != "alice" {
		t.Fatalf("returned token login ID=%q error=%v", id, err)
	}
	if !dtglobal.HasRole(ctx, "alice", "admin") || !dtglobal.HasPermission(ctx, "alice", "article:read") {
		t.Fatal("login did not seed the demo authorization data")
	}

	for _, path := range []string{"/me", "/admin", "/articles"} {
		for _, header := range []string{"Authorization", "dtoken"} {
			for _, value := range []string{token, "Bearer " + token} {
				request := hertzRequestWithBody("")
				request.Request.SetMethod(http.MethodGet)
				request.Request.SetRequestURI("http://example.com" + path)
				request.Request.Header.Set(header, value)
				h.ServeHTTP(ctx, request)
				data := assertHertzResponse(t, request, http.StatusOK, dterror.CodeSuccess)
				if path == "/me" {
					var info struct {
						LoginID string `json:"loginId"`
					}
					if err := json.Unmarshal(data, &info); err != nil || info.LoginID != "alice" {
						t.Fatalf("unexpected /me data=%s error=%v", data, err)
					}
				}
			}
		}
	}
	for _, path := range []string{"/me", "/admin", "/articles", "/logout"} {
		method := http.MethodGet
		if path == "/logout" {
			method = http.MethodPost
		}
		for _, invalidToken := range []string{"", "invalid-token"} {
			assertHertzResponse(t, requestHertz(h, method, path, "", invalidToken), http.StatusUnauthorized, dterror.CodeNotLogin)
		}
	}
	bobToken, err := dtglobal.Login(ctx, "bob")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/admin", "/articles"} {
		assertHertzResponse(t, requestHertz(h, http.MethodGet, path, "", bobToken), http.StatusForbidden, dterror.CodePermissionDenied)
	}
	assertHertzResponse(t, requestHertz(h, http.MethodPost, "/logout", "", token), http.StatusOK, dterror.CodeSuccess)
	assertHertzResponse(t, requestHertz(h, http.MethodGet, "/me", "", token), http.StatusUnauthorized, dterror.CodeNotLogin)
	if dtglobal.IsLogin(ctx, token) {
		t.Fatal("token remains logged in after logout")
	}
}

// TestDisabledAccountLogin verifies a blocked account receives an access restriction response. TestDisabledAccountLogin 验证封禁账号登录返回访问限制响应。
func TestDisabledAccountLogin(t *testing.T) {
	setupHertzManager(t)
	if err := dtglobal.Disable(context.Background(), "alice", time.Hour, ""); err != nil {
		t.Fatal(err)
	}
	h := server.Default()
	registerRoutes(h)
	assertHertzResponse(t, requestHertz(h, http.MethodPost, "/login", `{"username":"alice","password":"123456"}`, ""), http.StatusForbidden, dterror.CodeAccountDisabled)
}

// TestStorageFailuresStayServerErrors verifies handlers and middleware preserve backend failures without leaking details. TestStorageFailuresStayServerErrors 验证处理器和中间件正确报告后端故障且不泄露详情。
func TestStorageFailuresStayServerErrors(t *testing.T) {
	dtglobal.DeleteAllManager()
	t.Cleanup(dtglobal.DeleteAllManager)
	builtAuth1, err := dtoken.NewBuilder().Storage(&failingHertzStorage{Storage: dt2.NewStorage()}).
		AutoRenew(false).AsyncEvent(false).IsPrintBanner(false).Build()
	mgr := builtAuth1.Manager()

	if err != nil {
		t.Fatal(err)
	}
	dtglobal.SetManager(mgr)
	for _, withAuth := range []bool{false, true} {
		t.Run(fmt.Sprintf("withAuth=%t", withAuth), func(t *testing.T) {
			h := server.Default()
			if withAuth {
				registerRoutes(h)
			} else {
				// Register only the context so handler failures reach the handler's own error branch. 仅注册上下文，使处理器自身的错误分支可被验证。
				h.Use(hertzdt.RegisterDTokenContextMiddleware(context.Background(), hertzdt.WithAuth(builtAuth1)))
				h.POST("/login", handleLogin)
				h.GET("/me", handleMe)
				h.POST("/logout", handleLogout)
			}
			for _, tc := range []struct{ method, path, body string }{
				{http.MethodPost, "/login", `{"username":"alice","password":"123456"}`},
				{http.MethodGet, "/me", ""},
				{http.MethodPost, "/logout", ""},
			} {
				response := requestHertz(h, tc.method, tc.path, tc.body, "some-token")
				assertHertzResponse(t, response, http.StatusInternalServerError, dterror.CodeServerError)
				body := string(response.Response.BodyBytes())
				if strings.Contains(body, "private storage") || !strings.Contains(body, "internal server error") {
					t.Fatalf("unsafe server response: %s", body)
				}
			}
		})
	}
}

type failingHertzStorage struct{ dt1.Storage }

// Get simulates a backend failure containing private details. Get 模拟包含私有详情的后端故障。
func (*failingHertzStorage) Get(context.Context, string) (any, error) {
	return nil, errors.New("private storage connection details")
}

func setupHertzManager(t *testing.T) {
	t.Helper()
	dtglobal.DeleteAllManager()
	t.Cleanup(dtglobal.DeleteAllManager)
	initDToken()
}

func hertzRequestWithBody(body string) *hertzapp.RequestContext {
	ctx := hertzapp.NewContext(0)
	ctx.Request.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("http://example.com/login")
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Request.SetBodyString(body)

	// Match the headers populated by Hertz's HTTP body reader. 模拟 Hertz HTTP 请求体读取器填充的长度头。
	ctx.Request.Header.SetContentLength(len(body))
	return ctx
}

func requestHertz(h *server.Hertz, method, path, body, token string) *hertzapp.RequestContext {
	ctx := hertzRequestWithBody(body)
	ctx.Request.SetMethod(method)
	ctx.Request.SetRequestURI("http://example.com" + path)
	if token != "" {
		ctx.Request.Header.Set("Authorization", token)
	}
	h.ServeHTTP(context.Background(), ctx)
	return ctx
}

func assertHertzResponse(t *testing.T, ctx *hertzapp.RequestContext, status, code int) json.RawMessage {
	t.Helper()
	var response struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(ctx.Response.BodyBytes(), &response); err != nil {
		t.Fatalf("invalid JSON response: %v, body=%s", err, ctx.Response.BodyBytes())
	}
	if ctx.Response.StatusCode() != status || response.Code != code || !strings.HasPrefix(string(ctx.Response.Header.ContentType()), "application/json") {
		t.Fatalf("response=%d %s, want status=%d code=%d", ctx.Response.StatusCode(), ctx.Response.BodyBytes(), status, code)
	}
	return response.Data
}
