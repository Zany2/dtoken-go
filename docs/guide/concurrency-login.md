# Concurrent Login Policy

[中文文档](concurrency-login_zh.md) | English

## Overview

DToken-Go concurrent login behavior is controlled by these configuration fields:

- `IsConcurrent`
- `IsShare`
- `MaxLoginCount`
- `ConcurrencyScope`
- `OverflowLogoutMode`
- `ReplacedLoginExitMode`

Together they decide whether one account can stay online from multiple terminals, whether repeated login can reuse an existing token, how overflow is handled, and whether non-concurrent login replaces the old terminal or rejects the new one.

## IsConcurrent

`IsConcurrent(true)` allows multiple online terminals for the same account.

```go
mgr, err := dtoken.NewBuilder().
    IsConcurrent(true).
    Build()
```

`IsConcurrent(false)` means the same scope cannot have concurrent logins. New login is handled according to `ReplacedLoginExitMode`.

## IsShare

`IsShare(true)` allows repeated login to reuse an existing token when concurrent login is enabled.

Reuse depends on the device dimension:

- account-level login without device information can reuse the account token
- device-level login usually reuses only within the same device type and device ID
- `IsShare(false)` creates a new token for each login

## MaxLoginCount

`MaxLoginCount` limits how many terminals may remain online in the selected scope.

```go
mgr, err := dtoken.NewBuilder().
    IsConcurrent(true).
    IsShare(false).
    MaxLoginCount(2).
    Build()
```

When the limit is exceeded, the oldest terminal is handled by `OverflowLogoutMode`.

## ConcurrencyScope

`ConcurrencyScope` selects the dimension used by concurrency policies.

### Account Scope

```go
builder.ConcurrencyScope(config.ConcurrencyScopeAccount)
```

All devices under the same account are counted together. With `MaxLoginCount(2)`, web, mobile, and desktop can have only two terminals in total.

### Device Scope

```go
builder.ConcurrencyScope(config.ConcurrencyScopeDevice)
```

Each device type is counted separately. Web may have two terminals while mobile may also have two.

## OverflowLogoutMode

When `MaxLoginCount` is exceeded, old tokens are handled by `OverflowLogoutMode`:

| Mode | Behavior |
|------|----------|
| `config.LogoutModeLogout` | delete old token mapping |
| `config.LogoutModeKickout` | mark old token as kickout |
| `config.LogoutModeReplaced` | mark old token as replaced |

## ReplacedLoginExitMode

When `IsConcurrent(false)`, non-concurrent login is handled by `ReplacedLoginExitMode`:

| Mode | Behavior |
|------|----------|
| `config.ReplacedLoginExitModeOldDevice` | allow new login and mark old terminals as replaced |
| `config.ReplacedLoginExitModeNewDevice` | keep old terminals and reject the new login |

## Recommended Combinations

### Common Multi-Terminal Login

```go
dtoken.NewBuilder().
    IsConcurrent(true).
    IsShare(false).
    MaxLoginCount(5).
    ConcurrencyScope(config.ConcurrencyScopeAccount)
```

Good for common web and app scenarios.

### Token Reuse On Same Device

```go
dtoken.NewBuilder().
    IsConcurrent(true).
    IsShare(true).
    ConcurrencyScope(config.ConcurrencyScopeDevice)
```

Useful when repeated login from the same device should not create many tokens.

### New Login Replaces Old Login

```go
dtoken.NewBuilder().
    IsConcurrent(false).
    ReplacedLoginExitMode(config.ReplacedLoginExitModeOldDevice)
```

Good for admin systems that allow only one active terminal.

### Keep Old Login And Reject New Login

```go
dtoken.NewBuilder().
    IsConcurrent(false).
    ReplacedLoginExitMode(config.ReplacedLoginExitModeNewDevice)
```

Good for stricter security scenarios.

## Test Coverage

`TestConcurrencyPolicyFlow` in `tests/gin_core_flow` covers:

- same-device token sharing
- different device ID creating a new token
- account-scope max login overflow
- device-scope max login overflow
- logout, kickout, and replaced overflow modes
- non-concurrent login replacing old terminals
- non-concurrent login rejecting new terminals

## Related Documentation

- [Authentication](authentication.md)
- [Session And Terminal Management](session-terminal.md)
- [Core Flow Testing](core-flow-testing.md)
