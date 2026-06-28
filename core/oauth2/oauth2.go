// @Author daixk 2025/12/22 15:56:00
package oauth2

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/utils"
)

// Config defines OAuth2 server config Config 定义 OAuth2 服务端配置
type Config struct {
	// CodeExpiration stores authorization code ttl CodeExpiration 存储授权码有效期
	CodeExpiration time.Duration
	// TokenExpiration stores access token ttl TokenExpiration 存储访问令牌有效期
	TokenExpiration time.Duration
	// RefreshExpiration stores refresh token ttl RefreshExpiration 存储刷新令牌有效期
	RefreshExpiration time.Duration
}

// DefaultConfig returns default OAuth2 config DefaultConfig 返回默认 OAuth2 配置
func DefaultConfig() *Config {
	return &Config{
		CodeExpiration:    DefaultCodeExpiration,
		TokenExpiration:   DefaultTokenExpiration,
		RefreshExpiration: DefaultRefreshTTL,
	}
}

// Validate validates OAuth2 config Validate 验证 OAuth2 配置
func (c *Config) Validate() error {
	if c == nil {
		return nil
	}
	if c.CodeExpiration <= 0 {
		return fmt.Errorf("OAuth2Config.CodeExpiration must be a positive duration")
	}
	if c.TokenExpiration <= 0 {
		return fmt.Errorf("OAuth2Config.TokenExpiration must be a positive duration")
	}
	if c.RefreshExpiration <= 0 {
		return fmt.Errorf("OAuth2Config.RefreshExpiration must be a positive duration")
	}
	if c.RefreshExpiration <= c.TokenExpiration {
		return fmt.Errorf("OAuth2Config.RefreshExpiration must be greater than OAuth2Config.TokenExpiration")
	}
	return nil
}

// Clone returns a deep copy of OAuth2 config Clone 克隆 OAuth2 配置
func (c *Config) Clone() *Config {
	if c == nil {
		return nil
	}
	copyCfg := *c
	return &copyCfg
}

// Client OAuth2 client configuration OAuth2客户端配置
type Client struct {
	ClientID     string      // Client ID 客户端ID
	ClientSecret string      // Client secret 客户端密钥
	RedirectURIs []string    // Allowed redirect URIs 允许的回调URI
	GrantTypes   []GrantType // Allowed grant types 允许的授权类型
	Scopes       []string    // Allowed scopes 允许的权限范围
}

// AuthorizationCode authorization code information 授权码信息
type AuthorizationCode struct {
	Code                string   // Authorization code 授权码
	ClientID            string   // Client ID 客户端ID
	RedirectURI         string   // Redirect URI 回调URI
	UserID              string   // User ID 用户ID
	Scopes              []string // Requested scopes 请求的权限范围
	CodeChallenge       string   // PKCE code challenge PKCE 授权码挑战值
	CodeChallengeMethod string   // PKCE challenge method PKCE 授权码挑战方法
	CreateTime          int64    // Creation time 创建时间
	ExpiresIn           int64    // Expiration time in seconds 过期时间（秒）
	Used                bool     // Whether used 是否已使用
}

// AccessToken access token information 访问令牌信息
type AccessToken struct {
	Token        string   // Access token 访问令牌
	TokenType    string   // Token type (Bearer) 令牌类型（Bearer）
	ExpiresIn    int64    // Expiration time in seconds 过期时间（秒）
	RefreshToken string   // Refresh token 刷新令牌
	Scopes       []string // Granted scopes 授予的权限范围
	UserID       string   // User ID 用户ID
	ClientID     string   // Client ID 客户端ID
}

// TokenRequest Unified token request structure 统一的令牌请求结构
type TokenRequest struct {
	GrantType    GrantType // Required: grant type 必需：授权类型
	ClientID     string    // Required: client ID 必需：客户端ID
	ClientSecret string    // Required: client secret 必需：客户端密钥
	Code         string    // For authorization_code: authorization code 授权码模式：授权码
	RedirectURI  string    // For authorization_code: redirect URI 授权码模式：回调URI
	CodeVerifier string    // For authorization_code with PKCE: code verifier 授权码 PKCE 模式：校验码
	RefreshToken string    // For refresh_token: refresh token 刷新令牌模式：刷新令牌
	Username     string    // For password: username 密码模式：用户名
	Password     string    // For password: password 密码模式：密码
	Scopes       []string  // Optional: requested scopes 可选：请求的权限范围
}

