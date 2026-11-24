English | [中文文档](auto-renew_zh.md)

# Auto-Renew Design

Here auth is a root-built Auth, dt names the root package, gindt the Gin adapter, memory storage/memory, and adapter contract. Bind HTTP requests through WithAuth(auth) first. Internal maintenance lives in internal/engine/internal_maintenance.go and internal_lifecycle.go; applications do not import internal.

## Design Goals

The goals of auto-renew are:

- keep active users logged in without frequent re-authentication
- avoid heavy synchronous work on every login check
- control renew frequency with threshold and throttling
- improve stability under high concurrency with a worker pool

## Core Design

### Asynchronous Renew Strategy

Ordinary login and identity checks (`IsLogin`, `CheckLogin`, `GetLoginID`, and device queries) validate the stored token, account/device disable state, and inactivity timeout without loading the account Session on the normal path. Token records still require decoding. Session-dependent queries, access checks, and inactive-timeout cleanup load and validate the Session when needed.

`prepareLoginMaintenance` coalesces requests for the same token lifecycle into one pending task. Configuration determines whether to schedule automatic-renew eligibility checks or activity maintenance; token TTL, renewal threshold, and interval checks run in the worker. If inactivity tracking is enabled, admission records the latest successful request time. A worker that finds no renewal due and no activity to refresh exits without loading token or Session records.

Deleting or corrupting a Session alone no longer invalidates ordinary token checks. Use logout or kickout APIs to revoke credentials. Session-dependent operations still reject invalid Sessions, and background maintenance skips renewal and activity persistence when it cannot load a valid Session.

Inactivity checks use the later of the valid persisted activity timestamp and the accepted activity queued for the same lifecycle in this Manager. Login validation, terminal queries, and expired-terminal cleanup follow this rule, so pool delays do not discard accepted activity. Cancelled tasks and other lifecycles cannot contribute activity; missing or malformed persisted markers and storage errors still fail validation. Pending activity is local to this Manager; other instances see it only after persistence.

`submitAsync` prefers the configured pool and falls back to a tracked goroutine when no pool is available or submission fails. A closing Manager rejects new tasks.

Validation does not wait for background maintenance to finish. The worker may start before the caller receives its result; custom pools that execute inline or block submission can still delay the call. Task coalescing limits concurrent work for one lifecycle, but a later request can schedule another eligibility check after completion.

## Workflow

### Synchronous Part

```text
1. Load and decode the token record; reject expired or retired credentials
2. Check account/device disable state and ActiveTimeout
3. Load Session only if required by the operation or inactive-timeout cleanup
4. Reserve or update maintenance for the current token lifecycle
5. Submit after releasing any account lock, then return validation success
```

### Asynchronous Part

```text
One maintenance task
  ↓
1. Recheck task generation under the account lock; evaluate renewal TTL,
   threshold, and interval; exit if neither renewal nor activity refresh is needed
  ↓
2. Reload the token, verify its lifecycle and disable state, then load Session
  ↓
3. If renewal is due, extend the token using its own timeout,
   preserve the longer Session TTL, and write the renew interval marker
  ↓
4. If activity tracking is enabled, persist the latest request activity time
  ↓
5. Release the account lock; emit EventRenew only after successful renewal
```

Obsolete tasks cannot maintain a later login that reuses the same token value. Activity uses the recorded request time, rather than the worker execution time.

`LoginByToken` requests forced maintenance after validating the existing login: it bypasses `AutoRenew`, threshold, and interval eligibility checks. It is asynchronous; use `RenewTimeout` when the caller needs a synchronous renewal result.

## Renew Trigger Conditions

In the current implementation, auto-renew typically requires all of the following:

- `AutoRenew = true`
- `Timeout > 0`
- current token TTL is greater than 0
- `TTL <= RenewMaxRefresh`, or `RenewMaxRefresh <= 0`
- the renew-throttle condition of `RenewInterval` is not hit

The throttle normally uses the configured `RenewInterval`. If a token's positive lifetime is no longer than that interval, the marker instead lasts half that lifetime. For example, a 60-second token with `RenewInterval = 300` uses a 30-second interval. This rule applies to initial login, shared login, automatic and forced renewal, and `RenewTimeout`; manual renewal resets the marker using the new lifetime. The calculation uses duration precision, including subsecond lifetimes. Disabled intervals and unlimited token lifetimes retain their existing semantics. Renewal still requires an eligible request and timely execution before the token expires.

This means:

- renew does not happen on every request
- renew is more likely when the token is closer to expiry
- frequent requests do not endlessly refresh the same token

## Trigger Timing

Any scenario that goes through login validation may trigger auto-renew:

### 1. Middleware Authentication

```go
r.Use(gindt.AuthMiddleware(ctx))
```

### 2. Annotation-Style Login Check

```go
router.GET("/profile",
    gindt.CheckLoginMiddleware(ctx, nil, handleAuthFail),
    handleProfile,
)
```

### 3. Manual Login Check

```go
auth.IsLogin(ctx, token)
auth.CheckLogin(ctx, token)
```

### 4. Fetching Checked Login Information

```go
auth.GetLoginID(ctx, token)
auth.GetDevice(ctx, token)
```

`auth.GetTokenInfo` only loads the stored Token information and does not run login validation, so it does not trigger auto-renew.

## Configuration Options

### Enable Auto-Renew

```go
auth, err := dt.NewBuilder().
    Storage(memory.NewStorage()).
    Timeout(86400).
    AutoRenew(true).
    Build()
if err != nil {
    panic(err)
}
defer auth.Close()
```

### Set Renew Trigger Threshold

```go
auth, err := dt.NewBuilder().
    Storage(memory.NewStorage()).
    Timeout(86400).
    AutoRenew(true).
    RenewMaxRefresh(3600). // only renew within the last hour
    Build()
if err != nil {
    panic(err)
}
defer auth.Close()
```

### Set Minimum Renew Interval

```go
auth, err := dt.NewBuilder().
    Storage(memory.NewStorage()).
    Timeout(86400).
    AutoRenew(true).
    RenewMaxRefresh(3600).
    RenewInterval(300). // at most once every 5 minutes for the same token
    Build()
if err != nil {
    panic(err)
}
defer auth.Close()
```

### Combine with Active Timeout

```go
auth, err := dt.NewBuilder().
    Storage(memory.NewStorage()).
    Timeout(86400).
    ActiveTimeout(1800).
    AutoRenew(true).
    RenewMaxRefresh(3600).
    RenewInterval(300).
    Build()
if err != nil {
    panic(err)
}
defer auth.Close()
```

**Effect**:
- active users are auto-renewed when close to expiry
- inactive users are kicked out after `ActiveTimeout`
- renew work is not executed without limit under high request frequency

## Concurrency Safety

### Storage Interface Is Context-Aware

```go
type Storage interface {
    Set(ctx context.Context, key string, value any, expiration time.Duration) error
    Get(ctx context.Context, key string) (any, error)
    Delete(ctx context.Context, keys ...string) error
    Exists(ctx context.Context, key string) bool
    Expire(ctx context.Context, key string, expiration time.Duration) error
    TTL(ctx context.Context, key string) (time.Duration, error)
    Ping(ctx context.Context) error
}
```

### Worker Pool Support

When auto-renew is enabled, renew work prefers a renew task pool:

- `pool/ants`
- `adapter.Pool`
- `Builder.SetPool(...)`

If no pool is explicitly provided, a default renew pool may still be created internally in suitable cases.

## Renew Failure Handling

### Strategy

Async renew failure does not directly block the current request:

1. renewal failure does not change the successful validation result
2. renew failure only affects the extension of future lifetime
3. renew can still be retried on the next eligible request

### Impact Scope

- it does not directly fail the current request
- it may leave the token unextended
- if renew keeps failing, the token will eventually expire by its original TTL

## Best Practices

### Recommended Production Configuration

```go
auth, err := dt.NewBuilder().
    Storage(redisStorage).
    Timeout(86400).
    ActiveTimeout(1800).
    AutoRenew(true).
    RenewMaxRefresh(3600).
    RenewInterval(300).
    Build()
if err != nil {
    panic(err)
}
defer auth.Close()
```

### Recommended Development Configuration

```go
auth, err := dt.NewBuilder().
    Storage(memory.NewStorage()).
    Timeout(7200).
    AutoRenew(true).
    Build()
if err != nil {
    panic(err)
}
defer auth.Close()
```

### Security-First Configuration

```go
auth, err := dt.NewBuilder().
    Storage(redisStorage).
    Timeout(1800).
    AutoRenew(false).
    Build()
if err != nil {
    panic(err)
}
defer auth.Close()
```

## Next Steps

- [Architecture Design](architecture.md)
- [Modular Design](modular.md)
- [DToken API Documentation](../api/dtoken.md)
