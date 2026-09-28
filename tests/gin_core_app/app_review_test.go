package gin_core_app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/com/storage/memory"
	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/dtoken"
	"github.com/gin-gonic/gin"
)

// newReviewApp creates an isolated fixture with deterministic renewal settings. newReviewApp 创建续期设置固定的隔离测试应用。
func newReviewApp(t *testing.T) *App {
	t.Helper()
	app, err := NewApp(Config{ActiveTimeout: -1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Close)
	return app
}

// reviewRequest checks HTTP and business status without opening a listener. reviewRequest 不启动监听器即可检查 HTTP 和业务状态。
func reviewRequest(t *testing.T, app *App, method, path, body, token string, wantStatus, wantCode int) Response {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	app.Router().ServeHTTP(rec, req)
	var response Response
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("%s %s: invalid response: %v, body=%s", method, path, err, rec.Body.String())
	}
	if rec.Code != wantStatus || response.Code != wantCode {
		t.Fatalf("%s %s: status=%d response=%+v, want %d/%d", method, path, rec.Code, response, wantStatus, wantCode)
	}
	if wantStatus >= http.StatusBadRequest && response.Data != nil {
		t.Fatalf("%s %s: error response contains success data: %+v", method, path, response)
	}
	return response
}

// reviewLogin issues a token for protected handler checks. reviewLogin 为受保护处理器检查签发 Token。
func reviewLogin(t *testing.T, app *App) string {
	t.Helper()
	token, err := app.auth.Login(context.Background(), dtoken.LoginOptions{LoginID: "reader", Device: "web", DeviceID: "browser"})
	if err != nil {
		t.Fatal(err)
	}
	return token
}

// TestLoginRejectsMalformedJSONBeforeIssuingTokens covers all login routes. TestLoginRejectsMalformedJSONBeforeIssuingTokens 覆盖全部登录路由的签发前 JSON 校验。
func TestLoginRejectsMalformedJSONBeforeIssuingTokens(t *testing.T) {
	app := newReviewApp(t)
	for _, tc := range []struct {
		path string
		auth *dtoken.Auth
	}{
		{"/login", app.auth},
		{"/login/timeout", app.auth},
		{"/multi-auth/user/login", app.userAuth},
		{"/multi-auth/admin/login", app.adminAuth},
	} {
		t.Run(tc.path, func(t *testing.T) {
			loginID := "malformed" + strings.ReplaceAll(tc.path, "/", "-")
			valid := fmt.Sprintf(`{"username":%q,"password":"123456","seconds":60}`, loginID)
			for _, body := range []string{valid + " {}", valid + " null", valid + " trailing", "null", "[]", "{", ""} {
				reviewRequest(t, app, http.MethodPost, tc.path, body, "", http.StatusBadRequest, derror.CodeBadRequest)
			}
			if _, err := tc.auth.GetSession(context.Background(), loginID); !errors.Is(err, derror.ErrSessionNotFound) {
				t.Fatalf("invalid JSON created a session: %v", err)
			}
			reviewRequest(t, app, http.MethodPost, tc.path, valid+"\n\t", "", http.StatusOK, derror.CodeSuccess)
		})
	}
}