// UserValidator Function type for validating user credentials 验证用户凭证的函数类型
type UserValidator func(username, password string) (userID string, err error)

// OAuth2Server OAuth2 authorization server OAuth2授权服务器
type OAuth2Server struct {
	authType          string          // Authentication system type 认证体系类型
	keyPrefix         string          // Configurable prefix 可配置的前缀
	codeExpiration    time.Duration   // Authorization code expiration (10min) 授权码过期时间（10分钟）
	tokenExpiration   time.Duration   // Access token expiration (2h) 访问令牌过期时间（2小时）
	refreshExpiration time.Duration   // Refresh token expiration 刷新令牌过期时间
	serializer        adapter.Codec   // Codec adapter for encoding and decoding operations 编解码器适配器
	storage           adapter.Storage // Storage adapter (Redis, Memory, etc.) 存储适配器（如 Redis、Memory）
}

// NewDefaultOAuth2Server creates OAuth2 server with default config NewDefaultOAuth2Server 使用默认配置创建 OAuth2 服务端
func NewDefaultOAuth2Server(authType, prefix string, storage adapter.Storage, serializer adapter.Codec) *OAuth2Server {
	return NewOAuth2ServerWithConfig(authType, prefix, storage, serializer, DefaultConfig())
}

// NewOAuth2ServerWithConfig creates OAuth2 server with config NewOAuth2ServerWithConfig 使用配置创建 OAuth2 服务端
func NewOAuth2ServerWithConfig(authType, prefix string, storage adapter.Storage, serializer adapter.Codec, cfg *Config) *OAuth2Server {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	codeExpiration := cfg.CodeExpiration
	if codeExpiration <= 0 {
		codeExpiration = DefaultCodeExpiration
	}
	tokenExpiration := cfg.TokenExpiration
	if tokenExpiration <= 0 {
		tokenExpiration = DefaultTokenExpiration
	}
	refreshExpiration := cfg.RefreshExpiration
	if refreshExpiration <= 0 {
		refreshExpiration = DefaultRefreshTTL
	}

	return &OAuth2Server{
		authType:          authType,
		keyPrefix:         prefix,
		codeExpiration:    codeExpiration,
		tokenExpiration:   tokenExpiration,
		refreshExpiration: refreshExpiration,
		storage:           storage,
		serializer:        serializer,
	}
}

// NewOAuth2Server Creates a new OAuth2 server 创建新的OAuth2服务器
func NewOAuth2Server(authType, prefix string, storage adapter.Storage, serializer adapter.Codec) *OAuth2Server {
	return NewDefaultOAuth2Server(authType, prefix, storage, serializer)
}

// RegisterClient Registers an OAuth2 client 注册OAuth2客户端
func (s *OAuth2Server) RegisterClient(client *Client) error {
	if client == nil || client.ClientID == "" {
		return derror.ErrClientOrClientIDEmpty
	}
	if client.ClientSecret == "" {
		return derror.ErrInvalidClientCredentials
	}
	return s.saveClient(context.Background(), client)
}

// UnregisterClient Unregisters an OAuth2 client 注销OAuth2客户端
func (s *OAuth2Server) UnregisterClient(clientID string) error {
	return s.deleteClient(context.Background(), clientID)
}

// GetClient Gets client by ID 根据ID获取客户端
func (s *OAuth2Server) GetClient(clientID string) (*Client, error) {
	return s.getClient(context.Background(), clientID)
}

// Token Unified token endpoint that dispatches to appropriate handler based on grant type 统一的令牌端点，根据授权类型分发到相应的处理逻辑
func (s *OAuth2Server) Token(ctx context.Context, req *TokenRequest, validateUser UserValidator) (*AccessToken, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: token request cannot be nil", derror.ErrInvalidAuthCode)
	}

	switch req.GrantType {
	case GrantTypeAuthorizationCode:
		return s.ExchangeCodeForTokenWithPKCE(ctx, req.Code, req.ClientID, req.ClientSecret, req.RedirectURI, req.CodeVerifier)

	case GrantTypeClientCredentials:
		return s.ClientCredentialsToken(ctx, req.ClientID, req.ClientSecret, req.Scopes)

	case GrantTypePassword:
		return s.PasswordGrantToken(ctx, req.ClientID, req.ClientSecret, req.Username, req.Password, req.Scopes, validateUser)

	case GrantTypeRefreshToken:
		return s.RefreshAccessToken(ctx, req.ClientID, req.RefreshToken, req.ClientSecret)

	default:
		return nil, derror.ErrInvalidGrantType
	}
}

