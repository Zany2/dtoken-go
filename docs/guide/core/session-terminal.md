# Session And Terminal Management

Fragment aliases: dt = github.com/Zany2/dtoken-go.

[中文文档](../core/session-terminal_zh.md) | English

## Overview

DToken-Go uses Session to record the terminal index for a login ID, including entries that may already have expired. Each terminal is represented by `TerminalInfo`, with core fields such as:

- `LoginID`
- `Device`
- `DeviceID`
- `Token`
- `Index`

Token metadata is stored in `TokenInfo`, including auth type, login ID, device, device ID, create time, and timeout.

## Query Session

```go
ctx := context.Background()

sess, err := dt.GetSession(ctx, "10001")
sess, err = dt.GetSessionByToken(ctx, token)
```

Session exposes recorded terminals for an account; use live token lists or online-count APIs to determine current online status.

An encoded null Session is invalid and returns `ErrInvalidToken`; it is not converted into an empty session. Valid legacy session objects may omit authentication and account fields, which are filled from the current namespace and requested key. An empty terminal list alone does not invalidate a Session.

Login and renewal preserve a longer existing Session TTL, including unlimited lifetime. If the required TTL lookup fails, the Session is not overwritten with the requested lifetime. Synchronous operations return `ErrStorageUnavailable`; asynchronous renewal does not emit a success event. Earlier writes in a multi-step operation are not rolled back by this check.

## Session Data

```go
err := dt.SetSessionValueByToken(ctx, token, "theme", "dark")
value, exists, err := dt.GetSessionValueByToken(ctx, token, "theme")
err = dt.DeleteSessionValueByToken(ctx, token, "theme")
```

Token-based setters and deleters first read the compact primary record to locate the account, then recheck the same lifecycle and load the Session once under the account lock. This rejects logout or token reuse while waiting for the lock. The normal path reads the token record twice and the Session once, decoding the Session once. Inactivity checking adds one activity-marker read; Token Extra is not loaded. Token-based data queries read the primary record and Session once each.

The account-based `SetSessionValue`, `GetSessionValue` and `DeleteSessionValue` access the account Session directly without validating a token. Writes preserve the existing Session TTL; deleting a missing field skips encoding and persistence. These data APIs do not schedule automatic renewal or activity refresh. Token-based operations still reject missing, malformed or foreign-namespace Sessions, and inactivity cleanup events run after releasing the account lock. The read counts above cover normal operations; timeout cleanup and failure paths incur additional costs.

## Query Terminal

```go
terminal, err := dt.GetTerminalInfoByToken(ctx, token)
```

Count online terminals by account, device type, or concrete device:

```go
count, err := dt.GetOnlineTerminalCount(ctx, "10001")
webCount, err := dt.GetOnlineTerminalCountByDevice(ctx, "10001", "web")
deviceCount, err := dt.GetOnlineTerminalCountByDeviceAndDeviceID(ctx, "10001", "web", "browser-1")
```

## Query Token Lists

```go
tokens, err := dt.GetTokenValueListByLoginID(ctx, "10001", true)
tokens, err = dt.GetTokenValueListByDevice(ctx, "10001", "web", true)
tokens, err = dt.GetTokenValueListByDeviceAndDeviceID(ctx, "10001", "web", "browser-1", true)
```

The last boolean parameter controls whether only alive tokens should be returned.

## Query Terminal Lists

```go
terminals, err := dt.GetTerminalListByLoginID(ctx, "10001")
terminals, err = dt.GetTerminalListByLoginIDAndDevice(ctx, "10001", "web")
```

## Get Latest Token

Manager's `GetTerminalListByLoginID` and `GetTokenValueByLoginID` accept at most one optional device type; extra arguments return an error.

Manager token-list methods accept at most one optional `checkAlive` value. The root token-list functions shown here require an explicit boolean; select the namespace through dt.AuthType(name). Live lists, online counts, latest-token queries, and token-based terminal details match account, device, creation time, and lifecycle sequence against the token record, ignoring stale entries.

Raw terminal lists, visitors, and token lists with `checkAlive` disabled return stored snapshots that may include expired or stale entries; queries do not delete them. The boolean returned by `GetSessionValue` / `GetSessionValueByToken` distinguishes a stored nil value from a missing key.

Login no longer probes every historical token. New logins verify live slots when the recorded count reaches the limit; shared logins prune only when it exceeds the limit. Device-scoped policies inspect only that device type. Exclusive rejection stops at the first live slot, while replacement prunes its target scope first. Temporary device bans still occupy slots.

