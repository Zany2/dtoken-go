package sso

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// TestPartialProtocolConfigPreservesDefaults verifies partial overrides work across client URLs and server routes. TestPartialProtocolConfigPreservesDefaults 验证部分配置覆盖在客户端 URL 和服务端路由中保持可用。
func TestPartialProtocolConfigPreservesDefaults(t *testing.T) {
	endpoints := Endpoints{Authorize: "/center/authorize"}
	params := ParamNames{Client: "application"}
	app := NewClientApp(ClientConfig{
		ClientID:  "app-a",
		ServerURL: "https://center.example.com",
		Endpoints: endpoints,
		Params:    params,
	})
	wantEndpoints := DefaultEndpoints()
	wantEndpoints.Authorize = endpoints.Authorize
	wantParams := DefaultParamNames()
	wantParams.Client = params.Client
	if cfg := app.Config(); cfg.Endpoints != wantEndpoints || cfg.Params != wantParams {
		t.Fatalf("partial client config lost defaults or overrides: %+v", cfg)
	}

	server := NewServer()
	defer server.Close()
	client := newTestClient()
	if err := server.RegisterClient(client); err != nil {
		t.Fatal(err)
	}
	handler := NewHTTPServer(server, HTTPOptions{
		ServerOptions: ServerOptions{Endpoints: endpoints, Params: params},
		LoginIDResolver: func(*http.Request) (string, bool) {
			return "user-1001", true
		},
	})
	if handler.options.Endpoints != wantEndpoints || handler.options.Params != wantParams {
		t.Fatal("partial server config lost defaults or overrides")
	}

	// Exercise registration and authorization with both custom and default fields. 使用自定义字段和默认字段验证路由注册与授权。
	loginURL, err := app.AuthURL(client.RedirectURIs[0], nil)
	if err != nil {
		t.Fatal(err)
	}
	mux := handler.Handler()
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, loginURL, nil))
	if response.Code != http.StatusFound {
		t.Fatalf("authorize status = %d, body = %s", response.Code, response.Body.String())
	}
	callback, err := url.Parse(response.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if callback.Query().Get("ticket") == "" {
		t.Fatalf("authorization callback lacks default ticket parameter: %s", callback)
	}
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/sso/token", nil))
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("default token route status = %d, want 405", response.Code)
	}
	if endpoints.Token != "" || params.Ticket != "" {
		t.Fatal("constructor mutated caller configuration")
	}
}
