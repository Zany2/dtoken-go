# Permission Management

[中文文档](../core/permission_zh.md) | English

Here `dt` names the root package. `Build()` registers automatically; use `dt.XXX()` for the default namespace or `dt.AuthType("user").XXX()` for another namespace. Business method signatures match Auth methods. See [Auth API](../../api/auth.md).

## Overview

Permissions and roles come from the session or a business provider injected with `SetAccessProvider`. Non-nil provider results take precedence; nil falls back to the session, and an empty slice means no permissions or roles. Check/Get methods return provider errors; Has methods return false.

Token authorization validates credentials before resolving provider data. On the normal path, Session is read and decoded only for fallback. Combined HTTP permission and role checks share the lazily loaded Session without cross-request caching. Provider-owned authorization does not require Session existence; see [AccessProvider](access-provider.md).

Account queries supply the namespace and loginID; token queries also supply device and token information. The old CustomPermissionList/CustomRoleList callbacks have been removed; migrate through AccessProviderFunc.

The default permission matcher compares segments directly without allocating the two split slices on each wildcard comparison. Its existing rules remain: `*` alone matches any permission; otherwise a whole `*` segment matches one segment, including an empty one, and segment counts must agree. A pattern containing `/` uses `/` as its separator; other patterns use `:`. Partial stars such as `user*` remain literal. Default role matching remains exact, and custom Strategy matchers keep their behavior and invocation order.

## Initialization

<!-- dtoken-compile: complete-example -->
```go
package main

import (
    dt "github.com/Zany2/dtoken-go"

    "github.com/Zany2/dtoken-go/storage/memory"
)

func initDToken() {
    _, err := dt.NewBuilder().
        Storage(memory.NewStorage()).
        Build()
    if err != nil {
        panic(err)
    }
}
```

## Permission APIs

Token-based permission and role additions/removals validate credentials even when the list is `nil`, empty, or contains only empty strings. Invalid, retired, or inactive tokens return the corresponding error; reversible device restrictions require explicit checks at the access point. For valid credentials, an empty change does not write the session, emit change events, or schedule renewal or activity maintenance.

Permission and role removal filters the stored list once using a set sized to the requested values, without building a second set of all existing account values. Remaining entries keep their original order; change events contain each actually removed value once, in first-request order. No-match removals reuse the unchanged list and skip Session writes and change events. These are local processing improvements; storage reads and authentication checks are unchanged.

### Add Permissions

```go
ctx := context.Background()

token, _ := dt.Login(ctx, dt.LoginOptions{LoginID: "10001"})

_ = dt.AddPermissions(ctx, dt.PermissionOptions{LoginID: "10001", Permissions: []string{
    "user:read",
    "user:write",
    "admin:*",
}})

_ = dt.AddPermissionsByToken(ctx, token, []string{
    "article:publish",
})
```

### Remove Permissions

```go
ctx := context.Background()

_ = dt.RemovePermissions(ctx, dt.PermissionOptions{LoginID: "10001", Permissions: []string{"user:write"}})
_ = dt.RemovePermissionsByToken(ctx, token, []string{"article:publish"})
```

### Query Permissions

```go
ctx := context.Background()

permissions, err := dt.GetPermissions(ctx, "10001")
permissionsByToken, err := dt.GetPermissionsByToken(ctx, token)
```

## Permission Checks

### Single Permission

```go
ctx := context.Background()

hasPermission := dt.HasPermission(ctx, "10001", "user:read")
hasPermissionByToken := dt.HasPermissionByToken(ctx, token, "user:read")

err := dt.CheckPermission(ctx, dt.PermissionOptions{LoginID: "10001", Permission: "user:read"})
```

### AND Logic

```go
ctx := context.Background()

hasAll := dt.HasPermissionsAnd(ctx, "10001", []string{
    "user:read",
    "user:write",
})

err := dt.CheckPermissionAnd(ctx, "10001", []string{
    "user:read",
    "user:write",
})
```

### OR Logic

