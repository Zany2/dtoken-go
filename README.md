<p align="center">
  <img src="docs/assets/logo.png" alt="DToken-Go" width="100" height="100">
</p>

<h1 align="center">DToken-Go</h1>

<p align="center">
  A Go authentication, authorization, session management, and token lifecycle framework.
</p>

<p align="center">
  <a href="https://github.com/Zany2/dtoken-go"><img src="https://img.shields.io/badge/Go-1.20+-00ADD8?style=flat-square&logo=go" alt="Go"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-Apache--2.0-blue?style=flat-square" alt="License"></a>
  <a href="docs/README.md"><img src="https://img.shields.io/badge/Docs-English-brightgreen?style=flat-square" alt="Docs"></a>
  <a href="https://pkg.go.dev/github.com/Zany2/dtoken-go/dtoken"><img src="https://img.shields.io/badge/pkg.go.dev-dtoken-007D9C?style=flat-square&logo=go" alt="pkg.go.dev"></a>
</p>

<p align="center">
  English | <a href="README_zh.md">简体中文</a>
</p>

---

## What Is DToken-Go

DToken-Go is a modular and pluggable Go authentication and authorization framework. It already provides login authentication, Token management, Session management, terminal management, role and permission checks, account and device banning, Nonce anti-replay, OAuth2 server support, and event listeners. SSO, temporary Ticket credentials, short-key access credentials, Token Introspection, and standalone Refresh Token capabilities are under active development. The framework supports pluggable component replacement and custom extensions, and integrates with mainstream Go Web frameworks, so it can be used as an independent auth core or quickly embedded into existing business projects.

You can use it for:

- Admin systems, user centers, open platforms, and other systems that need unified authentication and authorization.
- Login and permission integration for Gin, Echo, Fiber, Chi, GoFrame, Hertz, Kratos, and other Go Web projects.
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
| SSO 🚧 | Unified login, ticket exchange, cross-system login-state sharing, unified logout, and application-level management |
| Temporary Ticket 🚧 | Ticket creation, validation, one-time consumption, revocation, TTL query, and status identification |
| Short-key access credential 🚧 | Generate random short keys for short-link access, QR confirmation, temporary authorization, and system-to-system ticket exchange |
| Token Introspection 🚧 | Standardized query for Token validity, ownership information, TTL, and invalid reason |
| Refresh Token 🚧 | Independent refresh token issuing, refreshing, revocation, expiration, rotation, and security checks |

> 🚧 means the feature is under development.

## Installation

### Default Core Usage

```bash
go get github.com/Zany2/dtoken-go/defaults
go get github.com/Zany2/dtoken-go/dtoken
```

### Web Framework Integration

If your project already uses a Web framework, you can use the corresponding DToken integration package directly. Integration packages wrap the Builder, middleware, context adapters, and common DToken APIs, making it easy to add authentication, authorization, and login-state management inside a specific framework.

```bash
go get github.com/Zany2/dtoken-go/integrations/gin
```

Available framework packages:

```bash
go get github.com/Zany2/dtoken-go/integrations/echo
go get github.com/Zany2/dtoken-go/integrations/fiber
go get github.com/Zany2/dtoken-go/integrations/chi
go get github.com/Zany2/dtoken-go/integrations/gf
go get github.com/Zany2/dtoken-go/integrations/hertz
go get github.com/Zany2/dtoken-go/integrations/kratos
```

### Pluggable Components

```bash
go get github.com/Zany2/dtoken-go/com/storage/redis
go get github.com/Zany2/dtoken-go/com/generator/dgenerator
go get github.com/Zany2/dtoken-go/com/pool/ants
```

## 5-Minute Quick Start

`defaults.NewBuilder()` wires the default memory storage, JSON codec, Token generator, and logger, making it suitable for quick experiments.

