# 登录认证指南

[English](../core/authentication.md) | 中文文档

本文 `dt` 表示根包。`Build()` 自动注册；默认体系使用 `dt.XXX()`，其他体系使用 `dt.AuthType("user").XXX()`，业务方法签名与实例一致。见 [Auth API](../../api/auth_zh.md)。

## 基本登录

### 初始化

<!-- dtoken-compile: complete-example -->
```go
package main

import (
    dt "github.com/Zany2/dtoken-go"
    "context"

    "github.com/Zany2/dtoken-go/storage/memory"
)

func initDToken() {
    _, err := dt.NewBuilder().
        Storage(memory.NewStorage()).
        Build()
    if err != nil {
        panic(err)
    }
}

func main() {
    initDToken()
    defer dt.CloseAll()

    ctx := context.Background()
    token, _ := dt.Login(ctx, dt.LoginOptions{LoginID: "10001"})
    _ = token
}
```

### 简单登录

```go
ctx := context.Background()

// 默认登录
token, err := dt.Login(ctx, dt.LoginOptions{LoginID: "10001"})

// 指定设备类型
token, err = dt.Login(ctx, dt.LoginOptions{LoginID: "10001", Device: "web"})

// 指定设备类型和设备 ID
token, err = dt.Login(ctx, dt.LoginOptions{LoginID: "10001", Device: "app", DeviceID: "ios-iphone-15"})
```

### 指定过期时间登录

```go
ctx := context.Background()

// 当前 token 单独设置 2 小时过期
token, err := dt.Login(ctx, dt.LoginOptions{LoginID: "10001", Timeout: 2*time.Hour})

// 同时指定设备类型和设备 ID
token, err = dt.Login(ctx, dt.LoginOptions{LoginID: "10001", Timeout: 2*time.Hour, Device: "web", DeviceID: "chrome-mac"})
```

### 基于旧 Token 续期登录

```go
ctx := context.Background()

err := dt.LoginByToken(ctx, token)
```

`LoginByToken()` 会在当前 token 仍然有效时，异步续期 token、session、活跃时间等相关信息。

自动续期使用存储返回的精确剩余时长比较阈值；不足一秒但尚未过期的 Token 仍可续期。共享登录会使此前排队的维护任务失效，防止旧任务覆盖较新的活跃时间。有限期 Token 若在实际续期前过期，不会发送续期成功事件。

## 检查登录状态

`IsLogin`、`CheckLogin` 和基础身份查询只校验凭据存在、状态及启用的活跃超时，不隐式查询账号或设备封禁。新格式且未启用活跃超时的正常前台校验只需一次 Token GET，不加载 Session、Extra 或通用 Codec；异步维护的读取另计。

需要执行访问限制时，HTTP 组合检查显式设置 `CheckDisable: true`，或在业务入口调用对应的 `CheckDisable` / `CheckDisableDevice` / `CheckDisableDeviceAndDeviceID`。角色权限检查和按 Token 访问 Session 也不代表已经执行封禁检查。自省、新登录和刷新签发继续检查限制。

```go
ctx := context.Background()

// 返回布尔值
isLogin := dt.IsLogin(ctx, token)

// 未登录时返回错误
err := dt.CheckLogin(ctx, token)
```

## 获取登录信息

### 获取登录 ID

```go
ctx := context.Background()

loginID, err := dt.GetLoginID(ctx, token)
```

### 获取 Token 信息

`GetTokenInfo` 读取已存储的元数据，成功返回不代表登录态有效；它不检查账号或设备封禁、活跃超时及 Session 有效性。普通登录态判断请使用 `IsLogin` 或 `CheckLogin`。这些方法以及 `GetLoginID` 等身份查询校验凭据及启用的活跃超时，正常路径不读取账号 Session 或封禁记录。访问凭据只解析固定字段，旧 Token 载荷须离线转换，运行时不回退 Codec 解码；存储 TTL 决定绝对过期，续期资格交由后台维护判断。

依赖 Session 内容的操作及 `IntrospectToken` 仍校验账号 Session。按账号或 Token 查询权限、角色均优先使用 AccessProvider，仅未配置或返回 nil 时加载 Session。不活跃超时清理也会加载 Session。单独删除或损坏 Session 不再使普通登录校验中的 Token 失效；撤销凭据应使用注销或踢人 API。自动续期只检查 Session TTL，键缺失时停止，不解码其内容。仅活跃维护不依赖 Session 存在或内容可解码，仍复核凭据生命周期及封禁状态。

```go
ctx := context.Background()

info, err := dt.GetTokenInfo(ctx, token)
if err == nil {
    fmt.Println("认证体系:", info.AuthType)
    fmt.Println("登录 ID:", info.LoginID)
    fmt.Println("设备类型:", info.Device)
    fmt.Println("设备 ID:", info.DeviceID)
    fmt.Println("创建时间:", info.CreateTime)
    fmt.Println("有效期:", info.Timeout)
    fmt.Println("不活跃超时:", info.ActiveTimeout)
    fmt.Println("扩展数据:", info.Extra)
}
```

