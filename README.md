<p align="center">
  <img src="docs/assets/logo.png" alt="DToken-Go" width="100" height="100">
</p>

<h1 align="center">DToken-Go</h1>

<p align="center">
  An authentication, authorization, session management, and SSO framework for Go applications.
</p>

<p align="center">
  <a href="https://github.com/Zany2/dtoken-go"><img src="https://img.shields.io/badge/Go-1.20+-00ADD8?style=flat-square&logo=go" alt="Go"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-Apache--2.0-blue?style=flat-square" alt="License"></a>
  <a href="docs/README.md"><img src="https://img.shields.io/badge/Docs-English-brightgreen?style=flat-square" alt="Docs"></a>
</p>

<p align="center">
  English | <a href="README_zh.md">中文</a>
</p>

---

## What Is DToken-Go

DToken-Go is a modular and pluggable authentication and authorization framework for Go. It provides login authentication, token management, session management, terminal management, role and permission checks, account banning, SSO, short-key login, temporary ticket credentials, token introspection, refresh tokens, nonce anti-replay utilities, OAuth2 server capabilities, and event listeners.

You can use it for:

- Admin systems, user centers, and open platforms.
- Gin, Echo, Fiber, Chi, GoFrame, Hertz, Kratos, and other Go web projects.
- Microservice gateways, centralized authentication centers, and cross-system SSO.
- App, mini-program, web, multi-device, and multi-terminal login-state management.
- QR-code login, one-time login, temporary authorization, and third-party token verification.

## Table Of Contents

