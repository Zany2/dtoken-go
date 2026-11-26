package oauth2

import (
	"context"
	"errors"
	"strings"
	"testing"

	derror "github.com/Zany2/dtoken-go/errors"
	listener "github.com/Zany2/dtoken-go/event"
	"github.com/Zany2/dtoken-go/storage/memory"
)

// TestServiceMemoryGrants checks each grant wrapper, token rotation and event outcomes. TestServiceMemoryGrants 验证各授权模式入口、令牌轮换和事件结果。
func TestServiceMemoryGrants(t *testing.T) {
	for _, grant := range []string{"code", "pkce", "credentials", "password"} {
		t.Run(grant, func(t *testing.T) {
			ctx := context.Background()
			server, err := NewDefaultOAuth2Server("service", "test", memory.NewStorage(), oauth2TestCodec{})
			if err != nil {
				t.Fatal(err)
			}
			events := map[listener.Event]int{}
			validations := []bool{}
			service := NewService(server, revokeEmitter(func(event listener.Event, loginID, token string, extra map[string]any) {
				events[event]++
				if event == listener.EventOAuth2TokenValidate {
					result, ok := extra[listener.ExtraKeyResult].(bool)
					if !ok {
						t.Error("missing validation result")
					}
					validations = append(validations, result)
				}
				if event == listener.EventOAuth2TokenIssue && extra[listener.ExtraKeyClientID] != "client-1" {
					t.Errorf("wrong event client: %v", extra)
				}
			}))
			client := oauth2TestClient()
			if err = service.RegisterOAuth2Client(client); err != nil {
				t.Fatal(err)
			}
			var token *AccessToken
			switch grant {
			case "code":
				code, codeErr := service.GenerateOAuth2AuthorizationCode(ctx, client.ClientID, "user", client.RedirectURIs[0], []string{"read"})
				if codeErr != nil {
					t.Fatal(codeErr)
				}
				token, err = service.ExchangeOAuth2CodeForToken(ctx, code.Code, client.ClientID, client.ClientSecret, client.RedirectURIs[0])
				if err != nil {
					t.Fatal(err)
				}
				if _, err = service.ExchangeOAuth2CodeForToken(ctx, code.Code, client.ClientID, client.ClientSecret, client.RedirectURIs[0]); err == nil {
					t.Fatal("code replay accepted")
				}
			case "pkce":
				verifier := strings.Repeat("v", 43)
				code, codeErr := service.GenerateOAuth2AuthorizationCodeWithPKCE(ctx, client.ClientID, "user", client.RedirectURIs[0], []string{"read"}, oauth2PKCEChallenge(verifier), CodeChallengeMethodS256)
				if codeErr != nil {
					t.Fatal(codeErr)
				}
				if _, err = service.ExchangeOAuth2CodeForTokenWithPKCE(ctx, code.Code, client.ClientID, client.ClientSecret, client.RedirectURIs[0], strings.Repeat("x", 43)); !errors.Is(err, derror.ErrInvalidCodeVerifier) {
					t.Fatalf("wrong verifier=%v", err)
				}
				token, err = service.ExchangeOAuth2CodeForTokenWithPKCE(ctx, code.Code, client.ClientID, client.ClientSecret, client.RedirectURIs[0], verifier)
				if err != nil {
					t.Fatal(err)
				}
			case "credentials":
				if _, err = service.OAuth2ClientCredentialsToken(ctx, client.ClientID, "wrong", []string{"read"}); err == nil {
					t.Fatal("wrong secret accepted")
				}
				token, err = service.OAuth2ClientCredentialsToken(ctx, client.ClientID, client.ClientSecret, []string{"read"})
				if err != nil {
					t.Fatal(err)
				}
			case "password":
				validate := func(username, password string) (string, error) {
					if username != "user" || password != "password" {
						return "", errors.New("bad credentials")
					}
					return "user", nil
				}
				if _, err = service.OAuth2PasswordGrantToken(ctx, client.ClientID, client.ClientSecret, "user", "wrong", []string{"read"}, validate); err == nil {
					t.Fatal("wrong password accepted")
				}
				token, err = service.OAuth2PasswordGrantToken(ctx, client.ClientID, client.ClientSecret, "user", "password", []string{"read"}, validate)
				if err != nil {
					t.Fatal(err)
				}
			}
			if events[listener.EventOAuth2TokenIssue] != 1 {
				t.Fatalf("issuance events=%v", events)
			}
			if !service.ValidateOAuth2AccessToken(ctx, token.Token) {
				t.Fatal("issued token invalid")
			}
			rotated, err := service.RefreshOAuth2AccessToken(ctx, client.ClientID, token.RefreshToken, client.ClientSecret)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = service.RefreshOAuth2AccessToken(ctx, client.ClientID, token.RefreshToken, client.ClientSecret); err == nil {
				t.Fatal("refresh replay accepted")
			}
			if service.ValidateOAuth2AccessToken(ctx, token.Token) {
				t.Fatal("rotated access still valid")
			}
			if info, err := service.ValidateOAuth2AccessTokenAndGetInfo(ctx, rotated.Token); err != nil || info.ClientID != client.ClientID || info.UserID != token.UserID {
				t.Fatalf("rotated info=%+v,%v", info, err)
			}
			if err = service.RevokeOAuth2Token(ctx, rotated.Token); err != nil {
				t.Fatal(err)
			}
			if service.ValidateOAuth2AccessToken(ctx, rotated.Token) {
				t.Fatal("revoked access accepted")
			}
			if events[listener.EventOAuth2TokenRefresh] != 1 || events[listener.EventOAuth2TokenRevoke] != 1 {
				t.Fatalf("lifecycle events=%v", events)
			}
			if len(validations) != 4 || !validations[0] || validations[1] || !validations[2] || validations[3] {
				t.Fatalf("validation results=%v", validations)
			}
		})
	}
}

