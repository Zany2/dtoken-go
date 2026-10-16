package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	beegodt "github.com/Zany2/dtoken-go/integrations/beego"
	web "github.com/beego/beego/v2/server/web"
)

// TestInitDTokenConfiguration verifies the actual example's renewal configuration. TestInitDTokenConfiguration 验证实际示例的续期配置。
func TestInitDTokenConfiguration(t *testing.T) {
	setupBeegoManager(t)
	mgr, err := beegodt.GetManager()
	if err != nil {
		t.Fatal(err)
	}
	cfg := mgr.GetConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	if !cfg.AutoRenew || cfg.Timeout != 7200 || cfg.RenewMaxRefresh != 3600 {
		t.Fatalf("unexpected renewal configuration: %+v", cfg)
	}
}

// TestLoginParsing covers partial form results and errors before Beego's automatic parser. TestLoginParsing 覆盖表单部分解析结果，以及 Beego 自动解析前的错误分支。
func TestLoginParsing(t *testing.T) {
	setupBeegoManager(t)
	router := newBeegoExampleRouter()
	for _, tc := range []struct{ path, body string }{
		{"/login", ""},
		{"/login", "username=alice"},
		{"/login", "username=alice&password=123456&bad=%zz"},
		{"/login", "username=alice&password=123456&bad=a;b"},
		{"/login?username=alice&password=123456&bad=%zz", ""},
		{"/login?username=alice&password=123456", "bad=%zz"},
	} {
		t.Run(tc.path+tc.body, func(t *testing.T) {
			assertBeegoResponse(t, requestBeego(router, http.MethodPost, tc.path, tc.body, ""), http.StatusBadRequest, beegodt.CodeBadRequest)
		})
	}
	for _, req := range []*http.Request{
		httptest.NewRequest(http.MethodPost, "/login?username=alice&password=123456", iotest.ErrReader(errors.New("private body read failure"))),
		httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("malformed multipart body")),
	} {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if req.URL.RawQuery == "" {
			req.Header.Set("Content-Type", "multipart/form-data; boundary=missing")
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		assertBeegoResponse(t, rec, http.StatusBadRequest, beegodt.CodeBadRequest)
		if strings.Contains(rec.Body.String(), "private") {
			t.Fatal("body read error leaked")
		}
	}

	// Keep native form precedence and multipart support compatible with Input.Query. 保持 Input.Query 原有的表单优先级及 multipart 支持。
	assertBeegoResponse(t, requestBeego(router, http.MethodPost, "/login?username=alice&password=123456", "password=bad", ""), http.StatusUnauthorized, beegodt.CodeNotLogin)
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	if err := w.WriteField("username", "alice"); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteField("password", "123456"); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/login", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if req.MultipartForm != nil {
		t.Cleanup(func() { _ = req.MultipartForm.RemoveAll() })
	}
	assertBeegoResponse(t, rec, http.StatusOK, beegodt.CodeSuccess)
}

// TestLoginBodyLimit ensures the early parser retains Beego's request size bound. TestLoginBodyLimit 确保前置解析保留 Beego 的请求体大小限制。
func TestLoginBodyLimit(t *testing.T) {
	setupBeegoManager(t)
	previous := web.BConfig.MaxMemory
	web.BConfig.MaxMemory = 32
	t.Cleanup(func() { web.BConfig.MaxMemory = previous })
	rec := requestBeego(newBeegoExampleRouter(), http.MethodPost, "/login?username=alice&password=123456", "padding="+strings.Repeat("x", 64), "")
	assertBeegoResponse(t, rec, http.StatusBadRequest, beegodt.CodeBadRequest)

	previousUpload := web.BConfig.MaxUploadSize
	web.BConfig.MaxUploadSize = 32
	t.Cleanup(func() { web.BConfig.MaxUploadSize = previousUpload })
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	if err := w.WriteField("username", "alice"); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteField("password", "123456"); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	encoded := body.String()
	for _, limit := range []int64{32, int64(len(encoded) + 1)} {
		web.BConfig.MaxUploadSize = limit
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(encoded))
		req.Header.Set("Content-Type", w.FormDataContentType())
		rec := httptest.NewRecorder()
		newBeegoExampleRouter().ServeHTTP(rec, req)
		if req.MultipartForm != nil {
			_ = req.MultipartForm.RemoveAll()
		}
		if limit == 32 {
			assertBeegoResponse(t, rec, http.StatusBadRequest, beegodt.CodeBadRequest)
		} else {
			// Multipart input may exceed MaxMemory while fitting MaxUploadSize. multipart 请求可超过 MaxMemory，但必须符合 MaxUploadSize。
			assertBeegoResponse(t, rec, http.StatusOK, beegodt.CodeSuccess)
		}
	}
}

