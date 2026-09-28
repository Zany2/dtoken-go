English | [中文文档](auto-renew_zh.md)

# Auto-Renew Design

## Design Goals

The goals of auto-renew are:

- keep active users logged in without frequent re-authentication
- avoid heavy synchronous work on every login check
- control renew frequency with threshold and throttling
- improve stability under high concurrency with a worker pool

## Core Design

### Asynchronous Renew Strategy

Login validation checks the token lifecycle, Session, account/device disable state, and inactivity timeout before scheduling maintenance.

`prepareLoginMaintenance` coalesces requests for the same token lifecycle into one pending task. If inactivity tracking is enabled, it records the latest successful request time. Without inactivity tracking, a task is submitted only when automatic renewal is due.

`submitAsync` prefers the configured pool and falls back to a tracked goroutine when no pool is available or submission fails. A closing Manager rejects new tasks.

## Workflow

### Synchronous Part

```text
1. Load and validate TokenInfo and Session
2. Check account/device disable state and ActiveTimeout
3. Reserve or update maintenance for the current token lifecycle
4. Submit after releasing the account lock, then return validation success
```

### Asynchronous Part

```text
One maintenance task
  ↓
1. Recheck task generation and token lifecycle under the account lock
  ↓
2. Reload Session and recheck disable state
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
annotation.GET("/profile",
    gindt.CheckLoginMiddleware(ctx, handleProfile, handleAuthFail))
```

### 3. Manual Login Check

```go
dtoken.IsLogin(ctx, token)
dtoken.CheckLogin(ctx, token)
```

### 4. Fetching Checked Login Information

```go
dtoken.GetLoginID(ctx, token)
dtoken.GetDevice(ctx, token)
```

`dtoken.GetTokenInfo` only loads the stored Token information and does not run login validation, so it does not trigger auto-renew.

## Configuration Options

### Enable Auto-Renew

```go
defaults.NewBuilder().
    SetStorage(memory.NewStorage()).
    Timeout(86400).
    AutoRenew(true).
    Build()
```

### Set Renew Trigger Threshold

```go
defaults.NewBuilder().
    SetStorage(memory.NewStorage()).
    Timeout(86400).
    AutoRenew(true).
    RenewMaxRefresh(3600). // only renew within the last hour
    Build()
```

### Set Minimum Renew Interval

```go
defaults.NewBuilder().
    SetStorage(memory.NewStorage()).
    Timeout(86400).
    AutoRenew(true).
    RenewMaxRefresh(3600).
    RenewInterval(300). // at most once every 5 minutes for the same token
    Build()
```

### Combine with Active Timeout

```go
defaults.NewBuilder().
    SetStorage(memory.NewStorage()).
    Timeout(86400).
    ActiveTimeout(1800).
    AutoRenew(true).
    RenewMaxRefresh(3600).
    RenewInterval(300).
    Build()
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

- `com/pool/ants`
- `adapter.Pool`
- `Builder.SetPool(...)`

If no pool is explicitly provided, a default renew pool may still be created internally in suitable cases.

## Renew Failure Handling

### Strategy

Async renew failure does not directly block the current request:

1. the current login validation result has already been returned
2. renew failure only affects the extension of future lifetime
3. renew can still be retried on the next eligible request

### Impact Scope

- it does not directly fail the current request
- it may leave the token unextended
- if renew keeps failing, the token will eventually expire by its original TTL

## Best Practices

### Recommended Production Configuration

```go
defaults.NewBuilder().
    SetStorage(redisStorage).
    Timeout(86400).
    ActiveTimeout(1800).
    AutoRenew(true).
    RenewMaxRefresh(3600).
    RenewInterval(300).
    Build()
```

### Recommended Development Configuration

```go
defaults.NewBuilder().
    SetStorage(memory.NewStorage()).
    Timeout(7200).
    AutoRenew(true).
    Build()
```

### Security-First Configuration

```go
defaults.NewBuilder().
    SetStorage(redisStorage).
    Timeout(1800).
    AutoRenew(false).
    Build()
```

## Next Steps

- [Architecture Design](architecture.md)
- [Modular Design](modular.md)
- [DToken API Documentation](../api/dtoken.md)
