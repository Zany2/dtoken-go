package fiber

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/com/storage/memory"
	"github.com/Zany2/dtoken-go/core/adapter"
	corecontext "github.com/Zany2/dtoken-go/core/context"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/dtoken"
	gofiber "github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
)

// fiberReviewChecks exercises all auth entry points through Fiber routing. fiberReviewChecks 通过 Fiber 路由覆盖全部鉴权入口。
var fiberReviewChecks = []struct {
	name  string
	build func(context.Context, ...AuthOption) gofiber.Handler
}{
	{"login", AuthMiddleware},
	{"access", AccessMiddleware},
	{"permission", func(ctx context.Context, opts ...AuthOption) gofiber.Handler {
		return PermissionMiddleware(ctx, []string{"read"}, opts...)
	}},
	{"role", func(ctx context.Context, opts ...AuthOption) gofiber.Handler {
		return RoleMiddleware(ctx, []string{"reader"}, opts...)
	}},
	{"annotation", func(ctx context.Context, opts ...AuthOption) gofiber.Handler {
		options := defaultAuthOptions()
		for _, opt := range opts {
			opt(options)
		}
		return GetHandler(ctx, nil, options.FailFunc, &Annotation{
			AuthType: options.AuthType, CheckLogin: true, CheckPermission: []string{"read"},
		})
	}},
}

// fiberReviewRequest creates a fasthttp request for synchronous Fiber dispatch. fiberReviewRequest 创建供 Fiber 同步处理的 fasthttp 请求。
func fiberReviewRequest(method, uri string) *fasthttp.RequestCtx {
	var req fasthttp.Request
	req.Header.SetMethod(method)
	req.SetRequestURI(uri)
	raw := &fasthttp.RequestCtx{}
	raw.Init(&req, nil, nil)
	return raw
}

// newFiberReviewContext acquires a request context for direct facade checks. newFiberReviewContext 获取用于直接验证门面的请求上下文。
func newFiberReviewContext(t *testing.T) *gofiber.Ctx {
	t.Helper()
	app := gofiber.New()
	c := app.AcquireCtx(fiberReviewRequest(http.MethodGet, "/"))
	t.Cleanup(func() { app.ReleaseCtx(c) })
	return c
}

