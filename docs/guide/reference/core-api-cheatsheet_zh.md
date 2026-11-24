# 全局 API 速查

代码片段的导入别名：global = github.com/Zany2/dtoken-go/global.

本页整理 `global` API 的常用调用方式，适合在已经完成 Manager 初始化后快速查阅。

本页保留旧 global 调用速查。Build 自动注册，退出时先排空请求再关闭实例或调用 dtoken.CloseAll；BuildInstance 不自动注册。新应用的实例入口及签名对照见 [Auth API](../../api/auth_zh.md)；global.Login 的位置参数不能直接用于 auth.Login。

## 登录认证

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

## 角色权限

```go
_ = global.AddRoles(ctx, "user-1001", []string{"admin", "auditor"})
_ = global.AddPermissions(ctx, "user-1001", []string{"order:read", "order:*"})

hasRole := global.HasRole(ctx, "user-1001", "admin")
hasAnyRole := global.HasRolesOr(ctx, "user-1001", []string{"admin", "owner"})
hasPermission := global.HasPermission(ctx, "user-1001", "order:read")
hasAllPermission := global.HasPermissionsAnd(ctx, "user-1001", []string{"order:read", "order:write"})

_, _, _, _ = hasRole, hasAnyRole, hasPermission, hasAllPermission
```

## 在线状态控制

```go
_ = global.Kickout(ctx, token)
_ = global.KickoutByLoginID(ctx, "user-1001")
_ = global.Replace(ctx, token)
_ = global.ReplaceByLoginID(ctx, "user-1001")
```

## 封禁控制

```go
_ = global.Disable(ctx, "user-1001", time.Hour, "risk_control")
disabled := global.IsDisable(ctx, "user-1001")
disableInfo, err := global.GetDisableInfo(ctx, "user-1001")
_ = global.Untie(ctx, "user-1001")

_, _ = disabled, disableInfo
```

更多完整 API 见 [DToken API 文档](../../api/dtoken_zh.md)。
