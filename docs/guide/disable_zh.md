# 封禁体系

[English](disable.md) | 中文文档

DToken-Go 的封禁体系不只支持账号封禁，还支持服务封禁、服务等级封禁、设备类型封禁和具体设备封禁。封禁状态会参与登录态校验，被封禁对象会在访问时得到对应错误。

## 封禁类型

| 类型 | 说明 | 常见场景 |
| --- | --- | --- |
| 账号封禁 | 禁止账号继续使用登录态 | 风控、违规、冻结账号 |
| 服务封禁 | 禁止账号使用某个业务服务 | 禁言、禁止发帖、禁止支付 |
| 服务等级封禁 | 按等级判断服务限制强度 | 轻度限制、重度限制、人工审核 |
| 设备类型封禁 | 禁止账号在某类设备上访问 | 禁止 App、禁止 Web |
| 具体设备封禁 | 禁止账号在某个设备 ID 上访问 | 拉黑异常设备、处理盗号终端 |

## 账号封禁

```go
ctx := context.Background()

// 封禁账号 2 小时，并记录原因。
err := dtoken.Disable(ctx, "10001", 2*time.Hour, "risk")

disabled := dtoken.IsDisable(ctx, "10001")
err = dtoken.CheckDisable(ctx, "10001")

info, err := dtoken.GetDisableInfo(ctx, "10001")
ttl, err := dtoken.GetDisableTTL(ctx, "10001")

// 解除账号封禁。
err = dtoken.Untie(ctx, "10001")
```

`GetDisableTTL` 返回值：

| 返回值 | 含义 |
| --- | --- |
| `-2` | 未封禁 |
| `-1` | 永久封禁 |
| `> 0` | 剩余秒数 |

## 服务封禁

服务封禁适合只限制某个业务能力，而不是踢掉整个账号。

```go
err := dtoken.DisableService(ctx, "10001", "comment", 30*time.Minute)
err = dtoken.DisableServiceWithReason(ctx, "10001", "comment", 30*time.Minute, "spam")

disabled := dtoken.IsDisableService(ctx, "10001", "comment")
err = dtoken.CheckDisableService(ctx, "10001", []string{"comment", "post"})

info, err := dtoken.GetDisableServiceInfo(ctx, "10001", "comment")
ttl, err := dtoken.GetDisableServiceTTL(ctx, "10001", "comment")

err = dtoken.UntieService(ctx, "10001", "comment")
```

## 服务等级封禁

服务等级封禁会记录一个整数等级。校验时，如果实际封禁等级大于等于目标等级，则认为命中封禁。

```go
err := dtoken.DisableServiceLevel(ctx, "10001", "pay", 3, time.Hour)

dtoken.IsDisableServiceLevel(ctx, "10001", "pay", 2) // true
dtoken.IsDisableServiceLevel(ctx, "10001", "pay", 3) // true
dtoken.IsDisableServiceLevel(ctx, "10001", "pay", 4) // false

err = dtoken.CheckDisableServiceLevel(ctx, "10001", "pay", 3)
```

这个能力适合把同一个服务拆成多个限制层级，例如：

| 等级 | 示例含义 |
| --- | --- |
| `1` | 只允许查看 |
| `2` | 禁止提交 |
| `3` | 禁止支付或提现 |

## 设备封禁

设备封禁分为设备类型和具体设备 ID。

```go
// 禁止账号在 app 设备类型访问。
err := dtoken.DisableDevice(ctx, "10001", "app", time.Hour)
disabled := dtoken.IsDisableDevice(ctx, "10001", "app")
err = dtoken.CheckDisableDevice(ctx, "10001", "app")
err = dtoken.UntieDevice(ctx, "10001", "app")

// 禁止账号在某个具体设备访问。
err = dtoken.DisableDeviceAndDeviceId(ctx, "10001", "app", "iphone-001", time.Hour)
disabled = dtoken.IsDisableDeviceAndDeviceId(ctx, "10001", "app", "iphone-001")
err = dtoken.CheckDisableDeviceAndDeviceId(ctx, "10001", "app", "iphone-001")
err = dtoken.UntieDeviceAndDeviceId(ctx, "10001", "app", "iphone-001")
```

查询封禁详情：

```go
deviceInfo, err := dtoken.GetDisableDeviceInfo(ctx, "10001", "app")
deviceTTL, err := dtoken.GetDisableDeviceTTL(ctx, "10001", "app")

concreteInfo, err := dtoken.GetDisableDeviceAndDeviceIdInfo(ctx, "10001", "app", "iphone-001")
concreteTTL, err := dtoken.GetDisableDeviceAndDeviceIdTTL(ctx, "10001", "app", "iphone-001")
```

## 和登录态校验的关系

账号封禁和设备封禁会参与 Token 校验：

- 账号被封禁后，该账号已有 Token 会在校验时失败。
- 设备类型被封禁后，匹配该 `device` 的 Token 会失败。
- 具体设备被封禁后，匹配该 `device + deviceId` 的 Token 会失败。
- 服务封禁不会自动阻止所有登录态，只应在对应业务入口显式调用 `CheckDisableService` 或 `CheckDisableServiceLevel`。

## 多认证体系

封禁 API 支持可选 `authType`：

```go
err := dtoken.Disable(ctx, "admin-1", time.Hour, "risk", "admin")
disabled := dtoken.IsDisable(ctx, "admin-1", "admin")
```

不同 `AuthType` 的封禁数据互相隔离。

## 使用建议

- 账号级风险使用账号封禁。
- 业务能力限制优先使用服务封禁，不要过度封整个账号。
- 对异常终端使用具体设备封禁，避免影响用户其他正常设备。
- 需要分层风控时使用服务等级封禁。
- 封禁原因建议写入明确、可审计的业务原因。

## 相关文档

- [权限管理](permission_zh.md)
- [Session 与终端管理](session-terminal_zh.md)
- [多认证体系](multi-auth_zh.md)
- [Core API 速查](core-api-cheatsheet_zh.md)
