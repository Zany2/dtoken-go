# 根包 API 速查

代码片段中，dt = github.com/Zany2/dtoken-go。Build 自动注册，默认体系使用 dt 包级函数，其他体系使用 dt.AuthType(name)，业务方法签名与 Auth 实例一致。关闭前先排空请求；未注册实例使用 BuildInstance 创建并单独关闭。完整参数见 [Auth API](../../api/auth_zh.md)。

## 登录认证

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

## 角色权限

```go
_ = dt.AddRoles(ctx, dt.RoleOptions{LoginID: "user-1001", Roles: []string{"admin", "auditor"}})
_ = dt.AddPermissions(ctx, dt.PermissionOptions{LoginID: "user-1001", Permissions: []string{"order:read", "order:*"}})

hasRole := dt.HasRole(ctx, "user-1001", "admin")
hasAnyRole := dt.HasRolesOr(ctx, "user-1001", []string{"admin", "owner"})
hasPermission := dt.HasPermission(ctx, "user-1001", "order:read")
hasAllPermission := dt.HasPermissionsAnd(ctx, "user-1001", []string{"order:read", "order:write"})

_, _, _, _ = hasRole, hasAnyRole, hasPermission, hasAllPermission
```

## 在线状态控制

```go
_ = dt.KickoutByToken(ctx, token)
_ = dt.KickoutByLoginID(ctx, "user-1001")
_ = dt.ReplaceByToken(ctx, token)
_ = dt.ReplaceByLoginID(ctx, "user-1001")
```

## 封禁控制

```go
_ = dt.Disable(ctx, dt.DisableOptions{LoginID: "user-1001", Duration: time.Hour, Reason: "risk_control"})
disabled := dt.IsDisable(ctx, "user-1001")
disableInfo, err := dt.GetDisableInfo(ctx, "user-1001")
_ = dt.Untie(ctx, "user-1001")

_, _ = disabled, disableInfo
```

更多完整 API 见 [DToken API 文档](../../api/dtoken_zh.md)。
