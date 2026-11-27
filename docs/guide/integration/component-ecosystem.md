# Component Ecosystem

Import aliases for code fragments: dt = github.com/Zany2/dtoken-go, adapter = github.com/Zany2/dtoken-go/contract.

**[中文文档](../integration/component-ecosystem_zh.md)**

DToken-Go decouples core authentication logic from component implementations. The core depends on interfaces, the root Builder selects and assembles bundled default components, and projects can replace storage, codec, logger, Token generator, and async task pool components as needed.

After a successful build, the engine owns injected storage, logging, and pools by default. On build failure, explicitly supplied components remain caller-owned. Use WithComponentOwnership to set a complete policy for shared components and close them from the application. Basic components ship with the root module; Redis, MsgPack, JSON v2, and GoFrame logging are separate modules.

## Component Overview

| Type | Interface / Entry | Built-in implementations | Typical use |
| --- | --- | --- | --- |
| Storage | `adapter.Storage` | Memory, Redis | Store Token, Session, disable, Nonce, and OAuth2 data |
| Atomic storage | `adapter.AtomicStorage` | Redis, Memory | Atomic get-and-delete operations such as one-time Nonce consumption |
| Update retaining TTL | `adapter.UpdateStorage` | Redis | Updates existing values with SET XX KEEPTTL; Memory and ordinary storage use non-atomic TTL + Set fallback |
| Persistent storage | `adapter.PersistentStorage` | Redis, Memory | Persist removes expiration from existing keys; optional for base Storage, with raw read/write fallback |
| Scanner storage | `adapter.ScannerStorage` | Redis, Memory | Search Tokens and Session IDs |
| Admin storage | `adapter.AdminStorage` | Redis, Memory | Test cleanup or administrative clear operations |
| Codec | `adapter.Codec` | JSON, MessagePack, Base64 | Serialize stored structures |
| Logger | `adapter.Log` | DLog, GoFrame, Nop | Runtime and error logging |
| Token generator | `adapter.Generator` | `dgenerator` | Generate UUID, random, JWT, and other Token styles |
| Async task pool | `adapter.Pool` | Ants | Auto-renew and async event tasks |

## Default Builder

`dt.NewBuilder()` wires default components:

<!-- dtoken-compile: default-components -->
```go
package main

import (
	dt "github.com/Zany2/dtoken-go"
	adapter "github.com/Zany2/dtoken-go/contract"
	"github.com/Zany2/dtoken-go/storage/memory"
)

func main() {
	storage := memory.NewStorage()
	var _ adapter.Storage = storage
	auth, err := dt.NewBuilder().
		Storage(storage).
		Build()
	if err != nil {
		panic(err)
	}
	defer auth.Close()
}
```

Memory storage is already provided by default. Set storage explicitly when using Redis or another backend; other default components can remain unchanged unless there is a specific requirement.

Component selection follows: explicit instance → custom factory → bundled default factory. Passing `nil` to `SetGeneratorFactory`, `SetStorageFactory`, `SetCodecFactory`, `SetLogFactory`, or `SetPoolFactory` restores the default factory without clearing an injected instance. Pass a `nil` instance to the corresponding component setter to remove that override. Once invoked, a factory that returns an error or a missing required component fails the build; it does not silently fall back to a default.

Bundled logger and pool settings are validated only when those default components are created. Disabled logging or custom logger/pool injection ignores unused bundled implementation settings. `RenewPoolPrintStatusInterval` also controls reporting for custom pools and must remain nonnegative. Without an explicitly injected pool, a pool is created only when `AutoRenew` is enabled; a custom pool factory may still return `(nil, nil)` to use no pool.

## Replace Components

```go
auth, err := dt.NewBuilder().
    Storage(storage).
    SetCodec(codec).
    SetLog(logger).
    SetGenerator(generator).
    SetPool(pool).
    Build()
```

Factories can also be used to create components lazily:

```go
auth, err := dt.NewBuilder().
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
- Nonce consumption explicitly requires `AtomicStorage`. Other operations use atomic extensions where available and retain ordinary storage fallbacks unless their public contract requires otherwise.

For Redis integration, see [Redis Storage](../integration/redis-storage.md).

## Token Generator

The default generator supports multiple Token styles:

```go
auth, err := dt.NewBuilder().
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

Leveled messages use `2026-11-27 15:30:00 [DTOKEN] [INFO] message` by default, with the prefix before the level label. Customize it through `LoggerConfig.Prefix`, Builder `LoggerPrefix("[APP] ")`, or runtime `SetPrefix`. An empty prefix at construction selects the default `[DTOKEN] `. The timestamp format remains configurable; `Print` / `Printf` omit the level label.

