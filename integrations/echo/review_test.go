package echo

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/com/storage/memory"
	"github.com/Zany2/dtoken-go/core/adapter"
	corecontext "github.com/Zany2/dtoken-go/core/context"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/dtoken"
	echo4 "github.com/labstack/echo/v4"
)

// echoReviewChecks exercises middleware and annotations with the same HTTP scenarios. echoReviewChecks 使用相同 HTTP 场景覆盖中间件与注解。
var echoReviewChecks = []struct {
	name  string
	build func(context.Context, ...AuthOption) echo4.MiddlewareFunc
}{
	{"login", AuthMiddleware},
	{"access", AccessMiddleware},
	{"permission", func(ctx context.Context, opts ...AuthOption) echo4.MiddlewareFunc {
		return PermissionMiddleware(ctx, []string{"read"}, opts...)
	}},
	{"role", func(ctx context.Context, opts ...AuthOption) echo4.MiddlewareFunc {
		return RoleMiddleware(ctx, []string{"reader"}, opts...)
	}},
	{"annotation", func(ctx context.Context, opts ...AuthOption) echo4.MiddlewareFunc {
		options := defaultAuthOptions()
		for _, opt := range opts {
			opt(options)
		}
		return func(next echo4.HandlerFunc) echo4.HandlerFunc {
			return GetHandler(ctx, next, options.FailFunc, &Annotation{
				AuthType: options.AuthType, CheckLogin: true, CheckPermission: []string{"read"},
			})
		}
	}},
}

