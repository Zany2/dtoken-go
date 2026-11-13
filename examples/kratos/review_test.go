package main

import (
	dtoken "github.com/Zany2/dtoken-go"
	dt1 "github.com/Zany2/dtoken-go/contract"
	dterror "github.com/Zany2/dtoken-go/errors"
	dt3 "github.com/Zany2/dtoken-go/global"
	dt2 "github.com/Zany2/dtoken-go/storage/memory"

	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"testing/iotest"
	"time"

	kratosdt "github.com/Zany2/dtoken-go/integrations/kratos"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

// TestInitDTokenConfiguration verifies the actual example uses a valid renewal threshold. TestInitDTokenConfiguration 验证示例实际使用的续期阈值合法。
func TestInitDTokenConfiguration(t *testing.T) {
	setupKratosManager(t)
	mgr, err := dt3.GetManager()
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

// TestLoginRejectsMalformedBodies covers decoder errors without creating a login. TestLoginRejectsMalformedBodies 覆盖解码错误，确保非法请求无法登录。
func TestLoginRejectsMalformedBodies(t *testing.T) {
	setupKratosManager(t)
	srv := newKratosExampleServer()
	for _, body := range []string{
		"", " \n\t ", "null", "[]", `{"username":"alice",`,
		`{"username":123,"password":"123456"}`,
		`{"username":"alice","password":123456}`,
		`{"username":"alice","password":"123456"} {}`,
		`{"username":"alice","password":"123456"} null`,
		`{"username":"alice","password":"123456"} trailing`,
	} {
		t.Run(body, func(t *testing.T) {
			assertKratosResponse(t, requestKratos(t, srv, http.MethodPost, "/login", body, ""), http.StatusBadRequest, dterror.CodeBadRequest)
		})
	}
	assertKratosResponse(t, requestKratos(t, srv, http.MethodPost, "/login?username=alice&password=123456", `{}`, ""), http.StatusBadRequest, dterror.CodeBadRequest)
	req := httptest.NewRequest(http.MethodPost, "/login", iotest.ErrReader(errors.New("private request read error")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	assertKratosResponse(t, rec, http.StatusBadRequest, dterror.CodeBadRequest)
	if strings.Contains(rec.Body.String(), "private request") {
		t.Fatal("request read error leaked")
	}
	assertKratosResponse(t, requestKratos(t, srv, http.MethodPost, "/login", `{"username":"alice","password":"123456"}`+"\n\t", ""), http.StatusOK, dterror.CodeSuccess)
}

// TestAccountAndDeviceRestrictions distinguishes access restrictions from invalid credentials. TestAccountAndDeviceRestrictions 区分访问限制与无效凭证。
func TestAccountAndDeviceRestrictions(t *testing.T) {
	setupKratosManager(t)
	ctx := context.Background()
	if err := dt3.Disable(ctx, "blocked", time.Hour, ""); err != nil {
		t.Fatal(err)
	}
	srv := newKratosExampleServer()
	assertKratosResponse(t, requestKratos(t, srv, http.MethodPost, "/login", `{"username":"blocked","password":"123456"}`, ""), http.StatusForbidden, dterror.CodeAccountDisabled)
	token, err := dt3.Login(ctx, "alice", "web")
	if err != nil {
		t.Fatal(err)
	}
	if err := dt3.DisableDevice(ctx, "alice", "web", time.Hour); err != nil {
		t.Fatal(err)
	}
	assertKratosResponse(t, requestKratos(t, srv, http.MethodGet, "/me", "", token), http.StatusForbidden, dterror.CodeAccountDisabled)
	if err := dt3.UntieDevice(ctx, "alice", "web"); err != nil {
		t.Fatal(err)
	}
	assertKratosResponse(t, requestKratos(t, srv, http.MethodGet, "/me", "", token), http.StatusOK, dterror.CodeSuccess)
}

// TestStorageErrorsUseUnifiedResponse covers raw handler errors and wrapped middleware failures. TestStorageErrorsUseUnifiedResponse 覆盖处理器原始错误和中间件包装错误。
func TestStorageErrorsUseUnifiedResponse(t *testing.T) {
	dt3.DeleteAllManager()
	t.Cleanup(dt3.DeleteAllManager)
	storage := &failingKratosStorage{Storage: dt2.NewStorage()}
	builtAuth1, err := dtoken.NewBuilder().Storage(storage).
		AutoRenew(false).AsyncEvent(false).IsPrintBanner(false).Build()
	mgr := builtAuth1.Manager()

	if err != nil {
		t.Fatal(err)
	}
	dt3.SetManager(mgr)
	for _, withAuth := range []bool{false, true} {
		t.Run(fmt.Sprintf("withAuth=%t", withAuth), func(t *testing.T) {
			handlerCalls := 0
			var srv *khttp.Server
			if withAuth {
				srv = newKratosExampleServer()
			} else {
				// Skip authentication middleware to reach each handler's own error branch. 跳过认证中间件以到达各处理器自身的错误分支。
				srv = khttp.NewServer(khttp.Middleware(kratosdt.RegisterDTokenContextMiddleware(kratosdt.WithAuth(builtAuth1))))
				r := srv.Route("/")
				counted := func(handler func(context.Context, khttp.Context) error) func(context.Context, khttp.Context) error {
					return func(ctx context.Context, c khttp.Context) error {
						handlerCalls++
						return handler(ctx, c)
					}
				}
				r.POST("/login", wrapHandler(counted(handleLogin)))
				r.GET("/me", wrapHandler(counted(handleMe)))
				r.POST("/logout", wrapHandler(counted(handleLogout)))
			}
			for _, tc := range []struct{ method, path, body string }{
				{http.MethodPost, "/login", `{"username":"alice","password":"123456"}`},
				{http.MethodGet, "/me", ""},
				{http.MethodPost, "/logout", ""},
			} {
				beforeStorage, beforeHandlers := storage.calls.Load(), handlerCalls
				rec := requestKratos(t, srv, tc.method, tc.path, tc.body, "some-token")
				if storage.calls.Load() <= beforeStorage {
					t.Fatalf("%s never reached storage", tc.path)
				}
				if !withAuth && handlerCalls != beforeHandlers+1 {
					t.Fatalf("%s never reached its handler", tc.path)
				}

				assertKratosResponse(t, rec, http.StatusInternalServerError, dterror.CodeServerError)
				if strings.Contains(rec.Body.String(), "private storage") || !strings.Contains(rec.Body.String(), "internal server error") {
					t.Fatalf("unsafe server response: %s", rec.Body.String())
				}
			}
		})
	}
}

// TestMissingManagerStopsHandlers verifies registration failures use the same error envelope. TestMissingManagerStopsHandlers 验证注册失败使用相同的错误响应结构。
func TestMissingManagerStopsHandlers(t *testing.T) {
	dt3.DeleteAllManager()
	t.Cleanup(dt3.DeleteAllManager)
	srv := newKratosExampleServer()
	assertKratosResponse(t, requestKratos(t, srv, http.MethodPost, "/login", `{"username":"alice","password":"123456"}`, ""), http.StatusInternalServerError, dterror.CodeServerError)
}

type failingKratosStorage struct {
	dt1.Storage
	calls atomic.Int64
}

// Get simulates a storage failure containing private connection details. Get 模拟包含私有连接详情的存储故障。
func (s *failingKratosStorage) Get(context.Context, string) (any, error) {
	s.calls.Add(1)
	return nil, errors.New("private storage connection details")
}

func assertKratosResponse(t *testing.T, rec *httptest.ResponseRecorder, status, code int) {
	t.Helper()
	var response Response
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("invalid JSON response: %v, body=%s", err, rec.Body.String())
	}
	if rec.Code != status || response.Code != code || !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("response=%d %s, want status=%d code=%d", rec.Code, rec.Body.String(), status, code)
	}
}
