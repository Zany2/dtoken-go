package gin_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Zany2/dtoken-go"
	derror "github.com/Zany2/dtoken-go/errors"
	requestcontext "github.com/Zany2/dtoken-go/httpauth/context"
	gindt "github.com/Zany2/dtoken-go/integrations/gin"
	"github.com/Zany2/dtoken-go/nonce"
	"github.com/gin-gonic/gin"
)

// TestExplicitCapabilitiesSurviveLoginMiddleware verifies middleware preserves request service bindings without global registration. TestExplicitCapabilitiesSurviveLoginMiddleware 验证登录中间件无需全局注册即可保留请求服务绑定。
func TestExplicitCapabilitiesSurviveLoginMiddleware(t *testing.T) {
	auth, err := dtoken.NewBuilder().IsPrintBanner(false).IsLog(false).AutoRenew(false).Build()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(auth.Close)
	ctx := context.Background()
	token, err := auth.LoginID(ctx, "user")
	if err != nil {
		t.Fatal(err)
	}
	cfg := auth.Config()
	service := nonce.NewService(nonce.NewDefaultNonceManager(cfg.AuthType, cfg.KeyPrefix, auth.Storage()), auth)
	bindings := requestcontext.Capabilities{Nonce: service}
	router := gin.New()
	router.Use(gindt.RegisterDTokenContextMiddleware(ctx, gindt.WithAuth(auth), gindt.WithCapabilities(bindings)))
	router.Use(gindt.AuthMiddleware(ctx))
	router.GET("/nonce", func(c *gin.Context) {
		value, err := gindt.GenerateNonceByContext(c)
		if err != nil {
			t.Error(err)
			c.Status(http.StatusInternalServerError)
			return
		}
		c.String(http.StatusOK, value)
	})
	request := httptest.NewRequest(http.MethodGet, "/nonce", nil)
	request.Header.Set(cfg.TokenName, token)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !service.VerifyNonce(ctx, response.Body.String()) {
		t.Fatalf("request did not use the injected service: status=%d body=%q", response.Code, response.Body.String())
	}
}

// TestManagerSwitchClearsCapabilities verifies a new auth scope cannot borrow another instance's services. TestManagerSwitchClearsCapabilities 验证新认证作用域不会借用另一实例的服务。
func TestManagerSwitchClearsCapabilities(t *testing.T) {
	first, err := dtoken.NewBuilder().AuthType("first").IsPrintBanner(false).IsLog(false).AutoRenew(false).Build()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(first.Close)
	second, err := dtoken.NewBuilder().AuthType("second").IsPrintBanner(false).IsLog(false).AutoRenew(false).Build()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(second.Close)
	ctx := context.Background()
	token, err := second.LoginID(ctx, "user")
	if err != nil {
		t.Fatal(err)
	}
	cfg := first.Config()
	service := nonce.NewService(nonce.NewDefaultNonceManager(cfg.AuthType, cfg.KeyPrefix, first.Storage()), first)
	router := gin.New()
	router.Use(gindt.RegisterDTokenContextMiddleware(ctx, gindt.WithAuth(first), gindt.WithCapabilities(requestcontext.Capabilities{Nonce: service})))
	router.Use(gindt.AuthMiddleware(ctx, gindt.WithAuth(second)))
	router.GET("/nonce", func(c *gin.Context) {
		_, err := gindt.GenerateNonceByContext(c)
		if !errors.Is(err, derror.ErrModuleNotEnabled) {
			t.Errorf("new scope nonce error = %v, want ErrModuleNotEnabled", err)
		}
		c.Status(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodGet, "/nonce", nil)
	request.Header.Set(second.Config().TokenName, token)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("switched scope response = %d, want 204", response.Code)
	}
}
