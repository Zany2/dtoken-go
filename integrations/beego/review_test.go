package beego

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/Zany2/dtoken-go/com/storage/memory"
	"github.com/Zany2/dtoken-go/core/adapter"
	corecontext "github.com/Zany2/dtoken-go/core/context"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/dtoken"
	web "github.com/beego/beego/v2/server/web"
	beegocontext "github.com/beego/beego/v2/server/web/context"
)

// beegoReviewChecks covers authentication filters and annotation-based access. beegoReviewChecks 覆盖鉴权过滤器及基于注解的访问检查。
var beegoReviewChecks = []struct {
	name  string
	build func(context.Context, ...AuthOption) web.FilterFunc
}{
	{"login", AuthMiddleware},
	{"access", AccessMiddleware},
	{"permission", func(ctx context.Context, opts ...AuthOption) web.FilterFunc {
		return PermissionMiddleware(ctx, []string{"read"}, opts...)
	}},
	{"path", func(ctx context.Context, opts ...AuthOption) web.FilterFunc {
		return PermissionPathMiddleware(ctx, []string{"read"}, opts...)
	}},
	{"role", func(ctx context.Context, opts ...AuthOption) web.FilterFunc {
		return RoleMiddleware(ctx, []string{"reader"}, opts...)
	}},
	{"annotation", func(ctx context.Context, opts ...AuthOption) web.FilterFunc {
		return AccessMiddleware(ctx, append(opts, WithRouteAccessHandler(RouteAccessHandlerFromAnnotations(&Annotation{CheckPermission: []string{"read"}})))...)
	}},
}

