# 并发登录策略

[English](../core/concurrency-login.md) | 中文文档

## 概览

DToken-Go 的并发登录由几个配置共同决定：

- `IsConcurrent`
- `IsShare`
- `MaxLoginCount`
- `ConcurrencyScope`
- `OverflowLogoutMode`
- `ReplacedLoginExitMode`

这些配置会影响同一账号是否能多端在线、是否复用 Token、超过最大登录数时如何处理旧 Token，以及非并发登录时是顶掉旧端还是拒绝新端。

## IsConcurrent

`IsConcurrent(true)` 允许同一账号同时存在多个在线终端。

```go
mgr, err := dtoken.NewBuilder().
    IsConcurrent(true).
    Build()
```

`IsConcurrent(false)` 表示同一作用域下不允许并发登录，新登录会按 `ReplacedLoginExitMode` 处理。

## IsShare

`IsShare(true)` 表示在允许并发登录时，重复登录可以复用已有 Token。

复用是否发生取决于设备维度：

- 账号维度下，没有设备信息时更容易复用账号已有 Token
- 设备维度下，通常同设备类型和同设备 ID 才复用
- `IsShare(false)` 时，每次登录都生成新 Token

## MaxLoginCount

`MaxLoginCount` 控制同一作用域下允许保留的最大登录数量。

```go
mgr, err := dtoken.NewBuilder().
    IsConcurrent(true).
    IsShare(false).
    MaxLoginCount(2).
    Build()
```

超过数量后，会按 `OverflowLogoutMode` 处理最旧的终端。

## ConcurrencyScope

`ConcurrencyScope` 决定并发控制的作用域。

### 账号级

```go
builder.ConcurrencyScope(config.ConcurrencyScopeAccount)
```

账号下所有设备一起计数。比如 `MaxLoginCount(2)` 时，web、mobile、desktop 总共最多两个终端。

### 设备级

```go
builder.ConcurrencyScope(config.ConcurrencyScopeDevice)
```

按设备类型分别计数。比如 web 最多两个，mobile 也可以再有两个。

## OverflowLogoutMode

当登录数超过 `MaxLoginCount` 时，旧 Token 的处理模式由 `OverflowLogoutMode` 决定：

| 模式 | 行为 |
|------|------|
| `config.LogoutModeLogout` | 删除旧 Token 映射 |
| `config.LogoutModeKickout` | 标记旧 Token 为 kickout |
| `config.LogoutModeReplaced` | 标记旧 Token 为 replaced |

## ReplacedLoginExitMode

当 `IsConcurrent(false)` 时，非并发登录策略由 `ReplacedLoginExitMode` 决定：

| 模式 | 行为 |
|------|------|
| `config.ReplacedLoginExitModeOldDevice` | 新登录成功，旧终端被标记为 replaced |
| `config.ReplacedLoginExitModeNewDevice` | 保留旧终端，拒绝新登录 |

## 推荐组合

### 常见多端登录

```go
dtoken.NewBuilder().
    IsConcurrent(true).
    IsShare(false).
    MaxLoginCount(5).
    ConcurrencyScope(config.ConcurrencyScopeAccount)
```

适合大多数 Web + App 场景。

### 同设备复用 Token

```go
dtoken.NewBuilder().
    IsConcurrent(true).
    IsShare(true).
    ConcurrencyScope(config.ConcurrencyScopeDevice)
```

适合同一个设备重复登录时避免产生大量 Token。

### 新登录顶掉旧登录

```go
dtoken.NewBuilder().
    IsConcurrent(false).
    ReplacedLoginExitMode(config.ReplacedLoginExitModeOldDevice)
```

适合只允许单端在线的后台、管理端。

### 保留旧登录，拒绝新登录

```go
dtoken.NewBuilder().
    IsConcurrent(false).
    ReplacedLoginExitMode(config.ReplacedLoginExitModeNewDevice)
```

适合高安全场景。

## 测试覆盖

`tests/gin_core_flow` 中的 `TestConcurrencyPolicyFlow` 覆盖了：

- 同设备 Token 复用
- 不同设备 ID 生成新 Token
- 账号级最大登录数溢出
- 设备级最大登录数溢出
- logout、kickout、replaced 三种溢出模式
- 非并发登录替换旧端
- 非并发登录拒绝新端

## 相关文档

- [登录认证指南](../core/authentication_zh.md)
- [Session 与终端管理](../core/session-terminal_zh.md)
- [核心流程测试指南](../integration/core-flow-testing_zh.md)
