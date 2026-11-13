<p align="center">
  <img src="docs/assets/logo.png" alt="DToken-Go" width="100" height="100">
</p>

<h1 align="center">DToken-Go</h1>

<p align="center">
  A Go authentication, authorization, session management, and token lifecycle framework.
</p>

<p align="center">
  <a href="https://github.com/Zany2/dtoken-go"><img src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go" alt="Go"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-Apache--2.0-blue?style=flat-square" alt="License"></a>
  <a href="docs/README.md"><img src="https://img.shields.io/badge/Docs-English-brightgreen?style=flat-square" alt="Docs"></a>
  <a href="https://pkg.go.dev/github.com/Zany2/dtoken-go"><img src="https://img.shields.io/badge/pkg.go.dev-dtoken-007D9C?style=flat-square&logo=go" alt="pkg.go.dev"></a>
</p>

<p align="center">
  English | <a href="README_zh.md">简体中文</a>
</p>

---

## What Is DToken-Go

DToken-Go is a modular and pluggable Go authentication and authorization framework. It already provides login authentication, Token management, Session management, terminal management, role and permission checks, account and device banning, Nonce anti-replay, OAuth2 server support, event listeners, temporary tickets, short-key access credentials, Token Introspection, and standalone Refresh Token capabilities, while the SSO module is still under development. The framework supports pluggable component replacement and custom extensions, and integrates with mainstream Go Web frameworks, so it can be used as an independent auth core or quickly embedded into existing business projects.

You can use it for:

- Admin systems, user centers, open platforms, and other systems that need unified authentication and authorization.
- Login and permission integration for Gin, Echo, Fiber, Chi, GoFrame, Hertz, Kratos, Beego, and other Go Web projects.
- Microservice gateways, centralized authentication centers, cross-system SSO, and unified logout.
- Login state and session management across apps, mini-programs, web clients, multiple devices, and multiple terminals.
- QR confirmation, one-time access, temporary authorization, short-link credentials, and third-party Token validation.

## Core Features

| Feature | Description |
| --- | --- |
| Login authentication | Login, continued login, logout, login-state checks, Token info queries, TTL queries, manual renewal, and auto-renewal |
| Session management | Query and manage login state by account, Token, device, and device ID |
| Terminal management | Multi-terminal login, terminal tracking, online terminal statistics, terminal cleanup, kickout, and replacement |
| Roles and permissions | Role and permission mutation/query, AND/OR checks, Token-level checks, and wildcard permission matching |
| Concurrency control | Same-account concurrent login control, shared Token, max online terminal limit, account/device scope |
| Account and device banning | Account ban, service ban, device ban, unban, ban reason, and remaining ban time query |
| Multi-auth systems | Isolate multiple auth systems by AuthType so Tokens, Sessions, permissions, and roles do not cross-use |
| Nonce anti-replay | One-time random value generation, verification, and consumption to prevent replay attacks |
| OAuth2 | Authorization code, client credentials, password grant, refresh token, Token validation, and revocation |
| Event system | Listeners for login, logout, renewal, permissions, roles, bans, unbans, and other core lifecycle events |
| Pluggable components | Storage, codec, logger, Token generator, goroutine pool, and other components can be replaced |
| Framework integration | Middleware, context adapters, and API exports for mainstream Go Web frameworks |
| SSO (under development) | Unified login, ticket exchange, cross-system login-state sharing, unified logout, and application-level management |
| Temporary Ticket | Ticket creation, validation, one-time consumption, revocation, TTL query, and status identification |
| Short-key access credential | Generate random short keys for short-link access, QR confirmation, temporary authorization, and system-to-system ticket exchange |
| Token Introspection | Standardized query for Token validity, ownership information, TTL, and invalid reason |
| Refresh Token | Independent refresh token issuing, refreshing, revocation, expiration, rotation, and security checks |

## Installation

The root module requires Go 1.25+; repository workspace development requires Go 1.27+. The optional `codec/jsonv2` module requires `GOEXPERIMENT=jsonv2`.

```bash
go get github.com/Zany2/dtoken-go
go get github.com/Zany2/dtoken-go/integrations/gin
```

Framework integrations, SSO, Redis storage, MsgPack, JSON v2, and GoFrame logging use separate modules. See the [migration guide](docs/design/module-migration.md) for package paths and upgrades.

## 5-Minute Quick Start

<!-- dtoken-compile: instance -->
```go
package main

import (
    "context"
    "fmt"

    "github.com/Zany2/dtoken-go"
)

func main() {
    auth, err := dtoken.NewBuilder().
        AuthType("user").
        TokenName("Authorization").
        Timeout(7200).
        IsPrintBanner(false).
        Build()
    if err != nil {
        panic(err)
    }
    defer auth.Close()

    ctx := context.Background()
    token, err := auth.LoginID(ctx, "user-1001")
    if err != nil {
        panic(err)
    }
    _ = auth.AddRoles(ctx, dtoken.RoleOptions{LoginID: "user-1001", Roles: []string{"admin"}})
    fmt.Println(auth.IsLogin(ctx, token))
    _ = auth.LogoutByToken(ctx, token)
}
```

For global APIs, explicitly import `github.com/Zany2/dtoken-go/global`, call `global.SetAuth(auth)`, and clean up registrations with `global.DeleteManager` or `global.DeleteAllManager`. Instances can operate without global registration.

## Web Framework Integration

Build instances from the root package and inject them through framework `WithAuth` or `Context`.

