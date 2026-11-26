# 三种 SSO 部署方式

[English](DEPLOYMENT_MODES.md)

新增本地兑换入口为 `ClientConfig.ExchangeService`，可直接调用公共兑换核心。下文 `SharedStoreServer` 装配仍为兼容入口。新入口、部署校验和后续会话接入边界见[第一阶段说明](EXCHANGE_CORE_zh.md)。

统一登录的主线始终是：中心认证用户，应用认可可信登录结果，其他应用复用中心登录状态，退出时撤销关联会话。部署方式决定身份传递路径；`ModeTicket`、`ModeOAuth2`、`ModeSharedToken`、`ModeRemoteSession` 是凭证类型，不是以下部署分类。

| 部署方式 | 身份传递 | 本地会话与退出 | 当前接入 |
| --- | --- | --- | --- |
| 同域、共享认证 Redis | 共享 Cookie 内的 Token，读取同一份 Auth 会话 | 同一会话撤销后，各应用再次检查时失效 | 用根包 Auth 和 Cookie 工具显式组装，尚无独立的一站式模式一组件 |
| 跨域、共享认证 Redis | 中心跳转投递 Ticket，应用直接在共享存储兑换 | 应用建立独立本地会话，仍需退出通知 | `ClientConfig.SharedStoreServer` + `BrowserClient` |
| 跨域、独立 Redis | 中心跳转投递 Ticket，应用通过 HTTP 兑换 | 应用建立独立本地会话，退出通知负责撤销 | 默认 `ClientApp` + `BrowserClient`；见[部署示例](../examples/sso_deployment/README_zh.md) |

“同域”指浏览器实际可共享 Cookie 的域与路径范围。业务 Redis 可以不同，关键是认证状态是否共享。同域但认证 Redis 不共享时，也可以使用 HTTP 兑换。共享 Redis 地址不等于共享会话：数据库、存储前缀、AuthType、编解码配置等也必须匹配。

## 模式二：共享存储兑换

本模式没有增加第二个登录中心。中心仍提供登录页面、身份认证和 `/sso/authorize`；应用只在自己的进程中组装一个连接中心存储的协议服务，负责兑换。`HTTPServer` 同时承载已有协议规则，此处无需调用 `Register`、无需监听端口。

在应用进程中，`sharedStore` 是已初始化的共享 `contract.Storage`，`centerExchangeOptions` 是与中心一致的兑换规则，`clientConfig` 是已有应用配置：

```go
peer := sso.NewServer(
    sso.WithStorage(sharedStore),
    sso.WithKeyPrefix("company:sso:"), // 与中心一致
    sso.WithAuthType("sso"),           // 与中心一致
)

// 客户端注册由中心管理；应用读取共享记录，不覆盖注册。
centerExchangeOptions.Clients = nil
centerExchangeOptions.AllowAnonymousClient = false
localExchange, err := sso.NewHTTPServerChecked(peer, centerExchangeOptions)
if err != nil {
    _ = peer.Close()
    return err
}
clientConfig.SharedStoreServer = localExchange
clientConfig.UseHTTPCheck = false
clientApp, err := sso.NewClientAppChecked(clientConfig)
if err != nil {
    _ = peer.Close()
    return err
}
// 将 clientApp 传给现有 NewBrowserClient；登录与退出路由保持同一套接法。
// 应用停止并排空请求后关闭 peer；借用的 sharedStore 由其所有者关闭。
```

存储前缀和 AuthType 的值应取自中心实际配置，代码中的值仅为示意。完整公开 API 示例见 [example_shared_store_test.go](example_shared_store_test.go)，浏览器路由见[浏览器接入](BROWSER_CLIENT_zh.md)。

路由选择是显式的：

| SharedStoreServer | UseHTTPCheck | Ticket／Code 兑换路径 |
| --- | --- | --- |
| nil | false 或 true | HTTP，保持历史配置行为 |
| 非 nil | false | 本地共享存储 |
| 非 nil | true | 强制 HTTP，本地服务不参与 |

