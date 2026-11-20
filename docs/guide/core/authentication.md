# Authentication Guide

[中文文档](../core/authentication_zh.md) | English

This page uses `global` for registered-instance calls and `dt` for the root package. Build Auth and call `global.SetAuth(auth)` first; omitted auth types select the default `auth:` namespace. See [Auth API](../../api/auth.md) for instance calls and parameter differences.

## Basic Login

### Initialization

<!-- dtoken-compile: complete-example -->
```go
package main

import (
    dt "github.com/Zany2/dtoken-go"
    global "github.com/Zany2/dtoken-go/global"
    "context"

    "github.com/Zany2/dtoken-go/storage/memory"
)

func initDToken() {
    auth, err := dt.NewBuilder().
        Storage(memory.NewStorage()).
        Build()
    if err != nil {
        panic(err)
    }
    global.SetAuth(auth)
}

func main() {
    initDToken()
    defer global.DeleteAllManager()

    ctx := context.Background()
    token, _ := global.Login(ctx, "10001")
    _ = token
}
```

### Simple Login

```go
ctx := context.Background()

// Default login
token, err := global.Login(ctx, "10001")

// Login with device type
token, err = global.Login(ctx, "10001", "web")

// Login with device type and device ID
token, err = global.Login(ctx, "10001", "app", "ios-iphone-15")
```

### Login With Custom Timeout

```go
ctx := context.Background()

// Set a dedicated 2-hour TTL for the current token
token, err := global.LoginWithTimeout(ctx, "10001", 2*time.Hour)

// Combine timeout with device type and device ID
token, err = global.LoginWithTimeout(ctx, "10001", 2*time.Hour, "web", "chrome-mac")
```

### Renew Login By Existing Token

```go
ctx := context.Background()

err := global.LoginByToken(ctx, token)
```

`LoginByToken()` renews token, session, and activity metadata asynchronously when the token is still valid.

Automatic renewal compares the storage TTL against its threshold without rounding to whole seconds; an unexpired token with less than one second remaining can still renew. Shared login invalidates older queued maintenance so it cannot overwrite newer activity. A limited token that expires before renewal does not emit a successful renewal event.

## Check Login Status

```go
ctx := context.Background()

// Boolean check
isLogin := global.IsLogin(ctx, token)

// Returns an error if the token is invalid
err := global.CheckLogin(ctx, token)
```

## Get Login Information

### Get Login ID

```go
ctx := context.Background()

loginID, err := global.GetLoginID(ctx, token)
```

### Get Token Info

`GetTokenInfo` reads stored metadata; success does not prove the token is logged in. It does not check account/device bans, inactivity timeout, or session validity. Use `CheckLogin` or `IntrospectToken` for login validation. Identity queries such as `GetLoginID` validate both the token and the account session.

```go
ctx := context.Background()

info, err := global.GetTokenInfo(ctx, token)
if err == nil {
    fmt.Println("Auth Type:", info.AuthType)
    fmt.Println("Login ID:", info.LoginID)
    fmt.Println("Device:", info.Device)
    fmt.Println("Device ID:", info.DeviceID)
    fmt.Println("Create Time:", info.CreateTime)
    fmt.Println("Timeout:", info.Timeout)
    fmt.Println("Active Timeout:", info.ActiveTimeout)
    fmt.Println("Extra:", info.Extra)
}
```

Current `TokenInfo` includes:

- `AuthType`
- `LoginID`
- `Device`
- `DeviceID`
- `CreateTime`
- `Timeout` (seconds)
- `ActiveTimeout` (seconds; `0` means inherit the manager setting)
- `Extra`

### Get Additional Token Fields

```go
ctx := context.Background()

device, err := global.GetDevice(ctx, token)
deviceID, err := global.GetDeviceID(ctx, token)
createTime, err := global.GetTokenCreateTime(ctx, token)
ttl, err := global.GetTokenTTL(ctx, token)
```

## Logout

### Logout By Token

```go
ctx := context.Background()

err := global.Logout(ctx, token)
```

### Logout By Account Dimension

```go
ctx := context.Background()

// Logout all terminals of one account
err := global.LogoutByLoginID(ctx, "10001")

// Logout all terminals of one device type
err = global.LogoutByDevice(ctx, "10001", "web")

// Logout a single terminal by device type + device ID
err = global.LogoutByDeviceAndDeviceID(ctx, "10001", "app", "ios-iphone-15")
```

## Kickout

### Kickout By Token

```go
ctx := context.Background()

err := global.Kickout(ctx, token)
```

### Kickout By Account Dimension

