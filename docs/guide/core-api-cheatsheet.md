# Core API Cheatsheet

This page lists common `dtoken` global API calls. It is intended for quick lookup after the Manager has been initialized.

## Login Authentication

```go
token, err := dtoken.Login(ctx, "user-1001")
token, err = dtoken.Login(ctx, "user-1001", "web", "browser-001", "user")

isLogin := dtoken.IsLogin(ctx, token)
loginID, err := dtoken.GetLoginID(ctx, token)
tokenInfo, err := dtoken.GetTokenInfo(ctx, token)
ttl, err := dtoken.GetTokenTTL(ctx, token)

err = dtoken.RenewTimeout(ctx, token, 7200)
err = dtoken.Logout(ctx, token)
err = dtoken.LogoutByLoginID(ctx, "user-1001")
```

## Roles And Permissions

```go
_ = dtoken.AddRoles(ctx, "user-1001", []string{"admin", "auditor"})
_ = dtoken.AddPermissions(ctx, "user-1001", []string{"order:read", "order:*"})

hasRole := dtoken.HasRole(ctx, "user-1001", "admin")
hasAnyRole := dtoken.HasRolesOr(ctx, "user-1001", []string{"admin", "owner"})
hasPermission := dtoken.HasPermission(ctx, "user-1001", "order:read")
hasAllPermission := dtoken.HasPermissionsAnd(ctx, "user-1001", []string{"order:read", "order:write"})

_, _, _, _ = hasRole, hasAnyRole, hasPermission, hasAllPermission
```

## Online State Control

```go
_ = dtoken.Kickout(ctx, token)
_ = dtoken.KickoutByLoginID(ctx, "user-1001")
_ = dtoken.Replace(ctx, token)
_ = dtoken.ReplaceByLoginID(ctx, "user-1001")
```

## Ban Control

```go
_ = dtoken.Disable(ctx, "user-1001", 3600, "risk_control")
disabled := dtoken.IsDisable(ctx, "user-1001")
disableInfo, err := dtoken.GetDisableInfo(ctx, "user-1001")
_ = dtoken.Untie(ctx, "user-1001")

_, _ = disabled, disableInfo
```

For the full API reference, see [DToken API](../api/dtoken.md).
