// @Author daixk 2025/12/22 15:56:00
package oauth2

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math"
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
	Code                string    // Authorization code 授权码
	ClientID            string    // Client ID 客户端ID
	RedirectURI         string    // Redirect URI 回调URI
	UserID              string    // User ID 用户ID
	Scopes              []string  // Requested scopes 请求的权限范围
	CodeChallenge       string    // PKCE code challenge PKCE 授权码挑战值
	CodeChallengeMethod string    // PKCE challenge method PKCE 授权码挑战方法
	CreateTime          int64     // Creation time 创建时间
	ExpiresIn           int64     // Expiration time in seconds 过期时间（秒）
	ExpiresAt           time.Time // Precise deadline; zero preserves legacy timestamp validation. 精确截止时间，零值兼容旧时间戳校验。
	Used                bool      // Whether used 是否已使用
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
	Scopes       []string  // Optional scopes; refresh requests may only narrow the original grant. 可选权限范围，刷新请求只能缩小原授权。
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
		return s.refreshAccessToken(ctx, req.ClientID, req.RefreshToken, req.ClientSecret, req.Scopes)

	default:
		return nil, derror.ErrInvalidGrantType
	}
}

// GenerateAuthorizationCode generates an authorization code. GenerateAuthorizationCode 生成授权码。
func (s *OAuth2Server) GenerateAuthorizationCode(ctx context.Context, clientID, userID, redirectURI string, scopes []string) (*AuthorizationCode, error) {
	return s.GenerateAuthorizationCodeWithPKCE(ctx, clientID, userID, redirectURI, scopes, "", "")
}

// GenerateAuthorizationCodeWithPKCE generates an authorization code with an optional PKCE challenge. GenerateAuthorizationCodeWithPKCE 生成可携带 PKCE 挑战的授权码。
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
	if !s.isValidGrantType(client, GrantTypeAuthorizationCode) {
		return nil, derror.ErrInvalidGrantType
	}

	if !s.isValidScopes(client, scopes) {
		return nil, derror.ErrInvalidScope
	}
	codeChallengeMethod, err = normalizeCodeChallengeMethod(codeChallenge, codeChallengeMethod)
	if err != nil {
		return nil, err
	}

	codeBytes := make([]byte, CodeLength)
	if _, err = rand.Read(codeBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random code: %w", err)
	}
	code := hex.EncodeToString(codeBytes)

	now := time.Now()
	authCode := &AuthorizationCode{
		Code:                code,
		ClientID:            clientID,
		RedirectURI:         redirectURI,
		UserID:              userID,
		Scopes:              append([]string(nil), scopes...),
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: codeChallengeMethod,
		CreateTime:          now.Unix(),
		ExpiresIn:           durationSeconds(s.codeExpiration),
		ExpiresAt:           now.Add(s.codeExpiration),
		Used:                false,
	}

	encodeData, err := s.serializer.Encode(authCode)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}

	key := s.getCodeKey(code)
	ttl := remainingAuthCodeDuration(authCode)
	if ttl <= 0 {
		return nil, derror.ErrAuthCodeExpired
	}
	if err := s.storage.Set(ctx, key, encodeData, ttl); err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	return authCode, nil
}

// ExchangeCodeForToken exchanges an authorization code for an access token. ExchangeCodeForToken 使用授权码换取访问令牌。
func (s *OAuth2Server) ExchangeCodeForToken(ctx context.Context, code, clientID, clientSecret, redirectURI string) (*AccessToken, error) {
	return s.ExchangeCodeForTokenWithPKCE(ctx, code, clientID, clientSecret, redirectURI, "")
}