// GenerateAuthorizationCode generates authorization code.
func (s *OAuth2Server) GenerateAuthorizationCode(ctx context.Context, clientID, userID, redirectURI string, scopes []string) (*AuthorizationCode, error) {
	return s.GenerateAuthorizationCodeWithPKCE(ctx, clientID, userID, redirectURI, scopes, "", "")
}

// GenerateAuthorizationCodeWithPKCE generates authorization code with optional PKCE challenge.
func (s *OAuth2Server) GenerateAuthorizationCodeWithPKCE(ctx context.Context, clientID, userID, redirectURI string, scopes []string, codeChallenge, codeChallengeMethod string) (*AuthorizationCode, error) {
	if clientID == "" {
		return nil, derror.ErrClientOrClientIDEmpty
	}
	if userID == "" {
		return nil, derror.ErrUserIDEmpty
	}

	client, err := s.getClient(ctx, clientID)
	if err != nil {
		return nil, err
	}

	if !s.isValidRedirectURI(client, redirectURI) {
		return nil, derror.ErrInvalidRedirectURI
	}

	if !s.isValidScopes(client, scopes) {
		return nil, derror.ErrInvalidScope
	}
	codeChallenge = strings.TrimSpace(codeChallenge)
	codeChallengeMethod, err = normalizeCodeChallengeMethod(codeChallenge, codeChallengeMethod)
	if err != nil {
		return nil, err
	}

	codeBytes := make([]byte, CodeLength)
	if _, err = rand.Read(codeBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random code: %w", err)
	}
	code := hex.EncodeToString(codeBytes)

	authCode := &AuthorizationCode{
		Code:                code,
		ClientID:            clientID,
		RedirectURI:         redirectURI,
		UserID:              userID,
		Scopes:              append([]string(nil), scopes...),
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: codeChallengeMethod,
		CreateTime:          time.Now().Unix(),
		ExpiresIn:           int64(s.codeExpiration.Seconds()),
		Used:                false,
	}

	encodeData, err := s.serializer.Encode(authCode)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}

	key := s.getCodeKey(code)
	if err := s.storage.Set(ctx, key, encodeData, s.codeExpiration); err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	return authCode, nil
}

// ExchangeCodeForToken exchanges authorization code for access token.
func (s *OAuth2Server) ExchangeCodeForToken(ctx context.Context, code, clientID, clientSecret, redirectURI string) (*AccessToken, error) {
	return s.ExchangeCodeForTokenWithPKCE(ctx, code, clientID, clientSecret, redirectURI, "")
}

// ExchangeCodeForTokenWithPKCE exchanges authorization code with optional PKCE verifier.
func (s *OAuth2Server) ExchangeCodeForTokenWithPKCE(ctx context.Context, code, clientID, clientSecret, redirectURI, codeVerifier string) (*AccessToken, error) {
	client, err := s.getClient(ctx, clientID)
	if err != nil {
		return nil, err
	}

	if client.ClientSecret != clientSecret {
		return nil, derror.ErrInvalidClientCredentials
	}

	if !s.isValidGrantType(client, GrantTypeAuthorizationCode) {
		return nil, derror.ErrInvalidGrantType
	}

	key := s.getCodeKey(code)

	// Atomically consume the authorization code to prevent concurrent replay 原子消费授权码，防止并发重放攻击
	var rawData []byte
	if atomicStorage, ok := s.storage.(adapter.AtomicStorage); ok {
		data, delErr := atomicStorage.GetAndDelete(ctx, key)
		if delErr != nil {
			return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, delErr)
		}
		if data == nil {
			return nil, derror.ErrInvalidAuthCode
		}
		rawData, err = utils.ToBytes(data)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", derror.ErrTypeConvert, err)
		}
	} else {
		// Non-atomic fallback: concurrent replay possible on storages without GetAndDelete 非原子回退：不支持 GetAndDelete 的存储无法防止并发重放
		data, getErr := s.storage.Get(ctx, key)
		if getErr != nil {
			return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, getErr)
		}
		if data == nil {
			return nil, derror.ErrInvalidAuthCode
		}
		rawData, err = utils.ToBytes(data)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", derror.ErrTypeConvert, err)
		}
		if err = s.storage.Delete(ctx, key); err != nil {
			return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
	}

	var authCode AuthorizationCode
	if err = s.serializer.Decode(rawData, &authCode); err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}

	if authCode.ClientID != clientID {
		return nil, derror.ErrClientMismatch
	}

	if authCode.RedirectURI != redirectURI {
		return nil, derror.ErrRedirectURIMismatch
	}

	if time.Now().Unix() > authCode.CreateTime+authCode.ExpiresIn {
		return nil, derror.ErrAuthCodeExpired
	}
	if err = verifyCodeChallenge(authCode.CodeChallenge, authCode.CodeChallengeMethod, codeVerifier); err != nil {
		return nil, err
	}

	return s.generateAccessToken(ctx, authCode.UserID, authCode.ClientID, authCode.Scopes)
}

