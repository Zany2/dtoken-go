# Session 与终端管理

代码片段中，dt = github.com/Zany2/dtoken-go。

[English](../core/session-terminal.md) | 中文文档

## 概览

DToken-Go 使用 Session 记录一个登录 ID 下的终端索引，其中可能保留已过期的条目。每个终端由 `TerminalInfo` 表示，核心字段包括：

- `LoginID`
- `Device`
- `DeviceID`
- `Token`
- `Index`

Token 元信息由 `TokenInfo` 保存，包含认证体系、登录 ID、设备、设备 ID、创建时间和超时时间。

## 查询 Session

```go
ctx := context.Background()

sess, err := dt.GetSession(ctx, "10001")
sess, err = dt.GetSessionByToken(ctx, token)
```

Session 可用于查看账号的终端记录；判断当前在线状态请使用有效 Token 列表或在线数量接口。

编码空值 Session 无效，返回 `ErrInvalidToken`，不会被转换成空会话。合法旧版 Session 对象可以缺少认证体系和账号字段，读取时会按当前命名空间及请求的存储键补齐；仅终端列表为空不会使 Session 无效。

登录和续期会保留 Session 已有的更长 TTL，包括无限期。需要读取原 TTL 时，如果查询失败，不会按本次请求的有效期覆盖 Session。同步操作返回 `ErrStorageUnavailable`；异步续期不发布成功事件。此检查不会回滚多步骤操作中已经完成的其他写入。

## 会话扩展数据

```go
err := dt.SetSessionValueByToken(ctx, token, "theme", "dark")
value, exists, err := dt.GetSessionValueByToken(ctx, token, "theme")
err = dt.DeleteSessionValueByToken(ctx, token, "theme")
```

按 Token 设置或删除数据时，先读取轻量主记录定位账号，再在账号锁内复核同一生命周期并加载一次 Session，避免等待锁期间 Token 被注销或复用后继续写入。正常路径共读取两次 Token 主记录、一次 Session，Session 只解码一次；启用不活跃超时时另读一次活跃标记，不读取 Token Extra。按 Token 查询数据只读取一次主记录和一次 Session。

按账号调用 `SetSessionValue`、`GetSessionValue`、`DeleteSessionValue` 直接访问该账号的 Session，不校验 Token。数据写入保留已有 Session TTL，删除不存在的字段不编码或写回；这些数据接口不触发自动续期或活跃刷新。按 Token 操作仍拒绝缺失、损坏或属于其他认证体系的 Session，不活跃超时的清理事件在释放账号锁后发布。以上读取次数针对正常路径，超时清理和失败路径另有开销。

## 查询终端

```go
terminal, err := dt.GetTerminalInfoByToken(ctx, token)
```

按账号、设备类型、具体设备统计在线数量：

```go
count, err := dt.GetOnlineTerminalCount(ctx, "10001")
webCount, err := dt.GetOnlineTerminalCountByDevice(ctx, "10001", "web")
deviceCount, err := dt.GetOnlineTerminalCountByDeviceAndDeviceID(ctx, "10001", "web", "browser-1")
```

## 查询 Token 列表

```go
tokens, err := dt.GetTokenValueListByLoginID(ctx, "10001", true)
tokens, err = dt.GetTokenValueListByDevice(ctx, "10001", "web", true)
tokens, err = dt.GetTokenValueListByDeviceAndDeviceID(ctx, "10001", "web", "browser-1", true)
```

最后一个布尔参数表示是否只返回仍然有效的 Token。

Manager 的 Token 列表方法最多接收一个可选 `checkAlive` 值；本文展示的根包函数必须显式传入布尔值，通过 dt.AuthType(name) 选择体系。开启时，列表、在线计数和最新 Token 查询都会核对终端与 Token 的账号、设备、创建时间及生命周期序号，忽略不匹配的陈旧条目。按 Token 查询终端详情同样执行匹配，不返回旧登录的设备或扩展数据。

## 查询终端列表

```go
terminals, err := dt.GetTerminalListByLoginID(ctx, "10001")
terminals, err = dt.GetTerminalListByLoginIDAndDevice(ctx, "10001", "web")
```

## 获取最新 Token

Manager 的 `GetTerminalListByLoginID` 和 `GetTokenValueByLoginID` 最多接受一个可选设备类型，多余参数返回错误。

```go
token, err := dt.GetTokenValueByLoginID(ctx, "10001")
token, err = dt.GetTokenValueByLoginIDAndDevice(ctx, "10001", "web")
```

## 遍历终端

```go
err := dt.ForEachTerminal(ctx, "10001", func(info dt.TerminalInfo) bool {
    fmt.Println(info.Device, info.DeviceID, info.Token)
    return true
})

err = dt.ForEachTerminalByDevice(ctx, "10001", "web", func(info dt.TerminalInfo) bool {
    return true
})
```

回调返回 `false` 时会停止遍历。

