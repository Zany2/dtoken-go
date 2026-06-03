// @Author daixk 2025/12/22 15:56:00
package dtoken

import (
	"context"

	"github.com/Zany2/dtoken-go/core/oauth2"
)

// OAuth2Token handles OAuth2 token request.
func (a *Auth) OAuth2Token(ctx context.Context, req *oauth2.TokenRequest, validateUser oauth2.UserValidator) (*oauth2.AccessToken, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.OAuth2Token(ctx, req, validateUser)
}

// RegisterOAuth2Client registers an OAuth2 client.
func (a *Auth) RegisterOAuth2Client(client *oauth2.Client) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.RegisterOAuth2Client(client)
}

// GenerateOAuth2AuthorizationCode generates an authorization code.
func (a *Auth) GenerateOAuth2AuthorizationCode(ctx context.Context, clientID, userID, redirectURI string, scopes []string) (*oauth2.AuthorizationCode, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.GenerateOAuth2AuthorizationCode(ctx, clientID, userID, redirectURI, scopes)
}

// GenerateOAuth2AuthorizationCodeWithPKCE generates an authorization code with PKCE.
func (a *Auth) GenerateOAuth2AuthorizationCodeWithPKCE(ctx context.Context, clientID, userID, redirectURI string, scopes []string, codeChallenge, codeChallengeMethod string) (*oauth2.AuthorizationCode, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.GenerateOAuth2AuthorizationCodeWithPKCE(ctx, clientID, userID, redirectURI, scopes, codeChallenge, codeChallengeMethod)
}

// ExchangeOAuth2CodeForToken exchanges authorization code for access token.
func (a *Auth) ExchangeOAuth2CodeForToken(ctx context.Context, code, clientID, clientSecret, redirectURI string) (*oauth2.AccessToken, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.ExchangeOAuth2CodeForToken(ctx, code, clientID, clientSecret, redirectURI)
}

// ExchangeOAuth2CodeForTokenWithPKCE exchanges authorization code with PKCE verifier.
func (a *Auth) ExchangeOAuth2CodeForTokenWithPKCE(ctx context.Context, code, clientID, clientSecret, redirectURI, codeVerifier string) (*oauth2.AccessToken, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.ExchangeOAuth2CodeForTokenWithPKCE(ctx, code, clientID, clientSecret, redirectURI, codeVerifier)
}
