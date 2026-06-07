// @Author daixk 2026/06/05
package hertz

import (
	"github.com/Zany2/dtoken-go/core/oauth2"
	hertzapp "github.com/cloudwego/hertz/pkg/app"
)

// RegisterOAuth2ClientByContext delegates to DToken context RegisterOAuth2ClientByContext 转发到 DToken 上下文。
func RegisterOAuth2ClientByContext(ctx *hertzapp.RequestContext, client *oauth2.Client) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.OAuth2().RegisterClient(client)
}

// UnregisterOAuth2ClientByContext delegates to DToken context UnregisterOAuth2ClientByContext 转发到 DToken 上下文。
func UnregisterOAuth2ClientByContext(ctx *hertzapp.RequestContext, clientID string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.OAuth2().UnregisterClient(clientID)
}

// GetOAuth2ClientByContext delegates to DToken context GetOAuth2ClientByContext 转发到 DToken 上下文。
func GetOAuth2ClientByContext(ctx *hertzapp.RequestContext, clientID string) (*oauth2.Client, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.OAuth2().GetClient(clientID)
}

// OAuth2TokenByContext handles OAuth2 token request OAuth2TokenByContext  OAuth2 Token
func OAuth2TokenByContext(ctx *hertzapp.RequestContext, req *oauth2.TokenRequest, validateUser oauth2.UserValidator) (*oauth2.AccessToken, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.OAuth2().Token(requestContext(ctx), req, validateUser)
}

// GenerateOAuth2AuthorizationCodeByContext delegates to DToken context GenerateOAuth2AuthorizationCodeByContext 转发到 DToken 上下文。
func GenerateOAuth2AuthorizationCodeByContext(ctx *hertzapp.RequestContext, clientID, userID, redirectURI string, scopes []string) (*oauth2.AuthorizationCode, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.OAuth2().GenerateAuthorizationCode(requestContext(ctx), clientID, userID, redirectURI, scopes)
}

// GenerateOAuth2AuthorizationCodeWithPKCEByContext delegates to DToken context GenerateOAuth2AuthorizationCodeWithPKCEByContext 转发到 DToken 上下文。
func GenerateOAuth2AuthorizationCodeWithPKCEByContext(ctx *hertzapp.RequestContext, clientID, userID, redirectURI string, scopes []string, codeChallenge, codeChallengeMethod string) (*oauth2.AuthorizationCode, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.OAuth2().GenerateAuthorizationCodeWithPKCE(requestContext(ctx), clientID, userID, redirectURI, scopes, codeChallenge, codeChallengeMethod)
}

// ExchangeOAuth2CodeForTokenByContext exchanges OAuth2 code for token ExchangeOAuth2CodeForTokenByContext  OAuth2 ?Token
func ExchangeOAuth2CodeForTokenByContext(ctx *hertzapp.RequestContext, code, clientID, clientSecret, redirectURI string) (*oauth2.AccessToken, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.OAuth2().ExchangeCodeForToken(requestContext(ctx), code, clientID, clientSecret, redirectURI)
}

// ExchangeOAuth2CodeForTokenWithPKCEByContext exchanges OAuth2 code for token with PKCE ExchangeOAuth2CodeForTokenWithPKCEByContext  PKCE ?Token
func ExchangeOAuth2CodeForTokenWithPKCEByContext(ctx *hertzapp.RequestContext, code, clientID, clientSecret, redirectURI, codeVerifier string) (*oauth2.AccessToken, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.OAuth2().ExchangeCodeForTokenWithPKCE(requestContext(ctx), code, clientID, clientSecret, redirectURI, codeVerifier)
}

// OAuth2ClientCredentialsTokenByContext gets OAuth2 token by client credentials OAuth2ClientCredentialsTokenByContext ?OAuth2 Token
func OAuth2ClientCredentialsTokenByContext(ctx *hertzapp.RequestContext, clientID, clientSecret string, scopes []string) (*oauth2.AccessToken, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.OAuth2().ClientCredentialsToken(requestContext(ctx), clientID, clientSecret, scopes)
}

// OAuth2PasswordGrantTokenByContext gets OAuth2 token by password grant OAuth2PasswordGrantTokenByContext  OAuth2 Token
func OAuth2PasswordGrantTokenByContext(ctx *hertzapp.RequestContext, clientID, clientSecret, username, password string, scopes []string, validateUser oauth2.UserValidator) (*oauth2.AccessToken, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.OAuth2().PasswordGrantToken(requestContext(ctx), clientID, clientSecret, username, password, scopes, validateUser)
}

// RefreshOAuth2AccessTokenByContext refreshes OAuth2 access token RefreshOAuth2AccessTokenByContext  OAuth2  Token
func RefreshOAuth2AccessTokenByContext(ctx *hertzapp.RequestContext, clientID, refreshToken, clientSecret string) (*oauth2.AccessToken, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.OAuth2().RefreshAccessToken(requestContext(ctx), clientID, refreshToken, clientSecret)
}

// ValidateOAuth2AccessTokenByContext validates OAuth2 access token ValidateOAuth2AccessTokenByContext  OAuth2  Token
func ValidateOAuth2AccessTokenByContext(ctx *hertzapp.RequestContext, accessToken string) bool {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return false
	}
	return dCtx.OAuth2().ValidateAccessToken(requestContext(ctx), accessToken)
}

// ValidateOAuth2AccessTokenAndGetInfoByContext delegates to DToken context ValidateOAuth2AccessTokenAndGetInfoByContext 转发到 DToken 上下文。
func ValidateOAuth2AccessTokenAndGetInfoByContext(ctx *hertzapp.RequestContext, accessToken string) (*oauth2.AccessToken, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.OAuth2().ValidateAccessTokenAndGetInfo(requestContext(ctx), accessToken)
}

// RevokeOAuth2TokenByContext revokes OAuth2 token RevokeOAuth2TokenByContext  OAuth2 Token
func RevokeOAuth2TokenByContext(ctx *hertzapp.RequestContext, accessToken string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.OAuth2().RevokeToken(requestContext(ctx), accessToken)
}
