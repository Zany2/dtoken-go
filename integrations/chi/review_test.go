package chi

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

	"github.com/Zany2/dtoken-go/core/adapter"
	corecontext "github.com/Zany2/dtoken-go/core/context"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/dtoken"
	chiRouter "github.com/go-chi/chi/v5"
)

// chiReviewMiddlewares covers all authentication middleware through HTTP chains. chiReviewMiddlewares 通过 HTTP 链路覆盖全部鉴权中间件。
var chiReviewMiddlewares = []struct {
	name  string
	build func(...AuthOption) func(http.Handler) http.Handler
}{
	{"login", AuthMiddleware},
	{"access", AccessMiddleware},
	{"permission", func(opts ...AuthOption) func(http.Handler) http.Handler {
		return PermissionMiddleware([]string{"read"}, opts...)
	}},
	{"path", func(opts ...AuthOption) func(http.Handler) http.Handler {
		return PermissionPathMiddleware([]string{"read"}, opts...)
	}},
	{"role", func(opts ...AuthOption) func(http.Handler) http.Handler {
		return RoleMiddleware([]string{"reader"}, opts...)
	}},
}

// newChiReviewManager creates an isolated manager without asynchronous work. newChiReviewManager 创建不执行异步任务的独立 Manager。
func newChiReviewManager(t *testing.T, authType string) *manager.Manager {
	t.Helper()
	b := NewBuilder().IsPrintBanner(false).IsLog(false).AutoRenew(false).AsyncEvent(false)
	if authType != "" {
		b.AuthType(authType)
	}
	mgr, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mgr.CloseManager)
	return mgr
}

