# Authentication Guide

[中文文档](../core/authentication_zh.md) | English

Here `dt` names the root package. `Build()` registers automatically; use `dt.XXX()` for the default namespace or `dt.AuthType("user").XXX()` for another namespace. Business method signatures match Auth methods. See [Auth API](../../api/auth.md).

## Basic Login

### Initialization

<!-- dtoken-compile: complete-example -->
```go
package main

import (
    dt "github.com/Zany2/dtoken-go"
    "context"

    "github.com/Zany2/dtoken-go/storage/memory"
)

func initDToken() {
    _, err := dt.NewBuilder().
        Storage(memory.NewStorage()).
        Build()
    if err != nil {
        panic(err)
    }
}

func main() {
    initDToken()
    defer dt.CloseAll()

    ctx := context.Background()
    token, _ := dt.Login(ctx, dt.LoginOptions{LoginID: "10001"})
    _ = token
}
```

### Simple Login

```go
ctx := context.Background()

// Default login
token, err := dt.Login(ctx, dt.LoginOptions{LoginID: "10001"})

// Login with device type
token, err = dt.Login(ctx, dt.LoginOptions{LoginID: "10001", Device: "web"})

// Login with device type and device ID
token, err = dt.Login(ctx, dt.LoginOptions{LoginID: "10001", Device: "app", DeviceID: "ios-iphone-15"})
```

### Login With Custom Timeout

```go
ctx := context.Background()

// Set a dedicated 2-hour TTL for the current token
token, err := dt.Login(ctx, dt.LoginOptions{LoginID: "10001", Timeout: 2*time.Hour})

// Combine timeout with device type and device ID
token, err = dt.Login(ctx, dt.LoginOptions{LoginID: "10001", Timeout: 2*time.Hour, Device: "web", DeviceID: "chrome-mac"})
```

### Renew Login By Existing Token

```go
ctx := context.Background()

err := dt.LoginByToken(ctx, token)
```

`LoginByToken()` renews token, session, and activity metadata asynchronously when the token is still valid.

Automatic renewal compares the storage TTL against its threshold without rounding to whole seconds; an unexpired token with less than one second remaining can still renew. Shared login invalidates older queued maintenance so it cannot overwrite newer activity. A limited token that expires before renewal does not emit a successful renewal event.

## Check Login Status

```go
ctx := context.Background()

// Boolean check
isLogin := dt.IsLogin(ctx, token)

// Returns an error if the token is invalid
err := dt.CheckLogin(ctx, token)
```

## Get Login Information

### Get Login ID

```go
ctx := context.Background()

loginID, err := dt.GetLoginID(ctx, token)
```

### Get Token Info

`GetTokenInfo` reads stored metadata; success does not prove the token is logged in. It does not check account/device bans, inactivity timeout, or session validity. Use `IsLogin` or `CheckLogin` for ordinary login validation. These methods and identity queries such as `GetLoginID` validate the token and current restrictions without reading or decoding the account Session on the normal path. Token records still require decoding; storage TTL determines their expiry. Renewal eligibility is checked by background maintenance.

Session queries, token-based permission/role checks, and `IntrospectToken` still require a valid account Session. Account-based access queries may use AccessProvider directly; a nil provider result falls back to Session. Inactive-timeout cleanup also loads the Session. Deleting or corrupting a Session alone no longer revokes a token for ordinary login checks; use logout or kickout APIs to revoke credentials. Background maintenance does not renew or persist activity if the Session is invalid.

```go
ctx := context.Background()

info, err := dt.GetTokenInfo(ctx, token)
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

device, err := dt.GetDevice(ctx, token)
deviceID, err := dt.GetDeviceID(ctx, token)
createTime, err := dt.GetTokenCreateTime(ctx, token)
ttl, err := dt.GetTokenTTL(ctx, token)
```

## Logout

### Logout By Token

```go
ctx := context.Background()

err := dt.LogoutByToken(ctx, token)
```

### Logout By Account Dimension

```go
ctx := context.Background()

// Logout all terminals of one account
err := dt.LogoutByLoginID(ctx, "10001")

// Logout all terminals of one device type
err = dt.LogoutByDevice(ctx, "10001", "web")

// Logout a single terminal by device type + device ID
err = dt.LogoutByDeviceAndDeviceID(ctx, "10001", "app", "ios-iphone-15")
```

## Kickout

### Kickout By Token

```go
ctx := context.Background()

err := dt.KickoutByToken(ctx, token)
```

### Kickout By Account Dimension

```go
ctx := context.Background()

err := dt.KickoutByLoginID(ctx, "10001")
err = dt.KickoutByDevice(ctx, "10001", "web")
err = dt.KickoutByDeviceAndDeviceID(ctx, "10001", "app", "ios-iphone-15")
```

Kicked tokens are not deleted immediately. They are marked with the `kickout` state and will fail later checks with the corresponding error.

## Replace

### Replace By Token

```go
ctx := context.Background()

err := dt.ReplaceByToken(ctx, token)
```

### Replace By Account Dimension

```go
ctx := context.Background()

err := dt.ReplaceByLoginID(ctx, "10001")
err = dt.ReplaceByDevice(ctx, "10001", "web")
err = dt.ReplaceByDeviceAndDeviceID(ctx, "10001", "app", "ios-iphone-15")
```

Replaced tokens are marked with the `replaced` state, which is useful for "new login overrides old login" scenarios.

## Online Terminal Statistics

```go
ctx := context.Background()

count, err := dt.GetOnlineTerminalCount(ctx, "10001")
webCount, err := dt.GetOnlineTerminalCountByDevice(ctx, "10001", "web")
singleCount, err := dt.GetOnlineTerminalCountByDeviceAndDeviceID(ctx, "10001", "app", "ios-iphone-15")
```

## Login Configuration

The configuration fragments below are alternative builds. Do not build the same registered namespace repeatedly in one process; use BuildInstance for independent instances with the same name.

### Concurrent Login

```go
auth, err := dt.NewBuilder().
    Storage(memory.NewStorage()).
    IsConcurrent(true).
    Build()
if err != nil {
    panic(err)
}
defer auth.Close()
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
defer auth.Close()
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
defer auth.Close()
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
defer auth.Close()
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
defer auth.Close()
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
defer auth.Close()
```

## Related Documentation

- [Quick Start](../../tutorial/quick-start.md)
- [Permission Management](../core/permission.md)
- [Annotation Guide](../integration/annotation.md)
- [JWT Guide](../security/jwt.md)
