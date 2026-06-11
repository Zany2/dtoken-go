package fiber

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Zany2/dtoken-go/core/adapter"
	gofiber "github.com/gofiber/fiber/v2"
)

func TestFiberContextAdapterRequestAndResponse(t *testing.T) {
	app := gofiber.New()
	app.Post("/demo", func(c *gofiber.Ctx) error {
		ctx := NewFiberContext(c)

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
		if got := ctx.GetMethod(); got != http.MethodPost {
			t.Fatalf("GetMethod() = %q, want POST", got)
		}
		if got := ctx.GetPath(); got != "/demo" {
			t.Fatalf("GetPath() = %q, want /demo", got)
		}
		if got := ctx.GetURL(); got != "/demo?foo=bar" {
			t.Fatalf("GetURL() = %q, want /demo?foo=bar", got)
		}
		if got := ctx.GetUserAgent(); got != "fiber-test" {
			t.Fatalf("GetUserAgent() = %q, want fiber-test", got)
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
		_, err = ctx.Write([]byte("done"))
		return err
	})

	req, err := http.NewRequest(http.MethodPost, "/demo?foo=bar", strings.NewReader("hello"))
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
	if len(resp.Cookies()) == 0 {
		t.Fatal("SetCookieWithOptions() did not write cookie")
	}
}

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
