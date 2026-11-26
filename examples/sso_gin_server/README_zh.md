# Gin SSO Server 示例

需要把签名、防重放、PKCE、多子应用和 Redis 副本串联时，参见[完整接入示例](../sso_deployment/README_zh.md)及[部署与运维](../../sso/DEPLOYMENT_zh.md)。

示例在监听前执行 SSO 配置校验；中心使用 `RegisterClientChecked` / `NewHTTPServerChecked`，客户端检查 `ClientApp.Validate()`。非法配置会作为启动错误返回，完整规则见[配置与启动校验](../../sso/CONFIGURATION_zh.md)。

这个示例演示使用 Gin 部署一个统一登录中心，并挂载 `sso.HTTPServer` 的标准协议路由。

- `/login`：模拟统一登录页。
- `/sso/authorize`：校验中心登录态，生成 Ticket，并重定向回子系统。
- `/sso/token`：子系统使用 Ticket 换取登录主体信息。
- `/sso/logout`：清除中心 Cookie，并向已登记子系统推送统一登出回调。

这是本地模拟登录中心：`/login` 接受任意登录 ID，不校验密码，ID 为空时默认使用 `user-1001`。仅监听 `localhost:9100`。登录提交检查浏览器来源，并在签发 Cookie 前显式拒绝损坏的 URL 编码表单或 multipart 表单。协议 POST 仍允许子系统调用。为与 `examples/sso_gin_client` 配套，请求签名未开启；Ticket 交换仍校验客户端密钥和回调地址。

中心通过 `dtoken.NewBuilder().Build()` 构建实例，登录、身份解析与账号注销共用该实例。签名 Cookie 保存不透明的 Auth Token，解析时检查服务端状态。不同浏览器使用独立 Token，有效期固定为两小时，关闭自动续期。新登录会替换当前浏览器原有中心会话。中心注销成功后，账号的全部中心会话失效，复制的 Cookie 不能重新申请 Ticket，其他账号不受影响。Auth 与 SSO 分别使用内存存储，由 `run` 持有并关闭；重启会清空状态并更换 Cookie 签名密钥。

`AuthorizationExtra` 将 Ticket 绑定到已认证 Token 的哈希引用。`EnrichExchangeResult` 通过账号的 Token 索引解析引用，检查原会话，并返回 `deviceId` 与 `remainTokenTimeout`，不会泄露中心 Token，也不信任换票时替换的浏览器 Cookie。会话缺失、过期、撤销或凭证未绑定时换票失败；扩展失败不会恢复已消费的 Ticket。配套客户端使用返回的期限限制本地会话；示例不虚构账号 Session 有效期，已建立的本地会话仍需注销回调才能立即删除。

此 Gin 示例设置了 `LogoutCallbackBestEffort: true`：尝试通知已登记子系统后，即使子系统离线，也会撤销账号的 Auth 会话及 SSO 凭证，再清理 Cookie 和回调记录。不可用的子系统可能仍保留本地会话，因此中心返回成功不代表所有子系统均已注销。如需回调失败时使中心注销失败并保留状态重试，应设为 `false`。在此路由之外执行管理端注销时，也应撤销 SSO 凭证并按需通知子系统。

中心还配置了 `CheckCredentialActive`，为 introspect 与 userinfo 复用换票时的 Auth 会话查询，不触发续期。中心会话失效后，即使 SSO 凭证尚未过期，introspect 也返回 inactive，userinfo 返回 HTTP 401；查询故障仍返回错误。这不会删除子应用已经创建的本地会话。

## 启动

本示例将注销通知并发限制为 4，默认每个目标尝试 1 次，可通过 `LogoutCallbackMaxAttempts` 和 `LogoutCallbackRetryInterval` 启用请求内重试。尽力模式完成中心注销时，`data.notifications` 和 `observeLogoutResult` 日志仍会报告未确认的客户端；`code=0` 不代表所有子应用均已退出。详见 [单点注销可靠性](../../sso/LOGOUT_RELIABILITY_zh.md)。

```powershell
go run ./examples/sso_gin_server
```

在仓库根目录执行时，按 `go.work` 要求使用 Go 1.27 或更高版本。该示例模块独立使用时声明的最低版本为 Go 1.25。部署时还需接入真实认证、配置受管理的密钥并启用两端请求签名，以及使用 HTTPS 与 Secure Cookie。

默认监听：

```text
http://localhost:9100
```

同时启动 `examples/sso_gin_client` 后访问：

```text
http://localhost:9101/protected
```

## Redis 存储

示例默认使用内存存储，便于本地直接启动。生产环境可以把 `sso.NewServer()` 替换为 Redis 构造器：

```go
import ssoredis "github.com/Zany2/dtoken-go/sso/storage/redis"

server, err := ssoredis.NewServer(
	"redis://:password@127.0.0.1:6379/0",
	sso.WithConfig(sso.DefaultConfig()),
)
if err != nil {
	return err
}
defer server.Close()
```

在 `run` 中使用上述创建代码，让错误经由返回路径执行延迟清理。该替换只共享 SSO 协议存储；多实例中心还需配置共享 Auth 存储及统一管理的 Cookie 签名密钥，模拟登录逻辑仍需实际接入。Redis 适配器通过原子操作消费 Ticket，要求 Redis 6.2 或更高版本。
