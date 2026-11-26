[中文文档](dtoken_zh.md) | English

# Root API Reference

Here dt names `github.com/Zany2/dtoken-go`. Root functions use the default namespace auth:. Select another system with dt.AuthType(name); its business signatures match root functions and Auth methods. See [Auth API](auth.md) for options and instance details.

The global package has been removed. Signatures below belong to the root; login, permission mutations, and disable operations use typed options and do not accept a trailing legacy authType argument.

## Initialization And Namespace Selection

<!-- dtoken-compile: root-registration -->
```go
package main

import (
    "log"
    "context"
    "fmt"

    dt "github.com/Zany2/dtoken-go"
)

func main() {
    defer dt.CloseAll()
    if _, err := dt.NewBuilder().IsPrintBanner(false).Build(); err != nil {
        log.Print(err)
        return
    }
    if _, err := dt.NewBuilder().AuthType("user").IsPrintBanner(false).Build(); err != nil {
        log.Print(err)
        return
    }
    ctx := context.Background()
    token, err := dt.AuthType("user").Login(ctx, dt.LoginOptions{LoginID: "10001", Device: "web"})
    if err != nil {
        log.Print(err)
        return
    }
    if err := dt.AuthType("user").CheckLogin(ctx, token); err != nil {
        log.Print(err)
        return
    }
    fmt.Println(dt.AuthType("user").IsLogin(ctx, token))
}
```

Build registers and returns Auth; duplicate names return ErrManagerAlreadyRegistered. BuildInstance does not register and must be closed by its owner. AuthType selectors resolve each operation; a missing system returns ErrManagerNotFound, or false for boolean checks, without falling back. Fixed instances do not follow replacement. The declarations below are a signature index: use root types, the current request context, and time.Duration for duration parameters, distinct from builder settings measured in seconds.

## Login and token lifecycle

```go
func Login(ctx context.Context, opts dt.LoginOptions) (string, error)
func LoginID(ctx context.Context, loginID string) (string, error)
func LoginWithTimeout(ctx context.Context, loginID string, timeout time.Duration) (string, error)
func LoginWithRefreshToken(ctx context.Context, loginID string) (*dt.RefreshTokenPair, error)
func LoginWithRefreshTokenOptions(ctx context.Context, opts dt.RefreshTokenOptions) (*dt.RefreshTokenPair, error)
func RefreshToken(ctx context.Context, refreshToken string) (*dt.RefreshTokenPair, error)
func RevokeRefreshToken(ctx context.Context, refreshToken string) error
func GetRefreshTokenTTL(ctx context.Context, refreshToken string) (int64, error)
func IntrospectToken(ctx context.Context, token string) (*dt.TokenIntrospection, error)
func LoginByToken(ctx context.Context, token string) error
func LogoutByToken(ctx context.Context, token string) error
func LogoutByDeviceAndDeviceID(ctx context.Context, loginID, device, deviceID string) error
func LogoutByDevice(ctx context.Context, loginID, device string) error
func LogoutByLoginID(ctx context.Context, loginID string) error
func Logout(ctx context.Context, opts dt.LogoutOptions) error
func KickoutByToken(ctx context.Context, token string) error
func KickoutByDeviceAndDeviceID(ctx context.Context, loginID, device, deviceID string) error
func KickoutByDevice(ctx context.Context, loginID, device string) error
func KickoutByLoginID(ctx context.Context, loginID string) error
func Kickout(ctx context.Context, opts dt.LogoutOptions) error
func ReplaceByToken(ctx context.Context, token string) error
func ReplaceByDeviceAndDeviceID(ctx context.Context, loginID, device, deviceID string) error
func ReplaceByDevice(ctx context.Context, loginID, device string) error
func ReplaceByLoginID(ctx context.Context, loginID string) error
func Replace(ctx context.Context, opts dt.LogoutOptions) error
func IsLogin(ctx context.Context, token string) bool
func CheckLogin(ctx context.Context, token string) error
func GetLoginID(ctx context.Context, token string) (string, error)
func GetTokenInfo(ctx context.Context, token string) (*dt.TokenInfo, error)
func GetDevice(ctx context.Context, token string) (string, error)
func GetDeviceAndDeviceID(ctx context.Context, token string) (string, string, error)
func GetDeviceID(ctx context.Context, token string) (string, error)
func GetTokenCreateTime(ctx context.Context, token string) (int64, error)
func GetTokenTTL(ctx context.Context, token string) (int64, error)
func RenewTimeout(ctx context.Context, token string, timeout time.Duration) error
```

## Sessions and terminals

