package kratos

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"mime/multipart"
	"net"
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
	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// kratosReviewHeader implements transport headers without a network server. kratosReviewHeader 在不启动网络服务的情况下实现传输头。
type kratosReviewHeader http.Header

func (h kratosReviewHeader) Get(key string) string      { return http.Header(h).Get(key) }
func (h kratosReviewHeader) Set(key, value string)      { http.Header(h).Set(key, value) }
func (h kratosReviewHeader) Add(key, value string)      { http.Header(h).Add(key, value) }
func (h kratosReviewHeader) Values(key string) []string { return http.Header(h).Values(key) }
func (h kratosReviewHeader) Keys() []string {
	keys := make([]string, 0, len(h))
	for key := range h {
		keys = append(keys, key)
	}
	return keys
}

// kratosReviewTransport exercises the public HTTP and RPC transport contracts. kratosReviewTransport 验证公开的 HTTP 与 RPC 传输契约。
type kratosReviewTransport struct {
	kind           transport.Kind
	operation      string
	request        *http.Request
	writer         http.ResponseWriter
	headers, reply kratosReviewHeader
}

func (tr *kratosReviewTransport) Kind() transport.Kind            { return tr.kind }
func (tr *kratosReviewTransport) Endpoint() string                { return "" }
func (tr *kratosReviewTransport) Operation() string               { return tr.operation }
func (tr *kratosReviewTransport) RequestHeader() transport.Header { return tr.headers }
func (tr *kratosReviewTransport) ReplyHeader() transport.Header   { return tr.reply }
func (tr *kratosReviewTransport) Request() *http.Request          { return tr.request }
func (tr *kratosReviewTransport) Response() http.ResponseWriter   { return tr.writer }
func (tr *kratosReviewTransport) PathTemplate() string            { return tr.operation }

var _ khttp.ResponseTransporter = (*kratosReviewTransport)(nil)

// newKratosReviewContext creates a request carrying a framework transport. newKratosReviewContext 创建携带框架传输对象的请求上下文。
func newKratosReviewContext(kind transport.Kind, tokenName, token string) (context.Context, *kratosReviewTransport) {
	tr := &kratosReviewTransport{kind: kind, operation: "/review", headers: make(kratosReviewHeader), reply: make(kratosReviewHeader)}
	tr.headers.Set(tokenName, token)
	if kind == transport.KindHTTP {
		tr.request = httptest.NewRequest(http.MethodGet, "/review", nil)
		tr.request.Header = http.Header(tr.headers)
		tr.writer = httptest.NewRecorder()
		tr.reply = kratosReviewHeader(tr.writer.Header())
	}
	return transport.NewServerContext(context.Background(), tr), tr
}

