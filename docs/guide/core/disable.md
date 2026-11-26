# Disable System

**[中文文档](../core/disable_zh.md)**

Here `dt` names the root package. `Build()` registers automatically; use `dt.XXX()` for the default namespace or `dt.AuthType("user").XXX()` for another namespace. Business method signatures match Auth methods. See [Auth API](../../api/auth.md).

DToken-Go supports account, service, service-level, device-type, and concrete-device restrictions. Base login checks validate credentials only; access points that enforce restrictions must explicitly request the corresponding checks.

## Disable Types

| Type | Description | Typical use |
| --- | --- | --- |
| Account disable | Prevent an account from using login state | Risk control, violation handling, account freeze |
| Service disable | Prevent an account from using a business service | Mute, posting ban, payment ban |
| Service-level disable | Apply service restrictions by severity level | Light restriction, strict restriction, manual review |
| Device-type disable | Prevent an account from accessing through a device type | Ban App, ban Web |
| Concrete-device disable | Prevent an account from accessing through one device ID | Block suspicious terminal or stolen-device session |

## Account Disable

`Disable` destroys the current account session, synchronously retires the terminal credentials listed in it, and removes their linked refresh tokens. Base checks return `ErrInvalidToken` for those credentials; explicit restriction checks return an account-disabled error during the ban. They remain invalid after untie or ban expiry, requiring a new login. `Untie` only removes the ban; it does not restore the destroyed session.

Credential cleanup reads each nonempty token's primary key once, distinguishing missing values, retirement markers and credential ownership directly. Existing values whose ownership cannot be verified are not treated as missing keys. Valid credentials retain their details and remaining TTL while being marked revoked, without loading Extra. When an access credential has expired, refresh ownership is still checked before revocation; those refresh reads are additional costs. The Session is deleted only after cleanup succeeds; failures return an error and retain the index for retry.

A duration of `0` means permanent disable; negative durations are invalid. Setting a ban again replaces the duration and reason at its canonical key. Runtime reads, writes, and unties never probe or delete legacy keys. Use [PrepareAuthUpgrade](../../design/namespace-copy.md) to convert old restrictions before cutover. A canonical marker that cannot be decoded, lacks loginId/kind, or belongs to another identity cannot be overwritten.

```go
ctx := context.Background()

// Disable the account for 2 hours and record a reason.
err := dt.Disable(ctx, dt.DisableOptions{LoginID: "10001", Duration: 2*time.Hour, Reason: "risk"})

disabled := dt.IsDisable(ctx, "10001")
err = dt.CheckDisable(ctx, "10001")

info, err := dt.GetDisableInfo(ctx, "10001")
ttl, err := dt.GetDisableTTL(ctx, "10001")

// Remove account disable state.
err = dt.Untie(ctx, "10001")
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
err := dt.DisableService(ctx, dt.ServiceDisableOptions{LoginID: "10001", Service: "comment", Duration: 30*time.Minute})
err = dt.DisableServiceWithReason(ctx, "10001", "comment", 30*time.Minute, "spam")

disabled := dt.IsDisableService(ctx, "10001", "comment")
err = dt.CheckDisableService(ctx, "10001", []string{"comment", "post"})

info, err := dt.GetDisableServiceInfo(ctx, "10001", "comment")
ttl, err := dt.GetDisableServiceTTL(ctx, "10001", "comment")

err = dt.UntieService(ctx, "10001", "comment")
```

## Service-Level Disable

Service-level disable stores an integer level. A check is considered disabled when the stored level is greater than or equal to the target level.

```go
err := dt.DisableServiceLevel(ctx, "10001", "pay", 3, time.Hour)

dt.IsDisableServiceLevel(ctx, "10001", "pay", 2) // true
dt.IsDisableServiceLevel(ctx, "10001", "pay", 3) // true
dt.IsDisableServiceLevel(ctx, "10001", "pay", 4) // false

err = dt.CheckDisableServiceLevel(ctx, "10001", "pay", 3)
```

This fits layered risk control:

| Level | Example meaning |
| --- | --- |
| `1` | View-only |
| `2` | Submit disabled |
| `3` | Payment or withdrawal disabled |

## Device Disable

Device disable can target a device type or a concrete device ID.

Device bans preserve the session and terminal credentials. After untie, credentials remain usable if they have not expired or been revoked. Device-type and concrete-device bans are independent: removing one does not remove the other. Concrete-device `IsDisable` / `CheckDisable` checks both levels, while detail and TTL queries return only the requested level. Service bans also preserve the session. Replacing a service or device ban updates only its canonical key.

```go
// Disable app access for this account.
err := dt.DisableDevice(ctx, dt.DeviceDisableOptions{LoginID: "10001", Device: "app", Duration: time.Hour})
disabled := dt.IsDisableDevice(ctx, "10001", "app")
err = dt.CheckDisableDevice(ctx, "10001", "app")
err = dt.UntieDevice(ctx, "10001", "app")

// Disable one concrete device.
err = dt.DisableDeviceAndDeviceID(ctx, "10001", "app", "iphone-001", time.Hour)
disabled = dt.IsDisableDeviceAndDeviceID(ctx, "10001", "app", "iphone-001")
err = dt.CheckDisableDeviceAndDeviceID(ctx, "10001", "app", "iphone-001")
err = dt.UntieDeviceAndDeviceID(ctx, "10001", "app", "iphone-001")
```

Query disable details:

```go
deviceInfo, err := dt.GetDisableDeviceInfo(ctx, "10001", "app")
deviceTTL, err := dt.GetDisableDeviceTTL(ctx, "10001", "app")

concreteInfo, err := dt.GetDisableDeviceAndDeviceIDInfo(ctx, "10001", "app", "iphone-001")
concreteTTL, err := dt.GetDisableDeviceAndDeviceIDTTL(ctx, "10001", "app", "iphone-001")
```

## Relationship With Login Checks

Credential validation and access restrictions are separate:

- Account disabling still retires credentials referenced by Session; base checks return `ErrInvalidToken`, and lifting the ban does not revive them. Credentials outside the index require explicit restriction checks for complete enforcement.
- Device-type and concrete-device bans are reversible access restrictions, without retiring the credential; `CheckLogin` / `IsLogin` can succeed. HTTP `CheckDisable: true` checks the account, device type, and concrete device and returns the corresponding restriction error.
- New login, refresh issuance, and introspection still enforce these restrictions. Business code may compose the relevant account and device `CheckDisable*` methods; an account-only check does not replace device checks.
- Service disable does not block all login state automatically. Call `CheckDisableService` or `CheckDisableServiceLevel` explicitly in the corresponding business entry.

Restriction checks read canonical keys by scope: account, single-service, and device-type checks each use one GET. Concrete-device checks read the device type first, then the concrete device only if needed, for at most two GETs. TTL queries validate ownership with a GET before requesting TTL for a matching record. Untie reuses that record and calls Delete on ordinary storage or GetAndDelete when supported; missing records and deletion failures do not emit success events. Restriction details still use the configured Codec, and ordinary login validation does not gain restriction reads.

## Multi-Auth Systems

Disable APIs support optional `authType`:

```go
err := dt.AuthType("admin").Disable(ctx, dt.DisableOptions{LoginID: "admin-1", Duration: time.Hour, Reason: "risk"})
disabled := dt.AuthType("admin").IsDisable(ctx, "admin-1")
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