// newBeegoReviewManager creates an isolated manager without asynchronous work. newBeegoReviewManager 创建无异步任务的独立 Manager。
func newBeegoReviewManager(t *testing.T, authType string, storage adapter.Storage) *manager.Manager {
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

// beegoReviewLogin seeds credentials for the reviewed filters. beegoReviewLogin 准备过滤器复检所需凭据。
func beegoReviewLogin(t *testing.T, mgr *manager.Manager, token string) {
	t.Helper()
	ctx := context.Background()
	if _, err := mgr.LoginWithOptions(ctx, manager.LoginOptions{LoginID: "reader", Token: token}); err != nil {
		t.Fatal(err)
	}
	if err := mgr.AddPermissions(ctx, "reader", []string{"read", "/review"}); err != nil {
		t.Fatal(err)
	}
	if err := mgr.AddRoles(ctx, "reader", []string{"reader"}); err != nil {
		t.Fatal(err)
	}
}

// beegoReviewStorage observes request context at the storage boundary. beegoReviewStorage 在存储边界观察请求上下文。
type beegoReviewStorage struct {
	adapter.Storage
	inspect func(context.Context)
}

// Get verifies propagation and honors cancellation. Get 检查上下文传播并响应取消信号。
func (s *beegoReviewStorage) Get(ctx context.Context, key string) (any, error) {
	if s.inspect != nil {
		s.inspect(ctx)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.Storage.Get(ctx, key)
}

// beegoReviewRouter uses the native filter dispatcher without starting a server. beegoReviewRouter 使用原生过滤器分发器，不启动网络服务。
func beegoReviewRouter(t *testing.T, handler web.FilterFunc, filters ...web.FilterFunc) *web.ControllerRegister {
	t.Helper()
	router := web.NewControllerRegister()
	for _, filter := range filters {
		if err := router.InsertFilter("/*", web.BeforeRouter, filter, web.WithReturnOnOutput(true)); err != nil {
			t.Fatal(err)
		}
	}
	router.Get("/review", handler)
	return router
}

// TestBeegoReviewRequestContext verifies request values, deadlines and cancellation reach checks and hooks. TestBeegoReviewRequestContext 验证请求值、截止时间及取消信号传入鉴权和钩子。
func TestBeegoReviewRequestContext(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	for _, check := range beegoReviewChecks {
		t.Run(check.name, func(t *testing.T) {
			storage := &beegoReviewStorage{Storage: memory.NewStorage()}
			mgr := newBeegoReviewManager(t, "beego-request", storage)
			beegoReviewLogin(t, mgr, "request-token")
			registration, stop := context.WithCancel(context.Background())
			stop()
			for _, canceled := range []bool{false, true} {
				type requestKey struct{}
				ctx, cancel := context.WithTimeout(context.WithValue(context.Background(), requestKey{}, "current"), time.Minute)
				t.Cleanup(cancel)
				if canceled {
					cancel()
				}
				c, _ := newBeegoBehaviorContext(http.MethodGet, "/review", mgr.GetConfig().TokenName, "request-token")
				c.Request = c.Request.WithContext(ctx)
				reads, hooks, failures := 0, 0, 0
				inspect := func(got context.Context) {
					if got != ctx {
						t.Error("current request context was lost")
					}
				}
				storage.inspect = func(got context.Context) { reads++; inspect(got) }
				RegisterDTokenContextMiddleware(registration, WithManager(mgr))(c)
				check.build(registration,
					WithBeforeAuthHandler(func(got context.Context, _ *beegocontext.Context, _ *AuthHandleRequest) { hooks++; inspect(got) }),
					WithRouteAccessHandler(func(got context.Context, _ *beegocontext.Context, _ *RouteAccessRequest) { hooks++; inspect(got) }),
					WithFailFunc(func(c *beegocontext.Context, err error) {
						failures++
						inspect(requestContext(c))
						if err == nil {
							t.Error("missing failure error")
						}
					}),
				)(c)
				if reads == 0 || check.name != "annotation" && hooks != 1 {
					t.Fatalf("reads=%d hooks=%d", reads, hooks)
				}
				if canceled {
					if !c.ResponseWriter.Started || failures != 1 {
						t.Fatalf("canceled request passed: failures=%d", failures)
					}
				} else {
					if c.ResponseWriter.Started || failures != 0 {
						t.Fatalf("valid request failed: failures=%d", failures)
					}
					if got, err := GetLoginIDByContext(c); err != nil || got != "reader" {
						t.Fatalf("facade=%q,%v", got, err)
					}
				}
			}
		})
	}
}

// TestBeegoReviewFailureStopsNativeChain verifies even silent failure callbacks prevent controller execution. TestBeegoReviewFailureStopsNativeChain 验证失败回调即使不输出也会阻止控制器执行。
func TestBeegoReviewFailureStopsNativeChain(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	mgr := newBeegoReviewManager(t, "beego-failure", nil)
	for _, check := range beegoReviewChecks {
		for _, missingManager := range []bool{false, true} {
			for _, mode := range []string{"default", "silent", "custom"} {
				t.Run(check.name+"/"+mode+map[bool]string{false: "/missing-token", true: "/missing-manager"}[missingManager], func(t *testing.T) {
					handled, laterFilters, failures := 0, 0, 0
					var opts []AuthOption
					if mode != "default" {
						opts = append(opts, WithFailFunc(func(c *beegocontext.Context, err error) {
							failures++
							if err == nil || !c.ResponseWriter.Started {
								t.Error("callback ran before failure was marked")
							}
							AuthMiddleware(context.Background(), WithBeforeAuthHandler(func(context.Context, *beegocontext.Context, *AuthHandleRequest) {
								t.Error("callback reopened authentication")
							}))(c)
							if mode == "custom" {
								c.Output.SetStatus(http.StatusConflict)
								if err := c.Output.JSON(map[string]string{"error": "custom"}, false, false); err != nil {
									t.Error(err)
								}
							}
						}))
					}
					var filters []web.FilterFunc
					if !missingManager {
						filters = append(filters, RegisterDTokenContextMiddleware(context.Background(), WithManager(mgr)))
					}
					filters = append(filters, check.build(context.Background(), opts...), func(*beegocontext.Context) { laterFilters++ })
					router := beegoReviewRouter(t, func(*beegocontext.Context) { handled++ }, filters...)
					rec := httptest.NewRecorder()
					router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/review", nil))
					if handled != 0 || laterFilters != 0 {
						t.Fatalf("controller=%d later filters=%d", handled, laterFilters)
					}
					switch mode {
					case "default":
						want := http.StatusUnauthorized
						if missingManager {
							want = http.StatusNotFound
						}
						if rec.Code != want || !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
							t.Fatalf("default response=%d %s", rec.Code, rec.Body.String())
						}
					case "silent":
						if failures != 1 || rec.Body.Len() != 0 {
							t.Fatalf("silent callback=%d body=%s", failures, rec.Body.String())
						}
					case "custom":
						if failures != 1 || rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "custom") {
							t.Fatalf("custom response=%d %s callbacks=%d", rec.Code, rec.Body.String(), failures)
						}
					}
				})
			}
		}
	}
	router := beegoReviewRouter(t, func(*beegocontext.Context) { t.Fatal("failed registration reached controller") }, RegisterDTokenContextMiddleware(context.Background(), WithAuthType("missing"), WithFailFunc(func(*beegocontext.Context, error) {})))
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/review", nil))
}

