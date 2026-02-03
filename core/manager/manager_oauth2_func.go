// @Author daixk 2026/2/3 16:14:00
package manager

import (
	"context"

	"github.com/Zany2/dtoken-go/core/oauth2"
)

// ============================================================================
// OAuth2 Management - OAuth2 管理
// ============================================================================

// RegisterOAuth2Client registers an OAuth2 client.
// RegisterOAuth2Client 注册 OAuth2 客户端。
func (m *Manager) RegisterOAuth2Client(client *oauth2.Client) error {
	return m.oauth2Manager.RegisterClient(client)
}

// UnregisterOAuth2Client unregisters an OAuth2 client.
// UnregisterOAuth2Client 注销 OAuth2 客户端。
func (m *Manager) UnregisterOAuth2Client(clientID string) {
	m.oauth2Manager.UnregisterClient(clientID)
}

// GetOAuth2Client gets an OAuth2 client by ID.
// GetOAuth2Client 根据 ID 获取 OAuth2 客户端。
func (m *Manager) GetOAuth2Client(clientID string) (*oauth2.Client, error) {
	return m.oauth2Manager.GetClient(clientID)
}

// OAuth2Token unified token endpoint that dispatches to appropriate handler based on grant type.
// OAuth2Token 统一的令牌端点，根据授权类型分发到相应的处理逻辑。
func (m *Manager) OAuth2Token(ctx context.Context, req *oauth2.TokenRequest, validateUser oauth2.UserValidator) (*oauth2.AccessToken, error) {
	return m.oauth2Manager.Token(ctx, req, validateUser)
}

// GenerateOAuth2AuthorizationCode generates an authorization code.
// GenerateOAuth2AuthorizationCode 生成授权码。
func (m *Manager) GenerateOAuth2AuthorizationCode(ctx context.Context, clientID, userID, redirectURI string, scopes []string) (*oauth2.AuthorizationCode, error) {
	return m.oauth2Manager.GenerateAuthorizationCode(ctx, clientID, userID, redirectURI, scopes)
}

// ExchangeOAuth2CodeForToken exchanges authorization code for access token.
// ExchangeOAuth2CodeForToken 用授权码换取访问令牌。
func (m *Manager) ExchangeOAuth2CodeForToken(ctx context.Context, code, clientID, clientSecret, redirectURI string) (*oauth2.AccessToken, error) {
	return m.oauth2Manager.ExchangeCodeForToken(ctx, code, clientID, clientSecret, redirectURI)
}

// OAuth2ClientCredentialsToken gets access token using client credentials grant.
// OAuth2ClientCredentialsToken 使用客户端凭证模式获取访问令牌。
func (m *Manager) OAuth2ClientCredentialsToken(ctx context.Context, clientID, clientSecret string, scopes []string) (*oauth2.AccessToken, error) {
	return m.oauth2Manager.ClientCredentialsToken(ctx, clientID, clientSecret, scopes)
}

// OAuth2PasswordGrantToken gets access token using resource owner password credentials grant.
// OAuth2PasswordGrantToken 使用密码模式获取访问令牌。
func (m *Manager) OAuth2PasswordGrantToken(ctx context.Context, clientID, clientSecret, username, password string, scopes []string, validateUser oauth2.UserValidator) (*oauth2.AccessToken, error) {
	return m.oauth2Manager.PasswordGrantToken(ctx, clientID, clientSecret, username, password, scopes, validateUser)
}

// RefreshOAuth2AccessToken refreshes access token using refresh token.
// RefreshOAuth2AccessToken 使用刷新令牌刷新访问令牌。
func (m *Manager) RefreshOAuth2AccessToken(ctx context.Context, clientID, refreshToken, clientSecret string) (*oauth2.AccessToken, error) {
	return m.oauth2Manager.RefreshAccessToken(ctx, clientID, refreshToken, clientSecret)
}

// ValidateOAuth2AccessToken validates an access token.
// ValidateOAuth2AccessToken 验证访问令牌。
func (m *Manager) ValidateOAuth2AccessToken(ctx context.Context, accessToken string) bool {
	return m.oauth2Manager.ValidateAccessToken(ctx, accessToken)
}

// ValidateOAuth2AccessTokenAndGetInfo validates access token and gets info.
// ValidateOAuth2AccessTokenAndGetInfo 验证访问令牌并获取信息。
func (m *Manager) ValidateOAuth2AccessTokenAndGetInfo(ctx context.Context, accessToken string) (*oauth2.AccessToken, error) {
	return m.oauth2Manager.ValidateAccessTokenAndGetInfo(ctx, accessToken)
}

// RevokeOAuth2Token revokes an access token and its refresh token.
// RevokeOAuth2Token 撤销访问令牌及其刷新令牌。
func (m *Manager) RevokeOAuth2Token(ctx context.Context, accessToken string) error {
	return m.oauth2Manager.RevokeToken(ctx, accessToken)
}
