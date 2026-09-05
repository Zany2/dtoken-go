package manager

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/listener"
	"github.com/Zany2/dtoken-go/core/oauth2"
)

// TestManagerOAuth2ValidationUsesDecodedToken verifies both validation entry points reject mismatched payloads and report exact results. TestManagerOAuth2ValidationUsesDecodedToken 验证两个校验入口都会拒绝错位载荷并报告准确结果。
func TestManagerOAuth2ValidationUsesDecodedToken(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManagerWithOAuth2(t)
	client := managerOAuth2TestClient()
	if err := mgr.RegisterOAuth2Client(client); err != nil {
		t.Fatalf("RegisterOAuth2Client() error = %v", err)
	}
	token, err := mgr.OAuth2ClientCredentialsToken(ctx, client.ClientID, client.ClientSecret, []string{"read"})
	if err != nil {
		t.Fatalf("OAuth2ClientCredentialsToken() error = %v", err)
	}

	var events []*listener.EventData
	mgr.GetEventManager().RegisterFuncWithConfig(listener.EventOAuth2TokenValidate, func(data *listener.EventData) {
		copyData := *data
		events = append(events, &copyData)
	}, listener.ListenerConfig{Async: false})

	if !mgr.ValidateOAuth2AccessToken(ctx, token.Token) {
		t.Fatal("ValidateOAuth2AccessToken(valid) = false, want true")
	}
	if _, err = mgr.ValidateOAuth2AccessTokenAndGetInfo(ctx, token.Token); err != nil {
		t.Fatalf("ValidateOAuth2AccessTokenAndGetInfo(valid) error = %v", err)
	}

	mismatched := *token
	mismatched.Token = "other-access-token"
	encoded, err := mgr.GetSerializer().Encode(&mismatched)
	if err != nil {
		t.Fatalf("Encode(mismatched token) error = %v", err)
	}
	tokenKey := mgr.GetConfig().KeyPrefix + mgr.GetConfig().AuthType + oauth2.TokenKeySuffix + token.Token
	if err = mgr.GetStorage().Set(ctx, tokenKey, encoded, time.Minute); err != nil {
		t.Fatalf("Set(mismatched token) error = %v", err)
	}

	if mgr.ValidateOAuth2AccessToken(ctx, token.Token) {
		t.Fatal("ValidateOAuth2AccessToken(mismatched payload) = true, want false")
	}
	if _, err = mgr.ValidateOAuth2AccessTokenAndGetInfo(ctx, token.Token); !errors.Is(err, derror.ErrInvalidAccessToken) {
		t.Fatalf("ValidateOAuth2AccessTokenAndGetInfo(mismatched payload) error = %v, want ErrInvalidAccessToken", err)
	}
	if len(events) != 4 {
		t.Fatalf("OAuth2 validation events = %d, want 4", len(events))
	}
	for index, event := range events {
		wantResult := index < 2
		if event.Token != token.Token || event.Extra[listener.ExtraKeyAction] != listener.ActionValidate || event.Extra[listener.ExtraKeyResult] != wantResult {
			t.Fatalf("validation event[%d] = %+v, want token %q result %v", index, event, token.Token, wantResult)
		}
	}

	revokeEvents := 0
	mgr.GetEventManager().RegisterFuncWithConfig(listener.EventOAuth2TokenRevoke, func(*listener.EventData) {
		revokeEvents++
	}, listener.ListenerConfig{Async: false})
	if err = mgr.RevokeOAuth2Token(ctx, token.Token); !errors.Is(err, derror.ErrInvalidAccessToken) {
		t.Fatalf("RevokeOAuth2Token(mismatched payload) error = %v, want ErrInvalidAccessToken", err)
	}
	refreshKey := mgr.GetConfig().KeyPrefix + mgr.GetConfig().AuthType + oauth2.RefreshKeySuffix + token.RefreshToken
	if !mgr.GetStorage().Exists(ctx, tokenKey) || !mgr.GetStorage().Exists(ctx, refreshKey) {
		t.Fatal("RevokeOAuth2Token(mismatched payload) removed credential data")
	}
	if revokeEvents != 0 {
		t.Fatalf("RevokeOAuth2Token(mismatched payload) events = %d, want 0", revokeEvents)
	}
}