```go
ctx := context.Background()

hasAny := dt.HasPermissionsOr(ctx, "10001", []string{
    "admin:read",
    "report:read",
})

err := dt.CheckPermissionOr(ctx, "10001", []string{
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

_ = dt.AddPermissions(ctx, dt.PermissionOptions{LoginID: "10001", Permissions: []string{
    "admin:*",
    "user:*:view",
}})

dt.HasPermission(ctx, "10001", "admin:read")        // true
dt.HasPermission(ctx, "10001", "admin:delete")      // true
dt.HasPermission(ctx, "10001", "user:profile:view") // true
dt.HasPermission(ctx, "10001", "user:view")         // false
```

## Role APIs

```go
ctx := context.Background()

_ = dt.AddRoles(ctx, dt.RoleOptions{LoginID: "10001", Roles: []string{"admin", "editor"}})
_ = dt.AddRolesByToken(ctx, token, []string{"reviewer"})

_ = dt.RemoveRoles(ctx, dt.RoleOptions{LoginID: "10001", Roles: []string{"editor"}})
_ = dt.RemoveRolesByToken(ctx, token, []string{"reviewer"})

roles, err := dt.GetRoles(ctx, "10001")
rolesByToken, err := dt.GetRolesByToken(ctx, token)

hasRole := dt.HasRole(ctx, "10001", "admin")
hasAnyRole := dt.HasRolesOr(ctx, "10001", []string{"admin", "manager"})
hasAllRoles := dt.HasRolesAnd(ctx, "10001", []string{"admin", "editor"})

err = dt.CheckRole(ctx, dt.RoleOptions{LoginID: "10001", Role: "admin"})
err = dt.CheckRoleOr(ctx, "10001", []string{"admin", "manager"})
err = dt.CheckRoleAnd(ctx, "10001", []string{"admin", "editor"})
```

## Disable APIs

### Account Disable

```go
ctx := context.Background()

_ = dt.Disable(ctx, dt.DisableOptions{LoginID: "10001", Duration: 2*time.Hour, Reason: "abuse"})

disabled := dt.IsDisable(ctx, "10001")
err := dt.CheckDisable(ctx, "10001")

info, err := dt.GetDisableInfo(ctx, "10001")
ttl, err := dt.GetDisableTTL(ctx, "10001")

_ = dt.Untie(ctx, "10001")
```

`GetDisableTTL()` returns:

- `-2`: not disabled
- `-1`: permanently disabled
- `>0`: remaining seconds

### Service-Level Disable

```go
ctx := context.Background()

_ = dt.DisableService(ctx, dt.ServiceDisableOptions{LoginID: "10001", Service: "comment", Duration: 30*time.Minute})
_ = dt.DisableServiceWithReason(ctx, "10001", "comment", 30*time.Minute, "spam")
_ = dt.DisableServiceLevel(ctx, "10001", "comment", 2, 30*time.Minute)
_ = dt.DisableServiceLevelWithReason(ctx, "10001", "comment", 3, 30*time.Minute, "risk")

serviceDisabled := dt.IsDisableService(ctx, "10001", "comment")
levelDisabled := dt.IsDisableServiceLevel(ctx, "10001", "comment", 2)

err := dt.CheckDisableService(ctx, "10001", []string{"comment", "post"})
err = dt.CheckDisableServiceLevel(ctx, "10001", "comment", 2)

serviceInfo, err := dt.GetDisableServiceInfo(ctx, "10001", "comment")
serviceTTL, err := dt.GetDisableServiceTTL(ctx, "10001", "comment")

_ = dt.UntieService(ctx, "10001", "comment")
```

## Custom Permission And Role Provider

This configures and registers an instance with a custom provider. Use it in place of the initialization above to avoid registering the default namespace twice. PermissionFunc and RoleFunc use AccessSubject instead of separate normal and Ext callbacks.

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

Build automatically registers this instance for the root calls above. For HTTP, inject the same instance with WithAuth(auth). See [AccessProvider](access-provider.md).

## Using With Gin Routes

```go
ctx := context.Background()

auth, err := dt.NewBuilder().Build()
if err != nil { panic(err) }
defer auth.Close()

authOption1, authErr := gindt.WithAuth(auth)
if authErr != nil {
    return authErr
}
r.Use(gindt.RegisterDTokenContextMiddleware(ctx, authOption1))

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