// chiReviewLogin grants the access rules exercised by the shared middleware cases. chiReviewLogin 为共用中间件用例创建具备相应权限的登录态。
func chiReviewLogin(t *testing.T, mgr *manager.Manager, id string) string {
	t.Helper()
	ctx := context.Background()
	token, err := mgr.Login(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if err := mgr.AddPermissions(ctx, id, []string{"read", "/protected"}); err != nil {
		t.Fatal(err)
	}
	if err := mgr.AddRoles(ctx, id, []string{"reader"}); err != nil {
		t.Fatal(err)
	}
	return token
}

// TestChiReviewManagerSelection verifies inheritance, explicit overrides and registry fallback. TestChiReviewManagerSelection 验证请求 Manager 继承、显式覆盖与注册表回退。
func TestChiReviewManagerSelection(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	cached := newChiReviewManager(t, "chi-cached")
	global := newChiReviewManager(t, "")
	selected := newChiReviewManager(t, "chi-selected")
	dtoken.SetManager(global)
	dtoken.SetManager(selected)
	cachedToken := chiReviewLogin(t, cached, "cached-user")
	globalToken := chiReviewLogin(t, global, "global-user")
	selectedToken := chiReviewLogin(t, selected, "selected-user")

	for _, check := range chiReviewMiddlewares {
		for _, tc := range []struct {
			name     string
			register bool
			opts     []AuthOption
			token    string
			want     *manager.Manager
		}{
			{"cached", true, nil, cachedToken, cached},
			{"auth_type", true, []AuthOption{WithAuthType("chi-selected")}, selectedToken, selected},
			{"explicit", true, []AuthOption{WithManager(selected), WithAuthType("missing")}, selectedToken, selected},
			{"missing_type", true, []AuthOption{WithAuthType("missing")}, cachedToken, nil},
			{"global", false, nil, globalToken, global},
		} {
			t.Run(check.name+"/"+tc.name, func(t *testing.T) {
				router := chiRouter.NewRouter()
				if tc.register {
					router.Use(RegisterDTokenContextMiddleware(WithManager(cached)))
				}
				router.Use(check.build(tc.opts...))
				called := false
				router.Get("/protected", func(w http.ResponseWriter, r *http.Request) {
					called = true
					if got, err := GetManagerByCtx(r.Context()); err != nil || got != tc.want {
						t.Errorf("manager = %p, %v, want %p", got, err, tc.want)
					}
					w.WriteHeader(http.StatusNoContent)
				})
				r := httptest.NewRequest(http.MethodGet, "/protected", nil)
				r.Header.Set(cached.GetConfig().TokenName, tc.token)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, r)
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

// TestChiReviewCurrentRequestBinding verifies cloned requests and wrapped writers replace stale bindings. TestChiReviewCurrentRequestBinding 验证克隆请求与包装写入器不会继续使用旧绑定。
func TestChiReviewCurrentRequestBinding(t *testing.T) {
	mgr := newChiReviewManager(t, "chi-binding")
	oldToken := chiReviewLogin(t, mgr, "old")
	newToken := chiReviewLogin(t, mgr, "new")
	checks := append(chiReviewMiddlewares[:len(chiReviewMiddlewares):len(chiReviewMiddlewares)], struct {
		name  string
		build func(...AuthOption) func(http.Handler) http.Handler
	}{"annotation", func(...AuthOption) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return GetHandler(next.ServeHTTP, &Annotation{CheckLogin: true})
		}
	}})
	for _, check := range checks {
		for _, replacement := range []string{newToken, "invalid"} {
			t.Run(check.name+"/"+map[bool]string{true: "valid", false: "invalid"}[replacement == newToken], func(t *testing.T) {
				type markerKey struct{}
				var original *corecontext.DTokenContext
				var wrapped *chiReviewWriter
				called := false
				inspect := func(r *http.Request) {
					t.Helper()
					rc, err := GetRequestContextByCtx(r.Context())
					if err != nil {
						t.Fatal(err)
					}
					ext := rc.(adapter.RequestContextExt)
					if ext.GetHeader(mgr.GetConfig().TokenName) != replacement || ext.GetRawRequest().Context().Value(markerKey{}) != "current" {
						t.Error("adapter retained the old request")
					}
					if ext.GetRawResponseWriter() != wrapped {
						t.Error("adapter bypassed the wrapped response writer")
					}
					rc.Set("review-hook", "visible")
				}
				router := chiRouter.NewRouter()
				router.Use(RegisterDTokenContextMiddleware(WithManager(mgr)))
				router.Use(func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						original, _ = GetDTokenContext(r)
						r = r.Clone(context.WithValue(r.Context(), markerKey{}, "current"))
						r.Header.Set(mgr.GetConfig().TokenName, replacement)
						wrapped = &chiReviewWriter{ResponseWriter: w}
						if got := GetTokenFromRequest(wrapped, r); got != replacement {
							t.Errorf("GetTokenFromRequest = %q, want replacement", got)
						}
						if got := IsLoginFromRequest(wrapped, r); got != (replacement == newToken) {
							t.Errorf("IsLoginFromRequest = %v", got)
						}
						if id, err := GetLoginIDFromRequest(wrapped, r); replacement == newToken && (err != nil || id != "new") {
							t.Errorf("GetLoginIDFromRequest = %q, %v", id, err)
						}
						next.ServeHTTP(wrapped, r)
					})
				})
				router.Use(check.build(
					WithBeforeAuthHandler(func(_ http.ResponseWriter, r *http.Request, _ *AuthHandleRequest) { inspect(r) }),
					WithRouteAccessHandler(func(_ http.ResponseWriter, r *http.Request, _ *RouteAccessRequest) { inspect(r) }),
				))
				router.Get("/protected", func(_ http.ResponseWriter, r *http.Request) {
					called = true
					if check.name != "annotation" && r.Context().Value("review-hook") != "visible" {
						t.Error("hook values were not propagated downstream")
					}
					inspect(r)
					if id, err := GetLoginIDByCtx(r.Context()); err != nil || id != "new" {
						t.Errorf("GetLoginIDByCtx = %q, %v", id, err)
					}
					rc, _ := GetRequestContextByCtx(r.Context())
					rc.SetStatusCode(http.StatusNoContent)
				})
				r := httptest.NewRequest(http.MethodGet, "/protected", nil)
				r.Header.Set(mgr.GetConfig().TokenName, oldToken)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, r)
				wantStatus := http.StatusUnauthorized
				if replacement == newToken {
					wantStatus = http.StatusNoContent
				}
				if w.Code != wantStatus || called != (replacement == newToken) || wrapped.writes != 1 {
					t.Fatalf("status=%d called=%v wrapped writes=%d", w.Code, called, wrapped.writes)
				}
				if original.GetTokenValue() != oldToken {
					t.Fatal("rebinding mutated the parent request adapter")
				}
			})
		}
	}
}