// TestManagerOAuth2RejectsMismatchedStoredIdentities verifies OAuth2 credentials stay bound to their storage keys. TestManagerOAuth2RejectsMismatchedStoredIdentities 验证 OAuth2 凭证始终绑定到其存储键。
func TestManagerOAuth2RejectsMismatchedStoredIdentities(t *testing.T) {
	t.Run("client", func(t *testing.T) {
		ctx := context.Background()
		mgr := newTestManagerWithOAuth2(t)
		client := managerOAuth2TestClient()
		if err := mgr.RegisterOAuth2Client(client); err != nil {
			t.Fatalf("RegisterOAuth2Client() error = %v", err)
		}

		mismatched := *client
		mismatched.ClientID = "other-client"
		encoded, err := mgr.GetSerializer().Encode(&mismatched)
		if err != nil {
			t.Fatalf("Encode(mismatched client) error = %v", err)
		}
		key := mgr.GetConfig().KeyPrefix + mgr.GetConfig().AuthType + oauth2.ClientKeySuffix + client.ClientID
		if err = mgr.GetStorage().Set(ctx, key, encoded, time.Minute); err != nil {
			t.Fatalf("Set(mismatched client) error = %v", err)
		}
		if _, err = mgr.GetOAuth2Client(client.ClientID); !errors.Is(err, derror.ErrClientNotFound) {
			t.Fatalf("GetOAuth2Client(mismatched payload) error = %v, want ErrClientNotFound", err)
		}
	})

	t.Run("authorization code", func(t *testing.T) {
		ctx := context.Background()
		mgr := newTestManagerWithOAuth2(t)
		client := managerOAuth2TestClient()
		if err := mgr.RegisterOAuth2Client(client); err != nil {
			t.Fatalf("RegisterOAuth2Client() error = %v", err)
		}
		code, err := mgr.GenerateOAuth2AuthorizationCode(ctx, client.ClientID, "code-user", client.RedirectURIs[0], []string{"read"})
		if err != nil {
			t.Fatalf("GenerateOAuth2AuthorizationCode() error = %v", err)
		}

		mismatched := *code
		mismatched.Code = "other-code"
		encoded, err := mgr.GetSerializer().Encode(&mismatched)
		if err != nil {
			t.Fatalf("Encode(mismatched code) error = %v", err)
		}
		key := mgr.GetConfig().KeyPrefix + mgr.GetConfig().AuthType + oauth2.CodeKeySuffix + code.Code
		if err = mgr.GetStorage().Set(ctx, key, encoded, time.Minute); err != nil {
			t.Fatalf("Set(mismatched code) error = %v", err)
		}
		if _, err = mgr.ExchangeOAuth2CodeForToken(ctx, code.Code, client.ClientID, client.ClientSecret, client.RedirectURIs[0]); !errors.Is(err, derror.ErrInvalidAuthCode) {
			t.Fatalf("ExchangeOAuth2CodeForToken(mismatched payload) error = %v, want ErrInvalidAuthCode", err)
		}
	})

	t.Run("refresh token", func(t *testing.T) {
		ctx := context.Background()
		mgr := newTestManagerWithOAuth2(t)
		client := managerOAuth2TestClient()
		if err := mgr.RegisterOAuth2Client(client); err != nil {
			t.Fatalf("RegisterOAuth2Client() error = %v", err)
		}
		token, err := mgr.OAuth2ClientCredentialsToken(ctx, client.ClientID, client.ClientSecret, []string{"read"})
		if err != nil {
			t.Fatalf("OAuth2ClientCredentialsToken() error = %v", err)
		}

		mismatched := *token
		mismatched.RefreshToken = "other-refresh-token"
		encoded, err := mgr.GetSerializer().Encode(&mismatched)
		if err != nil {
			t.Fatalf("Encode(mismatched refresh token) error = %v", err)
		}
		key := mgr.GetConfig().KeyPrefix + mgr.GetConfig().AuthType + oauth2.RefreshKeySuffix + token.RefreshToken
		if err = mgr.GetStorage().Set(ctx, key, encoded, time.Minute); err != nil {
			t.Fatalf("Set(mismatched refresh token) error = %v", err)
		}
		if _, err = mgr.RefreshOAuth2AccessToken(ctx, client.ClientID, token.RefreshToken, client.ClientSecret); !errors.Is(err, derror.ErrInvalidRefreshToken) {
			t.Fatalf("RefreshOAuth2AccessToken(mismatched payload) error = %v, want ErrInvalidRefreshToken", err)
		}
		accessKey := mgr.GetConfig().KeyPrefix + mgr.GetConfig().AuthType + oauth2.TokenKeySuffix + token.Token
		if !mgr.GetStorage().Exists(ctx, accessKey) {
			t.Fatal("RefreshOAuth2AccessToken(mismatched payload) removed the existing access token")
		}
	})
}
