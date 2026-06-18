# Framework Integration Usage Guide

**[中文文档](../integration/framework-integration_zh.md)**

## Overview

DToken-Go keeps core authentication APIs and framework integrations separate. Use `dtoken` for login, logout, session, permission, role, and global manager APIs. Use `defaults` to create managers with bundled default components. Use `integrations/*` packages only for framework middleware, annotations, and request-context helpers.

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

    mgr := defaults.NewBuilder().
        SetStorage(storage).
        TokenName("token").
        Timeout(2 * 60 * 60).
        Build()
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
