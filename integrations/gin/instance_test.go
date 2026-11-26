package gin_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Zany2/dtoken-go"
	gindt "github.com/Zany2/dtoken-go/integrations/gin"
	"github.com/gin-gonic/gin"
)

// TestInstanceAndExplicitRegistry checks automatic registration and explicit HTTP resolver binding. TestInstanceAndExplicitRegistry 验证自动注册与 HTTP 解析器的显式绑定。
func TestInstanceAndExplicitRegistry(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	var builder *dtoken.Builder = gindt.NewBuilder()
	auth, err := builder.IsPrintBanner(false).IsLog(false).AutoRenew(false).Build()
	if err != nil {
		t.Fatal(err)
	}
	defer auth.Close()
	token, err := auth.LoginID(context.Background(), "user")
	if err != nil {
		t.Fatal(err)
	}
	authOption, err := gindt.WithAuth(auth)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		options []gindt.AuthOption
		want    int
	}{
		{"unbound", nil, http.StatusNotFound},
		{"instance", []gindt.AuthOption{authOption}, http.StatusNoContent},
		{"registry", []gindt.AuthOption{gindt.WithManagerResolver(dtoken.GetManager)}, http.StatusNoContent},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.Use(gindt.RegisterDTokenContextMiddleware(context.Background(), tc.options...))
			router.Use(gindt.RequireLogin())
			router.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.Header.Set(auth.Config().TokenName, token)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != tc.want {
				t.Fatalf("status = %d, want %d", response.Code, tc.want)
			}
		})
	}
}

// TestNamedRegistryResolverFollowsReplacement verifies existing middleware selects the current engine and never falls back for a missing namespace. TestNamedRegistryResolverFollowsReplacement 验证既有中间件选择当前引擎，缺失体系不会回退。
func TestNamedRegistryResolverFollowsReplacement(t *testing.T) {
	dtoken.CloseAll()
	t.Cleanup(dtoken.CloseAll)
	ctx := context.Background()
	builder := dtoken.NewBuilder().IsPrintBanner(false).IsLog(false).AutoRenew(false).AsyncEvent(false)
	defaultAuth, err := builder.Clone().Build()
	if err != nil {
		t.Fatal(err)
	}
	userAuth, err := builder.Clone().AuthType("user").Build()
	if err != nil {
		t.Fatal(err)
	}
	tokenName := userAuth.Config().TokenName
	login := func(auth *dtoken.Auth) string {
		t.Helper()
		token, err := auth.LoginID(ctx, "user")
		if err != nil {
			t.Fatal(err)
		}
		return token
	}
	defaultToken, oldToken := login(defaultAuth), login(userAuth)
	router := gin.New()
	router.Use(gindt.RegisterDTokenContextMiddleware(ctx, gindt.WithManagerResolver(dtoken.GetManager), gindt.WithAuthType("user")))
	router.Use(gindt.RequireLogin())
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	check := func(token string, want int) {
		t.Helper()
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set(tokenName, token)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != want {
			t.Fatalf("status = %d, want %d", response.Code, want)
		}
	}
	check(oldToken, http.StatusNoContent)
	check(defaultToken, http.StatusUnauthorized)

	replacement, err := builder.Clone().AuthType("user").BuildInstance()

	if err != nil {

		t.Fatal(err)

	}
	t.Cleanup(replacement.Close)
	if err := dtoken.SetAuth(replacement); err != nil {
		t.Fatalf("register authentication: %v", err)
	}
	check(login(replacement), http.StatusNoContent)
	check(oldToken, http.StatusUnauthorized)
	dtoken.AuthType("user").Close()
	check(defaultToken, http.StatusNotFound)
}
