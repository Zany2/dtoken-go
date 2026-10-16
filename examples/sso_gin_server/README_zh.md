# Gin SSO Server 示例

这个示例演示使用 Gin 部署一个统一登录中心，并挂载 `sso.HTTPServer` 的标准协议路由。

- `/login`：模拟统一登录页。
- `/sso/authorize`：校验中心登录态，生成 Ticket，并重定向回子系统。
- `/sso/token`：子系统使用 Ticket 换取登录主体信息。
- `/sso/logout`：清除中心 Cookie，并向已登记子系统推送统一登出回调。

这是本地模拟登录中心：`/login` 接受任意登录 ID，不校验密码，ID 为空时默认使用 `user-1001`。仅监听 `localhost:9100`。登录提交检查浏览器来源，并在签发 Cookie 前显式拒绝损坏的 URL 编码表单或 multipart 表单。协议 POST 仍允许子系统调用。为与 `examples/sso_gin_client` 配套，请求签名未开启；Ticket 交换仍校验客户端密钥和回调地址。

中心使用内存存储，每次进程启动时随机生成 Cookie 签名密钥。重启会清空会话记录并使旧中心 Cookie 失效。Cookie 有效期两小时。清除浏览器 Cookie 不会在签名到期前吊销已复制的 Cookie；需要立即吊销时，应使用服务端会话解析器并同步删除会话。

此 Gin 示例设置了 `LogoutCallbackBestEffort: true`：尝试通知已登记子系统后，即使子系统离线，也会清理中心 Cookie 和回调记录。不可用的子系统可能仍保留本地会话，因此中心返回成功不代表所有子系统均已注销。如需回调失败时使中心注销失败并保留状态重试，应设为 `false`。

## 启动

```powershell
go run ./examples/sso_gin_server
```

在仓库根目录使用 Go 1.25 或更高版本执行。部署时还需接入真实认证、配置受管理的密钥并启用两端请求签名，以及使用 HTTPS 与 Secure Cookie。

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

在 `run` 中使用上述创建代码，让错误经由返回路径执行延迟清理。Redis 提供共享协议存储；仅替换存储不会替换模拟认证或进程独立的 Cookie 密钥。Redis 适配器通过原子操作消费 Ticket，要求 Redis 6.2 或更高版本。
