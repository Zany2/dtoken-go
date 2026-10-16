package hertz

import (
	"net/http"
	"testing"

	"github.com/Zany2/dtoken-go/core/adapter"
	hertzapp "github.com/cloudwego/hertz/pkg/app"
)

// TestHertzContextAdapterRequestAndResponse verifies Hertz request and response adaptation. TestHertzContextAdapterRequestAndResponse 验证 Hertz 请求与响应适配。
func TestHertzContextAdapterRequestAndResponse(t *testing.T) {
	hertzCtx := hertzapp.NewContext(0)
	hertzCtx.Request.SetMethod(http.MethodPost)
	hertzCtx.Request.SetRequestURI("/demo?foo=bar")
	hertzCtx.Request.Header.Set("X-Token", "token")
	hertzCtx.Request.Header.Set("User-Agent", "hertz-test")
	hertzCtx.Request.Header.SetCookie("sid", "cookie-token")
	hertzCtx.Request.SetBodyString("hello")
	ctx := NewHertzContext(hertzCtx)

	if got := ctx.GetHeader("X-Token"); got != "token" {
		t.Fatalf("GetHeader() = %q, want token", got)
	}
	if got := ctx.GetQuery("foo"); got != "bar" {
		t.Fatalf("GetQuery() = %q, want bar", got)
	}
	if got := ctx.GetCookie("sid"); got != "cookie-token" {
		t.Fatalf("GetCookie() = %q, want cookie-token", got)
	}
	if got := ctx.GetMethod(); got != http.MethodPost {
		t.Fatalf("GetMethod() = %q, want POST", got)
	}
	if got := ctx.GetPath(); got != "/demo" {
		t.Fatalf("GetPath() = %q, want /demo", got)
	}
	if got := ctx.GetURL(); got != "/demo?foo=bar" {
		t.Fatalf("GetURL() = %q, want /demo?foo=bar", got)
	}
	if got := ctx.GetUserAgent(); got != "hertz-test" {
		t.Fatalf("GetUserAgent() = %q, want hertz-test", got)
	}
	body, err := ctx.GetBody()
	if err != nil || string(body) != "hello" {
		t.Fatalf("GetBody() = %q, %v, want hello", body, err)
	}

	ctx.Set("name", "dtoken")
	if got, ok := ctx.Get("name"); !ok || got != "dtoken" {
		t.Fatalf("Get(name) = %v, %v, want dtoken,true", got, ok)
	}
	if got := ctx.GetString("name"); got != "dtoken" {
		t.Fatalf("GetString(name) = %q, want dtoken", got)
	}
	if got := ctx.MustGet("name"); got != "dtoken" {
		t.Fatalf("MustGet(name) = %v, want dtoken", got)
	}

	ctx.SetHeader("X-Result", "ok")
	ctx.SetCookieWithOptions(&adapter.CookieOptions{Name: "dt", Value: "v", Path: "/", SameSite: "Strict"})
	ctx.SetStatusCode(http.StatusAccepted)
	if _, err = ctx.Write([]byte("done")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if got := string(hertzCtx.Response.Header.Peek("X-Result")); got != "ok" {
		t.Fatalf("response header = %q, want ok", got)
	}
	if got := string(hertzCtx.Response.BodyBytes()); got != "done" {
		t.Fatalf("response body = %q, want done", got)
	}
	if hertzCtx.Response.StatusCode() != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", hertzCtx.Response.StatusCode())
	}
	ctx.Abort()
	if !ctx.IsAborted() {
		t.Fatal("IsAborted() = false, want true")
	}
}

// TestHertzContextMustGetPanicsWhenMissing verifies missing values panic. TestHertzContextMustGetPanicsWhenMissing 验证缺失值会触发 panic。
func TestHertzContextMustGetPanicsWhenMissing(t *testing.T) {
	ctx := NewHertzContext(hertzapp.NewContext(0))
	defer func() {
		if recover() == nil {
			t.Fatal("MustGet(missing) should panic")
		}
	}()
	ctx.MustGet("missing")
}
