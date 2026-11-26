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

DToken-Go is an authentication and authorization framework for Go applications, providing login validation, automatic token renewal, roles and permissions, account bans, and multi-device session management. It supports isolated authentication systems, concurrent login control, and terminal logout. A unified API and adapters for mainstream web frameworks simplify integration, while storage, codecs, logging, and other components can be replaced as needed. Applications can compose Nonce, OAuth2, Ticket, ShortKey, and the independent SSO module. The framework is suitable for monolithic applications, distributed systems, and microservice architectures, with support for cross-system single sign-on and unified logout.

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

# Add the framework adapter when using Gin
go get github.com/Zany2/dtoken-go/integrations/gin
```

The root module includes Auth, public contracts, the internal engine, basic components, shared HTTP support, and the Nonce, OAuth2, Ticket, and ShortKey packages. These capabilities are composed independently but released with the root module.

Framework integrations, Redis storage, MsgPack, JSON v2, GoFrame logging, SSO, and SSO Redis storage use separate modules; add them as needed. Package paths and some APIs have changed from the previous directory structure. See the [migration guide](docs/design/module-migration.md) for upgrade steps and release version conventions.

## 5-Minute Quick Start

The default builder uses process-local memory storage and bundled basic components, so no external service is required. This example issues a token for an already identified user; the application verifies passwords or other identity credentials.

<!-- dtoken-compile: instance -->
```go
package main

import (
    "log"
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
        log.Print(err)
        return
    }
    defer auth.Close()

    ctx := context.Background()
    token, err := auth.LoginID(ctx, "user-1001")
    if err != nil {
        log.Print(err)
        return
    }
    if err := auth.AddRoles(ctx, dtoken.RoleOptions{LoginID: "user-1001", Roles: []string{"admin"}}); err != nil {
        log.Print(err)
        return
    }
    fmt.Println(auth.IsLogin(ctx, token))
    if err := auth.LogoutByToken(ctx, token); err != nil {
        log.Print(err)
        return
    }
}
```

Build automatically registers and returns an Auth. BuildInstance creates an unregistered instance. Default memory storage is not shared across processes; configure [Redis storage](docs/guide/integration/redis-storage.md) or implement `contract.Storage` when multiple instances need shared authentication data.

### Building And Calling With AuthType

Set `AuthType` when building separate user, administrator, or other authentication systems. `Build()` automatically registers and returns `*Auth`, with no additional `SetAuth` call. Construction errors are returned to the caller. This standalone example gives user tokens a one-day lifetime and administrator tokens a 30-minute lifetime; `Timeout` is measured in seconds:

<!-- dtoken-compile: multi-auth -->
```go
package main

import (
    "log"
    "context"
    "fmt"

    "github.com/Zany2/dtoken-go"
)