// TestBeegoReviewHookControlFlow verifies native output, abort, next and exit semantics. TestBeegoReviewHookControlFlow 验证原生响应、中止、放行及退出语义。
func TestBeegoReviewHookControlFlow(t *testing.T) {
	mgr := newBeegoReviewManager(t, "beego-hooks", nil)
	for _, check := range beegoReviewChecks {
		if check.name == "annotation" {
			continue
		}
		for _, mode := range []string{"next_twice", "exit_next", "abort", "native_output"} {
			if check.name == "access" && (mode == "next_twice" || mode == "exit_next") {
				continue
			}
			t.Run(check.name+"/"+mode, func(t *testing.T) {
				handled, failures := 0, 0
				stop := func(c *beegocontext.Context) {
					if mode == "native_output" {
						c.Output.SetStatus(http.StatusAccepted)
						if err := c.Output.JSON("handled", false, false); err != nil {
							t.Error(err)
						}
					} else {
						NewBeegoContext(c).Abort()
					}
				}
				filter := check.build(context.Background(), WithFailFunc(func(*beegocontext.Context, error) { failures++ }),
					WithBeforeAuthHandler(func(_ context.Context, c *beegocontext.Context, req *AuthHandleRequest) {
						switch mode {
						case "next_twice":
							req.Next()
							req.Next()
							req.Exit()
						case "exit_next":
							req.Exit()
							req.Next()
						default:
							stop(c)
							req.Next()
						}
					}), WithRouteAccessHandler(func(_ context.Context, c *beegocontext.Context, req *RouteAccessRequest) { stop(c); req.SkipAuth() }))
				router := beegoReviewRouter(t, func(*beegocontext.Context) { handled++ }, RegisterDTokenContextMiddleware(context.Background(), WithManager(mgr)), filter)
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/review", nil))
				want := 0
				if mode == "next_twice" {
					want = 1
				}
				if handled != want || failures != 0 {
					t.Fatalf("controller=%d want=%d failures=%d", handled, want, failures)
				}
				if mode == "native_output" && rec.Code != http.StatusAccepted {
					t.Fatalf("status=%d", rec.Code)
				}
			})
		}
	}
	next, exit := 0, 0
	req := newAuthHandleRequest(defaultAuthOptions(), func() { next++ }, func() { exit++ })
	req.Next()
	req.Next()
	req.Exit()
	if next != 1 || exit != 0 {
		t.Fatalf("next=%d exit=%d", next, exit)
	}
	req = newAuthHandleRequest(defaultAuthOptions(), func() { next++ }, func() { exit++ })
	req.Exit()
	req.Exit()
	req.Next()
	if next != 1 || exit != 1 {
		t.Fatalf("next=%d exit=%d", next, exit)
	}
}

