# SSO Redis Storage

[中文文档](README_zh.md) | English

`github.com/Zany2/dtoken-go/sso/storage/redis` provides Redis-backed constructors for `sso.Server`. It composes the independent `sso` and `storage/redis` modules, reusing the shared Redis component without duplicating its implementation. SSO storage is selected through these constructors or `sso.WithStorage`; configuring a root `dtoken.Auth` does not configure an SSO server.

The base `sso.NewServer()` uses process-local in-memory storage. Use Redis when multiple SSO instances need to share clients and credentials. This adapter uses `GETDEL` for atomic Ticket and OAuth2 Code consumption and requires Redis 6.2 or later.

This module requires Go 1.25 or later; development through the repository's `go.work` requires Go 1.27 or later. See the [module migration guide](../../../docs/design/module-migration.md) for dependency paths and release versions.

## Constructor selection

| Constructor | Input | Connection check | Default cleanup owner |
| --- | --- | --- | --- |
| `NewServer` | Redis URL | `PING` during construction; returns an error on failure | SSO server |
| `NewServerFromConfig` | `*redisstorage.Config` | `PING` during construction; returns an error on failure | SSO server |
| `NewServerFromStorage` | Existing `*redisstorage.Storage` | No automatic check | Caller |
| `NewServerFromClient` | Existing `*goredis.Client` | No automatic check | Caller |

## Usage

The following snippets show alternative construction methods. Place imports at file scope and construction code inside an application function that returns an error. Creating an SSO server does not start an HTTP listener; see the [SSO guide](../../README.md) for protocol and HTTP setup.

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

URL/config constructors validate the connection with `PING` and close their newly created client if initialization fails. On success, `server.Close()` closes that client once. Their Redis storage and ownership take precedence over `WithStorage` and `WithStorageOwnership`; other SSO options remain effective. This prevents a storage override from orphaning the connection.

If you already have a `*redisstorage.Storage` from `github.com/Zany2/dtoken-go/storage/redis`, borrow it for the server's lifetime:

```go
server := ssoredis.NewServerFromStorage(storage)
defer server.Close()
```

The caller closes that storage after all borrowers have stopped using it. If you already have a `*goredis.Client`:

```go
import goredis "github.com/redis/go-redis/v9"

client := goredis.NewClient(&goredis.Options{
	Addr: "127.0.0.1:6379",
})
defer client.Close()

server := ssoredis.NewServerFromClient(client)
defer server.Close()
```

`NewServerFromClient` and `NewServerFromStorage` do not issue `PING`, change connection options, or close injected resources by default. The caller owns connectivity checks, timeout settings, and cleanup. Passing nil does not select in-memory storage; storage operations return an error. These constructors retain ordinary SSO option ordering: `WithStorage` may replace the injected adapter, and `WithStorageOwnership(true)` after the final storage selection transfers its cleanup responsibility to the server.

Stop HTTP request handling before closing the server. With injected resources, close the shared client or storage only after all users have finished. In the client example, deferred calls close the server before the client.

## Verification

Redis integration tests read `DTOKEN_REDIS_URL`; only tests requiring a live Redis instance are skipped when it is unset. Constructor and ownership tests can still run without Redis. Integration tests use a unique key prefix per run and clean up their records before closing the connection. See the [SSO testing guide](../../TESTING.md) for verification instructions.
