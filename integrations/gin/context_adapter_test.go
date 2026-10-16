package gin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Zany2/dtoken-go/core/adapter"
	gingonic "github.com/gin-gonic/gin"
)

// TestGinContextCookieOptionsAreAppliedBeforeWrite verifies SameSite reaches emitted cookies. TestGinContextCookieOptionsAreAppliedBeforeWrite 验证 SameSite 在写入前生效。
func TestGinContextCookieOptionsAreAppliedBeforeWrite(t *testing.T) {
	gingonic.SetMode(gingonic.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gingonic.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := NewGinContext(c)

	ctx.SetCookieWithOptions(&adapter.CookieOptions{
		Name:     "strict-cookie",
		Value:    "value",
		Path:     "/",
		SameSite: "Strict",
	})
	ctx.SetCookie("legacy-cookie", "value", 60, "/", "", false, true)

	cookies := rec.Header().Values("Set-Cookie")
	if len(cookies) != 2 {
		t.Fatalf("Set-Cookie count = %d, want 2", len(cookies))
	}
	if !strings.Contains(cookies[0], "SameSite=Strict") {
		t.Fatalf("options cookie = %q, want SameSite=Strict", cookies[0])
	}
	if !strings.Contains(cookies[1], "SameSite=Lax") {
		t.Fatalf("legacy cookie = %q, want SameSite=Lax", cookies[1])
	}
}

// TestGinContextValuesAndAbort verifies context values and abort state. TestGinContextValuesAndAbort 验证上下文值和中止状态。
func TestGinContextValuesAndAbort(t *testing.T) {
	gingonic.SetMode(gingonic.TestMode)
	c, _ := gingonic.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := NewGinContext(c)

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
	ctx.Abort()
	if !ctx.IsAborted() {
		t.Fatal("IsAborted() = false, want true")
	}
}

// TestGinContextRequestCollections verifies headers and repeated query values are preserved. TestGinContextRequestCollections 验证请求头与重复查询参数不会丢失。
func TestGinContextRequestCollections(t *testing.T) {
	gingonic.SetMode(gingonic.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gingonic.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/demo?tag=one&tag=two", nil)
	c.Request.Header.Add("X-Test", "one")
	c.Request.Header.Add("X-Test", "two")
	ctx := NewGinContext(c)

	if values := ctx.GetHeaders()["X-Test"]; len(values) != 2 || values[0] != "one" || values[1] != "two" {
		t.Fatalf("GetHeaders()[X-Test] = %v, want [one two]", values)
	}
	if values := ctx.GetQueryAll()["tag"]; len(values) != 2 || values[0] != "one" || values[1] != "two" {
		t.Fatalf("GetQueryAll()[tag] = %v, want [one two]", values)
	}
}