func main() {
    defer dtoken.CloseAll()
    userAuth, err := dtoken.NewBuilder().
        AuthType("user").TokenName("user-token").Timeout(86400).RenewMaxRefresh(3600).
        IsPrintBanner(false).Build()
    if err != nil {
        fmt.Println(err)
        return
    }
    if _, err := dtoken.NewBuilder().
        AuthType("admin").TokenName("admin-token").Timeout(1800).RenewMaxRefresh(300).
        IsPrintBanner(false).Build(); err != nil {
        fmt.Println(err)
        return
    }

    ctx := context.Background()
    token, err := dtoken.AuthType("user").Login(ctx, dtoken.LoginOptions{
        LoginID: "10001",
        Device: "web",
    })
    if err != nil {
        log.Print(err)
        return
    }
    if err := userAuth.CheckLogin(ctx, token); err != nil {
        log.Print(err)
        return
    }
    fmt.Println(dtoken.AuthType("user").IsLogin(ctx, token))  // true
    fmt.Println(dtoken.AuthType("admin").IsLogin(ctx, token)) // false
}
```

| Calling style | Authentication system |
| --- | --- |
| `dtoken.CheckLogin(ctx, token)` | Default namespace `auth:`, registered by omitting AuthType on the builder |
| `dtoken.AuthType("user").CheckLogin(ctx, token)` | Currently registered user system |
| `userAuth.CheckLogin(ctx, token)` | Fixed instance returned by the builder, sharing its engine with that registration |

This example registers only user and admin, with no default system. Missing systems do not fall back to the default: `CheckLogin` returns `ErrManagerNotFound`, and `IsLogin` returns false. Duplicate `Build` calls return `ErrManagerAlreadyRegistered`; use `BuildInstance()` for unregistered instances. Root business functions and instance methods have matching signatures. Select the system through the calling entry point; operation options contain only business parameters.

On application shutdown, stop accepting requests and drain active handlers before calling `dtoken.CloseAll()` to close current registrations. Close unregistered instances separately with `auth.Close()`. See [multi-auth](docs/guide/core/multi-auth.md) and [API and lifecycle](docs/design/api-lifecycle.md) for more details.

## Web Framework Integration

Gin, Echo, Fiber, Chi, GoFrame, Hertz, Kratos, and Beego are supported. Build instances from the root package and inject them through framework `WithAuth`. The Gin example below uses `Context(auth)` to bind an instance to requests and `RequireLogin()` to protect routes.

The example `/login` endpoint uses a fixed demo account to illustrate token issuance. Applications should verify identity before calling `LoginID`.

<!-- dtoken-compile: gin -->
```go
package main

import (
    "log"
    "net/http"

    "github.com/Zany2/dtoken-go"
    gindt "github.com/Zany2/dtoken-go/integrations/gin"
    "github.com/gin-gonic/gin"
)

