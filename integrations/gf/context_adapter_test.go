package gf

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gogf/gf/v2/net/ghttp"
)

// TestGFContextValuesAndCookies verifies value and cookie semantics. TestGFContextValuesAndCookies 验证上下文值和 Cookie 语义。
func TestGFContextValuesAndCookies(t *testing.T) {
	req := &ghttp.Request{Request: httptest.NewRequest(http.MethodGet, "http://example.com/demo?name=request-param", nil)}
	req.AddCookie(&http.Cookie{Name: "sid", Value: "cookie-token"})
	req.Cookie = ghttp.GetCookie(req)
	ctx := NewGFContext(req)

	if value, ok := ctx.Get("missing"); ok || value != nil {
		t.Fatalf("Get(missing) = %v, %v, want nil,false", value, ok)
	}

	ctx.Set("name", "dtoken")
	if value, ok := ctx.Get("name"); !ok || value != "dtoken" {
		t.Fatalf("Get(name) = %v, %v, want dtoken,true", value, ok)
	}
	if value := ctx.GetString("name"); value != "dtoken" {
		t.Fatalf("GetString(name) = %q, want dtoken", value)
	}
	if value := ctx.MustGet("name"); value != "dtoken" {
		t.Fatalf("MustGet(name) = %v, want dtoken", value)
	}

	if value := ctx.GetCookie("sid"); value != "cookie-token" {
		t.Fatalf("GetCookie(sid) = %q, want cookie-token", value)
	}
	if value := ctx.GetCookie("missing"); value != "" {
		t.Fatalf("GetCookie(missing) = %q, want empty string", value)
	}
}

// TestGFContextMustGetPanicsWhenMissing verifies missing values panic. TestGFContextMustGetPanicsWhenMissing 验证缺失值会触发 panic。
func TestGFContextMustGetPanicsWhenMissing(t *testing.T) {
	req := &ghttp.Request{Request: httptest.NewRequest(http.MethodGet, "/", nil)}
	req.Cookie = ghttp.GetCookie(req)
	ctx := NewGFContext(req)

	defer func() {
		if recover() == nil {
			t.Fatal("MustGet(missing) should panic")
		}
	}()
	ctx.MustGet("missing")
}