```go
package main

import (
	"context"
	"fmt"

	"github.com/Zany2/dtoken-go/defaults"
	"github.com/Zany2/dtoken-go/dtoken"
)

func main() {
	ctx := context.Background()

	mgr, err := defaults.NewBuilder().
		TokenName("Authorization"). // Read Token from Authorization
		Timeout(7200).              // Token lifetime: 7200 seconds
		AutoRenew(true).            // Auto-renew during login-state checks
		IsPrintBanner(false).       // Hide banner in this example
		Build()
	if err != nil {
		panic(err)
	}

	// Register the global Manager, then use dtoken global APIs.
	dtoken.SetManager(mgr)

	// Login and issue a Token.
	token, err := dtoken.Login(ctx, "user-1001")
	if err != nil {
		panic(err)
	}

	// Bind roles and permissions to the user.
	_ = dtoken.AddRoles(ctx, "user-1001", []string{"admin"})
	_ = dtoken.AddPermissions(ctx, "user-1001", []string{"article:read"})

	// Resolve login ID from Token and check access.
	loginID, _ := dtoken.GetLoginID(ctx, token)

	fmt.Println("token:", token)
	fmt.Println("loginID:", loginID)
	fmt.Println("is login:", dtoken.IsLogin(ctx, token))
	fmt.Println("is admin:", dtoken.HasRole(ctx, loginID, "admin"))
	fmt.Println("can read article:", dtoken.HasPermission(ctx, loginID, "article:read"))

	// Logout invalidates the Token.
	_ = dtoken.Logout(ctx, token)
}
```

See [examples/quick_start](examples/quick_start/) for a complete example.

## Web Framework Integration

The following Gin example only imports `integrations/gin`. Integration usually has three steps: initialize DToken, register middleware, and use login-state and permission capabilities in business routes.

```go
package main

import (
	"context"
	"net/http"

	gindt "github.com/Zany2/dtoken-go/integrations/gin"
	"github.com/gin-gonic/gin"
)

func main() {
	ctx := context.Background()

	mgr, err := gindt.NewBuilder().
		TokenName("Authorization"). // Read Token from Authorization Header
		Timeout(7200).              // Token lifetime: 7200 seconds
		AutoRenew(true).            // Auto-renew during login-state checks
		IsPrintBanner(false).       // Hide banner in this example
		Build()
	if err != nil {
		panic(err)
	}

	// Register the global Manager, then use APIs exported by gindt.
	gindt.SetManager(mgr)

	r := gin.Default()

	// Register context middleware, then read request auth context with GetDTokenContext.
	r.Use(gindt.RegisterDTokenContextMiddleware(ctx))

	// Customize response format for authentication or authorization failures.
	failFunc := gindt.WithFailFunc(func(c *gin.Context, err error) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    gindt.CodeNotLogin,
			"message": err.Error(),
		})
	})

	r.POST("/login", func(c *gin.Context) {
		// Issue a Token after login. Client should send Authorization: Bearer <token>.
		token, err := gindt.Login(c.Request.Context(), "user-1001")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Example only: normally roles come from your database or permission center.
		_ = gindt.AddRoles(c.Request.Context(), "user-1001", []string{"admin"})

		c.JSON(http.StatusOK, gin.H{"token": token})
	})

	user := r.Group("/user")

	// AuthMiddleware checks whether the request is logged in.
	user.Use(gindt.AuthMiddleware(ctx, failFunc))
	user.GET("/me", func(c *gin.Context) {
		// Resolve the current login account from request context.
		dCtx, _ := gindt.GetDTokenContext(c)
		loginID, _ := dCtx.GetLoginID(c.Request.Context())
		c.JSON(http.StatusOK, gin.H{"loginId": loginID})
	})

	admin := r.Group("/admin")

	// RoleMiddleware checks roles after login validation.
	admin.Use(gindt.AuthMiddleware(ctx, failFunc), gindt.RoleMiddleware(ctx, []string{"admin"}, failFunc))
	admin.GET("/dashboard", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	_ = r.Run(":8080")
}
```

