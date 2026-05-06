# 权限管理

[English](permission.md) | 中文文档

## 概览

当前版本的权限与角色数据主要有两种来源：

1. `Session` 中维护的 `Permissions`、`Roles`
2. 通过 `builder.NewBuilder()` 注入的自定义回调

优先级方面：

- 按 `loginID` 查询时：`CustomPermissionListFunc` / `CustomRoleListFunc` 优先于 Session
- 按 `token` 查询时：`CustomPermissionListExtFunc` / `CustomRoleListExtFunc` 优先，其次普通回调，最后才是 Session

## 初始化

```go
package main

import (
    "context"

    "github.com/Zany2/dtoken-go/com/storage/memory"
    "github.com/Zany2/dtoken-go/core/builder"
    "github.com/Zany2/dtoken-go/dtoken"
)

func initDToken() {
    dtoken.SetManager(
        builder.NewBuilder().
            SetStorage(memory.NewStorage()).
            Build(),
    )
}
```

## 权限管理

### 添加权限

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

### 删除权限

```go
ctx := context.Background()

_ = dtoken.RemovePermissions(ctx, "10001", []string{"user:write"})
_ = dtoken.RemovePermissionsByToken(ctx, token, []string{"article:publish"})
```

### 查询权限

```go
ctx := context.Background()

permissions, err := dtoken.GetPermissions(ctx, "10001")
permissionsByToken, err := dtoken.GetPermissionsByToken(ctx, token)
```

## 权限校验

### 单个权限

```go
ctx := context.Background()

hasPermission := dtoken.HasPermission(ctx, "10001", "user:read")
hasPermissionByToken := dtoken.HasPermissionByToken(ctx, token, "user:read")

err := dtoken.CheckPermission(ctx, "10001", "user:read")
```

### 多权限 AND

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

### 多权限 OR

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

## 权限通配符

当前权限匹配支持 `*` 通配符，并按分段匹配：

| 模式 | 说明 | 示例 |
|------|------|------|
| `*` | 匹配全部权限 | 任意权限 |
| `user:*` | 匹配两段式 `user` 权限 | `user:read`、`user:write` |
| `user:*:view` | 匹配三段式权限 | `user:profile:view` |
| `admin/*` | 也支持 `/` 作为分隔符 | `admin/read` |

当前实现的两个关键点：

1. 分隔符会自动根据权限模式识别，优先使用 `:`，若模式里包含 `/` 则使用 `/`
2. 分段数量必须一致，避免因为通配过宽造成越权

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

## 角色管理

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

## 封禁管理

### 账号封禁

```go
ctx := context.Background()

_ = dtoken.Disable(ctx, "10001", 2*time.Hour, "abuse")

disabled := dtoken.IsDisable(ctx, "10001")
err := dtoken.CheckDisable(ctx, "10001")

info, err := dtoken.GetDisableInfo(ctx, "10001")
ttl, err := dtoken.GetDisableTTL(ctx, "10001")

_ = dtoken.Untie(ctx, "10001")
```

`GetDisableTTL()` 返回值约定：

- `-2`：未封禁
- `-1`：永久封禁
- `>0`：剩余秒数

### 服务级封禁

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

## 自定义权限与角色回调

```go
dtoken.SetManager(
    builder.NewBuilder().
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

扩展回调还支持拿到 `device`、`deviceId`：

```go
builder.NewBuilder().
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

## Gin 路由中使用

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

## 最佳实践

1. 先登录，再维护角色和权限，避免因为 Session 不存在导致写入失败
2. 业务权限变化频繁时，优先使用自定义回调而不是长期缓存到 Session
3. 对外接口建议使用 `Check*`，内部逻辑判断建议使用 `Has*`
4. 服务级封禁适合评论、发帖、支付等细粒度能力，不必总是封禁整个账号

## 相关文档

- [登录认证](authentication_zh.md)
- [注解鉴权](annotation_zh.md)
- [JWT 指南](jwt_zh.md)
