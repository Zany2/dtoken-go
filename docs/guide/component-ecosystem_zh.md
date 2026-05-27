# 组件生态

[English](component-ecosystem.md) | 中文文档

DToken-Go 将核心认证逻辑和组件实现解耦。核心只依赖接口，默认实现由 `defaults` 组装，也可以按业务需要替换存储、编解码、日志、Token 生成器和异步任务池。

## 组件总览

| 类型 | 接口 / 入口 | 内置实现 | 典型用途 |
| --- | --- | --- | --- |
| 存储 | `adapter.Storage` | Memory、Redis | 保存 Token、Session、封禁、Nonce、OAuth2 数据 |
| 原子存储 | `adapter.AtomicStorage` | Redis、Memory | Nonce 一次性消费等需要原子读删的场景 |
| 扫描存储 | `adapter.ScannerStorage` | Redis、Memory | 搜索 Token、Session ID |
| 管理存储 | `adapter.AdminStorage` | Redis、Memory | 测试清理或管理性清空 |
| 编解码 | `adapter.Codec` | JSON、JSON v2、MessagePack、Base64 | 序列化存储结构 |
| 日志 | `adapter.Log` | DLog、GoFrame、Nop | 运行日志、错误日志 |
| Token 生成器 | `adapter.Generator` | `dgenerator` | 生成 UUID、随机串、JWT 等 Token |
| 异步任务池 | `adapter.Pool` | Ants | 自动续期、异步事件等后台任务 |

## 默认 Builder

`defaults.NewBuilder()` 会装配默认组件：

```go
mgr, err := defaults.NewBuilder().
    SetStorage(memory.NewStorage()).
    Build()
```

通常只需要显式设置存储。其他组件没有特殊诉求时，可以继续使用默认实现。

## 替换具体组件

```go
mgr, err := defaults.NewBuilder().
    SetStorage(storage).
    SetCodec(codec).
    SetLog(logger).
    SetGenerator(generator).
    SetPool(pool).
    Build()
```

也可以通过工厂延迟创建组件：

```go
mgr, err := defaults.NewBuilder().
    SetStorageFactory(func(cfg *config.Config) (adapter.Storage, error) {
        return redis.NewStorage("redis://localhost:6379/0")
    }).
    Build()
```

工厂适合组件依赖配置项的场景，例如根据 `AuthType`、`KeyPrefix` 或部署环境创建不同实例。

## 存储组件建议

- 单进程测试、示例项目、本地开发：使用 Memory Storage。
- 多实例部署、跨服务共享登录态、生产环境：使用 Redis Storage。
- 需要 `SearchTokenValue`、`SearchSessionId` 等搜索能力时，存储应实现 `ScannerStorage`。
- 需要安全的一次性消费时，存储应实现 `AtomicStorage`。

Redis 接入方式见 [Redis 存储](redis-storage_zh.md)。

## Token 生成器

默认生成器支持多种 Token 风格：

```go
mgr, err := defaults.NewBuilder().
    TokenStyle(adapter.TokenStyleRandom64).
    SetStorage(storage).
    Build()
```

如果需要完全自定义生成规则，实现 `adapter.Generator` 即可：

```go
type MyGenerator struct{}

func (MyGenerator) Generate(loginID, device, deviceID string) (string, error) {
    return "my-token-value", nil
}
```

更多风格说明见 [Token 风格](token-style_zh.md)。

## 日志与异步任务

- `IsLog(false)` 时会使用空日志实现，不需要额外注入日志组件。
- `IsLog(true)` 时应确保存在可用日志组件或日志工厂。
- `AutoRenew(true)` 时，默认 Builder 可装配异步任务池，用于续期任务。
- `AsyncEvent(true)` 时，事件监听可以异步执行，适合审计、日志、通知等不阻塞主流程的操作。

## 使用建议

- 优先使用 `defaults.NewBuilder()`，只替换确实需要替换的组件。
- 自定义组件时先满足核心接口，再考虑额外接口能力。
- 生产环境建议使用 Redis Storage，并设置清晰的 `KeyPrefix`。
- 自定义 Token 生成器要保证足够随机性和不可预测性。
- 编解码组件应保持版本稳定，避免历史数据无法反序列化。

## 相关文档

- [配置指南](configuration_zh.md)
- [Redis 存储](redis-storage_zh.md)
- [Token 风格](token-style_zh.md)
- [事件监听](listener_zh.md)
