# SSO Client 示例

需要把签名、防重放、PKCE、多子应用和 Redis 副本串联时，参见[完整接入示例](../sso_deployment/README_zh.md)及[部署与运维](../../sso/DEPLOYMENT_zh.md)。

示例在监听前执行 SSO 配置校验；中心使用 `RegisterClientChecked` / `NewHTTPServerChecked`，客户端检查 `ClientApp.Validate()`。非法配置会作为启动错误返回，完整规则见[配置与启动校验](../../sso/CONFIGURATION_zh.md)。

这个示例演示一个接入 SSO 的业务系统：

- `/protected`：受保护资源，未登录时跳转 SSO Server。
- `/sso/callback`：接收 Ticket，调用 SSO Server `/sso/token` 换取 `loginId`。
- `/sso/logout-callback`：接收 SSO Server 的统一登出回调，并清理本地会话。
- `/logout`：删除当前本地会话并清除对应 Cookie。

示例客户端会把本地登录态存到进程内存，并在 Cookie 中只保存本地 `sessionId`。这样 SSO Server 推送统一登出回调时，可以根据 `loginId` 删除该用户在子系统内的所有本地会话。`/sso/logout-callback` 使用 `ClientApp.LogoutCallbackHandler` 处理，实际项目开启签名后也会在这里完成验签。

本地会话在服务端最多两小时后过期，即使手动提交复制的 Cookie 也不能继续使用。换票结果中的正数 `remainTokenTimeout`、`remainSessionTimeout` 会按最小值进一步缩短期限。这两个字段表示中心会话的剩余秒数，不是 Ticket 有效期。零表示未指定，-1 表示永不过期，都不会延长本地上限；小于 -1 的值会被拒绝。截止时间从换票请求发出前计算，保守计入网络耗时；期限已耗尽时不能创建会话。Cookie 的 `Expires` 使用同一截止时间，服务端独立检查期限，不依赖 Cookie 的秒级精度。

访问时会删除当前过期会话，创建新会话时会扫描回收过期记录。新登录成功后替换当前浏览器的旧会话，交换或有效期校验失败则保留旧会话。重启客户端会清空所有本地会话。配套示例中心通过 `EnrichExchangeResult` 返回原 Auth Token 的剩余有效期，用于限制本地会话；未返回有效期字段的其他中心仍使用本地上限。期限同步是登录时的快照，后续中心注销仍依赖注销回调立即删除本地会话。

请从 `/protected` 发起登录。客户端通过 SSO 的 `back` 参数传递随机值，并将其绑定到有效期五分钟的签名 HttpOnly Cookie。回调必须匹配该 Cookie，才会交换 Ticket。每个浏览器只保留一个待完成登录流程，新流程会替换旧状态。接受回调后即清除状态 Cookie，包括交换失败的情况；登录超时或交换失败时，请从 `/protected` 重新发起。

配套示例仅监听 `localhost:9001`，使用模拟认证和公开的演示客户端凭证。为了与 `examples/sso_server` 配套，请求及注销回调签名均未开启；回调的客户端 ID 和时间戳检查本身不能认证发送方。用于部署时，应在两端配置受管理的密钥并启用签名，配置 HTTPS/Secure Cookie，并按需使用共享会话存储。

本地 `/logout` 不会注销认证中心。因此，再次访问 `/protected` 时可能通过仍有效的中心 Cookie 自动登录。需要联动注销时，请保持两个应用运行，并访问 `http://localhost:9000/sso/logout`。

## 启动

也可选择新的完整浏览器接入模式：`go run ./examples/sso_client -browser-flow`。此模式通过 `BrowserClient` 管理一次性状态和多标签页登录，复用应用本地会话；登录后页面提供 POST 本地退出及全局退出按钮。`/logout-all` 会先注销本地会话再跳转中心，最终结果由中心返回。上文“每浏览器一个待完成流程”和 GET 退出链接描述的是默认旧模式。两种模式均使用开发环境内存存储与未启用签名的配套配置，详见 [浏览器接入指南](../../sso/BROWSER_CLIENT_zh.md)。

先启动 SSO Server：

```powershell
go run ./examples/sso_server
```

再启动 SSO Client：

```powershell
go run ./examples/sso_client
```

访问：

```text
http://localhost:9001/protected
```

浏览器会自动跳转到 SSO Server 登录页，登录后带 Ticket 回到 Client，并创建子系统本地登录态。

在 SSO Server 执行 `/sso/logout` 时，Server 会调用本示例的 `/sso/logout-callback`，Client 收到回调后会删除对应用户的本地会话。

### 浏览器模式的 PKCE 绑定

`-browser-flow` 现已启用 PKCE S256，授权只发送挑战，verifier 保存到服务端流程状态并在换票 POST 中提交。请先升级配套中心；普通手动登录流程保持原有配置。若将中心该客户端注册改为 `RequirePKCE: true`，所有使用该客户端的 Ticket/Code 流程都必须携带证明，未迁移的手动流程将被拒绝。详见 [PKCE 指南](../../sso/PKCE_zh.md)。
