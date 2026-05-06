package gin

import (
	"context"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/builder"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/core/oauth2"
	"github.com/Zany2/dtoken-go/dtoken"
)

type (
	Config = config.Config

	Manager = manager.Manager

	TokenInfo = manager.TokenInfo

	DisableInfo = manager.DisableInfo

	Session = manager.Session

	Builder = builder.Builder

	DTokenError = derror.DTokenError

	TokenStyle = adapter.TokenStyle
)

const (
	CodeSuccess = derror.CodeSuccess

	CodeBadRequest = derror.CodeBadRequest

	CodeNotLogin = derror.CodeNotLogin

	CodePermissionDenied = derror.CodePermissionDenied

	CodeNotFound = derror.CodeNotFound

	CodeServerError = derror.CodeServerError

	CodeTokenInvalid = derror.CodeTokenInvalid

	CodeTokenExpired = derror.CodeTokenExpired

	CodeAccountDisabled = derror.CodeAccountDisabled
)

var (
	ErrNotLogin = derror.ErrNotLogin

	ErrInvalidToken = derror.ErrInvalidToken

	ErrTokenExpired = derror.ErrTokenExpired

	ErrPermissionDenied = derror.ErrPermissionDenied

	ErrRoleDenied = derror.ErrRoleDenied

	ErrAccountDisabled = derror.ErrAccountDisabled
)

const (
	TokenStyleUUID = adapter.TokenStyleUUID

	TokenStyleSimple = adapter.TokenStyleSimple

	TokenStyleRandom32 = adapter.TokenStyleRandom32

	TokenStyleRandom64 = adapter.TokenStyleRandom64

	TokenStyleRandom128 = adapter.TokenStyleRandom128

	TokenStyleJWT = adapter.TokenStyleJWT
)

func SetManager(mgr *manager.Manager) {
	dtoken.SetManager(mgr)
}

func GetManager(authType ...string) (*manager.Manager, error) {
	return dtoken.GetManager(authType...)
}

func DeleteManager(authType ...string) error {
	return dtoken.DeleteManager(authType...)
}

func DeleteAllManager() {
	dtoken.DeleteAllManager()
}

func NewDefaultBuilder() *builder.Builder {
	return builder.NewBuilder()
}

func NewDefaultConfig() *config.Config {
	return config.DefaultConfig()
}

func Login(ctx context.Context, loginID string, params ...string) (string, error) {
	return dtoken.Login(ctx, loginID, params...)
}

func LoginByToken(ctx context.Context, tokenValue string, authType ...string) error {
	return dtoken.LoginByToken(ctx, tokenValue, authType...)
}

func Logout(ctx context.Context, tokenValue string, authType ...string) error {
	return dtoken.Logout(ctx, tokenValue, authType...)
}

func LogoutByDeviceAndDeviceId(ctx context.Context, loginID string, params ...string) error {
	return dtoken.LogoutByDeviceAndDeviceId(ctx, loginID, params...)
}

func LogoutByDevice(ctx context.Context, loginID string, device string, authType ...string) error {
	return dtoken.LogoutByDevice(ctx, loginID, device, authType...)
}

func LogoutByLoginID(ctx context.Context, loginID string, authType ...string) error {
	return dtoken.LogoutByLoginID(ctx, loginID, authType...)
}

func Kickout(ctx context.Context, tokenValue string, authType ...string) error {
	return dtoken.Kickout(ctx, tokenValue, authType...)
}

func KickoutByDeviceAndDeviceId(ctx context.Context, loginID string, params ...string) error {
	return dtoken.KickoutByDeviceAndDeviceId(ctx, loginID, params...)
}

func KickoutByDevice(ctx context.Context, loginID string, device string, authType ...string) error {
	return dtoken.KickoutByDevice(ctx, loginID, device, authType...)
}

func KickoutByLoginID(ctx context.Context, loginID string, authType ...string) error {
	return dtoken.KickoutByLoginID(ctx, loginID, authType...)
}

func Replace(ctx context.Context, tokenValue string, authType ...string) error {
	return dtoken.Replace(ctx, tokenValue, authType...)
}

