// @Author daixk 2026/06/05
package fiber

import (
	"github.com/Zany2/dtoken-go/core/oauth2"
	gofiber "github.com/gofiber/fiber/v2"
)

// RegisterOAuth2ClientByContext registers OAuth2 client RegisterOAuth2ClientByContext 注册 OAuth2 客户端
func RegisterOAuth2ClientByContext(c *gofiber.Ctx, client *oauth2.Client) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.OAuth2().RegisterClient(client)
}

// UnregisterOAuth2ClientByContext unregisters OAuth2 client UnregisterOAuth2ClientByContext 注销 OAuth2 客户端
func UnregisterOAuth2ClientByContext(c *gofiber.Ctx, clientID string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.OAuth2().UnregisterClient(clientID)
}

// GetOAuth2ClientByContext gets OAuth2 client GetOAuth2ClientByContext 获取 OAuth2 客户端
func GetOAuth2ClientByContext(c *gofiber.Ctx, clientID string) (*oauth2.Client, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.OAuth2().GetClient(clientID)
}

// OAuth2TokenByContext handles OAuth2 token request OAuth2TokenByContext 处理 OAuth2 Token 请求
func OAuth2TokenByContext(c *gofiber.Ctx, req *oauth2.TokenRequest, validateUser oauth2.UserValidator) (*oauth2.AccessToken, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.OAuth2().Token(requestContext(c), req, validateUser)
}

// GenerateOAuth2AuthorizationCodeByContext creates OAuth2 authorization code GenerateOAuth2AuthorizationCodeByContext 创建 OAuth2 授权码
func GenerateOAuth2AuthorizationCodeByContext(c *gofiber.Ctx, clientID, userID, redirectURI string, scopes []string) (*oauth2.AuthorizationCode, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.OAuth2().GenerateAuthorizationCode(requestContext(c), clientID, userID, redirectURI, scopes)
}

// GenerateOAuth2AuthorizationCodeWithPKCEByContext creates OAuth2 authorization code with PKCE GenerateOAuth2AuthorizationCodeWithPKCEByContext 使用 PKCE 创建 OAuth2 授权码
func GenerateOAuth2AuthorizationCodeWithPKCEByContext(c *gofiber.Ctx, clientID, userID, redirectURI string, scopes []string, codeChallenge, codeChallengeMethod string) (*oauth2.AuthorizationCode, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.OAuth2().GenerateAuthorizationCodeWithPKCE(requestContext(c), clientID, userID, redirectURI, scopes, codeChallenge, codeChallengeMethod)
}

// ExchangeOAuth2CodeForTokenByContext exchanges OAuth2 code for token ExchangeOAuth2CodeForTokenByContext 使用 OAuth2 授权码换取 Token
func ExchangeOAuth2CodeForTokenByContext(c *gofiber.Ctx, code, clientID, clientSecret, redirectURI string) (*oauth2.AccessToken, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.OAuth2().ExchangeCodeForToken(requestContext(c), code, clientID, clientSecret, redirectURI)
}

// ExchangeOAuth2CodeForTokenWithPKCEByContext exchanges OAuth2 code for token with PKCE ExchangeOAuth2CodeForTokenWithPKCEByContext 使用 PKCE 授权码换取 Token
func ExchangeOAuth2CodeForTokenWithPKCEByContext(c *gofiber.Ctx, code, clientID, clientSecret, redirectURI, codeVerifier string) (*oauth2.AccessToken, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.OAuth2().ExchangeCodeForTokenWithPKCE(requestContext(c), code, clientID, clientSecret, redirectURI, codeVerifier)
}

// OAuth2ClientCredentialsTokenByContext gets OAuth2 token by client credentials OAuth2ClientCredentialsTokenByContext 使用客户端凭证获取 OAuth2 Token
func OAuth2ClientCredentialsTokenByContext(c *gofiber.Ctx, clientID, clientSecret string, scopes []string) (*oauth2.AccessToken, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.OAuth2().ClientCredentialsToken(requestContext(c), clientID, clientSecret, scopes)
}

// OAuth2PasswordGrantTokenByContext gets OAuth2 token by password grant OAuth2PasswordGrantTokenByContext 使用密码模式获取 OAuth2 Token
func OAuth2PasswordGrantTokenByContext(c *gofiber.Ctx, clientID, clientSecret, username, password string, scopes []string, validateUser oauth2.UserValidator) (*oauth2.AccessToken, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.OAuth2().PasswordGrantToken(requestContext(c), clientID, clientSecret, username, password, scopes, validateUser)
}

// RefreshOAuth2AccessTokenByContext refreshes OAuth2 access token RefreshOAuth2AccessTokenByContext 刷新 OAuth2 访问 Token
func RefreshOAuth2AccessTokenByContext(c *gofiber.Ctx, clientID, refreshToken, clientSecret string) (*oauth2.AccessToken, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.OAuth2().RefreshAccessToken(requestContext(c), clientID, refreshToken, clientSecret)
}

// ValidateOAuth2AccessTokenByContext validates OAuth2 access token ValidateOAuth2AccessTokenByContext 校验 OAuth2 访问 Token
func ValidateOAuth2AccessTokenByContext(c *gofiber.Ctx, accessToken string) bool {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return false
	}
	return dCtx.OAuth2().ValidateAccessToken(requestContext(c), accessToken)
}

// ValidateOAuth2AccessTokenAndGetInfoByContext validates OAuth2 access token and gets info ValidateOAuth2AccessTokenAndGetInfoByContext 校验 OAuth2 访问 Token 并获取信息
func ValidateOAuth2AccessTokenAndGetInfoByContext(c *gofiber.Ctx, accessToken string) (*oauth2.AccessToken, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.OAuth2().ValidateAccessTokenAndGetInfo(requestContext(c), accessToken)
}

// RevokeOAuth2TokenByContext revokes OAuth2 token RevokeOAuth2TokenByContext 撤销 OAuth2 Token
func RevokeOAuth2TokenByContext(c *gofiber.Ctx, accessToken string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.OAuth2().RevokeToken(requestContext(c), accessToken)
}
