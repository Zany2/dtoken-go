<p align="center">
  <img src="docs/assets/logo.png" alt="DToken-Go" width="180">
</p>

<h1 align="center">DToken-Go</h1>

<p align="center">
  Authentication, authorization, and session management for Go applications.
</p>

<p align="center">
  <a href="README.md">中文</a> | English
</p>

---

## Overview

DToken-Go is a modular authentication and authorization framework for Go. It provides token-based login, session management, role and permission checks, account disabling, nonce anti-replay utilities, OAuth2 server capabilities, and integrations for popular Go web frameworks.

The project is split into focused modules:

- `dtoken` provides the global facade API for application code.
- `defaults` provides an out-of-the-box Builder with memory storage, JSON codec, default token generator, and default logging components.
- `core` contains the core interfaces, configuration, manager, context, events, nonce, and OAuth2 implementation.
- `com` contains pluggable component implementations such as storage, codecs, logs, token generators, and goroutine pools.
- `integrations` contains middleware and request-context adapters for Gin, Echo, Fiber, Chi, GoFrame, Hertz, and Kratos.

## Features

- Token authentication: login, logout, token checks, auto-renewal, and multi-terminal management.
- Session management: session storage, terminal tracking, and online terminal statistics.
- Permission management: add, remove, query, AND/OR checks, token-based checks, and custom permission callbacks.
- Role management: add, remove, query, AND/OR checks, token-based checks, and custom role callbacks.
- Account disabling: timed disabling, untying, disable reasons, and remaining TTL queries.
- Online state controls: kickout and replace by token, account, device, or device ID.
- Nonce utilities: one-time nonce generation, verification, and consumption for anti-replay scenarios.
- OAuth2 server: authorization code, client credentials, password grant, refresh token, validation, and revocation.
- Multiple auth systems: isolate multiple authentication systems with `authType`.
- Pluggable architecture: replace storage, codec, logger, token generator, and goroutine pool.
- Framework integrations: middleware, annotation-style checks, and request-context adapters.

## Installation

Core usage:

```bash
go get github.com/Zany2/dtoken-go/defaults
go get github.com/Zany2/dtoken-go/dtoken
```

Optional storage components:

```bash
go get github.com/Zany2/dtoken-go/com/storage/redis
go get github.com/Zany2/dtoken-go/com/storage/postgresql
```

Optional framework integrations:

```bash
go get github.com/Zany2/dtoken-go/integrations/gin
go get github.com/Zany2/dtoken-go/integrations/echo
go get github.com/Zany2/dtoken-go/integrations/fiber
go get github.com/Zany2/dtoken-go/integrations/chi
go get github.com/Zany2/dtoken-go/integrations/gf
go get github.com/Zany2/dtoken-go/integrations/hertz
go get github.com/Zany2/dtoken-go/integrations/kratos
```

## Quick Start

