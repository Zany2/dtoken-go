// @Author daixk 2025/12/22 15:56:00
package oauth2

import (
	"time"

	capability "github.com/Zany2/dtoken-go/contract/oauth2"
)

const (
	// DefaultCodeExpiration stores code expiration DefaultCodeExpiration 存储授权码过期时间
	DefaultCodeExpiration = 10 * time.Minute
	// DefaultTokenExpiration stores token expiration DefaultTokenExpiration 存储访问令牌过期时间
	DefaultTokenExpiration = 2 * time.Hour
	// DefaultRefreshTTL stores refresh expiration DefaultRefreshTTL 存储刷新令牌过期时间
	DefaultRefreshTTL = 30 * 24 * time.Hour

	// CodeLength stores code byte length CodeLength 存储授权码字节长度
	CodeLength = 32
	// AccessTokenLength stores access token byte length AccessTokenLength 存储访问令牌字节长度
	AccessTokenLength = 32
	// RefreshTokenLength stores refresh token byte length RefreshTokenLength 存储刷新令牌字节长度
	RefreshTokenLength = 32

	// CodeKeySuffix stores code key suffix CodeKeySuffix 存储授权码键后缀
	CodeKeySuffix = "oauth2:code:"
	// TokenKeySuffix stores token key suffix TokenKeySuffix 存储令牌键后缀
	TokenKeySuffix = "oauth2:token:"
	// RefreshKeySuffix stores refresh key suffix RefreshKeySuffix 存储刷新令牌键后缀
	RefreshKeySuffix = "oauth2:refresh:"
	// ClientKeySuffix stores client key suffix. ClientKeySuffix 存储客户端键后缀。
	ClientKeySuffix = "oauth2:client:"

	// TokenTypeBearer stores bearer token type TokenTypeBearer 存储 Bearer 令牌类型
	TokenTypeBearer = "Bearer"

	// CodeChallengeMethodPlain stores plain PKCE method CodeChallengeMethodPlain 存储 plain PKCE 方法
	CodeChallengeMethodPlain = "plain"
	// CodeChallengeMethodS256 stores S256 PKCE method CodeChallengeMethodS256 存储 S256 PKCE 方法
	CodeChallengeMethodS256 = "S256"
)

// GrantType aliases the public capability state. GrantType 引用公开能力状态。
type GrantType = capability.GrantType

const (
	// GrantTypeAuthorizationCode identifies the capability state. GrantTypeAuthorizationCode 标识能力状态。
	GrantTypeAuthorizationCode = capability.GrantTypeAuthorizationCode
	// GrantTypeRefreshToken identifies the capability state. GrantTypeRefreshToken 标识能力状态。
	GrantTypeRefreshToken = capability.GrantTypeRefreshToken
	// GrantTypeClientCredentials identifies the capability state. GrantTypeClientCredentials 标识能力状态。
	GrantTypeClientCredentials = capability.GrantTypeClientCredentials
	// GrantTypePassword identifies the capability state. GrantTypePassword 标识能力状态。
	GrantTypePassword = capability.GrantTypePassword
)
