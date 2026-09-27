package gf

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
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
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gsession"
	"github.com/gogf/gf/v2/util/guid"
)

// gfReviewChecks covers all authentication entry points. gfReviewChecks 覆盖全部鉴权入口。
var gfReviewChecks = []struct {
	name  string
	build func(context.Context, ...AuthOption) ghttp.HandlerFunc
}{
	{"login", AuthMiddleware},
	{"access", AccessMiddleware},
	{"permission", func(ctx context.Context, opts ...AuthOption) ghttp.HandlerFunc {
		return PermissionMiddleware(ctx, []string{"read"}, opts...)
	}},
	{"path", func(ctx context.Context, opts ...AuthOption) ghttp.HandlerFunc {
		return PermissionPathMiddleware(ctx, []string{"read"}, opts...)
	}},
	{"role", func(ctx context.Context, opts ...AuthOption) ghttp.HandlerFunc {
		return RoleMiddleware(ctx, []string{"reader"}, opts...)
	}},
	{"annotation", func(ctx context.Context, opts ...AuthOption) ghttp.HandlerFunc {
		options := defaultAuthOptions()
		for _, opt := range opts {
			opt(options)
		}
		return GetHandler(ctx, nil, options.FailFunc, &Annotation{
			AuthType: options.AuthType, CheckLogin: true, CheckPermission: []string{"read"},
		})
	}},
}

