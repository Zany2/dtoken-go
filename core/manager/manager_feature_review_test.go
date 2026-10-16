package manager

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/listener"
	"github.com/Zany2/dtoken-go/core/nonce"
	"github.com/Zany2/dtoken-go/core/oauth2"
	"github.com/Zany2/dtoken-go/core/shortkey"
	"github.com/Zany2/dtoken-go/core/ticket"
)

// TestManagerCredentialEventDeadlines verifies events honor precise deadlines and legacy payloads. TestManagerCredentialEventDeadlines 验证事件遵循精确截止时间并兼容旧载荷。
func TestManagerCredentialEventDeadlines(t *testing.T) {
	for _, kind := range []string{"ticket", "short key"} {
		for _, mode := range []string{"precise live", "precise expired", "precise without legacy ttl", "legacy live", "legacy expired"} {
			t.Run(kind+"/"+mode, func(t *testing.T) {
				mgr := newTestManager(t, nil)
				now := time.Now()
				created, expiresIn := now.Unix()-600, int64(1)
				deadline := now.Add(time.Minute)
				wantLive := true
				switch mode {
				case "precise expired":
					created, expiresIn = now.Unix(), 600
					deadline, wantLive = now.Add(-time.Minute), false
				case "precise without legacy ttl":
					expiresIn = 0
				case "legacy live":
					created, expiresIn, deadline = now.Unix(), 60, time.Time{}
				case "legacy expired":
					deadline, wantLive = time.Time{}, false
				}
				events := registerTerminalLifecycleEvents(mgr)
				if kind == "ticket" {
					value := &ticket.Ticket{Ticket: "event-ticket", LoginID: "event-user", CreateTime: created, ExpiresIn: expiresIn, ExpiresAt: deadline}
					mgr.triggerTicketEvent(listener.EventTicketValidate, value, listener.ActionValidate)
				} else {
					value := &shortkey.ShortKey{Key: "event-key", LoginID: "event-user", CreateTime: created, ExpiresIn: expiresIn, ExpiresAt: deadline}
					mgr.triggerShortKeyEvent(listener.EventShortKeyValidate, value, listener.ActionValidate)
				}
				if len(events()) != 1 {
					t.Fatalf("events = %d, want 1", len(events()))
				}
				ttl, ok := events()[0].Extra[listener.ExtraKeyTTL].(int64)
				if !ok || (wantLive && (ttl < 1 || ttl > 60)) || (!wantLive && ttl != 0) {
					t.Fatalf("event TTL = %v, want live=%v", events()[0].Extra[listener.ExtraKeyTTL], wantLive)
				}
			})
		}
	}
	if remainingTicketTTLSeconds(nil) != 0 || remainingShortKeyTTLSeconds(nil) != 0 {
		t.Fatal("nil credential reported a lifetime")
	}
}

// TestManagerCredentialConsumeErrorsHaveNoSuccessEvent checks constraints, storage errors, and replay through the wrappers. TestManagerCredentialConsumeErrorsHaveNoSuccessEvent 检查封装层的约束、存储错误和重复消费。
func TestManagerCredentialConsumeErrorsHaveNoSuccessEvent(t *testing.T) {
	ctx := context.Background()
	for _, kind := range []string{"ticket", "short key"} {
		t.Run(kind, func(t *testing.T) {
			mgr := newTestManager(t, nil)
			storage := &managerFailingStorage{Storage: mgr.storage}
			var consume func(string) error
			var mismatch error
			var consumedEvent listener.Event
			if kind == "ticket" {
				WithTicketManager(ticket.NewDefaultManager(mgr.config.AuthType, mgr.config.KeyPrefix, storage, mgr.serializer))(mgr)
				value, err := mgr.CreateTicket(ctx, ticket.CreateOptions{LoginID: "consume-user", TargetApp: "admin"})
				if err != nil {
					t.Fatal(err)
				}
				consume = func(target string) error {
					_, err := mgr.ConsumeTicket(ctx, value.Ticket, ticket.ValidateOptions{TargetApp: target})
					return err
				}
				mismatch, consumedEvent = ticket.ErrTicketMismatch, listener.EventTicketConsume
			} else {
				WithShortKeyManager(shortkey.NewDefaultManager(mgr.config.AuthType, mgr.config.KeyPrefix, storage, mgr.serializer))(mgr)
				value, err := mgr.CreateShortKey(ctx, shortkey.CreateOptions{LoginID: "consume-user", TargetApp: "admin"})
				if err != nil {
					t.Fatal(err)
				}
				consume = func(target string) error {
					_, err := mgr.ConsumeShortKey(ctx, value.Key, shortkey.ValidateOptions{TargetApp: target})
					return err
				}
				mismatch, consumedEvent = shortkey.ErrShortKeyMismatch, listener.EventShortKeyConsume
			}
			events := registerManagerValidationEventCollector(mgr, consumedEvent)
			if err := consume("other-app"); !errors.Is(err, mismatch) {
				t.Fatalf("mismatched consume = %v, want %v", err, mismatch)
			}
			storage.deleteErr = errors.New("consume storage unavailable")
			if err := consume("admin"); !errors.Is(err, derror.ErrStorageUnavailable) {
				t.Fatalf("failed storage consume = %v", err)
			}
			if len(events()) != 0 {
				t.Fatal("failed consume emitted a success event")
			}
			storage.deleteErr = nil
			if err := consume("admin"); err != nil {
				t.Fatal(err)
			}
			if err := consume("admin"); err == nil {
				t.Fatal("replayed credential consumed twice")
			}
			if len(events()) != 1 || events()[0].LoginID != "consume-user" {
				t.Fatalf("consume events = %+v, want one successful subject event", events())
			}
		})
	}
}

