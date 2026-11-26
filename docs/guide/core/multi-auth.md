# Multi-Auth Systems

Import aliases for code fragments: dt = github.com/Zany2/dtoken-go.

**[中文文档](../core/multi-auth_zh.md)**

Multi-auth systems let one project maintain multiple isolated authentication systems, such as end users, administrators, and open-platform clients. DToken-Go separates them with `AuthType`.

Build registers and returns each Auth automatically. Use the returned instance, dt.AuthType(name), or dt package functions for the default namespace. BuildInstance creates isolated instances without registration.

## Core Concept

`AuthType` participates in manager registration, Token lookup, Session lookup, permission and role resolution, and storage key generation. A Token can only be recognized by its own auth system.

```text
dtoken:user:v2:token:xxx
dtoken:admin:v2:token:yyy
dtoken:app:v2:session:client-001
```

`AuthType("user")` is normalized to `user:`. Builders, `dt.AuthType(name)`, and `dt.GetManager(name)` accept either `"user"` or `"user:"`. TokenName selects the HTTP field used to read tokens; KeyPrefix and AuthType determine the storage namespace.

Manager and bundled Nonce, OAuth2, Ticket, and ShortKey instances encode and retain their fixed storage namespace during construction. Key generation reuses it instead of repeatedly escaping KeyPrefix and AuthType. Mutating the supplied configuration or a GetConfig() snapshot does not switch a constructed Manager to another namespace; construct the appropriate instance for another system. This optimization preserves the existing key format and requires no additional migration.

## Build And Register Multiple Systems

<!-- dtoken-compile: multi-auth -->
```go
package main

import (
	"log"
	"context"

	dt "github.com/Zany2/dtoken-go"
	gindt "github.com/Zany2/dtoken-go/integrations/gin"
	"github.com/Zany2/dtoken-go/storage/memory"
	"github.com/gin-gonic/gin"
)

func main() {
	defer dt.CloseAll()
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
		log.Print(err)
		return
	}

	adminAuth, err := dt.NewBuilder().
		AuthType("admin").
		KeyPrefix("dtoken").
		TokenName("admin-token").
		Storage(storage).
		UseManagerOption(sharedOwnership).
		Build()
	if err != nil {
		log.Print(err)
		return
	}

	userToken, _ := dt.AuthType("user").LoginID(ctx, "10001")
	adminToken, _ := adminAuth.LoginID(ctx, "admin-1")
	_, _ = userToken, adminToken
	if err := registerRoutes(ctx, gin.New(), userAuth, adminAuth); err != nil {
		log.Print(err)
		return
	}
}
```

Managers own and close Storage, Logger, and Pool components configured through a Builder by default. `ComponentOwnership` is a complete policy: every field must be set explicitly because omitted boolean fields are `false`. For shared components, set the corresponding field to `false`; the caller then closes that shared component once during application shutdown. A caller-owned Pool must remain running until all managers that use it have closed, because each manager waits for its accepted asynchronous tasks during shutdown.

Call `Auth.Close()` (or advanced `Manager.CloseManager()`) from the application's external shutdown flow after stopping new business requests and waiting for in-flight synchronous calls. Do not call it synchronously inside a manager task or event listener: shutdown waits for accepted tasks and listeners, which can include the caller itself. A callback that needs shutdown should signal the application lifecycle controller and return. Business calls after shutdown are not supported; `CloseManager` is not a request-draining barrier.

Duplicate Build returns `errors.ErrManagerAlreadyRegistered` without replacing an instance. For explicit replacement, build with BuildInstance and call dt.SetAuth(auth).

## Call A Specific Auth System

Select the namespace, then call the same methods as the instance:

```go
user := dt.AuthType("user")
token, err := user.Login(ctx, dt.LoginOptions{LoginID: "10001", Device: "web", DeviceID: "chrome"})

loginID, err := user.GetLoginID(ctx, token)
err = user.CheckPermission(ctx, dt.PermissionOptions{LoginID: "10001", Permission: "user:read"})
roles, err := user.GetRoles(ctx, "10001")
```

Package calls such as dt.CheckLogin use the default `auth:` namespace. The example above builds only user and admin, with no default registration. Build another instance without AuthType to enable default calls.

| Entry point | Selection behavior |
| --- | --- |
| `dt.CheckLogin(ctx, token)` | Look up the default namespace auth: |
| `dt.AuthType("user").CheckLogin(ctx, token)` | Look up the current user engine on every operation |
| `userAuth.CheckLogin(ctx, token)` | Use the fixed engine returned by the builder |

For a missing system, CheckLogin returns ErrManagerNotFound and IsLogin returns false, without falling back. Selectors can be cached and follow explicit replacement; fixed Auth instances do not. Operation options no longer contain AuthType; the entry points above select the engine.

## Unregistered Instances And Explicit Replacement

BuildInstance returns (*Auth, error) without accessing the registry. Use it for tests, dependency injection, or independent engines with the same name, and close each through its owner. Unregistered engines with the same name do not automatically isolate data in a shared backend: configure AuthType, KeyPrefix, or separate storage appropriately.

To replace a registered system, first stop new requests to the old engine and drain its active requests, then execute:

```go
func replaceUserAuth() error {
    replacement, err := dt.NewBuilder().AuthType("user").BuildInstance()
    if err != nil {
        return err
    }
    if err := dt.SetAuth(replacement); err != nil {
        replacement.Close()
        return err
    }
    return nil
}
```

SetAuth registers the new engine and closes the old one. Existing fixed Auth instances and WithAuth bindings still refer to the old engine and cannot continue using it. CloseAll closes current registrations, excludes unregistered instances, and allows later builds. See [API and lifecycle](../../design/api-lifecycle.md) for the complete contract.

## Use With Framework Middleware

Different route groups can bind different auth systems:

<!-- dtoken-compile: multi-auth -->
```go
func registerRoutes(ctx context.Context, r *gin.Engine, userAuth, adminAuth *dt.Auth) error {
	userGroup := r.Group("/api")
	authOption1, authErr := gindt.WithAuth(userAuth)
	if authErr != nil {
	    return authErr
	}
	userGroup.Use(gindt.AuthMiddleware(ctx, authOption1))

	adminGroup := r.Group("/admin")
	authOption2, authErr := gindt.WithAuth(adminAuth)
	if authErr != nil {
	    return authErr
	}
	adminGroup.Use(gindt.AuthMiddleware(ctx, authOption2))
	adminGroup.Use(gindt.RoleMiddleware(ctx, []string{"admin"}, authOption2))
	return nil
}
```

With this setup, `/api` only recognizes Tokens from `user`, and `/admin` only recognizes Tokens from `admin`.

Each group binds its instance through `WithAuth`, independently of registry registration. For dynamic registry lookup, explicitly configure `WithManagerResolver(dt.GetManager)` and select a namespace with `WithAuthType`.

## Redis Key Isolation

Redis keys include both `KeyPrefix` and `AuthType`. If multiple auth systems share Redis, core login-state data remains isolated as long as `AuthType` differs.

```text
dtoken:user:v2:token:...
dtoken:user:v2:session:...
dtoken:admin:v2:token:...
dtoken:admin:v2:session:...
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
eventManager, _ := dt.GetEventManager("admin")
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
