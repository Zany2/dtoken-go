# Component Ecosystem

**[中文文档](../integration/component-ecosystem_zh.md)**

DToken-Go decouples core authentication logic from component implementations. The core depends on interfaces, `defaults` wires bundled implementations, and projects can replace storage, codec, logger, Token generator, and async task pool components as needed.

## Component Overview

| Type | Interface / Entry | Built-in implementations | Typical use |
| --- | --- | --- | --- |
| Storage | `adapter.Storage` | Memory, Redis | Store Token, Session, disable, Nonce, and OAuth2 data |
| Atomic storage | `adapter.AtomicStorage` | Redis, Memory | Atomic get-and-delete operations such as one-time Nonce consumption |
| Scanner storage | `adapter.ScannerStorage` | Redis, Memory | Search Tokens and Session IDs |
| Admin storage | `adapter.AdminStorage` | Redis, Memory | Test cleanup or administrative clear operations |
| Codec | `adapter.Codec` | JSON, JSON v2, MessagePack, Base64 | Serialize stored structures |
| Logger | `adapter.Log` | DLog, GoFrame, Nop | Runtime and error logging |
| Token generator | `adapter.Generator` | `dgenerator` | Generate UUID, random, JWT, and other Token styles |
| Async task pool | `adapter.Pool` | Ants | Auto-renew and async event tasks |

## Default Builder

`defaults.NewBuilder()` wires default components:

```go
mgr, err := defaults.NewBuilder().
    SetStorage(memory.NewStorage()).
    Build()
```

In most projects, storage is the only component that must be explicitly set. Other default components can remain unchanged unless there is a specific requirement.

## Replace Components

```go
mgr, err := defaults.NewBuilder().
    SetStorage(storage).
    SetCodec(codec).
    SetLog(logger).
    SetGenerator(generator).
    SetPool(pool).
    Build()
```

Factories can also be used to create components lazily:

```go
mgr, err := defaults.NewBuilder().
    SetStorageFactory(func(cfg *config.Config) (adapter.Storage, error) {
        return redis.NewStorage("redis://localhost:6379/0")
    }).
    Build()
```

Factories are useful when component creation depends on config values such as `AuthType`, `KeyPrefix`, or deployment environment.

## Storage Recommendations

- Single-process tests, examples, and local development: use Memory Storage.
- Multi-instance deployment, cross-service login-state sharing, and production: use Redis Storage.
- If `SearchTokenValue` or `SearchSessionId` is needed, storage should implement `ScannerStorage`.
- If one-time consumption must be safe, storage should implement `AtomicStorage`.

For Redis integration, see [Redis Storage](../integration/redis-storage.md).

## Token Generator

The default generator supports multiple Token styles:

```go
mgr, err := defaults.NewBuilder().
    TokenStyle(adapter.TokenStyleRandom64).
    SetStorage(storage).
    Build()
```

To fully customize generation rules, implement `adapter.Generator`:

```go
type MyGenerator struct{}

func (MyGenerator) Generate(loginID, device, deviceID string) (string, error) {
    return "my-token-value", nil
}
```

For all built-in styles, see [Token Styles](../core/token-style.md).

## Logging And Async Tasks

- When `IsLog(false)` is used, a no-op logger is used automatically.
- When `IsLog(true)` is used, a logger component or logger factory must be available.
- When `AutoRenew(true)` is used, the default builder can wire an async task pool for renewal tasks.
- When `AsyncEvent(true)` is used, event listeners can run asynchronously, which fits audit, logging, and notification use cases.

## Suggestions

- Prefer `defaults.NewBuilder()` and replace only the components you truly need to replace.
- Custom components should satisfy core interfaces first, then add optional capabilities when needed.
- Production deployments should usually use Redis Storage with a clear `KeyPrefix`.
- Custom Token generators must provide enough randomness and unpredictability.
- Codec implementations should remain stable to avoid breaking old serialized data.

## Related Documentation

- [Configuration Guide](../reference/configuration.md)
- [Redis Storage](../integration/redis-storage.md)
- [Token Styles](../core/token-style.md)
- [Event Listener](../core/listener.md)