// chiReviewWriter records status writes made through a middleware wrapper. chiReviewWriter 记录经过中间件包装器的状态码写入。
type chiReviewWriter struct {
	http.ResponseWriter
	writes int
}

// WriteHeader counts responses that pass through the wrapper. WriteHeader 统计经过包装器的响应。
func (w *chiReviewWriter) WriteHeader(status int) {
	w.writes++
	w.ResponseWriter.WriteHeader(status)
}

// TestChiReviewBodyTokenIsolation rejects query-only tokens even when body reading is enabled. TestChiReviewBodyTokenIsolation 验证启用请求体读取也不会接受仅在查询参数中的 Token。
func TestChiReviewBodyTokenIsolation(t *testing.T) {
	mgr, err := NewBuilder().IsPrintBanner(false).IsLog(false).AutoRenew(false).AsyncEvent(false).
		IsReadHeader(false).IsReadCookie(false).IsReadQuery(false).IsReadBody(true).Build()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mgr.CloseManager)
	token := chiReviewLogin(t, mgr, "body-user")
	key := mgr.GetConfig().TokenName
	for _, contentType := range []string{"urlencoded", "multipart"} {
		for _, inBody := range []bool{false, true} {
			name := contentType + "/query_only"
			if inBody {
				name = contentType + "/body"
			}
			t.Run(name, func(t *testing.T) {
				fields := url.Values{"unrelated": {"value"}}
				if inBody {
					fields.Set(key, token)
				}
				body := fields.Encode()
				header := "application/x-www-form-urlencoded"
				if contentType == "multipart" {
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
				r := httptest.NewRequest(http.MethodPost, "/?"+url.Values{key: {token}}.Encode(), strings.NewReader(body))
				r.Header.Set("Content-Type", header)
				w := httptest.NewRecorder()
				called := false
				AuthMiddleware(WithManager(mgr))(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					called = true
					w.WriteHeader(http.StatusNoContent)
				})).ServeHTTP(w, r)
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

// TestChiReviewHookControlFlow verifies custom termination and at-most-once continuation. TestChiReviewHookControlFlow 验证自定义终止与最多一次的后续处理。
func TestChiReviewHookControlFlow(t *testing.T) {
	mgr := newChiReviewManager(t, "chi-hooks")
	for _, check := range chiReviewMiddlewares {
		for _, mode := range []string{"abort", "abort_next", "exit_next", "next_twice"} {
			if check.name == "access" && mode != "abort" {
				continue
			}
			t.Run(check.name+"/"+mode, func(t *testing.T) {
				calls, failures := 0, 0
				hook := func(w http.ResponseWriter, r *http.Request, req *AuthHandleRequest) {
					if mode == "next_twice" {
						req.Next()
						req.Next()
						return
					}
					w.WriteHeader(http.StatusAccepted)
					if mode == "exit_next" {
						req.Exit()
						req.Next()
						return
					}
					rc, err := GetRequestContextByCtx(r.Context())
					if err != nil {
						t.Fatal(err)
					}
					rc.Abort()
					if mode == "abort_next" {
						req.Next()
					}
				}
				router := chiRouter.NewRouter()
				router.Use(RegisterDTokenContextMiddleware(WithManager(mgr)))
				router.Use(check.build(
					WithBeforeAuthHandler(hook),
					WithRouteAccessHandler(func(w http.ResponseWriter, r *http.Request, req *RouteAccessRequest) {
						hook(w, r, nil)
						req.SkipAuth()
					}),
					WithFailFunc(func(http.ResponseWriter, *http.Request, error) { failures++ }),
				))
				router.Get("/protected", func(w http.ResponseWriter, _ *http.Request) {
					calls++
					w.WriteHeader(http.StatusNoContent)
				})
				w := httptest.NewRecorder()
				router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/protected", nil))
				wantStatus, wantCalls := http.StatusAccepted, 0
				if mode == "next_twice" {
					wantStatus, wantCalls = http.StatusNoContent, 1
				}
				if w.Code != wantStatus || calls != wantCalls || failures != 0 {
					t.Fatalf("status=%d downstream=%d failures=%d", w.Code, calls, failures)
				}
			})
		}
	}
}

// TestChiReviewInvalidContextManager verifies all facade paths reject missing or closed managers. TestChiReviewInvalidContextManager 验证各门面入口拒绝缺失或已关闭的 Manager。
func TestChiReviewInvalidContextManager(t *testing.T) {
	closed := newChiReviewManager(t, "chi-closed")
	closed.CloseManager()
	rc := NewChiContext(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	var typedNil *corecontext.DTokenContext
	for _, tc := range []struct {
		name string
		ctx  context.Context
		want error
	}{
		{"nil", nil, ErrNotLogin},
		{"missing", context.Background(), ErrNotLogin},
		{"typed_nil", context.WithValue(context.Background(), DTokenCtxKey, typedNil), ErrNotLogin},
		{"nil_manager", context.WithValue(context.Background(), DTokenCtxKey, corecontext.NewContext(rc, nil)), ErrManagerNotFound},
		{"closed", context.WithValue(context.Background(), DTokenCtxKey, corecontext.NewContext(rc, closed)), ErrManagerNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, tokenErr := GetTokenValueByCtx(tc.ctx)
			_, idErr := GetLoginIDByCtx(tc.ctx)
			_, infoErr := GetTokenInfoByCtx(tc.ctx)
			_, introspectErr := IntrospectTokenByCtx(tc.ctx)
			for _, err := range []error{tokenErr, idErr, infoErr, introspectErr} {
				if !errors.Is(err, tc.want) {
					t.Errorf("error=%v, want %v", err, tc.want)
				}
			}
			if IsLoginByCtx(tc.ctx) {
				t.Fatal("invalid manager reported a login")
			}
		})
	}
}

// TestChiReviewClosedRequestManager prevents fallback to an unrelated global manager. TestChiReviewClosedRequestManager 验证请求 Manager 关闭后不会回退到无关的全局实例。
func TestChiReviewClosedRequestManager(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	global := newChiReviewManager(t, "")
	dtoken.SetManager(global)
	token := chiReviewLogin(t, global, "global-user")
	closed := newChiReviewManager(t, "chi-closed-request")
	closed.CloseManager()
	for _, check := range chiReviewMiddlewares {
		t.Run(check.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/protected", nil)
			r.Header.Set(global.GetConfig().TokenName, token)
			chiCtx := NewChiContext(w, r).(*ChiContext)
			getDTokenContext(chiCtx, closed)
			called := false
			check.build()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })).ServeHTTP(w, chiCtx.r)
			if called || w.Code != http.StatusNotFound {
				t.Fatalf("called=%v status=%d, want false,404", called, w.Code)
			}
		})
	}
}

