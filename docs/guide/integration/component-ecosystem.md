# Component Ecosystem

Import aliases for code fragments: dt = github.com/Zany2/dtoken-go.

**[中文文档](../integration/component-ecosystem_zh.md)**

DToken-Go decouples core authentication logic from component implementations. The core depends on interfaces, `defaults` wires bundled implementations, and projects can replace storage, codec, logger, Token generator, and async task pool components as needed.

## Component Overview

| Type | Interface / Entry | Built-in implementations | Typical use |
| --- | --- | --- | --- |
| Storage | `adapter.Storage` | Memory, Redis | Store Token, Session, disable, Nonce, and OAuth2 data |
| Atomic storage | `adapter.AtomicStorage` | Redis, Memory | Atomic get-and-delete operations such as one-time Nonce consumption |
| Scanner storage | `adapter.ScannerStorage` | Redis, Memory | Search Tokens and Session IDs |
| Admin storage | `adapter.AdminStorage` | Redis, Memory | Test cleanup or administrative clear operations |
| Codec | `adapter.Codec` | JSON, MessagePack, Base64 | Serialize stored structures |
| Logger | `adapter.Log` | DLog, GoFrame, Nop | Runtime and error logging |
| Token generator | `adapter.Generator` | `dgenerator` | Generate UUID, random, JWT, and other Token styles |
| Async task pool | `adapter.Pool` | Ants | Auto-renew and async event tasks |

## Default Builder

`dt.NewBuilder()` wires default components:

```go
mgr, err := dt.NewBuilder().
    Storage(memory.NewStorage()).
    Build()
```

Memory storage is already provided by default. Set storage explicitly when using Redis or another backend; other default components can remain unchanged unless there is a specific requirement.

## Replace Components

```go
mgr, err := dt.NewBuilder().
    Storage(storage).
    SetCodec(codec).
    SetLog(logger).
    SetGenerator(generator).
    SetPool(pool).
    Build()
```

Factories can also be used to create components lazily:

```go
mgr, err := dt.NewBuilder().
    SetStorageFactory(func(cfg *config.Config) (adapter.Storage, error) {
        return redis.NewStorage("redis://localhost:6379/0")
    }).
    Build()
```

Factories are useful when component creation depends on config values such as `AuthType`, `KeyPrefix`, or deployment environment.

`codec/jsonv2` provides an adapter backed by Go 1.27's standard-library `encoding/json/v2` and requires `GOEXPERIMENT=jsonv2`. Use JSON, MessagePack, Base64, or a custom `adapter.Codec` when JSON v2's experimental behavior is not needed.

## Storage Recommendations

- Single-process tests, examples, and local development: use Memory Storage.
- Multi-instance deployment, cross-service login-state sharing, and production: use Redis Storage.
- If `SearchTokenValue` or `SearchSessionId` is needed, storage should implement `ScannerStorage`.
- If one-time consumption must be safe, storage should implement `AtomicStorage`.

For Redis integration, see [Redis Storage](../integration/redis-storage.md).

## Token Generator

The default generator supports multiple Token styles:

```go
mgr, err := dt.NewBuilder().
    TokenStyle(adapter.TokenStyleRandom64).
    Storage(storage).
    Build()
```

To fully customize generation rules, implement `adapter.Generator`:

```go
import (
    "crypto/rand"
    "encoding/hex"
)

type MyGenerator struct{}

func (MyGenerator) Generate(loginID, device, deviceID string) (string, error) {
    var value [32]byte
    if _, err := rand.Read(value[:]); err != nil {
        return "", err
    }
    return hex.EncodeToString(value[:]), nil
}
```

For all built-in styles, see [Token Styles](../core/token-style.md).

## Logging And Async Tasks

The default `dlog` writes synchronously through Zap. Lumberjack manages size-based rotation, retention, and optional gzip compression. The core continues to use `contract.Log` / `contract.LogControl`.

```go
auth, err := dt.NewBuilder().
    IsLog(true).
    LoggerPath("logs").
    LoggerFilename("dtoken.log").
    LoggerMaxSize(10).
    LoggerMaxBackups(10).
    LoggerMaxAge(7).
    LoggerCompress(true).
    Build()
```

`MaxSize` is measured in MiB (1024 × 1024 bytes), and `MaxAge` in days. A zero `MaxBackups` or `MaxAge` disables that retention limit. `dlog.DefaultLoggerConfig()` supplies the complete defaults of 10 MiB, 10 backups, and 7 days; zero retention limits in a partial configuration remain zero. The default active file is `dtoken_logs/DTOKEN.log`, with a fixed basename and Lumberjack-generated backup names. Lumberjack rejects an individual entry larger than `MaxSize`; Zap reports write errors to stderr.

This refactor removes the custom async queue and timed rotation. `QueueSize`, `RotateExpire`, and their Builder methods are removed. `FileFormat` becomes `Filename`, without date templates. Byte-based `RotateSize` becomes MiB-based `MaxSize`. `RotateBackupLimit` and `RotateBackupDays` become `MaxBackups` and `MaxAge`; the Builder methods use the corresponding new names.

Logging and runtime-control methods remain compatible. `Print` / `Printf` bypass level filtering. `DropCount()` always returns zero because there is no async queue overflow; it does not count file-write failures. With file output enabled, initialization and `SetConfig` first create or open the target in append mode, then close it without modifying existing content; a failed check preserves active outputs. `MaxAge` is limited to 106751 days to prevent overflow in the underlying duration calculation. Each active log file should have a single logger instance managing its rotation. `Close` waits for active synchronous writes and closes the owned file without closing stdout. `Flush` calls Zap's sync interface; Lumberjack exposes no file `fsync` operation, and stdout skips `fsync` to avoid blocking Windows pipes. Lumberjack compresses and cleans backups asynchronously, and `Flush` / `Close` do not wait for that housekeeping.

- When `IsLog(false)` is used, a no-op logger is used automatically.
- When `IsLog(true)` is used, a logger component or logger factory must be available.
- When `AutoRenew(true)` is used, the default builder can wire an async task pool for renewal tasks.
- When `AsyncEvent(true)` is used, event listeners can run asynchronously, which fits audit, logging, and notification use cases.

## Suggestions

- Prefer `dt.NewBuilder()` and replace only the components you truly need to replace.
- Custom components should satisfy core interfaces first, then add optional capabilities when needed.
- Production deployments should usually use Redis Storage with a clear `KeyPrefix`.
- Custom Token generators must provide enough randomness and unpredictability.
- Codec implementations should remain stable to avoid breaking old serialized data.

## Related Documentation

- [Configuration Guide](../reference/configuration.md)
- [Redis Storage](../integration/redis-storage.md)
- [Token Styles](../core/token-style.md)
- [Event Listener](../event.md)