// TestDisableRejectsMalformedJSONWithoutChangingState covers optional and required bodies. TestDisableRejectsMalformedJSONWithoutChangingState 覆盖可选和必填请求体，确保错误输入不修改封禁状态。
func TestDisableRejectsMalformedJSONWithoutChangingState(t *testing.T) {
	for _, path := range []string{"/api/disable/account", "/api/disable/service/payment", "/api/disable/service/payment/level", "/api/disable/device/web", "/api/disable/device/web/browser"} {
		t.Run(path, func(t *testing.T) {
			app := newReviewApp(t)
			token := reviewLogin(t, app)
			for _, body := range []string{
				`{"level":3,"reason":42}`,
				`{"level":3,"reason":"risk"} {}`,
				`{"level":3,"reason":"risk"} trailing`,
				`{"level":3,`, "null", "[]",
			} {
				reviewRequest(t, app, http.MethodPost, path, body, token, http.StatusBadRequest, derror.CodeBadRequest)
			}
			ctx := context.Background()
			mgr := app.Manager()
			if _, err := mgr.GetDisableInfo(ctx, "reader"); !errors.Is(err, derror.ErrAccountNotDisabled) {
				t.Fatalf("account state changed: %v", err)
			}
			if _, err := mgr.GetDisableServiceInfo(ctx, "reader", "payment"); !errors.Is(err, derror.ErrServiceNotDisabled) {
				t.Fatalf("service state changed: %v", err)
			}
			if _, err := mgr.GetDisableDeviceInfo(ctx, "reader", "web"); !errors.Is(err, derror.ErrDeviceNotDisabled) {
				t.Fatalf("device state changed: %v", err)
			}
			if _, err := mgr.GetDisableDeviceAndDeviceIDInfo(ctx, "reader", "web", "browser"); !errors.Is(err, derror.ErrDeviceNotDisabled) {
				t.Fatalf("concrete device state changed: %v", err)
			}
			body := ""
			if strings.HasSuffix(path, "/level") {
				reviewRequest(t, app, http.MethodPost, path, body, token, http.StatusBadRequest, derror.CodeBadRequest)
				body = `{"level":3}`
			}
			reviewRequest(t, app, http.MethodPost, path, body, token, http.StatusOK, derror.CodeSuccess)
		})
	}
}

// TestLifetimeOverflowIsRejected covers each seconds-to-duration HTTP boundary. TestLifetimeOverflowIsRejected 覆盖所有秒数转时长的 HTTP 边界。
func TestLifetimeOverflowIsRejected(t *testing.T) {
	app := newReviewApp(t)
	token := reviewLogin(t, app)
	for _, path := range []string{"/login/timeout", "/api/token/renew", "/nonce/timeout"} {
		t.Run(path, func(t *testing.T) {
			for _, seconds := range []int64{0, -1, math.MaxInt64/int64(time.Second) + 1, 18446744074, math.MaxInt64} {
				body := fmt.Sprintf(`{"username":"overflow-user","password":"123456","seconds":%d}`, seconds)
				reviewRequest(t, app, http.MethodPost, path, body, token, http.StatusBadRequest, derror.CodeBadRequest)
			}
			if _, err := app.auth.GetSession(context.Background(), "overflow-user"); !errors.Is(err, derror.ErrSessionNotFound) {
				t.Fatalf("invalid lifetime issued a token: %v", err)
			}
			if err := app.auth.CheckLogin(context.Background(), token); err != nil {
				t.Fatalf("invalid renewal invalidated the existing token: %v", err)
			}
		})
	}
	for _, path := range []string{"/login/timeout", "/api/token/renew", "/nonce/timeout"} {
		reviewRequest(t, app, http.MethodPost, path, `{"username":"valid-user","password":"123456","seconds":60}`, token, http.StatusOK, derror.CodeSuccess)
	}
}

// TestMePropagatesAccessProviderErrors verifies failures cannot masquerade as empty access lists. TestMePropagatesAccessProviderErrors 验证提供器错误不会伪装成空权限列表。
func TestMePropagatesAccessProviderErrors(t *testing.T) {
	for _, stage := range []string{"roles", "permissions"} {
		t.Run(stage, func(t *testing.T) {
			provider := manager.AccessProviderFunc{
				RoleFunc: func(context.Context, manager.AccessSubject) ([]string, error) {
					if stage == "roles" {
						return nil, errors.New("private role backend failure")
					}
					return []string{}, nil
				},
				PermissionFunc: func(context.Context, manager.AccessSubject) ([]string, error) {
					return nil, errors.New("private permission backend failure")
				},
			}
			mgr, err := dtoken.NewBuilder().AutoRenew(false).ActiveTimeout(-1).AsyncEvent(false).
				IsLog(false).IsPrintBanner(false).SetAccessProvider(provider).Build()
			if err != nil {
				t.Fatal(err)
			}
			auth := dtoken.New(mgr)
			app := &App{auth: auth, userAuth: auth, adminAuth: auth}
			t.Cleanup(app.Close)
			app.router = app.buildRouter()
			token := reviewLogin(t, app)
			for _, path := range []string{"/api/me", "/multi-auth/user/me", "/multi-auth/admin/me"} {
				response := reviewRequest(t, app, http.MethodGet, path, "", token, http.StatusInternalServerError, derror.CodeServerError)
				if response.Message != "internal server error" {
					t.Fatalf("backend error leaked: %+v", response)
				}
			}
		})
	}
}

