package gin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/com/storage/memory"
	"github.com/Zany2/dtoken-go/core/adapter"
	corecontext "github.com/Zany2/dtoken-go/core/context"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/dtoken"
	gingonic "github.com/gin-gonic/gin"
)

// ginReviewChecks exercises every authentication entry point through real Gin handler chains. ginReviewChecks 使用真实 Gin 处理链覆盖各鉴权入口。
var ginReviewChecks = []struct {
	name  string
	build func(context.Context, ...AuthOption) gingonic.HandlerFunc
}{
	{"login", AuthMiddleware},
	{"access", AccessMiddleware},
	{"permission", func(ctx context.Context, opts ...AuthOption) gingonic.HandlerFunc {
		return PermissionMiddleware(ctx, []string{"read"}, opts...)
	}},
	{"role", func(ctx context.Context, opts ...AuthOption) gingonic.HandlerFunc {
		return RoleMiddleware(ctx, []string{"reader"}, opts...)
	}},
	{"annotation", func(ctx context.Context, opts ...AuthOption) gingonic.HandlerFunc {
		options := defaultAuthOptions()
		for _, opt := range opts {
			opt(options)
		}
		return GetHandler(ctx, nil, options.FailFunc, &Annotation{
			AuthType: options.AuthType, CheckLogin: true, CheckPermission: []string{"read"},
		})
	}},
}

// newGinReviewManager creates a deterministic manager with synchronous events and no automatic renewal. newGinReviewManager 创建使用同步事件且不自动续期的测试 Manager。
func newGinReviewManager(t *testing.T, authType string, storage adapter.Storage) *manager.Manager {
	t.Helper()
	b := NewBuilder().IsPrintBanner(false).IsLog(false).AutoRenew(false).AsyncEvent(false).Timeout(600)
	if authType != "" {
		b.AuthType(authType)
	}
	if storage != nil {
		b.SetStorage(storage)
	}
	mgr, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mgr.CloseManager)
	return mgr
}

// TestGinReviewRequestContextAndManager verifies request values/cancellation and inherited explicit managers. TestGinReviewRequestContextAndManager 验证请求上下文值、取消信号及显式 Manager 的继承。
func TestGinReviewRequestContextAndManager(t *testing.T) {
	gingonic.SetMode(gingonic.TestMode)
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	for _, check := range ginReviewChecks {
		t.Run(check.name, func(t *testing.T) {
			storage := &ginReviewContextStorage{Storage: memory.NewStorage()}
			mgr := newGinReviewManager(t, "request-scoped", storage)
			token, err := mgr.Login(context.Background(), "reader")
			if err != nil {
				t.Fatal(err)
			}
			if err := mgr.AddPermissions(context.Background(), "reader", []string{"read"}); err != nil {
				t.Fatal(err)
			}
			if err := mgr.AddRoles(context.Background(), "reader", []string{"reader"}); err != nil {
				t.Fatal(err)
			}
			var expected context.Context
			reads, hooks, handled := 0, 0, 0
			storage.inspect = func(ctx context.Context) {
				reads++
				if ctx != expected {
					t.Error("storage did not receive the current HTTP request context")
				}
			}
			inspectHook := func(ctx context.Context) {
				hooks++
				if ctx != expected {
					t.Error("hook did not receive the current HTTP request context")
				}
			}
			router := gingonic.New()
			router.Use(RegisterDTokenContextMiddleware(context.Background(), WithManager(mgr)))
			router.Use(check.build(context.Background(),
				WithBeforeAuthHandler(func(ctx context.Context, _ *gingonic.Context, _ *AuthHandleRequest) { inspectHook(ctx) }),
				WithRouteAccessHandler(func(ctx context.Context, _ *gingonic.Context, _ *RouteAccessRequest) { inspectHook(ctx) }),
			))
			router.GET("/", func(c *gingonic.Context) { handled++; c.Status(http.StatusNoContent) })
			for _, canceled := range []bool{false, true} {
				// Distinct request contexts also catch accidental reuse across requests. 不同请求上下文同时验证不会跨请求复用。
				type requestKey struct{}
				requestCtx, cancel := context.WithTimeout(context.WithValue(context.Background(), requestKey{}, canceled), time.Minute)
				t.Cleanup(cancel)
				if canceled {
					cancel()
				}
				expected = requestCtx
				reads, hooks, handled = 0, 0, 0
				req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(requestCtx)
				req.Header.Set(mgr.GetConfig().TokenName, token)
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)
				if reads == 0 || check.name != "annotation" && hooks != 1 {
					t.Fatalf("request did not reach expected storage/hook: reads=%d hooks=%d", reads, hooks)
				}
				if canceled {
					if handled != 0 || rec.Code < 400 {
						t.Fatalf("canceled storage request reached handler: handled=%d status=%d", handled, rec.Code)
					}
				} else if handled != 1 || rec.Code != http.StatusNoContent {
					t.Fatalf("valid request with injected manager failed: handled=%d status=%d", handled, rec.Code)
				}
			}
		})
	}
}