// TestBeegoReviewManagerIsolation verifies priority, invalid managers and manager-local renewal. TestBeegoReviewManagerIsolation 验证选择优先级、无效 Manager 及实例内续期。
func TestBeegoReviewManagerIsolation(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	global := newBeegoReviewManager(t, "", nil)
	local := newBeegoReviewManager(t, "beego-local", nil)
	selected := newBeegoReviewManager(t, "beego-selected", nil)
	dtoken.SetManager(global)
	dtoken.SetManager(selected)
	beegoReviewLogin(t, global, "same-token")
	beegoReviewLogin(t, local, "same-token")
	c, _ := newBeegoBehaviorContext(http.MethodGet, "/review", local.GetConfig().TokenName, "same-token")
	getDContext(c, local)
	for _, tc := range []struct {
		explicit *manager.Manager
		authType string
		want     *manager.Manager
	}{
		{nil, "", local}, {nil, "beego-selected", selected}, {local, "missing", local},
	} {
		if got, err := resolveRequestManager(c, tc.explicit, tc.authType); err != nil || got != tc.want {
			t.Fatalf("manager=%p,%v want=%p", got, err, tc.want)
		}
	}
	if err := RenewTimeoutByContext(c, time.Minute); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		mgr     *manager.Manager
		timeout int64
	}{{global, 600}, {local, 60}} {
		info, err := tc.mgr.GetTokenInfo(context.Background(), "same-token")
		if err != nil || info.Timeout != tc.timeout {
			t.Fatalf("token info=%+v,%v want=%d", info, err, tc.timeout)
		}
	}
	var typedNil *DTokenContext
	c.Input.SetData(DTokenCtxKey, typedNil)
	if got, err := GetManagerByContext(c); err != nil || got != global {
		t.Fatalf("typed nil fallback=%p,%v", got, err)
	}
	local.CloseManager()
	for _, mgr := range []*manager.Manager{nil, local} {
		c.Input.SetData(DTokenCtxKey, corecontext.NewContext(NewBeegoContext(c), mgr))
		_, tokenErr := GetTokenValueByContext(c)
		_, loginErr := GetLoginIDByContext(c)
		_, infoErr := IntrospectTokenByContext(c)
		for _, err := range []error{tokenErr, loginErr, infoErr, RenewTimeoutByContext(c, time.Minute)} {
			if !errors.Is(err, ErrManagerNotFound) {
				t.Errorf("invalid cache=%v", err)
			}
		}
	}
}

// TestBeegoReviewTokenSources verifies Query, Body and route parameters remain separate. TestBeegoReviewTokenSources 验证 Query、Body 与路由参数保持独立。
func TestBeegoReviewTokenSources(t *testing.T) {
	for _, bodyOnly := range []bool{false, true} {
		b := NewBuilder().IsPrintBanner(false).IsLog(false).AutoRenew(false).AsyncEvent(false).
			IsReadHeader(false).IsReadCookie(false).IsReadQuery(!bodyOnly).IsReadBody(bodyOnly)
		mgr, err := b.Build()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(mgr.CloseManager)
		key := mgr.GetConfig().TokenName
		var multipartBody bytes.Buffer
		writer := multipart.NewWriter(&multipartBody)
		if err := writer.WriteField(key, "body-token"); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		for _, tc := range []struct{ query, contentType, body, wantQuery, wantBody string }{
			{"query-token", "application/x-www-form-urlencoded", url.Values{key: {"body-token"}}.Encode(), "query-token", "body-token"},
			{"query-token", "application/x-www-form-urlencoded", "other=value", "query-token", ""},
			{"", "application/x-www-form-urlencoded", url.Values{key: {"body-token"}}.Encode(), "", "body-token"},
			{"query-token", writer.FormDataContentType(), multipartBody.String(), "query-token", "body-token"},
			{"", "application/json", `{"` + key + `":"json-token"}`, "", ""},
		} {
			c, _ := newBeegoBehaviorContext(http.MethodPost, "/review?"+url.Values{key: {tc.query}}.Encode(), "", "")
			c.Request.Body = io.NopCloser(strings.NewReader(tc.body))
			c.Request.Header.Set("Content-Type", tc.contentType)
			c.Input.SetParam(key, "route-token")
			want := tc.wantQuery
			if bodyOnly {
				want = tc.wantBody
			}
			if got := getDContext(c, mgr).GetTokenValue(); got != want {
				t.Fatalf("bodyOnly=%v token=%q want=%q", bodyOnly, got, want)
			}
			if c.Request.MultipartForm != nil {
				_ = c.Request.MultipartForm.RemoveAll()
			}
		}
	}
	c, _ := newBeegoBehaviorContext(http.MethodGet, "/review?token=old", "", "")
	_ = c.Input.Query("token")
	c.Request = httptest.NewRequest(http.MethodGet, "/review?token=current", nil)
	if got := NewBeegoContext(c).GetQuery("token"); got != "current" {
		t.Fatalf("stale query=%q", got)
	}
}

