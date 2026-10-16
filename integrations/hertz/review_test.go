package hertz

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"testing"
	"testing/iotest"
	"time"

	"github.com/Zany2/dtoken-go/com/storage/memory"
	"github.com/Zany2/dtoken-go/core/adapter"
	corecontext "github.com/Zany2/dtoken-go/core/context"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/dtoken"
	hertzapp "github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/network"
	"github.com/cloudwego/hertz/pkg/protocol"
)

// hertzReviewChecks covers each authentication entry point. hertzReviewChecks 覆盖各个鉴权入口。
var hertzReviewChecks = []struct {
	name  string
	build func(context.Context, ...AuthOption) hertzapp.HandlerFunc
}{
	{"login", AuthMiddleware},
	{"access", AccessMiddleware},
	{"permission", func(c context.Context, opts ...AuthOption) hertzapp.HandlerFunc {
		return PermissionMiddleware(c, []string{"read"}, opts...)
	}},
	{"role", func(c context.Context, opts ...AuthOption) hertzapp.HandlerFunc {
		return RoleMiddleware(c, []string{"reader"}, opts...)
	}},
	{"annotation", func(c context.Context, opts ...AuthOption) hertzapp.HandlerFunc {
		options := defaultAuthOptions()
		for _, opt := range opts {
			opt(options)
		}
		return GetHandler(c, nil, options.FailFunc, &Annotation{
			AuthType: options.AuthType, CheckLogin: true, CheckPermission: []string{"read"},
		})
	}},
}

