package context

import (
	"context"
	"errors"
	"testing"

	"github.com/Zany2/dtoken-go"
	derror "github.com/Zany2/dtoken-go/errors"
	"github.com/Zany2/dtoken-go/event"
	"github.com/Zany2/dtoken-go/nonce"
	"github.com/Zany2/dtoken-go/oauth2"
	"github.com/Zany2/dtoken-go/shortkey"
	"github.com/Zany2/dtoken-go/ticket"
)

// TestCapabilitiesRequireExplicitBindings verifies missing services retain the disabled-module contract. TestCapabilitiesRequireExplicitBindings 验证未绑定服务保留模块未启用契约。
func TestCapabilitiesRequireExplicitBindings(t *testing.T) {
	dCtx := NewContext(&testRequestContext{}, nil)
	ctx := context.Background()
	_, nonceErr := dCtx.Nonce().Generate(ctx)
	_, ticketErr := dCtx.Ticket().Create(ctx, ticket.CreateOptions{})
	_, shortKeyErr := dCtx.ShortKey().Create(ctx, shortkey.CreateOptions{})
	oauth2Err := dCtx.OAuth2().RegisterClient(&oauth2.Client{})
	for name, err := range map[string]error{"nonce": nonceErr, "ticket": ticketErr, "shortkey": shortKeyErr, "oauth2": oauth2Err} {
		if !errors.Is(err, derror.ErrModuleNotEnabled) {
			t.Errorf("%s error = %v, want ErrModuleNotEnabled", name, err)
		}
	}
}

// TestCapabilitiesUseComposedServices verifies request facades share capability storage and instance events. TestCapabilitiesUseComposedServices 验证请求门面复用组合服务的存储与实例事件。
func TestCapabilitiesUseComposedServices(t *testing.T) {
	auth, err := dtoken.NewBuilder().AuthType("capabilities").IsPrintBanner(false).IsLog(false).AutoRenew(false).AsyncEvent(false).Build()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(auth.Close)
	ctx := context.Background()
	cfg := auth.Config()
	nonceService := nonce.NewService(nonce.NewDefaultNonceManager(cfg.AuthType, cfg.KeyPrefix, auth.Storage()), auth)
	ticketService := ticket.NewService(ticket.NewDefaultManager(cfg.AuthType, cfg.KeyPrefix, auth.Storage(), auth.Codec()), auth)
	shortKeyService := shortkey.NewService(shortkey.NewDefaultManager(cfg.AuthType, cfg.KeyPrefix, auth.Storage(), auth.Codec()), auth)
	oauth2Service := oauth2.NewService(oauth2.NewDefaultOAuth2Server(cfg.AuthType, cfg.KeyPrefix, auth.Storage(), auth.Codec()), auth)
	bindings := Capabilities{Nonce: nonceService, Ticket: ticketService, ShortKey: shortKeyService, OAuth2: oauth2Service}
	dCtx := NewContext(&testRequestContext{}, auth.Manager()).WithCapabilities(&bindings)
	bindings = Capabilities{}
	dCtx.WithCapabilities(nil)

	var emittedAuthType string
	auth.EventManager().RegisterFuncWithConfig(event.EventNonceGenerate, func(data *event.EventData) {
		emittedAuthType = data.AuthType
	}, event.ListenerConfig{Async: false})
	value, err := dCtx.Nonce().Generate(ctx)
	if err != nil || !nonceService.VerifyNonce(ctx, value) || emittedAuthType != cfg.AuthType {
		t.Fatalf("nonce composition failed: value=%q event auth type=%q error=%v", value, emittedAuthType, err)
	}

	credential, err := dCtx.Ticket().Create(ctx, ticket.CreateOptions{LoginID: "user"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ticketService.ConsumeTicket(ctx, credential.Ticket); err != nil {
		t.Fatal(err)
	}
	key, err := dCtx.ShortKey().Create(ctx, shortkey.CreateOptions{LoginID: "user"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := shortKeyService.ValidateShortKey(ctx, key.Key); err != nil {
		t.Fatal(err)
	}
	client := &oauth2.Client{ClientID: "app", ClientSecret: "secret", GrantTypes: []oauth2.GrantType{oauth2.GrantTypeClientCredentials}}
	if err := dCtx.OAuth2().RegisterClient(client); err != nil {
		t.Fatal(err)
	}
	if _, err := oauth2Service.GetOAuth2Client(client.ClientID); err != nil {
		t.Fatal(err)
	}

	dCtx.WithCapabilities(&Capabilities{})
	if _, err := dCtx.Nonce().Generate(ctx); !errors.Is(err, derror.ErrModuleNotEnabled) {
		t.Fatalf("cleared binding error = %v, want ErrModuleNotEnabled", err)
	}
}

// TestTypedNilCapabilitiesAreDisabled verifies all interface bindings normalize typed nil services. TestTypedNilCapabilitiesAreDisabled 验证全部接口绑定会归一化带类型空服务。
func TestTypedNilCapabilitiesAreDisabled(t *testing.T) {
	var n *nonce.Service
	var o *oauth2.Service
	var tk *ticket.Service
	var sk *shortkey.Service
	scope := NewContext(nil, nil).WithCapabilities(&Capabilities{Nonce: n, OAuth2: o, Ticket: tk, ShortKey: sk})
	_, nErr := scope.Nonce().Generate(context.Background())
	oErr := scope.OAuth2().RegisterClient(nil)
	_, tErr := scope.Ticket().Create(context.Background(), ticket.CreateOptions{})
	_, sErr := scope.ShortKey().Create(context.Background(), shortkey.CreateOptions{})
	for _, err := range []error{nErr, oErr, tErr, sErr} {
		if !errors.Is(err, derror.ErrModuleNotEnabled) {
			t.Fatalf("typed nil error = %v", err)
		}
	}
}