`defaults.NewBuilder()` already wires memory storage by default, so the minimal example does not require Redis or a database.

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
	_ = dtoken.AddPermissions(ctx, "user-1001", []string{"article:read"})

	loginID, _ := dtoken.GetLoginID(ctx, token)
	hasRole := dtoken.HasRole(ctx, loginID, "admin")
	hasPermission := dtoken.HasPermission(ctx, loginID, "article:read")

	fmt.Println(token, loginID, hasRole, hasPermission)
	_ = dtoken.Logout(ctx, token)
}
```

See [examples/quick_start](examples/quick_start/) for a complete quick-start example.

## Gin Integration Example

For framework examples, prefer importing the matching `integrations/*` package as the DToken entrypoint. Here is a Gin example:

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
		_ = gindt.AddRoles(c.Request.Context(), "user-1001", []string{"admin"})
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

More framework examples:

| Framework | Example | Integration package |
| --- | --- | --- |
| Gin | [examples/gin](examples/gin/) | `github.com/Zany2/dtoken-go/integrations/gin` |
| Echo | [examples/echo](examples/echo/) | `github.com/Zany2/dtoken-go/integrations/echo` |
| Fiber | [examples/fiber](examples/fiber/) | `github.com/Zany2/dtoken-go/integrations/fiber` |
| Chi | [examples/chi](examples/chi/) | `github.com/Zany2/dtoken-go/integrations/chi` |
| GoFrame | [examples/gf](examples/gf/) | `github.com/Zany2/dtoken-go/integrations/gf` |
| Hertz | [examples/hertz](examples/hertz/) | `github.com/Zany2/dtoken-go/integrations/hertz` |
| Kratos | [examples/kratos](examples/kratos/) | `github.com/Zany2/dtoken-go/integrations/kratos` |

## Common APIs

### Login And Token

| API | Description |
| --- | --- |
| `Login(ctx, loginID, params...)` | Logs in and returns a token. `params` can include `device`, `deviceId`, and `authType` |
| `LoginByToken(ctx, tokenValue)` | Continues login with an existing token |
| `Logout(ctx, tokenValue)` | Logs out by token |
| `LogoutByLoginID(ctx, loginID)` | Logs out all terminals for an account |
| `IsLogin(ctx, tokenValue)` | Checks whether a token is logged in |
| `CheckLogin(ctx, tokenValue)` | Validates login state and returns an error on failure |
| `GetLoginID(ctx, tokenValue)` | Gets login ID from token |
| `GetTokenInfo(ctx, tokenValue)` | Gets token details |
| `GetTokenTTL(ctx, tokenValue)` | Gets remaining token TTL |

### Permissions And Roles

| API | Description |
| --- | --- |
| `AddPermissions(ctx, loginID, permissions)` | Adds permissions |
| `RemovePermissions(ctx, loginID, permissions)` | Removes permissions |
| `GetPermissions(ctx, loginID)` | Gets permission list |
| `HasPermission(ctx, loginID, permission)` | Checks one permission |
| `HasPermissionsAnd(ctx, loginID, permissions)` | Checks all permissions |
| `HasPermissionsOr(ctx, loginID, permissions)` | Checks any permission |
| `AddRoles(ctx, loginID, roles)` | Adds roles |
| `RemoveRoles(ctx, loginID, roles)` | Removes roles |
| `GetRoles(ctx, loginID)` | Gets role list |
| `HasRole(ctx, loginID, role)` | Checks one role |
| `HasRolesAnd(ctx, loginID, roles)` | Checks all roles |
| `HasRolesOr(ctx, loginID, roles)` | Checks any role |

### Online State And Disabling

| API | Description |
| --- | --- |
| `Kickout(ctx, tokenValue)` | Kicks out a token |
| `KickoutByLoginID(ctx, loginID)` | Kicks out all terminals for an account |
| `Replace(ctx, tokenValue)` | Replaces a token |
| `ReplaceByLoginID(ctx, loginID)` | Replaces all terminals for an account |
| `Disable(ctx, loginID, duration, reason...)` | Disables an account |
| `Untie(ctx, loginID)` | Unblocks an account |
| `IsDisable(ctx, loginID)` | Checks whether an account is disabled |
| `GetDisableInfo(ctx, loginID)` | Gets disable details |

### Nonce And OAuth2

| API | Description |
| --- | --- |
| `GenerateNonce(ctx)` | Generates a one-time nonce |
| `VerifyNonce(ctx, nonce)` | Verifies a nonce |
| `VerifyAndConsumeNonce(ctx, nonce)` | Verifies and consumes a nonce |
| `RegisterOAuth2Client(client)` | Registers an OAuth2 client |
| `GenerateOAuth2AuthorizationCode(...)` | Generates an authorization code |
| `ExchangeOAuth2CodeForToken(...)` | Exchanges an authorization code for tokens |
| `OAuth2ClientCredentialsToken(...)` | Gets a token with client credentials grant |
| `OAuth2PasswordGrantToken(...)` | Gets a token with password grant |
| `RefreshOAuth2AccessToken(...)` | Refreshes an access token |
| `ValidateOAuth2AccessToken(ctx, accessToken)` | Validates an access token |
| `RevokeOAuth2Token(ctx, accessToken)` | Revokes an access token |

## Builder Configuration

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

Common options:

| Option | Description |
| --- | --- |
| `AuthType` | Authentication system ID for multiple isolated auth systems |
| `KeyPrefix` | Storage key prefix |
| `TokenName` | Token name, usually a header or cookie name |
| `Timeout` | Absolute token timeout in seconds |
| `ActiveTimeout` | Maximum inactive duration in seconds |
| `AutoRenew` | Enables or disables auto-renewal |
| `RenewMaxRefresh` | Renewal trigger threshold |
| `RenewInterval` | Minimum renewal interval |
| `IsConcurrent` | Allows or disallows concurrent login for the same account |
| `IsShare` | Shares a token for concurrent logins |
| `MaxLoginCount` | Maximum online terminal count |
| `IsReadHeader` | Reads token from HTTP headers |
| `IsReadCookie` | Reads token from cookies |
| `IsReadBody` | Reads token from request body |
| `SetStorage` | Sets a custom storage adapter |

## Pluggable Components

| Type | Implementations | Module |
| --- | --- | --- |
| Storage | Memory, Redis, PostgreSQL | `com/storage/*` |
| Codec | JSON, JSON v2, MessagePack, Base64 | `com/codec/*` |
| Log | DLog, GoFrame, Nop | `com/log/*` |
| Token generator | UUID, JWT | `com/generator/dgenerator` |
| Goroutine pool | Ants | `com/pool/ants` |

## Project Structure

```text
dtoken-go/
├── com/              # Pluggable component implementations
├── core/             # Core interfaces, config, manager, context, nonce, OAuth2
├── defaults/         # Default Builder and default component wiring
├── docs/             # Documentation and image assets
├── dtoken/           # Global API facade
├── examples/         # Quick-start and framework integration examples
└── integrations/     # Web framework integration packages
```

## Documentation

- [Documentation Center](docs/README.md)
- [Quick Start](docs/tutorial/quick-start.md)
- [Authentication](docs/guide/authentication.md)
- [Permission Management](docs/guide/permission.md)
- [Framework Integration](docs/guide/framework-integration.md)
- [Redis Storage](docs/guide/redis-storage.md)
- [OAuth2](docs/guide/oauth2.md)
- [API Reference](docs/api/dtoken.md)

## License

MIT