// TestServiceDisabled rejects capability calls without an OAuth2 server. TestServiceDisabled 在未绑定 OAuth2 服务端时拒绝能力调用。
func TestServiceDisabled(t *testing.T) {
	ctx := context.Background()
	s := NewService(nil, nil)
	checks := []func() error{
		func() error { return s.RegisterOAuth2Client(nil) },
		func() error { return s.UnregisterOAuth2Client("client") },
		func() error { _, err := s.GetOAuth2Client("client"); return err },
		func() error { _, err := s.OAuth2Token(ctx, nil, nil); return err },
		func() error { _, err := s.GenerateOAuth2AuthorizationCode(ctx, "client", "user", "", nil); return err },
		func() error {
			_, err := s.GenerateOAuth2AuthorizationCodeWithPKCE(ctx, "client", "user", "", nil, "", "")
			return err
		},
		func() error { _, err := s.ExchangeOAuth2CodeForToken(ctx, "code", "client", "", ""); return err },
		func() error {
			_, err := s.ExchangeOAuth2CodeForTokenWithPKCE(ctx, "code", "client", "", "", "")
			return err
		},
		func() error { _, err := s.OAuth2ClientCredentialsToken(ctx, "client", "", nil); return err },
		func() error { _, err := s.OAuth2PasswordGrantToken(ctx, "client", "", "", "", nil, nil); return err },
		func() error { _, err := s.RefreshOAuth2AccessToken(ctx, "client", "refresh", ""); return err },
		func() error { _, err := s.ValidateOAuth2AccessTokenAndGetInfo(ctx, "token"); return err },
		func() error { return s.RevokeOAuth2Token(ctx, "token") },
	}
	for i, check := range checks {
		if err := check(); !errors.Is(err, derror.ErrModuleNotEnabled) {
			t.Errorf("operation %d=%v", i, err)
		}
	}
	if s.ValidateOAuth2AccessToken(ctx, "token") {
		t.Fatal("disabled validation succeeded")
	}
}