// ExchangeCodeForTokenWithPKCE exchanges an authorization code with an optional PKCE verifier. ExchangeCodeForTokenWithPKCE 使用可选 PKCE 校验器交换授权码。
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

	authCode, err := s.getAuthorizationCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if authCode.Used {
		return nil, derror.ErrAuthCodeUsed
	}

	if authCode.ClientID != clientID {
		return nil, derror.ErrClientMismatch
	}

	if authCode.RedirectURI != redirectURI {
		return nil, derror.ErrRedirectURIMismatch
	}

	if remainingAuthCodeDuration(authCode) <= 0 {
		return nil, derror.ErrAuthCodeExpired
	}

	// Recheck current client policy before consuming the code. 消费授权码前重新检查当前客户端策略。
	if !s.isValidRedirectURI(client, redirectURI) {
		return nil, derror.ErrInvalidRedirectURI
	}
	if !s.isValidScopes(client, authCode.Scopes) {
		return nil, derror.ErrInvalidScope
	}
	if err = verifyCodeChallenge(authCode.CodeChallenge, authCode.CodeChallengeMethod, codeVerifier); err != nil {
		return nil, err
	}
	if err = s.markAuthorizationCodeUsed(ctx, authCode); err != nil {
		return nil, err
	}

	return s.generateAccessToken(ctx, authCode.UserID, authCode.ClientID, authCode.Scopes)
}

// getAuthorizationCode loads and validates stored authorization-code metadata. getAuthorizationCode 加载并校验已存储的授权码元数据。
func (s *OAuth2Server) getAuthorizationCode(ctx context.Context, code string) (*AuthorizationCode, error) {
	if code == "" {
		return nil, derror.ErrInvalidAuthCode
	}
	data, err := s.storage.Get(ctx, s.getCodeKey(code))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	if data == nil {
		return nil, derror.ErrInvalidAuthCode
	}
	return s.decodeAuthorizationCode(data, code)
}

// decodeAuthorizationCode verifies the identity of a stored authorization code. decodeAuthorizationCode 校验已存储授权码的身份。
func (s *OAuth2Server) decodeAuthorizationCode(data any, code string) (*AuthorizationCode, error) {
	rawData, err := utils.ToBytes(data)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrTypeConvert, err)
	}
	var authCode AuthorizationCode
	if err = s.serializer.Decode(rawData, &authCode); err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}
	if authCode.Code != code || authCode.ClientID == "" || authCode.UserID == "" {
		return nil, derror.ErrInvalidAuthCode
	}
	return &authCode, nil
}

// markAuthorizationCodeUsed persists the used state while preserving the remaining TTL. markAuthorizationCodeUsed 保留剩余有效期并持久化已使用状态。
func (s *OAuth2Server) markAuthorizationCodeUsed(ctx context.Context, authCode *AuthorizationCode) error {
	if authCode == nil || authCode.Code == "" {
		return derror.ErrInvalidAuthCode
	}
	ttl := remainingAuthCodeDuration(authCode)
	if ttl <= 0 {
		return derror.ErrAuthCodeExpired
	}

	// Claim the current record when atomic storage is available; ordinary storage retains the sequential fallback. 原子存储下领取当前记录，普通存储保留顺序处理回退。
	if atomicStorage, ok := s.storage.(adapter.AtomicStorage); ok {
		data, err := atomicStorage.GetAndDelete(ctx, s.getCodeKey(authCode.Code))
		if err != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
		if data == nil {
			return derror.ErrInvalidAuthCode
		}
		claimed, err := s.decodeAuthorizationCode(data, authCode.Code)
		if err != nil {
			return err
		}
		if claimed.Used {
			// Keep the used marker so later requests retain the replay error. 保留已使用记录，使后续请求仍可识别重复兑换。
			if remaining := remainingAuthCodeDuration(claimed); remaining > 0 {
				if err = s.storage.Set(ctx, s.getCodeKey(claimed.Code), data, remaining); err != nil {
					return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
				}
			}
			return derror.ErrAuthCodeUsed
		}
	}

	authCode.Used = true
	encoded, err := s.serializer.Encode(authCode)
	if err != nil {
		return fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}

	// Encoding and claiming may consume the remaining lifetime. 编码及领取记录可能耗尽剩余有效期。
	ttl = remainingAuthCodeDuration(authCode)
	if ttl <= 0 {
		return derror.ErrAuthCodeExpired
	}
	if err = s.storage.Set(ctx, s.getCodeKey(authCode.Code), encoded, ttl); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	return nil
}

