package echo

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Zany2/dtoken-go/core/adapter"
	echo4 "github.com/labstack/echo/v4"
)

func TestEchoContextAdapterRequestAndResponse(t *testing.T) {
	engine := echo4.New()
	req := httptest.NewRequest(http.MethodPost, "/demo?foo=bar", strings.NewReader("hello"))
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
	if got := ctx.GetQuery("foo"); got != "bar" {
		t.Fatalf("GetQuery() = %q, want bar", got)
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
	if got := ctx.GetURL(); got != "/demo?foo=bar" {
		t.Fatalf("GetURL() = %q, want /demo?foo=bar", got)
	}
	if got := ctx.GetUserAgent(); got != "echo-test" {
		t.Fatalf("GetUserAgent() = %q, want echo-test", got)
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
	if got := rec.Header().Values("Set-Cookie"); len(got) == 0 {
		t.Fatal("SetCookieWithOptions() did not write Set-Cookie header")
	}
}

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
