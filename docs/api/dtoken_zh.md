[English](dtoken.md) | 中文文档

# 根包 API 参考

本文 dt 表示 `github.com/Zany2/dtoken-go`。根包函数使用默认体系 `auth:`；`dt.AuthType(name)` 选择其他体系，其业务方法与根包函数和 Auth 实例签名一致。完整选项与实例说明见 [Auth API](auth_zh.md)。

原 global 包已移除；下列签名属于根包，登录、权限变更和封禁等操作使用类型化选项，不接受旧版末尾 authType 参数。

## 初始化与体系选择

<!-- dtoken-compile: root-registration -->
```go
package main

import (
    "context"
    "fmt"

    dt "github.com/Zany2/dtoken-go"
)

func main() {
    defer dt.CloseAll()
    dt.NewBuilder().IsPrintBanner(false).MustBuild()
    dt.NewBuilder().AuthType("user").IsPrintBanner(false).MustBuild()
    ctx := context.Background()
    token, err := dt.AuthType("user").Login(ctx, dt.LoginOptions{LoginID: "10001", Device: "web"})
    if err != nil {
        panic(err)
    }
    if err := dt.AuthType("user").CheckLogin(ctx, token); err != nil {
        panic(err)
    }
    fmt.Println(dt.AuthType("user").IsLogin(ctx, token))
}
```

Build 自动注册并返回 Auth；同名注册返回 ErrManagerAlreadyRegistered。BuildInstance 不注册，需要持有者单独 Close。AuthType 选择器每次操作查找注册表，缺失体系返回 ErrManagerNotFound，布尔检查返回 false，不回退默认体系。固定实例不会跟随替换。以下为函数声明索引，类型均从根包引用；ctx 使用当前请求上下文，time.Duration 参数与 Builder 的秒数配置不同。

## 登录与 Token 生命周期

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

## Session 与终端

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

## 权限

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

## 角色

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

## 封禁状态

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

## 注册与关闭

| API | 行为 |
| --- | --- |
| `NewBuilder().Build()` | 自动注册并返回 Auth |
| `NewBuilder().BuildInstance()` | 创建未注册实例 |
| `GetManager(authType...)` | 立即查找当前 Manager，省略名称时选择默认体系 |
| `GetEventManager(authType...)` | 获取所选体系的事件管理器 |
| `SetAuth(auth) / SetManager(mgr)` | 显式注册或替换，关闭旧引擎 |
| `BuildAndSetManager(builder, authType...)` | 高级构建注册入口，保留显式替换行为 |
| `Close() / AuthType(name).Close()` | 关闭默认或当前指定体系 |
| `CloseAll() / DeleteAllManager()` | 关闭全部当前注册实例 |
| `DeleteManager(authType...)` | 移除并关闭所选 Manager，返回查找错误 |

关闭或替换前先排空请求。注册表与构建返回的实例共用引擎；共享组件通过 WithComponentOwnership 声明为调用方持有。详见 [API 与生命周期](../design/api-lifecycle_zh.md)。Nonce、OAuth2、Ticket、ShortKey 使用各自的 [Service](../guide/security/advanced-features_zh.md)，不作为根认证门面方法。