// newFiberReviewManager creates an isolated manager without background maintenance. newFiberReviewManager 创建不执行后台维护的独立 Manager。
func newFiberReviewManager(t *testing.T, authType string, storage adapter.Storage) *manager.Manager {
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

// fiberReviewLogin creates a token with the access rules used by the shared cases. fiberReviewLogin 创建具备共用用例所需权限的 Token。
func fiberReviewLogin(t *testing.T, mgr *manager.Manager, token string) {
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

// TestFiberReviewUserContext verifies user values, deadlines and cancellation reach checks and facades. TestFiberReviewUserContext 验证用户上下文值、截止时间与取消信号传入鉴权及门面。
func TestFiberReviewUserContext(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	for _, check := range fiberReviewChecks {
		t.Run(check.name, func(t *testing.T) {
			storage := &fiberReviewStorage{Storage: memory.NewStorage()}
			mgr := newFiberReviewManager(t, "fiber-request", storage)
			const token = "request-token"
			fiberReviewLogin(t, mgr, token)
			var expected context.Context
			reads, hooks, handled := 0, 0, 0
			storage.inspect = func(ctx context.Context) {
				reads++
				if ctx != expected {
					t.Error("storage did not receive the request UserContext")
				}
			}
			inspectHook := func(ctx context.Context) {
				hooks++
				if ctx != expected {
					t.Error("hook did not receive the request UserContext")
				}
			}
			app := gofiber.New()
			app.Use(func(c *gofiber.Ctx) error { c.SetUserContext(expected); return c.Next() })
			app.Use(RegisterDTokenContextMiddleware(context.Background(), WithManager(mgr)))
			app.Use(check.build(context.Background(),
				WithBeforeAuthHandler(func(ctx context.Context, _ *gofiber.Ctx, _ *AuthHandleRequest) { inspectHook(ctx) }),
				WithRouteAccessHandler(func(ctx context.Context, _ *gofiber.Ctx, _ *RouteAccessRequest) { inspectHook(ctx) }),
			))
			app.Get("/", func(c *gofiber.Ctx) error {
				handled++
				if id, err := GetLoginIDByContext(c); err != nil || id != "reader" {
					t.Errorf("GetLoginIDByContext=%q,%v", id, err)
				}
				return c.SendStatus(http.StatusNoContent)
			})
			for _, canceled := range []bool{false, true} {
				type requestKey struct{}
				ctx, cancel := context.WithTimeout(context.WithValue(context.Background(), requestKey{}, canceled), time.Minute)
				t.Cleanup(cancel)
				if canceled {
					cancel()
				}
				expected = ctx
				reads, hooks, handled = 0, 0, 0
				raw := fiberReviewRequest(http.MethodGet, "/")
				raw.Request.Header.Set(mgr.GetConfig().TokenName, token)
				app.Handler()(raw)
				if reads == 0 || check.name != "annotation" && hooks != 1 {
					t.Fatalf("reads=%d hooks=%d, expected request checks", reads, hooks)
				}
				status := raw.Response.StatusCode()
				if canceled {
					if handled != 0 || status < 400 {
						t.Fatalf("canceled request reached handler: handled=%d status=%d", handled, status)
					}
				} else if handled != 1 || status != http.StatusNoContent {
					t.Fatalf("valid request failed: handled=%d status=%d", handled, status)
				}
			}
		})
	}
}

// fiberReviewStorage records contexts and honors cancellation. fiberReviewStorage 记录上下文并响应取消信号。
type fiberReviewStorage struct {
	adapter.Storage
	inspect func(context.Context)
}

// Get observes the context passed to storage reads. Get 观察传入存储读取的上下文。
func (s *fiberReviewStorage) Get(ctx context.Context, key string) (any, error) {
	if s.inspect != nil {
		s.inspect(ctx)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.Storage.Get(ctx, key)
}

// TestFiberReviewManagerSelection verifies precedence, inheritance and closed-manager rejection. TestFiberReviewManagerSelection 验证选择优先级、请求继承与已关闭 Manager 的拒绝行为。
func TestFiberReviewManagerSelection(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	global := newFiberReviewManager(t, "", nil)
	local := newFiberReviewManager(t, "fiber-local", nil)
	closed := newFiberReviewManager(t, "fiber-closed", nil)
	closed.CloseManager()
	dtoken.SetManager(global)
	const token = "same-token"
	fiberReviewLogin(t, global, token)
	fiberReviewLogin(t, local, token)
	for _, check := range fiberReviewChecks {
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
				app := gofiber.New()
				app.Use(func(c *gofiber.Ctx) error {
					if tc.cached != nil {
						getDTokenContext(c, tc.cached)
					}
					return c.Next()
				})
				app.Use(check.build(context.Background(), tc.opts...))
				called := false
				app.Get("/", func(c *gofiber.Ctx) error {
					called = true
					if got, err := GetManagerByContext(c); err != nil || got != tc.want {
						t.Errorf("manager=%p,%v, want %p", got, err, tc.want)
					}
					return c.SendStatus(http.StatusNoContent)
				})
				raw := fiberReviewRequest(http.MethodGet, "/")
				raw.Request.Header.Set(local.GetConfig().TokenName, token)
				app.Handler()(raw)
				wantStatus := http.StatusNoContent
				if tc.want == nil {
					wantStatus = http.StatusNotFound
				}
				if raw.Response.StatusCode() != wantStatus || called != (tc.want != nil) {
					t.Fatalf("status=%d called=%v, want status=%d", raw.Response.StatusCode(), called, wantStatus)
				}
			})
		}
	}
}

// TestFiberReviewRenewalUsesRequestManager protects identical tokens in different namespaces. TestFiberReviewRenewalUsesRequestManager 验证不同认证空间内同值 Token 的续期归属。
func TestFiberReviewRenewalUsesRequestManager(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	global := newFiberReviewManager(t, "", nil)
	local := newFiberReviewManager(t, "fiber-renew", nil)
	dtoken.SetManager(global)
	const token = "renew-token"
	fiberReviewLogin(t, global, token)
	fiberReviewLogin(t, local, token)
	c := newFiberReviewContext(t)
	c.Request().Header.Set(local.GetConfig().TokenName, token)
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
			t.Fatalf("token info=%+v,%v, want timeout=%d", info, err, tc.want)
		}
	}
}

