# SSO Server 示例

需要把签名、防重放、PKCE、多子应用和 Redis 副本串联时，参见[完整接入示例](../sso_deployment/README_zh.md)及[部署与运维](../../sso/DEPLOYMENT_zh.md)。

示例在监听前执行 SSO 配置校验；中心使用 `RegisterClientChecked` / `NewHTTPServerChecked`，客户端检查 `ClientApp.Validate()`。非法配置会作为启动错误返回，完整规则见[配置与启动校验](../../sso/CONFIGURATION_zh.md)。

这个示例演示一个最小统一登录中心：

- `/login`：模拟 SSO 登录页。
- `/sso/authorize`：生成 Ticket 并重定向回子系统。
- `/sso/token`：让子系统使用 Ticket 换取登录主体信息。
- `/sso/logout`：清除 SSO 中心 Cookie，并推送子系统统一登出回调。

子系统在跳转登录时会携带 `callback` 参数，SSO Server 授权成功后会记录该回调地址。用户从登录中心退出时，Server 会通知已登记的子系统清理本地登录态。

这是本地流程演示：`/login` 接受任意登录 ID，不校验密码；ID 为空时默认使用 `user-1001`。服务仅监听 `localhost:9000`。登录提交只读取 POST 表单正文，并使用 Go 标准库的跨源请求保护。协议端点仍支持子系统到认证中心的请求；配套示例关闭了请求签名，但 Ticket 交换仍校验已登记的客户端密钥和回调地址。

认证中心通过 `dtoken.NewBuilder().Build()` 构建实例，登录、身份解析与账号注销共用该实例。签名 Cookie 保存不透明的 Auth Token，每次授权都检查其服务端状态。不同浏览器使用独立 Token，有效期固定为两小时，关闭自动续期。新登录会替换当前浏览器原有中心会话。中心注销成功后，账号的全部中心会话失效，复制的 Cookie 不能重新申请 Ticket，其他账号不受影响。Auth 与 SSO 分别使用内存存储，由 `run` 持有并关闭；重启会清空状态并更换 Cookie 签名密钥。

`AuthorizationExtra` 将 Ticket 绑定到已认证 Token 的哈希引用。`EnrichExchangeResult` 通过账号的 Token 索引解析引用，检查原会话，并返回 `deviceId` 与 `remainTokenTimeout`。它不会返回中心 Token，不接受换票时使用其他浏览器 Cookie 替代原会话，也不会虚构账号 Session 有效期。会话缺失、过期、撤销或凭证未绑定时不能完成换票；扩展失败后一次性 Ticket 仍已消费。配套客户端会使用返回的期限限制本地会话，已建立的本地会话仍需注销回调才能立即删除。

注销采用严格回调策略：已登记子系统离线或拒绝回调时，中心注销返回错误，保留 Auth 会话、Cookie 和回调记录以便重试；成功后先撤销账号的 Auth 会话及 SSO 凭证，再清理 Cookie 和回调记录。体验 `/sso/logout` 时请保持子系统运行。在此路由之外执行管理端注销时，也应撤销 SSO 凭证并按需通知子系统。

中心还配置了 `CheckCredentialActive`，为 introspect 与 userinfo 复用换票时的 Auth 会话查询，不触发续期。中心会话失效后，即使 SSO 凭证尚未过期，introspect 也返回 inactive，userinfo 返回 HTTP 401；查询故障仍返回错误。这不会删除子应用已经创建的本地会话。

## 启动

注销通知在本示例中最多并发 4 个，默认每个目标只尝试 1 次。`observeLogoutResult` 记录停止阶段和失败目标的安全摘要；需要单次请求内自动重试时，可调整 `LogoutCallbackMaxAttempts` 和 `LogoutCallbackRetryInterval`。HTTP 注销响应新增 `data.stage` 与 `data.notifications`，成功仍保留 `data.result="ok"`。详见 [单点注销可靠性](../../sso/LOGOUT_RELIABILITY_zh.md)。

```powershell
go run ./examples/sso_server
```

在仓库根目录执行时，按 `go.work` 要求使用 Go 1.27 或更高版本。该示例模块独立使用时声明的最低版本为 Go 1.25。用于部署时，应替换模拟登录、配置共享存储和受管理的密钥，并使用 HTTPS 与 Secure Cookie。

默认监听：

```text
http://localhost:9000
```

需要同时启动 `examples/sso_client`，然后访问：

```text
http://localhost:9001/protected
```
