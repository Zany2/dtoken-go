# Permission Management

[中文文档](../core/permission_zh.md) | English

## Overview

In the current version, permissions and roles come from two places:

1. `Session` fields: `Permissions` and `Roles`
2. Custom callbacks configured through `defaults.NewBuilder()`

Priority rules:

- By `loginID`: `CustomPermissionListFunc` / `CustomRoleListFunc` override session data
- By `token`: `CustomPermissionListExtFunc` / `CustomRoleListExtFunc` override normal callbacks, which override session data

## Initialization

```go
package main

import (
    "context"

    "github.com/Zany2/dtoken-go/com/storage/memory"
    "github.com/Zany2/dtoken-go/defaults"
    "github.com/Zany2/dtoken-go/dtoken"
)

func initDToken() {
    dtoken.SetManager(
        defaults.NewBuilder().
            SetStorage(memory.NewStorage()).
            Build(),
    )
}
```

## Permission APIs

### Add Permissions

```go
ctx := context.Background()

token, _ := dtoken.Login(ctx, "10001")

_ = dtoken.AddPermissions(ctx, "10001", []string{
    "user:read",
    "user:write",
    "admin:*",
})

_ = dtoken.AddPermissionsByToken(ctx, token, []string{
    "article:publish",
})
```

### Remove Permissions

```go
ctx := context.Background()

_ = dtoken.RemovePermissions(ctx, "10001", []string{"user:write"})
_ = dtoken.RemovePermissionsByToken(ctx, token, []string{"article:publish"})
```

### Query Permissions

```go
ctx := context.Background()

permissions, err := dtoken.GetPermissions(ctx, "10001")
permissionsByToken, err := dtoken.GetPermissionsByToken(ctx, token)
```

## Permission Checks

### Single Permission

```go
ctx := context.Background()

hasPermission := dtoken.HasPermission(ctx, "10001", "user:read")
hasPermissionByToken := dtoken.HasPermissionByToken(ctx, token, "user:read")

err := dtoken.CheckPermission(ctx, "10001", "user:read")
```

### AND Logic

```go
ctx := context.Background()

hasAll := dtoken.HasPermissionsAnd(ctx, "10001", []string{
    "user:read",
    "user:write",
})

err := dtoken.CheckPermissionAnd(ctx, "10001", []string{
    "user:read",
    "user:write",
})
```

### OR Logic

```go
ctx := context.Background()

hasAny := dtoken.HasPermissionsOr(ctx, "10001", []string{
    "admin:read",
    "report:read",
})

err := dtoken.CheckPermissionOr(ctx, "10001", []string{
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

_ = dtoken.AddPermissions(ctx, "10001", []string{
    "admin:*",
    "user:*:view",
})

dtoken.HasPermission(ctx, "10001", "admin:read")        // true
dtoken.HasPermission(ctx, "10001", "admin:delete")      // true
dtoken.HasPermission(ctx, "10001", "user:profile:view") // true
dtoken.HasPermission(ctx, "10001", "user:view")         // false
```

## Role APIs

```go
ctx := context.Background()

_ = dtoken.AddRoles(ctx, "10001", []string{"admin", "editor"})
_ = dtoken.AddRolesByToken(ctx, token, []string{"reviewer"})

_ = dtoken.RemoveRoles(ctx, "10001", []string{"editor"})
_ = dtoken.RemoveRolesByToken(ctx, token, []string{"reviewer"})

roles, err := dtoken.GetRoles(ctx, "10001")
rolesByToken, err := dtoken.GetRolesByToken(ctx, token)

hasRole := dtoken.HasRole(ctx, "10001", "admin")
hasAnyRole := dtoken.HasRolesOr(ctx, "10001", []string{"admin", "manager"})
hasAllRoles := dtoken.HasRolesAnd(ctx, "10001", []string{"admin", "editor"})

err = dtoken.CheckRole(ctx, "10001", "admin")
err = dtoken.CheckRoleOr(ctx, "10001", []string{"admin", "manager"})
err = dtoken.CheckRoleAnd(ctx, "10001", []string{"admin", "editor"})
```

## Disable APIs

### Account Disable

```go
ctx := context.Background()

_ = dtoken.Disable(ctx, "10001", 2*time.Hour, "abuse")

disabled := dtoken.IsDisable(ctx, "10001")
err := dtoken.CheckDisable(ctx, "10001")

info, err := dtoken.GetDisableInfo(ctx, "10001")
ttl, err := dtoken.GetDisableTTL(ctx, "10001")

_ = dtoken.Untie(ctx, "10001")
```

`GetDisableTTL()` returns:

- `-2`: not disabled
- `-1`: permanently disabled
- `>0`: remaining seconds

### Service-Level Disable

```go
ctx := context.Background()

_ = dtoken.DisableService(ctx, "10001", "comment", 30*time.Minute)
_ = dtoken.DisableServiceWithReason(ctx, "10001", "comment", 30*time.Minute, "spam")
_ = dtoken.DisableServiceLevel(ctx, "10001", "comment", 2, 30*time.Minute)
_ = dtoken.DisableServiceLevelWithReason(ctx, "10001", "comment", 3, 30*time.Minute, "risk")

serviceDisabled := dtoken.IsDisableService(ctx, "10001", "comment")
levelDisabled := dtoken.IsDisableServiceLevel(ctx, "10001", "comment", 2)

err := dtoken.CheckDisableService(ctx, "10001", []string{"comment", "post"})
err = dtoken.CheckDisableServiceLevel(ctx, "10001", "comment", 2)

serviceInfo, err := dtoken.GetDisableServiceInfo(ctx, "10001", "comment")
serviceTTL, err := dtoken.GetDisableServiceTTL(ctx, "10001", "comment")

_ = dtoken.UntieService(ctx, "10001", "comment")
```

## Custom Permission and Role Callbacks

```go
dtoken.SetManager(
    defaults.NewBuilder().
        SetStorage(memory.NewStorage()).
        SetCustomPermissionListFunc(func(loginID, authType string) ([]string, error) {
            if loginID == "10001" {
                return []string{"user:read", "user:write"}, nil
            }
            return []string{"user:read"}, nil
        }).
        SetCustomRoleListFunc(func(loginID, authType string) ([]string, error) {
            if loginID == "10001" {
                return []string{"admin"}, nil
            }
            return []string{"user"}, nil
        }).
        Build(),
)
```

Extended callbacks can also use `device` and `deviceId`:

```go
defaults.NewBuilder().
    SetCustomPermissionListExtFunc(func(loginID, device, deviceId, authType string) ([]string, error) {
        if device == "app" {
            return []string{"mobile:read", "mobile:write"}, nil
        }
        return []string{"web:read"}, nil
    }).
    SetCustomRoleListExtFunc(func(loginID, device, deviceId, authType string) ([]string, error) {
        if device == "app" {
            return []string{"mobile-user"}, nil
        }
        return []string{"web-user"}, nil
    })
```

## Using With Gin Routes

```go
ctx := context.Background()

r.Use(gindt.RegisterDTokenContextMiddleware(ctx))

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