<!-- dtoken-compile: logging -->
```go
package main

import dt "github.com/Zany2/dtoken-go"

func main() {
	auth, err := dt.NewBuilder().
		IsLog(true).
		LoggerPath("logs").
		LoggerFilename("dtoken.log").
		LoggerPrefix("[DTOKEN] ").
		LoggerMaxSize(10).
		LoggerMaxBackups(15).
		LoggerMaxAge(15).
		LoggerCompress(true).
		LoggerLocalTime(true).
		Build()
	if err != nil {
		panic(err)
	}
	defer auth.Close()
}
```

The default Builder and `dlog.DefaultLoggerConfig()` use these policies directly through Lumberjack, without custom scheduling, naming, or cleanup logic:

| Setting | Default | Meaning |
| --- | --- | --- |
| `MaxSize` | `10` | Per-file limit in MiB (1024 × 1024 bytes) |
| `MaxAge` | `15` | Retain backups for 15 × 24 hours from the rotation timestamp in their names |
| `MaxBackups` | `15` | At most 15 historical backups plus one active file; `0` disables the count limit |
| `Compress` | `true` | gzip historical files |
| `LocalTime` | `true` | Local time in backup names; `false` uses UTC and does not change the log message timestamp format |

The active file defaults to `dtoken_logs/DTOKEN.log`. When the next write would exceed the size limit, Lumberjack archives it as `DTOKEN-2026-11-27T15-30-00.123.log` (`.log.gz` after compression) and writes to a new active file. The name records rotation time, not a guarantee that all entries belong to one day. Daily automatic rotation and date templates are not provided. Lumberjack rejects an individual entry larger than `MaxSize`; Zap reports write errors to stderr.

A zero `MaxBackups` or `MaxAge` disables that limit. With both configured, either age or count can remove older backups, so a count cap may retain less than 15 days. Cleanup is triggered by Lumberjack opening or rotating files, not by a daily timer, and does not cap total directory size. `MaxAge` does not limit the active file's lifetime. A nil configuration selects all defaults; explicit zero retention limits and false switches in partial configurations remain unchanged. Start from `dlog.DefaultLoggerConfig()` to retain defaults while changing selected fields.

This refactor removes the custom async queue and timed rotation. `QueueSize`, `RotateExpire`, and their Builder methods are removed. `FileFormat` becomes `Filename`, without date templates. Byte-based `RotateSize` becomes MiB-based `MaxSize`. `RotateBackupLimit` and `RotateBackupDays` become `MaxBackups` and `MaxAge`; the Builder methods use the corresponding new names.

The default identifier is `[DTOKEN]`; `Prefix`, `SetPrefix`, and `LoggerPrefix` remain available for customization. `Print` / `Printf` bypass level filtering. `DropCount()` always returns zero because there is no async queue overflow; it does not count file-write failures. With file output enabled, initialization and `SetConfig` first create or open the target in append mode, then close it without modifying existing content; a failed check preserves active outputs. `MaxAge` is limited to 106751 days to prevent overflow in the underlying duration calculation. Each active log file should have a single logger instance managing its rotation. `Close` waits for active synchronous writes and closes the owned file without closing stdout. `Flush` calls Zap's sync interface; Lumberjack exposes no file `fsync` operation, and stdout skips `fsync` to avoid blocking Windows pipes. Lumberjack compresses and cleans backups asynchronously, and `Flush` / `Close` do not wait for that housekeeping.

- When `IsLog(false)` is used, a no-op logger is used automatically.
- When `IsLog(true)` is used, a logger component or logger factory must be available.
- When `AutoRenew(true)` is used, the default builder can wire an async task pool for renewal tasks.
- When `AsyncEvent(true)` is used, event listeners can run asynchronously, which fits audit, logging, and notification use cases.

## Suggestions

- Prefer `dt.NewBuilder()` and replace only the components you truly need to replace.
- Custom components should satisfy core interfaces first, then add optional capabilities when needed.
- Production deployments should usually use Redis Storage with a clear `KeyPrefix`.
- Custom Token generators must provide enough randomness and unpredictability.
- Codec implementations should remain stable to avoid breaking old serialized data. New token primary records use the engine's fixed format and bypass Codec; sessions, token Extra, refresh credentials, and capability objects still use Codec. Custom Codec compression or encryption does not cover identity fields in new token primary records.

## Related Documentation

- [Configuration Guide](../reference/configuration.md)
- [Redis Storage](../integration/redis-storage.md)
- [Token Styles](../core/token-style.md)
- [Event Listener](../core/listener.md)