```go
ctx := context.Background()

err := global.KickoutByLoginID(ctx, "10001")
err = global.KickoutByDevice(ctx, "10001", "web")
err = global.KickoutByDeviceAndDeviceID(ctx, "10001", "app", "ios-iphone-15")
```

Kicked tokens are not deleted immediately. They are marked with the `kickout` state and will fail later checks with the corresponding error.

## Replace

### Replace By Token

```go
ctx := context.Background()

err := global.Replace(ctx, token)
```

### Replace By Account Dimension

```go
ctx := context.Background()

err := global.ReplaceByLoginID(ctx, "10001")
err = global.ReplaceByDevice(ctx, "10001", "web")
err = global.ReplaceByDeviceAndDeviceID(ctx, "10001", "app", "ios-iphone-15")
```

Replaced tokens are marked with the `replaced` state, which is useful for "new login overrides old login" scenarios.

## Online Terminal Statistics

```go
ctx := context.Background()

count, err := global.GetOnlineTerminalCount(ctx, "10001")
webCount, err := global.GetOnlineTerminalCountByDevice(ctx, "10001", "web")
singleCount, err := global.GetOnlineTerminalCountByDeviceAndDeviceID(ctx, "10001", "app", "ios-iphone-15")
```

## Login Configuration

### Concurrent Login

```go
auth, err := dt.NewBuilder().
    Storage(memory.NewStorage()).
    IsConcurrent(true).
    Build()
if err != nil {
    panic(err)
}
global.SetAuth(auth)
```

`IsConcurrent(true)` allows multiple active terminals for the same account.  
`IsConcurrent(false)` lets the new login handle older terminals according to the current strategy.

### Shared Token

```go
auth, err := dt.NewBuilder().
    Storage(memory.NewStorage()).
    IsConcurrent(true).
    IsShare(true).
    Build()
if err != nil {
    panic(err)
}
global.SetAuth(auth)
```

`IsShare(true)` reuses an existing token during concurrent login, but only within the same device dimension.  
When `IsConcurrent(true)` and `IsShare(false)` are both set, each login creates a new token.

### Max Login Count

```go
auth, err := dt.NewBuilder().
    Storage(memory.NewStorage()).
    IsConcurrent(true).
    IsShare(false).
    MaxLoginCount(3).
    Build()
if err != nil {
    panic(err)
}
global.SetAuth(auth)
```

When `MaxLoginCount > 0`, older terminals beyond the limit are removed automatically.

## Auto Renew

### Configuration

```go
auth, err := dt.NewBuilder().
    Storage(memory.NewStorage()).
    Timeout(24 * 60 * 60).
    AutoRenew(true).
    RenewMaxRefresh(30 * 60).
    RenewInterval(60).
    Build()
if err != nil {
    panic(err)
}
global.SetAuth(auth)
```

### How It Works

In the current implementation, auto-renew does not refresh on every `IsLogin()` call unconditionally. Instead it:

1. Validates the token
2. Checks whether the account is disabled
3. Checks `ActiveTimeout`
4. Renews asynchronously only when `AutoRenew=true` and the current TTL matches the `RenewMaxRefresh` and `RenewInterval` conditions

This avoids unnecessary renewals on high-frequency traffic.

## Active Timeout

### Configuration

```go
auth, err := dt.NewBuilder().
    Storage(memory.NewStorage()).
    Timeout(24 * 60 * 60).
    ActiveTimeout(30 * 60).
    Build()
if err != nil {
    panic(err)
}
global.SetAuth(auth)
```

### How It Works

When `ActiveTimeout` is enabled:

1. The token gets an activity timestamp at login
2. Each authentication check verifies whether the activity timeout has been exceeded
3. If still active, the activity timestamp is refreshed asynchronously
4. If expired, the token is marked as `active-timeout` and later checks keep returning the corresponding error

`Timeout` is the absolute expiration time, while `ActiveTimeout` is the inactivity window.

## Complete Configuration Example

```go
auth, err := dt.NewBuilder().
    Storage(memory.NewStorage()).
    TokenName("dtoken").
    Timeout(24 * 60 * 60).
    ActiveTimeout(30 * 60).
    AutoRenew(true).
    RenewMaxRefresh(30 * 60).
    RenewInterval(60).
    IsConcurrent(true).
    IsShare(false).
    MaxLoginCount(3).
    IsReadHeader(true).
    Build()
if err != nil {
    panic(err)
}
global.SetAuth(auth)
```

## Related Documentation

- [Quick Start](../../tutorial/quick-start.md)
- [Permission Management](../core/permission.md)
- [Annotation Guide](../integration/annotation.md)
- [JWT Guide](../security/jwt.md)
