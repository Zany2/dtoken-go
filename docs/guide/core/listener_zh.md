# 事件监听指南

代码片段中，dt = github.com/Zany2/dtoken-go，listener = github.com/Zany2/dtoken-go/event。

[English](listener.md) | 中文文档

独立能力只有经 NewService(manager, auth) 绑定发布器并调用 Service 时，才向该 Auth 的事件管理器发布对应能力事件。直接调用能力 Manager 不经过服务封装。应用监听器应在开始接收请求前注册，在监听器之外通过 Auth.Close 统一关闭引擎。

## 当前状态

当前项目内部已经实现了完整的事件系统，底层代码位于 `event`。

它覆盖认证与会话生命周期、访问检查与变更、刷新令牌、Nonce、Ticket、ShortKey 和 OAuth2 等流程。

## 获取事件管理器

可通过根包注册表、Auth 实例或高级 `dt.Manager` 获取事件管理器：

```go
registryEventMgr, err := dt.GetEventManager("user")
if err != nil {
    // 处理查找错误。
}
instanceEventMgr := auth.EventManager()
coreEventMgr := mgr.GetEventManager()
```

取得 `*listener.Manager` 后，可使用 `RegisterFunc`、`RegisterFuncWithConfig` 等注册 API。也可通过 `dt.AuthType("user").EventManager()` 查找指定体系，不可用时返回 nil；`dt.GetEventManager("user")` 则返回查找错误。这些入口最终取得所选引擎持有的同一个事件管理器。

## 内部事件类型

当前定义在 `event/consts.go` 中的内置事件有：

| 事件 | 说明 |
|------|------|
| `EventLogin` | 登录事件 |
| `EventLogout` | 登出事件 |
| `EventKickout` | 踢下线事件 |
| `EventActiveTimeout` | 不活跃超时事件 |
| `EventReplace` | 顶下线事件 |
| `EventDisable` | 账号封禁事件 |
| `EventUntie` | 账号解封事件 |
| `EventRenew` | Token 续期事件 |
| `EventCreateSession` | Session 创建事件 |
| `EventDestroySession` | Session 销毁事件 |
| `EventPermissionCheck` | 权限校验事件 |
| `EventPermissionChange` | 权限变更事件 |
| `EventRoleCheck` | 角色校验事件 |
| `EventRoleChange` | 角色变更事件 |
| `EventDisableService` | 服务封禁事件 |
| `EventUntieService` | 服务解封事件 |
| `EventDisableDevice` | 设备封禁事件 |
| `EventUntieDevice` | 设备解封事件 |
| `EventRefreshTokenCreate` | 刷新令牌创建事件 |
| `EventRefreshTokenRotate` | 刷新令牌轮换事件 |
| `EventRefreshTokenRevoke` | 刷新令牌撤销事件 |
| `EventNonceGenerate` | Nonce 生成事件 |
| `EventNonceVerify` | Nonce 校验事件 |
| `EventTicketCreate` | Ticket 创建事件 |
| `EventTicketValidate` | Ticket 校验事件 |
| `EventTicketConsume` | Ticket 消费事件 |
| `EventTicketRevoke` | Ticket 撤销事件 |
| `EventShortKeyCreate` | ShortKey 创建事件 |
| `EventShortKeyConfirm` | ShortKey 确认事件 |
| `EventShortKeyValidate` | ShortKey 校验事件 |
| `EventShortKeyConsume` | ShortKey 消费事件 |
| `EventShortKeyRevoke` | ShortKey 撤销事件 |
| `EventOAuth2ClientRegister` | OAuth2 客户端注册事件 |
| `EventOAuth2ClientUnregister` | OAuth2 客户端注销事件 |
| `EventOAuth2CodeGenerate` | OAuth2 授权码生成事件 |
| `EventOAuth2TokenIssue` | OAuth2 Token 签发事件 |
| `EventOAuth2TokenRefresh` | OAuth2 Token 刷新事件 |
| `EventOAuth2TokenValidate` | OAuth2 Token 校验事件 |
| `EventOAuth2TokenRevoke` | OAuth2 Token 撤销事件 |
| `EventAll` | 通配事件 |

