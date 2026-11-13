// Package oauth2 defines capability contracts without runtime implementations. 包 oauth2 定义能力契约，不依赖运行时实现。
package oauth2

import (
	"context"
	"time"
)

// Client defines OAuth2 client configuration. Client 定义 OAuth2 客户端配置。
type Client struct {
	// Client ID 客户端ID
	ClientID string
	// Client secret 客户端密钥
	ClientSecret string
	// Allowed redirect URIs 允许的回调URI
	RedirectURIs []string
	// Allowed grant types 允许的授权类型
	GrantTypes []GrantType
	// Allowed scopes 允许的权限范围
	Scopes []string
}

// AuthorizationCode authorization code information 授权码信息
type AuthorizationCode struct {
	// Authorization code 授权码
	Code string
	// Client ID 客户端ID
	ClientID string
	// Redirect URI 回调URI
	RedirectURI string
	// User ID 用户ID
	UserID string
	// Requested scopes 请求的权限范围
	Scopes []string
	// PKCE code challenge PKCE 授权码挑战值
	CodeChallenge string
	// PKCE challenge method PKCE 授权码挑战方法
	CodeChallengeMethod string
	// Creation time 创建时间
	CreateTime int64
	// Expiration time in seconds 过期时间（秒）
	ExpiresIn int64
	// Precise deadline; zero preserves legacy timestamp validation. 精确截止时间，零值兼容旧时间戳校验。
	ExpiresAt time.Time
	// Whether used 是否已使用
	Used bool
}

// AccessToken access token information 访问令牌信息
type AccessToken struct {
	// Access token 访问令牌
	Token string
	// Token type (Bearer) 令牌类型（Bearer）
	TokenType string
	// Expiration time in seconds 过期时间（秒）
	ExpiresIn int64
	// Refresh token 刷新令牌
	RefreshToken string
	// Granted scopes 授予的权限范围
	Scopes []string
	// User ID 用户ID
	UserID string
	// Client ID 客户端ID
	ClientID string
}

// TokenRequest Unified token request structure 统一的令牌请求结构
type TokenRequest struct {
	// Required: grant type 必需：授权类型
	GrantType GrantType
	// Required: client ID 必需：客户端ID
	ClientID string
	// Required: client secret 必需：客户端密钥
	ClientSecret string
	// For authorization_code: authorization code 授权码模式：授权码
	Code string
	// For authorization_code: redirect URI 授权码模式：回调URI
	RedirectURI string
	// For authorization_code with PKCE: code verifier 授权码 PKCE 模式：校验码
	CodeVerifier string
	// For refresh_token: refresh token 刷新令牌模式：刷新令牌
	RefreshToken string
	// For password: username 密码模式：用户名
	Username string
	// For password: password 密码模式：密码
	Password string
	// Optional scopes; refresh requests may only narrow the original grant. 可选权限范围，刷新请求只能缩小原授权。
	Scopes []string
}

// UserValidator Function type for validating user credentials 验证用户凭证的函数类型
type UserValidator func(username, password string) (userID string, err error)

// GrantType defines oauth2 grant type GrantType 定义 OAuth2 授权类型
type GrantType string

const (
	// GrantTypeAuthorizationCode stores authorization code mode GrantTypeAuthorizationCode 存储授权码模式
	GrantTypeAuthorizationCode GrantType = "authorization_code"
	// GrantTypeRefreshToken stores refresh token mode GrantTypeRefreshToken 存储刷新令牌模式
	GrantTypeRefreshToken GrantType = "refresh_token"
	// GrantTypeClientCredentials stores client credentials mode GrantTypeClientCredentials 存储客户端凭证模式
	GrantTypeClientCredentials GrantType = "client_credentials"
	// GrantTypePassword stores password mode GrantTypePassword 存储密码模式
	GrantTypePassword GrantType = "password"
)

// Service is the request-facing capability contract; implementations retain resource ownership. Service 是面向请求的能力契约，资源所有权由具体实现约定。
type Service interface {
	// RegisterOAuth2Client registers OAuth2 client. RegisterOAuth2Client 注册 OAuth2 客户端。
	RegisterOAuth2Client(client *Client) error
	// UnregisterOAuth2Client unregisters OAuth2 client. UnregisterOAuth2Client 注销 OAuth2 客户端。
	UnregisterOAuth2Client(clientID string) error
	// GetOAuth2Client gets OAuth2 client. GetOAuth2Client 获取 OAuth2 客户端。
	GetOAuth2Client(clientID string) (*Client, error)
	// OAuth2Token dispatches token request. OAuth2Token 分发 OAuth2 令牌请求。
	OAuth2Token(ctx context.Context, req *TokenRequest, validateUser UserValidator) (*AccessToken, error)
	// GenerateOAuth2AuthorizationCode generates auth code. GenerateOAuth2AuthorizationCode 生成 OAuth2 授权码。
	GenerateOAuth2AuthorizationCode(ctx context.Context, clientID, userID, redirectURI string, scopes []string) (*AuthorizationCode, error)
	// GenerateOAuth2AuthorizationCodeWithPKCE generates auth code with PKCE. GenerateOAuth2AuthorizationCodeWithPKCE 使用 PKCE 生成 OAuth2 授权码。
	GenerateOAuth2AuthorizationCodeWithPKCE(ctx context.Context, clientID, userID, redirectURI string, scopes []string, codeChallenge, codeChallengeMethod string) (*AuthorizationCode, error)
	// ExchangeOAuth2CodeForToken exchanges code for token. ExchangeOAuth2CodeForToken 使用授权码换取令牌。
	ExchangeOAuth2CodeForToken(ctx context.Context, code, clientID, clientSecret, redirectURI string) (*AccessToken, error)
	// ExchangeOAuth2CodeForTokenWithPKCE exchanges code for token with PKCE verifier. ExchangeOAuth2CodeForTokenWithPKCE 使用 PKCE 校验码换取令牌。
	ExchangeOAuth2CodeForTokenWithPKCE(ctx context.Context, code, clientID, clientSecret, redirectURI, codeVerifier string) (*AccessToken, error)
	// OAuth2ClientCredentialsToken gets token by client credentials. OAuth2ClientCredentialsToken 使用客户端凭证获取令牌。
	OAuth2ClientCredentialsToken(ctx context.Context, clientID, clientSecret string, scopes []string) (*AccessToken, error)
	// OAuth2PasswordGrantToken gets token by password grant. OAuth2PasswordGrantToken 使用密码模式获取令牌。
	OAuth2PasswordGrantToken(ctx context.Context, clientID, clientSecret, username, password string, scopes []string, validateUser UserValidator) (*AccessToken, error)
	// RefreshOAuth2AccessToken refreshes access token. RefreshOAuth2AccessToken 刷新 OAuth2 访问令牌。
	RefreshOAuth2AccessToken(ctx context.Context, clientID, refreshToken, clientSecret string) (*AccessToken, error)
	// ValidateOAuth2AccessToken validates access token. ValidateOAuth2AccessToken 校验 OAuth2 访问令牌。
	ValidateOAuth2AccessToken(ctx context.Context, accessToken string) bool
	// ValidateOAuth2AccessTokenAndGetInfo validates token and gets info. ValidateOAuth2AccessTokenAndGetInfo 校验 OAuth2 访问令牌并获取信息。
	ValidateOAuth2AccessTokenAndGetInfo(ctx context.Context, accessToken string) (*AccessToken, error)
	// RevokeOAuth2Token revokes OAuth2 token. RevokeOAuth2Token 撤销 OAuth2 令牌。
	RevokeOAuth2Token(ctx context.Context, accessToken string) error
}