// remainingAuthCodeDuration calculates the remaining authorization-code lifetime. remainingAuthCodeDuration 计算授权码剩余有效期。
func remainingAuthCodeDuration(authCode *AuthorizationCode) time.Duration {
	if authCode == nil || authCode.ExpiresIn <= 0 {
		return 0
	}
	deadline := authCode.ExpiresAt
	if deadline.IsZero() {
		// Keep old records readable without overflowing timestamp addition. 兼容旧记录并避免时间戳相加溢出。
		if authCode.CreateTime > math.MaxInt64-authCode.ExpiresIn {
			return 0
		}
		deadline = time.Unix(authCode.CreateTime+authCode.ExpiresIn, 0)
	}
	ttl := time.Until(deadline)
	if ttl <= 0 {
		return 0
	}
	return ttl
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
	return s.refreshAccessToken(ctx, clientID, refreshToken, clientSecret, nil)
}

// refreshAccessToken rotates a token pair, optionally narrowing its granted scopes. refreshAccessToken 轮换令牌对，并可缩小授权范围。
func (s *OAuth2Server) refreshAccessToken(ctx context.Context, clientID, refreshToken, clientSecret string, scopes []string) (*AccessToken, error) {
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
	if accessTokenInfo.RefreshToken != refreshToken || accessTokenInfo.Token == "" || accessTokenInfo.UserID == "" || accessTokenInfo.ClientID == "" {
		return nil, derror.ErrInvalidRefreshToken
	}

	if accessTokenInfo.ClientID != clientID {
		return nil, derror.ErrClientMismatch
	}

	// A refresh can retain or narrow the original grant, never expand it. 刷新只能保留或缩小原授权范围，不能扩大。
	if len(scopes) == 0 {
		scopes = accessTokenInfo.Scopes
	} else {
		for _, requested := range scopes {
			found := false
			for _, granted := range accessTokenInfo.Scopes {
				if requested == granted {
					found = true
					break
				}
			}
			if !found {
				return nil, derror.ErrInvalidScope
			}
		}
	}
	if !s.isValidScopes(client, scopes) {
		return nil, derror.ErrInvalidScope
	}

	token, err := s.generateAccessToken(ctx, accessTokenInfo.UserID, accessTokenInfo.ClientID, scopes)
	if err != nil {
		return nil, err
	}
	if err = s.consumeRefreshToken(ctx, key); err != nil {
		_ = s.deleteTokenPair(ctx, token)
		return nil, err
	}

	_ = s.storage.Delete(ctx, s.getTokenKey(accessTokenInfo.Token))

	return token, nil
}

// consumeRefreshToken removes a refresh token with atomic semantics when available. consumeRefreshToken 优先使用原子语义消费刷新令牌。
func (s *OAuth2Server) consumeRefreshToken(ctx context.Context, key string) error {
	if atomicStorage, ok := s.storage.(adapter.AtomicStorage); ok {
		value, err := atomicStorage.GetAndDelete(ctx, key)
		if err != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
		if value == nil {
			return derror.ErrInvalidRefreshToken
		}
		return nil
	}
	if err := s.storage.Delete(ctx, key); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	return nil
}

// deleteTokenPair deletes the access-token and refresh-token mappings. deleteTokenPair 删除访问令牌与刷新令牌映射。
func (s *OAuth2Server) deleteTokenPair(ctx context.Context, token *AccessToken) error {
	if token == nil {
		return nil
	}
	keys := []string{s.getTokenKey(token.Token)}
	if token.RefreshToken != "" {
		keys = append(keys, s.getRefreshKey(token.RefreshToken))
	}
	if err := s.storage.Delete(ctx, keys...); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	return nil
}

// ValidateAccessToken Validates access token 验证访问令牌
func (s *OAuth2Server) ValidateAccessToken(ctx context.Context, accessToken string) bool {
	_, err := s.ValidateAccessTokenAndGetInfo(ctx, accessToken)
	return err == nil
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
	if accessTokenInfo.Token != accessToken || accessTokenInfo.UserID == "" || accessTokenInfo.ClientID == "" {
		return nil, derror.ErrInvalidAccessToken
	}

	return &accessTokenInfo, nil
}

// RevokeToken Revokes access token and its refresh token 撤销访问令牌及其刷新令牌
func (s *OAuth2Server) RevokeToken(ctx context.Context, accessToken string) error {
	accessTokenInfo, err := s.ValidateAccessTokenAndGetInfo(ctx, accessToken)
	if err != nil {
		return err
	}

	if accessTokenInfo.RefreshToken != "" {
		if err = s.storage.Delete(ctx, s.getRefreshKey(accessTokenInfo.RefreshToken)); err != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
	}
	if err = s.storage.Delete(ctx, s.getTokenKey(accessToken)); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	return nil
}