// TestManagerNoncePreservesCapabilityErrors verifies unsupported consumption cannot report success or delete the nonce. TestManagerNoncePreservesCapabilityErrors 验证不支持原子消费时不误报成功或删除 Nonce。
func TestManagerNoncePreservesCapabilityErrors(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)
	storage := &managerFailingStorage{Storage: mgr.storage}
	WithNonceManager(nonce.NewDefaultNonceManager(mgr.config.AuthType, mgr.config.KeyPrefix, storage))(mgr)
	value, err := mgr.GenerateNonce(ctx)
	if err != nil {
		t.Fatal(err)
	}
	events := registerManagerValidationEventCollector(mgr, listener.EventNonceVerify)
	if err := mgr.VerifyAndConsumeNonce(ctx, value); !errors.Is(err, derror.ErrStorageCapabilityUnsupported) {
		t.Fatalf("unsupported consume = %v", err)
	}
	if mgr.VerifyNonce(ctx, value) || !mgr.IsNonceValid(ctx, value) {
		t.Fatal("unsupported consumption succeeded or removed the nonce")
	}
	if len(events()) != 2 {
		t.Fatalf("verify events = %d, want 2", len(events()))
	}
	for _, event := range events() {
		if event.Extra[listener.ExtraKeyResult] != false || event.Token != value {
			t.Fatalf("incorrect failure event: %+v", event)
		}
	}
}

// TestManagerOAuth2PKCEFailureDoesNotIssueToken checks both exchange wrappers preserve PKCE verification. TestManagerOAuth2PKCEFailureDoesNotIssueToken 验证两种换取入口均保留 PKCE 校验。
func TestManagerOAuth2PKCEFailureDoesNotIssueToken(t *testing.T) {
	ctx := context.Background()
	for name, unified := range map[string]bool{"direct": false, "unified": true} {
		t.Run(name, func(t *testing.T) {
			mgr := newTestManagerWithOAuth2(t)
			client := managerOAuth2TestClient()
			if err := mgr.RegisterOAuth2Client(client); err != nil {
				t.Fatal(err)
			}
			verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
			code, err := mgr.GenerateOAuth2AuthorizationCodeWithPKCE(ctx, client.ClientID, "pkce-user", client.RedirectURIs[0], []string{"read"}, verifier, oauth2.CodeChallengeMethodPlain)
			if err != nil {
				t.Fatal(err)
			}
			events := registerManagerValidationEventCollector(mgr, listener.EventOAuth2TokenIssue)
			exchange := func(proof string) (*oauth2.AccessToken, error) {
				if unified {
					return mgr.OAuth2Token(ctx, &oauth2.TokenRequest{GrantType: oauth2.GrantTypeAuthorizationCode, ClientID: client.ClientID, ClientSecret: client.ClientSecret, Code: code.Code, RedirectURI: client.RedirectURIs[0], CodeVerifier: proof}, nil)
				}
				return mgr.ExchangeOAuth2CodeForTokenWithPKCE(ctx, code.Code, client.ClientID, client.ClientSecret, client.RedirectURIs[0], proof)
			}
			if token, err := exchange("x" + verifier[1:]); err == nil || token != nil || len(events()) != 0 {
				t.Fatalf("invalid PKCE exchange = %+v, %v, events=%d", token, err, len(events()))
			}
			if token, err := exchange(verifier); err != nil || token == nil || token.UserID != "pkce-user" || len(events()) != 1 {
				t.Fatalf("valid PKCE exchange = %+v, %v, events=%d", token, err, len(events()))
			}
		})
	}
}
