// @Author daixk 2026/06/05
package gf

import (
	"context"

	"github.com/Zany2/dtoken-go/core/oauth2"
)

// RegisterOAuth2ClientByCtx registers OAuth2 client RegisterOAuth2ClientByCtx 注册 OAuth2 客户端
func RegisterOAuth2ClientByCtx(ctx context.Context, client *oauth2.Client) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.OAuth2().RegisterClient(client)
}

// UnregisterOAuth2ClientByCtx unregisters OAuth2 client UnregisterOAuth2ClientByCtx 注销 OAuth2 客户端
func UnregisterOAuth2ClientByCtx(ctx context.Context, clientID string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.OAuth2().UnregisterClient(clientID)
}

// GetOAuth2ClientByCtx gets OAuth2 client GetOAuth2ClientByCtx 获取 OAuth2 客户端
func GetOAuth2ClientByCtx(ctx context.Context, clientID string) (*oauth2.Client, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.OAuth2().GetClient(clientID)
}

// OAuth2TokenByCtx handles OAuth2 token request OAuth2TokenByCtx 处理 OAuth2 Token 请求
func OAuth2TokenByCtx(ctx context.Context, req *oauth2.TokenRequest, validateUser oauth2.UserValidator) (*oauth2.AccessToken, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.OAuth2().Token(ctx, req, validateUser)
}

// GenerateOAuth2AuthorizationCodeByCtx creates OAuth2 authorization code GenerateOAuth2AuthorizationCodeByCtx 创建 OAuth2 授权码
func GenerateOAuth2AuthorizationCodeByCtx(ctx context.Context, clientID, userID, redirectURI string, scopes []string) (*oauth2.AuthorizationCode, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.OAuth2().GenerateAuthorizationCode(ctx, clientID, userID, redirectURI, scopes)
}

// GenerateOAuth2AuthorizationCodeWithPKCEByCtx creates OAuth2 authorization code with PKCE GenerateOAuth2AuthorizationCodeWithPKCEByCtx 使用 PKCE 创建 OAuth2 授权码
func GenerateOAuth2AuthorizationCodeWithPKCEByCtx(ctx context.Context, clientID, userID, redirectURI string, scopes []string, codeChallenge, codeChallengeMethod string) (*oauth2.AuthorizationCode, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.OAuth2().GenerateAuthorizationCodeWithPKCE(ctx, clientID, userID, redirectURI, scopes, codeChallenge, codeChallengeMethod)
}

// ExchangeOAuth2CodeForTokenByCtx exchanges OAuth2 code for token ExchangeOAuth2CodeForTokenByCtx 使用 OAuth2 授权码换取 Token
func ExchangeOAuth2CodeForTokenByCtx(ctx context.Context, code, clientID, clientSecret, redirectURI string) (*oauth2.AccessToken, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.OAuth2().ExchangeCodeForToken(ctx, code, clientID, clientSecret, redirectURI)
}

// ExchangeOAuth2CodeForTokenWithPKCEByCtx exchanges OAuth2 code for token with PKCE ExchangeOAuth2CodeForTokenWithPKCEByCtx 使用 PKCE 授权码换取 Token
func ExchangeOAuth2CodeForTokenWithPKCEByCtx(ctx context.Context, code, clientID, clientSecret, redirectURI, codeVerifier string) (*oauth2.AccessToken, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.OAuth2().ExchangeCodeForTokenWithPKCE(ctx, code, clientID, clientSecret, redirectURI, codeVerifier)
}

// OAuth2ClientCredentialsTokenByCtx gets OAuth2 token by client credentials OAuth2ClientCredentialsTokenByCtx 使用客户端凭证获取 OAuth2 Token
func OAuth2ClientCredentialsTokenByCtx(ctx context.Context, clientID, clientSecret string, scopes []string) (*oauth2.AccessToken, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.OAuth2().ClientCredentialsToken(ctx, clientID, clientSecret, scopes)
}

// OAuth2PasswordGrantTokenByCtx gets OAuth2 token by password grant OAuth2PasswordGrantTokenByCtx 使用密码模式获取 OAuth2 Token
func OAuth2PasswordGrantTokenByCtx(ctx context.Context, clientID, clientSecret, username, password string, scopes []string, validateUser oauth2.UserValidator) (*oauth2.AccessToken, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.OAuth2().PasswordGrantToken(ctx, clientID, clientSecret, username, password, scopes, validateUser)
}

// RefreshOAuth2AccessTokenByCtx refreshes OAuth2 access token RefreshOAuth2AccessTokenByCtx 刷新 OAuth2 访问 Token
func RefreshOAuth2AccessTokenByCtx(ctx context.Context, clientID, refreshToken, clientSecret string) (*oauth2.AccessToken, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.OAuth2().RefreshAccessToken(ctx, clientID, refreshToken, clientSecret)
}

// ValidateOAuth2AccessTokenByCtx validates OAuth2 access token ValidateOAuth2AccessTokenByCtx 校验 OAuth2 访问 Token
func ValidateOAuth2AccessTokenByCtx(ctx context.Context, accessToken string) bool {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return false
	}
	return dCtx.OAuth2().ValidateAccessToken(ctx, accessToken)
}

// ValidateOAuth2AccessTokenAndGetInfoByCtx validates OAuth2 access token and gets info ValidateOAuth2AccessTokenAndGetInfoByCtx 校验 OAuth2 访问 Token 并获取信息
func ValidateOAuth2AccessTokenAndGetInfoByCtx(ctx context.Context, accessToken string) (*oauth2.AccessToken, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.OAuth2().ValidateAccessTokenAndGetInfo(ctx, accessToken)
}

// RevokeOAuth2TokenByCtx revokes OAuth2 token RevokeOAuth2TokenByCtx 撤销 OAuth2 Token
func RevokeOAuth2TokenByCtx(ctx context.Context, accessToken string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.OAuth2().RevokeToken(ctx, accessToken)
}
