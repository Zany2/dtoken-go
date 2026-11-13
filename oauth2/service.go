// @Author daixk 2025/12/22 15:56:00
package oauth2

import (
	"context"

	capability "github.com/Zany2/dtoken-go/contract/oauth2"
	derror "github.com/Zany2/dtoken-go/errors"
	listener "github.com/Zany2/dtoken-go/event"
)

// RegisterOAuth2Client registers OAuth2 client. RegisterOAuth2Client 注册 OAuth2 客户端。
func (m *Service) RegisterOAuth2Client(client *Client) error {
	if m.oauth2Manager == nil {
		return derror.ErrModuleNotEnabled
	}
	err := m.oauth2Manager.RegisterClient(client)
	if err == nil && client != nil {
		listener.Emit(m.events, listener.EventOAuth2ClientRegister, "", "", "", "", map[string]any{
			listener.ExtraKeyAction:   listener.ActionRegister,
			listener.ExtraKeyClientID: client.ClientID,
			listener.ExtraKeyScopes:   client.Scopes,
		})
	}
	return err
}

// UnregisterOAuth2Client unregisters OAuth2 client. UnregisterOAuth2Client 注销 OAuth2 客户端。
func (m *Service) UnregisterOAuth2Client(clientID string) error {
	if m.oauth2Manager == nil {
		return derror.ErrModuleNotEnabled
	}
	client, getErr := m.oauth2Manager.GetClient(clientID)
	err := m.oauth2Manager.UnregisterClient(clientID)
	if err == nil && getErr == nil && client != nil {
		listener.Emit(m.events, listener.EventOAuth2ClientUnregister, "", "", "", "", map[string]any{
			listener.ExtraKeyAction:   listener.ActionUnregister,
			listener.ExtraKeyClientID: clientID,
		})
	}
	return err
}

// GetOAuth2Client gets OAuth2 client. GetOAuth2Client 获取 OAuth2 客户端。
func (m *Service) GetOAuth2Client(clientID string) (*Client, error) {
	if m.oauth2Manager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	return m.oauth2Manager.GetClient(clientID)
}

// OAuth2Token dispatches token request. OAuth2Token 分发 OAuth2 令牌请求。
func (m *Service) OAuth2Token(ctx context.Context, req *TokenRequest, validateUser UserValidator) (*AccessToken, error) {
	if m.oauth2Manager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	token, err := m.oauth2Manager.Token(ctx, req, validateUser)
	if err != nil {
		return nil, err
	}
	event := listener.EventOAuth2TokenIssue
	action := listener.ActionIssue
	grantType := ""
	if req != nil {
		grantType = string(req.GrantType)
		if req.GrantType == GrantTypeRefreshToken {
			event = listener.EventOAuth2TokenRefresh
			action = listener.ActionRefresh
		}
	}
	m.triggerOAuth2TokenEvent(event, token, action, grantType)
	return token, nil
}