// TestChiReviewNilAnnotation rejects an explicitly nil annotation without panicking or dispatching. TestChiReviewNilAnnotation 验证显式空注解返回错误且不会崩溃或继续处理。
func TestChiReviewNilAnnotation(t *testing.T) {
	called := false
	w := httptest.NewRecorder()
	GetHandler(func(http.ResponseWriter, *http.Request) { called = true }, nil)(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if called || w.Code != http.StatusBadRequest {
		t.Fatalf("called=%v status=%d, want false,400", called, w.Code)
	}
}

// TestChiReviewClientIPTrust ignores untrusted forwarding headers and supports normalized addresses. TestChiReviewClientIPTrust 忽略不可信转发头并支持上游规范化后的地址。
func TestChiReviewClientIPTrust(t *testing.T) {
	for _, tc := range []struct{ remote, want string }{
		{"198.51.100.7:12345", "198.51.100.7"},
		{"[2001:db8::1]:12345", "2001:db8::1"},
		{"203.0.113.2", "203.0.113.2"},
	} {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.RemoteAddr = tc.remote
		r.Header.Set("X-Real-IP", "192.0.2.1")
		r.Header.Set("X-Forwarded-For", "192.0.2.2, 192.0.2.3")
		if got := NewChiContext(httptest.NewRecorder(), r).GetClientIP(); got != tc.want {
			t.Errorf("remote=%q: IP=%q, want %q", tc.remote, got, tc.want)
		}
	}
}
