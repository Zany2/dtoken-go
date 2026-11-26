package sso

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/contract"
)

// TestPKCERFC7636Vector checks the published S256 vector and rejects unsupported encodings. TestPKCERFC7636Vector 检查公开的 S256 向量并拒绝不支持的编码。
func TestPKCERFC7636Vector(t *testing.T) {
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	want := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
	got, err := PKCEChallenge(verifier)
	if err != nil || got != want {
		t.Fatalf("S256 = %q, %v", got, err)
	}
	for _, invalid := range []string{"", strings.Repeat("x", 42), strings.Repeat("x", 129), strings.Repeat("x", 42) + "=", strings.Repeat("x", 42) + "中"} {
		if _, err := PKCEChallenge(invalid); !errors.Is(err, ErrInvalidPKCE) {
			t.Fatalf("accepted invalid verifier length %d", len(invalid))
		}
	}
	for _, pair := range [][2]string{{want, "plain"}, {want, ""}, {"", PKCEMethodS256}, {want + "=", PKCEMethodS256}, {strings.Repeat("!", 43), PKCEMethodS256}, {want[:42] + "N", PKCEMethodS256}} {
		if err := validatePKCEChallenge(pair[0], pair[1], false); !errors.Is(err, ErrInvalidPKCE) {
			t.Fatalf("accepted invalid challenge: %v", pair)
		}
	}
	for _, length := range []int{43, 128} {
		if _, err := PKCEChallenge(strings.Repeat("~", length)); err != nil {
			t.Fatal(err)
		}
	}
	if first, second := NewPKCEVerifier(), NewPKCEVerifier(); first == second || !validPKCEVerifier(first) || !validPKCEVerifier(second) {
		t.Fatal("invalid random verifier generation")
	}
}

// issuePKCETestCredential issues either one-time mode through the public options API. issuePKCETestCredential 通过公开选项 API 签发任一一次性凭证模式。
func issuePKCETestCredential(s *Server, client *Client, mode Mode, options OneTimeCredentialOptions) (string, error) {
	if mode == ModeTicket {
		value, err := s.GenerateTicketWithOptions(context.Background(), client.ClientID, "alice", client.RedirectURIs[0], nil, nil, options)
		if err != nil {
			return "", err
		}
		return value.Ticket, nil
	}
	value, err := s.GenerateOAuth2CodeWithOptions(context.Background(), client.ClientID, "alice", client.RedirectURIs[0], nil, nil, options)
	if err != nil {
		return "", err
	}
	return value.Code, nil
}

// consumePKCETestCredential exercises either the legacy or proof-aware consumption entry point. consumePKCETestCredential 调用旧消费入口或支持证明的消费入口。
func consumePKCETestCredential(s *Server, client *Client, mode Mode, value, verifier string, legacy bool) error {
	ctx := context.Background()
	if mode == ModeTicket {
		if legacy {
			_, err := s.ConsumeTicket(ctx, value, client.ClientID, client.ClientSecret, client.RedirectURIs[0])
			return err
		}
		_, err := s.ConsumeTicketWithVerifier(ctx, value, client.ClientID, client.ClientSecret, client.RedirectURIs[0], verifier)
		return err
	}
	if legacy {
		_, err := s.ConsumeOAuth2Code(ctx, value, client.ClientID, client.ClientSecret, client.RedirectURIs[0])
		return err
	}
	_, err := s.ConsumeOAuth2CodeWithVerifier(ctx, value, client.ClientID, client.ClientSecret, client.RedirectURIs[0], verifier)
	return err
}