// TestFiberReviewCachedContextValidation rejects unusable cached managers and replaces typed nils. TestFiberReviewCachedContextValidation 拒绝不可用的缓存 Manager 并替换带类型空值。
func TestFiberReviewCachedContextValidation(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	global := newFiberReviewManager(t, "", nil)
	dtoken.SetManager(global)
	c := newFiberReviewContext(t)
	var typedNil *corecontext.DTokenContext
	c.Locals(DTokenCtxKey, typedNil)
	if got, ok := GetDTokenContext(c); got != nil || ok {
		t.Fatalf("typed nil lookup=%v,%v", got, ok)
	}
	if got, err := GetManagerByContext(c); err != nil || got != global {
		t.Fatalf("typed nil fallback=%p,%v", got, err)
	}
	c.Locals(DTokenCtxKey, typedNil)
	if got := getDTokenContext(c, global); got == nil || got.GetManager() != global {
		t.Fatal("typed nil cache was not replaced")
	}
	closed := newFiberReviewManager(t, "fiber-cached-closed", nil)
	closed.CloseManager()
	for _, mgr := range []*manager.Manager{nil, closed} {
		c.Locals(DTokenCtxKey, corecontext.NewContext(NewFiberContext(c), mgr))
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

// TestFiberReviewHookControlFlow verifies termination, single continuation and original errors. TestFiberReviewHookControlFlow 验证终止、单次放行及原始错误传播。
func TestFiberReviewHookControlFlow(t *testing.T) {
	mgr := newFiberReviewManager(t, "fiber-hooks", nil)
	for _, check := range fiberReviewChecks {
		if check.name == "annotation" {
			continue
		}
		for _, mode := range []string{"abort", "abort_next", "exit_next", "next_twice"} {
			if check.name == "access" && mode != "abort" {
				continue
			}
			t.Run(check.name+"/"+mode, func(t *testing.T) {
				calls, failures, errorsHandled := 0, 0, 0
				wantErr := errors.New("downstream failed")
				app := gofiber.New(gofiber.Config{ErrorHandler: func(c *gofiber.Ctx, err error) error {
					errorsHandled++
					if !errors.Is(err, wantErr) {
						t.Errorf("error=%v, want original downstream error", err)
					}
					return c.SendStatus(http.StatusTeapot)
				}})
				app.Use(RegisterDTokenContextMiddleware(context.Background(), WithManager(mgr)))
				hook := func(_ context.Context, c *gofiber.Ctx, req *AuthHandleRequest) {
					if mode == "next_twice" {
						req.Next()
						req.Next()
						return
					}
					c.Status(http.StatusAccepted)
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
				app.Use(check.build(context.Background(),
					WithBeforeAuthHandler(hook),
					WithRouteAccessHandler(func(ctx context.Context, c *gofiber.Ctx, req *RouteAccessRequest) {
						hook(ctx, c, nil)
						req.SkipAuth()
					}),
					WithFailFunc(func(*gofiber.Ctx, error) { failures++ }),
				))
				app.Get("/", func(*gofiber.Ctx) error { calls++; return wantErr })
				raw := fiberReviewRequest(http.MethodGet, "/")
				app.Handler()(raw)
				wantStatus, wantCalls, wantErrors := http.StatusAccepted, 0, 0
				if mode == "next_twice" {
					wantStatus, wantCalls, wantErrors = http.StatusTeapot, 1, 1
				}
				if raw.Response.StatusCode() != wantStatus || calls != wantCalls || failures != 0 || errorsHandled != wantErrors {
					t.Fatalf("status=%d calls=%d failures=%d errors=%d", raw.Response.StatusCode(), calls, failures, errorsHandled)
				}
			})
		}
	}
}

// TestFiberReviewBodyTokenIsolation keeps query values out of body-only authentication. TestFiberReviewBodyTokenIsolation 验证仅允许请求体 Token 时不会混入查询参数。
func TestFiberReviewBodyTokenIsolation(t *testing.T) {
	mgr, err := NewBuilder().IsPrintBanner(false).IsLog(false).AutoRenew(false).AsyncEvent(false).
		IsReadHeader(false).IsReadCookie(false).IsReadQuery(false).IsReadBody(true).Build()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mgr.CloseManager)
	const token = "body-token"
	fiberReviewLogin(t, mgr, token)
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
				body, contentType := fields.Encode(), "application/x-www-form-urlencoded"
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
					body, contentType = b.String(), writer.FormDataContentType()
				}
				app := gofiber.New()
				app.Use(AuthMiddleware(context.Background(), WithManager(mgr)))
				called := false
				app.Post("/", func(c *gofiber.Ctx) error { called = true; return c.SendStatus(http.StatusNoContent) })
				raw := fiberReviewRequest(http.MethodPost, "/?"+url.Values{key: {queryToken}}.Encode())
				raw.Request.Header.SetContentType(contentType)
				raw.Request.SetBodyString(body)
				app.Handler()(raw)
				wantStatus := http.StatusUnauthorized
				if inBody {
					wantStatus = http.StatusNoContent
				}
				if raw.Response.StatusCode() != wantStatus || called != inBody {
					t.Fatalf("status=%d called=%v, want status=%d", raw.Response.StatusCode(), called, wantStatus)
				}
			})
		}
	}
}

