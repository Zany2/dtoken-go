[English](auto-renew.md) | 中文文档

# 自动续签设计

## 设计目标

自动续签的目标是：

- 让活跃用户尽量不因 Token 到期而频繁重新登录
- 避免每次登录校验都做重型同步操作
- 通过阈值和节流机制控制续签频率
- 通过协程池提升高并发下的稳定性

## 核心设计

### 异步续签策略

登录校验先检查 Token 生命周期、Session、账号和设备封禁状态，以及不活跃超时，再调度维护任务。

`prepareLoginMaintenance` 将同一 Token 生命周期的并发请求合并为一个待执行任务。启用活跃跟踪时会记录最近一次成功请求的时间；未启用活跃跟踪时，仅在满足自动续期条件后提交任务。

`submitAsync` 优先使用配置的协程池；没有池或提交失败时，回退到受 Manager 跟踪的 goroutine。Manager 关闭期间拒绝新任务。

## 工作流程

### 同步部分

```text
1. 读取并校验 TokenInfo 和 Session
2. 检查账号、设备封禁状态和 ActiveTimeout
3. 为当前 Token 生命周期预留或更新维护任务
4. 释放账号锁后提交任务，返回登录校验成功
```

### 异步部分

```text
单个维护任务
  ↓
1. 在账号锁内复核任务代次和 Token 生命周期
  ↓
2. 重新加载 Session，并复核封禁状态
  ↓
3. 满足续期条件时，按 Token 自身的有效期续期，
   保留更长的 Session TTL，并写入续期间隔标记
  ↓
4. 启用活跃跟踪时，持久化最近一次请求的活跃时间
  ↓
5. 释放账号锁，仅在续期成功后触发 EventRenew
```

旧任务不能维护后来复用同一 Token 值的新登录。活跃时间采用记录的请求时间，而不是工作线程实际执行时间。

`LoginByToken` 校验已有登录后请求强制维护，不受 `AutoRenew`、阈值和间隔资格检查限制。它仍是异步操作；调用方需要同步续期结果时，应使用 `RenewTimeout`。

## 续签触发条件

当前实现中，自动续签通常需要同时满足这些条件：

- `AutoRenew = true`
- `Timeout > 0`
- Token 当前 TTL 大于 0
- `TTL <= RenewMaxRefresh`，或者 `RenewMaxRefresh <= 0`
- 未命中 `RenewInterval` 的节流限制

这意味着：

- 不是每次请求都会续期
- 越接近过期，越有可能触发续期
- 可以避免高频请求不断刷新同一 Token

## 触发时机

凡是内部走登录校验流程的场景，都可能触发自动续签：

### 1. 中间件认证

```go
r.Use(gindt.AuthMiddleware(ctx))
```

### 2. 注解式登录校验

```go
annotation.GET("/profile",
    gindt.CheckLoginMiddleware(ctx, handleProfile, handleAuthFail))
```

### 3. 手动检查登录

```go
dtoken.IsLogin(ctx, token)
dtoken.CheckLogin(ctx, token)
```

### 4. 获取已校验的登录信息

```go
dtoken.GetLoginID(ctx, token)
dtoken.GetDevice(ctx, token)
```

`dtoken.GetTokenInfo` 只读取存储中的 Token 信息，不执行登录校验，因此不会触发自动续签。

## 配置选项

### 启用自动续签

```go
defaults.NewBuilder().
    SetStorage(memory.NewStorage()).
    Timeout(86400).
    AutoRenew(true).
    Build()
```

### 指定续签触发阈值

```go
defaults.NewBuilder().
    SetStorage(memory.NewStorage()).
    Timeout(86400).
    AutoRenew(true).
    RenewMaxRefresh(3600). // 剩余 1 小时内才触发续签
    Build()
```

### 指定最小续期间隔

```go
defaults.NewBuilder().
    SetStorage(memory.NewStorage()).
    Timeout(86400).
    AutoRenew(true).
    RenewMaxRefresh(3600).
    RenewInterval(300). // 同一 Token 至少 5 分钟才续一次
    Build()
```

### 结合最大不活跃时长

```go
defaults.NewBuilder().
    SetStorage(memory.NewStorage()).
    Timeout(86400).
    ActiveTimeout(1800).
    AutoRenew(true).
    RenewMaxRefresh(3600).
    RenewInterval(300).
    Build()
```

**效果**：
- 活跃用户在接近过期时自动续签
- 超过 `ActiveTimeout` 未活跃会被踢出
- 续签不会因高频请求无限制执行

## 并发安全

### Storage 接口是上下文化的

```go
type Storage interface {
    Set(ctx context.Context, key string, value any, expiration time.Duration) error
    Get(ctx context.Context, key string) (any, error)
    Delete(ctx context.Context, keys ...string) error
    Exists(ctx context.Context, key string) bool
    Expire(ctx context.Context, key string, expiration time.Duration) error
    TTL(ctx context.Context, key string) (time.Duration, error)
    Ping(ctx context.Context) error
}
```

### 协程池支持

默认情况下，开启自动续签时会优先使用续期协程池：

- `com/pool/ants`
- `adapter.Pool`
- `Builder.SetPool(...)`

如果未显式传入池，内部也会在合适场景创建默认续期池。

## 续签失败处理

### 处理策略

异步续签失败不会直接阻塞当前请求：

1. 当前登录校验结果已返回
2. 本次续期失败只影响后续有效期延长
3. 下一次满足条件时仍可重试续期

### 影响范围

- 不会直接让当前请求失败
- 可能导致 Token 没有被成功延长
- 若连续失败，Token 最终会按原始 TTL 到期

## 最佳实践

### 生产环境建议

```go
defaults.NewBuilder().
    SetStorage(redisStorage).
    Timeout(86400).
    ActiveTimeout(1800).
    AutoRenew(true).
    RenewMaxRefresh(3600).
    RenewInterval(300).
    Build()
```

### 开发环境建议

```go
defaults.NewBuilder().
    SetStorage(memory.NewStorage()).
    Timeout(7200).
    AutoRenew(true).
    Build()
```

### 安全优先配置

```go
defaults.NewBuilder().
    SetStorage(redisStorage).
    Timeout(1800).
    AutoRenew(false).
    Build()
```

## 下一步

- [架构设计](architecture_zh.md)
- [模块化设计](modular_zh.md)
- [DToken API 文档](../api/dtoken_zh.md)