// reviewStorage injects service lookup failures and observes resource ownership. reviewStorage 注入服务查询错误并观察资源所有权。
type reviewStorage struct {
	adapter.Storage
	failService bool // failService fails only service-disable reads. failService 仅使服务封禁查询失败。
	closeCalls  int  // closeCalls counts resource releases. closeCalls 记录资源释放次数。
}

// Get optionally fails service-disable reads. Get 可按需使服务封禁查询失败。
func (s *reviewStorage) Get(ctx context.Context, key string) (any, error) {
	if s.failService && strings.Contains(key, manager.DisableServiceKeyPrefix) {
		return nil, errors.New("private storage failure")
	}
	return s.Storage.Get(ctx, key)
}

// Close records cleanup calls. Close 记录清理调用。
func (s *reviewStorage) Close() error {
	s.closeCalls++
	return nil
}

// TestServiceStatusPropagatesStorageErrors separates lookup failures from disable decisions. TestServiceStatusPropagatesStorageErrors 区分查询错误与封禁结果。
func TestServiceStatusPropagatesStorageErrors(t *testing.T) {
	storage := &reviewStorage{Storage: memory.NewStorage()}
	mgr, err := newDemoManager(Config{ActiveTimeout: -1}, "service-review", storage)
	if err != nil {
		t.Fatal(err)
	}
	auth := dtoken.New(mgr)
	app := &App{auth: auth, userAuth: auth, adminAuth: auth}
	t.Cleanup(app.Close)
	app.router = app.buildRouter()
	token := reviewLogin(t, app)
	path := "/api/disable/service/payment/level/1"
	response := reviewRequest(t, app, http.MethodGet, path, "", token, http.StatusOK, derror.CodeSuccess)
	if response.Data.(map[string]any)["disabled"] != false {
		t.Fatalf("enabled service reported disabled: %+v", response)
	}
	reviewRequest(t, app, http.MethodPost, "/api/disable/service/payment/level", `{"level":1}`, token, http.StatusOK, derror.CodeSuccess)
	response = reviewRequest(t, app, http.MethodGet, path, "", token, http.StatusOK, derror.CodeSuccess)
	if response.Data.(map[string]any)["disabled"] != true {
		t.Fatalf("disabled service reported enabled: %+v", response)
	}
	storage.failService = true
	reviewRequest(t, app, http.MethodGet, path, "", token, http.StatusInternalServerError, derror.CodeServerError)
}

// TestNewDemoManagerStorageOwnership checks cleanup after success and configuration failure. TestNewDemoManagerStorageOwnership 检查构建成功与配置失败后的清理行为。
func TestNewDemoManagerStorageOwnership(t *testing.T) {
	for _, timeout := range []time.Duration{0, -time.Nanosecond, -time.Second} {
		t.Run(timeout.String(), func(t *testing.T) {
			storage := &reviewStorage{Storage: memory.NewStorage()}
			mgr, err := newDemoManager(Config{TokenTimeout: timeout, ActiveTimeout: -1}, "ownership-review", storage)
			if err == nil {
				t.Cleanup(mgr.CloseManager)
			}
			if timeout < 0 {
				if err == nil || mgr != nil || storage.closeCalls != 1 {
					t.Fatalf("invalid config: manager=%v err=%v closes=%d", mgr, err, storage.closeCalls)
				}
				return
			}
			if err != nil || storage.closeCalls != 0 {
				t.Fatalf("valid config: err=%v closes=%d", err, storage.closeCalls)
			}
			if mgr.GetConfig().Timeout != 30 {
				t.Fatalf("default timeout=%d, want 30", mgr.GetConfig().Timeout)
			}
			mgr.CloseManager()
			mgr.CloseManager()
			if storage.closeCalls != 1 {
				t.Fatalf("storage closed %d times, want 1", storage.closeCalls)
			}
		})
	}
}

