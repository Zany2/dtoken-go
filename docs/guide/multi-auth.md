# Multi-Auth Systems

**[中文文档](multi-auth_zh.md)**

Multi-auth systems let one project maintain multiple isolated authentication systems, such as end users, administrators, and open-platform clients. DToken-Go separates them with `AuthType`.

## Core Concept

`AuthType` participates in manager registration, Token lookup, Session lookup, permission and role resolution, and storage key generation. A Token can only be recognized by its own auth system.

```text
dtoken:user:token:xxx
dtoken:admin:token:yyy
dtoken:app:session:client-001
```

`AuthType("user")` is normalized to `user:`. Global APIs accept either `"user"` or `"user:"`.

## Register Multiple Managers

```go
ctx := context.Background()
storage := memory.NewStorage()

userMgr, err := defaults.NewBuilder().
    AuthType("user").
    KeyPrefix("dtoken").
    TokenName("user-token").
    SetStorage(storage).
    Build()
if err != nil {
    panic(err)
}
dtoken.SetManager(userMgr)

adminMgr, err := defaults.NewBuilder().
    AuthType("admin").
    KeyPrefix("dtoken").
    TokenName("admin-token").
    SetStorage(storage).
    Build()
if err != nil {
    panic(err)
}
dtoken.SetManager(adminMgr)

userToken, _ := dtoken.Login(ctx, "10001", "", "", "user")
adminToken, _ := dtoken.Login(ctx, "admin-1", "", "", "admin")
```

`BuildAndSetManager` can also override and register `AuthType` during construction:

```go
_, err := dtoken.BuildAndSetManager(
    defaults.NewBuilder().
        KeyPrefix("dtoken").
        SetStorage(storage),
    "admin",
)
```

## Call A Specific Auth System

Most global APIs accept an optional final `authType` argument:

```go
token, err := dtoken.Login(ctx, "10001", "web", "chrome", "user")

loginID, err := dtoken.GetLoginID(ctx, token, "user")
err = dtoken.CheckPermission(ctx, "10001", "user:read", "user")
roles, err := dtoken.GetRoles(ctx, "10001", "user")
```

If `authType` is omitted, the default auth system `auth:` is used.

## Use With Framework Middleware

Different route groups can bind different auth systems:

```go
r.Use(gindt.RegisterDTokenContextMiddleware(ctx))

userGroup := r.Group("/api")
userGroup.Use(gindt.AuthMiddleware(ctx, gindt.WithAuthType("user")))

adminGroup := r.Group("/admin")
adminGroup.Use(gindt.AuthMiddleware(ctx, gindt.WithAuthType("admin")))
adminGroup.Use(gindt.RoleMiddleware(ctx, []string{"admin"}, gindt.WithAuthType("admin")))
```

With this setup, `/api` only recognizes Tokens from `user`, and `/admin` only recognizes Tokens from `admin`.

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
defaults.NewBuilder().
    AuthType("admin").
    KeyPrefix("dtoken:prod")
```

## Events And Permissions

Event data carries `AuthType`, which is useful for audit logs:

```go
eventManager, _ := dtoken.GetEventManager("admin")
```

`AccessProvider` receives `AuthType` through `AccessSubject`, so one permission service can return different permissions per auth system.

## Suggestions

- Use separate `AuthType` values for end users and administrators.
- Multi-tenancy does not always require `AuthType`; if tenants share one auth rule set, tenant IDs usually belong in business permissions or account data.
- Multiple auth systems can share Redis, but production deployments should use clear `KeyPrefix` values.
- Avoid reusing the same `TokenName` across auth systems unless routes are clearly separated.

## Related Documentation

- [Configuration Guide](configuration.md)
- [Redis Storage](redis-storage.md)
- [Framework Integration](framework-integration.md)
- [AccessProvider](access-provider.md)
