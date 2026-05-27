# AccessProvider

[English](access-provider.md) | 中文文档

`AccessProvider` 用于从业务系统动态解析权限和角色。它适合权限数据保存在数据库、RBAC 服务、配置中心或第三方权限平台的场景。

## 和 Session 权限的关系

DToken-Go 支持两种权限来源：

1. 写入 Session 的权限和角色，例如 `AddPermissions`、`AddRoles`。
2. 通过 `AccessProvider` 或兼容回调动态返回权限和角色。

当配置了 `AccessProvider` 时，Provider 返回的非 `nil` 数据优先于 Session 中缓存的数据。Provider 返回 `nil` 时，才会回退到 Session 数据。

## AccessSubject

Provider 会收到一个 `AccessSubject`：

```go
type AccessSubject struct {
    AuthType string
    LoginID  string
    Device   string
    DeviceID string
    Token    string
}
```

这些字段可以用于按认证体系、账号、设备类型、具体设备或 Token 维度返回不同权限。

## 基本用法

```go
provider := manager.AccessProviderFunc{
    PermissionFunc: func(ctx context.Context, subject manager.AccessSubject) ([]string, error) {
        if subject.AuthType == "admin:" {
            return []string{"admin:*"}, nil
        }
        return []string{"user:read"}, nil
    },
    RoleFunc: func(ctx context.Context, subject manager.AccessSubject) ([]string, error) {
        if subject.LoginID == "10001" {
            return []string{"admin"}, nil
        }
        return []string{"user"}, nil
    },
}

mgr, err := defaults.NewBuilder().
    SetStorage(storage).
    SetAccessProvider(provider).
    Build()
```

## 回退到 Session

Provider 返回 `nil` 表示不接管本次查询，让框架继续使用 Session 中的数据：

```go
provider := manager.AccessProviderFunc{
    PermissionFunc: func(ctx context.Context, subject manager.AccessSubject) ([]string, error) {
        if subject.LoginID == "" {
            return nil, nil
        }
        return nil, nil // 回退到 Session 权限
    },
}
```

如果返回空切片 `[]string{}`，表示明确没有任何权限，不会回退。

## 错误处理

Provider 返回错误时，权限或角色解析会失败。用于布尔判断的 `Has*` 方法会按安全拒绝处理，避免权限服务异常时放行请求。

```go
PermissionFunc: func(ctx context.Context, subject manager.AccessSubject) ([]string, error) {
    return nil, errors.New("permission service unavailable")
}
```

## 兼容回调

Builder 也保留了更简单的回调方式：

```go
mgr, err := defaults.NewBuilder().
    SetStorage(storage).
    SetCustomPermissionListFunc(func(loginID, authType string) ([]string, error) {
        return []string{"user:read"}, nil
    }).
    SetCustomRoleListFunc(func(loginID, authType string) ([]string, error) {
        return []string{"user"}, nil
    }).
    SetCustomPermissionListExtFunc(func(loginID, device, deviceID, authType string) ([]string, error) {
        if device == "app" {
            return []string{"mobile:read"}, nil
        }
        return nil, nil
    }).
    Build()
```

新项目更推荐直接使用 `SetAccessProvider`，表达能力更完整。

## 和多认证体系搭配

`AccessSubject.AuthType` 会携带当前认证体系：

```go
PermissionFunc: func(ctx context.Context, subject manager.AccessSubject) ([]string, error) {
    switch subject.AuthType {
    case "admin:":
        return loadAdminPermissions(subject.LoginID)
    case "user:":
        return loadUserPermissions(subject.LoginID)
    default:
        return nil, nil
    }
}
```

## 使用建议

- 权限变更频繁时，优先使用 `AccessProvider` 动态查询。
- 权限较稳定、只依赖登录态时，可以写入 Session。
- Provider 查询外部服务时要设置超时，避免阻塞认证链路。
- Provider 返回 `nil` 和 `[]string{}` 含义不同，需要明确区分。
- 生产环境建议对权限服务异常做监控和告警。

## 相关文档

- [权限管理](permission_zh.md)
- [多认证体系](multi-auth_zh.md)
- [框架集成](framework-integration_zh.md)
