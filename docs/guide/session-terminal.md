# Session And Terminal Management

[中文文档](session-terminal_zh.md) | English

## Overview

DToken-Go uses Session to record the online terminal list for a login ID. Each terminal is represented by `TerminalInfo`, with core fields such as:

- `LoginID`
- `Device`
- `DeviceId`
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
deviceCount, err := dtoken.GetOnlineTerminalCountByDeviceAndDeviceId(ctx, "10001", "web", "browser-1")
```

## Query Token Lists

```go
tokens, err := dtoken.GetTokenValueListByLoginID(ctx, "10001", true)
tokens, err = dtoken.GetTokenValueListByDevice(ctx, "10001", "web", true)
tokens, err = dtoken.GetTokenValueListByDeviceAndDeviceId(ctx, "10001", "web", "browser-1", true)
```

The last boolean parameter controls whether only alive tokens should be returned.

## Query Terminal Lists

```go
terminals, err := dtoken.GetTerminalListByLoginID(ctx, "10001")
terminals, err = dtoken.GetTerminalListByLoginIDAndDevice(ctx, "10001", "web")
```

## Get Latest Token

```go
token, err := dtoken.GetTokenValueByLoginID(ctx, "10001")
token, err = dtoken.GetTokenValueByLoginIDAndDevice(ctx, "10001", "web")
```

## Iterate Terminals

```go
err := dtoken.ForEachTerminal(ctx, "10001", func(info manager.TerminalInfo) bool {
    fmt.Println(info.Device, info.DeviceId, info.Token)
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
6. account, device, or service disable makes validation fail

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

- [Authentication](authentication.md)
- [Concurrent Login Policy](concurrency-login.md)
- [Core Flow Testing](core-flow-testing.md)
