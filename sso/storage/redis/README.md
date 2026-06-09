# SSO Redis Storage

`github.com/Zany2/dtoken-go/sso/storage/redis` provides production-oriented Redis constructors for SSO. It reuses `com/storage/redis` and injects Redis storage into `sso.Server`.

The base `sso.NewServer()` uses in-memory storage and is intended only for local verification and unit tests. Real SSO services should use Redis because Ticket and OAuth2 Code consumption require atomic get-and-delete behavior.

## Usage

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

You can also use a Redis config:

```go
import redisstorage "github.com/Zany2/dtoken-go/com/storage/redis"

server, err := ssoredis.NewServerFromConfig(&redisstorage.Config{
	Host:     "127.0.0.1",
	Port:     6379,
	Password: "password",
	Database: 0,
})
```

If you already have a `*redis.Storage`:

```go
server := ssoredis.NewServerFromStorage(storage)
```

If you already have a `*go-redis` client:

```go
import goredis "github.com/redis/go-redis/v9"

client := goredis.NewClient(&goredis.Options{
	Addr: "127.0.0.1:6379",
})

server := ssoredis.NewServerFromClient(client)
```