// ginReviewContextStorage observes authentication contexts and honors cancellation like remote storage. ginReviewContextStorage 观察鉴权上下文并模拟远程存储响应取消信号。
type ginReviewContextStorage struct {
	adapter.Storage
	inspect func(context.Context)
}

func (s *ginReviewContextStorage) Get(ctx context.Context, key string) (any, error) {
	if s.inspect != nil {
		s.inspect(ctx)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.Storage.Get(ctx, key)
}

// TestGinReviewFailureCallbackCannotResume rejects requests before invoking custom failure handlers. TestGinReviewFailureCallbackCannotResume 验证失败回调不能恢复受保护处理链。
func TestGinReviewFailureCallbackCannotResume(t *testing.T) {
	gingonic.SetMode(gingonic.TestMode)
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	mgr := newGinReviewManager(t, "failure-callback", nil)
	for _, check := range ginReviewChecks {
		for _, missingManager := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/missing-manager=%t", check.name, missingManager), func(t *testing.T) {
				handled, failures := 0, 0
				router := gingonic.New()
				if !missingManager {
					router.Use(RegisterDTokenContextMiddleware(context.Background(), WithManager(mgr)))
				}
				router.Use(check.build(context.Background(), WithFailFunc(func(c *gingonic.Context, _ error) {
					failures++
					if !c.IsAborted() {
						t.Error("failure callback ran before request abort")
					}
					c.Next()
					c.Status(http.StatusUnauthorized)
				})))
				router.GET("/", func(*gingonic.Context) { handled++ })
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
				if handled != 0 || failures != 1 || rec.Code != http.StatusUnauthorized {
					t.Fatalf("failure chain: handled=%d failures=%d status=%d", handled, failures, rec.Code)
				}
			})
		}
	}
}

// TestGinReviewRegistrationFailureCannotResume covers the context-only middleware's error path. TestGinReviewRegistrationFailureCannotResume 覆盖仅注册上下文的中间件失败路径。
func TestGinReviewRegistrationFailureCannotResume(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	handled, failures := 0, 0
	router := gingonic.New()
	router.Use(RegisterDTokenContextMiddleware(context.Background(), WithFailFunc(func(c *gingonic.Context, _ error) {
		failures++
		c.Next()
		c.Status(http.StatusNotFound)
	})))
	router.GET("/", func(*gingonic.Context) { handled++ })
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if handled != 0 || failures != 1 || rec.Code != http.StatusNotFound {
		t.Fatalf("registration failure: handled=%d failures=%d status=%d", handled, failures, rec.Code)
	}
}

