# Permission Management

[中文文档](../core/permission_zh.md) | English

This page uses `global` for registered-instance calls and `dt` for the root package. Build Auth and call `global.SetAuth(auth)` first; omitted auth types select the default `auth:` namespace. See [Auth API](../../api/auth.md) for instance calls and parameter differences.

## Overview

Permissions and roles come from the session or a business provider injected with `SetAccessProvider`. Non-nil provider results take precedence; nil falls back to the session, and an empty slice means no permissions or roles. Check/Get methods return provider errors; Has methods return false.

Account queries supply the namespace and loginID; token queries also supply device and token information. The old CustomPermissionList/CustomRoleList callbacks have been removed; migrate through AccessProviderFunc.

## Initialization

<!-- dtoken-compile: complete-example -->
```go
package main

import (
    dt "github.com/Zany2/dtoken-go"
    global "github.com/Zany2/dtoken-go/global"

    "github.com/Zany2/dtoken-go/storage/memory"
)

func initDToken() {
    auth, err := dt.NewBuilder().
        Storage(memory.NewStorage()).
        Build()
    if err != nil {
        panic(err)
    }
    global.SetAuth(auth)
}
```

## Permission APIs

Token-based permission and role additions/removals validate the full login state even when the list is `nil`, empty, or contains only empty strings. Invalid, disabled, or inactive tokens return the corresponding error. For a valid login, an empty change does not write the session, emit change events, or schedule renewal or activity maintenance.

### Add Permissions

```go
ctx := context.Background()

token, _ := global.Login(ctx, "10001")

_ = global.AddPermissions(ctx, "10001", []string{
    "user:read",
    "user:write",
    "admin:*",
})

_ = global.AddPermissionsByToken(ctx, token, []string{
    "article:publish",
})
```

### Remove Permissions

```go
ctx := context.Background()

_ = global.RemovePermissions(ctx, "10001", []string{"user:write"})
_ = global.RemovePermissionsByToken(ctx, token, []string{"article:publish"})
```

### Query Permissions

```go
ctx := context.Background()

permissions, err := global.GetPermissions(ctx, "10001")
permissionsByToken, err := global.GetPermissionsByToken(ctx, token)
```

## Permission Checks

### Single Permission

```go
ctx := context.Background()

hasPermission := global.HasPermission(ctx, "10001", "user:read")
hasPermissionByToken := global.HasPermissionByToken(ctx, token, "user:read")

err := global.CheckPermission(ctx, "10001", "user:read")
```

### AND Logic

```go
ctx := context.Background()

hasAll := global.HasPermissionsAnd(ctx, "10001", []string{
    "user:read",
    "user:write",
})

err := global.CheckPermissionAnd(ctx, "10001", []string{
    "user:read",
    "user:write",
})
```

### OR Logic

```go
ctx := context.Background()

hasAny := global.HasPermissionsOr(ctx, "10001", []string{
    "admin:read",
    "report:read",
})

err := global.CheckPermissionOr(ctx, "10001", []string{
    "admin:read",
    "report:read",
})
```

## Wildcard Matching

Wildcard matching supports `*` and works segment by segment:

| Pattern | Description | Example |
|------|------|------|
| `*` | matches all permissions | any permission |
| `user:*` | matches two-part `user` permissions | `user:read`, `user:write` |
| `user:*:view` | matches three-part permissions | `user:profile:view` |
| `admin/*` | `/` is also supported as a separator | `admin/read` |

Two important implementation details:

1. The separator is detected automatically, defaulting to `:` and switching to `/` when the pattern contains `/`
2. Segment counts must be equal, which prevents overly broad matches

```go
ctx := context.Background()

_ = global.AddPermissions(ctx, "10001", []string{
    "admin:*",
    "user:*:view",
})

global.HasPermission(ctx, "10001", "admin:read")        // true
global.HasPermission(ctx, "10001", "admin:delete")      // true
global.HasPermission(ctx, "10001", "user:profile:view") // true
global.HasPermission(ctx, "10001", "user:view")         // false
```

## Role APIs

