package oauth2

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/derror"
)

// TestOAuth2AuthorizationCodeFlow verifies authorization code exchange is single-use. TestOAuth2AuthorizationCodeFlow 验证授权码交换只能使用一次。
func TestOAuth2AuthorizationCodeFlow(t *testing.T) {
	ctx := context.Background()
	server := newOAuth2TestServer()
	client := oauth2TestClient()

	if err := server.RegisterClient(client); err != nil {
		t.Fatalf("RegisterClient() error = %v", err)
	}
	code, err := server.GenerateAuthorizationCode(ctx, client.ClientID, "user-1", client.RedirectURIs[0], []string{"read"})
	if err != nil {
		t.Fatalf("GenerateAuthorizationCode() error = %v", err)
	}

	token, err := server.ExchangeCodeForToken(ctx, code.Code, client.ClientID, client.ClientSecret, client.RedirectURIs[0])
	if err != nil {
		t.Fatalf("ExchangeCodeForToken() error = %v", err)
	}
	if token.Token == "" || token.RefreshToken == "" || token.UserID != "user-1" || token.ClientID != client.ClientID {
		t.Fatalf("AccessToken = %+v, want populated token for user/client", token)
	}
	if !server.ValidateAccessToken(ctx, token.Token) {
		t.Fatal("ValidateAccessToken() = false, want true")
	}

	if _, err = server.ExchangeCodeForToken(ctx, code.Code, client.ClientID, client.ClientSecret, client.RedirectURIs[0]); !errors.Is(err, derror.ErrAuthCodeUsed) {
		t.Fatalf("second ExchangeCodeForToken() error = %v, want ErrAuthCodeUsed", err)
	}
}

// TestOAuth2TokenEndpointDispatch verifies token endpoint grants and validation failures. TestOAuth2TokenEndpointDispatch 验证令牌端点分发和校验失败。
func TestOAuth2TokenEndpointDispatch(t *testing.T) {
	ctx := context.Background()
	server := newOAuth2TestServer()
	client := oauth2TestClient()
	if err := server.RegisterClient(client); err != nil {
		t.Fatalf("RegisterClient() error = %v", err)
	}

	clientToken, err := server.Token(ctx, &TokenRequest{
		GrantType:    GrantTypeClientCredentials,
		ClientID:     client.ClientID,
		ClientSecret: client.ClientSecret,
		Scopes:       []string{"read"},
	}, nil)
	if err != nil {
		t.Fatalf("Token(client_credentials) error = %v", err)
	}
	if clientToken.UserID != client.ClientID {
		t.Fatalf("client credentials UserID = %q, want client id", clientToken.UserID)
	}

	passwordToken, err := server.Token(ctx, &TokenRequest{
		GrantType:    GrantTypePassword,
		ClientID:     client.ClientID,
		ClientSecret: client.ClientSecret,
		Username:     "alice",
		Password:     "secret",
		Scopes:       []string{"write"},
	}, func(username, password string) (string, error) {
		if username == "alice" && password == "secret" {
			return "user-alice", nil
		}
		return "", derror.ErrInvalidUserCredentials
	})
	if err != nil {
		t.Fatalf("Token(password) error = %v", err)
	}
	if passwordToken.UserID != "user-alice" {
		t.Fatalf("password token UserID = %q, want user-alice", passwordToken.UserID)
	}

	if _, err = server.Token(ctx, &TokenRequest{
		GrantType:    GrantTypeClientCredentials,
		ClientID:     client.ClientID,
		ClientSecret: "wrong",
	}, nil); !errors.Is(err, derror.ErrInvalidClientCredentials) {
		t.Fatalf("Token(wrong secret) error = %v, want ErrInvalidClientCredentials", err)
	}
	if _, err = server.Token(ctx, &TokenRequest{
		GrantType:    GrantTypeClientCredentials,
		ClientID:     client.ClientID,
		ClientSecret: client.ClientSecret,
		Scopes:       []string{"admin"},
	}, nil); !errors.Is(err, derror.ErrInvalidScope) {
		t.Fatalf("Token(invalid scope) error = %v, want ErrInvalidScope", err)
	}
}

