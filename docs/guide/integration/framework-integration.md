# Framework Integration Usage Guide

**[中文文档](../integration/framework-integration_zh.md)**

## Overview

DToken-Go keeps core authentication APIs and framework integrations separate. Use `dtoken` for login, logout, session, permission, role, and global manager APIs. Use `defaults` to create managers with bundled default components. Use `integrations/*` packages only for framework middleware, annotations, and request-context helpers.

Context registration and authentication middleware across frameworks select managers in this order: explicit `WithManager`, explicit `WithAuthType`, the existing request manager, then the global default. Repeated registration without an explicit selection preserves the request manager. A missing request context, including a typed nil, permits fallback. An existing request context with a nil or closed manager returns `ErrManagerNotFound` instead of implicitly switching to the global instance. Explicit selection can still override an existing request scope.

## Installation

```bash
go get github.com/Zany2/dtoken-go/defaults
go get github.com/Zany2/dtoken-go/integrations/gin
go get github.com/Zany2/dtoken-go/com/storage/memory
```

Other supported integration packages follow the same pattern:

```bash
go get github.com/Zany2/dtoken-go/integrations/echo
go get github.com/Zany2/dtoken-go/integrations/fiber
go get github.com/Zany2/dtoken-go/integrations/chi
go get github.com/Zany2/dtoken-go/integrations/gf
go get github.com/Zany2/dtoken-go/integrations/hertz
go get github.com/Zany2/dtoken-go/integrations/kratos
go get github.com/Zany2/dtoken-go/integrations/beego
```

## Minimal Framework Examples

All framework packages re-export common `dtoken` facade APIs and provide framework-specific middleware. The following snippets show the smallest login plus protected route setup. Full runnable examples live in `examples/gf`, `examples/gin`, `examples/echo`, and `examples/fiber`.

### GoFrame

```go
ctx := context.Background()
mgr, err := gfdt.NewBuilder().IsPrintBanner(false).Build()
if err != nil {
    panic(err)
}
gfdt.SetManager(mgr)

s := g.Server()
s.Use(gfdt.RegisterDTokenContextMiddleware(ctx))
s.Group("/", func(group *ghttp.RouterGroup) {
    group.POST("/login", func(r *ghttp.Request) {
        token, _ := gfdt.Login(r.Context(), "1000")
        r.Response.WriteJson(g.Map{"token": token})
    })
    group.Group("/", func(group *ghttp.RouterGroup) {
        group.Middleware(gfdt.AuthMiddleware(ctx))
        group.GET("/me", func(r *ghttp.Request) {
            dCtx, _ := gfdt.GetDTokenContext(r)
            loginID, _ := dCtx.Auth().GetLoginID(r.Context())
            r.Response.WriteJson(g.Map{"loginId": loginID})
        })
    })
})
```

### Gin

Gin authentication middleware, annotation checks, and authentication hooks use the current `c.Request.Context()`, preserving request values, deadlines, and cancellation. The registration functions retain their `ctx` parameter for source compatibility.

If a hook replaces `c.Request`, subsequent authentication checks use the replacement request context. Hook calls to `req.Next()` and `req.Exit()` honor the first decision; repeated calls or a later opposite choice do not advance or abort the chain again.

Query and body token lookup also reads the current request instead of credentials cached by Gin before replacement. Body tokens come only from form fields, and multipart parsing continues to honor Gin's `MaxMultipartMemory` setting.

After `RegisterDTokenContextMiddleware(ctx, WithManager(mgr))` injects a manager, authentication middleware without an explicit manager or `AuthType` reuses that request instance. An explicit `WithManager` takes precedence, followed by the registry entry selected by `WithAuthType`; the global default is used only when no request manager is cached. The `...ByContext` helpers also use the request manager and reject closed instances.

Authentication failures abort the Gin handler chain before invoking the failure callback, so calling `c.Next()` from that callback cannot run business handlers. If a before-auth or route-access hook calls `c.Abort()`, authentication processing stops immediately.