// TestGinReviewManagerSelection preserves explicit overrides and rejects a closed cached manager. TestGinReviewManagerSelection 保留显式覆盖优先级并拒绝已关闭的缓存 Manager。
func TestGinReviewManagerSelection(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	global := newGinReviewManager(t, "", nil)
	local := newGinReviewManager(t, "local-selection", nil)
	dtoken.SetManager(global)
	const token = "selection-token"
	for _, mgr := range []*manager.Manager{global, local} {
		if _, err := mgr.LoginWithOptions(context.Background(), manager.LoginOptions{LoginID: "reader", Token: token}); err != nil {
			t.Fatal(err)
		}
		if err := mgr.AddPermissions(context.Background(), "reader", []string{"read"}); err != nil {
			t.Fatal(err)
		}
		if err := mgr.AddRoles(context.Background(), "reader", []string{"reader"}); err != nil {
			t.Fatal(err)
		}
	}
	for _, check := range ginReviewChecks {
		for _, selection := range []struct {
			name string
			opts []AuthOption
			want *manager.Manager
		}{
			{"cached", nil, local},
			{"auth type", []AuthOption{WithAuthType(global.GetConfig().AuthType)}, global},
			{"explicit manager", []AuthOption{WithManager(global), WithAuthType("missing")}, global},
		} {
			if check.name == "annotation" && selection.name == "explicit manager" {
				continue
			}
			t.Run(check.name+"/"+selection.name, func(t *testing.T) {
				router := gingonic.New()
				router.Use(RegisterDTokenContextMiddleware(context.Background(), WithManager(local)))
				router.Use(check.build(context.Background(), selection.opts...))
				var selected *manager.Manager
				router.GET("/", func(c *gingonic.Context) {
					var err error
					selected, err = GetManagerByContext(c)
					if err != nil {
						t.Error(err)
					}
					c.Status(http.StatusNoContent)
				})
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.Header.Set(local.GetConfig().TokenName, token)
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)
				if rec.Code != http.StatusNoContent || selected != selection.want {
					t.Fatalf("manager selection: status=%d selected=%p want=%p", rec.Code, selected, selection.want)
				}
			})
		}
	}
	c := newGinTestContext(http.MethodGet, "/", local.GetConfig().TokenName, token)
	getDContext(c, local)
	local.CloseManager()
	var gotErr error
	AuthMiddleware(context.Background(), WithFailFunc(func(_ *gingonic.Context, err error) { gotErr = err }))(c)
	if !c.IsAborted() || !errors.Is(gotErr, ErrManagerNotFound) {
		t.Fatalf("closed cached manager fell back to the global instance: aborted=%v err=%v", c.IsAborted(), gotErr)
	}
}

// TestGinReviewHooksHonorNativeAbort avoids duplicate responses after hooks finish a request. TestGinReviewHooksHonorNativeAbort 避免钩子结束请求后鉴权继续追加响应。
func TestGinReviewHooksHonorNativeAbort(t *testing.T) {
	gingonic.SetMode(gingonic.TestMode)
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	for _, check := range ginReviewChecks {
		if check.name == "annotation" {
			continue
		}
		t.Run(check.name, func(t *testing.T) {
			handled, failures := 0, 0
			router := gingonic.New()
			router.Use(check.build(context.Background(),
				WithBeforeAuthHandler(func(_ context.Context, c *gingonic.Context, _ *AuthHandleRequest) {
					c.AbortWithStatusJSON(http.StatusForbidden, gingonic.H{"error": "hook"})
				}),
				WithRouteAccessHandler(func(_ context.Context, c *gingonic.Context, _ *RouteAccessRequest) {
					c.AbortWithStatusJSON(http.StatusForbidden, gingonic.H{"error": "hook"})
				}),
				WithFailFunc(func(*gingonic.Context, error) { failures++ }),
			))
			router.GET("/", func(*gingonic.Context) { handled++ })
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
			if handled != 0 || failures != 0 || rec.Code != http.StatusForbidden || rec.Body.String() != `{"error":"hook"}` {
				t.Fatalf("aborted hook: handled=%d failures=%d response=%d %s", handled, failures, rec.Code, rec.Body.String())
			}
		})
	}
}

// TestGinReviewRenewalUsesRequestManager protects identical token values in different auth namespaces. TestGinReviewRenewalUsesRequestManager 保护不同认证空间内同值 Token 的续期归属。
func TestGinReviewRenewalUsesRequestManager(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	global := newGinReviewManager(t, "", nil)
	local := newGinReviewManager(t, "local-renewal", nil)
	dtoken.SetManager(global)
	const token = "shared-token-value"
	for _, mgr := range []*manager.Manager{global, local} {
		if _, err := mgr.LoginWithOptions(context.Background(), manager.LoginOptions{LoginID: "renew", Token: token}); err != nil {
			t.Fatal(err)
		}
	}
	c := newGinTestContext(http.MethodGet, "/", local.GetConfig().TokenName, token)
	RegisterDTokenContextMiddleware(context.Background(), WithManager(local))(c)
	if err := RenewTimeoutByContext(c, time.Minute); err != nil {
		t.Fatal(err)
	}
	for _, input := range []struct {
		mgr  *manager.Manager
		want int64
	}{{global, 600}, {local, 60}} {
		info, err := input.mgr.GetTokenInfo(context.Background(), token)
		if err != nil || info.Timeout != input.want {
			t.Fatalf("renewal metadata = %+v, %v; want timeout %d", info, err, input.want)
		}
	}
}

