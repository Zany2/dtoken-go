package kratos

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Zany2/dtoken-go/core/adapter"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

// TestKratosContextAdapterRequestAndResponse verifies Kratos request and response adaptation. TestKratosContextAdapterRequestAndResponse 验证 Kratos 请求与响应适配。
func TestKratosContextAdapterRequestAndResponse(t *testing.T) {
	server := khttp.NewServer()
	server.HandleFunc("/demo", func(w http.ResponseWriter, req *http.Request) {
		ctx := NewKratosContext(req.Context())
		ext := ctx.(adapter.RequestContextExt)
		if got := ctx.GetHeader("X-Token"); got != "token" {
			t.Errorf("GetHeader() = %q, want token", got)
		}
		if got := ctx.GetQuery("foo"); got != "bar" {
			t.Errorf("GetQuery() = %q, want bar", got)
		}
		if got := ctx.GetCookie("sid"); got != "cookie-token" {
			t.Errorf("GetCookie() = %q, want cookie-token", got)
		}
		if got := ctx.GetMethod(); got != http.MethodPost {
			t.Errorf("GetMethod() = %q, want POST", got)
		}
		if got := ctx.GetPath(); got != "/demo" {
			t.Errorf("GetPath() = %q, want /demo", got)
		}
		if got := ctx.GetURL(); got != "/demo?foo=bar" {
			t.Errorf("GetURL() = %q, want /demo?foo=bar", got)
		}
		if got := ctx.GetUserAgent(); got != "kratos-test" {
			t.Errorf("GetUserAgent() = %q, want kratos-test", got)
		}
		if got := ctx.GetClientIP(); got != "203.0.113.8" {
			t.Errorf("GetClientIP() = %q, want forwarded client IP", got)
		}

		body, err := ctx.GetBody()
		if err != nil || string(body) != "hello" {
			t.Errorf("GetBody() = %q, %v, want hello", body, err)
		}
		body, err = ctx.GetBody()
		if err != nil || string(body) != "hello" {
			t.Errorf("GetBody(second) = %q, %v, want hello", body, err)
		}

		ctx.Set("name", "dtoken")
		if got, ok := ctx.Get("name"); !ok || got != "dtoken" {
			t.Errorf("Get(name) = %v, %v, want dtoken,true", got, ok)
		}
		if got := ctx.GetString("name"); got != "dtoken" {
			t.Errorf("GetString(name) = %q, want dtoken", got)
		}
		if got := ctx.MustGet("name"); got != "dtoken" {
			t.Errorf("MustGet(name) = %v, want dtoken", got)
		}

		ctx.SetHeader("X-Result", "ok")
		ctx.SetCookieWithOptions(&adapter.CookieOptions{Name: "dt", Value: "v", Path: "/", SameSite: "Strict"})
		ctx.SetStatusCode(http.StatusAccepted)
		if _, err = ctx.Write([]byte("done")); err != nil {
			t.Errorf("Write() error = %v", err)
		}
		if ext.GetRawRequest() != req {
			t.Error("GetRawRequest() did not return the transport request")
		}
		if ext.GetRawResponseWriter() == nil {
			t.Error("GetRawResponseWriter() returned nil")
		}
		ctx.Abort()
		if !ctx.IsAborted() {
			t.Error("IsAborted() = false, want true")
		}
	})

	req := httptest.NewRequest(http.MethodPost, "/demo?foo=bar", strings.NewReader("hello"))
	req.Header.Set("X-Token", "token")
	req.Header.Set("X-Forwarded-For", "203.0.113.8, 10.0.0.1")
	req.Header.Set("User-Agent", "kratos-test")
	req.AddCookie(&http.Cookie{Name: "sid", Value: "cookie-token"})
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

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

// TestKratosContextMustGetPanicsWhenMissing verifies missing values panic. TestKratosContextMustGetPanicsWhenMissing 验证缺失值会触发 panic。
func TestKratosContextMustGetPanicsWhenMissing(t *testing.T) {
	ctx := NewKratosContext(nil)
	defer func() {
		if recover() == nil {
			t.Fatal("MustGet(missing) should panic")
		}
	}()
	ctx.MustGet("missing")
}

// TestKratosContextJSON verifies JSON response encoding and status. TestKratosContextJSON 验证 JSON 响应编码和状态码。
func TestKratosContextJSON(t *testing.T) {
	server := khttp.NewServer()
	server.HandleFunc("/json", func(w http.ResponseWriter, req *http.Request) {
		ctx := NewKratosContext(req.Context())
		if err := ctx.(adapter.RequestContextExt).JSON(http.StatusCreated, map[string]string{"ok": "true"}); err != nil {
			t.Errorf("JSON() error = %v", err)
		}
	})
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/json", nil))
	if rec.Code != http.StatusCreated {
		t.Fatalf("JSON status = %d, want 201", rec.Code)
	}
	var payload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("JSON body decode error = %v", err)
	}
	if payload["ok"] != "true" {
		t.Fatalf("JSON payload = %v, want ok=true", payload)
	}
}
