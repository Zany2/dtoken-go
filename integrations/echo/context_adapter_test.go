package echo

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Zany2/dtoken-go/core/adapter"
	echo4 "github.com/labstack/echo/v4"
)

// TestEchoContextAdapterRequestAndResponse verifies request and response adaptation. TestEchoContextAdapterRequestAndResponse 验证请求与响应适配。
func TestEchoContextAdapterRequestAndResponse(t *testing.T) {
	engine := echo4.New()
	req := httptest.NewRequest(http.MethodPost, "/demo?foo=bar&foo=baz", strings.NewReader("hello"))
	req.Header.Set("X-Token", "token")
	req.Header.Set("X-Forwarded-For", "203.0.113.2")
	req.Header.Set("User-Agent", "echo-test")
	req.AddCookie(&http.Cookie{Name: "sid", Value: "cookie-token"})
	rec := httptest.NewRecorder()
	echoCtx := engine.NewContext(req, rec)

	ctx := NewEchoContext(echoCtx)
	if got := ctx.GetHeader("X-Token"); got != "token" {
		t.Fatalf("GetHeader() = %q, want token", got)
	}
	if got := ctx.GetHeaders()["X-Token"]; len(got) != 1 || got[0] != "token" {
		t.Fatalf("GetHeaders()[X-Token] = %v, want [token]", got)
	}
	if got := ctx.GetQuery("foo"); got != "bar" {
		t.Fatalf("GetQuery() = %q, want bar", got)
	}
	query := ctx.GetQueryAll()["foo"]
	if len(query) != 2 || query[0] != "bar" || query[1] != "baz" {
		t.Fatalf("GetQueryAll()[foo] = %v, want [bar baz]", query)
	}
	if got := ctx.GetCookie("sid"); got != "cookie-token" {
		t.Fatalf("GetCookie() = %q, want cookie-token", got)
	}
	body, err := ctx.GetBody()
	if err != nil || string(body) != "hello" {
		t.Fatalf("GetBody() = %q, %v, want hello", body, err)
	}
	body, err = ctx.GetBody()
	if err != nil || string(body) != "hello" {
		t.Fatalf("GetBody(second) = %q, %v, want hello", body, err)
	}
	if got := ctx.GetClientIP(); got != "203.0.113.2" {
		t.Fatalf("GetClientIP() = %q, want forwarded client IP", got)
	}
	if got := ctx.GetMethod(); got != http.MethodPost {
		t.Fatalf("GetMethod() = %q, want POST", got)
	}
	if got := ctx.GetPath(); got != "/demo" {
		t.Fatalf("GetPath() = %q, want /demo", got)
	}
	if got := ctx.GetURL(); got != "/demo?foo=bar&foo=baz" {
		t.Fatalf("GetURL() = %q, want /demo?foo=bar&foo=baz", got)
	}
	if got := ctx.GetUserAgent(); got != "echo-test" {
		t.Fatalf("GetUserAgent() = %q, want echo-test", got)
	}
	if got := ctx.GetPostForm("missing"); got != "" {
		t.Fatalf("GetPostForm(missing) = %q, want empty", got)
	}
	if ctx.IsTLS() {
		t.Fatal("IsTLS() = true for HTTP request")
	}

	ctx.Set("name", "dtoken")
	if got := ctx.GetString("name"); got != "dtoken" {
		t.Fatalf("GetString() = %q, want dtoken", got)
	}
	if got := ctx.MustGet("name"); got != "dtoken" {
		t.Fatalf("MustGet() = %v, want dtoken", got)
	}
	if got, ok := ctx.Get("name"); !ok || got != "dtoken" {
		t.Fatalf("Get() = %v, %v, want dtoken,true", got, ok)
	}
	ctx.Abort()
	if !ctx.IsAborted() {
		t.Fatal("IsAborted() = false, want true")
	}

	ctx.SetHeader("X-Result", "ok")
	ctx.SetCookie("legacy", "value", 60, "/", "", false, true)
	ctx.SetCookieWithOptions(&adapter.CookieOptions{Name: "dt", Value: "v", Path: "/", SameSite: "None"})
	ctx.SetStatusCode(http.StatusAccepted)
	if _, err = ctx.Write([]byte("done")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", rec.Code)
	}
	if got := rec.Header().Get("X-Result"); got != "ok" {
		t.Fatalf("response header = %q, want ok", got)
	}
	if got := rec.Body.String(); got != "done" {
		t.Fatalf("response body = %q, want done", got)
	}
	if got := rec.Header().Values("Set-Cookie"); len(got) != 2 {
		t.Fatalf("Set-Cookie count = %d, want 2", len(got))
	}
}

// TestEchoContextMustGetPanicsWhenMissing verifies missing values trigger a panic. TestEchoContextMustGetPanicsWhenMissing 验证缺失值会触发 panic。
func TestEchoContextMustGetPanicsWhenMissing(t *testing.T) {
	engine := echo4.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := NewEchoContext(engine.NewContext(req, httptest.NewRecorder()))
	defer func() {
		if recover() == nil {
			t.Fatal("MustGet(missing) should panic")
		}
	}()
	ctx.MustGet("missing")
}
