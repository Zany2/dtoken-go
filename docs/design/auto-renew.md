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

Ordinary login and identity checks (`IsLogin`, `CheckLogin`, `GetLoginID`, and device queries) validate the stored token and enabled inactivity timeout without loading Session, Extra, or restriction records on the normal path. Access credentials use fixed-field parsing only; legacy Token payloads must be converted offline and are rejected at runtime without Codec decoding. Without inactivity timeout, foreground validation needs one Token GET. Explicit combined `CheckDisable` checks enforce account and device restrictions before scheduling maintenance; background maintenance still rechecks restrictions. Session-dependent queries, permission checks, and inactive-timeout cleanup load and validate the Session when needed.

`prepareLoginMaintenance` coalesces requests for the same token lifecycle into one pending task. Configuration determines whether to schedule automatic-renew eligibility checks or activity maintenance; token TTL, renewal threshold, and interval checks run in the worker. If inactivity tracking is enabled, admission records the latest successful request time. A worker that finds no renewal due and no activity to refresh exits without loading token or Session records.

Deleting or corrupting a Session alone no longer invalidates ordinary token checks. Use logout or kickout APIs to revoke credentials. Operations requiring Session content still reject invalid Sessions. Automatic renewal checks only Session key TTL and stops if the key is missing, without decoding its content. Activity-only maintenance does not require Session existence or readable content; it still rechecks credential lifecycle and restrictions.

Extending Session lifetime no longer serializes or rewrites its value: longer or unlimited TTLs remain, shorter TTLs use Expire, and conversion to unlimited lifetime prefers optional `contract.PersistentStorage.Persist`. Memory and Redis implement it. Ordinary storage falls back to Get of the raw value followed by Set(0), without decoding or cross-operation atomicity. Renewal does not create missing Sessions. Manual renewal still validates the full Session first, and shared login still reads terminal indexes; neither rewrites unchanged Session content merely for renewal.

Manual `RenewTimeout` changes the activity marker's lifetime without refreshing its timestamp. After validation, finite lifetimes use one `Expire` instead of another Get + Set; unlimited lifetimes prefer `Persist`, with raw Get + Set(0) as the ordinary-storage fallback. Token and Extra lifetimes follow the requested duration, including shortening, while Session retains a longer or unlimited lifetime. If the activity marker disappears after validation, renewal returns an error without initializing a new timestamp or emitting a renewal-success event. Earlier writes are not rolled back. The ordinary-storage persistence fallback is non-atomic; native TTL operations preserve concurrent value updates. Background activity maintenance still writes the accepted request time.

Inactivity checks use the later of the valid persisted activity timestamp and the accepted activity queued for the same lifecycle in this Manager. Login validation, terminal queries, and expired-terminal cleanup follow this rule, so pool delays do not discard accepted activity. Cancelled tasks and other lifecycles cannot contribute activity; missing or malformed persisted markers and storage errors still fail validation. Pending activity is local to this Manager; other instances see it only after persistence.

`submitAsync` prefers the configured pool. With no pool or a rejected submission, optional automatic renewal is skipped and its reservation released for later requests. Activity refresh falls back inline without automatic renewal. Explicit `LoginByToken` maintenance and event delivery also fall back inline. Fallback work remains tracked without spawning extra goroutines. A closing Manager rejects new tasks.

Session value updates and activity writes without renewal prefer optional `contract.UpdateStorage.Update`, replacing existing values while retaining expiration. Redis uses one `SET XX KEEPTTL` command and reports missing keys. Memory and ordinary custom storage retain the non-atomic TTL + Set fallback. Explicit renewal still writes its requested lifetime.

When a pool accepts the task, validation does not wait for background maintenance; the worker may start before the caller receives its result. Synchronous activity fallback adds work to the current request. `contract.Pool.Submit` must be concurrency-safe and nonblocking: accept work for asynchronous execution exactly once, or return an error without executing it. Custom pools must not execute inline or wait for capacity. The bundled ants pool rejects `NonBlocking=false` during configuration validation. Task coalescing limits concurrent work for one lifecycle, but a later request can schedule another eligibility check after completion.

A combined `httpauth.Check` reuses token identity and the same Session for permission and role checks. The normal valid path reads and decodes the token once. Login/disable-only checks skip Session; access rules read and decode it once. Disable checks and enabled inactivity checks remain synchronous, and inactive-timeout cleanup may require extra reads. Reuse ends with this check: subsequent calls read current state, and background maintenance revalidates independently.

## Workflow

### Synchronous Part

```text
1. Read the primary value and parse fixed credential fields; reject missing or retired credentials
2. Check enabled ActiveTimeout; check account/device restrictions only when explicitly requested
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
2. Reload the token and verify its lifecycle and disable state
  ↓
3. If renewal is due, extend the token using its own timeout,
   check and preserve the longer Session TTL without reading/writing its value, then write the renew interval marker
  ↓
4. If activity tracking is enabled, persist the latest request activity time
  ↓
5. Release the account lock; emit EventRenew only after successful renewal
```

Obsolete tasks cannot maintain a later login that reuses the same token value. Activity uses the recorded request time, rather than the worker execution time.

`LoginByToken` requests forced maintenance after validating the existing login: it bypasses `AutoRenew`, threshold, and interval eligibility checks. It runs asynchronously when accepted by the pool and falls back inline otherwise. Maintenance errors are not returned through this API; use `RenewTimeout` when the caller needs a synchronous renewal result.

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

Maintenance accepted by the pool runs independently of the validation result. Pool rejection can run activity maintenance inline and add request latency; optional automatic renewal is skipped in that fallback.

1. maintenance failure does not retroactively change successful validation
2. renewal spans multiple keys; failure can leave earlier TTL updates applied, without rollback
3. later eligible requests can retry; failed activity writes can affect subsequent inactivity checks

### Impact Scope

- The remaining lifetime depends on which writes succeeded; failure does not guarantee that the original TTL remains unchanged.
- Automatic `EventRenew` reports success of the credential/Session renewal path, not a guarantee that a subsequent activity write succeeded; activity-write errors are logged separately.
- Manual `RenewTimeout` returns errors synchronously and emits its success event only after its required steps complete. Earlier writes still are not rolled back.

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
