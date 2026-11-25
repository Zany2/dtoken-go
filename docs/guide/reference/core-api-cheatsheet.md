# Root API Cheatsheet

Fragment alias: dt = github.com/Zany2/dtoken-go. Build registers automatically. Use root functions for the default system or dt.AuthType(name) for another system; business signatures match Auth methods. Drain requests before closing. BuildInstance creates unregistered instances that must be closed separately. See [Auth API](../../api/auth.md) for options.

## Login Authentication

```go
token, err := dt.Login(ctx, dt.LoginOptions{LoginID: "user-1001"})
token, err = dt.Login(ctx, dt.LoginOptions{LoginID: "user-1001", Device: "web", DeviceID: "browser-001"})

isLogin := dt.IsLogin(ctx, token)
loginID, err := dt.GetLoginID(ctx, token)
tokenInfo, err := dt.GetTokenInfo(ctx, token)
ttl, err := dt.GetTokenTTL(ctx, token)

err = dt.RenewTimeout(ctx, token, 2*time.Hour)
err = dt.LogoutByToken(ctx, token)
err = dt.LogoutByLoginID(ctx, "user-1001")
```

## Roles And Permissions

```go
_ = dt.AddRoles(ctx, dt.RoleOptions{LoginID: "user-1001", Roles: []string{"admin", "auditor"}})
_ = dt.AddPermissions(ctx, dt.PermissionOptions{LoginID: "user-1001", Permissions: []string{"order:read", "order:*"}})

hasRole := dt.HasRole(ctx, "user-1001", "admin")
hasAnyRole := dt.HasRolesOr(ctx, "user-1001", []string{"admin", "owner"})
hasPermission := dt.HasPermission(ctx, "user-1001", "order:read")
hasAllPermission := dt.HasPermissionsAnd(ctx, "user-1001", []string{"order:read", "order:write"})

_, _, _, _ = hasRole, hasAnyRole, hasPermission, hasAllPermission
```

## Online State Control

```go
_ = dt.KickoutByToken(ctx, token)
_ = dt.KickoutByLoginID(ctx, "user-1001")
_ = dt.ReplaceByToken(ctx, token)
_ = dt.ReplaceByLoginID(ctx, "user-1001")
```

## Ban Control

```go
_ = dt.Disable(ctx, dt.DisableOptions{LoginID: "user-1001", Duration: time.Hour, Reason: "risk_control"})
disabled := dt.IsDisable(ctx, "user-1001")
disableInfo, err := dt.GetDisableInfo(ctx, "user-1001")
_ = dt.Untie(ctx, "user-1001")

_, _ = disabled, disableInfo
```

For the full API reference, see [DToken API](../../api/dtoken.md).
