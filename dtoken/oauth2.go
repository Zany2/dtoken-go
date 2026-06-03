// @Author daixk 2025/12/22 15:56:00
package dtoken

import (
	"context"

	"github.com/Zany2/dtoken-go/core/oauth2"
)

// RegisterOAuth2Client registers an OAuth2 client.
func RegisterOAuth2Client(client *oauth2.Client, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.RegisterOAuth2Client(client)
}

// UnregisterOAuth2Client unregisters an OAuth2 client.
func UnregisterOAuth2Client(clientID string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.UnregisterOAuth2Client(clientID)
}

// GetOAuth2Client gets an OAuth2 client by id.
func GetOAuth2Client(clientID string, authType ...string) (*oauth2.Client, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GetOAuth2Client(clientID)
}

// OAuth2Token handles OAuth2 token requests.
func OAuth2Token(ctx context.Context, req *oauth2.TokenRequest, validateUser oauth2.UserValidator, authType ...string) (*oauth2.AccessToken, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.OAuth2Token(ctx, req, validateUser)
}

// GenerateOAuth2AuthorizationCode generates an authorization code.
func GenerateOAuth2AuthorizationCode(ctx context.Context, clientID, userID, redirectURI string, scopes []string, authType ...string) (*oauth2.AuthorizationCode, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GenerateOAuth2AuthorizationCode(ctx, clientID, userID, redirectURI, scopes)
}

// GenerateOAuth2AuthorizationCodeWithPKCE generates an authorization code with PKCE.
func GenerateOAuth2AuthorizationCodeWithPKCE(ctx context.Context, clientID, userID, redirectURI string, scopes []string, codeChallenge, codeChallengeMethod string, authType ...string) (*oauth2.AuthorizationCode, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GenerateOAuth2AuthorizationCodeWithPKCE(ctx, clientID, userID, redirectURI, scopes, codeChallenge, codeChallengeMethod)
}

// ExchangeOAuth2CodeForToken exchanges an authorization code for an access token.
func ExchangeOAuth2CodeForToken(ctx context.Context, code, clientID, clientSecret, redirectURI string, authType ...string) (*oauth2.AccessToken, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.ExchangeOAuth2CodeForToken(ctx, code, clientID, clientSecret, redirectURI)
}

// ExchangeOAuth2CodeForTokenWithPKCE exchanges an authorization code with PKCE verifier.
func ExchangeOAuth2CodeForTokenWithPKCE(ctx context.Context, code, clientID, clientSecret, redirectURI, codeVerifier string, authType ...string) (*oauth2.AccessToken, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.ExchangeOAuth2CodeForTokenWithPKCE(ctx, code, clientID, clientSecret, redirectURI, codeVerifier)
}

// OAuth2ClientCredentialsToken gets an access token with client credentials grant.
func OAuth2ClientCredentialsToken(ctx context.Context, clientID, clientSecret string, scopes []string, authType ...string) (*oauth2.AccessToken, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.OAuth2ClientCredentialsToken(ctx, clientID, clientSecret, scopes)
}

// OAuth2PasswordGrantToken gets an access token with password grant.
func OAuth2PasswordGrantToken(ctx context.Context, clientID, clientSecret, username, password string, scopes []string, validateUser oauth2.UserValidator, authType ...string) (*oauth2.AccessToken, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.OAuth2PasswordGrantToken(ctx, clientID, clientSecret, username, password, scopes, validateUser)
}

// RefreshOAuth2AccessToken refreshes an access token with a refresh token.
func RefreshOAuth2AccessToken(ctx context.Context, clientID, refreshToken, clientSecret string, authType ...string) (*oauth2.AccessToken, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.RefreshOAuth2AccessToken(ctx, clientID, refreshToken, clientSecret)
}

// ValidateOAuth2AccessToken validates an access token.
func ValidateOAuth2AccessToken(ctx context.Context, accessToken string, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.ValidateOAuth2AccessToken(ctx, accessToken)
}

// ValidateOAuth2AccessTokenAndGetInfo validates an access token and returns its info.
func ValidateOAuth2AccessTokenAndGetInfo(ctx context.Context, accessToken string, authType ...string) (*oauth2.AccessToken, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.ValidateOAuth2AccessTokenAndGetInfo(ctx, accessToken)
}

// RevokeOAuth2Token revokes an access token and its refresh token.
func RevokeOAuth2Token(ctx context.Context, accessToken string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.RevokeOAuth2Token(ctx, accessToken)
}
