# SSO Redis 存储

`github.com/Zany2/dtoken-go/sso/storage/redis` 提供适合生产环境的 SSO Redis 构造器。它复用 `com/storage/redis`，并为 `sso.Server` 注入 Redis 存储。

基础 `sso.NewServer()` 使用进程内的内存存储。多个 SSO 实例需要共享客户端和凭证时，应使用 Redis。本适配器通过 `GETDEL` 原子消费 Ticket 和 OAuth2 Code，要求 Redis 6.2 或更高版本。

Redis 集成测试读取 `DTOKEN_REDIS_URL`；未设置该变量时，测试会自动跳过。

## 使用方式

```go
import (
	"github.com/Zany2/dtoken-go/sso"
	ssoredis "github.com/Zany2/dtoken-go/sso/storage/redis"
)

server, err := ssoredis.NewServer(
	"redis://:password@127.0.0.1:6379/0",
	sso.WithConfig(sso.DefaultConfig()),
)
if err != nil {
	return err
}
defer server.Close()
```

也可以使用 Redis 配置对象：

```go
import redisstorage "github.com/Zany2/dtoken-go/com/storage/redis"

server, err := ssoredis.NewServerFromConfig(&redisstorage.Config{
	Host:     "127.0.0.1",
	Port:     6379,
	Password: "password",
	Database: 0,
})
if err != nil {
	return err
}
defer server.Close()
```

URL/Config 构造器会通过 `PING` 验证连接，初始化失败时关闭新建客户端。成功后由 `server.Close()` 负责关闭，且只执行一次。这两种构造器的 Redis 存储及其所有权优先于 `WithStorage` 和 `WithStorageOwnership`；其他 SSO 选项仍然有效，避免覆盖存储后遗留无人关闭的连接。

如果你已经创建了 `*redis.Storage`：

```go
server := ssoredis.NewServerFromStorage(storage)
```

如果你已经有 `*go-redis` 客户端：

```go
import goredis "github.com/redis/go-redis/v9"

client := goredis.NewClient(&goredis.Options{
	Addr: "127.0.0.1:6379",
})
defer client.Close()

server := ssoredis.NewServerFromClient(client)
```

`NewServerFromClient` 和 `NewServerFromStorage` 默认不会发送 `PING`、修改连接配置或关闭注入资源；连通性检查、超时设置和清理由调用方负责。传入 nil 不会回退到内存存储，使用存储时会返回错误。这两种构造器保留普通 SSO 选项的顺序语义：`WithStorage` 可以替换注入适配器，在最终存储选项之后传入 `WithStorageOwnership(true)` 可将其关闭责任转移给 Server。

集成测试每次运行使用独立的键前缀，并在关闭连接前清理测试记录。设置 `DTOKEN_REDIS_URL` 指向测试 Redis 实例后即可运行。
