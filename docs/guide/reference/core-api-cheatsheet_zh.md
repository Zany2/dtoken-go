# Core API 速查

本页整理 `dtoken` 全局 API 的常用调用方式，适合在已经完成 Manager 初始化后快速查阅。

## 登录认证

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

## 角色权限

```go
_ = dtoken.AddRoles(ctx, "user-1001", []string{"admin", "auditor"})
_ = dtoken.AddPermissions(ctx, "user-1001", []string{"order:read", "order:*"})

hasRole := dtoken.HasRole(ctx, "user-1001", "admin")
hasAnyRole := dtoken.HasRolesOr(ctx, "user-1001", []string{"admin", "owner"})
hasPermission := dtoken.HasPermission(ctx, "user-1001", "order:read")
hasAllPermission := dtoken.HasPermissionsAnd(ctx, "user-1001", []string{"order:read", "order:write"})

_, _, _, _ = hasRole, hasAnyRole, hasPermission, hasAllPermission
```

## 在线状态控制

```go
_ = dtoken.Kickout(ctx, token)
_ = dtoken.KickoutByLoginID(ctx, "user-1001")
_ = dtoken.Replace(ctx, token)
_ = dtoken.ReplaceByLoginID(ctx, "user-1001")
```

## 封禁控制

```go
_ = dtoken.Disable(ctx, "user-1001", 3600, "risk_control")
disabled := dtoken.IsDisable(ctx, "user-1001")
disableInfo, err := dtoken.GetDisableInfo(ctx, "user-1001")
_ = dtoken.Untie(ctx, "user-1001")

_, _ = disabled, disableInfo
```

更多完整 API 见 [DToken API 文档](../../api/dtoken_zh.md)。
