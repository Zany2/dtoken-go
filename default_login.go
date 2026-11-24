package dtoken

import (
	"context"
	"time"

	manager "github.com/Zany2/dtoken-go/internal/engine"
)

// Login delegates to the default authentication namespace. Login 委托默认认证体系执行。
func Login(ctx context.Context, opts LoginOptions) (string, error) {
	return AuthType("").Login(ctx, opts)
}

// LoginID delegates to the default authentication namespace. LoginID 委托默认认证体系执行。
func LoginID(ctx context.Context, loginID string) (string, error) {
	return AuthType("").LoginID(ctx, loginID)
}

// LoginWithTimeout delegates to the default authentication namespace. LoginWithTimeout 委托默认认证体系执行。
func LoginWithTimeout(ctx context.Context, loginID string, timeout time.Duration) (string, error) {
	return AuthType("").LoginWithTimeout(ctx, loginID, timeout)
}

// LoginWithRefreshToken delegates to the default authentication namespace. LoginWithRefreshToken 委托默认认证体系执行。
func LoginWithRefreshToken(ctx context.Context, loginID string) (*manager.RefreshTokenPair, error) {
	return AuthType("").LoginWithRefreshToken(ctx, loginID)
}

// LoginWithRefreshTokenOptions delegates to the default authentication namespace. LoginWithRefreshTokenOptions 委托默认认证体系执行。
func LoginWithRefreshTokenOptions(ctx context.Context, opts RefreshTokenOptions) (*manager.RefreshTokenPair, error) {
	return AuthType("").LoginWithRefreshTokenOptions(ctx, opts)
}

// RefreshToken delegates to the default authentication namespace. RefreshToken 委托默认认证体系执行。
func RefreshToken(ctx context.Context, refreshToken string) (*manager.RefreshTokenPair, error) {
	return AuthType("").RefreshToken(ctx, refreshToken)
}

// RevokeRefreshToken delegates to the default authentication namespace. RevokeRefreshToken 委托默认认证体系执行。
func RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	return AuthType("").RevokeRefreshToken(ctx, refreshToken)
}

// GetRefreshTokenTTL delegates to the default authentication namespace. GetRefreshTokenTTL 委托默认认证体系执行。
func GetRefreshTokenTTL(ctx context.Context, refreshToken string) (int64, error) {
	return AuthType("").GetRefreshTokenTTL(ctx, refreshToken)
}

// IntrospectToken delegates to the default authentication namespace. IntrospectToken 委托默认认证体系执行。
func IntrospectToken(ctx context.Context, token string) (*manager.TokenIntrospection, error) {
	return AuthType("").IntrospectToken(ctx, token)
}

// LoginByToken delegates to the default authentication namespace. LoginByToken 委托默认认证体系执行。
func LoginByToken(ctx context.Context, token string) error {
	return AuthType("").LoginByToken(ctx, token)
}

// LogoutByToken delegates to the default authentication namespace. LogoutByToken 委托默认认证体系执行。
func LogoutByToken(ctx context.Context, token string) error {
	return AuthType("").LogoutByToken(ctx, token)
}

// LogoutByDeviceAndDeviceID delegates to the default authentication namespace. LogoutByDeviceAndDeviceID 委托默认认证体系执行。
func LogoutByDeviceAndDeviceID(ctx context.Context, loginID, device, deviceID string) error {
	return AuthType("").LogoutByDeviceAndDeviceID(ctx, loginID, device, deviceID)
}

// LogoutByDevice delegates to the default authentication namespace. LogoutByDevice 委托默认认证体系执行。
func LogoutByDevice(ctx context.Context, loginID, device string) error {
	return AuthType("").LogoutByDevice(ctx, loginID, device)
}

// LogoutByLoginID delegates to the default authentication namespace. LogoutByLoginID 委托默认认证体系执行。
func LogoutByLoginID(ctx context.Context, loginID string) error {
	return AuthType("").LogoutByLoginID(ctx, loginID)
}

// Logout delegates to the default authentication namespace. Logout 委托默认认证体系执行。
func Logout(ctx context.Context, opts LogoutOptions) error {
	return AuthType("").Logout(ctx, opts)
}

// KickoutByToken delegates to the default authentication namespace. KickoutByToken 委托默认认证体系执行。
func KickoutByToken(ctx context.Context, token string) error {
	return AuthType("").KickoutByToken(ctx, token)
}

// KickoutByDeviceAndDeviceID delegates to the default authentication namespace. KickoutByDeviceAndDeviceID 委托默认认证体系执行。
func KickoutByDeviceAndDeviceID(ctx context.Context, loginID, device, deviceID string) error {
	return AuthType("").KickoutByDeviceAndDeviceID(ctx, loginID, device, deviceID)
}

