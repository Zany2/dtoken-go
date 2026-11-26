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

`IsLogin`, `CheckLogin`, and basic identity queries validate credential presence, state, and enabled inactivity timeout without implicitly reading account or device restrictions. A normal foreground check of a new-format token with inactivity checks disabled uses one token GET, without loading Session, Extra, or the general Codec. Background maintenance reads are separate.

For restricted access, explicitly set `CheckDisable: true` in HTTP checks or call the appropriate `CheckDisable`, `CheckDisableDevice`, and `CheckDisableDeviceAndDeviceID` operations at the business entry. Permission/role checks and token-based Session access do not imply restriction checks. Introspection, new login, and refresh issuance continue to enforce restrictions.

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

`GetTokenInfo` reads stored metadata; success does not prove the token is logged in. It does not check account/device bans, inactivity timeout, or session validity. Use `IsLogin` or `CheckLogin` for ordinary login validation. These methods and identity queries such as `GetLoginID` validate credentials and enabled inactivity timeout without reading Session or restriction records on the normal path. Access credentials use fixed-field parsing only; legacy Token payloads must be converted offline and are rejected at runtime without Codec decoding. Storage TTL determines absolute expiry, and background maintenance checks renewal eligibility.

Operations requiring Session content and `IntrospectToken` still validate the account Session. Permission/role queries by account or token prefer AccessProvider; Session is loaded only when the provider is absent or returns nil. Inactive-timeout cleanup also loads Session. Deleting or corrupting a Session alone no longer revokes a token for ordinary login checks; use logout or kickout APIs to revoke credentials. Automatic renewal checks Session TTL and stops if the key is missing, without decoding its contents. Activity-only maintenance does not depend on Session existence or readable contents; it still rechecks credential lifecycle and restrictions.

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

### Retirement Operation Costs

Account, device-type and concrete-device operations load the Session once under the account lock and select the target terminals. Each normally valid primary credential is read once and reused for ownership checks, liveness checks and Extra deletion, without reading the Extra payload. Kickout and replacement use the matched terminal directly, avoiding repeated Session scans and copies of terminal, role or permission lists.

Logout deletes selected primary records and their Extra keys in a batch. Kickout and replacement write individual state markers, preferring the current remaining TTL. Renewal and activity keys are removed in one batch; linked refresh credentials are still read and checked before revocation. The changed Session is saved once, or deleted if empty, and events are dispatched after releasing the account lock.

The single primary read applies to normal account/device cleanup, not to the total storage cost of retirement. Direct `Logout`, `Kickout`, and `Replace` validate credentials and enabled inactivity tracking before entering the account lock, then load Session for cleanup under that lock. The normal direct path reads and decodes Session once instead of twice, without loading Extra. Primary-record capture, validation, lock-entry lifecycle recheck and cleanup ownership checks remain separate; this is not a one-read retirement operation. Missing Sessions still allow explicit credential cleanup, while corrupt or foreign Sessions and storage failures return errors. Suspected inactivity timeout still loads context and rechecks under the lock before changing state, preserving its cause and events. Inactive credentials, foreign ownership and abnormal refresh bindings can require additional reads.

These steps run sequentially without a multi-key transaction. A storage failure returns an error and can leave some credentials already retired. The Session index is persisted only after cleanup succeeds, allowing another account/device cleanup attempt. Retrying kickout or replacement does not emit duplicate events for state markers already written by the failed operation.

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

### Session Writes During Login

A new login combines expired-entry pruning, overflow eviction or replacement, and terminal insertion into one final Session save. It no longer serializes and writes the entire session after each removal. Shared login saves no Session content when the list is unchanged, and saves once after necessary pruning or eviction. Token states, auxiliary keys, and TTL maintenance still have separate costs; these counts cover Session content writes only.

These counts describe one internal login phase. Refresh-token rotation also retires the old terminal, whose logout phase can write the Session separately.

Ordinary replacement preserves account data, historical counts, and a longer existing Session TTL. If on-demand pruning finds every old terminal invalid, the existing reset behavior remains: create a fresh Session with new account data and history, using this login's lifetime. Successful reset overwrites the old session without an intermediate Session deletion. Destroy, create, and login events still run in order after releasing the account lock.

Login-triggered replacement, overflow eviction and eviction during shared login reuse the primary record checked under the account lock for state writes and Extra deletion. These steps do not reread the primary or load the Extra payload. Occupancy validation still has its own read: a normal candidate is checked once, then rechecked if selected for retirement. This does not mean the whole login performs only one GET. State markers use the current remaining TTL, falling back to the known credential's lifetime when TTL is unavailable.

Early failures attempt to persist completed list cleanup; failed token persistence still removes the terminal appended by this login. Failure paths can require additional writes. Already retired credentials are not restored, and the operation does not provide a transaction across keys.

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
2. Synchronously checks account and device restrictions only when explicitly requested by an access check
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
