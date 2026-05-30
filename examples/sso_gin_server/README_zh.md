# Gin SSO Server 示例

这个示例演示使用 Gin 部署一个统一登录中心，并挂载 `sso.HTTPServer` 的标准协议路由。

- `/login`：模拟统一登录页。
- `/sso/authorize`：校验中心登录态，生成 Ticket，并重定向回子系统。
- `/sso/token`：子系统使用 Ticket 换取登录主体信息。
- `/sso/logout`：清除中心 Cookie，并向已登记子系统推送统一登出回调。

## 启动

```powershell
go run ./examples/sso_gin_server
```

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
	log.Fatal(err)
}
```
