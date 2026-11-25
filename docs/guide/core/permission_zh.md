# 权限管理

[English](../core/permission.md) | 中文文档

本文 `dt` 表示根包。`Build()` 自动注册；默认体系使用 `dt.XXX()`，其他体系使用 `dt.AuthType("user").XXX()`，业务方法签名与实例一致。见 [Auth API](../../api/auth_zh.md)。

## 概览

权限和角色来自 Session 或 `SetAccessProvider` 注入的业务提供器。配置 Provider 时，其非 nil 结果优先；nil 回退到 Session，空切片表示没有权限或角色。Provider 错误由 Check/Get 方法返回，Has 方法返回 false。

按账号查询提供认证体系和 loginID；按 Token 查询还包含设备与 Token 信息。旧 CustomPermissionList/CustomRoleList 回调入口已移除，使用 AccessProviderFunc 迁移。

## 初始化

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

## 权限管理

按 Token 添加或删除权限、角色时，即使列表为 `nil`、空列表或仅含空字符串，也会校验 Token 的完整登录态。无效、封禁或活跃超时的 Token 会返回对应错误；有效登录的空变更不写入 Session、不触发变更事件，也不安排续期或活跃维护任务。

### 添加权限

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

### 删除权限

```go
ctx := context.Background()

_ = dt.RemovePermissions(ctx, dt.PermissionOptions{LoginID: "10001", Permissions: []string{"user:write"}})
_ = dt.RemovePermissionsByToken(ctx, token, []string{"article:publish"})
```

### 查询权限

```go
ctx := context.Background()

permissions, err := dt.GetPermissions(ctx, "10001")
permissionsByToken, err := dt.GetPermissionsByToken(ctx, token)
```

## 权限校验

### 单个权限

```go
ctx := context.Background()

hasPermission := dt.HasPermission(ctx, "10001", "user:read")
hasPermissionByToken := dt.HasPermissionByToken(ctx, token, "user:read")

err := dt.CheckPermission(ctx, dt.PermissionOptions{LoginID: "10001", Permission: "user:read"})
```

### 多权限 AND

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

### 多权限 OR

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

_ = dt.AddPermissions(ctx, dt.PermissionOptions{LoginID: "10001", Permissions: []string{
    "admin:*",
    "user:*:view",
}})

dt.HasPermission(ctx, "10001", "admin:read")        // true
dt.HasPermission(ctx, "10001", "admin:delete")      // true
dt.HasPermission(ctx, "10001", "user:profile:view") // true
dt.HasPermission(ctx, "10001", "user:view")         // false
```

## 角色管理

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

## 封禁管理

### 账号封禁

```go
ctx := context.Background()

_ = dt.Disable(ctx, dt.DisableOptions{LoginID: "10001", Duration: 2*time.Hour, Reason: "abuse"})

disabled := dt.IsDisable(ctx, "10001")
err := dt.CheckDisable(ctx, "10001")

info, err := dt.GetDisableInfo(ctx, "10001")
ttl, err := dt.GetDisableTTL(ctx, "10001")

_ = dt.Untie(ctx, "10001")
```

`GetDisableTTL()` 返回值约定：

- `-2`：未封禁
- `-1`：永久封禁
- `>0`：剩余秒数

### 服务级封禁

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

## 自定义权限与角色提供器

下面配置并注册使用自定义提供器的实例。请用它替换前面的初始化，避免重复注册默认体系。PermissionFunc 与 RoleFunc 使用同一个 AccessSubject，不再区分普通和 Ext 回调。

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

Build 自动注册实例，前文根包调用可直接使用；用于 HTTP 时通过 WithAuth(auth) 注入同一个实例。详见 [AccessProvider](access-provider_zh.md)。

## Gin 路由中使用

```go
ctx := context.Background()

auth, err := dt.NewBuilder().Build()
if err != nil { panic(err) }
defer auth.Close()

r.Use(gindt.RegisterDTokenContextMiddleware(ctx, gindt.WithAuth(auth)))

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

- [登录认证](../core/authentication_zh.md)
- [注解鉴权](../integration/annotation_zh.md)
- [AccessProvider](../core/access-provider_zh.md)
- [封禁体系](../core/disable_zh.md)
- [JWT 指南](../security/jwt_zh.md)
