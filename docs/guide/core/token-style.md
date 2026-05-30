# Token Styles

**[中文文档](../core/token-style_zh.md)**

DToken-Go uses `TokenStyle` to control how Token strings are generated. Regardless of the style, login state, Sessions, permissions, disable state, kickout, and replace still rely on server-side storage.

## Supported Styles

| Style | Constant | Characteristics | Suitable for |
| --- | --- | --- | --- |
| UUID | `adapter.TokenStyleUUID` | Default style, fixed length, broadly compatible | Most applications |
| Simple | `adapter.TokenStyleSimple` | Simple random string | Internal systems and tests |
| Random32 | `adapter.TokenStyleRandom32` | 32-character random string | Shorter random Tokens |
| Random64 | `adapter.TokenStyleRandom64` | 64-character random string | Stronger randomness |
| Random128 | `adapter.TokenStyleRandom128` | 128-character random string | Higher security requirements |
| Hash | `adapter.TokenStyleHash` | SHA256 hash style | Hash-like Token appearance |
| Timestamp | `adapter.TokenStyleTimestamp` | Includes timestamp characteristics | Debugging and internal traces |
| Tik | `adapter.TokenStyleTik` | Short ID style | Scenarios that prefer shorter Tokens |
| JWT | `adapter.TokenStyleJWT` | Parseable claims, longer Token | Gateway cooperation, debugging, claim-aware scenarios |

## Basic Usage

```go
mgr, err := defaults.NewBuilder().
    TokenStyle(adapter.TokenStyleRandom64).
    SetStorage(storage).
    Build()
```

JWT style requires a signing secret:

```go
mgr, err := defaults.NewBuilder().
    TokenStyle(adapter.TokenStyleJWT).
    JwtSecretKey("your-very-strong-secret").
    SetStorage(storage).
    Build()
```

Shortcut:

```go
mgr, err := defaults.NewBuilder().
    JwtSecret("your-very-strong-secret").
    SetStorage(storage).
    Build()
```

## JWT Is Not Fully Stateless

`TokenStyleJWT` only means the Token string is formatted as JWT. It does not turn DToken-Go into a fully stateless authentication system.

The following capabilities still use server-side storage:

- Login-state checks
- Session and terminal management
- Auto-renew and active timeout
- Permissions and roles
- Account, service, and device disable
- `Logout`, `Kickout`, and `Replace`

For full JWT details, see [JWT Guide](../security/jwt.md).

## Custom Generator

If built-in styles are not enough, implement `adapter.Generator`:

```go
type MyGenerator struct{}

func (MyGenerator) Generate(loginID, device, deviceID string) (string, error) {
    return "custom-token-value", nil
}

mgr, err := defaults.NewBuilder().
    SetGenerator(MyGenerator{}).
    SetStorage(storage).
    Build()
```

A custom generator should ensure:

- Tokens are hard to guess.
- Collision probability remains low under high generation volume.
- Sensitive business data is not leaked directly.
- Generation failures return clear errors.

## Selection Guide

| Requirement | Recommended style |
| --- | --- |
| Default general usage | UUID |
| Stronger randomness | Random64 or Random128 |
| Parseable claims | JWT |
| Shorter Token | Tik or Simple |
| Timestamp characteristics for internal debugging | Timestamp |
| Enterprise-wide Token rules | Custom Generator |

## Related Documentation

- [JWT Guide](../security/jwt.md)
- [Configuration Guide](../reference/configuration.md)
- [Component Ecosystem](../integration/component-ecosystem.md)
