package chi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Zany2/dtoken-go/core/adapter"
)

func TestChiContextAdapterRequestAndResponse(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/demo?foo=bar", strings.NewReader("hello"))
	req.Header.Set("X-Token", "token")
	req.Header.Set("X-Forwarded-For", "203.0.113.1, 10.0.0.1")
	req.Header.Set("User-Agent", "chi-test")
	req.AddCookie(&http.Cookie{Name: "sid", Value: "cookie-token"})
	rec := httptest.NewRecorder()

	ctx := NewChiContext(rec, req)
	ext, ok := ctx.(adapter.RequestContextExt)
	if !ok {
		t.Fatal("NewChiContext() should implement adapter.RequestContextExt")
	}

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
	if got := ctx.GetClientIP(); got != "203.0.113.1" {
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
	if got := ctx.GetUserAgent(); got != "chi-test" {
		t.Fatalf("GetUserAgent() = %q, want chi-test", got)
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

	jsonRec := httptest.NewRecorder()
	jsonCtx := NewChiContext(jsonRec, req).(adapter.RequestContextExt)
	if err = jsonCtx.JSON(http.StatusCreated, map[string]string{"ok": "true"}); err != nil {
		t.Fatalf("JSON() error = %v", err)
	}
	if jsonRec.Code != http.StatusCreated {
		t.Fatalf("JSON status = %d, want 201", jsonRec.Code)
	}
	var payload map[string]string
	if err = json.Unmarshal(jsonRec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("JSON body decode error = %v", err)
	}
	if payload["ok"] != "true" {
		t.Fatalf("JSON payload = %v, want ok=true", payload)
	}
	if ext.GetRawRequest() != req {
		t.Fatal("GetRawRequest() did not return original request")
	}
}

func TestChiContextMustGetPanicsWhenMissing(t *testing.T) {
	ctx := NewChiContext(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	defer func() {
		if recover() == nil {
			t.Fatal("MustGet(missing) should panic")
		}
	}()
	ctx.MustGet("missing")
}

func TestChiContextFallsBackToRemoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(context.Background())
	req.RemoteAddr = "198.51.100.7:12345"
	ctx := NewChiContext(httptest.NewRecorder(), req)
	if got := ctx.GetClientIP(); got != "198.51.100.7" {
		t.Fatalf("GetClientIP() = %q, want remote host", got)
	}
}
