package gin

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

	"github.com/Zany2/dtoken-go/com/storage/memory"
	corecontext "github.com/Zany2/dtoken-go/core/context"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/dtoken"
	gingonic "github.com/gin-gonic/gin"
)

// TestGinReplacedRequestTokenSources rejects stale credentials cached by native query and form readers. TestGinReplacedRequestTokenSources 验证请求替换后不会使用原生 Query/Form 缓存中的旧凭据。
func TestGinReplacedRequestTokenSources(t *testing.T) {
	for _, source := range []string{"query", "form", "multipart"} {
		t.Run(source, func(t *testing.T) {
			mgr, err := NewBuilder().IsPrintBanner(false).IsLog(false).AutoRenew(false).AsyncEvent(false).
				IsReadHeader(false).IsReadCookie(false).IsReadQuery(source == "query").IsReadBody(source != "query").Build()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(mgr.CloseManager)
			for _, id := range []string{"old", "current"} {
				if _, err := mgr.LoginWithOptions(context.Background(), manager.LoginOptions{LoginID: id, Token: id + "-token"}); err != nil {
					t.Fatal(err)
				}
				if err := mgr.AddPermissions(context.Background(), id, []string{"read"}); err != nil {
					t.Fatal(err)
				}
				if err := mgr.AddRoles(context.Background(), id, []string{"reader"}); err != nil {
					t.Fatal(err)
				}
			}
			key := mgr.GetConfig().TokenName
			newRequest := func(token string) *http.Request {
				values := url.Values{}
				if token != "" {
					values.Set(key, token)
				}
				if source == "query" {
					return httptest.NewRequest(http.MethodGet, "/?"+values.Encode(), nil)
				}
				if source == "form" {
					r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(values.Encode()))
					r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
					return r
				}
				var body bytes.Buffer
				writer := multipart.NewWriter(&body)
				if token != "" {
					if err := writer.WriteField(key, token); err != nil {
						t.Fatal(err)
					}
				}
				if err := writer.Close(); err != nil {
					t.Fatal(err)
				}
				r := httptest.NewRequest(http.MethodPost, "/", &body)
				r.Header.Set("Content-Type", writer.FormDataContentType())
				return r
			}
			for _, check := range ginReviewChecks {
				for _, token := range []string{"current-token", "", "invalid-token"} {
					t.Run(check.name+"/"+token, func(t *testing.T) {
						router := gingonic.New()
						router.Use(RegisterDTokenContextMiddleware(context.Background(), WithManager(mgr)))
						router.Use(func(c *gingonic.Context) {
							cached := c.Query(key)
							if source != "query" {
								cached = c.PostForm(key)
							}
							if cached != "old-token" {
								t.Fatalf("native cache=%q", cached)
							}
							original := c.Request
							c.Request = newRequest(token)
							defer func() {
								for _, r := range []*http.Request{original, c.Request} {
									if r.MultipartForm != nil {
										_ = r.MultipartForm.RemoveAll()
									}
								}
							}()
							c.Next()
						})
						router.Use(check.build(context.Background()))
						var loginID string
						router.Any("/", func(c *gingonic.Context) {
							var err error
							loginID, err = GetLoginIDByContext(c)
							if err != nil {
								t.Error(err)
							}
							c.Status(http.StatusNoContent)
						})
						rec := httptest.NewRecorder()
						router.ServeHTTP(rec, newRequest("old-token"))
						if token == "current-token" {
							if loginID != "current" || rec.Code != http.StatusNoContent {
								t.Fatalf("loginID=%q status=%d", loginID, rec.Code)
							}
						} else if loginID != "" || rec.Code < http.StatusBadRequest {
							t.Fatalf("stale credentials passed: loginID=%q status=%d", loginID, rec.Code)
						}
					})
				}
			}
		})
	}
}

// TestGinRepeatedRegistration preserves request ownership unless a later registration explicitly overrides it. TestGinRepeatedRegistration 验证重复注册保留请求归属，除非后续注册显式覆盖。
func TestGinRepeatedRegistration(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	global := newGinReviewManager(t, "", nil)
	local := newGinReviewManager(t, "gin-registration", nil)
	dtoken.SetManager(global)
	for _, tc := range []struct {
		name string
		opts []AuthOption
		want *manager.Manager
	}{
		{"inherit", nil, local},
		{"auth_type", []AuthOption{WithAuthType(global.GetConfig().AuthType)}, global},
		{"explicit", []AuthOption{WithManager(local), WithAuthType("missing")}, local},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router := gingonic.New()
			router.Use(RegisterDTokenContextMiddleware(context.Background(), WithManager(local)))
			router.Use(RegisterDTokenContextMiddleware(context.Background(), tc.opts...))
			called := false
			router.GET("/", func(c *gingonic.Context) {
				called = true
				if got, err := GetManagerByContext(c); err != nil || got != tc.want {
					t.Errorf("manager=%p error=%v want=%p", got, err, tc.want)
				}
				c.Status(http.StatusNoContent)
			})
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
			if !called || rec.Code != http.StatusNoContent {
				t.Fatalf("called=%v status=%d", called, rec.Code)
			}
		})
	}
}