// TestAccountAndDeviceRestrictions distinguishes restrictions from invalid credentials. TestAccountAndDeviceRestrictions 区分封禁限制与无效凭证。
func TestAccountAndDeviceRestrictions(t *testing.T) {
	setupBeegoManager(t)
	ctx := context.Background()
	if err := beegodt.Disable(ctx, "blocked", time.Hour, ""); err != nil {
		t.Fatal(err)
	}
	router := newBeegoExampleRouter()
	assertBeegoResponse(t, requestBeego(router, http.MethodPost, "/login", "username=blocked&password=123456", ""), http.StatusForbidden, beegodt.CodeAccountDisabled)
	token, err := beegodt.Login(ctx, "alice", "web")
	if err != nil {
		t.Fatal(err)
	}
	if err := beegodt.DisableDevice(ctx, "alice", "web", time.Hour); err != nil {
		t.Fatal(err)
	}
	assertBeegoResponse(t, requestBeego(router, http.MethodGet, "/me", "", token), http.StatusForbidden, beegodt.CodeAccountDisabled)
	if err := beegodt.UntieDevice(ctx, "alice", "web"); err != nil {
		t.Fatal(err)
	}
	assertBeegoResponse(t, requestBeego(router, http.MethodGet, "/me", "", token), http.StatusOK, beegodt.CodeSuccess)
}

// TestStorageErrorsUseUnifiedResponse verifies both handler and filter failures are sanitized. TestStorageErrorsUseUnifiedResponse 验证处理器与过滤器的故障响应均隐藏内部详情。
func TestStorageErrorsUseUnifiedResponse(t *testing.T) {
	beegodt.DeleteAllManager()
	t.Cleanup(beegodt.DeleteAllManager)
	mgr, err := beegodt.NewBuilder().SetStorage(&failingBeegoStorage{Storage: beegodt.NewMemoryStorage()}).
		AutoRenew(false).AsyncEvent(false).IsPrintBanner(false).Build()
	if err != nil {
		t.Fatal(err)
	}
	beegodt.SetManager(mgr)
	for _, withAuth := range []bool{false, true} {
		t.Run(fmt.Sprintf("withAuth=%t", withAuth), func(t *testing.T) {
			router := web.NewControllerRegister()
			if withAuth {
				registerRoutes(router)
			} else {
				// Reach handler errors independently of the authentication filter. 独立覆盖处理器中的错误分支，避免被认证过滤器提前拦截。
				if err := router.InsertFilter("/*", web.BeforeRouter, beegodt.RegisterDTokenContextMiddleware(context.Background(), beegodt.WithFailFunc(writeAuthError))); err != nil {
					t.Fatal(err)
				}
				router.Post("/login", handleLogin)
				router.Get("/me", handleMe)
				router.Post("/logout", handleLogout)
			}
			for _, tc := range []struct{ method, path, body string }{
				{http.MethodPost, "/login", "username=alice&password=123456"},
				{http.MethodGet, "/me", ""},
				{http.MethodPost, "/logout", ""},
			} {
				rec := requestBeego(router, tc.method, tc.path, tc.body, "some-token")
				assertBeegoResponse(t, rec, http.StatusInternalServerError, beegodt.CodeServerError)
				if strings.Contains(rec.Body.String(), "private storage") || !strings.Contains(rec.Body.String(), "internal server error") {
					t.Fatalf("unsafe server response: %s", rec.Body.String())
				}
			}
		})
	}
}

// TestMissingManagerStopsHandlers verifies context-registration failure stops native routing. TestMissingManagerStopsHandlers 验证上下文注册失败后原生路由不会继续处理请求。
func TestMissingManagerStopsHandlers(t *testing.T) {
	beegodt.DeleteAllManager()
	t.Cleanup(beegodt.DeleteAllManager)
	router := newBeegoExampleRouter()
	for _, path := range []string{"/login?username=alice&password=123456", "/logout"} {
		assertBeegoResponse(t, requestBeego(router, http.MethodPost, path, "", ""), http.StatusInternalServerError, beegodt.CodeServerError)
	}
}

type failingBeegoStorage struct{ beegodt.Storage }

// Get simulates a backend failure containing private details. Get 模拟包含私有详情的后端故障。
func (*failingBeegoStorage) Get(context.Context, string) (any, error) {
	return nil, errors.New("private storage connection details")
}