// newGFReviewServer initializes GoFrame routing and sessions on an ephemeral loopback port. newGFReviewServer 通过本机临时端口初始化 GoFrame 路由和会话。
func newGFReviewServer(t *testing.T, handler ghttp.HandlerFunc, middleware ...ghttp.HandlerFunc) *ghttp.Server {
	t.Helper()
	s := ghttp.GetServer(guid.S())
	s.SetAddr("127.0.0.1:0")
	s.SetDumpRouterMap(false)
	s.SetLogStdout(false)
	s.SetAccessLogEnabled(false)
	s.SetErrorLogEnabled(false)
	s.SetSessionStorage(gsession.NewStorageMemory())
	s.BindMiddlewareDefault(middleware...)
	s.BindHandler("/", handler)
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

// newGFReviewManager creates an isolated manager without asynchronous maintenance. newGFReviewManager 创建不执行异步维护的独立 Manager。
func newGFReviewManager(t *testing.T, authType string, storage adapter.Storage) *manager.Manager {
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

// gfReviewLogin seeds the permissions needed by shared cases. gfReviewLogin 准备共用用例所需的登录态和权限。
func gfReviewLogin(t *testing.T, mgr *manager.Manager, token string) {
	t.Helper()
	ctx := context.Background()
	if _, err := mgr.LoginWithOptions(ctx, manager.LoginOptions{LoginID: "reader", Token: token}); err != nil {
		t.Fatal(err)
	}
	if err := mgr.AddPermissions(ctx, "reader", []string{"read", "/"}); err != nil {
		t.Fatal(err)
	}
	if err := mgr.AddRoles(ctx, "reader", []string{"reader"}); err != nil {
		t.Fatal(err)
	}
}

// gfReviewStorage observes the request context used by authentication. gfReviewStorage 观察鉴权实际使用的请求上下文。
type gfReviewStorage struct {
	adapter.Storage
	inspect func(context.Context)
}

// Get checks propagation at the storage boundary. Get 在存储边界检查上下文传播。
func (s *gfReviewStorage) Get(ctx context.Context, key string) (any, error) {
	if s.inspect != nil {
		s.inspect(ctx)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.Storage.Get(ctx, key)
}

// TestGFReviewRequestContextAndManager verifies inheritance, cancellation and request values through real routing. TestGFReviewRequestContextAndManager 通过真实路由验证 Manager 继承、取消信号与请求上下文值。
func TestGFReviewRequestContextAndManager(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	for _, check := range gfReviewChecks {
		t.Run(check.name, func(t *testing.T) {
			storage := &gfReviewStorage{Storage: memory.NewStorage()}
			mgr := newGFReviewManager(t, "gf-request", storage)
			const token = "request-token"
			gfReviewLogin(t, mgr, token)
			type requestKey struct{}
			var expected context.Context
			reads, hooks, handled := 0, 0, 0
			inspect := func(ctx context.Context) {
				gotDeadline, gotOK := ctx.Deadline()
				wantDeadline, wantOK := expected.Deadline()
				if ctx.Value(requestKey{}) != "request" || ctx.Done() != expected.Done() || gotOK != wantOK || !gotDeadline.Equal(wantDeadline) {
					t.Error("request values, deadline or cancellation were lost")
				}
			}
			storage.inspect = func(ctx context.Context) { reads++; inspect(ctx) }
			s := newGFReviewServer(t, func(r *ghttp.Request) {
				handled++
				if id, err := GetLoginIDByCtx(r.Context()); err != nil || id != "reader" {
					t.Errorf("GetLoginIDByCtx=%q,%v", id, err)
				}
				r.Response.WriteHeader(http.StatusNoContent)
			}, RegisterDTokenContextMiddleware(context.Background(), WithManager(mgr)),
				check.build(context.Background(),
					WithBeforeAuthHandler(func(ctx context.Context, _ *ghttp.Request, _ *AuthHandleRequest) { hooks++; inspect(ctx) }),
					WithRouteAccessHandler(func(ctx context.Context, _ *ghttp.Request, _ *RouteAccessRequest) { hooks++; inspect(ctx) }),
				))
			for _, canceled := range []bool{false, true} {
				ctx, cancel := context.WithTimeout(context.WithValue(context.Background(), requestKey{}, "request"), time.Minute)
				t.Cleanup(cancel)
				if canceled {
					cancel()
				}
				expected = ctx
				reads, hooks, handled = 0, 0, 0
				req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
				req.Header.Set(mgr.GetConfig().TokenName, token)
				response := httptest.NewRecorder()
				s.ServeHTTP(response, req)
				if reads == 0 || check.name != "annotation" && hooks != 1 {
					t.Fatalf("reads=%d hooks=%d, expected authentication checks", reads, hooks)
				}
				if canceled {
					if handled != 0 || response.Code < 400 {
						t.Fatalf("canceled request passed: handled=%d status=%d", handled, response.Code)
					}
				} else if handled != 1 || response.Code != http.StatusNoContent {
					t.Fatalf("valid request failed: handled=%d status=%d body=%s", handled, response.Code, response.Body.String())
				}
			}
		})
	}
}

// TestGFReviewManagerSelection verifies explicit selection and closed-manager rejection. TestGFReviewManagerSelection 验证显式实例选择和关闭实例的拒绝行为。
func TestGFReviewManagerSelection(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	global := newGFReviewManager(t, "", nil)
	selected := newGFReviewManager(t, "gf-selected", nil)
	local := newGFReviewManager(t, "gf-local", nil)
	dtoken.SetManager(global)
	dtoken.SetManager(selected)
	r := &ghttp.Request{Request: httptest.NewRequest(http.MethodGet, "/", nil)}
	getDContext(r, local)
	for _, tc := range []struct {
		explicit *manager.Manager
		authType string
		want     *manager.Manager
	}{
		{nil, "", local}, {nil, "gf-selected", selected}, {local, "missing", local},
	} {
		if got, err := resolveRequestManager(r, tc.explicit, tc.authType); err != nil || got != tc.want {
			t.Fatalf("manager=%p,%v want %p", got, err, tc.want)
		}
	}
	local.CloseManager()
	if _, err := resolveRequestManager(r, nil, ""); !errors.Is(err, ErrManagerNotFound) {
		t.Fatalf("closed cache error=%v", err)
	}
	if _, err := resolveRequestManager(r, local, "gf-selected"); !errors.Is(err, ErrManagerNotFound) {
		t.Fatalf("closed explicit manager error=%v", err)
	}
}

// TestGFReviewRenewalAndInvalidCache verifies namespace isolation and invalid cached managers. TestGFReviewRenewalAndInvalidCache 验证续期命名空间隔离以及无效缓存实例。
func TestGFReviewRenewalAndInvalidCache(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	global := newGFReviewManager(t, "", nil)
	local := newGFReviewManager(t, "gf-renew", nil)
	dtoken.SetManager(global)
	const token = "shared-token"
	gfReviewLogin(t, global, token)
	gfReviewLogin(t, local, token)
	r := &ghttp.Request{Request: httptest.NewRequest(http.MethodGet, "/", nil)}
	r.Header.Set(local.GetConfig().TokenName, token)
	getDContext(r, local)
	if err := RenewTimeoutByCtx(r.Context(), time.Minute); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		mgr     *manager.Manager
		timeout int64
	}{{global, 600}, {local, 60}} {
		info, err := tc.mgr.GetTokenInfo(context.Background(), token)
		if err != nil || info.Timeout != tc.timeout {
			t.Fatalf("token info=%+v,%v want timeout=%d", info, err, tc.timeout)
		}
	}
	var typedNil *corecontext.DTokenContext
	r.SetCtxVar(DTokenCtxKey, typedNil)
	if got, err := GetManagerByCtx(r.Context()); err != nil || got != global {
		t.Fatalf("typed nil fallback=%p,%v", got, err)
	}
	local.CloseManager()
	for _, mgr := range []*manager.Manager{nil, local} {
		r.SetCtxVar(DTokenCtxKey, corecontext.NewContext(NewGFContext(r), mgr))
		_, tokenErr := GetTokenValueByCtx(r.Context())
		_, loginErr := GetLoginIDByCtx(r.Context())
		_, infoErr := IntrospectTokenByCtx(r.Context())
		renewErr := RenewTimeoutByCtx(r.Context(), time.Minute)
		for _, err := range []error{tokenErr, loginErr, infoErr, renewErr} {
			if !errors.Is(err, ErrManagerNotFound) {
				t.Errorf("error=%v, want ErrManagerNotFound", err)
			}
		}
	}
}

// TestGFReviewFailureStopsNativeNext verifies failures cannot resume the native middleware chain. TestGFReviewFailureStopsNativeNext 验证失败回调无法重新放行原生中间件链。
func TestGFReviewFailureStopsNativeNext(t *testing.T) {
	mgr := newGFReviewManager(t, "gf-fail", nil)
	for _, check := range gfReviewChecks {
		t.Run(check.name, func(t *testing.T) {
			failures, handled := 0, 0
			s := newGFReviewServer(t, func(*ghttp.Request) { handled++ },
				RegisterDTokenContextMiddleware(context.Background(), WithManager(mgr)),
				check.build(context.Background(), WithFailFunc(func(r *ghttp.Request, err error) {
					failures++
					if err == nil || !r.IsExited() {
						t.Error("failure callback ran before chain termination")
					}
					r.Response.WriteHeader(http.StatusUnauthorized)
					r.Response.Write("denied")
					r.Middleware.Next()
				})))
			response := httptest.NewRecorder()
			s.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
			if failures != 1 || handled != 0 || response.Code != http.StatusUnauthorized || response.Body.String() != "denied" {
				t.Fatalf("failures=%d handled=%d response=%d %s", failures, handled, response.Code, response.Body.String())
			}
		})
	}
}

// TestGFReviewHookControlFlow verifies adapter aborts and native exit handling. TestGFReviewHookControlFlow 验证适配器终止标记和原生退出流程。
func TestGFReviewHookControlFlow(t *testing.T) {
	mgr := newGFReviewManager(t, "gf-hooks", nil)
	for _, check := range gfReviewChecks {
		if check.name == "annotation" {
			continue
		}
		for _, mode := range []string{"abort", "abort_next", "exit", "next_twice"} {
			if check.name == "access" && mode != "abort" {
				continue
			}
			t.Run(check.name+"/"+mode, func(t *testing.T) {
				handled, failures := 0, 0
				s := newGFReviewServer(t, func(r *ghttp.Request) { handled++; r.Response.WriteHeader(http.StatusNoContent) },
					RegisterDTokenContextMiddleware(context.Background(), WithManager(mgr)),
					check.build(context.Background(),
						WithFailFunc(func(*ghttp.Request, error) { failures++ }),
						WithBeforeAuthHandler(func(_ context.Context, r *ghttp.Request, req *AuthHandleRequest) {
							r.Response.WriteHeader(http.StatusNoContent)
							switch mode {
							case "exit":
								req.Exit()
							case "next_twice":
								req.Next()
								req.Next()
							default:
								cached, _ := GetDTokenContext(r)
								cached.GetRequestContext().Abort()
								if mode == "abort_next" {
									req.Next()
								}
							}
						}),
						WithRouteAccessHandler(func(_ context.Context, r *ghttp.Request, req *RouteAccessRequest) {
							r.Response.WriteHeader(http.StatusNoContent)
							cached, _ := GetDTokenContext(r)
							cached.GetRequestContext().Abort()
							req.SkipAuth()
						})))
				response := httptest.NewRecorder()
				s.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
				want := 0
				if mode == "next_twice" {
					want = 1
				}
				if handled != want || failures != 0 || response.Code != http.StatusNoContent {
					t.Fatalf("handled=%d want=%d failures=%d status=%d", handled, want, failures, response.Code)
				}
			})
		}
	}
	next, exits := 0, 0
	req := newAuthHandleRequest(defaultAuthOptions(), func() { next++ }, func() { exits++ })
	req.Exit()
	req.Next()
	req.Exit()
	if next != 0 || exits != 1 {
		t.Fatalf("next=%d exits=%d", next, exits)
	}
	req = newAuthHandleRequest(defaultAuthOptions(), func() { next++ }, func() { exits++ })
	req.Next()
	req.Next()
	req.Exit()
	if next != 1 || exits != 1 {
		t.Fatalf("next=%d exits=%d", next, exits)
	}
}

// TestGFReviewAbortedEntry skips checks even for ignored annotations and registration. TestGFReviewAbortedEntry 验证已终止请求不会继续鉴权、忽略注解或上下文注册。
func TestGFReviewAbortedEntry(t *testing.T) {
	mgr := newGFReviewManager(t, "gf-aborted", nil)
	called := 0
	opts := []AuthOption{
		WithBeforeAuthHandler(func(context.Context, *ghttp.Request, *AuthHandleRequest) { called++ }),
		WithRouteAccessHandler(func(context.Context, *ghttp.Request, *RouteAccessRequest) { called++ }),
		WithFailFunc(func(*ghttp.Request, error) { called++ }),
	}
	handlers := []ghttp.HandlerFunc{
		RegisterDTokenContextMiddleware(context.Background(), WithManager(mgr)),
		GetHandler(context.Background(), func(*ghttp.Request) { called++ }, nil, &Annotation{Ignore: true}),
		GetHandler(context.Background(), func(*ghttp.Request) { called++ }, nil),
	}
	for _, check := range gfReviewChecks {
		handlers = append(handlers, check.build(context.Background(), opts...))
	}
	for _, handler := range handlers {
		r := &ghttp.Request{Request: httptest.NewRequest(http.MethodGet, "/", nil)}
		getDContext(r, mgr).GetRequestContext().Abort()
		handler(r)
	}
	if called != 0 {
		t.Fatalf("aborted request invoked %d callbacks", called)
	}
}

// TestGFReviewNilAnnotationAndErrorJSON verifies invalid annotations fail with a JSON response. TestGFReviewNilAnnotationAndErrorJSON 验证空注解通过 JSON 响应报告参数错误。
func TestGFReviewNilAnnotationAndErrorJSON(t *testing.T) {
	handled := 0
	s := newGFReviewServer(t, GetHandler(context.Background(), func(*ghttp.Request) { handled++ }, nil, nil))
	response := httptest.NewRecorder()
	s.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	var payload struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if handled != 0 || response.Code != http.StatusBadRequest || payload.Code != CodeBadRequest || !strings.HasPrefix(response.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("handled=%d response=%d %s %s", handled, response.Code, response.Header(), response.Body.String())
	}
}

// TestGFReviewResponseAndCookies verifies response headers, status-only writes and independent cookie options. TestGFReviewResponseAndCookies 验证响应头、纯状态码设置以及独立 Cookie 选项。
func TestGFReviewResponseAndCookies(t *testing.T) {
	s := newGFReviewServer(t, func(r *ghttp.Request) {
		ctx := NewGFContext(r)
		ctx.SetHeader("X-Token", "response")
		if ctx.GetHeader("X-Token") != "request" {
			t.Error("response header changed the request")
		}
		ctx.SetStatusCode(http.StatusCreated)
		ctx.SetCookieWithOptions(&adapter.CookieOptions{Name: "strict", Value: "token", Path: "/", MaxAge: 60, Secure: true, HttpOnly: true, SameSite: "Strict"})
		ctx.SetCookie("legacy", "token", 60, "/", "", false, false)
		ctx.SetCookieWithOptions(&adapter.CookieOptions{Name: "expired", Path: "/", MaxAge: -1, SameSite: "None", Secure: true})
		if n, err := ctx.Write([]byte(`{"ok":true}`)); err != nil || n != 11 {
			t.Errorf("Write=%d,%v", n, err)
		}
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Token", "request")
	response := httptest.NewRecorder()
	s.ServeHTTP(response, req)
	if response.Code != http.StatusCreated || response.Header().Get("X-Token") != "response" || response.Body.String() != `{"ok":true}` {
		t.Fatalf("response=%d %v %s", response.Code, response.Header(), response.Body.String())
	}
	cookies := map[string]*http.Cookie{}
	for _, cookie := range response.Result().Cookies() {
		cookies[cookie.Name] = cookie
	}
	if c := cookies["strict"]; c == nil || c.SameSite != http.SameSiteStrictMode || !c.Secure || !c.HttpOnly {
		t.Errorf("strict cookie=%+v", c)
	}
	if c := cookies["legacy"]; c == nil || c.SameSite != http.SameSiteLaxMode || c.Secure || c.HttpOnly {
		t.Errorf("legacy cookie=%+v", c)
	}
	if c := cookies["expired"]; c == nil || c.MaxAge != -1 || c.SameSite != http.SameSiteNoneMode {
		t.Errorf("expired cookie=%+v", c)
	}
}

// gfReviewBrokenBody simulates a truncated request body. gfReviewBrokenBody 模拟读取失败的残缺请求体。
type gfReviewBrokenBody struct{ err error }

// Read returns partial data and its read error together. Read 同时返回部分数据及读取错误。
func (b gfReviewBrokenBody) Read(p []byte) (int, error) { return copy(p, "partial"), b.err }

// Close satisfies the request body contract. Close 满足请求体接口。
func (b gfReviewBrokenBody) Close() error { return nil }

// TestGFReviewBodyAndClientIP verifies repeatable body reads, read errors and untrusted forwarding headers. TestGFReviewBodyAndClientIP 验证重复读取、读取错误及不可信转发头。
func TestGFReviewBodyAndClientIP(t *testing.T) {
	r := &ghttp.Request{Request: httptest.NewRequest(http.MethodPost, "/?token=query", strings.NewReader("hello"))}
	ctx := NewGFContext(r)
	for i := 0; i < 2; i++ {
		if body, err := ctx.GetBody(); err != nil || string(body) != "hello" {
			t.Fatalf("GetBody=%q,%v", body, err)
		}
	}
	if body, err := io.ReadAll(r.Body); err != nil || string(body) != "hello" {
		t.Fatalf("request body=%q,%v", body, err)
	}
	wantErr := errors.New("body interrupted")
	r = &ghttp.Request{Request: httptest.NewRequest(http.MethodPost, "/", nil)}
	r.Body = gfReviewBrokenBody{err: wantErr}
	ctx = NewGFContext(r)
	for i := 0; i < 2; i++ {
		if body, err := ctx.GetBody(); !errors.Is(err, wantErr) || body != nil {
			t.Fatalf("broken body=%q,%v", body, err)
		}
	}
	for _, address := range []struct{ remote, ip string }{{"192.0.2.8:8080", "192.0.2.8"}, {"[2001:db8::8]:8080", "2001:db8::8"}} {
		r.RemoteAddr = address.remote
		for _, header := range []string{"X-Forwarded-For", "X-Real-IP", "Proxy-Client-IP", "WL-Proxy-Client-IP", "HTTP_CLIENT_IP", "HTTP_X_FORWARDED_FOR"} {
			r.Header.Set(header, "198.51.100.99")
		}
		if got := ctx.GetClientIP(); got != address.ip {
			t.Fatalf("IP=%q want=%q", got, address.ip)
		}
	}
}

// TestGFReviewBodyTokenSources preserves GoFrame body formats without allowing query fallback. TestGFReviewBodyTokenSources 保留 GoFrame 请求体格式支持，并防止回退至查询参数。
func TestGFReviewBodyTokenSources(t *testing.T) {
	mgr, err := NewBuilder().IsPrintBanner(false).IsLog(false).AutoRenew(false).AsyncEvent(false).
		IsReadHeader(false).IsReadCookie(false).IsReadQuery(false).IsReadBody(true).Build()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mgr.CloseManager)
	key := mgr.GetConfig().TokenName
	var multipartBody bytes.Buffer
	writer := multipart.NewWriter(&multipartBody)
	if err := writer.WriteField(key, "multipart-token"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, contentType, body, want string }{
		{"query-only", "application/x-www-form-urlencoded", "other=value", ""},
		{"form", "application/x-www-form-urlencoded", key + "=form-token", "form-token"},
		{"json", "application/json", `{"` + key + `":"json-token"}`, "json-token"},
		{"multipart", writer.FormDataContentType(), multipartBody.String(), "multipart-token"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &ghttp.Request{
				Request: httptest.NewRequest(http.MethodPost, "/?"+key+"=query-token", strings.NewReader(tc.body)),
				Server:  ghttp.GetServer(guid.S()),
			}
			r.Header.Set("Content-Type", tc.contentType)
			r.Header.Set(key, "header-token")
			dCtx := corecontext.NewContext(NewGFContext(r), mgr)
			if got := dCtx.GetTokenValue(); got != tc.want {
				t.Fatalf("token=%q want=%q", got, tc.want)
			}
			if r.MultipartForm != nil {
				t.Cleanup(func() { _ = r.MultipartForm.RemoveAll() })
			}
		})
	}
}

// gfReviewExpectExit handles GoFrame's native exit panic in direct handler tests. gfReviewExpectExit 在直接调用处理器的测试中接收 GoFrame 原生退出信号。
func gfReviewExpectExit(t *testing.T, r *ghttp.Request, handler ghttp.HandlerFunc) {
	t.Helper()
	// Obtain the framework's private control signal without treating arbitrary panics as success. 获取框架私有控制信号，避免将其他 panic 误判为正常退出。
	var exitSignal any
	func() {
		defer func() { exitSignal = recover() }()
		(&ghttp.Request{}).ExitAll()
	}()
	defer func() {
		if recovered := recover(); recovered != exitSignal || !r.IsExited() {
			t.Fatalf("expected GoFrame ExitAll, got %v", recovered)
		}
	}()
	handler(r)
}