当前 `TokenInfo` 包含：

- `AuthType`
- `LoginID`
- `Device`
- `DeviceID`
- `CreateTime`
- `Timeout`（秒）
- `ActiveTimeout`（秒；`0` 表示继承 Manager 配置）
- `Extra`

### 获取 Token 其他信息

```go
ctx := context.Background()

device, err := dt.GetDevice(ctx, token)
deviceID, err := dt.GetDeviceID(ctx, token)
createTime, err := dt.GetTokenCreateTime(ctx, token)
ttl, err := dt.GetTokenTTL(ctx, token)
```

## 登出

### 根据 Token 登出

```go
ctx := context.Background()

err := dt.LogoutByToken(ctx, token)
```

### 根据账号维度登出

```go
ctx := context.Background()

// 登出指定账号的所有终端
err := dt.LogoutByLoginID(ctx, "10001")

// 登出指定账号下某个设备类型的所有终端
err = dt.LogoutByDevice(ctx, "10001", "web")

// 登出指定账号下某个设备类型 + 设备 ID 的终端
err = dt.LogoutByDeviceAndDeviceID(ctx, "10001", "app", "ios-iphone-15")
```

## 踢人下线

### 根据 Token 踢人

```go
ctx := context.Background()

err := dt.KickoutByToken(ctx, token)
```

### 根据账号维度踢人

```go
ctx := context.Background()

err := dt.KickoutByLoginID(ctx, "10001")
err = dt.KickoutByDevice(ctx, "10001", "web")
err = dt.KickoutByDeviceAndDeviceID(ctx, "10001", "app", "ios-iphone-15")
```

被踢下线的 token 不会立即从存储中删除，而是会被标记为 `kickout` 状态，后续校验时会返回对应错误。

## 顶人下线

### 根据 Token 顶人

```go
ctx := context.Background()

err := dt.ReplaceByToken(ctx, token)
```

### 根据账号维度顶人

```go
ctx := context.Background()

err := dt.ReplaceByLoginID(ctx, "10001")
err = dt.ReplaceByDevice(ctx, "10001", "web")
err = dt.ReplaceByDeviceAndDeviceID(ctx, "10001", "app", "ios-iphone-15")
```

被顶下线的 token 会被标记为 `replaced` 状态，适合用在“新登录挤掉旧登录”的场景中。

### 下线操作的处理成本

账号、设备类型和具体设备入口在账号锁内读取一次 Session，筛选目标终端。正常有效凭据的主记录每个只读取一次：归属检查、存活判断和 Extra 清理复用该记录，不读取 Extra 的内容。踢下线、顶替直接使用已匹配的终端条目，不逐项重新遍历 Session，也不复制整份终端、角色或权限列表。

普通登出批量删除目标主记录及其 Extra；踢下线和顶替逐项保存状态，优先沿用当前剩余 TTL。续期键和活跃键统一批量删除；存在刷新关联时，仍读取并核对刷新凭据后撤销。最后只保存一次变化后的 Session，或删除空 Session，并在释放账号锁后发布事件。

上述“主记录一次读取”适用于正常的账号/设备批量清理阶段，不代表整个下线操作只访问一次存储。直接调用 `Logout`、`Kickout`、`Replace` 时，锁前只校验凭据及已启用的活跃超时，Session 留待账号锁内清理时加载。正常单 Token 路径的 Session 读取及解码由两次降为一次，不加载 Extra；主记录捕获、校验、入锁生命周期复核及清理归属检查仍分别执行，不能理解为整个操作只读一次。Session 缺失时仍可清理显式指定的凭据，损坏、其他认证体系的 Session 或存储故障则返回错误。疑似不活跃超时仍加载上下文并在锁内复核后变更状态，保留超时原因和事件；失效、外部归属或刷新关联异常等路径可能需要额外读取。

这些步骤按顺序执行，不提供跨键事务：中途存储失败返回错误，可能已有部分凭据下线；Session 索引在清理成功后才写回，以便再次按账号或设备清理。重试踢下线或顶替时，不会为此前已经写入的状态重复发布事件。

## 在线终端统计

```go
ctx := context.Background()

count, err := dt.GetOnlineTerminalCount(ctx, "10001")
webCount, err := dt.GetOnlineTerminalCountByDevice(ctx, "10001", "web")
singleCount, err := dt.GetOnlineTerminalCountByDeviceAndDeviceID(ctx, "10001", "app", "ios-iphone-15")
```

## 登录配置

下面各配置片段是独立的构建方案，不要在同一进程中连续构建同名体系；需要同名独立实例时使用 BuildInstance。

### 并发登录

```go
auth, err := dt.NewBuilder().
    Storage(memory.NewStorage()).
    IsConcurrent(true).
    Build()
if err != nil {
    panic(err)
}
defer auth.Close()
```

