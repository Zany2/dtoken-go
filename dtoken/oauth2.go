// @Author daixk 2025/12/22 15:56:00
package dtoken

import (
	"context"

	"github.com/Zany2/dtoken-go/core/oauth2"
)

// RegisterOAuth2Client registers an OAuth2 client. RegisterOAuth2Client 注册 OAuth2 客户端。
func RegisterOAuth2Client(client *oauth2.Client, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.RegisterOAuth2Client(client)
}

// UnregisterOAuth2Client unregisters an OAuth2 client. UnregisterOAuth2Client 注销 OAuth2 客户端。
func UnregisterOAuth2Client(clientID string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	mgr.UnregisterOAuth2Client(clientID)
	return nil
}

// GetOAuth2Client gets an OAuth2 client by id. GetOAuth2Client 根据 ID 获取 OAuth2 客户端。
func GetOAuth2Client(clientID string, authType ...string) (*oauth2.Client, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GetOAuth2Client(clientID)
}

// OAuth2Token handles OAuth2 token requests. OAuth2Token 处理 OAuth2 令牌请求。
func OAuth2Token(ctx context.Context, req *oauth2.TokenRequest, validateUser oauth2.UserValidator, authType ...string) (*oauth2.AccessToken, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.OAuth2Token(ctx, req, validateUser)
}

// GenerateOAuth2AuthorizationCode generates an authorization code. GenerateOAuth2AuthorizationCode 生成 OAuth2 授权码。
func GenerateOAuth2AuthorizationCode(ctx context.Context, clientID, userID, redirectURI string, scopes []string, authType ...string) (*oauth2.AuthorizationCode, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GenerateOAuth2AuthorizationCode(ctx, clientID, userID, redirectURI, scopes)
}

// ExchangeOAuth2CodeForToken exchanges an authorization code for an access token. ExchangeOAuth2CodeForToken 使用授权码换取访问令牌。
func ExchangeOAuth2CodeForToken(ctx context.Context, code, clientID, clientSecret, redirectURI string, authType ...string) (*oauth2.AccessToken, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.ExchangeOAuth2CodeForToken(ctx, code, clientID, clientSecret, redirectURI)
}

// OAuth2ClientCredentialsToken gets an access token with client credentials grant. OAuth2ClientCredentialsToken 使用客户端凭证模式获取访问令牌。
func OAuth2ClientCredentialsToken(ctx context.Context, clientID, clientSecret string, scopes []string, authType ...string) (*oauth2.AccessToken, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.OAuth2ClientCredentialsToken(ctx, clientID, clientSecret, scopes)
}

// OAuth2PasswordGrantToken gets an access token with password grant. OAuth2PasswordGrantToken 使用密码模式获取访问令牌。
func OAuth2PasswordGrantToken(ctx context.Context, clientID, clientSecret, username, password string, scopes []string, validateUser oauth2.UserValidator, authType ...string) (*oauth2.AccessToken, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.OAuth2PasswordGrantToken(ctx, clientID, clientSecret, username, password, scopes, validateUser)
}

// RefreshOAuth2AccessToken refreshes an access token with a refresh token. RefreshOAuth2AccessToken 使用刷新令牌刷新访问令牌。
func RefreshOAuth2AccessToken(ctx context.Context, clientID, refreshToken, clientSecret string, authType ...string) (*oauth2.AccessToken, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.RefreshOAuth2AccessToken(ctx, clientID, refreshToken, clientSecret)
}

// ValidateOAuth2AccessToken validates an access token. ValidateOAuth2AccessToken 验证访问令牌。
func ValidateOAuth2AccessToken(ctx context.Context, accessToken string, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.ValidateOAuth2AccessToken(ctx, accessToken)
}

// ValidateOAuth2AccessTokenAndGetInfo validates an access token and returns its info. ValidateOAuth2AccessTokenAndGetInfo 验证访问令牌并返回信息。
func ValidateOAuth2AccessTokenAndGetInfo(ctx context.Context, accessToken string, authType ...string) (*oauth2.AccessToken, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.ValidateOAuth2AccessTokenAndGetInfo(ctx, accessToken)
}

// RevokeOAuth2Token revokes an access token and its refresh token. RevokeOAuth2Token 撤销访问令牌及其刷新令牌。
func RevokeOAuth2Token(ctx context.Context, accessToken string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.RevokeOAuth2Token(ctx, accessToken)
}
