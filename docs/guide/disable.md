# Disable System

**[中文文档](disable_zh.md)**

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

```go
ctx := context.Background()

// Disable the account for 2 hours and record a reason.
err := dtoken.Disable(ctx, "10001", 2*time.Hour, "risk")

disabled := dtoken.IsDisable(ctx, "10001")
err = dtoken.CheckDisable(ctx, "10001")

info, err := dtoken.GetDisableInfo(ctx, "10001")
ttl, err := dtoken.GetDisableTTL(ctx, "10001")

// Remove account disable state.
err = dtoken.Untie(ctx, "10001")
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
err := dtoken.DisableService(ctx, "10001", "comment", 30*time.Minute)
err = dtoken.DisableServiceWithReason(ctx, "10001", "comment", 30*time.Minute, "spam")

disabled := dtoken.IsDisableService(ctx, "10001", "comment")
err = dtoken.CheckDisableService(ctx, "10001", []string{"comment", "post"})

info, err := dtoken.GetDisableServiceInfo(ctx, "10001", "comment")
ttl, err := dtoken.GetDisableServiceTTL(ctx, "10001", "comment")

err = dtoken.UntieService(ctx, "10001", "comment")
```

## Service-Level Disable

Service-level disable stores an integer level. A check is considered disabled when the stored level is greater than or equal to the target level.

```go
err := dtoken.DisableServiceLevel(ctx, "10001", "pay", 3, time.Hour)

dtoken.IsDisableServiceLevel(ctx, "10001", "pay", 2) // true
dtoken.IsDisableServiceLevel(ctx, "10001", "pay", 3) // true
dtoken.IsDisableServiceLevel(ctx, "10001", "pay", 4) // false

err = dtoken.CheckDisableServiceLevel(ctx, "10001", "pay", 3)
```

This fits layered risk control:

| Level | Example meaning |
| --- | --- |
| `1` | View-only |
| `2` | Submit disabled |
| `3` | Payment or withdrawal disabled |

## Device Disable

Device disable can target a device type or a concrete device ID.

```go
// Disable app access for this account.
err := dtoken.DisableDevice(ctx, "10001", "app", time.Hour)
disabled := dtoken.IsDisableDevice(ctx, "10001", "app")
err = dtoken.CheckDisableDevice(ctx, "10001", "app")
err = dtoken.UntieDevice(ctx, "10001", "app")

// Disable one concrete device.
err = dtoken.DisableDeviceAndDeviceId(ctx, "10001", "app", "iphone-001", time.Hour)
disabled = dtoken.IsDisableDeviceAndDeviceId(ctx, "10001", "app", "iphone-001")
err = dtoken.CheckDisableDeviceAndDeviceId(ctx, "10001", "app", "iphone-001")
err = dtoken.UntieDeviceAndDeviceId(ctx, "10001", "app", "iphone-001")
```

Query disable details:

```go
deviceInfo, err := dtoken.GetDisableDeviceInfo(ctx, "10001", "app")
deviceTTL, err := dtoken.GetDisableDeviceTTL(ctx, "10001", "app")

concreteInfo, err := dtoken.GetDisableDeviceAndDeviceIdInfo(ctx, "10001", "app", "iphone-001")
concreteTTL, err := dtoken.GetDisableDeviceAndDeviceIdTTL(ctx, "10001", "app", "iphone-001")
```

## Relationship With Login Checks

Account disable and device disable participate in Token checks:

- After an account is disabled, existing Tokens for that account fail login checks.
- After a device type is disabled, Tokens matching that `device` fail.
- After a concrete device is disabled, Tokens matching `device + deviceId` fail.
- Service disable does not block all login state automatically. Call `CheckDisableService` or `CheckDisableServiceLevel` explicitly in the corresponding business entry.

## Multi-Auth Systems

Disable APIs support optional `authType`:

```go
err := dtoken.Disable(ctx, "admin-1", time.Hour, "risk", "admin")
disabled := dtoken.IsDisable(ctx, "admin-1", "admin")
```

Disable data is isolated by `AuthType`.

## Suggestions

- Use account disable for account-level risk.
- Prefer service disable when only one business capability should be restricted.
- Use concrete-device disable for suspicious terminals so other normal devices are not affected.
- Use service-level disable for layered risk control.
- Store clear and auditable business reasons.

## Related Documentation

- [Permission Management](permission.md)
- [Session And Terminal Management](session-terminal.md)
- [Multi-Auth Systems](multi-auth.md)
- [Core API Cheatsheet](core-api-cheatsheet.md)