// TestFiberReviewTokenSnapshots protects retained tokens when fasthttp reuses source buffers. TestFiberReviewTokenSnapshots 验证 fasthttp 复用来源缓冲区后已读取的 Token 仍保持不变。
func TestFiberReviewTokenSnapshots(t *testing.T) {
	const original, changed = "original-token", "replaced-token"
	for _, source := range []string{"header", "query", "cookie", "form"} {
		t.Run(source, func(t *testing.T) {
			c := newFiberReviewContext(t)
			rc := NewFiberContext(c)
			var set func(string)
			var read func() string
			switch source {
			case "header":
				set = func(value string) { c.Request().Header.Set("X-Token", value) }
				read = func() string { return rc.GetHeader("X-Token") }
			case "query":
				set = func(value string) { c.Context().QueryArgs().Set("token", value) }
				read = func() string { return rc.GetQuery("token") }
			case "cookie":
				set = func(value string) { c.Request().Header.SetCookie("token", value) }
				read = func() string { return rc.GetCookie("token") }
			case "form":
				c.Request().Header.SetMethod(http.MethodPost)
				c.Request().Header.SetContentType("application/x-www-form-urlencoded")
				set = func(value string) { c.Context().PostArgs().Set("token", value) }
				read = func() string { return rc.GetPostForm("token") }
			}
			set(original)
			saved := read()
			set(changed)
			if got := read(); got != changed || saved != original {
				t.Fatalf("saved=%q current=%q, want %q and %q", saved, got, original, changed)
			}
		})
	}
}

// TestFiberReviewRawBody preserves compressed bytes and isolates returned buffers. TestFiberReviewRawBody 保留压缩正文的原始字节并隔离返回缓冲区。
func TestFiberReviewRawBody(t *testing.T) {
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write([]byte("payload")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, encoding string
		body           []byte
	}{
		{"plain", "", []byte("payload")},
		{"gzip", "gzip", compressed.Bytes()},
		{"invalid_gzip", "gzip", []byte("not a gzip body")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newFiberReviewContext(t)
			c.Request().Header.Set("Content-Encoding", tc.encoding)
			c.Request().SetBody(tc.body)
			rc := NewFiberContext(c)
			body, err := rc.GetBody()
			if err != nil || !bytes.Equal(body, tc.body) {
				t.Fatalf("body=%q error=%v, want raw bytes", body, err)
			}
			body[0] ^= 0xff
			if !bytes.Equal(c.BodyRaw(), tc.body) {
				t.Fatal("returned body aliases the request buffer")
			}
			if again, err := rc.GetBody(); err != nil || !bytes.Equal(again, tc.body) {
				t.Fatalf("second read=%q,%v", again, err)
			}
		})
	}
}

// TestFiberReviewNilAnnotation rejects explicit nil and preserves custom failure responses. TestFiberReviewNilAnnotation 拒绝显式空注解并保留自定义失败响应。
func TestFiberReviewNilAnnotation(t *testing.T) {
	for _, custom := range []bool{false, true} {
		app := gofiber.New()
		calls, failures := 0, 0
		var failFunc func(*gofiber.Ctx, error)
		if custom {
			failFunc = func(c *gofiber.Ctx, err error) {
				failures++
				if !errors.Is(err, ErrInvalidParam) {
					t.Errorf("failure=%v, want ErrInvalidParam", err)
				}
				c.Status(http.StatusUnprocessableEntity)
			}
		}
		app.Get("/", GetHandler(context.Background(), nil, failFunc, nil), func(*gofiber.Ctx) error { calls++; return nil })
		raw := fiberReviewRequest(http.MethodGet, "/")
		app.Handler()(raw)
		wantStatus, wantFailures := http.StatusBadRequest, 0
		if custom {
			wantStatus, wantFailures = http.StatusUnprocessableEntity, 1
		}
		if calls != 0 || failures != wantFailures || raw.Response.StatusCode() != wantStatus {
			t.Fatalf("calls=%d failures=%d status=%d", calls, failures, raw.Response.StatusCode())
		}
	}
}
