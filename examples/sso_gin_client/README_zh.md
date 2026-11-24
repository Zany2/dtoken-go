# Gin SSO Client 示例

需要把签名、防重放、PKCE、多子应用和 Redis 副本串联时，参见[完整接入示例](../sso_deployment/README_zh.md)及[部署与运维](../../sso/DEPLOYMENT_zh.md)。

示例在监听前执行 SSO 配置校验；中心使用 `RegisterClientChecked` / `NewHTTPServerChecked`，客户端检查 `ClientApp.Validate()`。非法配置会作为启动错误返回，完整规则见[配置与启动校验](../../sso/CONFIGURATION_zh.md)。

这个示例演示使用 Gin 接入统一登录中心的业务系统。

- `/protected`：受保护资源，未登录时跳转 SSO Server。
- `/sso/callback`：先校验浏览器状态，再调用 SSO Server `/sso/token` 将 Ticket 换成 `loginId`。
- `/sso/logout-callback`：接收统一登出回调，并清理本地会话。
- `/logout`：只清理当前子系统本地登录态。

示例客户端仅监听 `localhost:9101`。本地会话保存在进程内存，HttpOnly、SameSite=Lax Cookie 中只保存随机 `sessionId`。本地有效期上限为两小时，换票返回的正数 `remainTokenTimeout`、`remainSessionTimeout` 会按最小值进一步缩短期限。这两个字段表示中心会话剩余秒数，不是 Ticket 有效期。零表示未指定，-1 表示永不过期，均不延长本地上限；小于 -1 的值会被拒绝。截止时间从换票前开始计算，保守计入网络耗时；期限已耗尽时不能创建会话。Cookie 的 `Expires` 使用同一截止时间，服务端独立检查期限，不依赖 Cookie 的秒级精度。重放复制的 Cookie 不能延长服务端截止时间；访问过期会话及创建新会话时会回收过期记录。重启客户端会清空全部本地会话。

配套示例中心通过 `EnrichExchangeResult` 返回原 Auth Token 的剩余有效期，用于限制本地会话；未返回有效期字段的其他中心仍使用本地上限。有效期校验失败会保留原有本地会话。截止时间是登录时的快照，后续中心注销仍依赖注销回调立即删除本地会话。

未登录时访问 `/protected` 会生成随机浏览器状态，写入带签名、有效期五分钟的 Cookie，并由认证中心通过 `back` 参数回传。回调必须先校验二者，再交换 Ticket。每个浏览器只保留一个待完成流程，新发起的授权会覆盖旧状态 Cookie。通过校验后，即使换票失败也会清除浏览器状态 Cookie；重试时请重新访问 `/protected`，不要刷新回调地址。签名密钥按进程随机生成，重启也会使待完成流程失效。换票成功才替换浏览器原有本地会话，失败时保留原会话。

认证相关响应使用 `Cache-Control: no-store`，回调响应另外设置 `Referrer-Policy: no-referrer`。Gin 访问日志跳过 `/sso/callback`，避免该日志记录 Ticket 和浏览器状态。

这对示例使用公开演示凭证和 `CheckSign: false`，仅用于本地演示。`/sso/logout-callback` 由 `ClientApp.LogoutCallbackHandler` 校验客户端 ID 及五分钟内的时间戳，但这两项检查本身不能认证发送方。部署时需要配置私有凭证、两端请求签名、HTTPS 和 Secure Cookie；本示例不构成生产认证方案。

## 启动

先启动 Gin SSO Server：

```powershell
go run ./examples/sso_gin_server
```

再启动 Gin SSO Client：

```powershell
go run ./examples/sso_gin_client
```

访问：

```text
http://localhost:9101/protected
```

浏览器会跳转到 SSO Server 登录页，登录后带 Ticket 回到 Client，并创建子系统本地登录态。

## 验证统一登出

登录成功后访问：

```text
http://localhost:9100/sso/logout
```

SSO Server 会推送 `/sso/logout-callback`，Client 收到回调后会删除当前登录主体的全部本地会话，不影响其他用户。

`/logout` 只清理当前浏览器的本地会话，认证中心仍保持登录，因此再次访问 `/protected` 时可能立即重新登录。Gin 认证中心启用了尽力执行的单点注销，回调超时为三秒：即使 Client 回调失败，中心仍会清理自身登录态。这种情况下，Client 的既有会话仍可使用，直到本地注销或到达计算出的截止时间；中心注销成功不代表所有 Client 会话均已删除。
