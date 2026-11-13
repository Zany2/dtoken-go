package gin_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Zany2/dtoken-go"
	"github.com/Zany2/dtoken-go/global"
	gindt "github.com/Zany2/dtoken-go/integrations/gin"
	"github.com/gin-gonic/gin"
)

// TestInstanceAndExplicitRegistry checks the public builder and opt-in registry across real middleware. TestInstanceAndExplicitRegistry 通过真实中间件验证公开构建器与显式注册表。
func TestInstanceAndExplicitRegistry(t *testing.T) {
	global.DeleteAllManager()
	t.Cleanup(global.DeleteAllManager)
	var builder *dtoken.Builder = gindt.NewBuilder()
	auth, err := builder.IsPrintBanner(false).IsLog(false).AutoRenew(false).Build()
	if err != nil {
		t.Fatal(err)
	}
	defer auth.Close()
	global.SetManager(auth.Manager())
	token, err := auth.LoginID(context.Background(), "user")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		options []gindt.AuthOption
		want    int
	}{
		{"unbound", nil, http.StatusNotFound},
		{"instance", []gindt.AuthOption{gindt.WithAuth(auth)}, http.StatusNoContent},
		{"registry", []gindt.AuthOption{gindt.WithManagerResolver(global.GetManager)}, http.StatusNoContent},
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
