# 配置指南

[English](../reference/configuration.md) | 中文文档

本页整理 DToken-Go 常用配置项、默认行为和配置约束。大多数业务只需要使用 `defaults.NewBuilder()`，按需覆盖少量配置即可。

## 基本示例

```go
mgr, err := defaults.NewBuilder().
    // AuthType 用于区分多套认证体系，如 user、admin、app。
    AuthType("user").
    // KeyPrefix 是存储层 key 的公共前缀，用于区分项目或环境。
    KeyPrefix("dtoken").
    // TokenName 是从 Header、Cookie 或 Body 读取 Token 时使用的名称。
    TokenName("Authorization").
    // Timeout 是 Token 绝对有效期，单位是秒。
    Timeout(7200).
    // RefreshTokenTimeout 是 Refresh Token 绝对有效期，单位是秒。
    RefreshTokenTimeout(30 * 24 * 60 * 60).
    // AutoRenew 开启后，校验登录态时可自动延长 Token 有效期。
    AutoRenew(true).
    // RenewMaxRefresh 表示剩余 TTL 小于等于该值时才触发续期。
    RenewMaxRefresh(3600).
    // RenewInterval 表示同一个 Token 两次自动续期之间的最小间隔。
    RenewInterval(60).
    // IsConcurrent 表示同一账号是否允许多端同时在线。
    IsConcurrent(true).
    // IsShare 表示命中共享条件时是否复用已有 Token。
    IsShare(false).
    // MaxLoginCount 表示同一作用域下最多保留多少个在线终端。
    MaxLoginCount(5).
    // Token 默认从 Header 读取，也可以开启 Cookie 或 Body 读取。
    IsReadHeader(true).
    IsReadCookie(false).
    IsReadBody(false).
    // 生产环境一般建议关闭 Banner，日志按业务需要开启。
    IsLog(false).
    IsPrintBanner(false).
    SetStorage(storage).
    Build()
```

## 常用配置

| 配置 | 默认值 | 说明 |
| --- | --- | --- |
| `AuthType` | `auth:` | 认证体系标识，用于隔离多套 Manager、Token、Session、权限和角色 |
| `KeyPrefix` | `dtoken:` | 存储 key 前缀，建议按项目或环境区分 |
| `TokenName` | `dtoken` | 读取 Token 时使用的 Header、Cookie 或 Body 字段名 |
| `Timeout` | `2592000` | Token 绝对过期时间，单位秒 |
| `RefreshTokenTimeout` | `2592000` | Refresh Token 绝对过期时间，单位秒 |
| `AutoRenew` | `true` | 是否在登录态校验时自动续期 |
| `RenewMaxRefresh` | `Timeout / 2` | 自动续期触发阈值 |
| `RenewInterval` | `-1` | 同一 Token 最小续期间隔，`-1` 表示不限制 |
| `ActiveTimeout` | `-1` | 最大不活跃时长，`-1` 表示不限制 |
| `ConcurrencyScope` | `account` | 并发控制作用域，支持账号级和设备级 |
| `IsConcurrent` | `true` | 是否允许同一账号并发登录 |
| `IsShare` | `true` | 并发登录时是否复用已有 Token |
| `MaxLoginCount` | `12` | 最大在线终端数量 |
| `ReplacedLoginExitMode` | `old_device` | 非并发登录时保留新登录还是拒绝新登录 |
| `OverflowLogoutMode` | `kickout` | 超过最大登录数时旧 Token 的处理方式 |
| `TokenStyle` | `uuid` | Token 生成风格 |
| `JwtSecretKey` | `dtoken-go` | JWT 风格使用的签名密钥 |
| `IsReadHeader` | `true` | 是否从 Header 读取 Token |
| `IsReadCookie` | `false` | 是否从 Cookie 读取 Token |
| `IsReadBody` | `false` | 是否从请求体读取 Token |
| `AsyncEvent` | `true` | 是否异步触发事件监听 |
| `IsLog` | `false` | 是否启用日志组件 |
| `IsPrintBanner` | `true` | 是否打印启动 Banner |

## 命名空间规则

`AuthType` 和 `KeyPrefix` 会自动补齐结尾的 `:`：

```go
defaults.NewBuilder().
    AuthType("admin").
    KeyPrefix("dtoken")
```

最终等价于：

```text
AuthType  = "admin:"
KeyPrefix = "dtoken:"
```

存储 key 通常会组合为：

```text
{KeyPrefix}{AuthType}{业务前缀}{业务值}
```

例如：

```text
dtoken:admin:token:xxx
dtoken:admin:session:10001
```

`AuthType`、`KeyPrefix`、`TokenName` 不能包含空白字符，并且长度不能超过 `64` 个字符。

## 时间配置约束

时间类配置使用秒数，支持 `-1` 表示不限制：

| 配置 | 允许值 |
| --- | --- |
| `Timeout` | `-1` 或 `> 0` |
| `RefreshTokenTimeout` | `-1` 或 `> 0` |
| `RenewMaxRefresh` | `-1` 或 `> 0` |
| `RenewInterval` | `-1` 或 `> 0` |
| `ActiveTimeout` | `-1` 或 `> 0` |
| `MaxLoginCount` | `-1` 或 `> 0` |

自动续期有额外约束：

- `AutoRenew(true)` 时，`Timeout` 不能为 `-1`。
- `RenewMaxRefresh` 不能大于 `Timeout`。
- `RenewInterval` 必须小于 `Timeout`。
- 如果启用了 `ActiveTimeout`，`RenewInterval` 也必须小于 `ActiveTimeout`。

Refresh Token 有效期也可以使用 `time.Duration` 配置：

```go
defaults.NewBuilder().
    RefreshTokenTimeoutDuration(30 * 24 * time.Hour)
```

## Token 读取来源

DToken-Go 至少需要开启一个 Token 来源：

```go
defaults.NewBuilder().
    IsReadHeader(true).
    IsReadCookie(false).
    IsReadBody(false)
```

如果三者都关闭，构建 Manager 会失败。

## Cookie 配置

开启 Cookie 读取时，可以配置 Cookie 属性：

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

Cookie 配置约束：

- `CookieConfig` 不能为 `nil`，除非没有开启 `IsReadCookie`。
- `CookiePath` 不能为空，并且必须以 `/` 开头。
- `CookieMaxAge` 不能小于 `0`。
- `SameSiteNone` 必须搭配 `CookieSecure(true)`。

## 配置组合建议

| 场景 | 建议配置 |
| --- | --- |
| 普通后台系统 | `IsConcurrent(true)`、`IsShare(false)`、`MaxLoginCount(3~5)` |
| 同端复用 Token | `IsConcurrent(true)`、`IsShare(true)`、`ConcurrencyScope(device)` |
| 只允许单端登录 | `IsConcurrent(false)`、`ReplacedLoginExitMode(old_device)` |
| 保留旧登录拒绝新登录 | `IsConcurrent(false)`、`ReplacedLoginExitMode(new_device)` |
| 多认证体系 | 为每套体系设置独立 `AuthType`，共享或隔离 `KeyPrefix` 按部署策略决定 |
| Redis 生产部署 | 使用 Redis Storage，并按环境设置明确的 `KeyPrefix` |

## 相关文档

- [多认证体系](../core/multi-auth_zh.md)
- [并发登录策略](../core/concurrency-login_zh.md)
- [Token 风格](../core/token-style_zh.md)
- [Redis 存储](../integration/redis-storage_zh.md)