## EventData 结构

事件载荷结构在 `event/listener.go` 中定义：

```go
type EventData struct {
    Event     Event
    AuthType  string
    LoginID   string
    Device    string
    DeviceID  string
    Token     string
    Extra     map[string]any
    Timestamp int64
}
```

触发方法会取得事件载荷快照。每个监听器都会收到独立的标量字段副本、顶层 `Extra` map 副本，以及直接存放在该 map 中的 `[]string` 类型值的副本。复制不会递归进行：嵌套的 map、除这些直接存放的 `[]string` 值之外的 slice、指针和其他引用值仍然共享，过滤器和监听器必须将这些共享引用视为只读数据。

Manager 的触发诊断日志只记录事件类型、认证体系、时间戳和监听器数量，不会写入 `LoginID`、设备字段、Token 或 `Extra`。应用监听器和自定义 panic 处理器仍需自行保护收到的事件载荷。

其中：

- `AuthType`：当前认证体系
- `LoginID`：账号 ID
- `Device` / `DeviceID`：终端信息
- `Token`：相关 token
- `Extra`：额外信息，比如权限校验结果、服务封禁等级等

## Extra 字段常量

Ticket 和 ShortKey 事件中的 `ExtraKeyTTL` 表示事件生成时的剩余秒数，优先按凭证的精确 `ExpiresAt` 计算，正数不足一秒时向上取整；已到期返回 `0`。没有 `ExpiresAt` 的旧凭证仍按 `CreateTime + ExpiresIn` 计算。异步监听器收到事件时，这个值是生成事件时的快照。

当前额外字段常量包括：

- `ExtraKeyPermission`
- `ExtraKeyPermissions`
- `ExtraKeyRole`
- `ExtraKeyRoles`
- `ExtraKeyLogic`
- `ExtraKeyResult`
- `ExtraKeyAction`
- `ExtraKeyShared`
- `ExtraKeyService`
- `ExtraKeyLevel`
- `ExtraKeyTokenType`
- `ExtraKeyClientID`
- `ExtraKeyUserID`
- `ExtraKeyScopes`
- `ExtraKeySource`
- `ExtraKeySourceApp`
- `ExtraKeyTargetApp`
- `ExtraKeyScene`
- `ExtraKeyStatus`
- `ExtraKeyTTL`
- `ExtraKeyRefreshToken`
- `ExtraKeyGrantType`

## 事件管理器 API

`event` 是公开包，也可以独立使用。

### 创建监听管理器

```go
import (
    listener "github.com/Zany2/dtoken-go/event"
)

eventMgr := listener.NewManager()
```

### 注册监听器

```go
id := eventMgr.RegisterFunc(listener.EventLogin, func(data *listener.EventData) {
    println("login:", data.LoginID)
})

_ = id
```

### 使用配置注册

```go
id := eventMgr.RegisterFuncWithConfig(
    listener.EventLogin,
    func(data *listener.EventData) {
        println("login:", data.LoginID)
    },
    listener.ListenerConfig{
        Async:    true,
        Priority: 100,
        ID:       "login-audit",
    },
)
```

监听器 ID 在同一个管理器内全局唯一。使用重复的显式 ID 注册时会返回空字符串；自动生成 ID 时会跳过已占用的值。

`Register` 和 `RegisterFunc` 默认使用 `Async: true`、优先级 `0`。需要内联执行或调整优先级时，应使用 `RegisterFuncWithConfig`。

### 注销监听器

```go
ok := eventMgr.Unregister("login-audit")
_ = ok
```

### 触发事件

