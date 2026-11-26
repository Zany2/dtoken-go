# SSO 第一阶段：公共兑换核心与会话契约

本阶段完成公共兑换服务、部署装配校验和会话后端契约。默认 Auth 适配、中心 simple 授权、三种模式的一体化浏览器装配及浏览器范围统一退出仍按[架构方案](ARCHITECTURE_zh.md)后续阶段实现；新增配置校验通过不代表这些流程已经可用。

## 已可使用的公共兑换服务

`ExchangeService` 借用已有 `Server`，复用其客户端登记和协议存储，不注册客户端、不监听端口，也不关闭外部资源。它执行签名、防重放、客户端凭据、回调、PKCE、授权策略、一次性消费、结果补充和退出回调登记。

HTTP `/sso/token` 与本地兑换复用这套核心规则。HTTP 方法、查询串、请求体解析和响应编码仍留在 HTTP 层。本地入口接受 `url.Values` 形式的规范化协议参数，以保留自定义 `ParamNames`、签名和扩展字段；不会接受免校验标志或客户端自报的可信身份。一般应用通过已有的 `ClientApp.ExchangeTicket` 或 `ExchangeCredential` 调用，不需要自行构造签名参数。

以下为应用侧装配片段。`sharedStore` 是与中心共享协议数据的 `contract.Storage`；前缀、AuthType、密钥、防重放配置和授权策略均须与中心一致。

```go
peer := sso.NewServer(
    sso.WithStorage(sharedStore),
    sso.WithKeyPrefix("company:sso:"),
    sso.WithAuthType("sso"),
)

exchangeOptions := sso.DefaultExchangeOptions()
exchangeOptions.SecretKey = signingKey
exchangeOptions.ReplayProtection = replayProtection
exchangeOptions.AuthorizationPolicy = authorizationPolicy
// EnrichExchangeResult 使用 Context，不依赖 HTTP 请求。
exchangeOptions.EnrichExchangeResult = enrichSessionMetadata
exchange, err := sso.NewExchangeService(peer, exchangeOptions)
if err != nil {
    return err
}

clientConfig := sso.DefaultClientConfig()
clientConfig.ClientID = clientID
clientConfig.ClientSecret = clientSecret
clientConfig.SecretKey = signingKey
clientConfig.ServerURL = centerURL
clientConfig.ReplayProtection = replayProtection
clientConfig.ExchangeService = exchange
// 统一退出仍需登记回调并挂载实际撤销本地会话的通知处理器。
clientConfig.RegisterCallback = true
clientConfig.LogoutCallbackURL = logoutCallbackURL
clientApp, err := sso.NewClientAppChecked(clientConfig)
if err != nil {
    return err
}
// 将 clientApp 传入现有 BrowserClient，仍使用原有业务登录和退出钩子。
```

中心的客户端登记由中心管理，本地兑换构造器只读取现有记录。`ServerURL` 仍用于浏览器授权和退出；`Introspect`、`UserInfo`、`Revoke` 仍通过 HTTP。共享协议存储本身不等于共享 Auth 账号会话，标准模式二还需要下一阶段的会话接入。

应用结束时按既有所有权规则释放 `peer` 及存储；`ExchangeService` 没有需要单独关闭的资源。构造后不要并发修改配置所引用的依赖。

## 兼容与冲突规则

| 配置 | 行为 |
| --- | --- |
| 没有本地服务 | 保持原有 HTTP 兑换 |
| `SharedStoreServer` 且 `UseHTTPCheck=false` | 旧本地兼容入口，保留合成 HTTP 请求及旧钩子的请求信息 |
| `SharedStoreServer` 且 `UseHTTPCheck=true` | 保持原有强制 HTTP 语义 |
| `ExchangeService` | 直接调用公共核心，不创建 HTTP 请求 |
| `ExchangeService` 同时配置旧本地入口或 `UseHTTPCheck=true` | 配置错误，不静默切换路径 |

新旧本地入口均继续支持 Ticket 和 OAuth2 Code；它们是凭证类型。默认 HTTP 行为及旧钩子签名不变。

新 `HTTPOptions.ExchangeEnricher` 与 `ExchangeOptions.EnrichExchangeResult` 都接受 `ExchangeEnricher`：`func(context.Context, string, *TicketExchangeResult) error`。旧 `HTTPOptions.EnrichExchangeResult` 继续收到原请求及其表单；同一个 HTTPServer 不能同时配置新旧两个补充钩子。

补充钩子执行在凭证消费之后、回调登记之前，不得改变身份或扩大授权范围。失败不恢复凭证。签名和 nonce 的原有消费次序不变：已认证的请求即使随后证明错误，nonce 也可能已消费；重试必须重新签名，不能重发旧 nonce。

## 为统一登录准备的配置与契约

`DeploymentConfig.Validate()` 是显式的静态装配校验，不会自动改变现有 `BrowserClient` 行为，也不会发起网络或调用会话后端。

| 部署值 | 当前检查 |
| --- | --- |
| `DeploymentSharedCookie` | 要求会话后端，禁止本地兑换或强制 HTTP 兑换配置 |
| `DeploymentSharedStore` | 要求会话后端和本地兑换服务，禁止强制 HTTP |
| `DeploymentHTTP` | 要求会话后端，禁止配置本地兑换服务 |

标准部署要求底层配置使用 Ticket，模式一后续的 simple 分支不会实际兑换 Ticket。未指定部署值会报配置错误；旧 `ClientConfig` 不需要增加任何部署选择。

静态校验无法证明不同进程是否共享同一 Redis 数据范围、Auth 命名空间或 Cookie，也无法判断远端中心是否支持新流程。当前不把 `DeploymentConfig` 转换成登录处理器，避免误把模式一运行成普通 Ticket 流程。

`SessionBackend` 只负责 `Resolve`、`Create`、`Revoke`，与 Cookie 和业务用户名密码验证分离。无有效会话时 `Resolve` 返回 `nil, nil`，存储故障必须返回错误。`SessionInfo` 的 Token 和 `SessionRevoke` 的 Token 不参与 JSON 序列化。

`SessionGrant` 提供中心主体、登录生命周期、应用和期限契约。它本身不是认证凭据，也尚未加入 Ticket 或存储载荷。本地后端只撤销单个 Token 或本地账号会话；跨应用全局退出由后续 SSO 协调层解析关联后调用，不能把本地账号注销误称为全局注销。

## 验证范围

新增用例覆盖直接兑换的 Ticket／Code、普通／原子存储、签名与证明拒绝、策略拒绝、上下文传递、回调登记、一次性消费、防重放、请求大小、取消、HTTP 与本地消费互通、钩子失败、配置冲突和私有 Token 序列化。原有旧钩子及浏览器回归用例保留。

本轮只进行了代码静态审查、格式化和文档检查，未执行构建、测试、竞态检查、服务或真实 Redis 验证。运行验证由用户进行。
