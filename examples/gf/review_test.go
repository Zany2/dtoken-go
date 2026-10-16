package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gfdt "github.com/Zany2/dtoken-go/integrations/gf"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gsession"
	"github.com/gogf/gf/v2/util/guid"
)

// TestRouteAccessUsesMatchedRoute verifies normalization and route overrides retain the target's checks. TestRouteAccessUsesMatchedRoute 验证规范化和路由覆盖后仍执行目标路由的鉴权规则。
func TestRouteAccessUsesMatchedRoute(t *testing.T) {
	mgr := setupGFManager(t)
	s := newGFServer(t, registerRoutes)
	ctx := context.Background()
	token, err := mgr.Login(ctx, "reader")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/access/public", "/access/public/", "/access//public"} {
		assertGFResponse(t, requestGF(s, http.MethodGet, path, "", ""), http.StatusOK, gfdt.CodeSuccess)
	}
	for _, path := range []string{"/me", "/introspect", "/admin", "/articles", "/access/me", "/access/admin", "/access/articles"} {
		for _, invalidToken := range []string{"", "invalid-token"} {
			assertGFResponse(t, requestGF(s, http.MethodGet, path, "", invalidToken), http.StatusUnauthorized, gfdt.CodeNotLogin)
		}
	}
	assertGFResponse(t, requestGF(s, http.MethodGet, "/access/me", "", token), http.StatusOK, gfdt.CodeSuccess)
	assertGFResponse(t, requestGF(s, http.MethodPost, "/logout", "", ""), http.StatusUnauthorized, gfdt.CodeNotLogin)
	for _, path := range []string{"/admin", "/articles", "/access/admin", "/access/articles", "/access//admin", "/access//articles", "//access/admin/"} {
		assertGFResponse(t, requestGF(s, http.MethodGet, path, "", token), http.StatusForbidden, gfdt.CodePermissionDenied)
	}
	for _, target := range []string{"/access/admin", "/access/articles", "/access//admin"} {
		for _, credential := range []string{"", token} {
			req := httptest.NewRequest(http.MethodGet, "/access/public", nil)
			req.Header.Set(ghttp.HeaderXUrlPath, target)
			req.Header.Set("Authorization", credential)
			rec := httptest.NewRecorder()
			s.ServeHTTP(rec, req)
			status, code := http.StatusUnauthorized, gfdt.CodeNotLogin
			if credential != "" {
				status, code = http.StatusForbidden, gfdt.CodePermissionDenied
			}
			assertGFResponse(t, rec, status, code)
		}
	}
	if err := mgr.AddRoles(ctx, "reader", []string{"admin"}); err != nil {
		t.Fatal(err)
	}
	if err := mgr.AddPermissions(ctx, "reader", []string{"article:read"}); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/admin", "/articles", "/access/admin", "/access/articles", "/access//admin", "/access//articles"} {
		assertGFResponse(t, requestGF(s, http.MethodGet, path, "", token), http.StatusOK, gfdt.CodeSuccess)
	}
}