// newHertzReviewManager creates an isolated manager without asynchronous maintenance. newHertzReviewManager 创建无异步维护的独立 Manager。
func newHertzReviewManager(t *testing.T, authType string, storage adapter.Storage) *manager.Manager {
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

// hertzReviewLogin seeds the credentials shared by authentication cases. hertzReviewLogin 准备共用鉴权用例所需的凭据。
func hertzReviewLogin(t *testing.T, mgr *manager.Manager, token string) {
	t.Helper()
	c := context.Background()
	if _, err := mgr.LoginWithOptions(c, manager.LoginOptions{LoginID: "reader", Token: token}); err != nil {
		t.Fatal(err)
	}
	if err := mgr.AddPermissions(c, "reader", []string{"read"}); err != nil {
		t.Fatal(err)
	}
	if err := mgr.AddRoles(c, "reader", []string{"reader"}); err != nil {
		t.Fatal(err)
	}
}

// hertzReviewStorage observes contexts at the storage boundary. hertzReviewStorage 在存储边界观察上下文。
type hertzReviewStorage struct {
	adapter.Storage
	inspect func(context.Context)
}

// Get observes propagation and honors request cancellation. Get 观察上下文传播并响应请求取消。
func (s *hertzReviewStorage) Get(c context.Context, key string) (any, error) {
	if s.inspect != nil {
		s.inspect(c)
	}
	if err := c.Err(); err != nil {
		return nil, err
	}
	return s.Storage.Get(c, key)
}

// TestHertzReviewRequestContext verifies values, deadlines and cancellation reach checks, hooks and facades. TestHertzReviewRequestContext 验证值、截止时间及取消信号传入鉴权、钩子和门面。
func TestHertzReviewRequestContext(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	for _, check := range hertzReviewChecks {
		t.Run(check.name, func(t *testing.T) {
			storage := &hertzReviewStorage{Storage: memory.NewStorage()}
			mgr := newHertzReviewManager(t, "hertz-request", storage)
			const token = "request-token"
			hertzReviewLogin(t, mgr, token)
			registration, stopRegistration := context.WithCancel(context.Background())
			stopRegistration()
			for _, canceled := range []bool{false, true} {
				type requestKey struct{}
				c, cancel := context.WithTimeout(context.WithValue(context.Background(), requestKey{}, "request"), time.Minute)
				t.Cleanup(cancel)
				if canceled {
					cancel()
				}
				reads, hooks, handled, failures := 0, 0, 0, 0
				storage.inspect = func(got context.Context) {
					reads++
					if got != c {
						t.Error("storage did not receive the current handler context")
					}
				}
				inspectHook := func(got context.Context, ctx *hertzapp.RequestContext) {
					hooks++
					if got != c || requestContext(ctx) != c {
						t.Error("hook context was lost")
					}
				}
				ctx := hertzapp.NewContext(0)
				ctx.Request.Header.Set(mgr.GetConfig().TokenName, token)
				ctx.SetHandlers(hertzapp.HandlersChain{
					RegisterDTokenContextMiddleware(registration, WithManager(mgr)),
					check.build(registration,
						WithBeforeAuthHandler(func(got context.Context, ctx *hertzapp.RequestContext, _ *AuthHandleRequest) { inspectHook(got, ctx) }),
						WithRouteAccessHandler(func(got context.Context, ctx *hertzapp.RequestContext, _ *RouteAccessRequest) { inspectHook(got, ctx) }),
						WithFailFunc(func(got context.Context, ctx *hertzapp.RequestContext, err error) {
							failures++
							if got != c || requestContext(ctx) != c || err == nil {
								t.Error("failure lost its request context or error")
							}
						})),
					func(got context.Context, ctx *hertzapp.RequestContext) {
						handled++
						if got != c {
							t.Error("downstream context changed")
						}
						if id, err := GetLoginIDByContext(ctx); err != nil || id != "reader" {
							t.Errorf("login ID=%q,%v", id, err)
						}
					},
				})
				ctx.Next(c)
				if reads == 0 || check.name != "annotation" && hooks != 1 {
					t.Fatalf("reads=%d hooks=%d", reads, hooks)
				}
				if canceled {
					if handled != 0 || failures != 1 || !ctx.IsAborted() {
						t.Fatalf("canceled request passed: handled=%d failures=%d", handled, failures)
					}
				} else if handled != 1 || failures != 0 {
					t.Fatalf("valid request failed: handled=%d failures=%d", handled, failures)
				}
			}
		})
	}
}

// TestHertzReviewContextScopes verifies nested middleware restores the caller's context. TestHertzReviewContextScopes 验证嵌套中间件返回后恢复调用方上下文。
func TestHertzReviewContextScopes(t *testing.T) {
	mgr := newHertzReviewManager(t, "hertz-scopes", nil)
	type scopeKey struct{}
	outer := context.WithValue(context.Background(), scopeKey{}, "outer")
	inner, cancel := context.WithCancel(context.WithValue(outer, scopeKey{}, "inner"))
	defer cancel()
	ctx := hertzapp.NewContext(0)
	visited := 0
	ctx.SetHandlers(hertzapp.HandlersChain{
		RegisterDTokenContextMiddleware(context.Background(), WithManager(mgr)),
		func(c context.Context, ctx *hertzapp.RequestContext) {
			if requestContext(ctx) != outer {
				t.Error("outer context missing")
			}
			ctx.Next(inner)
			visited++
			if requestContext(ctx) != outer {
				t.Error("inner context leaked into outer middleware")
			}
		},
		GetHandler(context.Background(), nil, nil, &Annotation{Ignore: true}),
		func(c context.Context, ctx *hertzapp.RequestContext) {
			visited++
			if c != inner || requestContext(ctx) != inner {
				t.Error("ignored annotation did not bind current context")
			}
		},
	})
	ctx.Next(outer)
	if visited != 2 || requestContext(ctx) != context.Background() {
		t.Fatalf("visited=%d, context was not restored", visited)
	}
	ctx.Reset()
	if requestContext(ctx) != context.Background() {
		t.Fatal("pooled request retained an old context")
	}
}

// TestHertzReviewManagerSelection verifies precedence and invalid cached manager handling. TestHertzReviewManagerSelection 验证 Manager 优先级和无效缓存处理。
func TestHertzReviewManagerSelection(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	global := newHertzReviewManager(t, "", nil)
	selected := newHertzReviewManager(t, "hertz-selected", nil)
	local := newHertzReviewManager(t, "hertz-local", nil)
	dtoken.SetManager(global)
	dtoken.SetManager(selected)
	ctx := hertzapp.NewContext(0)
	getDTokenContext(ctx, local)
	for _, tc := range []struct {
		explicit *manager.Manager
		authType string
		want     *manager.Manager
	}{
		{nil, "", local}, {nil, "hertz-selected", selected}, {local, "missing", local},
	} {
		if got, err := resolveRequestManager(ctx, tc.explicit, tc.authType); err != nil || got != tc.want {
			t.Fatalf("manager=%p,%v want=%p", got, err, tc.want)
		}
	}
	var typedNil *corecontext.DTokenContext
	ctx.Set(DTokenCtxKey, typedNil)
	if got, err := GetManagerByContext(ctx); err != nil || got != global {
		t.Fatalf("typed nil fallback=%p,%v", got, err)
	}
	local.CloseManager()
	for _, mgr := range []*manager.Manager{nil, local} {
		ctx.Set(DTokenCtxKey, corecontext.NewContext(NewHertzContext(ctx), mgr))
		_, tokenErr := GetTokenValueByContext(ctx)
		_, loginErr := GetLoginIDByContext(ctx)
		_, infoErr := IntrospectTokenByContext(ctx)
		renewErr := RenewTimeoutByContext(ctx, time.Minute)
		for _, err := range []error{tokenErr, loginErr, infoErr, renewErr} {
			if !errors.Is(err, ErrManagerNotFound) {
				t.Errorf("error=%v, want ErrManagerNotFound", err)
			}
		}
	}
	if _, err := resolveRequestManager(ctx, nil, ""); !errors.Is(err, ErrManagerNotFound) {
		t.Fatalf("closed cache=%v", err)
	}
	if _, err := resolveRequestManager(ctx, local, "hertz-selected"); !errors.Is(err, ErrManagerNotFound) {
		t.Fatalf("closed explicit manager=%v", err)
	}
}

// TestHertzReviewFailureStopsNext verifies failure callbacks cannot enter business handlers. TestHertzReviewFailureStopsNext 验证失败回调无法继续进入业务处理器。
func TestHertzReviewFailureStopsNext(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	mgr := newHertzReviewManager(t, "hertz-fail", nil)
	for _, check := range hertzReviewChecks {
		for _, missingManager := range []bool{false, true} {
			name := check.name + "/missing-token"
			if missingManager {
				name = check.name + "/missing-manager"
			}
			t.Run(name, func(t *testing.T) {
				ctx := hertzapp.NewContext(0)
				if !missingManager {
					getDTokenContext(ctx, mgr)
				}
				failures, handled := 0, 0
				ctx.SetHandlers(hertzapp.HandlersChain{
					check.build(context.Background(), WithFailFunc(func(c context.Context, ctx *hertzapp.RequestContext, err error) {
						failures++
						if err == nil || !ctx.IsAborted() {
							t.Error("chain was not aborted before failure callback")
						}
						ctx.Next(c)
					})),
					func(context.Context, *hertzapp.RequestContext) { handled++ },
				})
				ctx.Next(context.Background())
				if failures != 1 || handled != 0 {
					t.Fatalf("failures=%d handled=%d", failures, handled)
				}
			})
		}
	}
	ctx := hertzapp.NewContext(0)
	handled := 0
	ctx.SetHandlers(hertzapp.HandlersChain{
		RegisterDTokenContextMiddleware(context.Background(), WithAuthType("missing"), WithFailFunc(func(c context.Context, ctx *hertzapp.RequestContext, _ error) { ctx.Next(c) })),
		func(context.Context, *hertzapp.RequestContext) { handled++ },
	})
	ctx.Next(context.Background())
	if handled != 0 || !ctx.IsAborted() {
		t.Fatal("failed registration continued the chain")
	}
}

// TestHertzReviewHooks verifies abort, exit and at-most-once continuation. TestHertzReviewHooks 验证中止、退出及最多一次放行。
func TestHertzReviewHooks(t *testing.T) {
	mgr := newHertzReviewManager(t, "hertz-hooks", nil)
	for _, check := range hertzReviewChecks {
		if check.name == "annotation" {
			continue
		}
		for _, mode := range []string{"abort", "adapter_abort", "abort_next", "exit_next", "next_twice"} {
			if check.name == "access" && mode != "abort" && mode != "adapter_abort" {
				continue
			}
			t.Run(check.name+"/"+mode, func(t *testing.T) {
				ctx := hertzapp.NewContext(0)
				getDTokenContext(ctx, mgr)
				handled, failures := 0, 0
				ctx.SetHandlers(hertzapp.HandlersChain{
					check.build(context.Background(),
						WithFailFunc(func(context.Context, *hertzapp.RequestContext, error) { failures++ }),
						WithBeforeAuthHandler(func(_ context.Context, ctx *hertzapp.RequestContext, req *AuthHandleRequest) {
							switch mode {
							case "next_twice":
								req.Next()
								req.Next()
							case "exit_next":
								req.Exit()
								req.Next()
							case "adapter_abort":
								getDTokenContext(ctx, mgr).GetRequestContext().Abort()
							default:
								ctx.Abort()
								if mode == "abort_next" {
									req.Next()
								}
							}
						}),
						WithRouteAccessHandler(func(_ context.Context, ctx *hertzapp.RequestContext, req *RouteAccessRequest) {
							if mode == "adapter_abort" {
								getDTokenContext(ctx, mgr).GetRequestContext().Abort()
							} else {
								ctx.Abort()
							}
							req.SkipAuth()
						})),
					func(context.Context, *hertzapp.RequestContext) { handled++ },
				})
				ctx.Next(context.Background())
				want := 0
				if mode == "next_twice" {
					want = 1
				}
				if handled != want || failures != 0 {
					t.Fatalf("handled=%d want=%d failures=%d", handled, want, failures)
				}
			})
		}
	}
	next, exits := 0, 0
	req := newAuthHandleRequest(defaultAuthOptions(), func() { next++ }, func() { exits++ })
	req.Next()
	req.Next()
	req.Exit()
	if next != 1 || exits != 0 {
		t.Fatalf("next=%d exits=%d", next, exits)
	}
	req = newAuthHandleRequest(defaultAuthOptions(), func() { next++ }, func() { exits++ })
	req.Exit()
	req.Exit()
	req.Next()
	if next != 1 || exits != 1 {
		t.Fatalf("next=%d exits=%d", next, exits)
	}
}

// TestHertzReviewAnnotationsAndAbortedEntry verifies invalid annotations and pre-aborted requests. TestHertzReviewAnnotationsAndAbortedEntry 验证空注解以及入口前已中止的请求。
func TestHertzReviewAnnotationsAndAbortedEntry(t *testing.T) {
	ctx := hertzapp.NewContext(0)
	handled := 0
	GetHandler(context.Background(), func(context.Context, *hertzapp.RequestContext) { handled++ }, nil, nil)(context.Background(), ctx)
	var payload struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &payload); err != nil {
		t.Fatal(err)
	}
	if handled != 0 || !ctx.IsAborted() || ctx.Response.StatusCode() != http.StatusBadRequest || payload.Code != CodeBadRequest {
		t.Fatalf("invalid annotation response=%s", ctx.Response.Body())
	}
	for _, ann := range []*Annotation{nil, {}, {Ignore: true}, {CheckLogin: true}} {
		GetHandler(context.Background(), func(context.Context, *hertzapp.RequestContext) { handled++ }, func(context.Context, *hertzapp.RequestContext, error) { handled++ }, ann)(context.Background(), ctx)
	}
	RegisterDTokenContextMiddleware(context.Background(), WithFailFunc(func(context.Context, *hertzapp.RequestContext, error) { handled++ }))(context.Background(), ctx)
	for _, check := range hertzReviewChecks {
		check.build(context.Background(), WithBeforeAuthHandler(func(context.Context, *hertzapp.RequestContext, *AuthHandleRequest) { handled++ }), WithRouteAccessHandler(func(context.Context, *hertzapp.RequestContext, *RouteAccessRequest) { handled++ }), WithFailFunc(func(context.Context, *hertzapp.RequestContext, error) { handled++ }))(context.Background(), ctx)
	}
	if handled != 0 {
		t.Fatalf("aborted entry ran %d callbacks", handled)
	}
}