// newKratosReviewManager creates an isolated manager without asynchronous work. newKratosReviewManager 创建无异步任务的独立 Manager。
func newKratosReviewManager(t *testing.T, authType string, storage adapter.Storage) *manager.Manager {
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

// kratosReviewStorage observes execution contexts at the storage boundary. kratosReviewStorage 在存储边界观察执行上下文。
type kratosReviewStorage struct {
	adapter.Storage
	inspect func(context.Context)
}

func (s *kratosReviewStorage) Get(ctx context.Context, key string) (any, error) {
	if s.inspect != nil {
		s.inspect(ctx)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.Storage.Get(ctx, key)
}

// kratosReviewChecks covers every authentication entry. kratosReviewChecks 覆盖所有鉴权入口。
var kratosReviewChecks = []struct {
	name  string
	build func(...AuthOption) middleware.Middleware
}{
	{"login", AuthMiddleware},
	{"access", AccessMiddleware},
	{"permission", func(opts ...AuthOption) middleware.Middleware { return PermissionMiddleware([]string{"read"}, opts...) }},
	{"path", func(opts ...AuthOption) middleware.Middleware {
		return PermissionPathMiddleware([]string{"read"}, opts...)
	}},
	{"role", func(opts ...AuthOption) middleware.Middleware { return RoleMiddleware([]string{"reader"}, opts...) }},
	{"annotation", func(opts ...AuthOption) middleware.Middleware {
		o := defaultAuthOptions()
		for _, opt := range opts {
			opt(o)
		}
		return GetHandler(o.FailFunc, &Annotation{AuthType: o.AuthType, CheckPermission: []string{"read"}})
	}},
}

// TestKratosReviewRequestContext verifies both transports inherit the registered manager and preserve request context. TestKratosReviewRequestContext 验证两种传输继承请求 Manager 并保留请求上下文。
func TestKratosReviewRequestContext(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	for _, kind := range []transport.Kind{transport.KindHTTP, transport.KindGRPC} {
		for _, check := range kratosReviewChecks {
			t.Run(string(kind)+"/"+check.name, func(t *testing.T) {
				storage := &kratosReviewStorage{Storage: memory.NewStorage()}
				mgr := newKratosReviewManager(t, "kratos-request", storage)
				token, err := mgr.Login(context.Background(), "reader")
				if err != nil {
					t.Fatal(err)
				}
				if err := mgr.AddPermissions(context.Background(), "reader", []string{"read", "/review"}); err != nil {
					t.Fatal(err)
				}
				if err := mgr.AddRoles(context.Background(), "reader", []string{"reader"}); err != nil {
					t.Fatal(err)
				}
				for _, canceled := range []bool{false, true} {
					base, _ := newKratosReviewContext(kind, mgr.GetConfig().TokenName, token)
					type requestKey struct{}
					ctx, cancel := context.WithTimeout(context.WithValue(base, requestKey{}, "current"), time.Minute)
					t.Cleanup(cancel)
					if canceled {
						cancel()
					}
					reads, handled, failures, hooks := 0, 0, 0, 0
					inspect := func(got context.Context) {
						deadline, ok := got.Deadline()
						want, _ := ctx.Deadline()
						if got.Value(requestKey{}) != "current" || !ok || !deadline.Equal(want) || got.Err() != ctx.Err() {
							t.Error("request values, deadline or cancellation were lost")
						}
					}
					storage.inspect = func(got context.Context) { reads++; inspect(got) }
					handler := middleware.Chain(RegisterDTokenContextMiddleware(WithManager(mgr)), check.build(
						WithBeforeAuthHandler(func(got context.Context, _ any, _ *AuthHandleRequest) { hooks++; inspect(got) }),
						WithRouteAccessHandler(func(got context.Context, _ any, _ *RouteAccessRequest) { hooks++; inspect(got) }),
						WithFailFunc(func(got context.Context, err error) error { failures++; inspect(got); return err }),
					))(func(got context.Context, _ any) (any, error) {
						handled++
						inspect(got)
						if selected, err := GetManagerByCtx(got); err != nil || selected != mgr {
							t.Errorf("manager=%p,%v", selected, err)
						}
						if id, err := GetLoginIDByCtx(got); err != nil || id != "reader" {
							t.Errorf("login=%q,%v", id, err)
						}
						return "ok", nil
					})
					result, err := handler(ctx, nil)
					if reads == 0 || check.name != "annotation" && hooks != 1 {
						t.Fatalf("reads=%d hooks=%d", reads, hooks)
					}
					if canceled {
						if err == nil || result != nil || handled != 0 || failures != 1 {
							t.Fatalf("canceled request: %v,%v handled=%d failures=%d", result, err, handled, failures)
						}
					} else if err != nil || result != "ok" || handled != 1 || failures != 0 {
						t.Fatalf("valid request: %v,%v handled=%d failures=%d", result, err, handled, failures)
					}
				}
			})
		}
	}
}

// TestKratosReviewManagerSelection verifies selection priority, invalid caches and renewal isolation. TestKratosReviewManagerSelection 验证选择优先级、无效缓存及续期隔离。
func TestKratosReviewManagerSelection(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	global := newKratosReviewManager(t, "", nil)
	local := newKratosReviewManager(t, "kratos-local", nil)
	selected := newKratosReviewManager(t, "kratos-selected", nil)
	dtoken.SetManager(global)
	dtoken.SetManager(selected)
	for _, mgr := range []*manager.Manager{global, local} {
		if _, err := mgr.LoginWithOptions(context.Background(), manager.LoginOptions{LoginID: "reader", Token: "same-token"}); err != nil {
			t.Fatal(err)
		}
	}
	base, _ := newKratosReviewContext(transport.KindHTTP, local.GetConfig().TokenName, "same-token")
	dCtx, ctx := getDTokenContext(base, local)
	dCtx.GetRequestContext().Set("marker", "kept")
	for _, tc := range []struct {
		explicit *manager.Manager
		authType string
		want     *manager.Manager
	}{
		{nil, "", local}, {nil, "kratos-selected", selected}, {local, "missing", local},
	} {
		if got, err := resolveRequestManager(ctx, tc.explicit, tc.authType); err != nil || got != tc.want {
			t.Fatalf("manager=%p,%v want=%p", got, err, tc.want)
		}
	}
	other, _ := getDTokenContext(ctx, selected)
	if other.GetRequestContext().GetString("marker") != "kept" {
		t.Fatal("manager selection discarded request values")
	}
	if err := RenewTimeoutByCtx(ctx, time.Minute); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		mgr     *manager.Manager
		timeout int64
	}{{global, 600}, {local, 60}} {
		info, err := tc.mgr.GetTokenInfo(context.Background(), "same-token")
		if err != nil || info.Timeout != tc.timeout {
			t.Fatalf("info=%+v,%v want=%d", info, err, tc.timeout)
		}
	}
	var typedNil *DTokenContext
	if got, err := resolveRequestManager(context.WithValue(base, DTokenCtxKey, typedNil), nil, ""); err != nil || got != global {
		t.Fatalf("typed nil=%p,%v", got, err)
	}
	local.CloseManager()
	for _, mgr := range []*manager.Manager{nil, local} {
		invalid := context.WithValue(base, DTokenCtxKey, corecontext.NewContext(NewKratosContext(base), mgr))
		_, tokenErr := GetTokenValueByCtx(invalid)
		_, loginErr := GetLoginIDByCtx(invalid)
		_, infoErr := IntrospectTokenByCtx(invalid)
		for _, err := range []error{tokenErr, loginErr, infoErr, RenewTimeoutByCtx(invalid, time.Minute)} {
			if !errors.Is(err, ErrManagerNotFound) {
				t.Errorf("invalid cache error=%v", err)
			}
		}
	}
}

// TestKratosReviewHooks verifies one continuation, preserved results and aborted requests. TestKratosReviewHooks 验证单次放行、保留返回值及请求中止。
func TestKratosReviewHooks(t *testing.T) {
	mgr := newKratosReviewManager(t, "kratos-hooks", nil)
	wantErr := errors.New("downstream failed")
	for _, check := range kratosReviewChecks {
		if check.name == "access" || check.name == "annotation" {
			continue
		}
		for _, mode := range []string{"next_twice", "exit_next", "abort", "abort_next"} {
			t.Run(check.name+"/"+mode, func(t *testing.T) {
				base, _ := newKratosReviewContext(transport.KindHTTP, "token", "")
				dCtx, ctx := getDTokenContext(base, mgr)
				calls, failures := 0, 0
				h := check.build(WithBeforeAuthHandler(func(_ context.Context, _ any, req *AuthHandleRequest) {
					switch mode {
					case "next_twice":
						req.Next()
						req.Next()
						req.Exit()
					case "exit_next":
						req.Exit()
						req.Next()
					default:
						dCtx.GetRequestContext().Abort()
						if mode == "abort_next" {
							req.Next()
						}
					}
				}), WithFailFunc(func(_ context.Context, err error) error { failures++; return err }))(func(context.Context, any) (any, error) { calls++; return "result", wantErr })
				result, err := h(ctx, nil)
				if mode == "next_twice" {
					if calls != 1 || result != "result" || !errors.Is(err, wantErr) {
						t.Fatalf("next=%d %v,%v", calls, result, err)
					}
				} else if calls != 0 || result != nil || err != nil {
					t.Fatalf("stopped=%d %v,%v", calls, result, err)
				}
				if failures != 0 {
					t.Fatalf("failures=%d", failures)
				}
			})
		}
	}
	_, ctx := getDTokenContext(context.Background(), mgr)
	h := AccessMiddleware(WithRouteAccessHandler(func(ctx context.Context, _ any, req *RouteAccessRequest) {
		dCtx, _ := GetDTokenContext(ctx)
		dCtx.GetRequestContext().Abort()
		req.SkipAuth()
	}))(func(context.Context, any) (any, error) { t.Fatal("aborted access hook continued"); return nil, nil })
	if _, err := h(ctx, nil); err != nil {
		t.Fatal(err)
	}
}

// TestKratosReviewAbortedEntry verifies no DToken entry reopens an aborted chain. TestKratosReviewAbortedEntry 验证各 DToken 入口不会重新放行已中止请求。
func TestKratosReviewAbortedEntry(t *testing.T) {
	mgr := newKratosReviewManager(t, "kratos-abort", nil)
	dCtx, ctx := getDTokenContext(context.Background(), mgr)
	dCtx.GetRequestContext().Abort()
	entries := []middleware.Middleware{RegisterDTokenContextMiddleware(), GetHandler(nil), GetHandler(nil, nil), IgnoreMiddleware(nil), PermissionMiddleware(nil), RoleMiddleware(nil)}
	for _, check := range kratosReviewChecks {
		entries = append(entries, check.build())
	}
	for _, entry := range entries {
		if result, err := entry(func(context.Context, any) (any, error) { t.Fatal("aborted request continued"); return nil, nil })(ctx, nil); result != nil || err != nil {
			t.Fatalf("aborted=%v,%v", result, err)
		}
	}
}

// TestKratosReviewFailureAndAnnotations verifies rejection, default AND logic and immutable annotation configuration. TestKratosReviewFailureAndAnnotations 验证失败拒绝、默认 AND 及不修改共享注解配置。
func TestKratosReviewFailureAndAnnotations(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	mgr := newKratosReviewManager(t, "kratos-annotation", nil)
	for _, check := range kratosReviewChecks {
		for _, registered := range []bool{false, true} {
			ctx, _ := newKratosReviewContext(transport.KindGRPC, mgr.GetConfig().TokenName, "")
			if registered {
				_, ctx = getDTokenContext(ctx, mgr)
			}
			failures := 0
			h := check.build(WithFailFunc(func(ctx context.Context, err error) error {
				failures++
				if err == nil || registered && !isRequestAborted(ctx) {
					t.Error("failure callback saw an active request")
				}
				return err
			}))(func(context.Context, any) (any, error) { t.Fatal("failed request continued"); return nil, nil })
			if _, err := h(ctx, nil); err == nil || failures != 1 {
				t.Fatalf("failure=%v calls=%d", err, failures)
			}
		}
	}
	called := 0
	next := func(context.Context, any) (any, error) { called++; return "ok", nil }
	if _, err := GetHandler(nil, nil)(next)(context.Background(), nil); !errors.Is(err, ErrInvalidParam) || called != 0 {
		t.Fatalf("nil annotation=%v called=%d", err, called)
	}
	if _, err := GetHandler(nil)(next)(context.Background(), nil); err != nil || called != 1 {
		t.Fatalf("absent annotation=%v called=%d", err, called)
	}
	token, err := mgr.Login(context.Background(), "reader")
	if err != nil {
		t.Fatal(err)
	}
	if err := mgr.AddPermissions(context.Background(), "reader", []string{"read"}); err != nil {
		t.Fatal(err)
	}
	ann := &Annotation{CheckPermission: []string{"read", "write"}}
	h := GetHandler(nil, ann)(next)
	for i := 0; i < 2; i++ {
		base, _ := newKratosReviewContext(transport.KindGRPC, mgr.GetConfig().TokenName, token)
		_, ctx := getDTokenContext(base, mgr)
		if _, err := h(ctx, nil); !errors.Is(err, ErrPermissionDenied) {
			t.Fatalf("default AND=%v", err)
		}
		if ann.LogicType != "" || called != 1 {
			t.Fatalf("annotation mutated or handler ran: %+v called=%d", ann, called)
		}
	}
}

// TestKratosReviewMultipart verifies multipart and body-only token sources. TestKratosReviewMultipart 验证 multipart 与仅请求体 Token 来源。
func TestKratosReviewMultipart(t *testing.T) {
	mgr, err := NewBuilder().IsPrintBanner(false).IsLog(false).AutoRenew(false).AsyncEvent(false).
		IsReadHeader(false).IsReadCookie(false).IsReadQuery(false).IsReadBody(true).Build()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mgr.CloseManager)
	key := mgr.GetConfig().TokenName
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	if err := w.WriteField(key, "body-token"); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ contentType, body, want string }{
		{w.FormDataContentType(), body.String(), "body-token"},
		{"application/x-www-form-urlencoded", url.Values{key: {"form-token"}}.Encode(), "form-token"},
		{"application/x-www-form-urlencoded", "other=value", ""},
		{"application/json", `{"` + key + `":"json-token"}`, ""},
	} {
		ctx, tr := newKratosReviewContext(transport.KindHTTP, key, "header-token")
		tr.request = httptest.NewRequest(http.MethodPost, "/?"+url.Values{key: {"query-token"}}.Encode(), strings.NewReader(tc.body))
		tr.request.Header.Set("Content-Type", tc.contentType)
		dCtx, _ := getDTokenContext(ctx, mgr)
		if got := dCtx.GetTokenValue(); got != tc.want {
			t.Fatalf("%s token=%q want=%q", tc.contentType, got, tc.want)
		}
		if tr.request.MultipartForm != nil {
			_ = tr.request.MultipartForm.RemoveAll()
		}
	}
}

