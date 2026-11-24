# Global API Cheatsheet

Import aliases for code fragments: global = github.com/Zany2/dtoken-go/global.

This page lists common `global` API calls. It is intended for quick lookup after the Manager has been initialized.

This cheatsheet retains legacy global calls. Build registers automatically; drain requests before closing instances or calling dtoken.CloseAll. BuildInstance does not register. See [Auth API](../../api/auth.md) for instance entry points and signature differences; positional global.Login arguments cannot be passed directly to auth.Login.

## Login Authentication

```go
token, err := global.Login(ctx, "user-1001")
token, err = global.Login(ctx, "user-1001", "web", "browser-001")

isLogin := global.IsLogin(ctx, token)
loginID, err := global.GetLoginID(ctx, token)
tokenInfo, err := global.GetTokenInfo(ctx, token)
ttl, err := global.GetTokenTTL(ctx, token)

err = global.RenewTimeout(ctx, token, 2*time.Hour)
err = global.Logout(ctx, token)
err = global.LogoutByLoginID(ctx, "user-1001")
```

## Roles And Permissions

```go
_ = global.AddRoles(ctx, "user-1001", []string{"admin", "auditor"})
_ = global.AddPermissions(ctx, "user-1001", []string{"order:read", "order:*"})

hasRole := global.HasRole(ctx, "user-1001", "admin")
hasAnyRole := global.HasRolesOr(ctx, "user-1001", []string{"admin", "owner"})
hasPermission := global.HasPermission(ctx, "user-1001", "order:read")
hasAllPermission := global.HasPermissionsAnd(ctx, "user-1001", []string{"order:read", "order:write"})

_, _, _, _ = hasRole, hasAnyRole, hasPermission, hasAllPermission
```

## Online State Control

```go
_ = global.Kickout(ctx, token)
_ = global.KickoutByLoginID(ctx, "user-1001")
_ = global.Replace(ctx, token)
_ = global.ReplaceByLoginID(ctx, "user-1001")
```

## Ban Control

```go
_ = global.Disable(ctx, "user-1001", time.Hour, "risk_control")
disabled := global.IsDisable(ctx, "user-1001")
disableInfo, err := global.GetDisableInfo(ctx, "user-1001")
_ = global.Untie(ctx, "user-1001")

_, _ = disabled, disableInfo
```

For the full API reference, see [DToken API](../../api/dtoken.md).