Sharing searches newest candidates first and stops after a valid match. Refresh rotation checks its original lifecycle without sweeping other terminals; expired access entries may remain. Exact lists and online counts validate each entry's identity, lifecycle, and validity without repeatedly searching the Session for an entry already supplied.

Device-scoped token lists and online counts filter and process entries directly in the loaded Session without first copying matching terminal records. Tokens outside the selected scope are not read. Latest-token queries likewise filter the original list in reverse order and stop at the first valid match. Raw token lists only read the Session; live lists and counts still read each selected primary record and required state, reusing account and device restriction checks within the query. These changes reduce temporary allocations and repeated traversal; Session loading and decoding remain necessary, result order is preserved, and queries do not perform cleanup.

On-demand pruning reads the primary record once for each selected nonempty token entry and reuses it for activity checks, Extra deletion and state writes. Natural primary-key expiry removes access-only activity and renewal metadata while preserving a still-valid refresh credential. Missing, malformed or expired activity markers retain their existing retirement and refresh-revocation behavior. Existing retirement markers require a refresh ownership check against the account and access token before auxiliary cleanup; those refresh reads are additional costs. Foreign namespaces and existing values parsed as invalid credentials only lose their stale Session entry, without auxiliary deletion. Storage-read and type-conversion errors still propagate.

Inactivity detected during authentication or refresh retains the lifecycle recheck before retirement and reuses its result for state and Extra cleanup. Pruning changes the terminal list in memory; the login boundary persists the Session. A cleanup failure does not persist an incomplete list and does not roll back credentials already retired.

When pruning is unnecessary, new login retains the existing Session and account data. Natural expiration does not emit an immediate logout event. Frequent login with unlimited terminals and a long-lived Session can grow the index; Session reads and serialization still scale with its size. No background full-store sweeper is introduced.

```go
token, err := dt.GetTokenValueByLoginID(ctx, "10001")
token, err = dt.GetTokenValueByLoginIDAndDevice(ctx, "10001", "web")
```

## Iterate Terminals

```go
err := dt.ForEachTerminal(ctx, "10001", func(info dt.TerminalInfo) bool {
    fmt.Println(info.Device, info.DeviceID, info.Token)
    return true
})

err = dt.ForEachTerminalByDevice(ctx, "10001", "web", func(info dt.TerminalInfo) bool {
    return true
})
```

Returning `false` stops iteration.

## Search

```go
tokens, err := dt.SearchTokenValue(ctx, "keyword", 0, 20)
sessions, err := dt.SearchSessionId(ctx, "keyword", 0, 20)
```

These APIs require key scanning support from the storage implementation. The built-in memory and Redis storage adapters both support scanning.

## Logout, Kickout, And Replace

| Operation | Behavior |
|-----------|----------|
| logout | delete token mapping; later checks behave as not logged in |
| kickout | keep a state marker; later checks behave as kicked out |
| replace | keep a state marker; later checks behave as replaced |

Token-based termination remains available during account or device bans. A residual token with verified ownership can be retired even when its session is missing, including its linked refresh token. Existing kickout, replace, and inactivity-timeout markers remain unchanged on repeated calls.

In advanced `Manager.Terminate`, a valid token takes precedence over account and device filters. Explicit whitespace-only token or device filters return an error instead of falling back to a broader scope.

The instance facade uses `auth.Logout`, `auth.Kickout`, or `auth.Replace` with root `LogoutOptions`; the advanced Manager uses `TerminateOptions`.

All three operations support:

- by token
- by login ID
- by device type
- by device type + device ID

## Token Lifecycle

A token can become invalid because:

1. its absolute TTL expires
2. `ActiveTimeout` expires
3. logout deletes it
4. kickout marks it
5. replace marks it
6. account disabling retires credentials indexed by the Session

Device restrictions preserve credentials and require explicit access checks; service restrictions only block the corresponding explicit service checks. Neither directly invalidates base login validation.

## Test Coverage

`tests/gin_core_flow` covers:

- session query
- multi-terminal login
- current terminal info
- token list query
- terminal list query
- online terminal count
- iteration and search
- account, device, and concrete-device logout/kickout/replace
- alive filtering

## Related Documentation

- [Authentication](../core/authentication.md)
- [Concurrent Login Policy](../core/concurrency-login.md)
- [Core Flow Testing](../integration/core-flow-testing.md)
