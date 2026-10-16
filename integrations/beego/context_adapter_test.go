package beego

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Zany2/dtoken-go/core/adapter"
	beegocontext "github.com/beego/beego/v2/server/web/context"
)

// TestBeegoContextAdapterRequestAndResponse verifies Beego request and response adaptation. TestBeegoContextAdapterRequestAndResponse 验证 Beego 请求与响应适配。
func TestBeegoContextAdapterRequestAndResponse(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "http://example.com/demo?foo=bar", strings.NewReader("hello"))
	req.Header.Set("X-Token", "token")
	req.Header.Set("User-Agent", "beego-test")
	req.AddCookie(&http.Cookie{Name: "sid", Value: "cookie-token"})
	rec := httptest.NewRecorder()
	beegoCtx := beegocontext.NewContext()
	beegoCtx.Reset(rec, req)
	ctx := NewBeegoContext(beegoCtx)

	if got := ctx.GetHeader("X-Token"); got != "token" {
		t.Fatalf("GetHeader() = %q, want token", got)
	}
	if got := ctx.GetQuery("foo"); got != "bar" {
		t.Fatalf("GetQuery() = %q, want bar", got)
	}
	if got := ctx.GetCookie("sid"); got != "cookie-token" {
		t.Fatalf("GetCookie() = %q, want cookie-token", got)
	}
	if got := ctx.GetCookie("missing"); got != "" {
		t.Fatalf("GetCookie(missing) = %q, want empty string", got)
	}
	if got := ctx.GetMethod(); got != http.MethodPost {
		t.Fatalf("GetMethod() = %q, want POST", got)
	}
	if got := ctx.GetPath(); got != "/demo" {
		t.Fatalf("GetPath() = %q, want /demo", got)
	}
	if got := ctx.GetURL(); got != "http://example.com/demo?foo=bar" {
		t.Fatalf("GetURL() = %q, want full URL", got)
	}
	if got := ctx.GetUserAgent(); got != "beego-test" {
		t.Fatalf("GetUserAgent() = %q, want beego-test", got)
	}

	body, err := ctx.GetBody()
	if err != nil || string(body) != "hello" {
		t.Fatalf("GetBody() = %q, %v, want hello", body, err)
	}
	body, err = ctx.GetBody()
	if err != nil || string(body) != "hello" {
		t.Fatalf("GetBody(second) = %q, %v, want hello", body, err)
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
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", rec.Code)
	}
	if got := rec.Header().Get("X-Result"); got != "ok" {
		t.Fatalf("response header = %q, want ok", got)
	}
	if got := rec.Body.String(); got != "done" {
		t.Fatalf("response body = %q, want done", got)
	}
	if len(rec.Header().Values("Set-Cookie")) == 0 {
		t.Fatal("SetCookieWithOptions() did not write Set-Cookie header")
	}
}

// TestBeegoContextMustGetPanicsWhenMissing verifies missing values panic. TestBeegoContextMustGetPanicsWhenMissing 验证缺失值会触发 panic。
func TestBeegoContextMustGetPanicsWhenMissing(t *testing.T) {
	beegoCtx := beegocontext.NewContext()
	beegoCtx.Reset(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	ctx := NewBeegoContext(beegoCtx)

	defer func() {
		if recover() == nil {
			t.Fatal("MustGet(missing) should panic")
		}
	}()
	ctx.MustGet("missing")
}
