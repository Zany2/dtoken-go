# 多认证体系

[English](multi-auth.md) | 中文文档

多认证体系用于在同一个项目里维护多套互相隔离的认证逻辑，例如普通用户、后台管理员、开放平台客户端。DToken-Go 通过 `AuthType` 区分不同认证体系。

## 核心概念

`AuthType` 会参与 Manager 注册、Token 查询、Session 查询、权限角色和存储 key 生成。同一个 Token 只能在所属的认证体系里被识别。

```text
dtoken:user:token:xxx
dtoken:admin:token:yyy
dtoken:app:session:client-001
```

`AuthType("user")` 会自动规范化为 `user:`，调用全局 API 时传入 `"user"` 或 `"user:"` 都可以。

## 注册多套 Manager

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

也可以使用 `BuildAndSetManager` 在构建时覆盖并注册 `AuthType`：

```go
_, err := dtoken.BuildAndSetManager(
    defaults.NewBuilder().
        KeyPrefix("dtoken").
        SetStorage(storage),
    "admin",
)
```

## 调用指定认证体系

全局 API 大多支持最后一个可选 `authType` 参数：

```go
token, err := dtoken.Login(ctx, "10001", "web", "chrome", "user")

loginID, err := dtoken.GetLoginID(ctx, token, "user")
err = dtoken.CheckPermission(ctx, "10001", "user:read", "user")
roles, err := dtoken.GetRoles(ctx, "10001", "user")
```

如果不传 `authType`，会使用默认认证体系 `auth:`。

## 框架中间件中使用

不同路由组可以绑定不同认证体系：

```go
r.Use(gindt.RegisterDTokenContextMiddleware(ctx))

userGroup := r.Group("/api")
userGroup.Use(gindt.AuthMiddleware(ctx, gindt.WithAuthType("user")))

adminGroup := r.Group("/admin")
adminGroup.Use(gindt.AuthMiddleware(ctx, gindt.WithAuthType("admin")))
adminGroup.Use(gindt.RoleMiddleware(ctx, []string{"admin"}, gindt.WithAuthType("admin")))
```

这样 `/api` 只能识别 `user` 体系的 Token，`/admin` 只能识别 `admin` 体系的 Token。

## Redis Key 隔离

Redis 存储 key 会包含 `KeyPrefix` 和 `AuthType`。如果两套认证体系共享 Redis，只要 `AuthType` 不同，核心登录态数据也不会串用。

```text
dtoken:user:token:...
dtoken:user:session:...
dtoken:admin:token:...
dtoken:admin:session:...
```

如果希望不同项目、环境或测试批次完全隔离，可以同时区分 `KeyPrefix`：

```go
defaults.NewBuilder().
    AuthType("admin").
    KeyPrefix("dtoken:prod")
```

## 事件与权限

事件数据会携带 `AuthType`，便于审计时区分来源：

```go
eventManager, _ := dtoken.GetEventManager("admin")
```

`AccessProvider` 的 `AccessSubject` 也会携带 `AuthType`，可以在同一个权限服务里按认证体系返回不同权限。

## 使用建议

- 普通用户和后台管理员建议使用不同 `AuthType`。
- 多租户不一定要使用 `AuthType`，如果租户共享一套认证规则，租户 ID 更适合放在业务权限或账号模型里。
- 多套认证体系可以共享同一个 Redis，但生产环境建议设置清晰的 `KeyPrefix`。
- 不要让不同认证体系复用同一个 `TokenName`，除非路由隔离非常明确。

## 相关文档

- [配置指南](configuration_zh.md)
- [Redis 存储](redis-storage_zh.md)
- [框架集成](framework-integration_zh.md)
- [AccessProvider](access-provider_zh.md)