// TestKratosReviewHTTPTransport verifies wrapped HTTP transports, cookies and raw body behavior. TestKratosReviewHTTPTransport 验证包装后的 HTTP 传输、Cookie 与原始请求体行为。
func TestKratosReviewHTTPTransport(t *testing.T) {
	ctx, tr := newKratosReviewContext(transport.KindHTTP, "X-Token", "original")
	a := NewKratosContext(ctx).(adapter.RequestContextExt)
	if a.GetRawRequest() != tr.request || a.GetRawResponseWriter() != tr.writer {
		t.Fatal("public HTTP transport contract was ignored")
	}
	adapterCtx := a.(adapter.RequestContext)
	tr.request.Header = http.Header{"X-Token": {"current"}}
	if got := adapterCtx.GetHeader("X-Token"); got != "current" {
		t.Fatalf("stale header=%q", got)
	}
	const token = "custom+/%2B=token"
	adapterCtx.SetCookieWithOptions(&adapter.CookieOptions{Name: "strict", Value: token, Path: "/", MaxAge: 60, SameSite: "Strict", Secure: true, HttpOnly: true})
	adapterCtx.SetCookie("legacy", token, 60, "/", "", false, false)
	adapterCtx.SetCookieWithOptions(&adapter.CookieOptions{Name: "expired", Path: "/", MaxAge: -1, SameSite: "None", Secure: true})
	cookies := (&http.Response{Header: tr.writer.Header()}).Cookies()
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
	tr.request.AddCookie(cookies[0])
	if got := adapterCtx.GetCookie("strict"); got != token {
		t.Fatalf("cookie=%q", got)
	}
	tr.request.Body = io.NopCloser(strings.NewReader("raw-body"))
	tr.request.Header.Set("Content-Encoding", "gzip")
	for i := 0; i < 2; i++ {
		if body, err := adapterCtx.GetBody(); err != nil || string(body) != "raw-body" {
			t.Fatalf("body=%q,%v", body, err)
		}
	}
	wantErr := errors.New("stream failed")
	tr.request.Body = io.NopCloser(iotest.ErrReader(wantErr))
	if _, err := adapterCtx.GetBody(); !errors.Is(err, wantErr) {
		t.Fatalf("body error=%v", err)
	}
}

