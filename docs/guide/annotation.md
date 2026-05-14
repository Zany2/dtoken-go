# Annotation Style Guide

[中文文档](annotation_zh.md) | English

## Overview

In the current project, “annotation-style authentication” is implemented as decorator-style middleware wrappers for each web framework, not as native Go annotations.

Each integration package follows the same pattern:

- `RegisterDTokenContextMiddleware(...)`
- `GetHandler(...)`
- `CheckLoginMiddleware(...)`
- `CheckRoleMiddleware(...)`
- `CheckPermissionMiddleware(...)`
- `CheckDisableMiddleware(...)`
- `IgnoreMiddleware(...)`

## Annotation Structure

Using `integrations/gin` as the example, the current `Annotation` struct supports:

| Field | Description |
|------|------|
| `AuthType` | target auth system |
| `CheckLogin` | whether login is required |
| `CheckRole` | required roles |
| `CheckPermission` | required permissions |
| `CheckDisable` | whether account disable should be checked |
| `Ignore` | skip auth entirely |
| `LogicType` | `OR` or `AND` for multi-role and multi-permission checks |

## Gin Usage

### Register Context Middleware First

```go
ctx := context.Background()

r := gin.Default()
r.Use(gindt.RegisterDTokenContextMiddleware(ctx))
```

The annotation middleware then reuses the cached `DTokenContext` on the request.

## Basic Examples

### Ignore Authentication

```go
r.GET("/public",
    gindt.IgnoreMiddleware(ctx, nil, nil),
    func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "public"})
    },
)
```

### Check Login

```go
r.GET("/user/info",
    gindt.CheckLoginMiddleware(ctx, nil, nil),
    func(c *gin.Context) {
        dCtx, _ := gindt.GetDTokenContext(c)
        loginID, _ := dCtx.GetLoginID(ctx)
        c.JSON(200, gin.H{"loginId": loginID})
    },
)
```

### Check Permission

```go
r.GET("/admin/users",
    gindt.CheckPermissionMiddleware(ctx, []string{"admin:user:read"}, nil, nil),
    listUsersHandler,
)
```

### Check Role

```go
r.GET("/admin/dashboard",
    gindt.CheckRoleMiddleware(ctx, []string{"admin"}, nil, nil),
    dashboardHandler,
)
```

### Check Disable

```go
r.GET("/comment/send",
    gindt.CheckDisableMiddleware(ctx, nil, nil),
    sendCommentHandler,
)
```

## Multi-Role and Multi-Permission Logic

### OR Logic

`OR` is the default behavior.

```go
r.GET("/reports",
    gindt.GetHandler(ctx, nil, nil, &gindt.Annotation{
        CheckLogin:      true,
        CheckRole:       []string{"admin", "manager"},
        CheckPermission: []string{"report:read", "report:*"},
        LogicType:       gindt.LogicOr,
    }),
    reportsHandler,
)
```

### AND Logic

```go
r.POST("/admin/publish",
    gindt.GetHandler(ctx, nil, nil, &gindt.Annotation{
        CheckLogin:      true,
        CheckRole:       []string{"admin", "editor"},
        CheckPermission: []string{"article:write", "article:publish"},
        LogicType:       gindt.LogicAnd,
    }),
    publishHandler,
)
```

## Combining Middlewares

```go
r.POST("/super-admin",
    gindt.CheckLoginMiddleware(ctx, nil, nil),
    gindt.CheckRoleMiddleware(ctx, []string{"admin"}, nil, nil),
    gindt.CheckPermissionMiddleware(ctx, []string{"super:*"}, nil, nil),
    superAdminHandler,
)
```

## Route Group Helpers

```go
ctx := context.Background()

api := r.Group("/api")

userGroup := gindt.AuthGroup(ctx, api.Group("/user"), nil, nil)
userGroup.GET("/profile", profileHandler)

adminGroup := gindt.RoleGroup(ctx, api.Group("/admin"), []string{"admin"}, nil, nil)
adminGroup.GET("/dashboard", dashboardHandler)

permGroup := gindt.PermissionGroup(ctx, api.Group("/report"), []string{"report:read"}, nil, nil)
permGroup.GET("/list", reportListHandler)
```

## Custom Failure Handling

```go
failFunc := func(c *gin.Context, err error) {
    c.JSON(200, gin.H{
        "code": 50001,
        "msg":  err.Error(),
    })
}

r.GET("/admin",
    gindt.CheckRoleMiddleware(ctx, []string{"admin"}, nil, failFunc),
    adminHandler,
)
```

## Other Frameworks

`gf`, `echo`, `fiber`, `chi`, `hertz`, and `kratos` already follow the same structure, with signatures adjusted for each framework.

The core idea stays the same: register `DTokenContext` first, then attach annotation-style middleware.

## Related Documentation

- [Authentication Guide](authentication.md)
- [Permission Management](permission.md)
- [Framework Integration Guide](framework-integration.md)