// TestBeegoReviewAnnotations verifies nil rejection while preserving existing absent and ignored policies. TestBeegoReviewAnnotations 验证空注解拒绝并保留未传及忽略注解的原有策略。
func TestBeegoReviewAnnotations(t *testing.T) {
	mgr := newBeegoReviewManager(t, "beego-annotation", nil)
	beegoReviewLogin(t, mgr, "annotation-token")
	for _, tc := range []struct {
		name  string
		anns  []*Annotation
		token string
		want  error
	}{
		{"nil", []*Annotation{nil}, "annotation-token", ErrInvalidParam},
		{"absent", nil, "", ErrTokenExpired},
		{"empty", []*Annotation{{}}, "", nil},
		{"ignore", []*Annotation{{Ignore: true}}, "", nil},
		{"and", []*Annotation{{CheckPermission: []string{"read", "write"}}}, "annotation-token", ErrPermissionDenied},
		{"or", []*Annotation{{CheckPermission: []string{"read", "write"}, LogicType: LogicOr}}, "annotation-token", nil},
		{"role", []*Annotation{{CheckRole: []string{"admin"}}}, "annotation-token", ErrRoleDenied},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := newBeegoBehaviorContext(http.MethodGet, "/review", mgr.GetConfig().TokenName, tc.token)
			var gotErr error
			AccessMiddleware(context.Background(), WithManager(mgr), WithRouteAccessHandler(RouteAccessHandlerFromAnnotations(tc.anns...)), WithFailFunc(func(_ *beegocontext.Context, err error) { gotErr = err }))(c)
			if !errors.Is(gotErr, tc.want) || c.ResponseWriter.Started != (tc.want != nil) {
				t.Fatalf("error=%v want=%v started=%v", gotErr, tc.want, c.ResponseWriter.Started)
			}
		})
	}
	c, rec := newBeegoBehaviorContext(http.MethodGet, "/review", "", "")
	AccessMiddleware(context.Background(), WithRouteAccessHandler(RouteAccessHandlerFromAnnotations(nil)))(c)
	var payload struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusBadRequest || payload.Code != CodeBadRequest {
		t.Fatalf("nil annotation response=%d %s", rec.Code, rec.Body.String())
	}
}