```go
func GetOnlineTerminalCount(ctx context.Context, loginID string) (int, error)
func GetOnlineTerminalCountByDevice(ctx context.Context, loginID, device string) (int, error)
func GetOnlineTerminalCountByDeviceAndDeviceID(ctx context.Context, loginID, device, deviceID string) (int, error)
func ForEachTerminal(ctx context.Context, loginID string, visitor dt.TerminalVisitor) error
func ForEachTerminalByDevice(ctx context.Context, loginID, device string, visitor dt.TerminalVisitor) error
func GetSession(ctx context.Context, loginID string) (*dt.Session, error)
func GetSessionByToken(ctx context.Context, token string) (*dt.Session, error)
func GetTokenValueListByLoginID(ctx context.Context, loginID string, checkAlive bool) ([]string, error)
func GetTokenValueListByDeviceAndDeviceID(ctx context.Context, loginID, device, deviceID string, checkAlive bool) ([]string, error)
func GetTokenValueListByDevice(ctx context.Context, loginID, device string, checkAlive bool) ([]string, error)
func GetTerminalListByLoginID(ctx context.Context, loginID string) ([]dt.TerminalInfo, error)
func GetTerminalListByLoginIDAndDevice(ctx context.Context, loginID, device string) ([]dt.TerminalInfo, error)
func GetTerminalInfoByToken(ctx context.Context, token string) (*dt.TerminalInfo, error)
func SetSessionValue(ctx context.Context, loginID, key string, value any) error
func GetSessionValue(ctx context.Context, loginID, key string) (any, bool, error)
func DeleteSessionValue(ctx context.Context, loginID, key string) error
func SetSessionValueByToken(ctx context.Context, token, key string, value any) error
func GetSessionValueByToken(ctx context.Context, token, key string) (any, bool, error)
func DeleteSessionValueByToken(ctx context.Context, token, key string) error
func GetTokenValueByLoginID(ctx context.Context, loginID string) (string, error)
func GetTokenValueByLoginIDAndDevice(ctx context.Context, loginID, device string) (string, error)
func SearchTokenValue(ctx context.Context, keyword string, start, size int) ([]string, error)
func SearchSessionId(ctx context.Context, keyword string, start, size int) ([]string, error)
```

## Permissions

```go
func CheckPermission(ctx context.Context, opts dt.PermissionOptions) error
func CheckPermissionsAnd(ctx context.Context, opts dt.PermissionOptions) error
func CheckPermissionsOr(ctx context.Context, opts dt.PermissionOptions) error
func AddPermissions(ctx context.Context, opts dt.PermissionOptions) error
func AddPermissionsByToken(ctx context.Context, token string, permissions []string) error
func RemovePermissions(ctx context.Context, opts dt.PermissionOptions) error
func RemovePermissionsByToken(ctx context.Context, token string, permissions []string) error
func GetPermissions(ctx context.Context, loginID string) ([]string, error)
func GetPermissionsByToken(ctx context.Context, token string) ([]string, error)
func HasPermission(ctx context.Context, loginID, permission string) bool
func HasPermissionByToken(ctx context.Context, token, permission string) bool
func HasPermissionsAnd(ctx context.Context, loginID string, permissions []string) bool
func HasPermissionsAndByToken(ctx context.Context, token string, permissions []string) bool
func HasPermissionsOr(ctx context.Context, loginID string, permissions []string) bool
func HasPermissionsOrByToken(ctx context.Context, token string, permissions []string) bool
func CheckPermissionByToken(ctx context.Context, token, permission string) error
func CheckPermissionAnd(ctx context.Context, loginID string, permissions []string) error
func CheckPermissionAndByToken(ctx context.Context, token string, permissions []string) error
func CheckPermissionOr(ctx context.Context, loginID string, permissions []string) error
func CheckPermissionOrByToken(ctx context.Context, token string, permissions []string) error
```

## Roles

```go
func CheckRole(ctx context.Context, opts dt.RoleOptions) error
func CheckRolesAnd(ctx context.Context, opts dt.RoleOptions) error
func CheckRolesOr(ctx context.Context, opts dt.RoleOptions) error
func AddRoles(ctx context.Context, opts dt.RoleOptions) error
func AddRolesByToken(ctx context.Context, token string, roles []string) error
func RemoveRoles(ctx context.Context, opts dt.RoleOptions) error
func RemoveRolesByToken(ctx context.Context, token string, roles []string) error
func GetRoles(ctx context.Context, loginID string) ([]string, error)
func GetRolesByToken(ctx context.Context, token string) ([]string, error)
func HasRole(ctx context.Context, loginID, role string) bool
func HasRoleByToken(ctx context.Context, token, role string) bool
func HasRolesAnd(ctx context.Context, loginID string, roles []string) bool
func HasRolesAndByToken(ctx context.Context, token string, roles []string) bool
func HasRolesOr(ctx context.Context, loginID string, roles []string) bool
func HasRolesOrByToken(ctx context.Context, token string, roles []string) bool
func CheckRoleByToken(ctx context.Context, token, role string) error
func CheckRoleAnd(ctx context.Context, loginID string, roles []string) error
func CheckRoleAndByToken(ctx context.Context, token string, roles []string) error
func CheckRoleOr(ctx context.Context, loginID string, roles []string) error
func CheckRoleOrByToken(ctx context.Context, token string, roles []string) error
```