// TestGinReviewCachedContextValidation rejects unusable cached managers without switching authentication domains. TestGinReviewCachedContextValidation 拒绝不可用缓存 Manager，避免切换认证空间。
func TestGinReviewCachedContextValidation(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	global := newGinReviewManager(t, "", nil)
	dtoken.SetManager(global)
	c := newGinTestContext(http.MethodGet, "/", "", "")
	c.Set(DTokenCtxKey, (*DTokenContext)(nil))
	if value, ok := GetDTokenContext(c); value != nil || ok {
		t.Fatal("typed nil cache was reported as a usable context")
	}
	if mgr, err := GetManagerByContext(c); err != nil || mgr != global {
		t.Fatalf("missing cache fallback = %v, %v", mgr, err)
	}
	for _, missing := range []bool{true, false} {
		var mgr *manager.Manager
		if !missing {
			mgr = newGinReviewManager(t, "closed-cache", nil)
			mgr.CloseManager()
		}
		c.Set(DTokenCtxKey, corecontext.NewContext(NewGinContext(c), mgr))
		if got, err := GetManagerByContext(c); got != nil || !errors.Is(err, ErrManagerNotFound) {
			t.Fatalf("unusable cached manager = %v, %v", got, err)
		}
		if _, err := LoginByContext(c, "should-not-login"); !errors.Is(err, ErrManagerNotFound) {
			t.Fatalf("login with unusable cached manager = %v", err)
		}
	}
}

// TestGinReviewNativeAbortAndCookieIsolation preserves Gin state outside the adapter. TestGinReviewNativeAbortAndCookieIsolation 验证适配器遵循 Gin 中止状态且不污染业务 Cookie 设置。
func TestGinReviewNativeAbortAndCookieIsolation(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gingonic.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	first, second := NewGinContext(c), NewGinContext(c)
	c.Abort()
	if !first.IsAborted() || !second.IsAborted() {
		t.Fatal("adapters did not observe Gin's native abort")
	}
	c.SetSameSite(http.SameSiteStrictMode)
	first.SetCookieWithOptions(&adapter.CookieOptions{Name: "token", Value: "a+b c", SameSite: "None", Secure: true, HttpOnly: true})
	first.SetCookie("legacy", "value", -1, "/", "", false, true)
	c.SetCookie("application", "value", 0, "/", "", true, true)
	cookies := rec.Result().Cookies()
	if len(cookies) != 3 || cookies[0].SameSite != http.SameSiteNoneMode || cookies[1].SameSite != http.SameSiteLaxMode || cookies[2].SameSite != http.SameSiteStrictMode {
		t.Fatalf("cookie SameSite isolation failed: %v", rec.Header().Values("Set-Cookie"))
	}
	if cookies[0].Path != "/" || !cookies[0].Secure || !cookies[0].HttpOnly || cookies[1].MaxAge != -1 {
		t.Fatalf("cookie options or deletion changed: %v", rec.Header().Values("Set-Cookie"))
	}
	c.Request.AddCookie(cookies[0])
	if value := first.GetCookie("token"); value != "a+b c" {
		t.Fatalf("cookie value did not round trip: %q", value)
	}
}

// TestGinReviewNilAnnotationRejectsRequest reports invalid configuration without panic or bypass. TestGinReviewNilAnnotationRejectsRequest 验证空注解返回配置错误且不会 panic 或放行。
func TestGinReviewNilAnnotationRejectsRequest(t *testing.T) {
	router := gingonic.New()
	handled := false
	router.GET("/", GetHandler(context.Background(), nil, nil, nil), func(*gingonic.Context) { handled = true })
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if handled || rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "code") {
		t.Fatalf("nil annotation: handled=%v status=%d body=%s", handled, rec.Code, rec.Body.String())
	}
}
