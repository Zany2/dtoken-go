# Session And Terminal Management

[中文文档](../core/session-terminal_zh.md) | English

## Overview

DToken-Go uses Session to record the online terminal list for a login ID. Each terminal is represented by `TerminalInfo`, with core fields such as:

- `LoginID`
- `Device`
- `DeviceID`
- `Token`
- `Index`

Token metadata is stored in `TokenInfo`, including auth type, login ID, device, device ID, create time, and timeout.

## Query Session

```go
ctx := context.Background()

sess, err := dtoken.GetSession(ctx, "10001")
sess, err = dtoken.GetSessionByToken(ctx, token)
```

Session helps you inspect the online terminals under one account.

## Query Terminal

```go
terminal, err := dtoken.GetTerminalInfoByToken(ctx, token)
```

Count online terminals by account, device type, or concrete device:

```go
count, err := dtoken.GetOnlineTerminalCount(ctx, "10001")
webCount, err := dtoken.GetOnlineTerminalCountByDevice(ctx, "10001", "web")
deviceCount, err := dtoken.GetOnlineTerminalCountByDeviceAndDeviceID(ctx, "10001", "web", "browser-1")
```

## Query Token Lists

```go
tokens, err := dtoken.GetTokenValueListByLoginID(ctx, "10001", true)
tokens, err = dtoken.GetTokenValueListByDevice(ctx, "10001", "web", true)
tokens, err = dtoken.GetTokenValueListByDeviceAndDeviceID(ctx, "10001", "web", "browser-1", true)
```

The last boolean parameter controls whether only alive tokens should be returned.

## Query Terminal Lists

```go
terminals, err := dtoken.GetTerminalListByLoginID(ctx, "10001")
terminals, err = dtoken.GetTerminalListByLoginIDAndDevice(ctx, "10001", "web")
```

## Get Latest Token

Manager's `GetTerminalListByLoginID` and `GetTokenValueByLoginID` accept at most one optional device type; extra arguments return an error.

The token-list APIs likewise accept at most one `checkAlive` value. Live lists, online counts, latest-token queries, and token-based terminal details match account, device, creation time, and lifecycle sequence against the token record, ignoring stale entries.

Raw terminal lists, visitors, and token lists without `checkAlive` return stored snapshots that may include expired or stale entries; queries do not delete them. The boolean returned by `GetSessionValue` / `GetSessionValueByToken` distinguishes a stored nil value from a missing key.

```go
token, err := dtoken.GetTokenValueByLoginID(ctx, "10001")
token, err = dtoken.GetTokenValueByLoginIDAndDevice(ctx, "10001", "web")
```

## Iterate Terminals

```go
err := dtoken.ForEachTerminal(ctx, "10001", func(info manager.TerminalInfo) bool {
    fmt.Println(info.Device, info.DeviceID, info.Token)
    return true
})

err = dtoken.ForEachTerminalByDevice(ctx, "10001", "web", func(info manager.TerminalInfo) bool {
    return true
})
```

Returning `false` stops iteration.

## Search

```go
tokens, err := dtoken.SearchTokenValue(ctx, "keyword", 0, 20)
sessions, err := dtoken.SearchSessionId(ctx, "keyword", 0, 20)
```

These APIs require key scanning support from the storage implementation. The built-in memory and Redis storage adapters both support scanning.

## Logout, Kickout, And Replace

| Operation | Behavior |
|-----------|----------|
| logout | delete token mapping; later checks behave as not logged in |
| kickout | keep a state marker; later checks behave as kicked out |
| replace | keep a state marker; later checks behave as replaced |

Token-based termination remains available during account or device bans. A residual token with verified ownership can be retired even when its session is missing, including its linked refresh token. Existing kickout, replace, and inactivity-timeout markers remain unchanged on repeated calls.

In `Terminate`, a valid token takes precedence over account and device filters. Explicit whitespace-only token or device filters return an error instead of falling back to a broader scope.

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
6. account or device disable makes login validation fail

Service disable only blocks the corresponding explicit service checks; it does not invalidate the general login state.

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