// TestPKCEConsumptionLifecycle covers proof failures, compatibility, revocation, and single consumption on both storage capabilities. TestPKCEConsumptionLifecycle 覆盖两种存储能力下的证明失败、兼容性、撤销和一次性消费。
func TestPKCEConsumptionLifecycle(t *testing.T) {
	for _, mode := range []Mode{ModeTicket, ModeOAuth2} {
		for _, basic := range []bool{false, true} {
			name := string(mode) + map[bool]string{false: "/atomic", true: "/ordinary"}[basic]
			t.Run(name, func(t *testing.T) {
				memory := NewMemoryStorage()
				var storage contract.Storage = memory
				if basic {
					storage = &basicSSOStorage{inner: memory}
				}
				s := NewServer(WithStorage(storage))
				client := newTestClient()
				client.Modes = []Mode{mode}
				if err := s.RegisterClient(client); err != nil {
					t.Fatal(err)
				}
				verifier := NewPKCEVerifier()
				challenge, _ := PKCEChallenge(verifier)
				options := OneTimeCredentialOptions{CodeChallenge: challenge, CodeChallengeMethod: PKCEMethodS256, Timeout: time.Minute}
				value, err := issuePKCETestCredential(s, client, mode, options)
				if err != nil {
					t.Fatal(err)
				}
				if err := consumePKCETestCredential(s, client, mode, value, "", true); !errors.Is(err, ErrInvalidPKCE) {
					t.Fatalf("legacy bypass: %v", err)
				}
				for _, proof := range []string{"", "short", NewPKCEVerifier()} {
					if err := consumePKCETestCredential(s, client, mode, value, proof, false); !errors.Is(err, ErrInvalidPKCE) {
						t.Fatalf("invalid proof accepted: %v", err)
					}
				}
				wrongClient := *client
				wrongClient.ClientSecret = "wrong"
				if err := consumePKCETestCredential(s, &wrongClient, mode, value, verifier, false); !errors.Is(err, ErrInvalidClientCredentials) {
					t.Fatalf("client authentication bypass: %v", err)
				}
				wrongClient = *client
				wrongClient.RedirectURIs = []string{"https://other.example/callback"}
				if err := consumePKCETestCredential(s, &wrongClient, mode, value, verifier, false); !errors.Is(err, ErrRedirectURIMismatch) {
					t.Fatalf("redirect binding bypass: %v", err)
				}
				var successes atomic.Int32
				var wg sync.WaitGroup
				for range 8 {
					wg.Add(1)
					go func() {
						defer wg.Done()
						if consumePKCETestCredential(s, client, mode, value, verifier, false) == nil {
							successes.Add(1)
						}
					}()
				}
				wg.Wait()
				if successes.Load() != 1 {
					t.Fatalf("successful consumers = %d", successes.Load())
				}
				legacy, err := issuePKCETestCredential(s, client, mode, OneTimeCredentialOptions{})
				if err != nil {
					t.Fatal(err)
				}
				if err := consumePKCETestCredential(s, client, mode, legacy, verifier, false); !errors.Is(err, ErrInvalidPKCE) {
					t.Fatalf("accepted stripped binding: %v", err)
				}
				if err := consumePKCETestCredential(s, client, mode, legacy, "", true); err != nil {
					t.Fatalf("legacy compatibility: %v", err)
				}
				revoked, err := issuePKCETestCredential(s, client, mode, options)
				if err != nil {
					t.Fatal(err)
				}
				if err := s.RevokeLoginCredentials(context.Background(), "alice"); err != nil {
					t.Fatal(err)
				}
				if err := consumePKCETestCredential(s, client, mode, revoked, verifier, false); err == nil {
					t.Fatal("proof reactivated a revoked credential")
				}
			})
		}
	}
}

// TestPKCEPolicyChanges enforces the current registration on old records and all issuance entry points. TestPKCEPolicyChanges 对旧记录和所有签发入口执行当前注册策略。
func TestPKCEPolicyChanges(t *testing.T) {
	for _, mode := range []Mode{ModeTicket, ModeOAuth2} {
		t.Run(string(mode), func(t *testing.T) {
			s := NewServer()
			defer s.Close()
			client := newTestClient()
			client.Modes = []Mode{mode}
			if err := s.RegisterClient(client); err != nil {
				t.Fatal(err)
			}
			legacy, err := issuePKCETestCredential(s, client, mode, OneTimeCredentialOptions{})
			if err != nil {
				t.Fatal(err)
			}
			client.RequirePKCE = true
			if err := s.RegisterClient(client); err != nil {
				t.Fatal(err)
			}
			if err := consumePKCETestCredential(s, client, mode, legacy, "", true); !errors.Is(err, ErrPKCERequired) {
				t.Fatalf("legacy policy bypass: %v", err)
			}
			if mode == ModeTicket {
				_, err = s.GenerateTicket(context.Background(), client.ClientID, "alice", client.RedirectURIs[0], nil, nil)
			} else {
				_, err = s.GenerateOAuth2Code(context.Background(), client.ClientID, "alice", client.RedirectURIs[0], nil, nil)
			}
			if !errors.Is(err, ErrPKCERequired) {
				t.Fatalf("unbound issuance: %v", err)
			}
			verifier := NewPKCEVerifier()
			challenge, _ := PKCEChallenge(verifier)
			bound, err := issuePKCETestCredential(s, client, mode, OneTimeCredentialOptions{CodeChallenge: challenge, CodeChallengeMethod: PKCEMethodS256})
			if err != nil {
				t.Fatal(err)
			}
			client.RequirePKCE = false
			if err := s.RegisterClient(client); err != nil {
				t.Fatal(err)
			}
			if err := consumePKCETestCredential(s, client, mode, bound, "", true); !errors.Is(err, ErrInvalidPKCE) {
				t.Fatalf("policy downgrade removed stored binding: %v", err)
			}
			if err := consumePKCETestCredential(s, client, mode, bound, verifier, false); err != nil {
				t.Fatal(err)
			}
		})
	}
}
