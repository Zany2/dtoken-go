# SSO Redis 存储

`github.com/Zany2/dtoken-go/sso/storage/redis` 提供适合生产环境的 SSO Redis 构造器。它复用 `com/storage/redis`，并为 `sso.Server` 注入 Redis 存储。

基础 `sso.NewServer()` 内置的是内存存储，只适合本地验证和单元测试。真实 SSO 服务建议使用 Redis，因为 Ticket 和 OAuth2 Code 都需要原子读删能力。

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
```

也可以使用 Redis 配置对象：

```go
server, err := ssoredis.NewServerFromConfig(&redis.Config{
	Host:     "127.0.0.1",
	Port:     6379,
	Password: "password",
	Database: 0,
})
```

如果你已经创建了 `*redis.Storage`：

```go
server := ssoredis.NewServerFromStorage(storage)
```
