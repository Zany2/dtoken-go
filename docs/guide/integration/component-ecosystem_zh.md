# 组件生态

代码片段的导入别名：dt = github.com/Zany2/dtoken-go.

[English](../integration/component-ecosystem.md) | 中文文档

DToken-Go 将核心认证逻辑和组件实现解耦。核心只依赖接口，默认实现由 `defaults` 组装，也可以按业务需要替换存储、编解码、日志、Token 生成器和异步任务池。

## 组件总览

| 类型 | 接口 / 入口 | 内置实现 | 典型用途 |
| --- | --- | --- | --- |
| 存储 | `adapter.Storage` | Memory、Redis | 保存 Token、Session、封禁、Nonce、OAuth2 数据 |
| 原子存储 | `adapter.AtomicStorage` | Redis、Memory | Nonce 一次性消费等需要原子读删的场景 |
| 扫描存储 | `adapter.ScannerStorage` | Redis、Memory | 搜索 Token、Session ID |
| 管理存储 | `adapter.AdminStorage` | Redis、Memory | 测试清理或管理性清空 |
| 编解码 | `adapter.Codec` | JSON、MessagePack、Base64 | 序列化存储结构 |
| 日志 | `adapter.Log` | DLog、GoFrame、Nop | 运行日志、错误日志 |
| Token 生成器 | `adapter.Generator` | `dgenerator` | 生成 UUID、随机串、JWT 等 Token |
| 异步任务池 | `adapter.Pool` | Ants | 自动续期、异步事件等后台任务 |

## 默认 Builder

`dt.NewBuilder()` 会装配默认组件：

```go
mgr, err := dt.NewBuilder().
    Storage(memory.NewStorage()).
    Build()
```

默认已经提供内存存储；使用 Redis 或其他后端时再显式设置存储。其他组件没有特殊诉求时，可以继续使用默认实现。

## 替换具体组件

```go
mgr, err := dt.NewBuilder().
    Storage(storage).
    SetCodec(codec).
    SetLog(logger).
    SetGenerator(generator).
    SetPool(pool).
    Build()
```

也可以通过工厂延迟创建组件：

```go
mgr, err := dt.NewBuilder().
    SetStorageFactory(func(cfg *config.Config) (adapter.Storage, error) {
        return redis.NewStorage("redis://localhost:6379/0")
    }).
    Build()
```

工厂适合组件依赖配置项的场景，例如根据 `AuthType`、`KeyPrefix` 或部署环境创建不同实例。

`codec/jsonv2` 提供基于 Go 1.27 标准库 `encoding/json/v2` 的适配器，需要启用 `GOEXPERIMENT=jsonv2`。如果不需要 JSON v2 的实验特性，可继续使用 JSON、MessagePack、Base64，或自行实现 `adapter.Codec`。

## 存储组件建议

- 单进程测试、示例项目、本地开发：使用 Memory Storage。
- 多实例部署、跨服务共享登录态、生产环境：使用 Redis Storage。
- 需要 `SearchTokenValue`、`SearchSessionId` 等搜索能力时，存储应实现 `ScannerStorage`。
- 需要安全的一次性消费时，存储应实现 `AtomicStorage`。

Redis 接入方式见 [Redis 存储](../integration/redis-storage_zh.md)。

## Token 生成器

默认生成器支持多种 Token 风格：

```go
mgr, err := dt.NewBuilder().
    TokenStyle(adapter.TokenStyleRandom64).
    Storage(storage).
    Build()
```

如果需要完全自定义生成规则，实现 `adapter.Generator` 即可：

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

更多风格说明见 [Token 风格](../core/token-style_zh.md)。

## 日志与异步任务

默认 `dlog` 基于 Zap 同步输出日志，由 Lumberjack 负责按文件大小轮转、备份保留和可选 gzip 压缩。核心继续通过 `contract.Log` / `contract.LogControl` 使用日志组件。

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

`MaxSize` 的单位是 MiB（1024 × 1024 字节），`MaxAge` 的单位是天；`MaxBackups` 或 `MaxAge` 为零时，不启用对应的保留限制。使用 `dlog.DefaultLoggerConfig()` 可获得 10 MiB、10 个备份、7 天的完整默认配置，部分配置结构体中的零值保留限制不会被默认值覆盖。活动文件默认为 `dtoken_logs/DTOKEN.log`，文件名固定，备份名称由 Lumberjack 生成。单条日志超过 `MaxSize` 时，Lumberjack 拒绝该次文件写入，Zap 将写入错误报告到标准错误输出。

本次重构移除自研异步队列和时间切分：`QueueSize`、`RotateExpire` 及对应 Builder 方法不再提供；`FileFormat` 改为 `Filename`，不再支持日期模板；原字节单位的 `RotateSize` 改为 MiB 单位的 `MaxSize`；`RotateBackupLimit`、`RotateBackupDays` 分别改为 `MaxBackups`、`MaxAge`，Builder 方法也使用对应新名称。

日志调用与运行时控制方法保持兼容。`Print` / `Printf` 不受级别阈值过滤；`DropCount()` 固定返回零，表示没有异步队列溢出丢弃，不代表所有文件写入均成功。启用文件输出时，初始化与 `SetConfig` 会预先创建或以追加方式打开目标文件，检查后关闭，不修改已有内容；检查失败时保留当前输出。`MaxAge` 最大为 106751 天，超过此值会导致底层时长计算溢出，因此会拒绝该配置。每个活动日志文件应由单个日志器实例管理轮转。`Close` 等待正在进行的同步写入并关闭自有文件，不关闭标准输出。`Flush` 调用 Zap 的同步接口；Lumberjack 不提供文件 `fsync` 接口，标准输出也不执行 `fsync`，以避免 Windows 管道同步阻塞。备份压缩与清理由 Lumberjack 异步执行，`Flush` 和 `Close` 不等待这些维护任务。

- `IsLog(false)` 时会使用空日志实现，不需要额外注入日志组件。
- `IsLog(true)` 时应确保存在可用日志组件或日志工厂。
- `AutoRenew(true)` 时，默认 Builder 可装配异步任务池，用于续期任务。
- `AsyncEvent(true)` 时，事件监听可以异步执行，适合审计、日志、通知等不阻塞主流程的操作。

## 使用建议

- 优先使用 `dt.NewBuilder()`，只替换确实需要替换的组件。
- 自定义组件时先满足核心接口，再考虑额外接口能力。
- 生产环境建议使用 Redis Storage，并设置清晰的 `KeyPrefix`。
- 自定义 Token 生成器要保证足够随机性和不可预测性。
- 编解码组件应保持版本稳定，避免历史数据无法反序列化。

## 相关文档

- [配置指南](../reference/configuration_zh.md)
- [Redis 存储](../integration/redis-storage_zh.md)
- [Token 风格](../core/token-style_zh.md)
- [事件监听](../event_zh.md)
