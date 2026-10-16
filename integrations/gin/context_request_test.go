package gin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gingonic "github.com/gin-gonic/gin"
)

// TestGinContextRequestAndResponse verifies the full Gin adapter surface. TestGinContextRequestAndResponse 验证 Gin 适配器的完整接口行为。
func TestGinContextRequestAndResponse(t *testing.T) {
	gingonic.SetMode(gingonic.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gingonic.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/demo?foo=bar", strings.NewReader("hello"))
	c.Request.Header.Set("X-Token", "token")
	c.Request.Header.Set("User-Agent", "gin-test")
	c.Request.AddCookie(&http.Cookie{Name: "sid", Value: "cookie-token"})
	ctx := NewGinContext(c)

	if got := ctx.GetHeader("X-Token"); got != "token" {
		t.Fatalf("GetHeader() = %q, want token", got)
	}
	if got := ctx.GetHeaders()["X-Token"]; len(got) != 1 || got[0] != "token" {
		t.Fatalf("GetHeaders()[X-Token] = %v, want [token]", got)
	}
	if got := ctx.GetQuery("foo"); got != "bar" {
		t.Fatalf("GetQuery() = %q, want bar", got)
	}
	if got := ctx.GetPostForm("missing"); got != "" {
		t.Fatalf("GetPostForm(missing) = %q, want empty", got)
	}
	if got := ctx.GetCookie("sid"); got != "cookie-token" {
		t.Fatalf("GetCookie() = %q, want cookie-token", got)
	}
	body, err := ctx.GetBody()
	if err != nil || string(body) != "hello" {
		t.Fatalf("GetBody() = %q, %v, want hello", body, err)
	}
	if got := ctx.GetClientIP(); got == "" {
		t.Fatal("GetClientIP() returned empty value")
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
	if got := ctx.GetUserAgent(); got != "gin-test" {
		t.Fatalf("GetUserAgent() = %q, want gin-test", got)
	}
	if ctx.IsTLS() {
		t.Fatal("IsTLS() = true for HTTP request")
	}

	ctx.SetHeader("X-Result", "ok")
	ctx.SetStatusCode(http.StatusAccepted)
	if _, err = ctx.Write([]byte("done")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", recorder.Code)
	}
	if got := recorder.Header().Get("X-Result"); got != "ok" {
		t.Fatalf("response header = %q, want ok", got)
	}
	if got := recorder.Body.String(); got != "done" {
		t.Fatalf("response body = %q, want done", got)
	}
}