// newEchoReviewManager creates an isolated manager with synchronous events. newEchoReviewManager 创建事件同步执行的独立 Manager。
func newEchoReviewManager(t *testing.T, authType string, storage adapter.Storage) *manager.Manager {
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

// echoReviewLogin creates a token with the access rules used by the shared cases. echoReviewLogin 创建具备共用用例所需权限的 Token。
func echoReviewLogin(t *testing.T, mgr *manager.Manager, token string) {
	t.Helper()
	ctx := context.Background()
	if _, err := mgr.LoginWithOptions(ctx, manager.LoginOptions{LoginID: "reader", Token: token}); err != nil {
		t.Fatal(err)
	}
	if err := mgr.AddPermissions(ctx, "reader", []string{"read"}); err != nil {
		t.Fatal(err)
	}
	if err := mgr.AddRoles(ctx, "reader", []string{"reader"}); err != nil {
		t.Fatal(err)
	}
}

// TestEchoReviewRequestContext verifies per-request values, deadlines and cancellation reach hooks and storage. TestEchoReviewRequestContext 验证请求值、截止时间与取消信号传入钩子及存储。
func TestEchoReviewRequestContext(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	for _, check := range echoReviewChecks {
		t.Run(check.name, func(t *testing.T) {
			storage := &echoReviewStorage{Storage: memory.NewStorage()}
			mgr := newEchoReviewManager(t, "echo-request", storage)
			const token = "request-token"
			echoReviewLogin(t, mgr, token)
			var expected context.Context
			reads, hooks, handled := 0, 0, 0
			storage.inspect = func(ctx context.Context) {
				reads++
				if ctx != expected {
					t.Error("storage did not receive the current request context")
				}
			}
			inspectHook := func(ctx context.Context) {
				hooks++
				if ctx != expected {
					t.Error("hook did not receive the current request context")
				}
			}
			e := echo4.New()
			e.Use(RegisterDTokenContextMiddleware(context.Background(), WithManager(mgr)))
			e.Use(check.build(context.Background(),
				WithBeforeAuthHandler(func(ctx context.Context, _ echo4.Context, _ *AuthHandleRequest) { inspectHook(ctx) }),
				WithRouteAccessHandler(func(ctx context.Context, _ echo4.Context, _ *RouteAccessRequest) { inspectHook(ctx) }),
			))
			e.GET("/", func(c echo4.Context) error { handled++; return c.NoContent(http.StatusNoContent) })
			for _, canceled := range []bool{false, true} {
				type requestKey struct{}
				ctx, cancel := context.WithTimeout(context.WithValue(context.Background(), requestKey{}, canceled), time.Minute)
				t.Cleanup(cancel)
				if canceled {
					cancel()
				}
				expected = ctx
				reads, hooks, handled = 0, 0, 0
				r := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
				r.Header.Set(mgr.GetConfig().TokenName, token)
				w := httptest.NewRecorder()
				e.ServeHTTP(w, r)
				if reads == 0 || check.name != "annotation" && hooks != 1 {
					t.Fatalf("reads=%d hooks=%d, expected request checks", reads, hooks)
				}
				if canceled {
					if handled != 0 || w.Code < 400 {
						t.Fatalf("canceled request reached handler: handled=%d status=%d", handled, w.Code)
					}
				} else if handled != 1 || w.Code != http.StatusNoContent {
					t.Fatalf("valid request failed: handled=%d status=%d", handled, w.Code)
				}
			}
		})
	}
}

// echoReviewStorage observes contexts and models a backend that honors cancellation. echoReviewStorage 观察上下文并模拟响应取消信号的存储。
type echoReviewStorage struct {
	adapter.Storage
	inspect func(context.Context)
}

// Get records the context before reading storage. Get 在读取存储前记录上下文。
func (s *echoReviewStorage) Get(ctx context.Context, key string) (any, error) {
	if s.inspect != nil {
		s.inspect(ctx)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.Storage.Get(ctx, key)
}

// TestEchoReviewManagerSelection verifies explicit precedence, inheritance and closed-manager rejection. TestEchoReviewManagerSelection 验证显式配置优先级、请求继承与已关闭 Manager 的拒绝行为。
func TestEchoReviewManagerSelection(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	global := newEchoReviewManager(t, "", nil)
	local := newEchoReviewManager(t, "echo-local", nil)
	closed := newEchoReviewManager(t, "echo-closed", nil)
	closed.CloseManager()
	dtoken.SetManager(global)
	const token = "same-token"
	echoReviewLogin(t, global, token)
	echoReviewLogin(t, local, token)
	for _, check := range echoReviewChecks {
		for _, tc := range []struct {
			name   string
			cached *manager.Manager
			opts   []AuthOption
			want   *manager.Manager
		}{
			{"cached", local, nil, local},
			{"global", nil, nil, global},
			{"auth_type", local, []AuthOption{WithAuthType(global.GetConfig().AuthType)}, global},
			{"explicit", local, []AuthOption{WithManager(global), WithAuthType("missing")}, global},
			{"missing_type", local, []AuthOption{WithAuthType("missing")}, nil},
			{"closed", closed, nil, nil},
		} {
			if check.name == "annotation" && tc.name == "explicit" {
				continue
			}
			t.Run(check.name+"/"+tc.name, func(t *testing.T) {
				e := echo4.New()
				e.Use(func(next echo4.HandlerFunc) echo4.HandlerFunc {
					return func(c echo4.Context) error {
						if tc.cached != nil {
							getDTokenContext(c, tc.cached)
						}
						return next(c)
					}
				})
				e.Use(check.build(context.Background(), tc.opts...))
				called := false
				e.GET("/", func(c echo4.Context) error {
					called = true
					if got, err := GetManagerByContext(c); err != nil || got != tc.want {
						t.Errorf("manager=%p, %v, want %p", got, err, tc.want)
					}
					return c.NoContent(http.StatusNoContent)
				})
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Set(local.GetConfig().TokenName, token)
				w := httptest.NewRecorder()
				e.ServeHTTP(w, r)
				wantStatus := http.StatusNoContent
				if tc.want == nil {
					wantStatus = http.StatusNotFound
				}
				if w.Code != wantStatus || called != (tc.want != nil) {
					t.Fatalf("status=%d called=%v, want status=%d", w.Code, called, wantStatus)
				}
			})
		}
	}
}

// TestEchoReviewRenewalUsesRequestManager protects identical tokens in different auth namespaces. TestEchoReviewRenewalUsesRequestManager 验证不同认证空间内同值 Token 的续期归属。
func TestEchoReviewRenewalUsesRequestManager(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	global := newEchoReviewManager(t, "", nil)
	local := newEchoReviewManager(t, "echo-renew", nil)
	dtoken.SetManager(global)
	const token = "renew-token"
	echoReviewLogin(t, global, token)
	echoReviewLogin(t, local, token)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set(local.GetConfig().TokenName, token)
	c := echo4.New().NewContext(r, httptest.NewRecorder())
	getDTokenContext(c, local)
	if err := RenewTimeoutByContext(c, time.Minute); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		mgr  *manager.Manager
		want int64
	}{{global, 600}, {local, 60}} {
		info, err := tc.mgr.GetTokenInfo(context.Background(), token)
		if err != nil || info == nil || info.Timeout != tc.want {
			t.Fatalf("token info=%+v, %v, want timeout=%d", info, err, tc.want)
		}
	}
}

// TestEchoReviewCachedContextValidation checks typed nils and unusable cached managers. TestEchoReviewCachedContextValidation 检查带类型空值与不可用的缓存 Manager。
func TestEchoReviewCachedContextValidation(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	global := newEchoReviewManager(t, "", nil)
	dtoken.SetManager(global)
	c := echo4.New().NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
	var typedNil *corecontext.DTokenContext
	c.Set(DTokenCtxKey, typedNil)
	if got, ok := GetDTokenContext(c); got != nil || ok {
		t.Fatalf("typed nil lookup=%v,%v", got, ok)
	}
	if got, err := GetManagerByContext(c); err != nil || got != global {
		t.Fatalf("typed nil fallback=%p,%v", got, err)
	}
	c.Set(DTokenCtxKey, typedNil)
	if got := getDTokenContext(c, global); got == nil || got.GetManager() != global {
		t.Fatal("typed nil cache was not replaced")
	}
	closed := newEchoReviewManager(t, "echo-cached-closed", nil)
	closed.CloseManager()
	for _, mgr := range []*manager.Manager{nil, closed} {
		c.Set(DTokenCtxKey, corecontext.NewContext(NewEchoContext(c), mgr))
		_, tokenErr := GetTokenValueByContext(c)
		_, idErr := GetLoginIDByContext(c)
		_, infoErr := IntrospectTokenByContext(c)
		renewErr := RenewTimeoutByContext(c, time.Minute)
		for _, err := range []error{tokenErr, idErr, infoErr, renewErr} {
			if !errors.Is(err, ErrManagerNotFound) {
				t.Errorf("error=%v, want ErrManagerNotFound", err)
			}
		}
		if IsLoginByContext(c) {
			t.Fatal("unusable manager reported a login")
		}
	}
}

// TestEchoReviewHookControlFlow checks termination, at-most-once continuation and error propagation. TestEchoReviewHookControlFlow 检查终止、最多一次的放行及错误传播。
func TestEchoReviewHookControlFlow(t *testing.T) {
	mgr := newEchoReviewManager(t, "echo-hooks", nil)
	for _, check := range echoReviewChecks {
		if check.name == "annotation" {
			continue
		}
		for _, mode := range []string{"abort", "abort_next", "exit_next", "next_twice"} {
			if check.name == "access" && mode != "abort" {
				continue
			}
			t.Run(check.name+"/"+mode, func(t *testing.T) {
				c := echo4.New().NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
				getDTokenContext(c, mgr)
				calls, failures := 0, 0
				wantErr := errors.New("downstream failed")
				hook := func(_ context.Context, c echo4.Context, req *AuthHandleRequest) {
					if mode == "next_twice" {
						req.Next()
						req.Next()
						return
					}
					if mode == "exit_next" {
						req.Exit()
						req.Next()
						return
					}
					rc, err := GetRequestContextByContext(c)
					if err != nil {
						t.Fatal(err)
					}
					rc.Abort()
					if mode == "abort_next" {
						req.Next()
					}
				}
				h := check.build(context.Background(),
					WithBeforeAuthHandler(hook),
					WithRouteAccessHandler(func(ctx context.Context, c echo4.Context, req *RouteAccessRequest) {
						hook(ctx, c, nil)
						req.SkipAuth()
					}),
					WithFailFunc(func(echo4.Context, error) error { failures++; return nil }),
				)(func(echo4.Context) error { calls++; return wantErr })
				err := h(c)
				if mode == "next_twice" {
					if calls != 1 || !errors.Is(err, wantErr) {
						t.Fatalf("calls=%d error=%v, want one call and original error", calls, err)
					}
				} else if calls != 0 || err != nil {
					t.Fatalf("terminated hook continued: calls=%d error=%v", calls, err)
				}
				if failures != 0 {
					t.Fatalf("authentication ran after handled hook: failures=%d", failures)
				}
			})
		}
	}
}

// TestEchoReviewBodyTokenIsolation rejects query-only tokens when query reading is disabled. TestEchoReviewBodyTokenIsolation 验证关闭查询参数读取后拒绝仅来自查询的 Token。
func TestEchoReviewBodyTokenIsolation(t *testing.T) {
	mgr, err := NewBuilder().IsPrintBanner(false).IsLog(false).AutoRenew(false).AsyncEvent(false).
		IsReadHeader(false).IsReadCookie(false).IsReadQuery(false).IsReadBody(true).Build()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mgr.CloseManager)
	const token = "body-token"
	echoReviewLogin(t, mgr, token)
	key := mgr.GetConfig().TokenName
	for _, encoding := range []string{"urlencoded", "multipart"} {
		for _, inBody := range []bool{false, true} {
			name := encoding + "/query_only"
			if inBody {
				name = encoding + "/body"
			}
			t.Run(name, func(t *testing.T) {
				fields := url.Values{"unrelated": {"value"}}
				queryToken := token
				if inBody {
					fields.Set(key, token)
					queryToken = "invalid"
				}
				body, header := fields.Encode(), "application/x-www-form-urlencoded"
				if encoding == "multipart" {
					var b bytes.Buffer
					writer := multipart.NewWriter(&b)
					for field, values := range fields {
						if err := writer.WriteField(field, values[0]); err != nil {
							t.Fatal(err)
						}
					}
					if err := writer.Close(); err != nil {
						t.Fatal(err)
					}
					body, header = b.String(), writer.FormDataContentType()
				}
				r := httptest.NewRequest(http.MethodPost, "/?"+url.Values{key: {queryToken}}.Encode(), strings.NewReader(body))
				r.Header.Set("Content-Type", header)
				w := httptest.NewRecorder()
				e := echo4.New()
				e.Use(AuthMiddleware(context.Background(), WithManager(mgr)))
				called := false
				e.POST("/", func(c echo4.Context) error { called = true; return c.NoContent(http.StatusNoContent) })
				e.ServeHTTP(w, r)
				wantStatus := http.StatusUnauthorized
				if inBody {
					wantStatus = http.StatusNoContent
				}
				if w.Code != wantStatus || called != inBody {
					t.Fatalf("status=%d called=%v, want status=%d", w.Code, called, wantStatus)
				}
			})
		}
	}
}