// TestKratosReviewPeerAndTLS verifies transport facts cannot be spoofed through forwarding headers. TestKratosReviewPeerAndTLS 验证转发头无法伪造传输连接信息。
func TestKratosReviewPeerAndTLS(t *testing.T) {
	ctx, tr := newKratosReviewContext(transport.KindHTTP, "X-Forwarded-For", "203.0.113.8")
	tr.request.RemoteAddr = "[2001:db8::8]:8080"
	tr.request.Header.Set("X-Real-IP", "203.0.113.9")
	tr.request.Header.Set("X-Forwarded-Proto", "https")
	tr.request.URL.Scheme = "https"
	a := NewKratosContext(ctx)
	if a.GetClientIP() != "2001:db8::8" || a.IsTLS() {
		t.Fatal("HTTP peer or TLS trusted forwarding data")
	}
	tr.request.TLS = &tls.ConnectionState{}
	if !a.IsTLS() {
		t.Fatal("HTTP TLS was missed")
	}
	rpc, _ := newKratosReviewContext(transport.KindGRPC, "X-Forwarded-For", "203.0.113.8")
	rpc = peer.NewContext(rpc, &peer.Peer{Addr: &net.TCPAddr{IP: net.ParseIP("192.0.2.8"), Port: 8080}})
	a = NewKratosContext(rpc)
	if a.GetClientIP() != "192.0.2.8" || a.IsTLS() || a.GetPath() != "/review" {
		t.Fatal("gRPC plaintext peer or operation was lost")
	}
	rpc = peer.NewContext(rpc, &peer.Peer{Addr: &net.TCPAddr{IP: net.ParseIP("192.0.2.8")}, AuthInfo: credentials.TLSInfo{State: tls.ConnectionState{HandshakeComplete: true}}})
	a = NewKratosContext(rpc)
	if !a.IsTLS() {
		t.Fatal("gRPC TLS was missed")
	}
	a.SetHeader("X-Result", "ok")
	if tr, _ := transport.FromServerContext(rpc); tr.ReplyHeader().Get("X-Result") != "ok" {
		t.Fatal("RPC reply metadata was lost")
	}
	if _, err := a.Write([]byte("body")); !errors.Is(err, http.ErrNotSupported) {
		t.Fatalf("RPC write=%v", err)
	}
	if err := a.(adapter.RequestContextExt).JSON(http.StatusOK, "body"); !errors.Is(err, http.ErrNotSupported) {
		t.Fatalf("RPC JSON=%v", err)
	}
	if empty := NewKratosContext(nil); empty.GetClientIP() != "" || empty.IsTLS() || empty.GetHeader("missing") != "" {
		t.Fatal("nil context did not remain empty")
	}
}

