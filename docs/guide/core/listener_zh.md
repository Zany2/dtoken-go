# 事件监听指南

[English](../core/listener.md) | 中文文档

## 当前状态

当前项目内部已经实现了完整的事件系统，底层代码位于 `core/listener`。

它覆盖认证与会话生命周期、访问检查与变更、刷新令牌、Nonce、Ticket、ShortKey 和 OAuth2 等流程。

## 获取事件管理器

当前版本可通过全局门面、实例门面和 `manager.Manager` 获取事件管理器：

```go
globalEventMgr, err := dtoken.GetEventManager("user")
if err != nil {
    // 处理查找错误。
}
instanceEventMgr := auth.EventManager()
coreEventMgr := mgr.GetEventManager()
```

取得 `*listener.Manager` 后，可使用 `RegisterFunc`、`RegisterFuncWithConfig` 等注册 API。全局门面、实例门面和核心 Manager 是三种可选入口，应选择当前应用实际持有的管理器。

## 内部事件类型

当前定义在 `core/listener/consts.go` 中的内置事件有：

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

事件载荷结构在 `core/listener/listener.go` 中定义：

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

触发方法会取得事件载荷快照。每个监听器都会收到独立的标量字段副本和顶层 `Extra` map 副本。这里采用浅拷贝：`Extra` 中嵌套的 map、slice、指针和其他引用值仍然共享，过滤器和监听器必须将这些嵌套引用视为只读数据。

Manager 的触发诊断日志只记录事件类型、认证体系、时间戳和监听器数量，不会写入 `LoginID`、设备字段、Token 或 `Extra`。应用监听器和自定义 panic 处理器仍需自行保护收到的事件载荷。

其中：

- `AuthType`：当前认证体系
- `LoginID`：账号 ID
- `Device` / `DeviceID`：终端信息
- `Token`：相关 token
- `Extra`：额外信息，比如权限校验结果、服务封禁等级等

## Extra 字段常量

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

`core/listener` 是公开包，也可以独立使用。

### 创建监听管理器

```go
import (
    "github.com/Zany2/dtoken-go/core/listener"
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

- `Trigger` 在当前 goroutine 中分发；同步监听器内联执行，异步监听器启动后不会等待其完成。
- `TriggerAsync` 在返回前取得载荷快照，再异步调度整个分发过程并立即返回。
- `TriggerSync` 在当前 goroutine 中分发，并等待本次分发直接启动的监听器完成；它不会等待无关异步任务，也不会等待监听器通过 `Trigger` 或 `TriggerAsync` 启动的后台后代任务，但嵌套调用 `TriggerSync` 仍属于当前监听器的完成过程。
- `Wait` 会等待事件管理器已接收的异步分发和异步监听器任务全部结束。

具体事件监听器和 `EventAll` 通配监听器会合并后按优先级降序遍历。对于异步监听器，优先级只控制启动遍历顺序；goroutine 调度不保证回调的实际执行或完成顺序。

```go
eventMgr.Trigger(data)
eventMgr.TriggerAsync(data)
eventMgr.TriggerSync(data)
```

## 高级能力

### 全局过滤器

```go
eventMgr.AddFilter(func(data *listener.EventData) bool {
    return data.AuthType == "dtoken:"
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

## 相关文档

- [登录认证](../core/authentication_zh.md)
- [权限管理](../core/permission_zh.md)
- [OAuth2 指南](../security/oauth2_zh.md)