// TestLoginRefreshLogout verifies the JSON envelope and the complete token-pair lifecycle. TestLoginRefreshLogout 验证 JSON 响应及令牌对的完整生命周期。
func TestLoginRefreshLogout(t *testing.T) {
	setupGFManager(t)
	s := newGFServer(t, registerRoutes)
	pair := decodeGFPair(t, requestGF(s, http.MethodPost, "/login", "username=alice&password=123456", ""))
	if pair.LoginID != "alice" || pair.Device != "web" || pair.DeviceID != "gf-example" {
		t.Fatalf("unexpected login identity: %+v", pair)
	}
	for _, header := range []string{"Authorization", "dtoken"} {
		for _, token := range []string{pair.AccessToken, "Bearer " + pair.AccessToken} {
			req := httptest.NewRequest(http.MethodGet, "/me", nil)
			req.Header.Set(header, token)
			rec := httptest.NewRecorder()
			s.ServeHTTP(rec, req)
			data := assertGFResponse(t, rec, http.StatusOK, gfdt.CodeSuccess)
			var info struct {
				LoginID     string   `json:"loginId"`
				Roles       []string `json:"roles"`
				Permissions []string `json:"permissions"`
			}
			if err := json.Unmarshal(data, &info); err != nil {
				t.Fatal(err)
			}
			if info.LoginID != "alice" || len(info.Roles) != 1 || info.Roles[0] != "admin" || len(info.Permissions) != 1 || info.Permissions[0] != "article:read" {
				t.Fatalf("unexpected login information: %+v", info)
			}
		}
	}
	rotated := decodeGFPair(t, requestGF(s, http.MethodPost, "/refresh", "refreshToken="+pair.RefreshToken, ""))
	if rotated.AccessToken == pair.AccessToken || rotated.RefreshToken == pair.RefreshToken {
		t.Fatal("refresh did not rotate both credentials")
	}
	assertGFResponse(t, requestGF(s, http.MethodGet, "/me", "", pair.AccessToken), http.StatusUnauthorized, gfdt.CodeNotLogin)
	assertGFResponse(t, requestGF(s, http.MethodPost, "/refresh", "refreshToken="+pair.RefreshToken, ""), http.StatusUnauthorized, gfdt.CodeNotLogin)
	data := assertGFResponse(t, requestGF(s, http.MethodGet, "/introspect", "", rotated.AccessToken), http.StatusOK, gfdt.CodeSuccess)
	var info gfdt.TokenIntrospection
	if err := json.Unmarshal(data, &info); err != nil {
		t.Fatal(err)
	}
	if !info.Active || info.LoginID != "alice" {
		t.Fatalf("unexpected introspection: %+v", info)
	}
	assertGFResponse(t, requestGF(s, http.MethodPost, "/logout", "", rotated.AccessToken), http.StatusOK, gfdt.CodeSuccess)
	assertGFResponse(t, requestGF(s, http.MethodGet, "/me", "", rotated.AccessToken), http.StatusUnauthorized, gfdt.CodeNotLogin)
	assertGFResponse(t, requestGF(s, http.MethodPost, "/refresh", "refreshToken="+rotated.RefreshToken, ""), http.StatusUnauthorized, gfdt.CodeNotLogin)
}