func ReplaceByDeviceAndDeviceId(ctx context.Context, loginID string, params ...string) error {
	return dtoken.ReplaceByDeviceAndDeviceId(ctx, loginID, params...)
}

func ReplaceByDevice(ctx context.Context, loginID string, device string, authType ...string) error {
	return dtoken.ReplaceByDevice(ctx, loginID, device, authType...)
}

func ReplaceByLoginID(ctx context.Context, loginID string, authType ...string) error {
	return dtoken.ReplaceByLoginID(ctx, loginID, authType...)
}

func IsLogin(ctx context.Context, tokenValue string, authType ...string) bool {
	return dtoken.IsLogin(ctx, tokenValue, authType...)
}

func CheckLogin(ctx context.Context, tokenValue string, authType ...string) error {
	return dtoken.CheckLogin(ctx, tokenValue, authType...)
}

func GetLoginID(ctx context.Context, tokenValue string, authType ...string) (string, error) {
	return dtoken.GetLoginID(ctx, tokenValue, authType...)
}

func GetTokenInfo(ctx context.Context, tokenValue string, authType ...string) (*manager.TokenInfo, error) {
	return dtoken.GetTokenInfo(ctx, tokenValue, authType...)
}

func GetDevice(ctx context.Context, tokenValue string, authType ...string) (string, error) {
	return dtoken.GetDevice(ctx, tokenValue, authType...)
}

func GetDeviceId(ctx context.Context, tokenValue string, authType ...string) (string, error) {
	return dtoken.GetDeviceId(ctx, tokenValue, authType...)
}

func GetTokenCreateTime(ctx context.Context, tokenValue string, authType ...string) (int64, error) {
	return dtoken.GetTokenCreateTime(ctx, tokenValue, authType...)
}

func GetTokenTTL(ctx context.Context, tokenValue string, authType ...string) (int64, error) {
	return dtoken.GetTokenTTL(ctx, tokenValue, authType...)
}

func GetOnlineTerminalCount(ctx context.Context, loginID string, authType ...string) (int, error) {
	return dtoken.GetOnlineTerminalCount(ctx, loginID, authType...)
}

func GetOnlineTerminalCountByDevice(ctx context.Context, loginID string, device string, authType ...string) (int, error) {
	return dtoken.GetOnlineTerminalCountByDevice(ctx, loginID, device, authType...)
}

func GetOnlineTerminalCountByDeviceAndDeviceId(ctx context.Context, loginID string, device string, deviceId string, authType ...string) (int, error) {
	return dtoken.GetOnlineTerminalCountByDeviceAndDeviceId(ctx, loginID, device, deviceId, authType...)
}

func Disable(ctx context.Context, loginID string, duration time.Duration, reason string, authType ...string) error {
	return dtoken.Disable(ctx, loginID, duration, reason, authType...)
}

func Untie(ctx context.Context, loginID string, authType ...string) error {
	return dtoken.Untie(ctx, loginID, authType...)
}

func IsDisable(ctx context.Context, loginID string, authType ...string) bool {
	return dtoken.IsDisable(ctx, loginID, authType...)
}

func GetDisableInfo(ctx context.Context, loginID string, authType ...string) (*manager.DisableInfo, error) {
	return dtoken.GetDisableInfo(ctx, loginID, authType...)
}

func GetDisableTTL(ctx context.Context, loginID string, authType ...string) (int64, error) {
	return dtoken.GetDisableTTL(ctx, loginID, authType...)
}

func GetSession(ctx context.Context, loginID string, authType ...string) (*manager.Session, error) {
	return dtoken.GetSession(ctx, loginID, authType...)
}

func GetSessionByToken(ctx context.Context, tokenValue string, authType ...string) (*manager.Session, error) {
	return dtoken.GetSessionByToken(ctx, tokenValue, authType...)
}

func GetTokenValueListByLoginID(ctx context.Context, loginID string, checkAlive bool, authType ...string) ([]string, error) {
	return dtoken.GetTokenValueListByLoginID(ctx, loginID, checkAlive, authType...)
}

func GetTokenValueListByDevice(ctx context.Context, loginID string, device string, checkAlive bool, authType ...string) ([]string, error) {
	return dtoken.GetTokenValueListByDevice(ctx, loginID, device, checkAlive, authType...)
}