// TestEchoReviewReplacedQuery ignores Echo's stale query cache after request replacement. TestEchoReviewReplacedQuery 验证替换请求后不会继续读取 Echo 的旧查询缓存。
func TestEchoReviewReplacedQuery(t *testing.T) {
	mgr, err := NewBuilder().IsPrintBanner(false).IsLog(false).AutoRenew(false).AsyncEvent(false).
		IsReadHeader(false).IsReadCookie(false).IsReadQuery(true).IsReadBody(false).Build()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mgr.CloseManager)
	const token = "query-token"
	echoReviewLogin(t, mgr, token)
	key := mgr.GetConfig().TokenName
	for _, replacement := range []string{token, "invalid"} {
		t.Run(replacement, func(t *testing.T) {
			e := echo4.New()
			e.Use(RegisterDTokenContextMiddleware(context.Background(), WithManager(mgr)))
			e.Use(func(next echo4.HandlerFunc) echo4.HandlerFunc {
				return func(c echo4.Context) error {
					_ = c.QueryParams()
					r := c.Request().Clone(c.Request().Context())
					r.URL.RawQuery = url.Values{key: {replacement}}.Encode()
					c.SetRequest(r)
					rc, err := GetRequestContextByContext(c)
					if err != nil {
						t.Fatal(err)
					}
					if values := rc.GetQueryAll()[key]; len(values) != 1 || values[0] != replacement {
						t.Errorf("query values=%v, want current request", values)
					}
					return next(c)
				}
			})
			e.Use(AuthMiddleware(context.Background()))
			called := false
			e.GET("/", func(c echo4.Context) error { called = true; return c.NoContent(http.StatusNoContent) })
			initial := token
			wantStatus := http.StatusUnauthorized
			if replacement == token {
				initial, wantStatus = "invalid", http.StatusNoContent
			}
			w := httptest.NewRecorder()
			e.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/?"+url.Values{key: {initial}}.Encode(), nil))
			if w.Code != wantStatus || called != (replacement == token) {
				t.Fatalf("status=%d called=%v, want status=%d", w.Code, called, wantStatus)
			}
		})
	}
}

