# 封禁体系

[English](../core/disable.md) | 中文文档

本文 `dt` 表示根包。`Build()` 自动注册；默认体系使用 `dt.XXX()`，其他体系使用 `dt.AuthType("user").XXX()`，业务方法签名与实例一致。见 [Auth API](../../api/auth_zh.md)。

DToken-Go 支持账号、服务、服务等级、设备类型和具体设备封禁。基础登录校验只检查凭据有效性；需要限制访问的入口必须显式声明对应封禁检查。

## 封禁类型

| 类型 | 说明 | 常见场景 |
| --- | --- | --- |
| 账号封禁 | 禁止账号继续使用登录态 | 风控、违规、冻结账号 |
| 服务封禁 | 禁止账号使用某个业务服务 | 禁言、禁止发帖、禁止支付 |
| 服务等级封禁 | 按等级判断服务限制强度 | 轻度限制、重度限制、人工审核 |
| 设备类型封禁 | 禁止账号在某类设备上访问 | 禁止 App、禁止 Web |
| 具体设备封禁 | 禁止账号在某个设备 ID 上访问 | 拉黑异常设备、处理盗号终端 |

## 账号封禁

`Disable` 会销毁账号当前的 Session，并同步使其中记录的已有终端凭证失效、清理关联刷新令牌。基础校验对这些凭据返回 `ErrInvalidToken`；显式封禁检查在封禁期间返回账号封禁错误。解封或封禁到期后，它们仍然无效，需要重新登录。`Untie` 只解除封禁，不恢复已销毁的 Session。

终端凭据清理对每个非空 Token 主键只读取一次，直接区分键缺失、下线标记和凭据归属；无法确认归属的已有值不会被当作键缺失清理。有效凭据保留原有详情与剩余 TTL，仅标记为已废弃，不读取 Extra 内容。访问凭据已经过期时，仍检查刷新关联归属后撤销刷新凭据；归属检查与刷新删除的读取成本另计。Session 在清理成功后才删除，清理失败会返回错误并保留索引供重试。

封禁时长为 `0` 表示永久封禁，负数无效。重新设置封禁只更新规范键中的期限和原因；运行时查询、写入和解封均不探测或删除旧键。切换前须通过 [PrepareAuthUpgrade](../../design/namespace-copy_zh.md) 转换旧封禁。规范键中的记录无法解码、缺少 loginId/kind 或属于其他身份时，拒绝覆盖。

```go
ctx := context.Background()

// 封禁账号 2 小时，并记录原因。
err := dt.Disable(ctx, dt.DisableOptions{LoginID: "10001", Duration: 2*time.Hour, Reason: "risk"})

disabled := dt.IsDisable(ctx, "10001")
err = dt.CheckDisable(ctx, "10001")

info, err := dt.GetDisableInfo(ctx, "10001")
ttl, err := dt.GetDisableTTL(ctx, "10001")

// 解除账号封禁。
err = dt.Untie(ctx, "10001")
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
err := dt.DisableService(ctx, dt.ServiceDisableOptions{LoginID: "10001", Service: "comment", Duration: 30*time.Minute})
err = dt.DisableServiceWithReason(ctx, "10001", "comment", 30*time.Minute, "spam")

disabled := dt.IsDisableService(ctx, "10001", "comment")
err = dt.CheckDisableService(ctx, "10001", []string{"comment", "post"})

info, err := dt.GetDisableServiceInfo(ctx, "10001", "comment")
ttl, err := dt.GetDisableServiceTTL(ctx, "10001", "comment")

err = dt.UntieService(ctx, "10001", "comment")
```

## 服务等级封禁

服务等级封禁会记录一个整数等级。校验时，如果实际封禁等级大于等于目标等级，则认为命中封禁。

```go
err := dt.DisableServiceLevel(ctx, "10001", "pay", 3, time.Hour)

dt.IsDisableServiceLevel(ctx, "10001", "pay", 2) // true
dt.IsDisableServiceLevel(ctx, "10001", "pay", 3) // true
dt.IsDisableServiceLevel(ctx, "10001", "pay", 4) // false

err = dt.CheckDisableServiceLevel(ctx, "10001", "pay", 3)
```

这个能力适合把同一个服务拆成多个限制层级，例如：