func GetTokenValueListByDeviceAndDeviceId(ctx context.Context, loginID string, device string, deviceId string, checkAlive bool, authType ...string) ([]string, error) {
	return dtoken.GetTokenValueListByDeviceAndDeviceId(ctx, loginID, device, deviceId, checkAlive, authType...)
}

func AddPermissions(ctx context.Context, loginID string, permissions []string, authType ...string) error {
	return dtoken.AddPermissions(ctx, loginID, permissions, authType...)
}

func AddPermissionsByToken(ctx context.Context, tokenValue string, permissions []string, authType ...string) error {
	return dtoken.AddPermissionsByToken(ctx, tokenValue, permissions, authType...)
}

func RemovePermissions(ctx context.Context, loginID string, permissions []string, authType ...string) error {
	return dtoken.RemovePermissions(ctx, loginID, permissions, authType...)
}

func RemovePermissionsByToken(ctx context.Context, tokenValue string, permissions []string, authType ...string) error {
	return dtoken.RemovePermissionsByToken(ctx, tokenValue, permissions, authType...)
}

func GetPermissions(ctx context.Context, loginID string, authType ...string) ([]string, error) {
	return dtoken.GetPermissions(ctx, loginID, authType...)
}

func GetPermissionsByToken(ctx context.Context, tokenValue string, authType ...string) ([]string, error) {
	return dtoken.GetPermissionsByToken(ctx, tokenValue, authType...)
}

func HasPermission(ctx context.Context, loginID string, permission string, authType ...string) bool {
	return dtoken.HasPermission(ctx, loginID, permission, authType...)
}

func HasPermissionByToken(ctx context.Context, tokenValue string, permission string, authType ...string) bool {
	return dtoken.HasPermissionByToken(ctx, tokenValue, permission, authType...)
}

func HasPermissionsAnd(ctx context.Context, loginID string, permissions []string, authType ...string) bool {
	return dtoken.HasPermissionsAnd(ctx, loginID, permissions, authType...)
}

func HasPermissionsAndByToken(ctx context.Context, tokenValue string, permissions []string, authType ...string) bool {
	return dtoken.HasPermissionsAndByToken(ctx, tokenValue, permissions, authType...)
}

func HasPermissionsOr(ctx context.Context, loginID string, permissions []string, authType ...string) bool {
	return dtoken.HasPermissionsOr(ctx, loginID, permissions, authType...)
}

func HasPermissionsOrByToken(ctx context.Context, tokenValue string, permissions []string, authType ...string) bool {
	return dtoken.HasPermissionsOrByToken(ctx, tokenValue, permissions, authType...)
}

func AddRoles(ctx context.Context, loginID string, roles []string, authType ...string) error {
	return dtoken.AddRoles(ctx, loginID, roles, authType...)
}

func AddRolesByToken(ctx context.Context, tokenValue string, roles []string, authType ...string) error {
	return dtoken.AddRolesByToken(ctx, tokenValue, roles, authType...)
}

func RemoveRoles(ctx context.Context, loginID string, roles []string, authType ...string) error {
	return dtoken.RemoveRoles(ctx, loginID, roles, authType...)
}

func RemoveRolesByToken(ctx context.Context, tokenValue string, roles []string, authType ...string) error {
	return dtoken.RemoveRolesByToken(ctx, tokenValue, roles, authType...)
}

func GetRoles(ctx context.Context, loginID string, authType ...string) ([]string, error) {
	return dtoken.GetRoles(ctx, loginID, authType...)
}

func GetRolesByToken(ctx context.Context, tokenValue string, authType ...string) ([]string, error) {
	return dtoken.GetRolesByToken(ctx, tokenValue, authType...)
}

func HasRole(ctx context.Context, loginID string, role string, authType ...string) bool {
	return dtoken.HasRole(ctx, loginID, role, authType...)
}

func HasRoleByToken(ctx context.Context, tokenValue string, role string, authType ...string) bool {
	return dtoken.HasRoleByToken(ctx, tokenValue, role, authType...)
}

func HasRolesAnd(ctx context.Context, loginID string, roles []string, authType ...string) bool {
	return dtoken.HasRolesAnd(ctx, loginID, roles, authType...)
}