`IsConcurrent(true)` 表示允许同账号多端并发登录。  
`IsConcurrent(false)` 表示新登录时会处理旧终端。

### 共享 Token

```go
auth, err := dt.NewBuilder().
    Storage(memory.NewStorage()).
    IsConcurrent(true).
    IsShare(true).
    Build()
if err != nil {
    panic(err)
}
defer auth.Close()
```

`IsShare(true)` 表示并发登录时可复用已有 token，但只会在相同设备维度内复用。  
如果同时设置 `IsConcurrent(true)` 且 `IsShare(false)`，则每次登录都会生成新 token。

### 最大登录数

```go
auth, err := dt.NewBuilder().
    Storage(memory.NewStorage()).
    IsConcurrent(true).
    IsShare(false).
    MaxLoginCount(3).
    Build()
if err != nil {
    panic(err)
}
defer auth.Close()
```

当 `MaxLoginCount > 0` 时，超过上限的旧终端会被自动清理。

### 登录中的 Session 写入

新登录将过期终端整理、超限淘汰或顶替，以及新终端追加合并到一次最终 Session 保存，不再每移除一个终端就序列化并写回整个会话。共享登录未改变列表时不保存 Session 内容；需要整理或淘汰时，完成后保存一次。Token 状态、附属键及 TTL 维护仍单独执行，上述次数仅指 Session 内容写入。

这里统计的是一次内部登录阶段；刷新令牌轮换还包含旧终端退出，退出阶段的 Session 写入另计。

普通顶替保留账号数据、历史计数及更长的已有 Session TTL；如果按需整理发现所有旧终端都失效，则仍按原规则重建 Session，重置账号数据和历史计数，并采用本次登录的有效期。重建成功时直接写入新会话，省去中间的 Session 删除，销毁、创建及登录事件仍在释放账号锁后按顺序发布。

登录触发的顶替、超限淘汰及共享登录淘汰，会将账号锁内核对过的主记录直接用于状态写入和 Extra 删除，不再次读取主记录或 Extra 内容。登录前核实有效占位仍有独立读取：正常候选被检查一次、选中下线后再核对一次；这不表示整个登录只有一次 GET。下线状态沿用当前剩余 TTL，TTL 不可用时复用已知凭据的期限。

中途失败会尝试保存已完成的列表清理；新 Token 持久化失败仍移除本次追加的终端。失败路径可能有额外写入，不恢复已经下线的旧凭据，也不提供跨键事务保证。

## 自动续期

### 配置方式

```go
auth, err := dt.NewBuilder().
    Storage(memory.NewStorage()).
    Timeout(24 * 60 * 60).
    AutoRenew(true).
    RenewMaxRefresh(30 * 60).
    RenewInterval(60).
    Build()
if err != nil {
    panic(err)
}
defer auth.Close()
```

### 工作机制

当前版本的自动续期并不是每次 `IsLogin()` 都无条件刷新，而是：

1. 校验 token 是否有效
2. 仅在显式访问检查声明封禁规则时，同步检查账号和设备限制
3. 检查 `ActiveTimeout`
4. 当 `AutoRenew=true` 且 token 还有剩余 TTL 时，根据 `RenewMaxRefresh` 与 `RenewInterval` 判断是否需要异步续期

这套机制可以减少高频访问时的重复续期。

## 活跃超时

### 配置方式

```go
auth, err := dt.NewBuilder().
    Storage(memory.NewStorage()).
    Timeout(24 * 60 * 60).
    ActiveTimeout(30 * 60).
    Build()
if err != nil {
    panic(err)
}
defer auth.Close()
```

### 工作机制

开启 `ActiveTimeout` 后，系统会为 token 维护活跃时间：

1. 登录成功时初始化活跃时间
2. 鉴权校验时检查是否已超过活跃超时
3. 未超时则异步刷新活跃时间
4. 已超时则标记为 `active-timeout`，后续校验会继续返回对应错误

`Timeout` 表示绝对过期时间，`ActiveTimeout` 表示不活跃超时，两者可以同时存在。

## 完整配置示例

```go
auth, err := dt.NewBuilder().
    Storage(memory.NewStorage()).
    TokenName("dtoken").
    Timeout(24 * 60 * 60).
    ActiveTimeout(30 * 60).
    AutoRenew(true).
    RenewMaxRefresh(30 * 60).
    RenewInterval(60).
    IsConcurrent(true).
    IsShare(false).
    MaxLoginCount(3).
    IsReadHeader(true).
    Build()
if err != nil {
    panic(err)
}
defer auth.Close()
```

## 相关文档

- [快速开始](../../tutorial/quick-start_zh.md)
- [权限管理](../core/permission_zh.md)
- [注解鉴权](../integration/annotation_zh.md)
- [JWT 指南](../security/jwt_zh.md)