func main() {
    auth, err := dtoken.NewBuilder().TokenName("Authorization").Build()
    if err != nil {
        log.Print(err)
        return
    }
    defer auth.Close()

    router := gin.Default()
    authMiddleware1, authErr := gindt.Context(auth)
    if authErr != nil {
        log.Print(authErr)
        return
    }
    router.Use(authMiddleware1)
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
        log.Print(err)
        return
    }
}
```

When requesting `/user/me`, put the token returned by login in the `Authorization` header. Middleware authenticates using the bound instance; explicitly configure `WithManagerResolver(dtoken.GetManager)` to select instances from the global registry. See the [framework integration guide](docs/guide/integration/framework-integration.md) for other integration options.

## Compose Independent Capabilities

Create Nonce, OAuth2, Ticket, and ShortKey through their package constructors. `NewService` accepts Auth as an optional event emitter. Services reuse the instance's namespace, storage, and codec where needed; the builder has no capability switches. HTTP binds capabilities through service interfaces under `contract`, allowing custom implementations.

In the Gin example above, add these imports and replace the instance-binding block with the binding fragment that follows, before registering any routes:

```go
import (
    "context"

    requestcontext "github.com/Zany2/dtoken-go/httpauth/context"
    "github.com/Zany2/dtoken-go/nonce"
)
```

```go
cfg := auth.Config()
nonceManager, err := nonce.NewDefaultNonceManager(cfg.AuthType, cfg.KeyPrefix, auth.Storage())
if err != nil {
    log.Print(err)
    return
}
service := nonce.NewService(nonceManager, auth)
authOption1, authErr := gindt.WithAuth(auth)
if authErr != nil {
    log.Print(authErr)
    return
}
options := []gindt.AuthOption{
    authOption1,
    gindt.WithCapabilities(requestcontext.Capabilities{Nonce: service}),
}
router.Use(gindt.RegisterDTokenContextMiddleware(context.Background(), options...))
```

Requests can now use the bound Nonce service; compose OAuth2, Ticket, and ShortKey in the same way. For unbound capabilities, operations returning errors report `ErrModuleNotEnabled`, while boolean checks return false. Services borrow instance components, so keep Auth open while they are in use. See the [Nonce guide](docs/guide/security/nonce.md) and [advanced capabilities guide](docs/guide/security/advanced-features.md) for complete usage.

## Architecture And Source Layout

Instance calls follow `application → Auth → authentication engine → component interfaces`. HTTP calls follow `framework adapter → shared HTTP layer → engine or capability service interfaces`. Applications assemble concrete components and capability services.

| Package | Responsibility |
| --- | --- |
| Root `dtoken` package | Instance/default APIs, AuthType selectors, shared registry, builder, and advanced extension types |
| `contract`, `config`, `errors`, `event` | Public contracts and shared models, core configuration, errors, event publishing and listener dispatch |
| `internal/engine` | Internal authentication engine; applications do not import it |
| `internal/builder` | Generic component assembly and bundled default component factories |
| `nonce`, `oauth2`, `ticket`, `shortkey` | Independently composed capabilities and services |
| `httpauth`, `httpauth/context` | Shared HTTP authentication and request contexts |
| `codec`, `generator`, `logging`, `pool`, `storage` | Pluggable component implementations |
| `integrations/<framework>` | Framework middleware, request adapters, and helpers |
| `sso` | Independent SSO subsystem with protocols, clients, and servers |
| `examples`, `tests`, `docs` | Integration examples, application checks, and documentation |
| `.github` | CI, repository check scripts, and collaboration configuration |

The engine depends on component contracts, and the shared HTTP layer depends on capability service contracts. They do not select concrete components or implicitly look up global instances. Package directories divide responsibilities, `go.mod` defines release modules, and `go.work` combines modules for local development.

Applications should prefer Auth; custom integrations can obtain the supported advanced API through `auth.Manager()`. Facades and middleware bound to the same engine share its lifecycle: closing Auth invalidates those bindings. On shutdown, stop accepting requests and wait for active requests to finish before closing Auth. Component ownership and registry replacement rules are documented in the lifecycle guide.

Maintainers can start with [architecture](docs/design/architecture.md), then use [source layout](docs/design/source-layout.md) to locate files and place changes. [Modular design](docs/design/modular.md) defines dependency and release boundaries; [API and lifecycle](docs/design/api-lifecycle.md) defines public contracts and resource ownership.

## Documentation And Examples

### Documentation

- [Documentation Center](docs/README.md)
- [Quick Start](docs/tutorial/quick-start.md)
- [Authentication](docs/guide/core/authentication.md)
- [Permission Management](docs/guide/core/permission.md)
- [Multi-Auth Systems](docs/guide/core/multi-auth.md)
- [Sessions And Terminals](docs/guide/core/session-terminal.md)
- [Disable System](docs/guide/core/disable.md)
- [Token Styles](docs/guide/core/token-style.md)
- [AccessProvider](docs/guide/core/access-provider.md)
- [Framework Integration](docs/guide/integration/framework-integration.md)
- [Event Listener](docs/guide/core/listener.md)
- [Nonce Anti-Replay](docs/guide/security/nonce.md)
- [JWT Integration](docs/guide/security/jwt.md)
- [Redis Storage](docs/guide/integration/redis-storage.md)
- [OAuth2](docs/guide/security/oauth2.md)
- [Ticket, ShortKey, And Advanced Capabilities](docs/guide/security/advanced-features.md)
- [SSO](sso/README.md)
- [SSO Redis Storage](sso/storage/redis/README.md)
- [SSO Testing](sso/TESTING.md)
- [Refresh Token](docs/guide/security/refresh-token.md)
- [Root API Reference](docs/api/dtoken.md)

### Example Projects

| Example | Description |
| --- | --- |
| [examples/quick_start](examples/quick_start/) | Auth instance with custom Gin handlers for login, role, and permission checks |
| [examples/gin](examples/gin/) | Gin middleware, login checks, and role checks |
| [examples/sso_gin_server](examples/sso_gin_server/) | Gin SSO login-center example |
| [examples/sso_gin_client](examples/sso_gin_client/) | Gin SSO client-app example |
| [examples/sso_server](examples/sso_server/) | net/http SSO login-center example |
| [examples/sso_client](examples/sso_client/) | net/http SSO client-app example |
| [examples/sso_deployment](examples/sso_deployment/README.md) | Complete signed SSO center and multiple apps, Redis, PKCE, replay protection, and deployment guide |
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
