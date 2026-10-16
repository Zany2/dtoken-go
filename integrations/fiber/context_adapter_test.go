package fiber

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Zany2/dtoken-go/core/adapter"
	gofiber "github.com/gofiber/fiber/v2"
)

// TestFiberContextAdapterRequestAndResponse verifies request and response adaptation. TestFiberContextAdapterRequestAndResponse 验证请求与响应适配。
func TestFiberContextAdapterRequestAndResponse(t *testing.T) {
	app := gofiber.New()
	app.Post("/demo", func(c *gofiber.Ctx) error {
		ctx := NewFiberContext(c)

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
		if got := ctx.GetMethod(); got != http.MethodPost {
			t.Fatalf("GetMethod() = %q, want POST", got)
		}
		if got := ctx.GetPath(); got != "/demo" {
			t.Fatalf("GetPath() = %q, want /demo", got)
		}
		if got := ctx.GetURL(); got != "/demo?foo=bar&foo=baz" {
			t.Fatalf("GetURL() = %q, want /demo?foo=bar&foo=baz", got)
		}
		if got := ctx.GetUserAgent(); got != "fiber-test" {
			t.Fatalf("GetUserAgent() = %q, want fiber-test", got)
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
		ctx.SetCookieWithOptions(&adapter.CookieOptions{Name: "dt", Value: "v", Path: "/", SameSite: "Strict"})
		ctx.SetStatusCode(http.StatusAccepted)
		_, err = ctx.Write([]byte("done"))
		return err
	})

	req, err := http.NewRequest(http.MethodPost, "/demo?foo=bar&foo=baz", strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("X-Token", "token")
	req.Header.Set("User-Agent", "fiber-test")
	req.AddCookie(&http.Cookie{Name: "sid", Value: "cookie-token"})

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", resp.StatusCode)
	}
	if got := resp.Header.Get("X-Result"); got != "ok" {
		t.Fatalf("response header = %q, want ok", got)
	}
	if string(body) != "done" {
		t.Fatalf("response body = %q, want done", body)
	}
	if len(resp.Cookies()) != 2 {
		t.Fatalf("Set-Cookie count = %d, want 2", len(resp.Cookies()))
	}
}

// TestFiberContextMustGetPanicsWhenMissing verifies missing values trigger a panic. TestFiberContextMustGetPanicsWhenMissing 验证缺失值会触发 panic。
func TestFiberContextMustGetPanicsWhenMissing(t *testing.T) {
	app := gofiber.New()
	app.Get("/", func(c *gofiber.Ctx) error {
		ctx := NewFiberContext(c)
		defer func() {
			if recover() == nil {
				t.Fatal("MustGet(missing) should panic")
			}
		}()
		ctx.MustGet("missing")
		return nil
	})

	req, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	if _, err = app.Test(req); err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
}
