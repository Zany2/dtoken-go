# Configuration Guide

**[中文文档](../reference/configuration_zh.md)**

This page summarizes common DToken-Go configuration options, default behavior, and validation rules. Most projects can start with `defaults.NewBuilder()` and override only the options they need.

## Basic Example

```go
mgr, err := defaults.NewBuilder().
    // AuthType separates auth systems such as user, admin, and app.
    AuthType("user").
    // KeyPrefix is the shared storage key prefix for a project or environment.
    KeyPrefix("dtoken").
    // TokenName is the header, cookie, query, or body field used to read the Token.
    TokenName("Authorization").
    // Timeout is the absolute Token lifetime in seconds.
    Timeout(7200).
    // RefreshTokenTimeout is the absolute Refresh Token lifetime in seconds.
    RefreshTokenTimeout(30 * 24 * 60 * 60).
    // AutoRenew allows login checks to extend Token lifetime.
    AutoRenew(true).
    // RenewMaxRefresh triggers renewal when remaining TTL is less than or equal to this value.
    RenewMaxRefresh(3600).
    // RenewInterval is the minimum interval between two auto-renew operations.
    RenewInterval(60).
    // IsConcurrent controls whether one account can be online from multiple terminals.
    IsConcurrent(true).
    // IsShare controls whether a matching existing Token can be reused.
    IsShare(false).
    // MaxLoginCount limits online terminals in the configured scope.
    MaxLoginCount(5).
    // Tokens are read from headers by default; cookie, query, and body sources are optional.
    IsReadHeader(true).
    IsReadCookie(false).
    IsReadQuery(false).
    IsReadBody(false).
    // Banner and logging can be adjusted for production needs.
    IsLog(false).
    IsPrintBanner(false).
    SetStorage(storage).
    Build()
```

## Common Options

| Option | Type | Default | Description |
| --- | --- | --- | --- |
| `AuthType` | `string` | `auth:` | Auth system identifier; separates managers, Tokens, Sessions, permissions, and roles |
| `KeyPrefix` | `string` | `dtoken:` | Storage key prefix, usually separated by project or environment |
| `TokenName` | `string` | `dtoken` | Header, cookie, query, or body field name used to read Tokens |
| `Timeout` | `int64` (seconds) | `2592000` | Absolute Token expiration time |
| `RefreshTokenTimeout` | `int64` (seconds) | `2592000` | Absolute Refresh Token expiration time |
| `AutoRenew` | `bool` | `true` | Whether login checks can automatically renew Tokens |
| `RenewMaxRefresh` | `int64` (seconds) | `Timeout / 2` | Auto-renew trigger threshold; renewal is considered when remaining TTL is less than or equal to this value |
| `RenewInterval` | `int64` (seconds) | `-1` | Minimum renewal interval for one Token; `-1` disables throttling |
| `ActiveTimeout` | `int64` (seconds) | `-1` | Maximum inactive duration; `-1` disables the inactivity check |
| `ConcurrencyScope` | `config.ConcurrencyScope` | `account` | Concurrency control scope: account or device |
| `IsConcurrent` | `bool` | `true` | Whether concurrent login is allowed for the same account |
| `IsShare` | `bool` | `true` | Whether concurrent login may reuse an existing Token |
| `MaxLoginCount` | `int64` (count) | `12` | Maximum online terminal count |
| `ReplacedLoginExitMode` | `config.ReplacedLoginExitMode` | `old_device` | Non-concurrent login strategy |
| `OverflowLogoutMode` | `config.LogoutMode` | `kickout` | How old Tokens are handled when max login count overflows |
| `TokenStyle` | `adapter.TokenStyle` | `uuid` | Token generation style |
| `JwtSecretKey` | `string` | `dtoken-go` | JWT signing secret |
| `IsReadHeader` | `bool` | `true` | Whether to read Token from headers |
| `IsReadCookie` | `bool` | `false` | Whether to read Token from cookies |
| `IsReadQuery` | `bool` | `false` | Whether to read Token from query parameters |
| `IsReadBody` | `bool` | `false` | Whether to read Token from request body |
| `AsyncEvent` | `bool` | `true` | Whether event listeners run asynchronously |
| `IsLog` | `bool` | `false` | Whether logging is enabled |
| `IsPrintBanner` | `bool` | `true` | Whether startup banner is printed |

## Namespace Rules

`AuthType` and `KeyPrefix` automatically receive a trailing `:`:

```go
defaults.NewBuilder().
    AuthType("admin").
    KeyPrefix("dtoken")
```

This is equivalent to:

```text
AuthType  = "admin:"
KeyPrefix = "dtoken:"
```

Storage keys usually follow this structure:

```text
{KeyPrefix}{AuthType}{business-prefix}{business-value}
```

For example:

```text
dtoken:admin:token:xxx
dtoken:admin:session:10001
```

`AuthType`, `KeyPrefix`, and `TokenName` must not contain whitespace and must not exceed `64` characters.

## Numeric Validation

Time options use seconds and support `-1` as unlimited:

| Option | Allowed values |
| --- | --- |
| `Timeout` | `-1` or `> 0` |
| `RefreshTokenTimeout` | `-1` or `> 0` |
| `RenewMaxRefresh` | `-1` or `> 0` |
| `RenewInterval` | `-1` or `> 0` |
| `ActiveTimeout` | `-1` or `> 0` |

`MaxLoginCount` is a count rather than a duration; it accepts `-1` for unlimited or a value greater than `0`.

The `-1` meaning depends on the option:

- `Timeout` and `RefreshTokenTimeout`: no expiration.
- `RenewMaxRefresh`: no TTL threshold; any positive TTL can be considered for renewal.
- `RenewInterval`: no renewal throttling.
- `ActiveTimeout`: disable the inactivity check.

Auto-renew has additional rules:

- `Timeout` cannot be `-1` when `AutoRenew(true)` is enabled.
- `RenewMaxRefresh` cannot be greater than `Timeout`.
- `RenewInterval` must be less than `Timeout`.
- If `ActiveTimeout` is enabled, `RenewInterval` must also be less than `ActiveTimeout`.

Token and refresh-token timeouts can also be configured with `time.Duration`. Positive durations are rounded up to whole seconds:

```go
defaults.NewBuilder().
    TimeoutDuration(2 * time.Hour).
    RefreshTokenTimeoutDuration(30 * 24 * time.Hour)
```

## Token Sources

DToken-Go must have at least one Token source enabled:

```go
defaults.NewBuilder().
    IsReadHeader(true).
    IsReadCookie(false).
    IsReadQuery(false).
    IsReadBody(false)
```

Manager construction fails if all four sources are disabled.

## Optional Modules

The base manager enables core login, logout, session, permission, role, refresh token, disable, and terminal management by default. The following modules are opt-in:

| Module | Default | Enable API |
| --- | --- | --- |
| Nonce | Disabled | `EnableNonce()`, `NonceConfig(...)`, `NonceTTL(...)`, or `SetNonceManager(...)` |
| OAuth2 | Disabled | `EnableOAuth2()`, `OAuth2Config(...)`, `OAuth2CodeExpiration(...)`, `OAuth2TokenExpiration(...)`, `OAuth2RefreshExpiration(...)`, or `SetOAuth2Manager(...)` |
| Ticket | Disabled | `EnableTicket()`, `TicketConfig(...)`, `TicketTTL(...)`, or `SetTicketManager(...)` |
| ShortKey | Disabled | `EnableShortKey()`, `ShortKeyConfig(...)`, `ShortKeyTTL(...)`, `ShortKeyLength(...)`, or `SetShortKeyManager(...)` |
| SSO | Separate module | Import and initialize `github.com/Zany2/dtoken-go/sso` explicitly |

Example:

```go
mgr, err := defaults.NewBuilder().
    EnableNonce().
    EnableTicket().
    ShortKeyTTL(5 * time.Minute). // also enables ShortKey
    Build()
```

Calling a module-specific config method enables that module automatically. Refresh Token is part of the core manager and can be used through `LoginWithRefreshToken` when needed.

## Cookie Configuration

When cookie reading is enabled, cookie attributes can be configured:

```go
mgr, err := defaults.NewBuilder().
    IsReadHeader(false).
    IsReadCookie(true).
    CookieDomain("example.com").
    CookiePath("/").
    CookieSecure(true).
    CookieHttpOnly(true).
    CookieSameSite(config.SameSiteNone).
    CookieMaxAge(7200).
    SetStorage(storage).
    Build()
```

Cookie validation rules:

- `CookieConfig` cannot be `nil` when `IsReadCookie` is enabled.
- `CookiePath` must not be empty and must start with `/`.
- `CookieMaxAge` cannot be negative.
- `SameSiteNone` requires `CookieSecure(true)`.

Default cookie attributes are:

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `Domain` | `string` | `""` | Cookie applies to the current host |
| `Path` | `string` | `/` | Cookie path |
| `Secure` | `bool` | `false` | Cookie may be sent over HTTP |
| `HttpOnly` | `bool` | `true` | JavaScript cannot read the cookie |
| `SameSite` | `config.SameSiteMode` | `Lax` | SameSite policy |
| `MaxAge` | `int64` (seconds) | `0` | Session cookie; no explicit max age |

## Recommended Combinations

| Scenario | Suggested configuration |
| --- | --- |
| Admin system | `IsConcurrent(true)`, `IsShare(false)`, `MaxLoginCount(3~5)` |
| Reuse Token on the same terminal | `IsConcurrent(true)`, `IsShare(true)`, `ConcurrencyScope(device)` |
| Single-terminal login | `IsConcurrent(false)`, `ReplacedLoginExitMode(old_device)` |
| Keep old login and reject new login | `IsConcurrent(false)`, `ReplacedLoginExitMode(new_device)` |
| Multiple auth systems | Use one `AuthType` per system; choose shared or separate `KeyPrefix` by deployment strategy |
| Redis production deployment | Use Redis Storage and set a clear environment-specific `KeyPrefix` |

## Related Documentation

- [Multi-Auth Systems](../core/multi-auth.md)
- [Concurrent Login Policy](../core/concurrency-login.md)
- [Token Styles](../core/token-style.md)
- [Redis Storage](../integration/redis-storage.md)