// TestGinInvalidManagerScopeCannotUseGlobal rejects valid global credentials when the request scope is invalid. TestGinInvalidManagerScopeCannotUseGlobal 验证请求作用域无效时不能借用全局有效凭据通过鉴权。
func TestGinInvalidManagerScopeCannotUseGlobal(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	global := newGinReviewManager(t, "", nil)
	dtoken.SetManager(global)
	token, err := global.Login(context.Background(), "reader")
	if err != nil {
		t.Fatal(err)
	}
	if err := global.AddPermissions(context.Background(), "reader", []string{"read"}); err != nil {
		t.Fatal(err)
	}
	if err := global.AddRoles(context.Background(), "reader", []string{"reader"}); err != nil {
		t.Fatal(err)
	}
	for _, check := range ginReviewChecks {
		t.Run(check.name, func(t *testing.T) {
			c := newGinTestContext(http.MethodGet, "/", global.GetConfig().TokenName, token)
			c.Set(DTokenCtxKey, corecontext.NewContext(NewGinContext(c), nil))
			var failure error
			check.build(context.Background(), WithFailFunc(func(_ *gingonic.Context, err error) { failure = err }))(c)
			if !c.IsAborted() || !errors.Is(failure, ErrManagerNotFound) {
				t.Fatalf("aborted=%v error=%v", c.IsAborted(), failure)
			}
		})
	}
}

// TestGinHooksRefreshRequestContext verifies checks use the request replaced by a hook, including cancellation. TestGinHooksRefreshRequestContext 验证鉴权使用钩子替换后的请求上下文及取消信号。
func TestGinHooksRefreshRequestContext(t *testing.T) {
	for _, check := range ginReviewChecks {
		if check.name == "annotation" {
			continue
		}
		for _, canceled := range []bool{false, true} {
			t.Run(check.name+map[bool]string{false: "/active", true: "/canceled"}[canceled], func(t *testing.T) {
				storage := &ginReviewContextStorage{Storage: memory.NewStorage()}
				mgr := newGinReviewManager(t, "gin-hook-context", storage)
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
				type requestKey struct{}
				ctx, cancel := context.WithCancel(context.WithValue(context.Background(), requestKey{}, "hook"))
				t.Cleanup(cancel)
				if canceled {
					cancel()
				}
				reads, hooks, handled, failures := 0, 0, 0, 0
				storage.inspect = func(got context.Context) {
					reads++
					if got != ctx {
						t.Error("authentication used the context captured before the hook")
					}
				}
				replace := func(c *gingonic.Context) { hooks++; c.Request = c.Request.WithContext(ctx) }
				router := gingonic.New()
				router.Use(check.build(context.Background(), WithManager(mgr),
					WithBeforeAuthHandler(func(_ context.Context, c *gingonic.Context, _ *AuthHandleRequest) { replace(c) }),
					WithRouteAccessHandler(func(_ context.Context, c *gingonic.Context, _ *RouteAccessRequest) { replace(c) }),
					WithFailFunc(func(*gingonic.Context, error) { failures++ }),
				))
				router.GET("/", func(c *gingonic.Context) { handled++; c.Status(http.StatusNoContent) })
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.Header.Set(mgr.GetConfig().TokenName, token)
				router.ServeHTTP(httptest.NewRecorder(), req)
				if reads == 0 || hooks != 1 {
					t.Fatalf("reads=%d hooks=%d", reads, hooks)
				}
				if canceled && (handled != 0 || failures != 1) || !canceled && (handled != 1 || failures != 0) {
					t.Fatalf("canceled=%v handled=%d failures=%d", canceled, handled, failures)
				}
			})
		}
	}
}

// TestGinHookDecisionRunsOnce prevents repeated hooks from advancing or aborting the chain again. TestGinHookDecisionRunsOnce 验证钩子重复调用不会再次推进或中止处理链。
func TestGinHookDecisionRunsOnce(t *testing.T) {
	for _, check := range ginReviewChecks {
		if check.name == "annotation" || check.name == "access" {
			continue
		}
		for _, exitFirst := range []bool{false, true} {
			t.Run(check.name+map[bool]string{false: "/next", true: "/exit"}[exitFirst], func(t *testing.T) {
				router := gingonic.New()
				handled := 0
				router.Use(check.build(context.Background(), WithBeforeAuthHandler(func(_ context.Context, c *gingonic.Context, req *AuthHandleRequest) {
					if exitFirst {
						req.Exit()
						req.Next()
						req.Exit()
					} else {
						req.Next()
						req.Next()
						req.Exit()
					}
					if c.IsAborted() != exitFirst {
						t.Error("later decision overrode the first one")
					}
				})))
				router.GET("/", func(*gingonic.Context) { handled++ })
				router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
				want := 1
				if exitFirst {
					want = 0
				}
				if handled != want {
					t.Fatalf("handled=%d want=%d", handled, want)
				}
			})
		}
	}
}