| 等级 | 示例含义 |
| --- | --- |
| `1` | 只允许查看 |
| `2` | 禁止提交 |
| `3` | 禁止支付或提现 |

## 设备封禁

设备封禁分为设备类型和具体设备 ID。

设备封禁保留 Session 和终端凭证；解封后，只要凭证尚未过期或被注销，就可以继续使用。设备类型与具体设备的封禁独立存在，解除其中一个不会解除另一个。具体设备的 `IsDisable` / `CheckDisable` 会同时检查两层限制，而详情和 TTL 查询只返回指定层级的记录。服务封禁同样不销毁 Session；重新设置服务或设备封禁只更新对应规范键。

```go
// 禁止账号在 app 设备类型访问。
err := dt.DisableDevice(ctx, dt.DeviceDisableOptions{LoginID: "10001", Device: "app", Duration: time.Hour})
disabled := dt.IsDisableDevice(ctx, "10001", "app")
err = dt.CheckDisableDevice(ctx, "10001", "app")
err = dt.UntieDevice(ctx, "10001", "app")

// 禁止账号在某个具体设备访问。
err = dt.DisableDeviceAndDeviceID(ctx, "10001", "app", "iphone-001", time.Hour)
disabled = dt.IsDisableDeviceAndDeviceID(ctx, "10001", "app", "iphone-001")
err = dt.CheckDisableDeviceAndDeviceID(ctx, "10001", "app", "iphone-001")
err = dt.UntieDeviceAndDeviceID(ctx, "10001", "app", "iphone-001")
```

查询封禁详情：

```go
deviceInfo, err := dt.GetDisableDeviceInfo(ctx, "10001", "app")
deviceTTL, err := dt.GetDisableDeviceTTL(ctx, "10001", "app")

concreteInfo, err := dt.GetDisableDeviceAndDeviceIDInfo(ctx, "10001", "app", "iphone-001")
concreteTTL, err := dt.GetDisableDeviceAndDeviceIDTTL(ctx, "10001", "app", "iphone-001")
```

## 和登录态校验的关系

凭据校验与访问限制分开执行：

- 账号封禁仍废弃 Session 索引中的已有凭据，基础校验返回 `ErrInvalidToken`，解封不恢复这些凭据。索引外的凭据需要显式封禁检查才能完整受限。
- 设备类型或具体设备封禁是可解除的访问限制，不直接废弃凭据；`CheckLogin` / `IsLogin` 可以成功。HTTP `CheckDisable: true` 检查账号、设备类型和具体设备，受限时返回相应错误。
- 新登录、刷新签发和自省仍检查这些限制。业务层可按需调用账号及设备的 `CheckDisable*` 方法；不能只调用账号检查来代替设备检查。
- 服务封禁不会自动阻止所有登录态，只应在对应业务入口显式调用 `CheckDisableService` 或 `CheckDisableServiceLevel`。

封禁查询按作用域读取规范键：账号、单个服务或设备类型检查各一次 GET；具体设备检查先看设备类型，未命中再看具体设备，最多两次 GET。TTL 查询先读取并核对归属，存在匹配记录时再查询一次 TTL。解封复用已读取的记录，普通存储执行 Delete，支持原子能力时执行 GetAndDelete；不存在记录或删除失败不发送成功事件。封禁详情仍使用配置的 Codec 解码，普通登录校验不因此增加封禁读取。

## 多认证体系

封禁 API 支持可选 `authType`：

```go
err := dt.AuthType("admin").Disable(ctx, dt.DisableOptions{LoginID: "admin-1", Duration: time.Hour, Reason: "risk"})
disabled := dt.AuthType("admin").IsDisable(ctx, "admin-1")
```

不同 `AuthType` 的封禁数据互相隔离。

## 使用建议

- 账号级风险使用账号封禁。
- 业务能力限制优先使用服务封禁，不要过度封整个账号。
- 对异常终端使用具体设备封禁，避免影响用户其他正常设备。
- 需要分层风控时使用服务等级封禁。
- 封禁原因建议写入明确、可审计的业务原因。

## 相关文档

- [权限管理](../core/permission_zh.md)
- [Session 与终端管理](../core/session-terminal_zh.md)
- [多认证体系](../core/multi-auth_zh.md)
- [Core API 速查](../reference/core-api-cheatsheet_zh.md)