```go
ctx := context.Background()
mgr, err := gindt.NewBuilder().IsPrintBanner(false).Build()
if err != nil {
    panic(err)
}
gindt.SetManager(mgr)

r := gin.Default()
r.Use(gindt.RegisterDTokenContextMiddleware(ctx))
r.POST("/login", func(c *gin.Context) {
    token, _ := gindt.Login(c.Request.Context(), "1000")
    c.JSON(http.StatusOK, gin.H{"token": token})
})

auth := r.Group("/")
auth.Use(gindt.AuthMiddleware(ctx))
auth.GET("/me", func(c *gin.Context) {
    dCtx, _ := gindt.GetDTokenContext(c)
    loginID, _ := dCtx.Auth().GetLoginID(c.Request.Context())
    c.JSON(http.StatusOK, gin.H{"loginId": loginID})
})
```

### Echo

```go
ctx := context.Background()
mgr, err := echodt.NewBuilder().IsPrintBanner(false).Build()
if err != nil {
    panic(err)
}
echodt.SetManager(mgr)

e := echo4.New()
e.Use(echodt.RegisterDTokenContextMiddleware(ctx))
e.POST("/login", func(c echo4.Context) error {
    token, _ := echodt.Login(c.Request().Context(), "1000")
    return c.JSON(http.StatusOK, echo4.Map{"token": token})
})

auth := e.Group("")
auth.Use(echodt.AuthMiddleware(ctx))
auth.GET("/me", func(c echo4.Context) error {
    dCtx, _ := echodt.GetDTokenContext(c)
    loginID, _ := dCtx.Auth().GetLoginID(c.Request().Context())
    return c.JSON(http.StatusOK, echo4.Map{"loginId": loginID})
})
```

### Fiber

```go
ctx := context.Background()
mgr, err := fiberdt.NewBuilder().IsPrintBanner(false).Build()
if err != nil {
    panic(err)
}
fiberdt.SetManager(mgr)

app := gofiber.New()
app.Use(fiberdt.RegisterDTokenContextMiddleware(ctx))
app.Post("/login", func(c *gofiber.Ctx) error {
    token, _ := fiberdt.Login(c.UserContext(), "1000")
    return c.JSON(gofiber.Map{"token": token})
})

auth := app.Group("")
auth.Use(fiberdt.AuthMiddleware(ctx))
auth.Get("/me", func(c *gofiber.Ctx) error {
    dCtx, _ := fiberdt.GetDTokenContext(c)
    loginID, _ := dCtx.Auth().GetLoginID(c.UserContext())
    return c.JSON(gofiber.Map{"loginId": loginID})
})
```
## Gin Example

```go
package main

import (
    "context"
    "net/http"

    "github.com/Zany2/dtoken-go/com/storage/memory"
    "github.com/Zany2/dtoken-go/defaults"
    "github.com/Zany2/dtoken-go/dtoken"
    gindt "github.com/Zany2/dtoken-go/integrations/gin"
    "github.com/gin-gonic/gin"
)

func main() {
    ctx := context.Background()
    storage := memory.NewStorage()

    mgr, err := defaults.NewBuilder().
        SetStorage(storage).
        TokenName("token").
        Timeout(2 * 60 * 60).
        Build()
    if err != nil {
        panic(err)
    }
    dtoken.SetManager(mgr)

    r := gin.Default()
    r.Use(gindt.RegisterDTokenContextMiddleware(ctx))

    r.POST("/login", func(c *gin.Context) {
        token, err := dtoken.Login(ctx, "1000")
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        c.JSON(http.StatusOK, gin.H{"token": token})
    })

    user := r.Group("/user")
    user.Use(gindt.AuthMiddleware(ctx))
    user.GET("/info", func(c *gin.Context) {
        dCtx, _ := gindt.GetDTokenContext(c)
        token := dCtx.GetTokenValue()
        loginID, _ := dtoken.GetLoginID(ctx, token)
        c.JSON(http.StatusOK, gin.H{"loginId": loginID})
    })

    _ = r.Run(":8080")
}
```

## Common API Surface

Core authentication APIs come from `dtoken`:

```go
dtoken.SetManager(mgr)
mgr, err := dtoken.GetManager()

token, err := dtoken.Login(ctx, "1000")
ok := dtoken.IsLogin(ctx, token)
loginID, err := dtoken.GetLoginID(ctx, token)
err = dtoken.Logout(ctx, token)

err = dtoken.AddPermissions(ctx, "1000", []string{"user:read"})
hasPermission := dtoken.HasPermission(ctx, "1000", "user:read")

err = dtoken.AddRoles(ctx, "1000", []string{"admin"})
hasRole := dtoken.HasRole(ctx, "1000", "admin")
```

Integration packages expose framework middleware:

```go
r.Use(gindt.RegisterDTokenContextMiddleware(ctx))
r.Use(gindt.AuthMiddleware(ctx))
r.Use(gindt.PermissionMiddleware(ctx, []string{"user:read"}))
r.Use(gindt.RoleMiddleware(ctx, []string{"admin"}))
```

## Middleware Options

Each framework integration package provides similar middleware options for auth system selection, access logic, and failure responses.

### Custom Failure Response

The default failure response is useful for quick validation. Production projects usually define a unified business error shape with `WithFailFunc`:

```go
failFunc := gindt.WithFailFunc(func(c *gin.Context, err error) {
    c.JSON(http.StatusUnauthorized, gin.H{
        "code":    401,
        "message": err.Error(),
    })
})

r.Use(gindt.AuthMiddleware(ctx, failFunc))
```

Permission and role middleware can use the same failure handler:

```go
r.GET("/admin",
    gindt.RoleMiddleware(ctx, []string{"admin"}, failFunc),
    adminHandler,
)
```

### Select Auth System

In multi-auth scenarios, `WithAuthType` lets different route groups use different managers:

```go
userGroup := r.Group("/api")
userGroup.Use(gindt.AuthMiddleware(ctx, gindt.WithAuthType("user")))

adminGroup := r.Group("/admin")
adminGroup.Use(gindt.AuthMiddleware(ctx, gindt.WithAuthType("admin")))
adminGroup.Use(gindt.PermissionMiddleware(ctx, []string{"admin:read"}, gindt.WithAuthType("admin")))
```

For applications that do not use the global manager registry, inject a manager directly with `WithManager`. The explicit manager takes precedence over `WithAuthType` and works with context, auth, access, permission, and role middleware:

```go
auth.Use(gindt.RegisterDTokenContextMiddleware(ctx, gindt.WithManager(mgr)))
auth.Use(gindt.AuthMiddleware(ctx, gindt.WithManager(mgr)))
```

Annotation handlers with an empty `AuthType` reuse the manager already stored in the request context. An explicit annotation `AuthType` continues to resolve through the global registry for multi-auth routes.

### Permission And Role Logic

Permission and role middleware support different logic modes:

```go
r.GET("/reports",
    gindt.PermissionMiddleware(
        ctx,
        []string{"report:read", "report:export"},
        gindt.WithLogicType(gindt.LogicAnd),
    ),
    reportHandler,
)

r.GET("/console",
    gindt.RoleMiddleware(
        ctx,
        []string{"admin", "operator"},
        gindt.WithLogicType(gindt.LogicOr),
    ),
    consoleHandler,
)
```

`LogicAnd` requires every item, while `LogicOr` requires any item.

## Chi Request Handling

After `RegisterDTokenContextMiddleware(WithManager(mgr))`, Chi authentication middleware without an explicit manager or auth type inherits that request manager. `WithManager` takes precedence; an explicit `WithAuthType` selects a manager from the registry. Authentication middleware and annotation handlers bind their context to the current request and response writer, including changes made by earlier middleware. Install middleware that replaces requests or wraps response writers before the DToken middleware used by your handlers.

`GetPostForm` reads body fields only, so disabling `IsReadQuery` also excludes query-only tokens when `IsReadBody` is enabled. `GetClientIP` reads `RemoteAddr` and does not trust `X-Real-IP` or `X-Forwarded-For` directly. Behind a proxy, use middleware that validates trusted proxy addresses and normalizes `RemoteAddr` before DToken runs.