// ClientCredentialsToken Gets access token using client credentials grant 使用客户端凭证模式获取访问令牌
func (s *OAuth2Server) ClientCredentialsToken(ctx context.Context, clientID, clientSecret string, scopes []string) (*AccessToken, error) {
	client, err := s.getClient(ctx, clientID)
	if err != nil {
		return nil, err
	}

	if client.ClientSecret != clientSecret {
		return nil, derror.ErrInvalidClientCredentials
	}

	if !s.isValidGrantType(client, GrantTypeClientCredentials) {
		return nil, derror.ErrInvalidGrantType
	}

	if !s.isValidScopes(client, scopes) {
		return nil, derror.ErrInvalidScope
	}

	return s.generateAccessToken(ctx, clientID, clientID, scopes)
}

// PasswordGrantToken Gets access token using resource owner password credentials grant 使用密码模式获取访问令牌
func (s *OAuth2Server) PasswordGrantToken(ctx context.Context, clientID, clientSecret, username, password string, scopes []string, validateUser UserValidator) (*AccessToken, error) {
	if validateUser == nil {
		return nil, fmt.Errorf("%w: user validator function is required", derror.ErrInvalidUserCredentials)
	}

	client, err := s.getClient(ctx, clientID)
	if err != nil {
		return nil, err
	}

	if client.ClientSecret != clientSecret {
		return nil, derror.ErrInvalidClientCredentials
	}

	if !s.isValidGrantType(client, GrantTypePassword) {
		return nil, derror.ErrInvalidGrantType
	}

	if !s.isValidScopes(client, scopes) {
		return nil, derror.ErrInvalidScope
	}

	userID, err := validateUser(username, password)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrInvalidUserCredentials, err)
	}

	if userID == "" {
		return nil, derror.ErrUserIDEmpty
	}

	return s.generateAccessToken(ctx, userID, clientID, scopes)
}

// RefreshAccessToken Refreshes access token using refresh token 使用刷新令牌刷新访问令牌
func (s *OAuth2Server) RefreshAccessToken(ctx context.Context, clientID, refreshToken, clientSecret string) (*AccessToken, error) {
	if refreshToken == "" {
		return nil, derror.ErrInvalidRefreshToken
	}

	client, err := s.getClient(ctx, clientID)
	if err != nil {
		return nil, err
	}

	if client.ClientSecret != clientSecret {
		return nil, derror.ErrInvalidClientCredentials
	}

	if !s.isValidGrantType(client, GrantTypeRefreshToken) {
		return nil, derror.ErrInvalidGrantType
	}

	key := s.getRefreshKey(refreshToken)
	data, err := s.storage.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	if data == nil {
		return nil, derror.ErrInvalidRefreshToken
	}

	rawData, err := utils.ToBytes(data)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrTypeConvert, err)
	}

	var accessTokenInfo AccessToken
	err = s.serializer.Decode(rawData, &accessTokenInfo)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}

	if accessTokenInfo.ClientID != clientID {
		return nil, derror.ErrClientMismatch
	}

	// Delete refresh token first to prevent concurrent replay 先删除刷新令牌，防止并发重放
	if err = s.storage.Delete(ctx, key); err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	token, err := s.generateAccessToken(ctx, accessTokenInfo.UserID, accessTokenInfo.ClientID, accessTokenInfo.Scopes)
	if err != nil {
		return nil, err
	}

	_ = s.storage.Delete(ctx, s.getTokenKey(accessTokenInfo.Token))

	return token, nil
}

