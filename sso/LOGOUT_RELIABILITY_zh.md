# 单点注销结果、并发与重试

SSO 注销按照“通知客户端 → 撤销应用管理的中心会话 → 撤销账号 SSO 凭证 → 清理客户端会话索引 → 清除中心 Cookie”的顺序执行。通知模块提供逐客户端结果、有限重试、实例级并发限制及结果观察钩子。它不负责后台投递或持久化重试队列。

## 配置

每次通知尝试在读取登记、传输返回及确认响应体读取后复查截止时间。即使自定义传输或读取器超时后仍返回成功，也按 `timeout` 失败分类并受最大尝试次数约束；父请求取消则记为 `canceled`，不继续重试。响应体仍会关闭。子应用注销钩子取消后不会返回成功确认，已消费 nonce 不恢复。

浏览器中心注销在中心会话撤销、账号凭证撤销及绑定清理返回后检查取消，结果保留发生取消的阶段，不清理中心 Cookie 或返回成功。已经完成的业务效果不回滚；恢复可使用受授权的 [LogoutAccount](ACCOUNT_LOGOUT_zh.md) 流程。

在原有 `HTTPOptions` 上配置：

```go
options.EnableSLO = true
options.LogoutCallbackBestEffort = false
options.LogoutCallbackConcurrency = 4
options.LogoutCallbackMaxAttempts = 3
options.LogoutCallbackRetryInterval = 200 * time.Millisecond
options.LogoutCallbackTimeout = 3 * time.Second
options.OnLogoutResult = func(r *http.Request, result sso.LogoutResult) {
    if result.Cause != nil {
        logger.Printf("logout stopped at stage=%s", result.Stage)
    }
    if result.Notifications != nil {
        for _, target := range result.Notifications.Clients {
            if target.Status == sso.LogoutCallbackFailed || target.Status == sso.LogoutCallbackCanceled {
                logger.Printf("client=%q attempts=%d reason=%s", target.ClientID, target.Attempts, target.Reason)
            }
        }
    }
}
handler := sso.NewHTTPServer(server, options)
```

| 配置 | 默认值 | 含义 |
| --- | --- | --- |
| `LogoutCallbackConcurrency` | 8 | 同一 `HTTPServer` 所有并发注销请求的通知尝试总上限 |
| `LogoutCallbackMaxAttempts` | 1 | 单次注销请求、每个客户端的最大尝试次数，包含首次 |
| `LogoutCallbackRetryInterval` | 100 ms | 暂时性失败后再次尝试前的固定等待间隔 |
| `LogoutCallbackTimeout` | 3 s | 单次尝试的超时，包含登记查询、HTTP 请求与确认响应读取 |

这些值非正时使用默认值。默认只发送一次，因此升级不会自动增加重试次数。排队等待不占用单次尝试超时；整个注销请求的总期限由宿主设置请求 Context deadline。若复用自定义 `LogoutHTTPClient`，其自身超时仍生效。重试等待不占用共享并发槽；总时长可能包含排队、多次超时及重试间隔。每个目标只由一个 worker 顺序处理，同轮已成功的目标不会再次发送。

并发限制仅覆盖一个 `HTTPServer` 实例，应用应复用处理器；多副本部署的整体并发需结合副本数规划。它不替代入口限流。自定义存储和 HTTP Transport 应遵守 Context 取消。

## 重试规则

| 情况 | 本轮是否重试 |
| --- | --- |
| HTTP 请求或确认响应读取失败、单次尝试超时 | 是，次数以内 |
| 客户端登记查询失败 | 是，次数以内 |
| HTTP 408、429、500–599 | 是，次数以内 |
| 其他非 2xx 状态 | 否 |
| 回调地址不再允许、缺少签名密钥、无效配置 | 否 |
| 2xx 响应内容过大、JSON 无效、缺少整数 `code`、`code != 0` | 否 |
| 父请求取消或截止时间到达 | 停止等待及新尝试 |

确认响应继续兼容 2xx 空响应、`ok` 文本及 `{"code":0}` JSON，上限为 64 KiB。HTTP 200 携带业务错误不会被推断为暂时性故障。重试采用配置的固定间隔，不解析 `Retry-After`。

每次尝试重新读取客户端登记、检查回调允许地址、选择当前签名密钥，并重新生成签名参数。启用防重放时，每次尝试都有新 nonce，不能直接重发旧签名请求。若客户端已经注销但响应丢失，后续尝试仍可能再次到达，因此客户端注销处理必须允许幂等执行。

`ClientApp.LogoutCallbackHandler` 返回固定公开错误文本：校验失败保留既有 HTTP 状态分类，正文使用该状态的标准短语；业务钩子失败返回 HTTP 500 和 `logout callback failed`。响应不回显存储诊断或钩子错误。`VerifyLogoutCallback` 仍返回供本地处理的诊断错误，支持通过 `errors.Is` 匹配存储原因；成功校验会消费 nonce，因此不要先调用它，再对同一个回调调用内置处理器。业务钩子的诊断应在钩子内记录后返回错误。钩子失败不会恢复已消费的 nonce，重试必须使用新签名通知。只有钩子成功且请求未取消时才确认成功。