// TestOAuth2RefreshAndRevoke verifies refresh rotation and revoke cleanup. TestOAuth2RefreshAndRevoke 验证刷新轮换和撤销清理。
func TestOAuth2RefreshAndRevoke(t *testing.T) {
	ctx := context.Background()
	server := newOAuth2TestServer()
	client := oauth2TestClient()
	if err := server.RegisterClient(client); err != nil {
		t.Fatalf("RegisterClient() error = %v", err)
	}

	token, err := server.ClientCredentialsToken(ctx, client.ClientID, client.ClientSecret, []string{"read"})
	if err != nil {
		t.Fatalf("ClientCredentialsToken() error = %v", err)
	}
	refreshed, err := server.RefreshAccessToken(ctx, client.ClientID, token.RefreshToken, client.ClientSecret)
	if err != nil {
		t.Fatalf("RefreshAccessToken() error = %v", err)
	}
	if refreshed.Token == token.Token || refreshed.RefreshToken == token.RefreshToken {
		t.Fatal("RefreshAccessToken() reused old token values, want rotation")
	}
	if server.ValidateAccessToken(ctx, token.Token) {
		t.Fatal("old access token is still valid after refresh, want false")
	}
	if _, err = server.RefreshAccessToken(ctx, client.ClientID, token.RefreshToken, client.ClientSecret); !errors.Is(err, derror.ErrInvalidRefreshToken) {
		t.Fatalf("second RefreshAccessToken() error = %v, want ErrInvalidRefreshToken", err)
	}
	if err = server.RevokeToken(ctx, refreshed.Token); err != nil {
		t.Fatalf("RevokeToken() error = %v", err)
	}
	if server.ValidateAccessToken(ctx, refreshed.Token) {
		t.Fatal("ValidateAccessToken() after revoke = true, want false")
	}
	if _, err = server.ValidateAccessTokenAndGetInfo(ctx, refreshed.Token); !errors.Is(err, derror.ErrInvalidAccessToken) {
		t.Fatalf("ValidateAccessTokenAndGetInfo() error = %v, want ErrInvalidAccessToken", err)
	}
}

func newOAuth2TestServer() *OAuth2Server {
	cfg := &Config{
		CodeExpiration:    time.Minute,
		TokenExpiration:   time.Minute,
		RefreshExpiration: time.Hour,
	}
	return NewOAuth2ServerWithConfig("auth:", "dtoken:", newOAuth2TestStorage(), oauth2TestCodec{}, cfg)
}

func oauth2TestClient() *Client {
	return &Client{
		ClientID:     "client-1",
		ClientSecret: "secret",
		RedirectURIs: []string{
			"https://example.com/callback",
		},
		GrantTypes: []GrantType{
			GrantTypeAuthorizationCode,
			GrantTypeClientCredentials,
			GrantTypePassword,
			GrantTypeRefreshToken,
		},
		Scopes: []string{"read", "write"},
	}
}

type oauth2TestCodec struct{}

func (oauth2TestCodec) Name() string { return "json-test" }

func (oauth2TestCodec) Encode(v any) ([]byte, error) { return json.Marshal(v) }

func (oauth2TestCodec) Decode(data []byte, v any) error { return json.Unmarshal(data, v) }

type oauth2TestStorage struct {
	values  map[string]any
	expires map[string]time.Time
}

func newOAuth2TestStorage() *oauth2TestStorage {
	return &oauth2TestStorage{values: map[string]any{}, expires: map[string]time.Time{}}
}

func (s *oauth2TestStorage) Set(_ context.Context, key string, value any, expiration time.Duration) error {
	s.values[key] = value
	if expiration > 0 {
		s.expires[key] = time.Now().Add(expiration)
	} else {
		delete(s.expires, key)
	}
	return nil
}

func (s *oauth2TestStorage) Get(_ context.Context, key string) (any, error) {
	if s.isExpired(key) {
		return nil, nil
	}
	return s.values[key], nil
}

func (s *oauth2TestStorage) Delete(_ context.Context, keys ...string) error {
	for _, key := range keys {
		delete(s.values, key)
		delete(s.expires, key)
	}
	return nil
}

func (s *oauth2TestStorage) Exists(_ context.Context, key string) bool {
	if s.isExpired(key) {
		return false
	}
	_, ok := s.values[key]
	return ok
}

func (s *oauth2TestStorage) Expire(_ context.Context, key string, expiration time.Duration) error {
	if !s.Exists(context.Background(), key) {
		return derror.ErrInvalidToken
	}
	if expiration > 0 {
		s.expires[key] = time.Now().Add(expiration)
	} else {
		delete(s.expires, key)
	}
	return nil
}

func (s *oauth2TestStorage) TTL(_ context.Context, key string) (time.Duration, error) {
	if s.isExpired(key) {
		return adapter.TTLNotFound, nil
	}
	if _, ok := s.values[key]; !ok {
		return adapter.TTLNotFound, nil
	}
	expireAt, ok := s.expires[key]
	if !ok {
		return adapter.TTLNoExpire, nil
	}
	ttl := time.Until(expireAt)
	if ttl <= 0 {
		_ = s.Delete(context.Background(), key)
		return adapter.TTLNotFound, nil
	}
	return ttl, nil
}

func (s *oauth2TestStorage) Ping(context.Context) error { return nil }

func (s *oauth2TestStorage) isExpired(key string) bool {
	expireAt, ok := s.expires[key]
	if !ok || time.Now().Before(expireAt) {
		return false
	}
	_ = s.Delete(context.Background(), key)
	return true
}