```go
ctx := context.Background()

_ = global.AddRoles(ctx, "10001", []string{"admin", "editor"})
_ = global.AddRolesByToken(ctx, token, []string{"reviewer"})

_ = global.RemoveRoles(ctx, "10001", []string{"editor"})
_ = global.RemoveRolesByToken(ctx, token, []string{"reviewer"})

roles, err := global.GetRoles(ctx, "10001")
rolesByToken, err := global.GetRolesByToken(ctx, token)

hasRole := global.HasRole(ctx, "10001", "admin")
hasAnyRole := global.HasRolesOr(ctx, "10001", []string{"admin", "manager"})
hasAllRoles := global.HasRolesAnd(ctx, "10001", []string{"admin", "editor"})

err = global.CheckRole(ctx, "10001", "admin")
err = global.CheckRoleOr(ctx, "10001", []string{"admin", "manager"})
err = global.CheckRoleAnd(ctx, "10001", []string{"admin", "editor"})
```

## Disable APIs

### Account Disable

```go
ctx := context.Background()

_ = global.Disable(ctx, "10001", 2*time.Hour, "abuse")

disabled := global.IsDisable(ctx, "10001")
err := global.CheckDisable(ctx, "10001")

info, err := global.GetDisableInfo(ctx, "10001")
ttl, err := global.GetDisableTTL(ctx, "10001")

_ = global.Untie(ctx, "10001")
```

`GetDisableTTL()` returns:

- `-2`: not disabled
- `-1`: permanently disabled
- `>0`: remaining seconds

### Service-Level Disable

```go
ctx := context.Background()

_ = global.DisableService(ctx, "10001", "comment", 30*time.Minute)
_ = global.DisableServiceWithReason(ctx, "10001", "comment", 30*time.Minute, "spam")
_ = global.DisableServiceLevel(ctx, "10001", "comment", 2, 30*time.Minute)
_ = global.DisableServiceLevelWithReason(ctx, "10001", "comment", 3, 30*time.Minute, "risk")

serviceDisabled := global.IsDisableService(ctx, "10001", "comment")
levelDisabled := global.IsDisableServiceLevel(ctx, "10001", "comment", 2)

err := global.CheckDisableService(ctx, "10001", []string{"comment", "post"})
err = global.CheckDisableServiceLevel(ctx, "10001", "comment", 2)

serviceInfo, err := global.GetDisableServiceInfo(ctx, "10001", "comment")
serviceTTL, err := global.GetDisableServiceTTL(ctx, "10001", "comment")

_ = global.UntieService(ctx, "10001", "comment")
```

## Custom Permission And Role Provider

This creates an independent instance. PermissionFunc and RoleFunc use AccessSubject instead of separate normal and Ext callbacks.

```go
provider := dt.AccessProviderFunc{
    PermissionFunc: func(ctx context.Context, subject dt.AccessSubject) ([]string, error) {
        if subject.Device == "app" {
            return []string{"mobile:read", "mobile:write"}, nil
        }
        return nil, nil
    },
    RoleFunc: func(ctx context.Context, subject dt.AccessSubject) ([]string, error) {
        if subject.LoginID == "10001" {
            return []string{"admin"}, nil
        }
        return []string{}, nil
    },
}
auth, err := dt.NewBuilder().SetAccessProvider(provider).Build()
if err != nil {
    panic(err)
}
defer auth.Close()
```

For the global calls above, explicitly register global.SetAuth(auth). For HTTP, inject the same instance with WithAuth(auth). See [AccessProvider](access-provider.md).

## Using With Gin Routes

```go
ctx := context.Background()

auth, err := dt.NewBuilder().Build()
if err != nil { panic(err) }
defer auth.Close()

r.Use(gindt.RegisterDTokenContextMiddleware(ctx, gindt.WithAuth(auth)))

r.GET("/users",
    gindt.CheckPermissionMiddleware(ctx, []string{"user:read"}, nil, nil),
    listUsersHandler,
)

r.POST("/admin",
    gindt.CheckRoleMiddleware(ctx, []string{"admin"}, nil, nil),
    adminHandler,
)
```

## Best Practices

1. Login before writing roles or permissions, otherwise the session may not exist yet
2. Use custom callbacks when permission data changes frequently
3. Use `Check*` APIs for request guards and `Has*` APIs for internal branching
4. Use service-level disable for capabilities such as comments, posts, or payment instead of disabling the full account every time

## Related Documentation

- [Authentication Guide](../core/authentication.md)
- [Annotation Guide](../integration/annotation.md)
- [AccessProvider](../core/access-provider.md)
- [Disable System](../core/disable.md)
- [JWT Guide](../security/jwt.md)