## Disable state

```go
func Disable(ctx context.Context, opts dt.DisableOptions) error
func Untie(ctx context.Context, loginID string) error
func IsDisable(ctx context.Context, loginID string) bool
func CheckDisable(ctx context.Context, loginID string) error
func GetDisableInfo(ctx context.Context, loginID string) (*dt.DisableInfo, error)
func GetDisableTTL(ctx context.Context, loginID string) (int64, error)
func DisableService(ctx context.Context, opts dt.ServiceDisableOptions) error
func DisableServiceWithReason(ctx context.Context, loginID, service string, duration time.Duration, reason string) error
func DisableServiceLevel(ctx context.Context, loginID, service string, level int, duration time.Duration) error
func DisableServiceLevelWithReason(ctx context.Context, loginID, service string, level int, duration time.Duration, reason string) error
func DisableDevice(ctx context.Context, opts dt.DeviceDisableOptions) error
func DisableDeviceWithReason(ctx context.Context, loginID, device string, duration time.Duration, reason string) error
func DisableDeviceAndDeviceID(ctx context.Context, loginID, device, deviceID string, duration time.Duration) error
func DisableDeviceAndDeviceIDWithReason(ctx context.Context, loginID, device, deviceID string, duration time.Duration, reason string) error
func UntieService(ctx context.Context, loginID, service string) error
func IsDisableService(ctx context.Context, loginID, service string) bool
func IsDisableServiceLevel(ctx context.Context, loginID, service string, level int) bool
func CheckDisableService(ctx context.Context, loginID string, services ...string) error
func CheckDisableServiceLevel(ctx context.Context, loginID, service string, level int) error
func GetDisableServiceInfo(ctx context.Context, loginID, service string) (*dt.ServiceDisableInfo, error)
func GetDisableServiceTTL(ctx context.Context, loginID, service string) (int64, error)
func UntieDevice(ctx context.Context, loginID, device string) error
func UntieDeviceAndDeviceID(ctx context.Context, loginID, device, deviceID string) error
func IsDisableDevice(ctx context.Context, loginID, device string) bool
func IsDisableDeviceAndDeviceID(ctx context.Context, loginID, device, deviceID string) bool
func CheckDisableDevice(ctx context.Context, loginID, device string) error
func CheckDisableDeviceAndDeviceID(ctx context.Context, loginID, device, deviceID string) error
func GetDisableDeviceInfo(ctx context.Context, loginID, device string) (*dt.DeviceDisableInfo, error)
func GetDisableDeviceAndDeviceIDInfo(ctx context.Context, loginID, device, deviceID string) (*dt.DeviceDisableInfo, error)
func GetDisableDeviceTTL(ctx context.Context, loginID, device string) (int64, error)
func GetDisableDeviceAndDeviceIDTTL(ctx context.Context, loginID, device, deviceID string) (int64, error)
```

## Registration And Shutdown

| API | Behavior |
| --- | --- |
| `NewBuilder().Build()` | Register and return Auth |
| `NewBuilder().BuildInstance()` | Create an unregistered instance |
| `GetManager(authType...)` | Look up the current Manager; omission selects the default |
| `GetEventManager(authType...)` | Get the selected event manager |
| `SetAuth(auth) / SetManager(mgr)` | Return an error for invalid registration; successful replacement closes the old engine |
| `Close() / AuthType(name).Close()` | Close the default or selected current system |
| `CloseAll() / DeleteAllManager()` | Close all current registrations |
| `DeleteManager(authType...)` | Remove and close the selected Manager, reporting lookup errors |

Drain requests before closing or replacing engines. Registered and returned instances share an engine; declare shared components caller-owned with WithComponentOwnership. See [API and lifecycle](../design/api-lifecycle.md). Nonce, OAuth2, Ticket, and ShortKey use their own [services](../guide/security/advanced-features.md), rather than root auth methods.
