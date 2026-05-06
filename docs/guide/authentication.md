# Authentication Guide

[中文文档](authentication_zh.md) | English

## Basic Login

### Initialization

```go
package main

import (
    "context"

    "github.com/Zany2/dtoken-go/com/storage/memory"
    "github.com/Zany2/dtoken-go/core/builder"
    "github.com/Zany2/dtoken-go/dtoken"
)

func initDToken() {
    dtoken.SetManager(
        builder.NewBuilder().
            SetStorage(memory.NewStorage()).
            Build(),
    )
}

func main() {
    ctx := context.Background()
    token, _ := dtoken.Login(ctx, "10001")
    _ = token
}
```

### Simple Login

```go
ctx := context.Background()

// Default login
token, err := dtoken.Login(ctx, "10001")

// Login with device type
token, err = dtoken.Login(ctx, "10001", "web")

// Login with device type and device ID
token, err = dtoken.Login(ctx, "10001", "app", "ios-iphone-15")
```

### Login With Custom Timeout

```go
ctx := context.Background()

// Set a dedicated 2-hour TTL for the current token
token, err := dtoken.LoginWithTimeout(ctx, "10001", 2*time.Hour)

// Combine timeout with device type and device ID
token, err = dtoken.LoginWithTimeout(ctx, "10001", 2*time.Hour, "web", "chrome-mac")
```

### Renew Login By Existing Token

```go
ctx := context.Background()

err := dtoken.LoginByToken(ctx, token)
```

`LoginByToken()` renews token, session, and activity metadata asynchronously when the token is still valid.

## Check Login Status

```go
ctx := context.Background()

// Boolean check
isLogin := dtoken.IsLogin(ctx, token)

// Returns an error if the token is invalid
err := dtoken.CheckLogin(ctx, token)
```

## Get Login Information

### Get Login ID

```go
ctx := context.Background()

loginID, err := dtoken.GetLoginID(ctx, token)
```

### Get Token Info

```go
ctx := context.Background()

info, err := dtoken.GetTokenInfo(ctx, token)
if err == nil {
    fmt.Println("Auth Type:", info.AuthType)
    fmt.Println("Login ID:", info.LoginID)
    fmt.Println("Device:", info.Device)
    fmt.Println("Device ID:", info.DeviceId)
    fmt.Println("Create Time:", info.CreateTime)
}
```

Current `TokenInfo` includes:

- `AuthType`
- `LoginID`
- `Device`
- `DeviceId`
- `CreateTime`

### Get Additional Token Fields

```go
ctx := context.Background()

device, err := dtoken.GetDevice(ctx, token)
deviceId, err := dtoken.GetDeviceId(ctx, token)
createTime, err := dtoken.GetTokenCreateTime(ctx, token)
ttl, err := dtoken.GetTokenTTL(ctx, token)
```

## Logout

### Logout By Token

```go
ctx := context.Background()

err := dtoken.Logout(ctx, token)
```

### Logout By Account Dimension

```go
ctx := context.Background()

// Logout all terminals of one account
err := dtoken.LogoutByLoginID(ctx, "10001")

// Logout all terminals of one device type
err = dtoken.LogoutByDevice(ctx, "10001", "web")

// Logout a single terminal by device type + device ID
err = dtoken.LogoutByDeviceAndDeviceId(ctx, "10001", "app", "ios-iphone-15")
```

## Kickout

### Kickout By Token

```go
ctx := context.Background()

err := dtoken.Kickout(ctx, token)
```

### Kickout By Account Dimension

```go
ctx := context.Background()

err := dtoken.KickoutByLoginID(ctx, "10001")
err = dtoken.KickoutByDevice(ctx, "10001", "web")
err = dtoken.KickoutByDeviceAndDeviceId(ctx, "10001", "app", "ios-iphone-15")
```

Kicked tokens are not deleted immediately. They are marked with the `kickout` state and will fail later checks with the corresponding error.

## Replace

### Replace By Token

```go
ctx := context.Background()

err := dtoken.Replace(ctx, token)
```

### Replace By Account Dimension

```go
ctx := context.Background()

err := dtoken.ReplaceByLoginID(ctx, "10001")
err = dtoken.ReplaceByDevice(ctx, "10001", "web")
err = dtoken.ReplaceByDeviceAndDeviceId(ctx, "10001", "app", "ios-iphone-15")
```

Replaced tokens are marked with the `replaced` state, which is useful for "new login overrides old login" scenarios.

## Online Terminal Statistics

```go
ctx := context.Background()

count, err := dtoken.GetOnlineTerminalCount(ctx, "10001")
webCount, err := dtoken.GetOnlineTerminalCountByDevice(ctx, "10001", "web")
singleCount, err := dtoken.GetOnlineTerminalCountByDeviceAndDeviceId(ctx, "10001", "app", "ios-iphone-15")
```

## Login Configuration

### Concurrent Login

```go
dtoken.SetManager(
    builder.NewBuilder().
        SetStorage(memory.NewStorage()).
        IsConcurrent(true).
        Build(),
)
```

`IsConcurrent(true)` allows multiple active terminals for the same account.  
`IsConcurrent(false)` lets the new login handle older terminals according to the current strategy.

### Shared Token

```go
dtoken.SetManager(
    builder.NewBuilder().
        SetStorage(memory.NewStorage()).
        IsConcurrent(true).
        IsShare(true).
        Build(),
)
```

`IsShare(true)` reuses an existing token during concurrent login.  
When `IsConcurrent(true)` and `IsShare(false)` are both set, each login creates a new token.

### Max Login Count

```go
dtoken.SetManager(
    builder.NewBuilder().
        SetStorage(memory.NewStorage()).
        IsConcurrent(true).
        IsShare(false).
        MaxLoginCount(3).
        Build(),
)
```

When `MaxLoginCount > 0`, older terminals beyond the limit are removed automatically.

## Auto Renew

### Configuration

```go
dtoken.SetManager(
    builder.NewBuilder().
        SetStorage(memory.NewStorage()).
        Timeout(24 * 60 * 60).
        AutoRenew(true).
        RenewMaxRefresh(30 * 60).
        RenewInterval(60).
        Build(),
)
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
dtoken.SetManager(
    builder.NewBuilder().
        SetStorage(memory.NewStorage()).
        Timeout(24 * 60 * 60).
        ActiveTimeout(30 * 60).
        Build(),
)
```

### How It Works

When `ActiveTimeout` is enabled:

1. The token gets an activity timestamp at login
2. Each authentication check verifies whether the activity timeout has been exceeded
3. If still active, the activity timestamp is refreshed asynchronously
4. If expired, the token fails with a not-logged-in style error

`Timeout` is the absolute expiration time, while `ActiveTimeout` is the inactivity window.

## Complete Configuration Example

```go
dtoken.SetManager(
    builder.NewBuilder().
        SetStorage(memory.NewStorage()).
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
        Build(),
)
```

## Related Documentation

- [Quick Start](../tutorial/quick-start.md)
- [Permission Management](permission.md)
- [Annotation Guide](annotation.md)
- [JWT Guide](jwt.md)