// ValidateAccessToken Validates access token 验证访问令牌
func (s *OAuth2Server) ValidateAccessToken(ctx context.Context, accessToken string) bool {
	if accessToken == "" {
		return false
	}
	return s.storage.Exists(ctx, s.getTokenKey(accessToken))
}

// ValidateAccessTokenAndGetInfo Validates access token and get info 验证访问令牌并获取信息
func (s *OAuth2Server) ValidateAccessTokenAndGetInfo(ctx context.Context, accessToken string) (*AccessToken, error) {
	if accessToken == "" {
		return nil, derror.ErrInvalidAccessToken
	}

	key := s.getTokenKey(accessToken)
	data, err := s.storage.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	if data == nil {
		return nil, derror.ErrInvalidAccessToken
	}

	rawData, err := utils.ToBytes(data)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrTypeConvert, err)
	}

	var accessTokenInfo AccessToken
	err = s.serializer.Decode(rawData, &accessTokenInfo)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}

	return &accessTokenInfo, nil
}

// RevokeToken Revokes access token and its refresh token 撤销访问令牌及其刷新令牌
func (s *OAuth2Server) RevokeToken(ctx context.Context, accessToken string) error {
	if accessToken == "" {
		return derror.ErrInvalidAccessToken
	}

	key := s.getTokenKey(accessToken)
	data, err := s.storage.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	if data == nil {
		return derror.ErrInvalidAccessToken
	}

	rawData, err := utils.ToBytes(data)
	if err != nil {
		return fmt.Errorf("%w: %v", derror.ErrTypeConvert, err)
	}

	var accessTokenInfo AccessToken
	err = s.serializer.Decode(rawData, &accessTokenInfo)
	if err != nil {
		return fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}

	// Delete access token first (higher security priority) 优先删除访问令牌
	if err = s.storage.Delete(ctx, key); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	if accessTokenInfo.RefreshToken != "" {
		if err = s.storage.Delete(ctx, s.getRefreshKey(accessTokenInfo.RefreshToken)); err != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
	}

	return nil
}

// getCodeKey Gets storage key for authorization code 获取授权码的存储键
func (s *OAuth2Server) getCodeKey(code string) string {
	return s.keyPrefix + s.authType + CodeKeySuffix + code
}

// getTokenKey Gets storage key for access token 获取访问令牌的存储键
func (s *OAuth2Server) getTokenKey(token string) string {
	return s.keyPrefix + s.authType + TokenKeySuffix + token
}

// getRefreshKey Gets storage key for refresh token 获取刷新令牌的存储键
func (s *OAuth2Server) getRefreshKey(refreshToken string) string {
	return s.keyPrefix + s.authType + RefreshKeySuffix + refreshToken
}

// getClientKey gets storage key for OAuth2 client. getClientKey 获取 OAuth2 客户端存储键。
func (s *OAuth2Server) getClientKey(clientID string) string {
	return s.keyPrefix + s.authType + ClientKeySuffix + clientID
}

// saveClient saves OAuth2 client through shared storage. saveClient 通过共享存储保存 OAuth2 客户端。
func (s *OAuth2Server) saveClient(ctx context.Context, client *Client) error {
	encodeData, err := s.serializer.Encode(client)
	if err != nil {
		return fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}
	if err = s.storage.Set(ctx, s.getClientKey(client.ClientID), encodeData, 0); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	return nil
}

// deleteClient deletes OAuth2 client through shared storage. deleteClient 通过共享存储删除 OAuth2 客户端。
func (s *OAuth2Server) deleteClient(ctx context.Context, clientID string) error {
	if err := s.storage.Delete(ctx, s.getClientKey(clientID)); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	return nil
}

// getClient gets OAuth2 client through shared storage. getClient 通过共享存储获取 OAuth2 客户端。
func (s *OAuth2Server) getClient(ctx context.Context, clientID string) (*Client, error) {
	data, err := s.storage.Get(ctx, s.getClientKey(clientID))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	if data == nil {
		return nil, derror.ErrClientNotFound
	}

	rawData, err := utils.ToBytes(data)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrTypeConvert, err)
	}

	var client Client
	if err = s.serializer.Decode(rawData, &client); err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}
	return &client, nil
}

