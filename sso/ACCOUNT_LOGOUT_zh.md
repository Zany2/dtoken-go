# SSO 账号强制下线

[English](ACCOUNT_LOGOUT.md) · [注销通知](LOGOUT_RELIABILITY_zh.md) · [部署](DEPLOYMENT_zh.md)

`HTTPServer.LogoutAccount(r, loginID)` 为可信后台流程组合中心会话撤销、账号 SSO 凭证失效、子应用通知及绑定清理。适用于管理员强制下线、账号封禁后的会话清理，以及中心 Cookie 已失效后的受授权重试。范围是账号在当前 SSO 存储命名空间内的全部客户端，不提供设备或单次登录筛选。

此入口不新增 HTTP 路由，不解析浏览器 Cookie、表单账号或 `LoginIDResolver`，也不执行管理员鉴权。调用方必须先验证操作者对目标账号的操作权限。`r` 传递 Context 及应用钩子需要的请求信息；`loginID` 必须来自已授权的目标选择，不能直接透传公开请求参数。

## 接入

复用已经配置好签名、防重放、注册信息及通知客户端的 `HTTPServer`，中心撤销函数必须显式提供：

```go
options := sso.DefaultHTTPOptions()
// 配置注册信息、签名、防重放、LoginIDResolver 等应用选项。
options.CenterLogout = func(r *http.Request, loginID string) error {
    return centerAuth.LogoutByLoginID(r.Context(), loginID)
}
handler, err := sso.NewHTTPServerChecked(server, options)
if err != nil {
    return err
}

// 此前已完成管理员鉴权和目标授权；封禁操作还应先阻止账号新登录／签发。
result, err := handler.LogoutAccount(r, authorizedLoginID)
// 应用审计记录 result.Stage、各阶段标记及逐目标状态；不要公开原始 err。
if err != nil {
    return err
}
_ = result.Complete
```

完整可编译示例源码见 [example_account_logout_test.go](example_account_logout_test.go)。`sso/httpserver.Server` 同样具备此方法，并导出 `AccountLogoutResult` 类型别名。完整部署示例保留 `application.centerHTTP`，供应用自己的可信后台逻辑接入；没有公开管理接口，固定账号演示认证器也不具备账号封禁能力。

`CenterLogout` 应撤销传入 `loginID` 的全部中心会话并支持重复调用，不能根据管理员自己的 Cookie 推断目标。未提供钩子时返回 `ErrAccountLogoutConfig`，在任何撤销或通知前停止。该要求仅针对新方法，不改变旧构造器和浏览器退出的可选钩子语义。

## 执行顺序和部分完成

返回值为 `(*AccountLogoutResult, error)`，失败时仍返回进度对象：

| 阶段 | 行为 | 成功标记 |
| --- | --- | --- |
| `validation` | 检查初始化、非空目标、请求、中心钩子及取消状态 | 无 |
| `center_logout` | 调用应用持有的 `CenterLogout` | `CenterRevoked` |
| `credentials` | 调用 `RevokeLoginCredentials` 替换账号凭证版本 | `CredentialsRevoked` |
| `notifications` | 启用 SLO 时发送注销通知 | `Notifications` 记录逐目标结果 |
| `cleanup` | 通知成功后清理账号回调绑定 | `BindingsCleared` |
| `complete` | 所有启用阶段成功 | `Complete` |

中心会话和旧凭证在通知开始前撤销，因此子应用不可达不会保留此前可撤销的中心状态。通知失败始终返回错误并保留全部回调绑定，即使配置了 `LogoutCallbackBestEffort`。已完成阶段不回滚；已收到通知的子应用可能已下线，不可达子应用的本地会话可能仍可用。

`EnableSLO=false` 时仍撤销中心会话与旧凭证，但不通知、不清理绑定；`Notifications=nil`、`BindingsCleared=false`，其余阶段成功时 `Complete=true`。因此 Complete 表示配置启用的流程完成，不承诺所有子应用已下线。未注册客户端或无回调的记录仍遵循现有通知模块的 `skipped` 语义。

阶段标记表示相应调用已经返回成功，不独立验证应用钩子的实际效果。存储或钩子返回错误时也可能已有部分副作用，应结合应用审计确认。方法在每个阶段前后检查取消；即使自定义钩子或存储忽略 Context 并返回成功，取消后的流程也不会继续报告整体成功，已确认的阶段标记仍保留。

`Cause` 保留与返回值相同的错误，支持 `errors.Is`；总体 `LoginID`、总体和逐目标 `Cause` 不参与 JSON 序列化。结果仍包含客户端 ID，应只提供给已授权管理端。方法不写 HTTP 响应、不清理管理员 Cookie，也不触发浏览器专用的 `OnLogoutResult`；调用方直接用返回结果记录审计和指标。

## 重试和账号禁用

通知复用当前登记、白名单、签名密钥、防重放 nonce、并发上限、超时、确认响应和请求内重试规则。一次调用失败后，用新的 Context 再次调用 `LogoutAccount`，会重新执行中心撤销、凭证版本替换及当前全部绑定的通知，包含此前成功目标。中心钩子和子应用注销处理器必须幂等。恢复不依赖已被撤销的中心浏览器会话，也不需要重放旧签名请求。

接口不保存持久化任务或成功目标进度，不自动后台重试。重试会影响期间新建立的同账号会话；永久封禁需在应用身份系统中先禁用登录，并通过 [AuthorizationPolicy](AUTHORIZATION_POLICY_zh.md) 阻止签发／兑换／检查，再完成下线和重试。恢复账号前先完成或取消待处理的管理任务，避免旧操作影响恢复后的会话。单纯强制退出后，允许重新认证的账号可以再次登录。

当前凭证版本机制覆盖本版本签发的 Ticket、授权码、共享 Token 和远程会话；版本字段为空的旧存储记录保留原兼容语义，不会因此被强制撤销。升级验收需另行处理旧记录有效期。直接可信核心签发 API 不执行 HTTP 授权策略，应用也须在这些入口检查账号状态。

多实例共享存储及相同前缀时，共享账号撤销版本和绑定索引。流程采用普通顺序存储操作，无事务回滚或跨实例锁。需要与新登录严格互斥时，由业务账号状态和接入入口协调。注销不撤回已经开始的业务请求。

## 验证状态

新增回归源码覆盖四种凭证、普通／原子存储、跨实例可见性、账号隔离、通知失败后的撤销效果、失败绑定保留和恢复、阶段存储故障、所有阶段取消、缺失配置、禁用 SLO、结果脱敏，以及部署示例实际 Auth 会话和签名／防重放回调的多应用闭环。

本轮只做格式化与静态检查，未执行测试、构建或服务。可在仓库根目录手动执行：

```powershell
go test ./sso/... -run 'TestAccountLogout|ExampleHTTPServer_LogoutAccount' -count=1
go test ./examples/sso_deployment/... -run TestDeploymentAccountLogout -count=1
go test -race ./sso/... ./examples/sso_deployment/... -run 'TestAccountLogout|TestDeploymentAccountLogout' -count=1
```

完整回归和 Redis／浏览器部署验证仍按[发布验收清单](RELEASE_CHECKLIST_zh.md)执行。
