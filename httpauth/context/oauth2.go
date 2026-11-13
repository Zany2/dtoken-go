// @Author daixk 2026/06/05
package context

import (
	"context"

	"github.com/Zany2/dtoken-go/contract/oauth2"
	derror "github.com/Zany2/dtoken-go/errors"
)

// RegisterClient registers OAuth2 client RegisterClient 注册 OAuth2 客户端
func (c *OAuth2Context) RegisterClient(client *oauth2.Client) error {
	service := c.d.oauth2Service()
	if service == nil {
		return derror.ErrModuleNotEnabled
	}
	return service.RegisterOAuth2Client(client)
}

// UnregisterClient unregisters OAuth2 client UnregisterClient 注销 OAuth2 客户端
func (c *OAuth2Context) UnregisterClient(clientID string) error {
	service := c.d.oauth2Service()
	if service == nil {
		return derror.ErrModuleNotEnabled
	}
	return service.UnregisterOAuth2Client(clientID)
}

// GetClient gets OAuth2 client GetClient 获取 OAuth2 客户端
func (c *OAuth2Context) GetClient(clientID string) (*oauth2.Client, error) {
	service := c.d.oauth2Service()
	if service == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	return service.GetOAuth2Client(clientID)
}

// Token dispatches OAuth2 token request Token 处理 OAuth2 令牌请求
func (c *OAuth2Context) Token(ctx context.Context, req *oauth2.TokenRequest, validateUser oauth2.UserValidator) (*oauth2.AccessToken, error) {
	service := c.d.oauth2Service()
	if service == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	return service.OAuth2Token(ctx, req, validateUser)
}

// GenerateAuthorizationCode generates auth code GenerateAuthorizationCode 生成授权码
func (c *OAuth2Context) GenerateAuthorizationCode(ctx context.Context, clientID, userID, redirectURI string, scopes []string) (*oauth2.AuthorizationCode, error) {
	service := c.d.oauth2Service()
	if service == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	return service.GenerateOAuth2AuthorizationCode(ctx, clientID, userID, redirectURI, scopes)
}

// GenerateAuthorizationCodeWithPKCE generates auth code with PKCE GenerateAuthorizationCodeWithPKCE 使用 PKCE 生成授权码
func (c *OAuth2Context) GenerateAuthorizationCodeWithPKCE(ctx context.Context, clientID, userID, redirectURI string, scopes []string, codeChallenge, codeChallengeMethod string) (*oauth2.AuthorizationCode, error) {
	service := c.d.oauth2Service()
	if service == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	return service.GenerateOAuth2AuthorizationCodeWithPKCE(ctx, clientID, userID, redirectURI, scopes, codeChallenge, codeChallengeMethod)
}

// ExchangeCodeForToken exchanges auth code for token ExchangeCodeForToken 使用授权码换取 Token
func (c *OAuth2Context) ExchangeCodeForToken(ctx context.Context, code, clientID, clientSecret, redirectURI string) (*oauth2.AccessToken, error) {
	service := c.d.oauth2Service()
	if service == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	return service.ExchangeOAuth2CodeForToken(ctx, code, clientID, clientSecret, redirectURI)
}

// ExchangeCodeForTokenWithPKCE exchanges auth code for token with PKCE ExchangeCodeForTokenWithPKCE 使用 PKCE 授权码换取 Token
func (c *OAuth2Context) ExchangeCodeForTokenWithPKCE(ctx context.Context, code, clientID, clientSecret, redirectURI, codeVerifier string) (*oauth2.AccessToken, error) {
	service := c.d.oauth2Service()
	if service == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	return service.ExchangeOAuth2CodeForTokenWithPKCE(ctx, code, clientID, clientSecret, redirectURI, codeVerifier)
}

// ClientCredentialsToken gets token by client credentials ClientCredentialsToken 使用客户端凭证获取 Token
func (c *OAuth2Context) ClientCredentialsToken(ctx context.Context, clientID, clientSecret string, scopes []string) (*oauth2.AccessToken, error) {
	service := c.d.oauth2Service()
	if service == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	return service.OAuth2ClientCredentialsToken(ctx, clientID, clientSecret, scopes)
}

// PasswordGrantToken gets token by password grant PasswordGrantToken 使用密码模式获取 Token
func (c *OAuth2Context) PasswordGrantToken(ctx context.Context, clientID, clientSecret, username, password string, scopes []string, validateUser oauth2.UserValidator) (*oauth2.AccessToken, error) {
	service := c.d.oauth2Service()
	if service == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	return service.OAuth2PasswordGrantToken(ctx, clientID, clientSecret, username, password, scopes, validateUser)
}

// RefreshAccessToken refreshes OAuth2 access token RefreshAccessToken 刷新 OAuth2 访问令牌
func (c *OAuth2Context) RefreshAccessToken(ctx context.Context, clientID, refreshToken, clientSecret string) (*oauth2.AccessToken, error) {
	service := c.d.oauth2Service()
	if service == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	return service.RefreshOAuth2AccessToken(ctx, clientID, refreshToken, clientSecret)
}

// ValidateAccessToken validates OAuth2 access token ValidateAccessToken 校验 OAuth2 访问令牌
func (c *OAuth2Context) ValidateAccessToken(ctx context.Context, accessToken string) bool {
	service := c.d.oauth2Service()
	if service == nil {
		return false
	}
	return service.ValidateOAuth2AccessToken(ctx, accessToken)
}

// ValidateAccessTokenAndGetInfo validates OAuth2 access token and gets info ValidateAccessTokenAndGetInfo 校验 OAuth2 访问令牌并获取信息
func (c *OAuth2Context) ValidateAccessTokenAndGetInfo(ctx context.Context, accessToken string) (*oauth2.AccessToken, error) {
	service := c.d.oauth2Service()
	if service == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	return service.ValidateOAuth2AccessTokenAndGetInfo(ctx, accessToken)
}

// RevokeToken revokes OAuth2 token RevokeToken 撤销 OAuth2 令牌
func (c *OAuth2Context) RevokeToken(ctx context.Context, accessToken string) error {
	service := c.d.oauth2Service()
	if service == nil {
		return derror.ErrModuleNotEnabled
	}
	return service.RevokeOAuth2Token(ctx, accessToken)
}