<!-- dtoken-compile: gin -->
```go
package main

import (
    "net/http"

    "github.com/Zany2/dtoken-go"
    gindt "github.com/Zany2/dtoken-go/integrations/gin"
    "github.com/gin-gonic/gin"
)

func main() {
    auth, err := dtoken.NewBuilder().TokenName("Authorization").Build()
    if err != nil {
        panic(err)
    }
    defer auth.Close()

    router := gin.Default()
    router.Use(gindt.Context(auth))
    router.POST("/login", func(c *gin.Context) {
        token, err := auth.LoginID(c.Request.Context(), "user-1001")
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
            return
        }
        c.JSON(http.StatusOK, gin.H{"token": token})
    })
    users := router.Group("/user")
    users.Use(gindt.RequireLogin())
    users.GET("/me", func(c *gin.Context) {
        loginID, err := gindt.GetLoginIDByContext(c)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
            return
        }
        c.JSON(http.StatusOK, gin.H{"loginId": loginID})
    })
    if err := router.Run(":8080"); err != nil {
        panic(err)
    }
}
```

## Compose Independent Capabilities

Create Nonce, OAuth2, Ticket, and ShortKey through their package constructors. `NewService` accepts an authentication instance as an optional event emitter. Compose services using the same namespace and storage as the instance; the builder has no capability switches. Inject request services with `WithCapabilities`; unbound capabilities return `ErrModuleNotEnabled`. Use this fragment before registering routes and also import the standard `context` package.

```go
// Imports: nonce "github.com/Zany2/dtoken-go/nonce",
// requestcontext "github.com/Zany2/dtoken-go/httpauth/context".
cfg := auth.Config()
service := nonce.NewService(
    nonce.NewDefaultNonceManager(cfg.AuthType, cfg.KeyPrefix, auth.Storage()),
    auth,
)
options := []gindt.AuthOption{
    gindt.WithAuth(auth),
    gindt.WithCapabilities(requestcontext.Capabilities{Nonce: service}),
}
router.Use(gindt.RegisterDTokenContextMiddleware(context.Background(), options...))
```

## Package Boundaries

| Package | Responsibility |
| --- | --- |
| `dtoken` | Instance facade and default builder |
| `global` | Global authentication API and registry |
| `contract`、`config`、`errors`、`event` | Public contracts, configuration, errors, and events |
| `internal/engine` | Internal authentication engine; applications do not import it |
| `nonce`、`oauth2`、`ticket`、`shortkey` | Independently composed capabilities and services |
| `httpauth`、`httpauth/context` | Shared HTTP authentication and request contexts |
| `codec`、`generator`、`logging`、`pool`、`storage` | Pluggable component implementations |
| `integrations/<framework>`、`sso` | Framework integrations and the independent SSO module |

## Documentation And Examples

### Documentation

- [Documentation Center](docs/README.md)
- [Quick Start](docs/tutorial/quick-start.md)
- [Authentication](docs/guide/core/authentication.md)
- [Permission Management](docs/guide/core/permission.md)
- [Multi-Auth Systems](docs/guide/core/multi-auth.md)
- [Disable System](docs/guide/core/disable.md)
- [Token Styles](docs/guide/core/token-style.md)
- [AccessProvider](docs/guide/core/access-provider.md)
- [Framework Integration](docs/guide/integration/framework-integration.md)
- [Event Listener](docs/guide/core/listener.md)
- [Nonce Anti-Replay](docs/guide/security/nonce.md)
- [JWT Integration](docs/guide/security/jwt.md)
- [Redis Storage](docs/guide/integration/redis-storage.md)
- [OAuth2](docs/guide/security/oauth2.md)
- [SSO](sso/README.md)
- [SSO Testing](sso/TESTING.md)
- [Refresh Token](docs/guide/security/refresh-token.md)
- [API Reference](docs/api/dtoken.md)

### Example Projects

| Example | Description |
| --- | --- |
| [examples/quick_start](examples/quick_start/) | Minimal usage with the default Builder and global API |
| [examples/gin](examples/gin/) | Gin middleware, login checks, and role checks |
| [examples/sso_gin_server](examples/sso_gin_server/) | Gin SSO login-center example |
| [examples/sso_gin_client](examples/sso_gin_client/) | Gin SSO client-app example |
| [examples/sso_server](examples/sso_server/) | net/http SSO login-center example |
| [examples/sso_client](examples/sso_client/) | net/http SSO client-app example |
| [examples/echo](examples/echo/) | Echo framework integration example |
| [examples/fiber](examples/fiber/) | Fiber framework integration example |
| [examples/chi](examples/chi/) | Chi framework integration example |
| [examples/gf](examples/gf/) | GoFrame framework integration example |
| [examples/hertz](examples/hertz/) | Hertz framework integration example |
| [examples/kratos](examples/kratos/) | Kratos framework integration example |
| [examples/beego](examples/beego/) | Beego framework integration example |

## Contributing

Issues, pull requests, and feedback are welcome. Please keep changes focused and follow the existing module boundaries and coding style.

See the [contributing guide](.github/CONTRIBUTING.md) for development and verification instructions, and the [security policy](.github/SECURITY.md) for vulnerability reporting.

## Star History

<picture>
  <source
    media="(prefers-color-scheme: dark)"
    srcset="https://api.star-history.com/svg?repos=Zany2/dtoken-go&type=Date&theme=dark"
  />
  <source
    media="(prefers-color-scheme: light)"
    srcset="https://api.star-history.com/svg?repos=Zany2/dtoken-go&type=Date"
  />
  <img
    alt="Star History Chart"
    src="https://api.star-history.com/svg?repos=Zany2/dtoken-go&type=Date"
  />
</picture>

## License

DToken-Go is open source under the [Apache-2.0](LICENSE) license.