func HasRolesAndByToken(ctx context.Context, tokenValue string, roles []string, authType ...string) bool {
	return dtoken.HasRolesAndByToken(ctx, tokenValue, roles, authType...)
}

func HasRolesOr(ctx context.Context, loginID string, roles []string, authType ...string) bool {
	return dtoken.HasRolesOr(ctx, loginID, roles, authType...)
}

func HasRolesOrByToken(ctx context.Context, tokenValue string, roles []string, authType ...string) bool {
	return dtoken.HasRolesOrByToken(ctx, tokenValue, roles, authType...)
}

func GenerateNonce(ctx context.Context, authType ...string) (string, error) {
	return dtoken.GenerateNonce(ctx, authType...)
}

func VerifyNonce(ctx context.Context, nonce string, authType ...string) bool {
	return dtoken.VerifyNonce(ctx, nonce, authType...)
}

func VerifyAndConsumeNonce(ctx context.Context, nonce string, authType ...string) error {
	return dtoken.VerifyAndConsumeNonce(ctx, nonce, authType...)
}

func IsNonceValid(ctx context.Context, nonce string, authType ...string) bool {
	return dtoken.IsNonceValid(ctx, nonce, authType...)
}

func RegisterOAuth2Client(client *oauth2.Client, authType ...string) error {
	return dtoken.RegisterOAuth2Client(client, authType...)
}

func UnregisterOAuth2Client(clientID string, authType ...string) error {
	return dtoken.UnregisterOAuth2Client(clientID, authType...)
}

func GetOAuth2Client(clientID string, authType ...string) (*oauth2.Client, error) {
	return dtoken.GetOAuth2Client(clientID, authType...)
}

func OAuth2Token(ctx context.Context, req *oauth2.TokenRequest, validateUser oauth2.UserValidator, authType ...string) (*oauth2.AccessToken, error) {
	return dtoken.OAuth2Token(ctx, req, validateUser, authType...)
}

func GenerateOAuth2AuthorizationCode(ctx context.Context, clientID, userID, redirectURI string, scopes []string, authType ...string) (*oauth2.AuthorizationCode, error) {
	return dtoken.GenerateOAuth2AuthorizationCode(ctx, clientID, userID, redirectURI, scopes, authType...)
}

func ExchangeOAuth2CodeForToken(ctx context.Context, code, clientID, clientSecret, redirectURI string, authType ...string) (*oauth2.AccessToken, error) {
	return dtoken.ExchangeOAuth2CodeForToken(ctx, code, clientID, clientSecret, redirectURI, authType...)
}

func OAuth2ClientCredentialsToken(ctx context.Context, clientID, clientSecret string, scopes []string, authType ...string) (*oauth2.AccessToken, error) {
	return dtoken.OAuth2ClientCredentialsToken(ctx, clientID, clientSecret, scopes, authType...)
}

func OAuth2PasswordGrantToken(ctx context.Context, clientID, clientSecret, username, password string, scopes []string, validateUser oauth2.UserValidator, authType ...string) (*oauth2.AccessToken, error) {
	return dtoken.OAuth2PasswordGrantToken(ctx, clientID, clientSecret, username, password, scopes, validateUser, authType...)
}

func RefreshOAuth2AccessToken(ctx context.Context, clientID, refreshToken, clientSecret string, authType ...string) (*oauth2.AccessToken, error) {
	return dtoken.RefreshOAuth2AccessToken(ctx, clientID, refreshToken, clientSecret, authType...)
}

func ValidateOAuth2AccessToken(ctx context.Context, accessToken string, authType ...string) bool {
	return dtoken.ValidateOAuth2AccessToken(ctx, accessToken, authType...)
}

func ValidateOAuth2AccessTokenAndGetInfo(ctx context.Context, accessToken string, authType ...string) (*oauth2.AccessToken, error) {
	return dtoken.ValidateOAuth2AccessTokenAndGetInfo(ctx, accessToken, authType...)
}

func RevokeOAuth2Token(ctx context.Context, accessToken string, authType ...string) error {
	return dtoken.RevokeOAuth2Token(ctx, accessToken, authType...)
}