In Chi authentication hooks, `req.Next()` continues downstream at most once, and `req.Exit()` stops processing after custom handling. Calling `Abort()` on the registered request adapter also stops the DToken chain. An explicitly nil annotation returns a bad-request response.

## Echo Request Handling

Echo authentication middleware, annotation handlers, and hooks use `c.Request().Context()` for request values, deadlines, and cancellation. The `ctx` argument accepted when registering handlers remains for API compatibility. After registration with `WithManager(mgr)`, authentication without an explicit manager or auth type inherits the request manager. `WithManager` takes precedence over `WithAuthType`, and an explicit auth type selects a manager from the registry. Context facade operations, including renewal, use the request manager.

Token extraction reads the current request URL after `SetRequest` and reads form fields only from the body. Disabling `IsReadQuery` excludes query-only tokens even when body reading is enabled. In hooks, `req.Next()` invokes downstream at most once and preserves its returned error; `req.Exit()` or `Abort()` on the registered request adapter stops DToken processing. An explicitly nil annotation produces an invalid-parameter failure.

Client IP extraction follows Echo's `IPExtractor`. Configure `e.IPExtractor = echo.ExtractIPDirect()` for direct connections, or configure a proxy extractor with the appropriate trust rules. Echo's unset legacy policy trusts forwarding headers, so configure it explicitly when client IP is used for security decisions.

## Fiber Request Handling

Fiber authentication, annotations, hooks, and context facades use `c.UserContext()`. Set a standard context with `c.SetUserContext(ctx)` in upstream middleware to propagate request values, deadlines, and cancellation. Without one, Fiber supplies `context.Background()`; client disconnects do not automatically cancel it. The registration-time `ctx` parameter remains for API compatibility. `Locals` remains accessible through the request adapter and is not merged into `UserContext`.

Authentication inherits the registered request manager unless `WithManager` or `WithAuthType` explicitly selects another one; `WithManager` has precedence. Renewal through the context facade uses that same request manager. Header, query, cookie, and form token values are copied so asynchronous maintenance does not retain Fiber's reusable request buffers. Body token extraction excludes query parameters, including for multipart forms. `GetBody` returns a copy of the raw body without automatic decompression or replacement with decoding error text.

Use `req.Next()` in a before-auth hook to continue once and preserve downstream errors, or `req.Exit()` to stop after custom handling. `Abort()` on the registered request adapter is honored by DToken entry points and hooks. Fiber has no native chain-wide abort primitive: failure callbacks must only write the response and must not call `c.Next()`. An explicitly nil annotation invokes the invalid-parameter failure path. Client IP and proxy scheme handling continue to follow Fiber's configured proxy trust policy.

## GoFrame Request Handling

GoFrame authentication middleware, annotation handlers, and hooks use the current `r.Context()`, preserving request values, deadlines, and cancellation. The registration-time `ctx` parameter remains for API compatibility. After `RegisterDTokenContextMiddleware(ctx, WithManager(mgr))`, checks without an explicit manager or auth type inherit the request manager. Explicit `WithManager` takes precedence, followed by `WithAuthType`; the global default is used when no request manager is available. Context facades, including renewal, use the request manager and reject nil or closed cached managers.

Authentication failures mark the entire GoFrame chain as exited with `ExitAll()` before invoking the failure callback. Calling `r.Middleware.Next()` from that callback cannot resume protected handlers. Default failures return JSON with the corresponding HTTP status. Hook `req.Next()` continues at most once; `req.Exit()` exits the chain. Calling `Abort()` on the registered adapter stops DToken processing, but does not itself interrupt the current Go function: return after writing your response and do not invoke the native `r.Middleware.Next()` afterward. An explicitly nil annotation is rejected as an invalid parameter.

