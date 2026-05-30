# Session 与终端管理

[English](../core/session-terminal.md) | 中文文档

## 概览

DToken-Go 使用 Session 记录一个登录 ID 下的在线终端列表。每个终端由 `TerminalInfo` 表示，核心字段包括：

- `LoginID`
- `Device`
- `DeviceId`
- `Token`
- `Index`

Token 元信息由 `TokenInfo` 保存，包含认证体系、登录 ID、设备、设备 ID、创建时间和超时时间。

## 查询 Session

```go
ctx := context.Background()

sess, err := dtoken.GetSession(ctx, "10001")
sess, err = dtoken.GetSessionByToken(ctx, token)
```

Session 可用于查看当前账号下有哪些终端在线。

## 查询终端

```go
terminal, err := dtoken.GetTerminalInfoByToken(ctx, token)
```

按账号、设备类型、具体设备统计在线数量：

```go
count, err := dtoken.GetOnlineTerminalCount(ctx, "10001")
webCount, err := dtoken.GetOnlineTerminalCountByDevice(ctx, "10001", "web")
deviceCount, err := dtoken.GetOnlineTerminalCountByDeviceAndDeviceId(ctx, "10001", "web", "browser-1")
```

## 查询 Token 列表

```go
tokens, err := dtoken.GetTokenValueListByLoginID(ctx, "10001", true)
tokens, err = dtoken.GetTokenValueListByDevice(ctx, "10001", "web", true)
tokens, err = dtoken.GetTokenValueListByDeviceAndDeviceId(ctx, "10001", "web", "browser-1", true)
```

最后一个布尔参数表示是否只返回仍然有效的 Token。

## 查询终端列表

```go
terminals, err := dtoken.GetTerminalListByLoginID(ctx, "10001")
terminals, err = dtoken.GetTerminalListByLoginIDAndDevice(ctx, "10001", "web")
```

## 获取最新 Token

```go
token, err := dtoken.GetTokenValueByLoginID(ctx, "10001")
token, err = dtoken.GetTokenValueByLoginIDAndDevice(ctx, "10001", "web")
```

## 遍历终端

```go
err := dtoken.ForEachTerminal(ctx, "10001", func(info manager.TerminalInfo) bool {
    fmt.Println(info.Device, info.DeviceId, info.Token)
    return true
})

err = dtoken.ForEachTerminalByDevice(ctx, "10001", "web", func(info manager.TerminalInfo) bool {
    return true
})
```

回调返回 `false` 时会停止遍历。

## 搜索

```go
tokens, err := dtoken.SearchTokenValue(ctx, "keyword", 0, 20)
sessions, err := dtoken.SearchSessionId(ctx, "keyword", 0, 20)
```

这些能力依赖存储实现支持 key 扫描。内置内存存储和 Redis 存储均支持。

## logout、kickout、replace 区别

| 操作 | 行为 |
|------|------|
| logout | 删除 Token 映射，后续表现为未登录 |
| kickout | 保留状态标记，后续表现为被踢下线 |
| replace | 保留状态标记，后续表现为被顶下线 |

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
6. 账号、设备或服务封禁导致校验失败

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
