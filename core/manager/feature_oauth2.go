// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"context"

	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/listener"
	"github.com/Zany2/dtoken-go/core/oauth2"
)

// RegisterOAuth2Client registers OAuth2 client. RegisterOAuth2Client 注册 OAuth2 客户端。
func (m *Manager) RegisterOAuth2Client(client *oauth2.Client) error {
	if m.oauth2Manager == nil {
		return derror.ErrModuleNotEnabled
	}
	err := m.oauth2Manager.RegisterClient(client)
	if err == nil && client != nil {
		m.triggerEvent(listener.EventOAuth2ClientRegister, "", "", "", "", map[string]any{
			listener.ExtraKeyAction:   listener.ActionRegister,
			listener.ExtraKeyClientID: client.ClientID,
			listener.ExtraKeyScopes:   client.Scopes,
		})
	}
	return err
}

// UnregisterOAuth2Client unregisters OAuth2 client. UnregisterOAuth2Client 注销 OAuth2 客户端。
func (m *Manager) UnregisterOAuth2Client(clientID string) error {
	if m.oauth2Manager == nil {
		return derror.ErrModuleNotEnabled
	}
	err := m.oauth2Manager.UnregisterClient(clientID)
	if err == nil {
		m.triggerEvent(listener.EventOAuth2ClientUnregister, "", "", "", "", map[string]any{
			listener.ExtraKeyAction:   listener.ActionUnregister,
			listener.ExtraKeyClientID: clientID,
		})
	}
	return err
}

// GetOAuth2Client gets OAuth2 client. GetOAuth2Client 获取 OAuth2 客户端。
func (m *Manager) GetOAuth2Client(clientID string) (*oauth2.Client, error) {
	if m.oauth2Manager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	return m.oauth2Manager.GetClient(clientID)
}

// OAuth2Token dispatches token request. OAuth2Token 分发 OAuth2 令牌请求。
func (m *Manager) OAuth2Token(ctx context.Context, req *oauth2.TokenRequest, validateUser oauth2.UserValidator) (*oauth2.AccessToken, error) {
	if m.oauth2Manager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	token, err := m.oauth2Manager.Token(ctx, req, validateUser)
	if err != nil {
		return nil, err
	}
	event := listener.EventOAuth2TokenIssue
	grantType := ""
	if req != nil {
		grantType = string(req.GrantType)
		if req.GrantType == oauth2.GrantTypeRefreshToken {
			event = listener.EventOAuth2TokenRefresh
		}
	}
	m.triggerOAuth2TokenEvent(event, token, listener.ActionIssue, grantType)
	return token, nil
}

// GenerateOAuth2AuthorizationCode generates auth code. GenerateOAuth2AuthorizationCode 生成 OAuth2 授权码。
func (m *Manager) GenerateOAuth2AuthorizationCode(ctx context.Context, clientID, userID, redirectURI string, scopes []string) (*oauth2.AuthorizationCode, error) {
	if m.oauth2Manager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	code, err := m.oauth2Manager.GenerateAuthorizationCode(ctx, clientID, userID, redirectURI, scopes)
	if err != nil {
		return nil, err
	}
	m.triggerOAuth2CodeEvent(code)
	return code, nil
}

// GenerateOAuth2AuthorizationCodeWithPKCE generates auth code with PKCE. GenerateOAuth2AuthorizationCodeWithPKCE 使用 PKCE 生成 OAuth2 授权码。
func (m *Manager) GenerateOAuth2AuthorizationCodeWithPKCE(ctx context.Context, clientID, userID, redirectURI string, scopes []string, codeChallenge, codeChallengeMethod string) (*oauth2.AuthorizationCode, error) {
	if m.oauth2Manager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	code, err := m.oauth2Manager.GenerateAuthorizationCodeWithPKCE(ctx, clientID, userID, redirectURI, scopes, codeChallenge, codeChallengeMethod)
	if err != nil {
		return nil, err
	}
	m.triggerOAuth2CodeEvent(code)
	return code, nil
}

// ExchangeOAuth2CodeForToken exchanges code for token. ExchangeOAuth2CodeForToken 使用授权码换取令牌。
func (m *Manager) ExchangeOAuth2CodeForToken(ctx context.Context, code, clientID, clientSecret, redirectURI string) (*oauth2.AccessToken, error) {
	if m.oauth2Manager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	token, err := m.oauth2Manager.ExchangeCodeForToken(ctx, code, clientID, clientSecret, redirectURI)
	if err != nil {
		return nil, err
	}
	m.triggerOAuth2TokenEvent(listener.EventOAuth2TokenIssue, token, listener.ActionIssue, string(oauth2.GrantTypeAuthorizationCode))
	return token, nil
}

// ExchangeOAuth2CodeForTokenWithPKCE exchanges code for token with PKCE verifier. ExchangeOAuth2CodeForTokenWithPKCE 使用 PKCE 校验码换取令牌。
func (m *Manager) ExchangeOAuth2CodeForTokenWithPKCE(ctx context.Context, code, clientID, clientSecret, redirectURI, codeVerifier string) (*oauth2.AccessToken, error) {
	if m.oauth2Manager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	token, err := m.oauth2Manager.ExchangeCodeForTokenWithPKCE(ctx, code, clientID, clientSecret, redirectURI, codeVerifier)
	if err != nil {
		return nil, err
	}
	m.triggerOAuth2TokenEvent(listener.EventOAuth2TokenIssue, token, listener.ActionIssue, string(oauth2.GrantTypeAuthorizationCode))
	return token, nil
}