Response headers are written to `r.Response`, and setting a status code does not append status text to the body. `GetBody()` retains GoFrame's repeatable reads and returns framework body-read failures as errors. Body token lookup retains GoFrame `GetForm` semantics, including supported JSON/XML bodies, without falling back to query parameters. `GetClientIP()` uses the connection's `RemoteAddr` because GoFrame's `GetClientIp()` trusts forwarding headers without verifying proxy addresses. Behind a proxy, validate trusted proxy addresses and normalize `RemoteAddr` in upstream middleware before DToken runs.

## Hertz Request Handling

Hertz authentication checks, annotations, and hooks use the standard `context.Context` passed to the current handler, preserving its values, deadline, and cancellation. The registration-time `ctx` argument remains for API compatibility. DToken middleware and annotations also bind that standard context for downstream `...ByContext` facades and restore the outer binding when a nested call returns. Register DToken middleware after middleware that derives the standard context. If a later handler derives a new context without entering another DToken middleware or annotation, pass it explicitly to `dCtx.Auth()` or another context facade group. Without a DToken binding, `...ByContext` retains its `context.Background()` fallback. Hertz request keys are not automatically merged into the standard context, and this binding does not create client-disconnect cancellation.

After `RegisterDTokenContextMiddleware(ctx, WithManager(mgr))`, authentication without an explicit selection inherits the request manager. Explicit `WithManager` takes precedence over `WithAuthType`; the global default is used when no request manager is available. Context facades reject nil or closed cached managers. Authentication failures call `Abort()` before invoking failure callbacks, so a callback's `Next()` cannot enter protected handlers. Hook `req.Next()` continues at most once, and native or adapter `Abort()` stops DToken checks. An explicitly nil annotation returns an invalid-parameter failure.

Cookie reads decode the escaping performed by Hertz's cookie writer, allowing custom tokens containing `+`, `/`, or `%` to round-trip. Body token lookup uses form fields only, including multipart fields, without falling back to Query. `IsTLS()` checks the actual connection's Hertz TLS interface; a URL scheme or forwarding header does not prove that the connection is encrypted.

Client IP extraction follows Hertz's configured `ClientIP` function. Its default configuration trusts all proxy addresses. For direct connections, use `h.SetClientIPFunc(app.ClientIPWithOption(app.ClientIPOptions{}))`; behind proxies, configure `TrustedCIDRs` and `RemoteIPHeaders` explicitly.

## Kratos Request Handling

Kratos HTTP and unary gRPC authentication middleware, annotations, hooks, and context facades use the current handler's `context.Context`, preserving values, deadlines, and cancellation. After `RegisterDTokenContextMiddleware(WithManager(mgr))`, checks without an explicit selection inherit the request manager. Explicit `WithManager` takes precedence over `WithAuthType`, followed by the cached manager and then the global default. Context facades reject nil or closed cached managers; renewal stays within the selected manager. Changing the manager preserves the request adapter's values and abort state.

Hook `req.Next()` invokes the downstream handler at most once and preserves its result and error. `req.Exit()` stops after custom handling. DToken entries and hooks honor `Abort()` on the registered adapter. Failure dispatch marks that adapter aborted before calling `FailFunc`; the callback's returned error is returned to Kratos, while returning nil means the callback has handled the response. The protected handler is not invoked in either case. Default failures retain their original cause and map to Kratos HTTP/gRPC status errors. An explicitly nil annotation is rejected; an omitted annotation remains a pass-through. Default AND logic is applied to a local annotation copy, without modifying shared route configuration.

HTTP request and cookie operations support the public Kratos transport interfaces, including transport wrappers. Install middleware that replaces the transport before DToken registration so the adapter captures the intended transport. Body token extraction supports URL-encoded and multipart form fields without falling back to Query. Cookie writes retain the standard `net/http` value and SameSite behavior. Cookies and raw HTTP response writing apply only to HTTP; gRPC uses request and reply metadata, and path permission checks use the RPC operation name.

HTTP client IP extraction uses `RemoteAddr`; gRPC uses the peer address. Forwarding headers are not trusted automatically. Behind HTTP proxies, validate trusted proxy addresses and normalize `RemoteAddr` upstream. TLS detection uses the HTTP request's TLS state or gRPC peer TLS credentials, rather than forwarding headers or URL schemes.

