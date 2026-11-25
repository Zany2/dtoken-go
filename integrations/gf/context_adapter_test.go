package gf

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	derror "github.com/Zany2/dtoken-go/errors"
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
	if value, err := ctx.GetRequired("name"); err != nil || value != "dtoken" {
		t.Fatalf("GetRequired(name) = %v, want dtoken", value)
	}

	if value := ctx.GetCookie("sid"); value != "cookie-token" {
		t.Fatalf("GetCookie(sid) = %q, want cookie-token", value)
	}
	if value := ctx.GetCookie("missing"); value != "" {
		t.Fatalf("GetCookie(missing) = %q, want empty string", value)
	}
}

// TestGFContextGetRequiredReturnsErrorWhenMissing verifies missing values return an error. TestGFContextGetRequiredReturnsErrorWhenMissing 验证缺失值返回错误。
func TestGFContextGetRequiredReturnsErrorWhenMissing(t *testing.T) {
	req := &ghttp.Request{Request: httptest.NewRequest(http.MethodGet, "/", nil)}
	req.Cookie = ghttp.GetCookie(req)
	ctx := NewGFContext(req)

	if value, err := ctx.GetRequired("missing"); value != nil || !errors.Is(err, derror.ErrKeyNotFound) {
		t.Fatalf("GetRequired(missing) = (%v, %v), want ErrKeyNotFound", value, err)
	}
}

// TestGFBodyPanicBecomesReadError verifies third-party body panics remain stable read errors. TestGFBodyPanicBecomesReadError 验证第三方请求体异常转换为可重复读取的错误。
func TestGFBodyPanicBecomesReadError(t *testing.T) {
	cause := errors.New("body reader failed")
	for _, failure := range []any{cause, "body reader panic"} {
		req := &ghttp.Request{Request: httptest.NewRequest(http.MethodPost, "/", nil)}
		reader := &panickingBodyReader{failure: failure}
		req.Body = reader
		ctx := NewGFContext(req)
		body, first := ctx.GetBody()
		if body != nil || first == nil {
			t.Fatalf("GetBody() = (%v, %v), want read error", body, first)
		}
		if failure == cause && !errors.Is(first, cause) {
			t.Fatalf("GetBody() lost error cause: %v", first)
		}
		if body, err := ctx.GetBody(); body != nil || err != first || reader.calls != 1 {
			t.Fatalf("repeated GetBody() = (%v, %v), calls=%d", body, err, reader.calls)
		}
	}
}

type panickingBodyReader struct {
	failure any
	calls   int
}

func (r *panickingBodyReader) Read([]byte) (int, error) {
	r.calls++
	panic(r.failure)
}

func (r *panickingBodyReader) Close() error { return nil }