- `Trigger` 在当前 goroutine 中分发；同步监听器内联执行，异步监听器被池接收后不等待，未配置池或拒绝时内联执行。
- `TriggerAsync` 在返回前取得载荷快照，将零值 `Timestamp` 补为提交时间，再提交整个分发过程到事件池；未配置池或拒绝时同步完成直接监听器。显式时间戳保持不变，调用方传入的载荷不会被修改。
- `TriggerSync` 在当前 goroutine 中按优先级依次执行直接监听器，忽略其 Async 标记；它不会等待无关异步任务，也不会等待监听器通过 `Trigger` 或 `TriggerAsync` 启动的后台后代任务，但嵌套调用 `TriggerSync` 仍属于当前监听器的完成过程。
- `Wait` 会等待事件管理器已接收的异步分发和异步监听器任务全部结束。

具体事件监听器和 `EventAll` 通配监听器会合并后按优先级降序遍历。对于异步监听器，优先级只控制启动遍历顺序；goroutine 调度不保证回调的实际执行或完成顺序。

监听器排序和合并在注册关系变化时完成，触发时复用不可变列表，不再逐次复制、合并和排序。相同优先级下，同类监听器保持注册顺序，具体事件监听器先于 `EventAll`。回调中的注册、注销或清空不会改变已经取得的分发快照；新配置供后续快照使用。过滤器列表也遵循这一规则，回调仍在管理器锁外执行。

### 跳过无处理需求的事件

禁用的事件会直接跳过。启用的事件只有在没有具体或通配监听器、没有过滤器、未开启统计，并且没有有效日志记录需求时，才会跳过载荷复制与异步任务提交。纯过滤、纯统计、纯日志事件仍执行；内置 `contract.NopLogger` 被识别为空日志，自定义日志器始终按有日志需求处理。

`eventMgr.NeedsDispatch(event)` 可用于构造昂贵载荷前的判断，包含上述全部条件；`HasListeners` 仍只检查指定事件的直接注册项。引擎已在构造 EventData、复制 Extra 及提交事件任务前使用该判断；调用方在调用前自行构造载荷的成本仍由调用方承担。

这个判断只反映当前配置，不预留监听器。调用时无处理需求而跳过的事件，不会因之后注册监听器或开启统计而补发。已接收的异步分发仍在实际执行时取得配置快照，载荷和默认时间戳则在提交时取得。建议在接收请求前完成配置。

合并列表只为已注册具体监听器的事件维护；触发任意自定义事件名不会增加缓存条目。注册变更承担列表复制和合并成本，换取高频分发时直接读取；每个监听器的载荷隔离复制仍然保留。

```go
eventMgr.Trigger(data)
eventMgr.TriggerAsync(data)
eventMgr.TriggerSync(data)
```

## 高级能力

### 可选的独立事件池

未配置事件池时，事件任务同步执行。可通过 `SetAsyncPool(pool contract.Pool)` 接入现有的池接口及实现。应为事件单独创建一个实例，不要传入引擎的续期池，并在接收请求前完成配置：

```go
import eventpool "github.com/Zany2/dtoken-go/pool/ants"

cfg := eventpool.DefaultRenewPoolConfig()
cfg.SetNonBlocking(true)
eventPool, err := eventpool.NewRenewPoolManagerWithConfig(cfg)
if err != nil {
    panic(err)
}
auth.EventManager().SetAsyncPool(eventPool)
```

现有构造器名称虽然包含 `Renew`，其实现可以执行通用任务，因此可以单独实例化为事件池。独立使用的 `listener.NewManager()` 同样支持此设置。框架不会自动创建事件池，引擎的 `AsyncEvent` 分发配置及续期池装配保持不变。

- `Trigger` 将异步监听器提交到事件池，同步监听器仍内联执行。
- `TriggerAsync` 将整个分发任务提交到事件池，其中的异步监听器也使用事件池。
- `TriggerSync` 内联执行直接监听器，不提交池任务，也不等待池中的其他任务，避免嵌套等待自身工作协程。
- 提交被拒绝时同步回退，保留事件投递，不额外创建 goroutine；回调耗时会计入调用方耗时，且多个调用方仍可并发执行回退。
- `SetAsyncPool(nil)` 使后续提交改为同步执行；已接收任务仍由原池执行，并继续由 `Wait` 跟踪。

