# Gin SSO Server Example

This example shows a centralized login center built with Gin and the standard `sso.HTTPServer` protocol routes.

- `/login`: mock centralized login page.
- `/sso/authorize`: checks center login state, issues a Ticket, and redirects back to the client app.
- `/sso/token`: exchanges a Ticket for login subject information.
- `/sso/logout`: clears the center Cookie and pushes unified logout callbacks to registered client apps.

## Run

```powershell
go run ./examples/sso_gin_server
```

Default address:

```text
http://localhost:9100
```

Start `examples/sso_gin_client` at the same time, then open:

```text
http://localhost:9101/protected
```

## Redis Storage

The example uses in-memory storage by default so it can run locally without Redis. For production, replace `sso.NewServer()` with the Redis constructor:

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