- [Core Features](#core-features)
- [Installation](#installation)
- [5-Minute Quick Start](#5-minute-quick-start)
- [Web Framework Integration](#web-framework-integration)
- [Core API Cheatsheet](#core-api-cheatsheet)
- [Advanced Features](#advanced-features)
- [Configuration Example](#configuration-example)
- [Component Ecosystem](#component-ecosystem)
- [Project Structure](#project-structure)
- [Documentation And Examples](#documentation-and-examples)

## Core Features

| Feature | Description |
| --- | --- |
| Login authentication | Login, continued login, logout, login-state checks, token info queries, TTL queries, manual renewal, and auto-renewal |
| Session management | Query and manage login states by account, token, device, device ID, and application |
| Terminal management | Multi-terminal login, terminal tracking, online terminal statistics, terminal cleanup, kickout, and replacement |
| Roles and permissions | Add, remove, query, AND/OR checks, token-level checks, and wildcard permission matching |
| Concurrency control | Concurrent login control for the same account, shared tokens, and maximum online terminal limits |
| Account banning | Account banning, device banning, unbanning, ban reasons, and remaining ban time queries |
| SSO | Centralized login, ticket exchange, shared cross-system login state, unified logout, and application-level management |
| Temporary tickets | Ticket creation, validation, one-time consumption, revocation, TTL queries, and status identification |
| Short-key login | Suitable for QR-code login, one-time login, temporary authorization, and system-to-system ticket exchange |
| Token introspection | Standardized token validity, ownership, TTL, and invalid-reason queries |
| Refresh tokens | Access-token refresh, refresh-token revocation, expiration, rotation, and security checks |
| Nonce anti-replay | One-time nonce generation, verification, and consumption to prevent replay attacks |
| OAuth2 | Authorization code, client credentials, password grant, refresh token, token validation, and revocation |
| Event system | Event listeners for login, logout, renewal, tickets, short keys, SSO, and more |
| Pluggable components | Replaceable storage, codec, logger, token generator, and goroutine pool |
| Multiple frameworks | Middleware, context adapters, and API exports for mainstream Go web frameworks |

## Installation

### Default Core Usage

```bash
go get github.com/Zany2/dtoken-go/defaults
go get github.com/Zany2/dtoken-go/dtoken
```

### Web Framework Integration

If your project already uses a web framework, it is recommended to import the corresponding integration package directly. Integration packages export the Builder, middleware, context adapters, and common DToken APIs.

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

`defaults.NewBuilder()` wires memory storage, JSON codec, the default token generator, and logging components by default, making it suitable for quick experiments.

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
		TokenName("Authorization").
		Timeout(7200).
		ActiveTimeout(1800).
		AutoRenew(true).
		IsPrintBanner(false).
		Build()
	if err != nil {
		panic(err)
	}
	dtoken.SetManager(mgr)

	token, err := dtoken.Login(ctx, "user-1001")
	if err != nil {
		panic(err)
	}

	_ = dtoken.AddRoles(ctx, "user-1001", []string{"admin"})
	_ = dtoken.AddPermissions(ctx, "user-1001", []string{"article:read", "article:write"})

	loginID, _ := dtoken.GetLoginID(ctx, token)
	hasRole := dtoken.HasRole(ctx, loginID, "admin")
	hasPermission := dtoken.HasPermission(ctx, loginID, "article:read")

	fmt.Println("token:", token)
	fmt.Println("loginID:", loginID)
	fmt.Println("has role:", hasRole)
	fmt.Println("has permission:", hasPermission)

	_ = dtoken.Logout(ctx, token)
}
```

See [examples/quick_start](examples/quick_start/) for a complete example.

## Web Framework Integration

The following Gin example only imports `integrations/gin` in application code:

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
		TokenName("Authorization").
		IsPrintBanner(false).
		Build()
	if err != nil {
		panic(err)
	}
	gindt.SetManager(mgr)

	r := gin.Default()
	r.Use(gindt.RegisterDTokenContextMiddleware(ctx))

	r.POST("/login", func(c *gin.Context) {
		token, err := gindt.Login(c.Request.Context(), "user-1001")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"token": token})
	})

	user := r.Group("/user")
	user.Use(gindt.AuthMiddleware(ctx))
	user.GET("/me", func(c *gin.Context) {
		dCtx, _ := gindt.GetDTokenContext(c)
		loginID, _ := dCtx.GetLoginID(c.Request.Context())
		c.JSON(http.StatusOK, gin.H{"loginId": loginID})
	})

	admin := r.Group("/admin")
	admin.Use(gindt.AuthMiddleware(ctx), gindt.RoleMiddleware(ctx, []string{"admin"}))
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

## Core API Cheatsheet

### Login Authentication

```go
token, err := dtoken.Login(ctx, "user-1001")
token, err = dtoken.Login(ctx, "user-1001", "web", "browser-001", "user")

isLogin := dtoken.IsLogin(ctx, token)
loginID, err := dtoken.GetLoginID(ctx, token)
tokenInfo, err := dtoken.GetTokenInfo(ctx, token)
ttl, err := dtoken.GetTokenTTL(ctx, token)

err = dtoken.RenewTimeout(ctx, token, 7200)
err = dtoken.Logout(ctx, token)
err = dtoken.LogoutByLoginID(ctx, "user-1001")
```

### Roles And Permissions

```go
_ = dtoken.AddRoles(ctx, "user-1001", []string{"admin", "auditor"})
_ = dtoken.AddPermissions(ctx, "user-1001", []string{"order:read", "order:*"})

hasRole := dtoken.HasRole(ctx, "user-1001", "admin")
hasAnyRole := dtoken.HasRolesOr(ctx, "user-1001", []string{"admin", "owner"})
hasPermission := dtoken.HasPermission(ctx, "user-1001", "order:read")
hasAllPermission := dtoken.HasPermissionsAnd(ctx, "user-1001", []string{"order:read", "order:write"})

_, _, _, _ = hasRole, hasAnyRole, hasPermission, hasAllPermission
```

### Online State Control

```go
_ = dtoken.Kickout(ctx, token)
_ = dtoken.KickoutByLoginID(ctx, "user-1001")
_ = dtoken.Replace(ctx, token)
_ = dtoken.ReplaceByLoginID(ctx, "user-1001")
```

### Account Banning

```go
_ = dtoken.Disable(ctx, "user-1001", 3600, "risk_control")
disabled := dtoken.IsDisable(ctx, "user-1001")
disableInfo, err := dtoken.GetDisableInfo(ctx, "user-1001")
_ = dtoken.Untie(ctx, "user-1001")

_, _ = disabled, disableInfo
```

## Advanced Features

### Token Introspection

```go
info, err := dtoken.IntrospectToken(ctx, token)
if err != nil {
	return err
}
if !info.Active {
	fmt.Println("invalid reason:", info.Reason)
}
```

### Refresh Token

```go
pair, err := dtoken.LoginWithRefreshToken(ctx, "user-1001")
if err != nil {
	return err
}

nextPair, err := dtoken.RefreshToken(ctx, pair.RefreshToken)
if err != nil {
	return err
}
_ = dtoken.RevokeRefreshToken(ctx, nextPair.RefreshToken)
```

### Temporary Ticket

```go
ticket, err := dtoken.CreateTicket(ctx, "user-1001")
if err != nil {
	return err
}

token, err := dtoken.ConsumeTicket(ctx, ticket)
if err != nil {
	return err
}
fmt.Println(token)
```

### Short-Key Login

```go
shortKey, err := dtoken.CreateShortKey(ctx, "user-1001")
if err != nil {
	return err
}

token, err := dtoken.ConsumeShortKey(ctx, shortKey)
if err != nil {
	return err
}
fmt.Println(token)
```

### SSO

```go
ssoTicket, err := dtoken.CreateSSOTicket(ctx, "user-1001")
if err != nil {
	return err
}

token, err := dtoken.ExchangeSSOTicket(ctx, ssoTicket)
if err != nil {
	return err
}

_ = dtoken.LogoutAllApps(ctx, "user-1001")
```

### Nonce Anti-Replay

```go
nonce, err := dtoken.GenerateNonce(ctx)
if err != nil {
	return err
}

ok, err := dtoken.VerifyAndConsumeNonce(ctx, nonce)
_, _ = ok, err
```

### Event Listener

```go
mgr.AddListener(func(event dtoken.Event) {
	fmt.Println(event.Type, event.LoginID, event.TokenValue)
})
```

## Configuration Example

```go
mgr, err := defaults.NewBuilder().
	AuthType("user").
	KeyPrefix("dtoken").
	TokenName("Authorization").
	Timeout(7200).
	ActiveTimeout(1800).
	AutoRenew(true).
	RenewMaxRefresh(3600).
	RenewInterval(60).
	IsConcurrent(true).
	IsShare(false).
	MaxLoginCount(5).
	IsReadHeader(true).
	IsReadCookie(false).
	IsReadBody(false).
	IsLog(false).
	IsPrintBanner(false).
	SetStorage(storage).
	Build()
```

| Option | Description |
| --- | --- |
| `AuthType` | Authentication system identifier for running multiple auth systems in parallel |
| `KeyPrefix` | Storage key prefix |
| `TokenName` | Token name, usually corresponding to a header or cookie name |
| `Timeout` | Absolute token expiration time, in seconds |
| `ActiveTimeout` | Maximum inactive duration, in seconds |
| `AutoRenew` | Whether to enable automatic renewal |
| `RenewMaxRefresh` | Renewal trigger threshold |
| `RenewInterval` | Minimum renewal interval |
| `IsConcurrent` | Whether to allow concurrent login for the same account |
| `IsShare` | Whether to share a token during concurrent login |
| `MaxLoginCount` | Maximum number of online terminals |
| `IsReadHeader` | Whether to read token from HTTP headers |
| `IsReadCookie` | Whether to read token from cookies |
| `IsReadBody` | Whether to read token from request body |
| `SetStorage` | Sets a custom storage adapter |

## Component Ecosystem

| Type | Implementations | Module |
| --- | --- | --- |
| Storage | Memory, Redis, PostgreSQL | `com/storage/*` |
| Codec | JSON, JSON v2, MessagePack, Base64 | `com/codec/*` |
| Logger | DLog, GoFrame, Nop | `com/log/*` |
| Token generator | UUID, JWT | `com/generator/dgenerator` |
| Goroutine pool | Ants | `com/pool/ants` |

## Project Structure

```text
dtoken-go/
├── com/              # Pluggable component implementations
├── core/             # Core interfaces, config, Manager, context, nonce, OAuth2
├── defaults/         # Default Builder and default component wiring
├── docs/             # Detailed documentation and image assets
├── dtoken/           # Global API facade
├── examples/         # Quick-start and framework integration examples
├── integrations/     # Web framework integration packages
└── go.work           # Go workspace
```

## Documentation And Examples

### Documentation

- [Documentation Center](docs/README.md)
- [Quick Start](docs/tutorial/quick-start.md)
- [Authentication](docs/guide/authentication.md)
- [Permission Management](docs/guide/permission.md)
- [Framework Integration](docs/guide/framework-integration.md)
- [Event Listener](docs/guide/listener.md)
- [Nonce Anti-Replay](docs/guide/nonce.md)
- [JWT Integration](docs/guide/jwt.md)
- [Redis Storage](docs/guide/redis-storage.md)
- [OAuth2](docs/guide/oauth2.md)
- [Refresh Token](docs/guide/refresh-token.md)
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

## License

DToken-Go is open source under the [Apache-2.0](LICENSE) license.
