# Disable System

**[中文文档](../core/disable_zh.md)**

This page uses `global` for registered-instance calls and `dt` for the root package. Build Auth and call `global.SetAuth(auth)` first; omitted auth types select the default `auth:` namespace. See [Auth API](../../api/auth.md) for instance calls and parameter differences.

DToken-Go supports more than account-level banning. It also supports service disable, service-level disable, device-type disable, and concrete-device disable. Disable state participates in login checks where applicable, and disabled subjects receive the corresponding error.

## Disable Types

| Type | Description | Typical use |
| --- | --- | --- |
| Account disable | Prevent an account from using login state | Risk control, violation handling, account freeze |
| Service disable | Prevent an account from using a business service | Mute, posting ban, payment ban |
| Service-level disable | Apply service restrictions by severity level | Light restriction, strict restriction, manual review |
| Device-type disable | Prevent an account from accessing through a device type | Ban App, ban Web |
| Concrete-device disable | Prevent an account from accessing through one device ID | Block suspicious terminal or stolen-device session |

## Account Disable

`Disable` destroys the current account session, synchronously retires the terminal credentials listed in it, and removes their linked refresh tokens. Those access tokens report an account-disabled error during the ban and remain invalid after untie or ban expiry. A new login is required. `Untie` only removes the ban; it does not restore the destroyed session.

A duration of `0` means permanent disable; negative durations are invalid. Setting a ban again replaces its duration and reason and removes matching legacy records, preventing an older ban from reappearing after the new one expires. The operation returns an error if a legacy record cannot be decoded or its ownership cannot be determined.

```go
ctx := context.Background()

// Disable the account for 2 hours and record a reason.
err := global.Disable(ctx, "10001", 2*time.Hour, "risk")

disabled := global.IsDisable(ctx, "10001")
err = global.CheckDisable(ctx, "10001")

info, err := global.GetDisableInfo(ctx, "10001")
ttl, err := global.GetDisableTTL(ctx, "10001")

// Remove account disable state.
err = global.Untie(ctx, "10001")
```

`GetDisableTTL` return values:

| Value | Meaning |
| --- | --- |
| `-2` | Not disabled |
| `-1` | Permanently disabled |
| `> 0` | Remaining seconds |

## Service Disable

Service disable is useful when only one business capability should be restricted instead of the entire account.

```go
err := global.DisableService(ctx, "10001", "comment", 30*time.Minute)
err = global.DisableServiceWithReason(ctx, "10001", "comment", 30*time.Minute, "spam")

disabled := global.IsDisableService(ctx, "10001", "comment")
err = global.CheckDisableService(ctx, "10001", []string{"comment", "post"})

info, err := global.GetDisableServiceInfo(ctx, "10001", "comment")
ttl, err := global.GetDisableServiceTTL(ctx, "10001", "comment")

err = global.UntieService(ctx, "10001", "comment")
```

## Service-Level Disable

Service-level disable stores an integer level. A check is considered disabled when the stored level is greater than or equal to the target level.

```go
err := global.DisableServiceLevel(ctx, "10001", "pay", 3, time.Hour)

global.IsDisableServiceLevel(ctx, "10001", "pay", 2) // true
global.IsDisableServiceLevel(ctx, "10001", "pay", 3) // true
global.IsDisableServiceLevel(ctx, "10001", "pay", 4) // false

err = global.CheckDisableServiceLevel(ctx, "10001", "pay", 3)
```

This fits layered risk control:

| Level | Example meaning |
| --- | --- |
| `1` | View-only |
| `2` | Submit disabled |
| `3` | Payment or withdrawal disabled |

## Device Disable

Device disable can target a device type or a concrete device ID.

Device bans preserve the session and terminal credentials. After untie, credentials remain usable if they have not expired or been revoked. Device-type and concrete-device bans are independent: removing one does not remove the other. Concrete-device `IsDisable` / `CheckDisable` checks both levels, while detail and TTL queries return only the requested level. Service bans also preserve the session. Replacing a service or device ban removes matching legacy records as well.

```go
// Disable app access for this account.
err := global.DisableDevice(ctx, "10001", "app", time.Hour)
disabled := global.IsDisableDevice(ctx, "10001", "app")
err = global.CheckDisableDevice(ctx, "10001", "app")
err = global.UntieDevice(ctx, "10001", "app")

// Disable one concrete device.
err = global.DisableDeviceAndDeviceID(ctx, "10001", "app", "iphone-001", time.Hour)
disabled = global.IsDisableDeviceAndDeviceID(ctx, "10001", "app", "iphone-001")
err = global.CheckDisableDeviceAndDeviceID(ctx, "10001", "app", "iphone-001")
err = global.UntieDeviceAndDeviceID(ctx, "10001", "app", "iphone-001")
```

Query disable details:

```go
deviceInfo, err := global.GetDisableDeviceInfo(ctx, "10001", "app")
deviceTTL, err := global.GetDisableDeviceTTL(ctx, "10001", "app")

concreteInfo, err := global.GetDisableDeviceAndDeviceIDInfo(ctx, "10001", "app", "iphone-001")
concreteTTL, err := global.GetDisableDeviceAndDeviceIDTTL(ctx, "10001", "app", "iphone-001")
```

## Relationship With Login Checks

Account disable and device disable participate in Token checks:

- After an account is disabled, existing Tokens for that account fail login checks.
- After a device type is disabled, Tokens matching that `device` fail.
- After a concrete device is disabled, Tokens matching `device + deviceID` fail.
- Service disable does not block all login state automatically. Call `CheckDisableService` or `CheckDisableServiceLevel` explicitly in the corresponding business entry.

## Multi-Auth Systems

Disable APIs support optional `authType`:

```go
err := global.Disable(ctx, "admin-1", time.Hour, "risk", "admin")
disabled := global.IsDisable(ctx, "admin-1", "admin")
```

Disable data is isolated by `AuthType`.

## Suggestions

- Use account disable for account-level risk.
- Prefer service disable when only one business capability should be restricted.
- Use concrete-device disable for suspicious terminals so other normal devices are not affected.
- Use service-level disable for layered risk control.
- Store clear and auditable business reasons.

## Related Documentation

- [Permission Management](../core/permission.md)
- [Session And Terminal Management](../core/session-terminal.md)
- [Multi-Auth Systems](../core/multi-auth.md)
- [Core API Cheatsheet](../reference/core-api-cheatsheet.md)