// OAuth2ClientCredentialsToken gets token by client credentials. OAuth2ClientCredentialsToken 使用客户端凭证获取令牌。
func (m *Manager) OAuth2ClientCredentialsToken(ctx context.Context, clientID, clientSecret string, scopes []string) (*oauth2.AccessToken, error) {
	if m.oauth2Manager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	token, err := m.oauth2Manager.ClientCredentialsToken(ctx, clientID, clientSecret, scopes)
	if err != nil {
		return nil, err
	}
	m.triggerOAuth2TokenEvent(listener.EventOAuth2TokenIssue, token, listener.ActionIssue, string(oauth2.GrantTypeClientCredentials))
	return token, nil
}

// OAuth2PasswordGrantToken gets token by password grant. OAuth2PasswordGrantToken 使用密码模式获取令牌。
func (m *Manager) OAuth2PasswordGrantToken(ctx context.Context, clientID, clientSecret, username, password string, scopes []string, validateUser oauth2.UserValidator) (*oauth2.AccessToken, error) {
	if m.oauth2Manager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	token, err := m.oauth2Manager.PasswordGrantToken(ctx, clientID, clientSecret, username, password, scopes, validateUser)
	if err != nil {
		return nil, err
	}
	m.triggerOAuth2TokenEvent(listener.EventOAuth2TokenIssue, token, listener.ActionIssue, string(oauth2.GrantTypePassword))
	return token, nil
}

// RefreshOAuth2AccessToken refreshes access token. RefreshOAuth2AccessToken 刷新 OAuth2 访问令牌。
func (m *Manager) RefreshOAuth2AccessToken(ctx context.Context, clientID, refreshToken, clientSecret string) (*oauth2.AccessToken, error) {
	if m.oauth2Manager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	token, err := m.oauth2Manager.RefreshAccessToken(ctx, clientID, refreshToken, clientSecret)
	if err != nil {
		return nil, err
	}
	m.triggerOAuth2TokenEvent(listener.EventOAuth2TokenRefresh, token, listener.ActionRefresh, string(oauth2.GrantTypeRefreshToken))
	return token, nil
}

// ValidateOAuth2AccessToken validates access token. ValidateOAuth2AccessToken 校验 OAuth2 访问令牌。
func (m *Manager) ValidateOAuth2AccessToken(ctx context.Context, accessToken string) bool {
	if m.oauth2Manager == nil {
		return false
	}
	ok := m.oauth2Manager.ValidateAccessToken(ctx, accessToken)
	m.triggerEvent(listener.EventOAuth2TokenValidate, "", "", "", accessToken, map[string]any{
		listener.ExtraKeyAction: listener.ActionValidate,
		listener.ExtraKeyResult: ok,
	})
	return ok
}

// ValidateOAuth2AccessTokenAndGetInfo validates token and gets info. ValidateOAuth2AccessTokenAndGetInfo 校验 OAuth2 访问令牌并获取信息。
func (m *Manager) ValidateOAuth2AccessTokenAndGetInfo(ctx context.Context, accessToken string) (*oauth2.AccessToken, error) {
	if m.oauth2Manager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	token, err := m.oauth2Manager.ValidateAccessTokenAndGetInfo(ctx, accessToken)
	if token != nil {
		m.triggerOAuth2TokenEvent(listener.EventOAuth2TokenValidate, token, listener.ActionValidate, "")
	}
	return token, err
}

// RevokeOAuth2Token revokes OAuth2 token. RevokeOAuth2Token 撤销 OAuth2 令牌。
func (m *Manager) RevokeOAuth2Token(ctx context.Context, accessToken string) error {
	if m.oauth2Manager == nil {
		return derror.ErrModuleNotEnabled
	}
	token, _ := m.oauth2Manager.ValidateAccessTokenAndGetInfo(ctx, accessToken)
	err := m.oauth2Manager.RevokeToken(ctx, accessToken)
	if err == nil {
		if token != nil {
			m.triggerOAuth2TokenEvent(listener.EventOAuth2TokenRevoke, token, listener.ActionRevoke, "")
		} else {
			m.triggerEvent(listener.EventOAuth2TokenRevoke, "", "", "", accessToken, map[string]any{
				listener.ExtraKeyAction: listener.ActionRevoke,
			})
		}
	}
	return err
}

func (m *Manager) triggerOAuth2CodeEvent(code *oauth2.AuthorizationCode) {
	if code == nil {
		return
	}
	m.triggerEvent(listener.EventOAuth2CodeGenerate, code.UserID, "", "", code.Code, map[string]any{
		listener.ExtraKeyAction:   listener.ActionCreate,
		listener.ExtraKeyClientID: code.ClientID,
		listener.ExtraKeyUserID:   code.UserID,
		listener.ExtraKeyScopes:   code.Scopes,
		listener.ExtraKeyTTL:      code.ExpiresIn,
	})
}

func (m *Manager) triggerOAuth2TokenEvent(event listener.Event, token *oauth2.AccessToken, action, grantType string) {
	if token == nil {
		return
	}
	m.triggerEvent(event, token.UserID, "", "", token.Token, map[string]any{
		listener.ExtraKeyAction:       action,
		listener.ExtraKeyClientID:     token.ClientID,
		listener.ExtraKeyUserID:       token.UserID,
		listener.ExtraKeyScopes:       token.Scopes,
		listener.ExtraKeyTokenType:    token.TokenType,
		listener.ExtraKeyTTL:          token.ExpiresIn,
		listener.ExtraKeyRefreshToken: token.RefreshToken,
		listener.ExtraKeyGrantType:    grantType,
	})
}
