package main

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/Zany2/dtoken-go/dtoken"
	hertzdt "github.com/Zany2/dtoken-go/integrations/hertz"
	hertzapp "github.com/cloudwego/hertz/pkg/app"
)

func TestHandleLoginValidation(t *testing.T) {
	setupHertzManager(t)
	ctx := context.Background()

	missing := hertzRequestWithBody(`{"username":"alice"}`)
	handleLogin(ctx, missing)
	if missing.Response.StatusCode() != http.StatusBadRequest {
		t.Fatalf("missing credentials status = %d, want %d", missing.Response.StatusCode(), http.StatusBadRequest)
	}

	invalid := hertzRequestWithBody(`{"username":"alice","password":"bad"}`)
	handleLogin(ctx, invalid)
	if invalid.Response.StatusCode() != http.StatusUnauthorized {
		t.Fatalf("invalid password status = %d, want %d", invalid.Response.StatusCode(), http.StatusUnauthorized)
	}
}

func TestProtectedHandlersAndLogout(t *testing.T) {
	setupHertzManager(t)
	ctx := context.Background()
	token, err := dtoken.Login(ctx, "alice")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err = dtoken.AddRoles(ctx, "alice", []string{"admin"}); err != nil {
		t.Fatalf("AddRoles() error = %v", err)
	}
	if err = dtoken.AddPermissions(ctx, "alice", []string{"article:read"}); err != nil {
		t.Fatalf("AddPermissions() error = %v", err)
	}

	me := hertzRequestWithToken(token)
	hertzdt.GetDTokenContext(me)
	handleMe(ctx, me)
	if me.Response.StatusCode() != http.StatusOK || !strings.Contains(string(me.Response.BodyBytes()), `"loginId":"alice"`) {
		t.Fatalf("/me response status=%d body=%q", me.Response.StatusCode(), me.Response.BodyBytes())
	}

	admin := hertzRequestWithToken(token)
	handleAdmin(ctx, admin)
	if admin.Response.StatusCode() != http.StatusOK {
		t.Fatalf("/admin status = %d, want %d", admin.Response.StatusCode(), http.StatusOK)
	}
	articles := hertzRequestWithToken(token)
	handleArticles(ctx, articles)
	if articles.Response.StatusCode() != http.StatusOK {
		t.Fatalf("/articles status = %d, want %d", articles.Response.StatusCode(), http.StatusOK)
	}

	logout := hertzRequestWithToken(token)
	hertzdt.GetDTokenContext(logout)
	handleLogout(ctx, logout)
	if logout.Response.StatusCode() != http.StatusOK {
		t.Fatalf("logout status = %d, want %d", logout.Response.StatusCode(), http.StatusOK)
	}
	if dtoken.IsLogin(ctx, token) {
		t.Fatal("token remains logged in after logout")
	}
}

func setupHertzManager(t *testing.T) {
	t.Helper()
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	initDToken()
}

func hertzRequestWithBody(body string) *hertzapp.RequestContext {
	ctx := hertzapp.NewContext(0)
	ctx.Request.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/login")
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Request.SetBodyString(body)
	return ctx
}

func hertzRequestWithToken(token string) *hertzapp.RequestContext {
	ctx := hertzapp.NewContext(0)
	ctx.Request.SetMethod(http.MethodGet)
	ctx.Request.SetRequestURI("/protected")
	mgr, _ := dtoken.GetManager()
	ctx.Request.Header.Set(mgr.GetConfig().TokenName, token)
	return ctx
}