// TestBeegoReviewAbortAndReset verifies independent adapters share native state and pooled requests reset it. TestBeegoReviewAbortAndReset 验证独立适配器共享原生状态且请求池重置时清除状态。
func TestBeegoReviewAbortAndReset(t *testing.T) {
	mgr := newBeegoReviewManager(t, "beego-reset", nil)
	c, _ := newBeegoBehaviorContext(http.MethodGet, "/review", "", "")
	getDContext(c, mgr)
	a := NewBeegoContext(c)
	a.Abort()
	if !NewBeegoContext(c).IsAborted() {
		t.Fatal("abort state was local to one adapter")
	}
	for _, check := range beegoReviewChecks {
		check.build(context.Background(), WithBeforeAuthHandler(func(context.Context, *beegocontext.Context, *AuthHandleRequest) { t.Error("aborted hook executed") }), WithFailFunc(func(*beegocontext.Context, error) { t.Error("aborted failure executed") }))(c)
	}
	RegisterDTokenContextMiddleware(context.Background(), WithAuthType("missing"), WithFailFunc(func(*beegocontext.Context, error) { t.Error("aborted registration executed") }))(c)
	c.Reset(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/review", nil))
	if a.IsAborted() {
		t.Fatal("pooled context retained an abort marker")
	}
	if _, ok := GetDTokenContext(c); ok {
		t.Fatal("pooled context retained a manager")
	}
}

// TestBeegoReviewResponseAndCookies verifies pending statuses and cookie options survive an abort marker. TestBeegoReviewResponseAndCookies 验证中止标记不影响暂存状态码和 Cookie 选项写入。
func TestBeegoReviewResponseAndCookies(t *testing.T) {
	c, rec := newBeegoBehaviorContext(http.MethodGet, "/review", "", "")
	a := NewBeegoContext(c)
	a.Abort()
	a.SetStatusCode(http.StatusAccepted)
	a.SetHeader("X-Result", "ok")
	const token = "custom+/%2B=token"
	a.SetCookieWithOptions(&adapter.CookieOptions{Name: "strict", Value: token, Path: "/", MaxAge: 60, SameSite: "Strict", Secure: true, HttpOnly: true})
	a.SetCookie("legacy", token, 60, "/", "", false, false)
	a.SetCookieWithOptions(&adapter.CookieOptions{Name: "expired", Path: "/", MaxAge: -1, SameSite: "None", Secure: true})
	if _, err := a.Write([]byte("done")); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusAccepted || rec.Body.String() != "done" || rec.Header().Get("X-Result") != "ok" {
		t.Fatalf("response=%d %s", rec.Code, rec.Body.String())
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 3 {
		t.Fatalf("cookies=%v", cookies)
	}
	if cookies[0].Value != token || cookies[0].SameSite != http.SameSiteStrictMode || !cookies[0].Secure || !cookies[0].HttpOnly || cookies[0].MaxAge != 60 {
		t.Fatalf("strict=%+v", cookies[0])
	}
	if cookies[1].SameSite != http.SameSiteLaxMode || cookies[1].Secure || cookies[1].HttpOnly {
		t.Fatalf("legacy=%+v", cookies[1])
	}
	if cookies[2].SameSite != http.SameSiteNoneMode || cookies[2].MaxAge != -1 {
		t.Fatalf("expired=%+v", cookies[2])
	}
	c.Request.AddCookie(cookies[0])
	if got := a.GetCookie("strict"); got != token {
		t.Fatalf("cookie=%q", got)
	}
}

// TestBeegoReviewBodyAndPeer verifies repeated body reads, error propagation and trusted connection facts. TestBeegoReviewBodyAndPeer 验证重复读取请求体、错误传播及可信连接信息。
func TestBeegoReviewBodyAndPeer(t *testing.T) {
	c, _ := newBeegoBehaviorContext(http.MethodPost, "https://example.com/review", "", "")
	c.Request.RemoteAddr = "[2001:db8::8]:8080"
	c.Request.TLS = nil
	c.Request.Header.Set("X-Forwarded-For", "203.0.113.8")
	c.Request.Header.Set("X-Real-IP", "203.0.113.9")
	c.Request.Header.Set("X-Forwarded-Proto", "https")
	a := NewBeegoContext(c)
	if a.GetClientIP() != "2001:db8::8" || a.IsTLS() {
		t.Fatal("forwarding data changed connection facts")
	}
	c.Request.TLS = &tls.ConnectionState{}
	if !a.IsTLS() {
		t.Fatal("TLS connection missed")
	}
	c.Request.Body = io.NopCloser(strings.NewReader("body"))
	for i := 0; i < 2; i++ {
		if body, err := a.GetBody(); err != nil || string(body) != "body" {
			t.Fatalf("body=%q,%v", body, err)
		}
	}
	c.Input.RequestBody = []byte("cached")
	body, err := a.GetBody()
	if err != nil || string(body) != "cached" {
		t.Fatalf("cached body=%q,%v", body, err)
	}
	body[0] = 'x'
	if string(c.Input.RequestBody) != "cached" {
		t.Fatal("body read modified Beego's cache")
	}
	c.Input.RequestBody = nil
	wantErr := errors.New("stream failed")
	c.Request.Body = io.NopCloser(iotest.ErrReader(wantErr))
	if _, err := a.GetBody(); !errors.Is(err, wantErr) {
		t.Fatalf("body error=%v", err)
	}
}