// TestCredentialBodyValidation covers malformed input and prevents query fallback. TestCredentialBodyValidation 覆盖损坏输入并防止回退到 Query 参数。
func TestCredentialBodyValidation(t *testing.T) {
	setupGFManager(t)
	s := newGFServer(t, registerRoutes)
	for _, tc := range []struct {
		path, body, contentType string
	}{
		{"/login", "", "application/json"},
		{"/login", " \n\t ", "application/json"},
		{"/login", "null", "application/json"},
		{"/login", "[]", "application/json"},
		{"/login", `{"username":"alice",`, "application/json"},
		{"/login", `{"username":123,"password":"123456"}`, "application/json"},
		{"/login", `{"username":"alice","password":123456}`, "application/json"},
		{"/login", `{"username":"alice","password":"123456"} {}`, "application/json"},
		{"/login", `{"username":"alice","password":"123456"} null`, "application/json"},
		{"/login", "username=alice&password=123456%xx", "application/x-www-form-urlencoded"},
		{"/login?username=alice&password=123456", "", "application/x-www-form-urlencoded"},
		{"/login?password=123456", "username=alice", "application/x-www-form-urlencoded"},
		{"/login", "bad multipart body", "multipart/form-data; boundary=example"},
		{"/login", "username=alice&password=123456", "text/plain"},
		{"/refresh", "", "application/x-www-form-urlencoded"},
		{"/refresh", `{"refreshToken":123}`, "application/json"},
		{"/refresh", `{"refreshToken":"invalid"} trailing`, "application/json"},
		{"/refresh?refreshToken=query-token", "", "application/x-www-form-urlencoded"},
	} {
		t.Run(tc.path+"/"+tc.body, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", tc.contentType)
			rec := httptest.NewRecorder()
			s.ServeHTTP(rec, req)
			assertGFResponse(t, rec, http.StatusBadRequest, gfdt.CodeBadRequest)
		})
	}
	assertGFResponse(t, requestGF(s, http.MethodPost, "/login", "username=alice&password=wrong", ""), http.StatusUnauthorized, gfdt.CodeNotLogin)
	assertGFResponse(t, requestGF(s, http.MethodPost, "/refresh", "refreshToken=invalid", ""), http.StatusUnauthorized, gfdt.CodeNotLogin)
	jsonRequest := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"username":"json-user","password":"123456"}`+"\n"))
	jsonRequest.Header.Set("Content-Type", "application/json; charset=utf-8")
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, jsonRequest)
	pair := decodeGFPair(t, rec)
	jsonRequest = httptest.NewRequest(http.MethodPost, "/refresh", strings.NewReader(`{"refreshToken":"`+pair.RefreshToken+`"}`))
	jsonRequest.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, jsonRequest)
	decodeGFPair(t, rec)
	var form bytes.Buffer
	writer := multipart.NewWriter(&form)
	for key, value := range map[string]string{"username": "form-user", "password": "123456"} {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/login", &form)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	decodeGFPair(t, rec)
}

// TestAccountAndDeviceRestrictions verifies 403 responses and retry after device unblocking. TestAccountAndDeviceRestrictions 验证封禁返回 403，设备解封后可重试刷新。
func TestAccountAndDeviceRestrictions(t *testing.T) {
	mgr := setupGFManager(t)
	s := newGFServer(t, registerRoutes)
	ctx := context.Background()
	if err := mgr.Disable(ctx, "blocked", time.Hour); err != nil {
		t.Fatal(err)
	}
	assertGFResponse(t, requestGF(s, http.MethodPost, "/login", "username=blocked&password=123456", ""), http.StatusForbidden, gfdt.CodeAccountDisabled)
	pair := decodeGFPair(t, requestGF(s, http.MethodPost, "/login", "username=alice&password=123456", ""))
	if err := mgr.DisableDeviceAndDeviceID(ctx, "alice", "web", "gf-example", time.Hour); err != nil {
		t.Fatal(err)
	}
	assertGFResponse(t, requestGF(s, http.MethodPost, "/login", "username=alice&password=123456", ""), http.StatusForbidden, gfdt.CodeAccountDisabled)
	assertGFResponse(t, requestGF(s, http.MethodPost, "/refresh", "refreshToken="+pair.RefreshToken, ""), http.StatusForbidden, gfdt.CodeAccountDisabled)
	if err := mgr.UntieDeviceAndDeviceID(ctx, "alice", "web", "gf-example"); err != nil {
		t.Fatal(err)
	}
	decodeGFPair(t, requestGF(s, http.MethodPost, "/refresh", "refreshToken="+pair.RefreshToken, ""))
}

// TestStorageFailuresStayServerErrors verifies handler and middleware failures do not leak backend details. TestStorageFailuresStayServerErrors 验证处理器和中间件均将存储故障作为服务端错误且不泄露详情。
func TestStorageFailuresStayServerErrors(t *testing.T) {
	gfdt.DeleteAllManager()
	t.Cleanup(gfdt.DeleteAllManager)
	mgr, err := gfdt.NewBuilder().SetStorage(&failingGFStorage{Storage: gfdt.NewMemoryStorage()}).
		AutoRenew(false).AsyncEvent(false).IsPrintBanner(false).Build()
	if err != nil {
		t.Fatal(err)
	}
	gfdt.SetManager(mgr)
	for _, withAuth := range []bool{false, true} {
		t.Run(map[bool]string{false: "handlers", true: "middleware"}[withAuth], func(t *testing.T) {
			register := registerRoutes
			if !withAuth {
				// Reach each handler directly to verify its own error mapping. 直接到达各处理器，验证其自身的错误分类。
				register = func(s *ghttp.Server) {
					s.Use(gfdt.RegisterDTokenContextMiddleware(context.Background()))
					s.BindHandler("POST:/login", handleLogin)
					s.BindHandler("POST:/refresh", handleRefresh)
					s.BindHandler("GET:/me", handleMe)
					s.BindHandler("GET:/introspect", handleIntrospect)
					s.BindHandler("POST:/logout", handleLogout)
				}
			}
			s := newGFServer(t, register)
			for _, tc := range []struct{ method, path, body string }{
				{http.MethodPost, "/login", "username=alice&password=123456"},
				{http.MethodPost, "/refresh", "refreshToken=some-token"},
				{http.MethodGet, "/me", ""},
				{http.MethodGet, "/introspect", ""},
				{http.MethodPost, "/logout", ""},
			} {
				rec := requestGF(s, tc.method, tc.path, tc.body, "some-token")
				assertGFResponse(t, rec, http.StatusInternalServerError, gfdt.CodeServerError)
				if strings.Contains(rec.Body.String(), "private storage") || !strings.Contains(rec.Body.String(), "internal server error") {
					t.Fatalf("unsafe server response: %s", rec.Body.String())
				}
			}
			if withAuth {
				for _, path := range []string{"/access/me", "/access/articles", "/access/admin"} {
					rec := requestGF(s, http.MethodGet, path, "", "some-token")
					assertGFResponse(t, rec, http.StatusInternalServerError, gfdt.CodeServerError)
					if strings.Contains(rec.Body.String(), "private storage") {
						t.Fatalf("unsafe access middleware response: %s", rec.Body.String())
					}
				}
			}
		})
	}
}

type failingGFStorage struct{ gfdt.Storage }

// Get simulates a storage failure containing private connection details. Get 模拟包含私有连接详情的存储故障。
func (*failingGFStorage) Get(context.Context, string) (any, error) {
	return nil, errors.New("private storage connection details")
}

func setupGFManager(t *testing.T) *gfdt.Manager {
	t.Helper()
	gfdt.DeleteAllManager()
	t.Cleanup(gfdt.DeleteAllManager)
	initDToken()
	mgr, err := gfdt.GetManager()
	if err != nil {
		t.Fatal(err)
	}
	return mgr
}

// newGFServer initializes real routing on a temporary loopback port for ServeHTTP tests. newGFServer 在本机临时端口初始化真实路由供 ServeHTTP 测试使用。
func newGFServer(t *testing.T, register func(*ghttp.Server)) *ghttp.Server {
	t.Helper()
	s := ghttp.GetServer(guid.S())
	s.SetAddr("127.0.0.1:0")
	s.SetDumpRouterMap(false)
	s.SetLogStdout(false)
	s.SetAccessLogEnabled(false)
	s.SetErrorLogEnabled(false)
	s.SetSessionStorage(gsession.NewStorageMemory())
	register(s)
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Shutdown(); err != nil {
			t.Error(err)
		}
	})
	return s
}

func requestGF(s *ghttp.Server, method, path, body, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "http://example.com"+path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	return rec
}

func assertGFResponse(t *testing.T, rec *httptest.ResponseRecorder, status, code int) json.RawMessage {
	t.Helper()
	var response struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("invalid JSON response: %v, body=%s", err, rec.Body.String())
	}
	if rec.Code != status || response.Code != code || !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("response=%d %s %s, want status=%d code=%d", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String(), status, code)
	}
	return response.Data
}

func decodeGFPair(t *testing.T, rec *httptest.ResponseRecorder) gfdt.RefreshTokenPair {
	t.Helper()
	data := assertGFResponse(t, rec, http.StatusOK, gfdt.CodeSuccess)
	var pair gfdt.RefreshTokenPair
	if err := json.Unmarshal(data, &pair); err != nil {
		t.Fatal(err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatalf("empty token pair: %+v", pair)
	}
	return pair
}