// isValidRedirectURI Checks if redirect URI is valid for client 检查回调URI是否有效
func (s *OAuth2Server) isValidRedirectURI(client *Client, redirectURI string) bool {
	if redirectURI == "" {
		return false
	}
	for _, uri := range client.RedirectURIs {
		if uri == redirectURI {
			return true
		}
	}
	return false
}

// isValidScopes Checks if requested scopes are allowed for client 检查请求的权限范围是否被允许
func (s *OAuth2Server) isValidScopes(client *Client, scopes []string) bool {
	if len(scopes) == 0 {
		return true
	}

	if len(client.Scopes) == 0 {
		return true
	}

	allowedScopes := make(map[string]struct{}, len(client.Scopes))
	for _, scope := range client.Scopes {
		allowedScopes[scope] = struct{}{}
	}

	for _, scope := range scopes {
		if _, ok := allowedScopes[scope]; !ok {
			return false
		}
	}

	return true
}

// isValidGrantType Checks if grant type is allowed for client 检查授权类型是否被允许
func (s *OAuth2Server) isValidGrantType(client *Client, grantType GrantType) bool {
	if len(client.GrantTypes) == 0 {
		return true
	}

	for _, gt := range client.GrantTypes {
		if gt == grantType {
			return true
		}
	}
	return false
}

// normalizeCodeChallengeMethod normalizes PKCE challenge method.
func normalizeCodeChallengeMethod(codeChallenge, method string) (string, error) {
	if codeChallenge == "" {
		return "", nil
	}
	method = strings.TrimSpace(method)
	if method == "" {
		return CodeChallengeMethodPlain, nil
	}
	switch method {
	case CodeChallengeMethodPlain, CodeChallengeMethodS256:
		return method, nil
	default:
		return "", derror.ErrInvalidParam
	}
}

// verifyCodeChallenge verifies PKCE verifier against stored challenge.
func verifyCodeChallenge(codeChallenge, method, codeVerifier string) error {
	if codeChallenge == "" {
		return nil
	}
	codeVerifier = strings.TrimSpace(codeVerifier)
	if codeVerifier == "" {
		return derror.ErrInvalidCodeVerifier
	}
	if method == "" {
		method = CodeChallengeMethodPlain
	}
	switch method {
	case CodeChallengeMethodPlain:
		if codeVerifier != codeChallenge {
			return derror.ErrInvalidCodeVerifier
		}
	case CodeChallengeMethodS256:
		sum := sha256.Sum256([]byte(codeVerifier))
		if base64.RawURLEncoding.EncodeToString(sum[:]) != codeChallenge {
			return derror.ErrInvalidCodeVerifier
		}
	default:
		return derror.ErrInvalidParam
	}
	return nil
}

// generateAccessToken Generates access token and refresh token 生成访问令牌和刷新令牌
func (s *OAuth2Server) generateAccessToken(ctx context.Context, userID, clientID string, scopes []string) (*AccessToken, error) {
	tokenBytes := make([]byte, AccessTokenLength)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}
	accessToken := hex.EncodeToString(tokenBytes)

	refreshBytes := make([]byte, RefreshTokenLength)
	if _, err := rand.Read(refreshBytes); err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}
	refreshToken := hex.EncodeToString(refreshBytes)

	token := &AccessToken{
		Token:        accessToken,
		TokenType:    TokenTypeBearer,
		ExpiresIn:    int64(s.tokenExpiration.Seconds()),
		RefreshToken: refreshToken,
		Scopes:       append([]string(nil), scopes...),
		UserID:       userID,
		ClientID:     clientID,
	}

	encodeData, err := s.serializer.Encode(token)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}

	tokenKey := s.getTokenKey(accessToken)
	refreshKey := s.getRefreshKey(refreshToken)

	if err = s.storage.Set(ctx, tokenKey, encodeData, s.tokenExpiration); err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	if err = s.storage.Set(ctx, refreshKey, encodeData, s.refreshExpiration); err != nil {
		_ = s.storage.Delete(ctx, tokenKey)
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	return token, nil
}