Streaming RPCs require a separate authentication setup: in Kratos v2.9.1, registering these handlers through `grpc.Middleware` does not authenticate stream establishment. `grpc.StreamMiddleware` runs around individual message sends and receives; use a stream interceptor when authentication must precede the streaming service handler.

## Beego Request Handling

Beego authentication filters and hooks use the current `c.Request.Context()`, preserving request values, deadlines, and cancellation. The registration-time `ctx` argument remains for API compatibility. After `RegisterDTokenContextMiddleware(ctx, WithManager(mgr))`, filters without an explicit manager or auth type inherit the request manager. Explicit `WithManager` takes precedence over `WithAuthType`, followed by the request cache and global default. Context facades reject nil or closed cached managers, and renewal uses the request manager.

Authentication failures mark `ResponseWriter.Started` before invoking `FailFunc`. This stops the native filter chain even when the callback writes no response, while still allowing the callback to write its own status and body. Adapter `Abort()` and hook `req.Exit()` use the same native state. DToken filters also stop when an upstream hook has already written a response. Beego filters have no nested `Next` call: `req.Next()` skips the current DToken check and lets Beego dispatch subsequent filters. `Next()` and `Exit()` honor the first decision only.

Register authentication filters before the protected handler and keep Beego's default `WithReturnOnOutput(true)`. The `RegisterAuthFilter`, `RegisterAccessFilter`, `RegisterPermissionFilter`, and `RegisterRoleFilter` helpers set this option explicitly at `BeforeRouter`. Setting it to false can allow Beego to continue into the controller after a rejected request. Static files can be served before `BeforeRouter`; protect them with an appropriate `BeforeStatic` filter when needed.

Query token lookup reads only the URL query; body lookup reads only URL-encoded or multipart form fields. Neither source falls back to the other or to route parameters. Client IP comes from `RemoteAddr`; deployments behind proxies should validate trusted proxy addresses and normalize `RemoteAddr` upstream. TLS detection uses the actual HTTP request TLS state. `SetStatusCode` followed by adapter `Write` commits the pending Beego status. `GetBody` preserves Beego's cached-body semantics: when `Input.RequestBody` is available it returns a copy; otherwise it reads and restores the current request body. Beego may already have transformed that body according to `CopyRequestBody` settings.

`RouteAccessHandlerFromAnnotations(nil)` produces an invalid-parameter failure through the configured `FailFunc`. Omitting annotations retains `AccessMiddleware`'s login check; an empty annotation or `Ignore: true` skips authentication, preserving the existing Beego annotation policy.

## Package Aliases

Recommended aliases:

| Framework | Import path | Alias |
| --- | --- | --- |
| Gin | `github.com/Zany2/dtoken-go/integrations/gin` | `gindt` |
| Echo | `github.com/Zany2/dtoken-go/integrations/echo` | `echodt` |
| Fiber | `github.com/Zany2/dtoken-go/integrations/fiber` | `fiberdt` |
| Chi | `github.com/Zany2/dtoken-go/integrations/chi` | `chidt` |
| GoFrame | `github.com/Zany2/dtoken-go/integrations/gf` | `gfdt` |
| Hertz | `github.com/Zany2/dtoken-go/integrations/hertz` | `hertzdt` |
| Kratos | `github.com/Zany2/dtoken-go/integrations/kratos` | `kratosdt` |
| Beego | `github.com/Zany2/dtoken-go/integrations/beego` | `beegodt` |

## When To Import Core Directly

Use `defaults`, `core/builder`, `core/manager`, or `dtoken` for framework-agnostic infrastructure, tests, shared libraries, and business handlers. Use `core/builder` directly only when you want to inject every adapter yourself. Use the integration package only where the code needs framework-specific request handling.

## Related Documents

- [DToken API](../../api/dtoken.md)
- [Authentication](../core/authentication.md)
- [Annotation Guide](../integration/annotation.md)
- [Multi-Auth Systems](../core/multi-auth.md)
- [AccessProvider](../core/access-provider.md)