// GenerateOAuth2AuthorizationCode generates auth code. GenerateOAuth2AuthorizationCode 生成 OAuth2 授权码。
func (m *Service) GenerateOAuth2AuthorizationCode(ctx context.Context, clientID, userID, redirectURI string, scopes []string) (*AuthorizationCode, error) {
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
func (m *Service) GenerateOAuth2AuthorizationCodeWithPKCE(ctx context.Context, clientID, userID, redirectURI string, scopes []string, codeChallenge, codeChallengeMethod string) (*AuthorizationCode, error) {
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
func (m *Service) ExchangeOAuth2CodeForToken(ctx context.Context, code, clientID, clientSecret, redirectURI string) (*AccessToken, error) {
	if m.oauth2Manager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	token, err := m.oauth2Manager.ExchangeCodeForToken(ctx, code, clientID, clientSecret, redirectURI)
	if err != nil {
		return nil, err
	}
	m.triggerOAuth2TokenEvent(listener.EventOAuth2TokenIssue, token, listener.ActionIssue, string(GrantTypeAuthorizationCode))
	return token, nil
}

// ExchangeOAuth2CodeForTokenWithPKCE exchanges code for token with PKCE verifier. ExchangeOAuth2CodeForTokenWithPKCE 使用 PKCE 校验码换取令牌。
func (m *Service) ExchangeOAuth2CodeForTokenWithPKCE(ctx context.Context, code, clientID, clientSecret, redirectURI, codeVerifier string) (*AccessToken, error) {
	if m.oauth2Manager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	token, err := m.oauth2Manager.ExchangeCodeForTokenWithPKCE(ctx, code, clientID, clientSecret, redirectURI, codeVerifier)
	if err != nil {
		return nil, err
	}
	m.triggerOAuth2TokenEvent(listener.EventOAuth2TokenIssue, token, listener.ActionIssue, string(GrantTypeAuthorizationCode))
	return token, nil
}

// OAuth2ClientCredentialsToken gets token by client credentials. OAuth2ClientCredentialsToken 使用客户端凭证获取令牌。
func (m *Service) OAuth2ClientCredentialsToken(ctx context.Context, clientID, clientSecret string, scopes []string) (*AccessToken, error) {
	if m.oauth2Manager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	token, err := m.oauth2Manager.ClientCredentialsToken(ctx, clientID, clientSecret, scopes)
	if err != nil {
		return nil, err
	}
	m.triggerOAuth2TokenEvent(listener.EventOAuth2TokenIssue, token, listener.ActionIssue, string(GrantTypeClientCredentials))
	return token, nil
}

// OAuth2PasswordGrantToken gets token by password grant. OAuth2PasswordGrantToken 使用密码模式获取令牌。
func (m *Service) OAuth2PasswordGrantToken(ctx context.Context, clientID, clientSecret, username, password string, scopes []string, validateUser UserValidator) (*AccessToken, error) {
	if m.oauth2Manager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	token, err := m.oauth2Manager.PasswordGrantToken(ctx, clientID, clientSecret, username, password, scopes, validateUser)
	if err != nil {
		return nil, err
	}
	m.triggerOAuth2TokenEvent(listener.EventOAuth2TokenIssue, token, listener.ActionIssue, string(GrantTypePassword))
	return token, nil
}

// RefreshOAuth2AccessToken refreshes access token. RefreshOAuth2AccessToken 刷新 OAuth2 访问令牌。
func (m *Service) RefreshOAuth2AccessToken(ctx context.Context, clientID, refreshToken, clientSecret string) (*AccessToken, error) {
	if m.oauth2Manager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	token, err := m.oauth2Manager.RefreshAccessToken(ctx, clientID, refreshToken, clientSecret)
	if err != nil {
		return nil, err
	}
	m.triggerOAuth2TokenEvent(listener.EventOAuth2TokenRefresh, token, listener.ActionRefresh, string(GrantTypeRefreshToken))
	return token, nil
}

// ValidateOAuth2AccessToken validates access token. ValidateOAuth2AccessToken 校验 OAuth2 访问令牌。
func (m *Service) ValidateOAuth2AccessToken(ctx context.Context, accessToken string) bool {
	if m.oauth2Manager == nil {
		return false
	}
	token, err := m.oauth2Manager.ValidateAccessTokenAndGetInfo(ctx, accessToken)
	ok := err == nil
	m.triggerOAuth2TokenValidateEvent(accessToken, token, ok)
	return ok
}

// ValidateOAuth2AccessTokenAndGetInfo validates token and gets info. ValidateOAuth2AccessTokenAndGetInfo 校验 OAuth2 访问令牌并获取信息。
func (m *Service) ValidateOAuth2AccessTokenAndGetInfo(ctx context.Context, accessToken string) (*AccessToken, error) {
	if m.oauth2Manager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	token, err := m.oauth2Manager.ValidateAccessTokenAndGetInfo(ctx, accessToken)
	m.triggerOAuth2TokenValidateEvent(accessToken, token, err == nil)
	return token, err
}

// RevokeOAuth2Token revokes OAuth2 token. RevokeOAuth2Token 撤销 OAuth2 令牌。
func (m *Service) RevokeOAuth2Token(ctx context.Context, accessToken string) error {
	if m.oauth2Manager == nil {
		return derror.ErrModuleNotEnabled
	}
	token, _ := m.oauth2Manager.ValidateAccessTokenAndGetInfo(ctx, accessToken)
	err := m.oauth2Manager.RevokeToken(ctx, accessToken)
	if err == nil {
		if token != nil {
			m.triggerOAuth2TokenEvent(listener.EventOAuth2TokenRevoke, token, listener.ActionRevoke, "")
		}
	}
	return err
}

// triggerOAuth2CodeEvent emits OAuth2 authorization code events. triggerOAuth2CodeEvent 触发 OAuth2 授权码事件。
func (m *Service) triggerOAuth2CodeEvent(code *AuthorizationCode) {
	if code == nil {
		return
	}
	listener.Emit(m.events, listener.EventOAuth2CodeGenerate, code.UserID, "", "", code.Code, map[string]any{
		listener.ExtraKeyAction:   listener.ActionCreate,
		listener.ExtraKeyClientID: code.ClientID,
		listener.ExtraKeyUserID:   code.UserID,
		listener.ExtraKeyScopes:   code.Scopes,
		listener.ExtraKeyTTL:      code.ExpiresIn,
	})
}

// triggerOAuth2TokenValidateEvent emits a complete validation result. triggerOAuth2TokenValidateEvent 触发完整的 OAuth2 令牌校验结果事件。
func (m *Service) triggerOAuth2TokenValidateEvent(accessToken string, token *AccessToken, result bool) {
	extra := map[string]any{
		listener.ExtraKeyAction: listener.ActionValidate,
		listener.ExtraKeyResult: result,
	}
	loginID := ""
	if token != nil {
		loginID = token.UserID
		extra[listener.ExtraKeyClientID] = token.ClientID
		extra[listener.ExtraKeyUserID] = token.UserID
		extra[listener.ExtraKeyScopes] = token.Scopes
		extra[listener.ExtraKeyTokenType] = token.TokenType
		extra[listener.ExtraKeyTTL] = token.ExpiresIn
		extra[listener.ExtraKeyRefreshToken] = token.RefreshToken
		extra[listener.ExtraKeyGrantType] = ""
	}
	listener.Emit(m.events, listener.EventOAuth2TokenValidate, loginID, "", "", accessToken, extra)
}

// triggerOAuth2TokenEvent emits OAuth2 token lifecycle events. triggerOAuth2TokenEvent 触发 OAuth2 令牌生命周期事件。
func (m *Service) triggerOAuth2TokenEvent(event listener.Event, token *AccessToken, action, grantType string) {
	if token == nil {
		return
	}
	listener.Emit(m.events, event, token.UserID, "", "", token.Token, map[string]any{
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

// Service coordinates capability operations and lifecycle events. Service 协调能力操作与生命周期事件。
type Service struct {
	oauth2Manager *OAuth2Server
	events        listener.Emitter
}

// NewService creates a capability service with an optional event emitter and borrows components without transferring ownership; callers must keep them available throughout its lifetime. NewService 创建可选接入事件发布器的能力服务，仅借用组件，不转移所有权；调用方须保证组件在服务使用期间可用。
func NewService(manager *OAuth2Server, events listener.Emitter) *Service {
	return &Service{oauth2Manager: manager, events: events}
}

// The bundled service satisfies the public contract. 内置服务实现公开契约。
var _ capability.Service = (*Service)(nil)
