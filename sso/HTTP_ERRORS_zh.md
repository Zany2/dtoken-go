# 中心 HTTP 错误与诊断

[English](HTTP_ERRORS.md) · [配置](CONFIGURATION_zh.md) · [测试](TESTING_zh.md)

中心六个端点（`authorize`、`token`、`introspect`、`userinfo`、`revoke`、`logout`）统一处理错误。已知协议错误使用固定分类消息，未知错误使用 `SSO operation failed`。存储、序列化及业务钩子的包装诊断不再复制到公开响应。JSON 保留 `code`、`message` 和可选 `data`，错误响应保留 `no-store`、`no-cache` 和 `nosniff`。

## 状态与兼容性

| 失败类型 | HTTP 状态 |
| --- | --- |
| 参数、绑定／证明或一次性凭证无效 | 400 |
| 签名／客户端凭据无效、请求过期、nonce 已使用 | 401 |
| 签发／兑换时策略明确拒绝 | 403 |
| 客户端登记不存在 | 404 |
| 方法不支持 | 405 |
| 请求超过配置上限 | 413 |
| 策略求值不可用 | 503 |
| 初始化／配置、存储、取消或未知钩子故障 | 500 |

初始化失败通过直接处理器、注册 mux 路由及 `Handler()` 均返回 500。配置／存储分类优先于内部包装的凭证类原因。策略基础设施失败即使包装拒绝原因也保持 503。端点既有覆盖保留，例如退出主体不匹配仍为 401。诊断文本及部分固定认证消息会变化，接入方应依据 HTTP 状态／业务 code 判断，不匹配原始消息。

普通无效凭证的 introspect 仍返回 HTTP 200 和 `active:false`，userinfo 返回 401。存储或活动检查钩子故障返回错误，不伪装成凭证无效。只返回 `(string, bool)` 的身份解析器无法携带错误，`false` 仍表示未认证；自有解析器的诊断仍由应用负责。

## 本地观察

通过 `HTTPOptions.OnHTTPError` 配置 `func(r *http.Request, status int, err error)`。`httpserver` 门面导出相同的 `HTTPErrorFunc`。回调在每次中心生成失败响应前同步调用一次，收到操作错误及已有错误链。回调不接收 ResponseWriter，不能替换响应。

```go
options.OnHTTPError = func(_ *http.Request, status int, err error) {
    category := "protocol"
    if errors.Is(err, sso.ErrStorageUnavailable) {
        category = "storage"
    }
    log.Printf("SSO rejection status=%d category=%s", status, category)
}
```

片段需要导入 `errors`、`net/http`、`log` 和 `sso`。完整公开 API 示例见 [example_http_error_test.go](example_http_error_test.go)。部署中心只记录 5xx 状态值，并保留退出阶段观察器。

请求和错误应只读。原始错误、URL、表单、请求头及 Cookie 可能包含秘密值，不得原样记录。使用固定分类及应用提供的可信关联信息。观察器应及时返回、支持并发请求并自行处理失败；框架不恢复观察器 panic，也不启动后台任务。已有错误链原样传递，不重建适配器或调用方已转成字符串的底层原因。

成功响应、登录跳转、无效凭证 introspect、mux 自行生成的 404、直接核心 API，以及客户端 BrowserClient／RemoteAuth／注销回调均不触发中心观察器。`LogoutAccount` 仍通过自己的结果／错误返回诊断。带校验的构造器失败时处理器尚未建立，应直接处理其返回错误以获取启动诊断。

已认证 HTTP 退出失败时，先调用 `OnLogoutResult`，再调用一次 `OnHTTPError`。公开阶段消息及报告不变。尽力模式 HTTP 200 携带失败通知时，只调用 `OnLogoutResult`；没有触发 `OnHTTPError` 不能证明通知全部成功。

## 生命周期与验证

错误报告不增加重试、回滚或状态恢复。成功占用 nonce 前拒绝不消费它；后续钩子失败不恢复已占用 nonce，重试需新签名请求。Ticket／Code 兑换前策略失败保留凭证，兑换后补充结果钩子失败保持已消费状态。删除失败可能使凭证仍有效。应根据操作结果决定恢复方式，不在观察器中执行补偿。

[http_errors_test.go](http_errors_test.go) 和 [http_errors_lifecycle_test.go](http_errors_lifecycle_test.go) 覆盖六个端点、错误分类、三种初始化调用入口、防重放存储恢复、四种凭证模式、兑换阶段及观察次数。既有退出阶段测试覆盖严格／尽力观察行为。源码已格式化并静态检查，**尚未执行**。准备好后从仓库根目录运行：

```powershell
go test ./sso/... -run 'TestHTTPError|TestLogoutNotificationHTTPStages|ExampleHTTPErrorFunc' -count=1
go test ./sso/... ./examples/sso_deployment/... -count=1
```

发布、竞态及 Redis 检查仍见[发布验收清单](RELEASE_CHECKLIST_zh.md)。
