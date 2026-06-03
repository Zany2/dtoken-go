// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"context"

	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/oauth2"
)

// RegisterOAuth2Client registers OAuth2 client.
func (m *Manager) RegisterOAuth2Client(client *oauth2.Client) error {
	if m.oauth2Manager == nil {
		return derror.ErrModuleNotEnabled
	}
	return m.oauth2Manager.RegisterClient(client)
}

// UnregisterOAuth2Client unregisters OAuth2 client.
func (m *Manager) UnregisterOAuth2Client(clientID string) error {
	if m.oauth2Manager == nil {
		return derror.ErrModuleNotEnabled
	}
	return m.oauth2Manager.UnregisterClient(clientID)
}

// GetOAuth2Client gets OAuth2 client.
func (m *Manager) GetOAuth2Client(clientID string) (*oauth2.Client, error) {
	if m.oauth2Manager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	return m.oauth2Manager.GetClient(clientID)
}

// OAuth2Token dispatches token request.
func (m *Manager) OAuth2Token(ctx context.Context, req *oauth2.TokenRequest, validateUser oauth2.UserValidator) (*oauth2.AccessToken, error) {
	if m.oauth2Manager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	return m.oauth2Manager.Token(ctx, req, validateUser)
}

// GenerateOAuth2AuthorizationCode generates auth code.
func (m *Manager) GenerateOAuth2AuthorizationCode(ctx context.Context, clientID, userID, redirectURI string, scopes []string) (*oauth2.AuthorizationCode, error) {
	if m.oauth2Manager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	return m.oauth2Manager.GenerateAuthorizationCode(ctx, clientID, userID, redirectURI, scopes)
}

// GenerateOAuth2AuthorizationCodeWithPKCE generates auth code with PKCE.
func (m *Manager) GenerateOAuth2AuthorizationCodeWithPKCE(ctx context.Context, clientID, userID, redirectURI string, scopes []string, codeChallenge, codeChallengeMethod string) (*oauth2.AuthorizationCode, error) {
	if m.oauth2Manager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	return m.oauth2Manager.GenerateAuthorizationCodeWithPKCE(ctx, clientID, userID, redirectURI, scopes, codeChallenge, codeChallengeMethod)
}

// ExchangeOAuth2CodeForToken exchanges code for token.
func (m *Manager) ExchangeOAuth2CodeForToken(ctx context.Context, code, clientID, clientSecret, redirectURI string) (*oauth2.AccessToken, error) {
	if m.oauth2Manager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	return m.oauth2Manager.ExchangeCodeForToken(ctx, code, clientID, clientSecret, redirectURI)
}

// ExchangeOAuth2CodeForTokenWithPKCE exchanges code for token with PKCE verifier.
func (m *Manager) ExchangeOAuth2CodeForTokenWithPKCE(ctx context.Context, code, clientID, clientSecret, redirectURI, codeVerifier string) (*oauth2.AccessToken, error) {
	if m.oauth2Manager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	return m.oauth2Manager.ExchangeCodeForTokenWithPKCE(ctx, code, clientID, clientSecret, redirectURI, codeVerifier)
}

// OAuth2ClientCredentialsToken gets token by client credentials.
func (m *Manager) OAuth2ClientCredentialsToken(ctx context.Context, clientID, clientSecret string, scopes []string) (*oauth2.AccessToken, error) {
	if m.oauth2Manager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	return m.oauth2Manager.ClientCredentialsToken(ctx, clientID, clientSecret, scopes)
}

// OAuth2PasswordGrantToken gets token by password grant.
func (m *Manager) OAuth2PasswordGrantToken(ctx context.Context, clientID, clientSecret, username, password string, scopes []string, validateUser oauth2.UserValidator) (*oauth2.AccessToken, error) {
	if m.oauth2Manager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	return m.oauth2Manager.PasswordGrantToken(ctx, clientID, clientSecret, username, password, scopes, validateUser)
}

// RefreshOAuth2AccessToken refreshes access token.
func (m *Manager) RefreshOAuth2AccessToken(ctx context.Context, clientID, refreshToken, clientSecret string) (*oauth2.AccessToken, error) {
	if m.oauth2Manager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	return m.oauth2Manager.RefreshAccessToken(ctx, clientID, refreshToken, clientSecret)
}

// ValidateOAuth2AccessToken validates access token.
func (m *Manager) ValidateOAuth2AccessToken(ctx context.Context, accessToken string) bool {
	if m.oauth2Manager == nil {
		return false
	}
	return m.oauth2Manager.ValidateAccessToken(ctx, accessToken)
}

// ValidateOAuth2AccessTokenAndGetInfo validates token and gets info.
func (m *Manager) ValidateOAuth2AccessTokenAndGetInfo(ctx context.Context, accessToken string) (*oauth2.AccessToken, error) {
	if m.oauth2Manager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	return m.oauth2Manager.ValidateAccessTokenAndGetInfo(ctx, accessToken)
}

// RevokeOAuth2Token revokes OAuth2 token.
func (m *Manager) RevokeOAuth2Token(ctx context.Context, accessToken string) error {
	if m.oauth2Manager == nil {
		return derror.ErrModuleNotEnabled
	}
	return m.oauth2Manager.RevokeToken(ctx, accessToken)
}