// TestNewAppPreservesGinMode prevents process-global changes during fixture construction. TestNewAppPreservesGinMode 防止构建测试应用时修改进程级模式。
func TestNewAppPreservesGinMode(t *testing.T) {
	original := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(original) })
	newReviewApp(t)
	if gin.Mode() != gin.TestMode {
		t.Fatalf("NewApp changed Gin mode to %q", gin.Mode())
	}
}

// TestErrorMappingKeepsBusinessFailuresOutOfHTTP500 covers wrapped error classification. TestErrorMappingKeepsBusinessFailuresOutOfHTTP500 覆盖包装错误分类，避免业务错误误报 HTTP 500。
func TestErrorMappingKeepsBusinessFailuresOutOfHTTP500(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
		code   int
	}{
		{derror.ErrClientMismatch, http.StatusBadRequest, derror.CodeBadRequest},
		{derror.ErrRedirectURIMismatch, http.StatusBadRequest, derror.CodeBadRequest},
		{derror.ErrClientOrClientIDEmpty, http.StatusBadRequest, derror.CodeBadRequest},
		{derror.ErrIDIsEmpty, http.StatusBadRequest, derror.CodeBadRequest},
		{derror.ErrEmptyLoginID, http.StatusBadRequest, derror.CodeBadRequest},
		{derror.ErrUserIDEmpty, http.StatusBadRequest, derror.CodeBadRequest},
		{derror.ErrAccountNotDisabled, http.StatusNotFound, derror.CodeNotFound},
		{derror.ErrServiceNotDisabled, http.StatusNotFound, derror.CodeNotFound},
		{derror.ErrDeviceNotDisabled, http.StatusNotFound, derror.CodeNotFound},
		{derror.ErrTokenKickout, http.StatusUnauthorized, derror.CodeNotLogin},
		{derror.ErrStorageUnavailable, http.StatusInternalServerError, derror.CodeServerError},
	} {
		t.Run(tc.err.Error(), func(t *testing.T) {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			writeDTokenError(c, fmt.Errorf("wrapped: %w", tc.err))
			var response Response
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if rec.Code != tc.status || response.Code != tc.code {
				t.Fatalf("status=%d response=%+v, want %d/%d", rec.Code, response, tc.status, tc.code)
			}
			if tc.status == http.StatusInternalServerError && response.Message != "internal server error" {
				t.Fatalf("internal error leaked: %+v", response)
			}
		})
	}
}

// TestBearerTokenPreservesRawTokensAndAcceptsSchemeCase checks header compatibility. TestBearerTokenPreservesRawTokensAndAcceptsSchemeCase 检查原始 Token 与认证方案大小写兼容性。
func TestBearerTokenPreservesRawTokensAndAcceptsSchemeCase(t *testing.T) {
	for _, tc := range []struct{ header, token string }{
		{"raw-token", "raw-token"}, {" Bearer token ", "token"}, {"bearer token", "token"},
		{"BEARER   token", "token"}, {"", ""}, {"Bearer ", ""}, {"Basic token", ""}, {"Bearer a b", ""},
	} {
		if got := bearerToken(tc.header); got != tc.token {
			t.Errorf("bearerToken(%q)=%q, want %q", tc.header, got, tc.token)
		}
	}
}