终端列表、遍历及未开启 `checkAlive` 的 Token 列表返回存储快照，可能包含过期或陈旧条目；查询不会删除这些记录。Session 数据中的“键存在且值为 nil”与“键不存在”由 `GetSessionValue` / `GetSessionValueByToken` 的布尔返回值区分。

登录不再无条件检查全部历史 Token。新登录在记录数触达数量上限时核实有效名额，共享登录只在记录数超过上限时整理；按设备限制时，只处理该设备类型。禁止并发的拒绝策略找到有效占位便返回，顶替策略先整理目标作用域。临时设备封禁仍占用名额，不被视作过期。

共享候选按新到旧查找，命中后不再全量清扫。刷新轮换只检查原生命周期，过期访问凭据的索引允许暂存。精确列表和在线数量逐项核对身份、生命周期及有效性，但不重复遍历 Session 查找已提供的终端。

按设备类型或具体设备查询 Token 列表、在线数量时，直接在已加载的 Session 中筛选并处理，不先复制匹配的终端列表；范围外的 Token 不读取。最新 Token 查询同样在原列表上反向筛选，遇到第一个有效匹配便返回。普通 Token 列表只读取 Session；有效列表和计数仍需逐项读取范围内的主记录及必要状态，单次查询复用账号和设备封禁检查。这些调整减少中间分配和重复遍历，不省略 Session 本身的读取、解码，也不改变结果顺序或增加清理副作用。

按需整理对每个选中的非空 Token 条目只读取一次主记录，并复用结果进行活跃超时判断、Extra 删除及状态写入。主键自然过期时，仅清理访问令牌的活跃和续期元数据，保留仍有效的刷新凭据；活跃标记缺失、损坏或已超时则按原规则下线并撤销刷新关联。已有下线标记在清理刷新关联前核对其账号和访问令牌归属，相关刷新读取另计。其他命名空间或解析为无效凭据的已有值仅移除陈旧 Session 条目，不据此删除附属数据；存储读取和类型转换错误仍返回调用方。

鉴权或刷新发现不活跃超时时，仍保留下线前的生命周期复核，并将复核结果用于状态和 Extra 清理。整理仅修改内存中的终端列表，Session 保存由登录流程统一完成；清理中途失败不会提前写回不完整列表，也不回滚此前已经完成的凭据下线。

未触发整理时，新登录保留仍存在的 Session 及其账号数据；自然过期条目不会触发即时下线事件。高频登录、无限终端数且 Session 长期存活时，索引可能持续增长；Session 读取和序列化仍随列表大小增加。本轮不增加后台全库清理器。

## 搜索

```go
tokens, err := dt.SearchTokenValue(ctx, "keyword", 0, 20)
sessions, err := dt.SearchSessionId(ctx, "keyword", 0, 20)
```

这些能力依赖存储实现支持 key 扫描。内置内存存储和 Redis 存储均支持。

## logout、kickout、replace 区别

| 操作 | 行为 |
|------|------|
| logout | 删除 Token 映射，后续表现为未登录 |
| kickout | 保留状态标记，后续表现为被踢下线 |
| replace | 保留状态标记，后续表现为被顶下线 |

按 Token 下线在账号或设备封禁期间仍可执行；已确认归属的残留 Token 即使 Session 缺失也会被清理，并撤销关联刷新令牌。已有 kickout、replace 或活跃超时标记保持幂等，不改写下线原因。

高级 `Manager.Terminate` 中有效 Token 优先于账号、设备筛选；显式传入纯空白 Token 或设备筛选字段会返回参数错误，不会回退到更大的操作范围。

实例门面通过 `auth.Logout`、`auth.Kickout` 或 `auth.Replace` 接收根包 `LogoutOptions`；高级 Manager 的 `Terminate` 使用 `TerminateOptions`，两者不能直接互换。

三类操作都支持：

- 按 Token
- 按登录 ID
- 按设备类型
- 按设备类型 + 设备 ID

## Token 生命周期

Token 可能因以下原因失效：

1. 超过绝对 TTL
2. 超过 `ActiveTimeout`
3. 被 logout 删除
4. 被 kickout 标记
5. 被 replace 标记
6. 账号封禁废弃 Session 索引中的凭据

设备封禁保留凭据，通过显式访问检查执行限制；服务封禁只阻止相应的显式服务校验。二者都不会直接使基础登录校验失败。

## 测试覆盖

`tests/gin_core_flow` 覆盖了：

- Session 查询
- 多端登录
- 当前终端信息
- Token 列表查询
- 终端列表查询
- 在线终端数量
- 遍历和搜索
- 按账号、设备、具体设备 logout/kickout/replace
- alive 过滤

## 相关文档

- [登录认证指南](../core/authentication_zh.md)
- [并发登录策略](../core/concurrency-login_zh.md)
- [核心流程测试指南](../integration/core-flow-testing_zh.md)
