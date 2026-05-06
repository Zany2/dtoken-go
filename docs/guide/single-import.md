# Single Import Usage Guide

**[中文文档](single-import_zh.md)**

## Overview

DToken-Go integration packages re-export the common core types, builder helpers, errors, constants, and global `dtoken` APIs. If you are already using a framework integration package, you can usually import only that integration package plus your chosen storage package.

This keeps application code focused on one DToken namespace, for example `gindt` for Gin.

## Installation

```bash
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
```

## Gin Example

```go
package main

import (
    "context"
    "net/http"

    "github.com/Zany2/dtoken-go/com/storage/memory"
    gindt "github.com/Zany2/dtoken-go/integrations/gin"
    "github.com/gin-gonic/gin"
)

func main() {
    ctx := context.Background()
    storage := memory.NewStorage()

    mgr := gindt.NewDefaultBuilder().
        SetStorage(storage).
        TokenName("token").
        Timeout(2 * 60 * 60).
        Build()
    gindt.SetManager(mgr)

    r := gin.Default()
    r.Use(gindt.RegisterDTokenContextMiddleware(ctx))

    r.POST("/login", func(c *gin.Context) {
        token, err := gindt.Login(ctx, "1000")
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
        loginID, _ := gindt.GetLoginID(ctx, token)
        c.JSON(http.StatusOK, gin.H{"loginId": loginID})
    })

    _ = r.Run(":8080")
}
```

## Common API Surface

Integration packages expose the same style of shortcuts as `dtoken`:

```go
gindt.SetManager(mgr)
mgr, err := gindt.GetManager()

token, err := gindt.Login(ctx, "1000")
ok := gindt.IsLogin(ctx, token)
loginID, err := gindt.GetLoginID(ctx, token)
err = gindt.Logout(ctx, token)

err = gindt.AddPermissions(ctx, "1000", []string{"user:read"})
hasPermission := gindt.HasPermission(ctx, "1000", "user:read")

err = gindt.AddRoles(ctx, "1000", []string{"admin"})
hasRole := gindt.HasRole(ctx, "1000", "admin")
```

They also expose framework middleware:

```go
r.Use(gindt.RegisterDTokenContextMiddleware(ctx))
r.Use(gindt.AuthMiddleware(ctx))
r.Use(gindt.PermissionMiddleware(ctx, []string{"user:read"}))
r.Use(gindt.RoleMiddleware(ctx, []string{"admin"}))
```

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

## When To Import Core Directly

Use direct imports like `core/builder`, `core/manager`, or `dtoken` when you are building framework-agnostic infrastructure, tests, or shared libraries. In normal web handlers, the integration package is usually enough.

## Related Documents

- [DToken API](../api/dtoken.md)
- [Authentication](authentication.md)
- [Annotation Guide](annotation.md)