// TestHertzReviewRenewalIsolation keeps renewal within the cached manager's namespace. TestHertzReviewRenewalIsolation 验证续期保持在缓存 Manager 的命名空间内。
func TestHertzReviewRenewalIsolation(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	global := newHertzReviewManager(t, "", nil)
	local := newHertzReviewManager(t, "hertz-renew", nil)
	dtoken.SetManager(global)
	const token = "same-token"
	hertzReviewLogin(t, global, token)
	hertzReviewLogin(t, local, token)
	ctx := hertzapp.NewContext(0)
	ctx.Request.Header.Set(local.GetConfig().TokenName, token)
	getDTokenContext(ctx, local)
	if err := RenewTimeoutByContext(ctx, time.Minute); err != nil {
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
}

// TestHertzReviewCookieRoundTrip verifies escaped custom tokens survive login cookies and cookie options remain independent. TestHertzReviewCookieRoundTrip 验证自定义 Token 经 Cookie 往返不变，且各 Cookie 选项独立。
func TestHertzReviewCookieRoundTrip(t *testing.T) {
	mgr := newHertzReviewManager(t, "hertz-cookie", nil)
	ctx := hertzapp.NewContext(0)
	getDTokenContext(ctx, mgr)
	const token = "custom+/%2B=token"
	if got, err := LoginWithCookieOptionsByContext(ctx, manager.LoginOptions{LoginID: "cookie-user", Token: token}); err != nil || got != token {
		t.Fatalf("LoginWithCookie=%q,%v", got, err)
	}
	var cookie protocol.Cookie
	cookie.SetKey(mgr.GetConfig().TokenName)
	if !ctx.Response.Header.Cookie(&cookie) {
		t.Fatal("login cookie missing")
	}
	ctx.Request.Header.SetCookie(mgr.GetConfig().TokenName, string(cookie.Value()))
	if got, err := GetLoginIDByContext(ctx); err != nil || got != "cookie-user" {
		t.Fatalf("cookie login=%q,%v", got, err)
	}
	adapterCtx := NewHertzContext(ctx)
	adapterCtx.SetCookieWithOptions(&adapter.CookieOptions{Name: "strict", Value: token, Path: "/", MaxAge: 60, SameSite: "Strict", Secure: true, HttpOnly: true})
	adapterCtx.SetCookie("legacy", token, 60, "/", "", false, false)
	adapterCtx.SetCookieWithOptions(&adapter.CookieOptions{Name: "expired", Path: "/", MaxAge: -1, SameSite: "None", Secure: true})
	for _, tc := range []struct {
		name             string
		sameSite         http.SameSite
		secure, httpOnly bool
		maxAge           int
	}{
		{"strict", http.SameSiteStrictMode, true, true, 60}, {"legacy", http.SameSiteLaxMode, false, false, 60}, {"expired", http.SameSiteNoneMode, true, false, -1},
	} {
		var raw string
		ctx.Response.Header.VisitAllCookie(func(key, value []byte) {
			if string(key) == tc.name {
				raw = string(value)
			}
		})
		response := &http.Response{Header: http.Header{"Set-Cookie": {raw}}}
		cookies := response.Cookies()
		if len(cookies) != 1 {
			t.Fatalf("invalid cookie: %s", raw)
		}
		got := cookies[0]
		if got.SameSite != tc.sameSite || got.Secure != tc.secure || got.HttpOnly != tc.httpOnly || got.MaxAge != tc.maxAge {
			t.Fatalf("cookie=%+v", got)
		}
	}
	ctx.Request.Header.SetCookie("malformed", "bad%value")
	if got := adapterCtx.GetCookie("malformed"); got != "bad%value" {
		t.Fatalf("malformed cookie=%q", got)
	}
}

// hertzReviewConn supplies a peer address without performing network I/O. hertzReviewConn 提供连接地址，不执行网络 I/O。
type hertzReviewConn struct {
	network.Conn
	remote net.Addr
}

// RemoteAddr returns the test peer. RemoteAddr 返回测试端地址。
func (c hertzReviewConn) RemoteAddr() net.Addr { return c.remote }

// hertzReviewTLSConn implements the same TLS contract used by Hertz's engine. hertzReviewTLSConn 实现 Hertz 引擎使用的 TLS 接口。
type hertzReviewTLSConn struct{ hertzReviewConn }

// Handshake satisfies Hertz's TLS interface. Handshake 满足 Hertz TLS 接口。
func (hertzReviewTLSConn) Handshake() error { return nil }

// ConnectionState reports a completed test handshake. ConnectionState 表示测试握手已完成。
func (hertzReviewTLSConn) ConnectionState() tls.ConnectionState {
	return tls.ConnectionState{HandshakeComplete: true}
}

// TestHertzReviewTLSAndClientIP verifies connection-based TLS and configured proxy trust. TestHertzReviewTLSAndClientIP 验证基于连接的 TLS 判断及配置的代理信任策略。
func TestHertzReviewTLSAndClientIP(t *testing.T) {
	ctx := hertzapp.NewContext(0)
	plain := hertzReviewConn{remote: &net.TCPAddr{IP: net.ParseIP("192.0.2.8"), Port: 8080}}
	ctx.SetConn(plain)
	ctx.Request.SetRequestURI("https://example.com/private")
	ctx.Request.Header.Set("X-Forwarded-Proto", "https")
	adapterCtx := NewHertzContext(ctx)
	if adapterCtx.IsTLS() {
		t.Fatal("forged HTTPS URI changed plaintext transport into TLS")
	}
	ctx.SetConn(hertzReviewTLSConn{plain})
	ctx.Request.SetRequestURI("http://example.com/private")
	if !adapterCtx.IsTLS() {
		t.Fatal("TLS connection was not recognized")
	}
	ctx.SetConn(plain)
	ctx.Request.Header.Set("X-Forwarded-For", "198.51.100.9")
	ctx.SetClientIPFunc(hertzapp.ClientIPWithOption(hertzapp.ClientIPOptions{}))
	if got := adapterCtx.GetClientIP(); got != "192.0.2.8" {
		t.Fatalf("direct IP=%q", got)
	}
	_, trusted, err := net.ParseCIDR("192.0.2.0/24")
	if err != nil {
		t.Fatal(err)
	}
	ctx.SetClientIPFunc(hertzapp.ClientIPWithOption(hertzapp.ClientIPOptions{TrustedCIDRs: []*net.IPNet{trusted}, RemoteIPHeaders: []string{"X-Forwarded-For"}}))
	if got := adapterCtx.GetClientIP(); got != "198.51.100.9" {
		t.Fatalf("trusted proxy IP=%q", got)
	}
}

// TestHertzReviewRequestData verifies body-only token extraction and owned token strings. TestHertzReviewRequestData 验证仅从 Body 读取 Token 以及 Token 字符串拥有独立内存。
func TestHertzReviewRequestData(t *testing.T) {
	mgr, err := NewBuilder().IsPrintBanner(false).IsLog(false).AutoRenew(false).AsyncEvent(false).
		IsReadHeader(false).IsReadCookie(false).IsReadQuery(false).IsReadBody(true).Build()
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
	for _, tc := range []struct{ contentType, body, want string }{
		{"application/x-www-form-urlencoded", "other=value", ""},
		{"application/x-www-form-urlencoded", url.Values{key: {"body-token"}}.Encode(), "body-token"},
		{writer.FormDataContentType(), multipartBody.String(), "body-token"},
		{"application/json", `{"` + key + `":"json-token"}`, ""},
	} {
		ctx := hertzapp.NewContext(0)
		ctx.Request.SetMethod(http.MethodPost)
		ctx.Request.SetRequestURI("/?" + url.Values{key: {"query-token"}}.Encode())
		ctx.Request.Header.SetContentType(tc.contentType)
		ctx.Request.SetBodyString(tc.body)
		if got := getDTokenContext(ctx, mgr).GetTokenValue(); got != tc.want {
			t.Fatalf("body token=%q want=%q", got, tc.want)
		}
		ctx.Request.Reset()
	}
	ctx := hertzapp.NewContext(0)
	ctx.Request.SetRequestURI("/?token=original")
	ctx.Request.Header.Set("X-Token", "original")
	ctx.Request.Header.SetCookie("token", "original")
	ctx.Request.Header.SetContentType("application/x-www-form-urlencoded")
	ctx.Request.SetBodyString("token=original")
	adapterCtx := NewHertzContext(ctx)
	values := []string{adapterCtx.GetHeader("X-Token"), adapterCtx.GetQuery("token"), adapterCtx.GetCookie("token"), adapterCtx.GetPostForm("token")}
	for _, buffer := range [][]byte{ctx.GetHeader("X-Token"), ctx.QueryArgs().Peek("token"), ctx.Cookie("token"), ctx.PostArgs().Peek("token")} {
		for i := range buffer {
			buffer[i] = 'x'
		}
	}
	for _, value := range values {
		if value != "original" {
			t.Fatalf("token string reused framework memory: %q", value)
		}
	}
	ctx.Request.SetBodyString("raw-body")
	ctx.Request.Header.Set("Content-Encoding", "gzip")
	if body, err := adapterCtx.GetBody(); err != nil || string(body) != "raw-body" {
		t.Fatalf("raw body=%q,%v", body, err)
	}
	wantErr := errors.New("stream failed")
	ctx.Request.SetBodyStream(iotest.ErrReader(wantErr), -1)
	if _, err := adapterCtx.GetBody(); !errors.Is(err, wantErr) {
		t.Fatalf("body error=%v", err)
	}
}