| Framework | Example | Integration Package |
| --- | --- | --- |
| Gin | [examples/gin](examples/gin/) | `github.com/Zany2/dtoken-go/integrations/gin` |
| Echo | [examples/echo](examples/echo/) | `github.com/Zany2/dtoken-go/integrations/echo` |
| Fiber | [examples/fiber](examples/fiber/) | `github.com/Zany2/dtoken-go/integrations/fiber` |
| Chi | [examples/chi](examples/chi/) | `github.com/Zany2/dtoken-go/integrations/chi` |
| GoFrame | [examples/gf](examples/gf/) | `github.com/Zany2/dtoken-go/integrations/gf` |
| Hertz | [examples/hertz](examples/hertz/) | `github.com/Zany2/dtoken-go/integrations/hertz` |
| Kratos | [examples/kratos](examples/kratos/) | `github.com/Zany2/dtoken-go/integrations/kratos` |

## Deep Reading

README keeps only the minimal getting-started path. For more API, configuration, and component details, see:

- [Core API Cheatsheet](docs/guide/reference/core-api-cheatsheet.md)
- [Advanced Features](docs/guide/security/advanced-features.md)
- [Configuration Example](docs/guide/reference/configuration.md)
- [Component Ecosystem](docs/guide/integration/component-ecosystem.md)
- [Multi-Auth Systems](docs/guide/core/multi-auth.md)
- [Disable System](docs/guide/core/disable.md)
- [Token Styles](docs/guide/core/token-style.md)
- [AccessProvider](docs/guide/core/access-provider.md)

## Project Structure

```text
dtoken-go/
├── core/                         # Framework core: config, Manager, Session, events, Nonce, OAuth2
├── dtoken/                       # Global and instance API facade
├── defaults/                     # Default Builder and default component wiring
├── com/                          # Pluggable component implementations
│   ├── codec/                    # Codec components, such as JSON, MessagePack, Base64
│   ├── generator/                # Token generators
│   ├── log/                      # Logger components
│   ├── pool/                     # Goroutine pool components
│   └── storage/                  # Storage components, such as Memory and Redis
├── integrations/                 # Web framework integration packages
│   ├── gin/                      # Gin middleware, context adapter, and API exports
│   ├── echo/                     # Echo integration
│   ├── fiber/                    # Fiber integration
│   ├── chi/                      # Chi integration
│   ├── gf/                       # GoFrame integration
│   ├── hertz/                    # Hertz integration
│   └── kratos/                   # Kratos integration
├── examples/                     # Quick-start and framework integration examples
├── tests/                        # Flow tests and test applications
│   ├── gin_core_app/             # Gin core flow test application
│   └── gin_core_flow/            # HTTP flow tests for core features
├── docs/                         # Guides, API references, and design docs
│   ├── guide/core/               # Core capabilities such as login, permissions, Session, terminals, and disable
│   ├── guide/security/           # Security and protocol features such as Nonce, OAuth2, SSO, JWT, and Refresh Token
│   ├── guide/integration/        # Web frameworks, annotations, components, Redis, and flow tests
│   ├── guide/reference/          # Configuration examples and Core API cheatsheet
│   ├── api/                      # API references
│   ├── design/                   # Architecture and design docs
│   └── tutorial/                 # Quick-start tutorials
├── README_zh.md                  # Chinese project README
├── README.md                     # English project README
└── go.work                       # Go workspace
```

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
- [Refresh Token](docs/guide/security/refresh-token.md)
- [API Reference](docs/api/dtoken.md)

### Example Projects

| Example | Description |
| --- | --- |
| [examples/quick_start](examples/quick_start/) | Minimal usage with the default Builder and global API |
| [examples/gin](examples/gin/) | Gin middleware, login checks, and role checks |
| [examples/echo](examples/echo/) | Echo framework integration example |
| [examples/fiber](examples/fiber/) | Fiber framework integration example |
| [examples/chi](examples/chi/) | Chi framework integration example |
| [examples/gf](examples/gf/) | GoFrame framework integration example |
| [examples/hertz](examples/hertz/) | Hertz framework integration example |
| [examples/kratos](examples/kratos/) | Kratos framework integration example |

## Contributing

Issues, pull requests, and feedback are welcome. Please keep changes focused and follow the existing module boundaries and coding style.

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