// TestEchoReviewNilAnnotation verifies default and custom failure responses. TestEchoReviewNilAnnotation 验证空注解的默认响应和自定义失败处理。
func TestEchoReviewNilAnnotation(t *testing.T) {
	for _, custom := range []bool{false, true} {
		w := httptest.NewRecorder()
		c := echo4.New().NewContext(httptest.NewRequest(http.MethodGet, "/", nil), w)
		called, failures := false, 0
		var failFunc func(echo4.Context, error) error
		wantErr := errors.New("custom failure")
		if custom {
			failFunc = func(_ echo4.Context, err error) error {
				failures++
				if !errors.Is(err, ErrInvalidParam) {
					t.Errorf("failure=%v, want ErrInvalidParam", err)
				}
				return wantErr
			}
		}
		err := GetHandler(context.Background(), func(echo4.Context) error { called = true; return nil }, failFunc, nil)(c)
		if called {
			t.Fatal("nil annotation reached handler")
		}
		if custom {
			if failures != 1 || !errors.Is(err, wantErr) {
				t.Fatalf("failures=%d error=%v", failures, err)
			}
		} else if err != nil || w.Code != http.StatusBadRequest {
			t.Fatalf("error=%v status=%d, want nil,400", err, w.Code)
		}
	}
}

// TestEchoReviewIPExtractor keeps the host application's proxy trust policy authoritative. TestEchoReviewIPExtractor 验证客户端 IP 遵循宿主应用的代理信任配置。
func TestEchoReviewIPExtractor(t *testing.T) {
	e := echo4.New()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "198.51.100.7:12345"
	r.Header.Set("X-Forwarded-For", "203.0.113.2")
	r.Header.Set("X-Real-IP", "192.0.2.1")
	c := NewEchoContext(e.NewContext(r, httptest.NewRecorder()))
	e.IPExtractor = echo4.ExtractIPDirect()
	if got := c.GetClientIP(); got != "198.51.100.7" {
		t.Fatalf("direct IP=%q", got)
	}
	e.IPExtractor = func(*http.Request) string { return "203.0.113.9" }
	if got := c.GetClientIP(); got != "203.0.113.9" {
		t.Fatalf("configured IP=%q", got)
	}
}