// getCodeKey Gets storage key for authorization code 获取授权码的存储键
func (s *OAuth2Server) getCodeKey(code string) string {
	return utils.StorageNamespace(s.keyPrefix, s.authType) + CodeKeySuffix + code
}

// getTokenKey Gets storage key for access token 获取访问令牌的存储键
func (s *OAuth2Server) getTokenKey(token string) string {
	return utils.StorageNamespace(s.keyPrefix, s.authType) + TokenKeySuffix + token
}

// getRefreshKey Gets storage key for refresh token 获取刷新令牌的存储键
func (s *OAuth2Server) getRefreshKey(refreshToken string) string {
	return utils.StorageNamespace(s.keyPrefix, s.authType) + RefreshKeySuffix + refreshToken
}

// getClientKey gets storage key for OAuth2 client. getClientKey 获取 OAuth2 客户端存储键。
func (s *OAuth2Server) getClientKey(clientID string) string {
	return utils.StorageNamespace(s.keyPrefix, s.authType) + ClientKeySuffix + clientID
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
	if clientID == "" {
		return derror.ErrClientOrClientIDEmpty
	}
	if err := s.storage.Delete(ctx, s.getClientKey(clientID)); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	return nil
}

// getClient gets OAuth2 client through shared storage. getClient 通过共享存储获取 OAuth2 客户端。
func (s *OAuth2Server) getClient(ctx context.Context, clientID string) (*Client, error) {
	if clientID == "" {
		return nil, derror.ErrClientOrClientIDEmpty
	}
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
	if client.ClientID != clientID {
		return nil, derror.ErrClientNotFound
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

// normalizeCodeChallengeMethod normalizes the PKCE challenge method. normalizeCodeChallengeMethod 规范化 PKCE 挑战方法。
func normalizeCodeChallengeMethod(codeChallenge, method string) (string, error) {
	if codeChallenge == "" {
		if method != "" {
			return "", derror.ErrInvalidParam
		}
		return "", nil
	}
	method = strings.TrimSpace(method)
	if method == "" {
		method = CodeChallengeMethodPlain
	}
	switch method {
	case CodeChallengeMethodPlain:
		if !validPKCEVerifier(codeChallenge) {
			return "", derror.ErrInvalidParam
		}
	case CodeChallengeMethodS256:
		if len(codeChallenge) != 43 {
			return "", derror.ErrInvalidParam
		}
		decoded, err := base64.RawURLEncoding.DecodeString(codeChallenge)
		if err != nil || len(decoded) != sha256.Size || base64.RawURLEncoding.EncodeToString(decoded) != codeChallenge {
			return "", derror.ErrInvalidParam
		}
	default:
		return "", derror.ErrInvalidParam
	}
	return method, nil
}

// validPKCEVerifier enforces RFC 7636 length and unreserved ASCII characters. validPKCEVerifier 校验 RFC 7636 要求的长度和 ASCII 非保留字符。
func validPKCEVerifier(value string) bool {
	if len(value) < 43 || len(value) > 128 {
		return false
	}
	for i := range len(value) {
		c := value[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '.' || c == '_' || c == '~') {
			return false
		}
	}
	return true
}

// verifyCodeChallenge verifies a PKCE verifier against the stored challenge. verifyCodeChallenge 根据已存储挑战校验 PKCE 校验器。
func verifyCodeChallenge(codeChallenge, method, codeVerifier string) error {
	method, err := normalizeCodeChallengeMethod(codeChallenge, method)
	if err != nil {
		return err
	}
	if codeChallenge == "" {
		if codeVerifier != "" {
			return derror.ErrInvalidCodeVerifier
		}
		return nil
	}
	if !validPKCEVerifier(codeVerifier) {
		return derror.ErrInvalidCodeVerifier
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
		ExpiresIn:    durationSeconds(s.tokenExpiration),
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

// durationSeconds rounds a positive duration up to whole seconds. durationSeconds 将正时长向上取整为秒。
func durationSeconds(duration time.Duration) int64 {
	seconds := duration / time.Second
	if duration%time.Second != 0 {
		seconds++
	}
	if seconds <= 0 {
		return 1
	}
	return int64(seconds)
}
