# SSO Redis 存储

[English](README.md) | 中文文档

`github.com/Zany2/dtoken-go/sso/storage/redis` 为 `sso.Server` 提供 Redis 存储构造器。它组合独立的 `sso` 与 `storage/redis` 模块，复用公共 Redis 组件，不重复实现存储逻辑。SSO 存储通过这些构造器或 `sso.WithStorage` 配置；根包 `dtoken.Auth` 的配置不会自动应用到 SSO Server。

基础 `sso.NewServer()` 使用进程内的内存存储。多个 SSO 实例需要共享客户端和凭证时，应使用 Redis。本适配器通过 `GETDEL` 原子消费 Ticket 和 OAuth2 Code，要求 Redis 6.2 或更高版本。

本模块要求 Go 1.25 或更高版本；通过仓库 `go.work` 开发时要求 Go 1.27 或更高版本。依赖路径和发布版本约定见[模块迁移指南](../../../docs/design/module-migration_zh.md)。

## 构造器选择

| 构造器 | 输入 | 连接检查 | 默认关闭责任方 |
| --- | --- | --- | --- |
| `NewServer` | Redis URL | 构造时执行 `PING`，失败时返回错误 | SSO Server |
| `NewServerFromConfig` | `*redisstorage.Config` | 构造时执行 `PING`，失败时返回错误 | SSO Server |
| `NewServerFromStorage` | 已有 `*redisstorage.Storage` | 不自动检查 | 调用方 |
| `NewServerFromClient` | 已有 `*goredis.Client` | 不自动检查 | 调用方 |

## 使用方式

以下片段展示不同的构造方式，选择其中一种即可。导入语句放在文件级，构造代码放在返回 error 的应用函数内。创建 SSO Server 不会启动 HTTP 监听；协议和 HTTP 接入见 [SSO 指南](../../README_zh.md)。

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
import redisstorage "github.com/Zany2/dtoken-go/storage/redis"

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

如果已经创建了 `github.com/Zany2/dtoken-go/storage/redis` 中的 `*redisstorage.Storage`，可以让 Server 在其生命周期内借用：

```go
server := ssoredis.NewServerFromStorage(storage)
defer server.Close()
```

调用方应在所有使用者结束后关闭该存储。如果已经有 `*goredis.Client`：

```go
import goredis "github.com/redis/go-redis/v9"

client := goredis.NewClient(&goredis.Options{
	Addr: "127.0.0.1:6379",
})
defer client.Close()

server := ssoredis.NewServerFromClient(client)
defer server.Close()
```

`NewServerFromClient` 和 `NewServerFromStorage` 默认不会发送 `PING`、修改连接配置或关闭注入资源；连通性检查、超时设置和清理由调用方负责。传入 nil 不会回退到内存存储，使用存储时会返回错误。这两种构造器保留普通 SSO 选项的顺序语义：`WithStorage` 可以替换注入适配器，在最终存储选项之后传入 `WithStorageOwnership(true)` 可将其关闭责任转移给 Server。

关闭 Server 前应先停止 HTTP 请求处理。使用注入资源时，应在所有使用者结束后再关闭共享客户端或存储。客户端示例中的 defer 会先关闭 Server，再关闭客户端。

## 验证

Redis 集成测试读取 `DTOKEN_REDIS_URL`；未设置时，仅跳过需要真实 Redis 的测试，构造器和所有权测试仍可在无 Redis 的环境中运行。集成测试每次运行使用独立的键前缀，并在关闭连接前清理测试记录。验证方式见 [SSO 测试指南](../../TESTING_zh.md)。