// TestKratosReviewErrorMapping verifies HTTP status, RPC status and original error identity. TestKratosReviewErrorMapping 验证 HTTP 状态、RPC 状态及原始错误身份。
func TestKratosReviewErrorMapping(t *testing.T) {
	for _, tc := range []struct {
		err      error
		httpCode int
		grpcCode codes.Code
	}{
		{ErrNotLogin, http.StatusUnauthorized, codes.Unauthenticated},
		{ErrTokenExpired, http.StatusUnauthorized, codes.Unauthenticated},
		{ErrPermissionDenied, http.StatusForbidden, codes.PermissionDenied},
		{ErrInvalidParam, http.StatusBadRequest, codes.InvalidArgument},
		{ErrManagerNotFound, http.StatusNotFound, codes.NotFound},
		{ErrStorageUnavailable, http.StatusInternalServerError, codes.Internal},
	} {
		err := dispatchFail(context.Background(), nil, tc.err)
		if !errors.Is(err, tc.err) || kerrors.Code(err) != tc.httpCode || status.Code(err) != tc.grpcCode {
			t.Fatalf("mapped error=%v HTTP=%d RPC=%v", err, kerrors.Code(err), status.Code(err))
		}
	}
}

// TestKratosReviewRPCOperationPermission verifies path checks reject a different RPC operation. TestKratosReviewRPCOperationPermission 验证路径权限检查拒绝未授权的 RPC 操作。
func TestKratosReviewRPCOperationPermission(t *testing.T) {
	mgr := newKratosReviewManager(t, "kratos-operation", nil)
	token, err := mgr.Login(context.Background(), "reader")
	if err != nil {
		t.Fatal(err)
	}
	if err := mgr.AddPermissions(context.Background(), "reader", []string{"/review"}); err != nil {
		t.Fatal(err)
	}
	for _, operation := range []string{"/review", "/blocked"} {
		ctx, tr := newKratosReviewContext(transport.KindGRPC, mgr.GetConfig().TokenName, token)
		tr.operation = operation
		handled := false
		h := PermissionPathMiddleware(nil, WithManager(mgr))(func(context.Context, any) (any, error) {
			handled = true
			return "ok", nil
		})
		result, err := h(ctx, nil)
		if operation == "/review" {
			if err != nil || !handled || result != "ok" {
				t.Fatalf("allowed operation=%v,%v handled=%v", result, err, handled)
			}
		} else if !errors.Is(err, ErrPermissionDenied) || handled || result != nil {
			t.Fatalf("blocked operation=%v,%v handled=%v", result, err, handled)
		}
	}
}
