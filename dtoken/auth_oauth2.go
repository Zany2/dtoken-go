// @Author daixk 2025/12/22 15:56:00
package dtoken

import (
	"context"

	"github.com/Zany2/dtoken-go/core/oauth2"
)

// OAuth2Token handles OAuth2 token request. OAuth2Token 处理 OAuth2 令牌请求。
func (a *Auth) OAuth2Token(ctx context.Context, req *oauth2.TokenRequest, validateUser oauth2.UserValidator) (*oauth2.AccessToken, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.OAuth2Token(ctx, req, validateUser)
}

// RegisterOAuth2Client registers an OAuth2 client. RegisterOAuth2Client 注册 OAuth2 客户端。
func (a *Auth) RegisterOAuth2Client(client *oauth2.Client) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.RegisterOAuth2Client(client)
}

// UnregisterOAuth2Client unregisters an OAuth2 client. UnregisterOAuth2Client 注销 OAuth2 客户端。
func (a *Auth) UnregisterOAuth2Client(clientID string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.UnregisterOAuth2Client(clientID)
}

// GetOAuth2Client gets an OAuth2 client by id. GetOAuth2Client 根据 ID 获取 OAuth2 客户端。
func (a *Auth) GetOAuth2Client(clientID string) (*oauth2.Client, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.GetOAuth2Client(clientID)
}

// GenerateOAuth2AuthorizationCode generates an authorization code. GenerateOAuth2AuthorizationCode 生成授权码。
func (a *Auth) GenerateOAuth2AuthorizationCode(ctx context.Context, clientID, userID, redirectURI string, scopes []string) (*oauth2.AuthorizationCode, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.GenerateOAuth2AuthorizationCode(ctx, clientID, userID, redirectURI, scopes)
}

// GenerateOAuth2AuthorizationCodeWithPKCE generates an authorization code with PKCE. GenerateOAuth2AuthorizationCodeWithPKCE 使用 PKCE 生成授权码。
func (a *Auth) GenerateOAuth2AuthorizationCodeWithPKCE(ctx context.Context, clientID, userID, redirectURI string, scopes []string, codeChallenge, codeChallengeMethod string) (*oauth2.AuthorizationCode, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.GenerateOAuth2AuthorizationCodeWithPKCE(ctx, clientID, userID, redirectURI, scopes, codeChallenge, codeChallengeMethod)
}

// ExchangeOAuth2CodeForToken exchanges authorization code for access token. ExchangeOAuth2CodeForToken 使用授权码换取访问令牌。
func (a *Auth) ExchangeOAuth2CodeForToken(ctx context.Context, code, clientID, clientSecret, redirectURI string) (*oauth2.AccessToken, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.ExchangeOAuth2CodeForToken(ctx, code, clientID, clientSecret, redirectURI)
}

// ExchangeOAuth2CodeForTokenWithPKCE exchanges authorization code with PKCE verifier. ExchangeOAuth2CodeForTokenWithPKCE 使用 PKCE 校验码换取访问令牌。
func (a *Auth) ExchangeOAuth2CodeForTokenWithPKCE(ctx context.Context, code, clientID, clientSecret, redirectURI, codeVerifier string) (*oauth2.AccessToken, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.ExchangeOAuth2CodeForTokenWithPKCE(ctx, code, clientID, clientSecret, redirectURI, codeVerifier)
}

// OAuth2ClientCredentialsToken gets an access token with client credentials grant. OAuth2ClientCredentialsToken 使用客户端凭证模式获取访问令牌。
func (a *Auth) OAuth2ClientCredentialsToken(ctx context.Context, clientID, clientSecret string, scopes []string) (*oauth2.AccessToken, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.OAuth2ClientCredentialsToken(ctx, clientID, clientSecret, scopes)
}

// OAuth2PasswordGrantToken gets an access token with password grant. OAuth2PasswordGrantToken 使用密码模式获取访问令牌。
func (a *Auth) OAuth2PasswordGrantToken(ctx context.Context, clientID, clientSecret, username, password string, scopes []string, validateUser oauth2.UserValidator) (*oauth2.AccessToken, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.OAuth2PasswordGrantToken(ctx, clientID, clientSecret, username, password, scopes, validateUser)
}

// RefreshOAuth2AccessToken refreshes an access token with a refresh token. RefreshOAuth2AccessToken 使用刷新令牌刷新访问令牌。
func (a *Auth) RefreshOAuth2AccessToken(ctx context.Context, clientID, refreshToken, clientSecret string) (*oauth2.AccessToken, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.RefreshOAuth2AccessToken(ctx, clientID, refreshToken, clientSecret)
}

// ValidateOAuth2AccessToken validates an access token. ValidateOAuth2AccessToken 校验访问令牌。
func (a *Auth) ValidateOAuth2AccessToken(ctx context.Context, accessToken string) bool {
	mgr, err := a.requireManager()
	return err == nil && mgr.ValidateOAuth2AccessToken(ctx, accessToken)
}

// ValidateOAuth2AccessTokenAndGetInfo validates an access token and returns its info. ValidateOAuth2AccessTokenAndGetInfo 校验访问令牌并返回信息。
func (a *Auth) ValidateOAuth2AccessTokenAndGetInfo(ctx context.Context, accessToken string) (*oauth2.AccessToken, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.ValidateOAuth2AccessTokenAndGetInfo(ctx, accessToken)
}

// RevokeOAuth2Token revokes an access token and its refresh token. RevokeOAuth2Token 撤销访问令牌及其刷新令牌。
func (a *Auth) RevokeOAuth2Token(ctx context.Context, accessToken string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.RevokeOAuth2Token(ctx, accessToken)
}