传入的池必须提供并发安全、非阻塞的提交：接受任务后恰好异步执行一次，或返回错误且不执行该任务。不能阻塞等待容量、静默丢弃已接收任务或内联执行任务。嵌套分发可能在池工作协程中再次提交，阻塞等待容量会造成死锁。内置 ants 池必须设置 `NonBlocking: true`，设为 false 会在构建时返回配置错误。

事件池由应用持有，事件管理器不会调用它的 `Stop`。关闭时先停止并排空请求及事件生产方，在回调之外调用 `auth.Close()`，再调用 `eventPool.Stop()`。独立事件管理器则先调用 `eventMgr.Wait()`，再停止池。替换池时，也应保持旧池存活直到其已接收任务排空。

### 全局过滤器

```go
eventMgr.AddFilter(func(data *listener.EventData) bool {
    return data.AuthType == "auth:"
})
```

过滤器返回 `false` 时，事件不会继续分发。

过滤器 panic 会被恢复并交给 panic 处理器，同时拒绝当前事件。

### 统计信息

```go
eventMgr.EnableStats(true)

stats := eventMgr.GetStats()
println(stats.TotalTriggered)
```

### Panic 处理

```go
eventMgr.SetPanicHandler(func(event listener.Event, data *listener.EventData, recovered any) {
    println("listener panic:", event, recovered)
})
```

监听器和过滤器的 panic 都会被恢复并交给该处理器。自定义 panic 处理器自身发生 panic 时也会被隔离，并由 Manager 日志器记录。

### 启用 / 禁用事件

```go
eventMgr.DisableEvent(listener.EventRenew, listener.EventPermissionCheck)
eventMgr.EnableEvent(listener.EventLogin, listener.EventLogout)
eventMgr.EnableEvent() // 不传参表示启用全部
```

`EnableEvent(events...)` 会替换当前事件白名单；`DisableEvent(events...)` 只禁用选定事件，不改变其他内置事件或自定义事件。两种方法都可使用 `EventAll` 表示全部事件。

### 等待异步监听器完成

```go
eventMgr.Wait()
```

应将 `Wait` 作为外部生命周期屏障使用：先停止调用方提交新事件，再在关闭阶段调用 `Wait`，最后关闭监听器依赖。不要在监听器内部调用它，因为当前监听器本身可能就是被跟踪的异步任务。`Wait` 返回后新提交的任务不在本次等待范围内。

## 逻辑常量

```go
listener.LogicAnd
listener.LogicOr
```

它们通常出现在权限 / 角色校验事件的 `Extra` 字段里。

## 使用建议

通过门面或 Manager 实例取得事件管理器，并在开始处理业务流量前完成应用监听器注册。关闭时先停止事件生产方，再调用 `Wait` 排空已接收的异步任务。

公开的 `Auth.Emit` 和 `Manager.Emit` 入口在认证引擎开始关闭后忽略新事件，不受 `AsyncEvent` 配置影响。已经接收的异步事件及内部续期完成事件仍会在释放资源前排空。这不会关闭单独持有的 `event.Manager`；直接调用其 `Trigger` 系列方法的调用方仍须在关闭前停止自己的事件生产方。

使用 `Manager.CloseManager` 时，应先停止并等待业务请求结束，再从任务和监听器之外调用。它会拒绝新的异步任务，停止后台状态日志，等待已接收的任务和异步监听器完成，最后释放自有 Pool、Storage 和 Logger；调用方持有的组件不会被关闭。已接收的续期任务会在自身任务内分发续期事件，因此关闭期间不会因再次提交被拒绝而丢失该事件，也不会再次占用同一个 Pool。监听器自身的 `Async` 配置仍然有效。

Manager 会恢复并记录异步任务中的 panic，包括无 Pool 和提交失败后的同步回退路径。恢复只隔离异常，不会重试任务或回滚已经完成的存储操作。

## 相关文档

- [登录认证](../core/authentication_zh.md)
- [权限管理](../core/permission_zh.md)
- [OAuth2 指南](../security/oauth2_zh.md)