// KickoutByDevice delegates to the default authentication namespace. KickoutByDevice 委托默认认证体系执行。
func KickoutByDevice(ctx context.Context, loginID, device string) error {
	return AuthType("").KickoutByDevice(ctx, loginID, device)
}

// KickoutByLoginID delegates to the default authentication namespace. KickoutByLoginID 委托默认认证体系执行。
func KickoutByLoginID(ctx context.Context, loginID string) error {
	return AuthType("").KickoutByLoginID(ctx, loginID)
}

// Kickout delegates to the default authentication namespace. Kickout 委托默认认证体系执行。
func Kickout(ctx context.Context, opts LogoutOptions) error {
	return AuthType("").Kickout(ctx, opts)
}

// ReplaceByToken delegates to the default authentication namespace. ReplaceByToken 委托默认认证体系执行。
func ReplaceByToken(ctx context.Context, token string) error {
	return AuthType("").ReplaceByToken(ctx, token)
}

// ReplaceByDeviceAndDeviceID delegates to the default authentication namespace. ReplaceByDeviceAndDeviceID 委托默认认证体系执行。
func ReplaceByDeviceAndDeviceID(ctx context.Context, loginID, device, deviceID string) error {
	return AuthType("").ReplaceByDeviceAndDeviceID(ctx, loginID, device, deviceID)
}

// ReplaceByDevice delegates to the default authentication namespace. ReplaceByDevice 委托默认认证体系执行。
func ReplaceByDevice(ctx context.Context, loginID, device string) error {
	return AuthType("").ReplaceByDevice(ctx, loginID, device)
}

// ReplaceByLoginID delegates to the default authentication namespace. ReplaceByLoginID 委托默认认证体系执行。
func ReplaceByLoginID(ctx context.Context, loginID string) error {
	return AuthType("").ReplaceByLoginID(ctx, loginID)
}

// Replace delegates to the default authentication namespace. Replace 委托默认认证体系执行。
func Replace(ctx context.Context, opts LogoutOptions) error {
	return AuthType("").Replace(ctx, opts)
}

// IsLogin delegates to the default authentication namespace. IsLogin 委托默认认证体系执行。
func IsLogin(ctx context.Context, token string) bool {
	return AuthType("").IsLogin(ctx, token)
}

// CheckLogin delegates to the default authentication namespace. CheckLogin 委托默认认证体系执行。
func CheckLogin(ctx context.Context, token string) error {
	return AuthType("").CheckLogin(ctx, token)
}

// GetLoginID delegates to the default authentication namespace. GetLoginID 委托默认认证体系执行。
func GetLoginID(ctx context.Context, token string) (string, error) {
	return AuthType("").GetLoginID(ctx, token)
}

// GetTokenInfo delegates to the default authentication namespace. GetTokenInfo 委托默认认证体系执行。
func GetTokenInfo(ctx context.Context, token string) (*manager.TokenInfo, error) {
	return AuthType("").GetTokenInfo(ctx, token)
}

// GetDevice delegates to the default authentication namespace. GetDevice 委托默认认证体系执行。
func GetDevice(ctx context.Context, token string) (string, error) {
	return AuthType("").GetDevice(ctx, token)
}

// GetDeviceAndDeviceID delegates to the default authentication namespace. GetDeviceAndDeviceID 委托默认认证体系执行。
func GetDeviceAndDeviceID(ctx context.Context, token string) (string, string, error) {
	return AuthType("").GetDeviceAndDeviceID(ctx, token)
}

// GetDeviceID delegates to the default authentication namespace. GetDeviceID 委托默认认证体系执行。
func GetDeviceID(ctx context.Context, token string) (string, error) {
	return AuthType("").GetDeviceID(ctx, token)
}

// GetTokenCreateTime delegates to the default authentication namespace. GetTokenCreateTime 委托默认认证体系执行。
func GetTokenCreateTime(ctx context.Context, token string) (int64, error) {
	return AuthType("").GetTokenCreateTime(ctx, token)
}

// GetTokenTTL delegates to the default authentication namespace. GetTokenTTL 委托默认认证体系执行。
func GetTokenTTL(ctx context.Context, token string) (int64, error) {
	return AuthType("").GetTokenTTL(ctx, token)
}

// RenewTimeout delegates to the default authentication namespace. RenewTimeout 委托默认认证体系执行。
func RenewTimeout(ctx context.Context, token string, timeout time.Duration) error {
	return AuthType("").RenewTimeout(ctx, token, timeout)
}
