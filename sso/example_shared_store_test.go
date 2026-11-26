package sso_test

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Zany2/dtoken-go/sso"
)

// ExampleClientApp_sharedStore composes separate center and application services without an HTTP exchange. ExampleClientApp_sharedStore 组装独立中心与应用服务，不使用 HTTP 兑换。
func ExampleClientApp_sharedStore() {
	ctx := context.Background()
	// Production processes use separate connections to the same Redis database and namespace. 生产进程使用独立连接访问同一个 Redis 数据库及命名空间。
	store := sso.NewMemoryStorage()
	center := sso.NewServer(sso.WithStorage(store), sso.WithKeyPrefix("company:sso:"))
	defer center.Close()
	client := &sso.Client{
		ClientID: "orders", ClientSecret: "exchange-secret", SecretKey: "signing-key",
		RedirectURIs: []string{"https://orders.example.net/sso/callback"},
		Modes:        []sso.Mode{sso.ModeTicket}, RequirePKCE: true,
	}
	if err := center.RegisterClientChecked(client); err != nil {
		panic(err)
	}

	peer := sso.NewServer(sso.WithStorage(store), sso.WithKeyPrefix("company:sso:"))
	defer peer.Close()
	options := sso.DefaultHTTPOptions()
	// This local service only exchanges credentials; it neither resolves browser login nor listens on a port. 此本地服务仅兑换凭证，不解析浏览器登录，也不监听端口。
	options.LoginIDResolver = func(*http.Request) (string, bool) { return "", false }
	local, err := sso.NewHTTPServerChecked(peer, options)
	if err != nil {
		panic(err)
	}
	config := sso.DefaultClientConfig()
	config.ClientID, config.ClientSecret, config.SecretKey = client.ClientID, client.ClientSecret, client.SecretKey
	config.ServerURL = "https://login.example.org"
	config.SharedStoreServer = local
	config.RegisterCallback, config.LogoutCallbackURL = true, "https://orders.example.net/sso/logout-callback"
	app, err := sso.NewClientAppChecked(config)
	if err != nil {
		panic(err)
	}

	// BrowserClient normally creates the verifier and drives the redirect; the center issues only after authentication. 通常由 BrowserClient 创建证明并驱动跳转，中心仅在认证成功后签发。
	verifier := sso.NewPKCEVerifier()
	challenge, err := sso.PKCEChallenge(verifier)
	if err != nil {
		panic(err)
	}
	ticket, err := center.GenerateTicketWithOptions(ctx, client.ClientID, "alice", client.RedirectURIs[0], nil, nil,
		sso.OneTimeCredentialOptions{CodeChallenge: challenge, CodeChallengeMethod: sso.PKCEMethodS256})
	if err != nil {
		panic(err)
	}
	req := sso.CredentialRequest{Mode: sso.ModeTicket, Ticket: ticket.Ticket, RedirectURI: client.RedirectURIs[0], CodeVerifier: verifier}
	identity, err := app.ExchangeCredential(ctx, req)
	if err != nil {
		panic(err)
	}
	_, replayErr := app.ExchangeCredential(ctx, req)
	bindings, err := center.GetClientSessions(ctx, identity.LoginID)
	if err != nil {
		panic(err)
	}
	fmt.Println(identity.LoginID, replayErr != nil, len(bindings))
	// Output: alice true 1
}
