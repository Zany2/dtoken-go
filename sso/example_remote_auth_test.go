package sso_test

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Zany2/dtoken-go/sso"
)

// ExampleRemoteAuth_Middleware demonstrates protecting an API with per-request center validation. ExampleRemoteAuth_Middleware 演示通过每次请求的中心校验保护 API。
func ExampleRemoteAuth_Middleware() {
	app := sso.NewClientApp(sso.ClientConfig{
		Mode:     sso.ModeRemoteSession,
		ClientID: "orders", ClientSecret: "replace-client-secret",
		ServerURL: "https://sso.example.com",
		CheckSign: true, SecretKey: "replace-signing-secret",
	})
	guard, err := sso.NewRemoteAuth(app, sso.RemoteAuthOptions{
		RequiredScopes: []string{"orders:read"}, Timeout: 2 * time.Second,
	})
	if err != nil {
		panic(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/orders", guard.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, _ := sso.RemoteIdentityFromContext(r.Context())
		_, _ = fmt.Fprintf(w, "Orders for %s", identity.LoginID)
	})))
	// Mount mux on the application's HTTP server; requests carry Authorization: Bearer <remote-session-id>. 将 mux 挂载到应用 HTTP 服务，请求携带 Authorization: Bearer <远程会话 ID>。
}
