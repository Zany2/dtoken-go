[English](auto-renew.md) | 中文文档

# 自动续签设计

本文的 auth 是根包构建的 Auth，dt 表示根包；gindt 表示 Gin 适配器，memory 表示 storage/memory，adapter 表示 contract。HTTP 示例先通过 WithAuth(auth) 绑定请求。内部流程位于 internal/engine/internal_maintenance.go 和 internal_lifecycle.go；应用不直接导入 internal。

## 设计目标

自动续签的目标是：

- 让活跃用户尽量不因 Token 到期而频繁重新登录
- 避免每次登录校验都做重型同步操作
- 通过阈值和节流机制控制续签频率
- 通过协程池提升高并发下的稳定性

## 核心设计

### 异步续签策略

普通登录和身份校验（`IsLogin`、`CheckLogin`、`GetLoginID` 及设备查询）检查存储中的 Token、账号和设备封禁状态，以及不活跃超时，正常路径不加载账号 Session。Token 记录本身仍需解码。依赖会话的查询、权限检查和不活跃超时清理按需加载并校验 Session。

`prepareLoginMaintenance` 将同一 Token 生命周期的并发请求合并为一个待执行任务。仅按配置决定是否调度自动续期资格检查或活跃维护；Token TTL、续期阈值和间隔检查放在工作线程中执行。启用活跃跟踪时，接收任务会记录最近一次成功请求的时间。既不满足续期条件又无需刷新活跃时间时，工作线程直接结束，不加载 Token 和 Session 记录。

单独删除或损坏 Session 不再使普通 Token 校验失败。撤销凭据应使用注销或踢人 API。依赖 Session 的操作仍拒绝无效会话，后台维护无法加载有效 Session 时也不会续期或持久化活跃时间。

不活跃超时判断取有效的持久化活跃时间与当前 Manager 内同一生命周期已接受、待写入的活跃时间中的较大值。登录校验、终端查询和过期终端清理统一遵循此规则，避免协程池排队导致已接受的活跃记录被忽略。已取消任务及其他生命周期的记录不参与判断；持久化标记缺失、损坏或读取失败时仍判定校验失败。排队活跃时间仅对当前 Manager 可见，其他实例需等待持久化后才能读取。

`submitAsync` 优先使用配置的协程池；没有池或提交失败时，回退到受 Manager 跟踪的 goroutine。Manager 关闭期间拒绝新任务。

校验不等待后台维护完成，但工作线程可能在调用方收到结果前启动；自定义池若内联执行任务或阻塞提交，仍可能延迟调用返回。任务合并限制同一生命周期的并发维护，任务完成后的新请求仍可再次调度资格检查。

## 工作流程

### 同步部分

```text
1. 读取并解码 Token 记录，拒绝已过期或已废弃的凭据
2. 检查账号、设备封禁状态和 ActiveTimeout
3. 仅在操作依赖会话或需要清理不活跃超时状态时加载 Session
4. 为当前 Token 生命周期预留或更新维护任务
5. 释放持有的账号锁后提交任务，返回登录校验成功
```

### 异步部分

```text
单个维护任务
  ↓
1. 在账号锁内复核任务代次，检查续期 TTL、阈值和间隔；
   无需续期且无需刷新活跃时间时结束
  ↓
2. 重新加载 Token，复核生命周期和封禁状态，再加载 Session
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

节流标记通常使用配置的 `RenewInterval`。当 Token 的正有效期不大于该间隔时，标记时长改为本次有效期的一半。例如 Token 有效期为 60 秒、`RenewInterval = 300` 时，实际间隔为 30 秒。首次登录、共享登录、自动或强制续期及 `RenewTimeout` 均遵循此规则；手动续期会按新有效期重设标记。计算保留时长精度，支持不足一秒的有效期。禁用间隔和无限期 Token 保留原有语义。续期仍需要在 Token 到期前有满足条件的请求，并及时执行维护任务。

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
router.GET("/profile",
    gindt.CheckLoginMiddleware(ctx, nil, handleAuthFail),
    handleProfile,
)
```

### 3. 手动检查登录

```go
auth.IsLogin(ctx, token)
auth.CheckLogin(ctx, token)
```

### 4. 获取已校验的登录信息

```go
auth.GetLoginID(ctx, token)
auth.GetDevice(ctx, token)
```

`auth.GetTokenInfo` 只读取存储中的 Token 信息，不执行登录校验，因此不会触发自动续签。

## 配置选项

### 启用自动续签

```go
auth, err := dt.NewBuilder().
    Storage(memory.NewStorage()).
    Timeout(86400).
    AutoRenew(true).
    Build()
if err != nil {
    panic(err)
}
defer auth.Close()
```

### 指定续签触发阈值

```go
auth, err := dt.NewBuilder().
    Storage(memory.NewStorage()).
    Timeout(86400).
    AutoRenew(true).
    RenewMaxRefresh(3600). // 剩余 1 小时内才触发续签
    Build()
if err != nil {
    panic(err)
}
defer auth.Close()
```

### 指定最小续期间隔

```go
auth, err := dt.NewBuilder().
    Storage(memory.NewStorage()).
    Timeout(86400).
    AutoRenew(true).
    RenewMaxRefresh(3600).
    RenewInterval(300). // 同一 Token 至少 5 分钟才续一次
    Build()
if err != nil {
    panic(err)
}
defer auth.Close()
```

### 结合最大不活跃时长

```go
auth, err := dt.NewBuilder().
    Storage(memory.NewStorage()).
    Timeout(86400).
    ActiveTimeout(1800).
    AutoRenew(true).
    RenewMaxRefresh(3600).
    RenewInterval(300).
    Build()
if err != nil {
    panic(err)
}
defer auth.Close()
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

- `pool/ants`
- `adapter.Pool`
- `Builder.SetPool(...)`

如果未显式传入池，内部也会在合适场景创建默认续期池。

## 续签失败处理

### 处理策略

异步续签失败不会直接阻塞当前请求：

1. 续期失败不改变本次校验成功的结果
2. 本次续期失败只影响后续有效期延长
3. 下一次满足条件时仍可重试续期

### 影响范围

- 不会直接让当前请求失败
- 可能导致 Token 没有被成功延长
- 若连续失败，Token 最终会按原始 TTL 到期

## 最佳实践

### 生产环境建议

```go
auth, err := dt.NewBuilder().
    Storage(redisStorage).
    Timeout(86400).
    ActiveTimeout(1800).
    AutoRenew(true).
    RenewMaxRefresh(3600).
    RenewInterval(300).
    Build()
if err != nil {
    panic(err)
}
defer auth.Close()
```

### 开发环境建议

```go
auth, err := dt.NewBuilder().
    Storage(memory.NewStorage()).
    Timeout(7200).
    AutoRenew(true).
    Build()
if err != nil {
    panic(err)
}
defer auth.Close()
```

### 安全优先配置

```go
auth, err := dt.NewBuilder().
    Storage(redisStorage).
    Timeout(1800).
    AutoRenew(false).
    Build()
if err != nil {
    panic(err)
}
defer auth.Close()
```

## 下一步

- [架构设计](architecture_zh.md)
- [模块化设计](modular_zh.md)
- [DToken API 文档](../api/dtoken_zh.md)
