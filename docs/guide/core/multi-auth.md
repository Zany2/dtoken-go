# Multi-Auth Systems

Import aliases for code fragments: dt = github.com/Zany2/dtoken-go, global = github.com/Zany2/dtoken-go/global.

**[中文文档](../core/multi-auth_zh.md)**

Multi-auth systems let one project maintain multiple isolated authentication systems, such as end users, administrators, and open-platform clients. DToken-Go separates them with `AuthType`.

Multiple instances do not require global registration: build userAuth and adminAuth and inject each through WithAuth. The following examples additionally enable the optional registry.

## Core Concept

`AuthType` participates in manager registration, Token lookup, Session lookup, permission and role resolution, and storage key generation. A Token can only be recognized by its own auth system.

```text
dtoken:user:token:xxx
dtoken:admin:token:yyy
dtoken:app:session:client-001
```

`AuthType("user")` is normalized to `user:`. Global APIs accept either `"user"` or `"user:"`.

## Register Multiple Managers

<!-- dtoken-compile: multi-auth -->
```go
package main

import (
	"context"

	dt "github.com/Zany2/dtoken-go"
	global "github.com/Zany2/dtoken-go/global"
	gindt "github.com/Zany2/dtoken-go/integrations/gin"
	"github.com/Zany2/dtoken-go/storage/memory"
	"github.com/gin-gonic/gin"
)

func main() {
	defer global.DeleteAllManager()
	ctx := context.Background()
	storage := memory.NewStorage()
	sharedOwnership := dt.WithComponentOwnership(dt.ComponentOwnership{
		Storage: false,
		Logger:  true,
		Pool:    true,
	})

	userAuth, err := dt.NewBuilder().
		AuthType("user").
		KeyPrefix("dtoken").
		TokenName("user-token").
		Storage(storage).
		UseManagerOption(sharedOwnership).
		Build()
	if err != nil {
		panic(err)
	}
	global.SetAuth(userAuth)

	adminAuth, err := dt.NewBuilder().
		AuthType("admin").
		KeyPrefix("dtoken").
		TokenName("admin-token").
		Storage(storage).
		UseManagerOption(sharedOwnership).
		Build()
	if err != nil {
		panic(err)
	}
	global.SetAuth(adminAuth)

	userToken, _ := global.Login(ctx, "10001", "", "", "user")
	adminToken, _ := global.Login(ctx, "admin-1", "", "", "admin")
	_, _ = userToken, adminToken
	registerRoutes(ctx, gin.New(), userAuth, adminAuth)
}
```

Managers own and close Storage, Logger, and Pool components configured through a Builder by default. `ComponentOwnership` is a complete policy: every field must be set explicitly because omitted boolean fields are `false`. For shared components, set the corresponding field to `false`; the caller then closes that shared component once during application shutdown. A caller-owned Pool must remain running until all managers that use it have closed, because each manager waits for its accepted asynchronous tasks during shutdown.

Call `Auth.Close()` (or advanced `Manager.CloseManager()`) from the application's external shutdown flow after stopping new business requests and waiting for in-flight synchronous calls. Do not call it synchronously inside a manager task or event listener: shutdown waits for accepted tasks and listeners, which can include the caller itself. A callback that needs shutdown should signal the application lifecycle controller and return. Business calls after shutdown are not supported; `CloseManager` is not a request-draining barrier.

`BuildAndSetManager` can also override and register `AuthType` during construction:

```go
_, err := global.BuildAndSetManager(
    dt.NewBuilder().
        KeyPrefix("dtoken").
        Storage(storage).
        UseManagerOption(sharedOwnership),
    "admin",
)
```

## Call A Specific Auth System

Most global APIs accept an optional final `authType` argument:

```go
token, err := global.Login(ctx, "10001", "web", "chrome", "user")

loginID, err := global.GetLoginID(ctx, token, "user")
err = global.CheckPermission(ctx, "10001", "user:read", "user")
roles, err := global.GetRoles(ctx, "10001", "user")
```

If `authType` is omitted, the default auth system `auth:` is used.

## Use With Framework Middleware

Different route groups can bind different auth systems:

<!-- dtoken-compile: multi-auth -->
```go
func registerRoutes(ctx context.Context, r *gin.Engine, userAuth, adminAuth *dt.Auth) {
	userGroup := r.Group("/api")
	userGroup.Use(gindt.AuthMiddleware(ctx, gindt.WithAuth(userAuth)))

	adminGroup := r.Group("/admin")
	adminGroup.Use(gindt.AuthMiddleware(ctx, gindt.WithAuth(adminAuth)))
	adminGroup.Use(gindt.RoleMiddleware(ctx, []string{"admin"}, gindt.WithAuth(adminAuth)))
}
```

With this setup, `/api` only recognizes Tokens from `user`, and `/admin` only recognizes Tokens from `admin`.

Each group binds its instance through `WithAuth`, independently of global registration. For dynamic registry lookup, explicitly configure `WithManagerResolver(global.GetManager)` and select a namespace with `WithAuthType`.

## Redis Key Isolation

Redis keys include both `KeyPrefix` and `AuthType`. If multiple auth systems share Redis, core login-state data remains isolated as long as `AuthType` differs.

```text
dtoken:user:token:...
dtoken:user:session:...
dtoken:admin:token:...
dtoken:admin:session:...
```

If projects, environments, or test runs need stronger isolation, separate `KeyPrefix` too:

```go
dt.NewBuilder().
    AuthType("admin").
    KeyPrefix("dtoken:prod")
```

## Events And Permissions

Event data carries `AuthType`, which is useful for audit logs:

```go
eventManager, _ := global.GetEventManager("admin")
```

`AccessProvider` receives `AuthType` through `AccessSubject`, so one permission service can return different permissions per auth system.

## Suggestions

- Use separate `AuthType` values for end users and administrators.
- Multi-tenancy does not always require `AuthType`; if tenants share one auth rule set, tenant IDs usually belong in business permissions or account data.
- Multiple auth systems can share Redis, but production deployments should use clear `KeyPrefix` values.
- Avoid reusing the same `TokenName` across auth systems unless routes are clearly separated.

## Related Documentation

- [Configuration Guide](../reference/configuration.md)
- [Redis Storage](../integration/redis-storage.md)
- [Framework Integration](../integration/framework-integration.md)
- [AccessProvider](../core/access-provider.md)