共享存储兑换失败会返回错误，不自动改走 HTTP；HTTP 失败也不会自动访问共享存储。`ServerURL` 仍用于浏览器授权及退出跳转。`Introspect`、`UserInfo`、`Revoke` 继续走 HTTP，本地兑换仅支持 Ticket 和授权码。活跃的共享存储配置配合 Shared Token 或 Remote Session 模式会在校验构造时拒绝。

## 同一套登录和退出生命周期

1. `BrowserClient.BeginLogin` 保存 state 与 PKCE 证明，浏览器跳转到中心。
2. 中心解析自己的登录会话；未登录则进入业务登录页，已登录则复用该会话签发 Ticket。
3. 浏览器回到应用，`CompleteLogin` 校验并消费 state。
4. `ClientApp.ExchangeCredential` 在本地执行与 `/sso/token` 相同的请求解析、签名、防重放、客户端认证、PKCE、授权策略、一次性消费、结果补充及退出登记流程。
5. 应用登录钩子建立本地会话。第二个应用重复上述流程，复用同一中心登录状态。
6. 全局退出仍通过中心执行：通知已登记应用撤销本地会话，撤销中心会话及旧 SSO 凭证，并清理退出绑定。

共享协议存储不会自动撤销应用独立的 Auth 会话。需要 `EnableSLO`、`RegisterCallback`、有效的 `LogoutCallbackURL` 和实际撤销本地会话的回调钩子；`CenterLogout` 负责业务持有的中心会话。只删除 Cookie 不等于撤销存储中的会话。直接存储兑换也会登记回调，因此中心可以读取并通知这些应用。

## 配置与运行边界

- 中心与本地兑换服务的参数名、签名密钥信任规则、防重放存储命名空间、客户端容量限制、授权策略、结果补充逻辑应保持一致。参数名不一致或本地服务未初始化会在客户端配置校验时拒绝；框架不能自动复制另一进程的 Go 函数或验证远端配置一致性。
- 共享认证存储意味着应用处于同一信任范围。应用不应在启动时重写中心注册记录；需要严格隔离应用存储权限时使用 HTTP 模式。
- 本地策略及 `EnrichExchangeResult` 在应用进程执行。补充钩子接收一个合成 POST 请求，包含表单、中心目标 URL 和调用方 Context，不包含原始浏览器 Cookie、网络来源或 HTTP 中间件注入的数据。依赖中心专有资源的钩子须改为访问共同资源，或保持 HTTP 兑换。
- `RequestLimits`、签名和 nonce 校验仍适用；`HTTPClient.Timeout` 与 `MaxResponseBytes` 不作用于本地调用。通过 Context 设置本地调用期限；存储和钩子须遵守 Context。浏览器路径保留请求 Context。
- 本地调用返回原始 Go 错误，供内部诊断；不经过 JSON 编码或 `OnHTTPError`。`BrowserClient` 的错误响应仍使用既有固定消息，不应自行把原始错误写到页面。
- 策略、证明或回调归属检查失败不会消费凭证；nonce 可能已被占用。凭证消费后补充钩子失败不会恢复凭证，浏览器 state 也不会恢复，应重新发起登录。
- 原子存储可跨实例一次性消费；普通存储保持既有 Get/Delete 回退，锁只覆盖当前实例。跨进程并发消费优先使用提供原子能力的 Redis 适配器。

## 验证状态

新增共享存储回归及浏览器闭环测试源码覆盖独立签发／消费实例、Ticket 与 Code、普通／原子存储、PKCE、授权缩减、请求拒绝、nonce、容量、取消、失败后生命周期、两个域名复用中心登录以及统一退出。测试使用内存和受控传输，不监听端口或连接真实 Redis。

本轮只做静态检查，未运行构建、测试、示例或服务。用户可运行：

```powershell
go test ./sso/... -run 'TestSharedStore|ExampleClientApp_sharedStore' -count=1
go test ./sso/... -count=1
```

共享内存用例不能证明真实 Redis、跨进程部署或网络退出通知已验收；这些仍需在实际环境验证。
