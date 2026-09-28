# SSO Redis Storage

`github.com/Zany2/dtoken-go/sso/storage/redis` provides production-oriented Redis constructors for SSO. It reuses `com/storage/redis` and injects Redis storage into `sso.Server`.

The base `sso.NewServer()` uses process-local in-memory storage. Use Redis when multiple SSO instances need to share clients and credentials. This adapter uses `GETDEL` for atomic Ticket and OAuth2 Code consumption and requires Redis 6.2 or later.

The Redis integration tests read `DTOKEN_REDIS_URL`; when it is unset, the tests are skipped.

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
defer server.Close()
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
if err != nil {
	return err
}
defer server.Close()
```

URL/config constructors validate the connection with `PING` and close their newly created client if initialization fails. On success, `server.Close()` closes that client once. Their Redis storage and ownership take precedence over `WithStorage` and `WithStorageOwnership`; other SSO options remain effective. This prevents a storage override from orphaning the connection.

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
defer client.Close()

server := ssoredis.NewServerFromClient(client)
```

`NewServerFromClient` and `NewServerFromStorage` do not issue `PING`, change connection options, or close injected resources by default. The caller owns connectivity checks, timeout settings, and cleanup. Passing nil does not select in-memory storage; storage operations return an error. These constructors retain ordinary SSO option ordering: `WithStorage` may replace the injected adapter, and `WithStorageOwnership(true)` after the final storage selection transfers its cleanup responsibility to the server.

Integration tests use a unique key prefix per run and clean up their records before closing the connection. Set `DTOKEN_REDIS_URL` to a test Redis instance to run them.