客户端登记被删除时标记 `skipped / unregistered`；没有配置回调地址的历史绑定标记 `skipped / no_callback`。跳过不表示该客户端的本地会话已确认撤销。

## HTTP 结果与观察钩子

原有成功响应 `code:0` 和 `data.result:"ok"` 保留，`data` 扩展为 `LogoutResult`。已通过签名、登录主体校验的注销失败响应也会携带阶段和已获得的通知结果：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "result": "ok",
    "stage": "complete",
    "notifications": {
      "complete": false,
      "clients": [
        {"clientId": "app-a", "status": "delivered", "attempts": 1, "httpStatus": 200},
        {"clientId": "app-b", "status": "failed", "attempts": 3, "httpStatus": 503, "reason": "http_status"}
      ]
    }
  }
}
```

这个例子表示尽力模式完成了中心注销，但 `app-b` 未确认注销。读取结果的应用应使用结构体或可容纳嵌套数据的对象；将整个 `data` 解码为 `map[string]string` 的调用方需要调整。

- `stage` 为 `notifications`、`center_logout`、`credentials`、`cleanup` 或 `complete`，表示停止或完成的阶段。进入后续阶段意味着之前阶段已经返回，但不是跨存储事务完成的保证。
- 每个目标的 `status` 为 `delivered`、`failed`、`skipped` 或 `canceled`，按客户端 ID 排序。
- `attempts` 包含登记／配置检查的尝试，不含排队；`httpStatus` 记录最后一次尝试收到的 HTTP 状态，无响应时省略。
- `notifications.complete` 表示没有失败／取消目标，并允许主动跳过，不能据此推断所有应用本地会话均已失效。
- 未启用 SLO 时不返回 `notifications`。身份校验未通过时不返回账号通知明细，也不调用观察钩子。
- `OnLogoutResult` 在最终响应前同步调用一次，能读取仅供应用使用的 `LoginID`、总体 `Cause` 与逐目标 `Cause`。这些字段不序列化；HTTP 错误消息只描述阶段，不回显回调 URL、原始响应或传输错误。
- 观察钩子不应写 HTTP 响应、阻塞或修改共享配置。框架复制结果切片再交给钩子，避免观察方修改响应数据。尽力模式的总体 `Cause` 可以为 nil，但逐客户端仍保留故障原因。

## 严格模式、尽力模式与生命周期边界

严格模式下，某目标最终失败会停止后续中心注销，并保留客户端索引和中心 Cookie，便于重新发起。已经成功执行的客户端回调不会回滚。

尽力模式下，个别目标失败不会阻止中心注销，结果中仍保留这些失败；索引读取失败或父请求取消会停止后续流程。应用中心会话注销、账号凭证失效、索引清理的故障也不会被尽力模式忽略。

新的一次注销请求会重新读取当前客户端绑定并开始新的一轮通知，包括上一轮已成功目标。当前模块不跨请求保存成功进度。中心会话已经撤销后发生的后续存储故障，可能使原浏览器无法重新认证；通过结果钩子定位阶段，再由应用执行受授权的修复，不应假设刷新链接能恢复所有阶段。框架不回滚已经完成的注销步骤。

回调仍按账号注销，不提供“只注销某次登录”的版本化投递语义。新登录与注销并发时由应用选择协调策略；不应把旧通知结果当作可长期保存的重试任务，否则可能影响后来建立的会话。

## 可信应用直接调用

`HTTPServer.NotifyLogoutClients(r, loginID)` 返回 `(*LogoutNotificationReport, error)`，可供应用在自有流程中组合通知。它始终返回真实通知错误，不因 `LogoutCallbackBestEffort` 隐藏错误，并保留 `errors.Is` 对底层原因的匹配。该方法仅发送通知，不撤销中心会话、清理索引、清除 Cookie 或触发 `OnLogoutResult`。

此方法不执行 HTTP 登录主体认证，调用方必须授权 `loginID`，不能直接把用户提交的账号参数传进去。需要完整标准注销时仍使用现有 `HandleLogout` 路由。

管理员强制下线可使用 [LogoutAccount](ACCOUNT_LOGOUT_zh.md) 组合完整服务端流程：先撤销中心会话和旧凭证，再通知并清理绑定。通知失败返回部分进度并保留绑定供受授权重试；它不写浏览器响应，也不调用 `OnLogoutResult`。`HandleLogout` 的原有通知优先和严格／尽力行为保持不变。

标准库和 Gin 中心示例显式配置并发数 4、尝试次数 1，并用观察钩子记录安全摘要。需要请求内重试时将尝试次数调大。验证命令和范围见 [测试指南](./TESTING_zh.md)。
