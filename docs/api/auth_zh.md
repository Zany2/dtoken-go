# Auth 与根包门面 API

[English](auth.md) | 中文文档

根包 `github.com/Zany2/dtoken-go` 是应用首选入口。`NewBuilder().Build()` 自动注册并返回 `*dtoken.Auth`；`BuildInstance()` 创建不注册的独立实例。完整初始化示例见[快速开始](../tutorial/quick-start_zh.md)。

## 三种调用方式

| 调用形式 | 选择的引擎 |
| --- | --- |
| `auth.CheckLogin(ctx, token)` | 构建返回的固定实例 |
| `dtoken.CheckLogin(ctx, token)` | 默认体系 `auth:` 当前注册的实例 |
| `dtoken.AuthType("user").CheckLogin(ctx, token)` | user 体系当前注册的实例 |

根包包级业务方法与 Auth 方法签名一致，例如 `dtoken.Login(ctx, dtoken.LoginOptions{LoginID: "10001"})`。省略 Builder.AuthType 注册默认体系；指定名称的体系不存在时返回 ErrManagerNotFound，布尔检查返回 false，不会回退到默认体系。选择器在每次操作时查找注册表，固定实例不会跟随替换。

Build / MustBuild 自动注册，同名 Build 返回 errors.ErrManagerAlreadyRegistered，MustBuild 失败时 panic。BuildInstance / MustBuildInstance 不注册，允许同名独立实例。显式替换使用 BuildInstance 后调用 dtoken.SetAuth，替换会关闭原引擎。普通 Build 后无需 SetAuth。

## 入口与选项

| 操作 | 实例调用 |
| --- | --- |
| 按账号登录 | `auth.LoginID(ctx, loginID)` |
| 设置设备、有效期和扩展数据后登录 | `auth.Login(ctx, dtoken.LoginOptions{...})` |
| 登录态校验 | `auth.CheckLogin(ctx, token)` / `auth.IsLogin(ctx, token)` |
| 获取账号 | `auth.GetLoginID(ctx, token)` |
| 无续期副作用检查 | `auth.IntrospectToken(ctx, token)` |
| 按 Token 登出、踢下线、顶下线 | `auth.LogoutByToken` / `auth.KickoutByToken` / `auth.ReplaceByToken` |
| 按条件下线 | `auth.Logout` / `auth.Kickout` / `auth.Replace`，传入 `dtoken.LogoutOptions` |
| 权限变更与单项检查 | `auth.AddPermissions` / `auth.RemovePermissions` / `auth.CheckPermission`，传入 `dtoken.PermissionOptions` |
| 角色变更与单项检查 | `auth.AddRoles` / `auth.RemoveRoles` / `auth.CheckRole`，传入 `dtoken.RoleOptions` |
| 布尔权限、角色查询 | `auth.HasPermission(ctx, loginID, permission)` / `auth.HasRole(ctx, loginID, role)` |
| 账号、服务、设备封禁 | `auth.Disable` / `auth.DisableService` / `auth.DisableDevice`，使用对应类型化选项 |
| Session 查询与持久化 | `auth.GetSession` / `auth.SetSessionValue` / `auth.GetSessionValue` |
| 刷新令牌 | `auth.LoginWithRefreshToken` / `auth.LoginWithRefreshTokenOptions` / `auth.RefreshToken` |

实例方法使用自身认证体系；选项中的 `AuthType` 不会让实例切换到其他引擎。要使用另一个认证体系，可调用另一个 Auth，或通过 dtoken.AuthType(name) 选择。该选项字段为源码兼容保留，不参与体系选择。

## 登录与权限示例

以下片段假设已构建 `auth`，并已验证用户身份。片段中的 `ctx` 来自当前请求。

```go
token, err := auth.Login(ctx, dtoken.LoginOptions{
    LoginID: "10001",
    Device: "web",
    DeviceID: "browser-1",
})
if err != nil {
    return err
}
if err := auth.AddPermissions(ctx, dtoken.PermissionOptions{
    LoginID: "10001", Permissions: []string{"user:read"},
}); err != nil {
    return err
}
if err := auth.CheckPermission(ctx, dtoken.PermissionOptions{
    Token: token, Permission: "user:read",
}); err != nil {
    return err
}
```

## 根包与实例调用

根包函数与实例方法使用相同的业务签名。默认体系使用 `dt.Login(ctx, dt.LoginOptions{LoginID: "10001"})`，指定体系使用 `dt.AuthType("user").Login(ctx, dt.LoginOptions{LoginID: "10001"})`，固定实例使用 `auth.Login(ctx, dt.LoginOptions{LoginID: "10001"})`。这里 dt 表示根包。原 global 包已移除，位置参数调用需要迁移到这些签名。详见[根包 API 参考](dtoken_zh.md)。

HTTP 通过 WithAuth(auth) 或显式 WithManagerResolver(dt.GetManager) 接入。请求上下文的分组 API 使用 Manager 选项类型，注意区分根 LoginOptions 与 ManagerLoginOptions。

## 生命周期与扩展

- `auth.Config()` 返回配置快照；修改快照不会改变引擎。
- `auth.Storage()` 和 `auth.Codec()` 用于借用组件、组合独立能力，使用期间保持 Auth 可用。
- `auth.Manager()` 暴露受支持的高级 API，从根包引用其类型，不导入 `internal`。
- `dtoken.New(auth.Manager())` 包装同一个引擎，关闭任一门面会影响全部共享者。
- `auth.Close()` 注销仍指向自身的注册项并释放自有资源；`dtoken.Close()` 关闭默认体系，`dtoken.AuthType(name).Close()` 关闭指定体系，`dtoken.CloseAll()` 关闭全部当前注册实例。未注册实例由调用方单独关闭；先排空 HTTP 请求。
- Nonce、OAuth2、Ticket 和 ShortKey 从各自 Service 调用，通过 `WithCapabilities` 接入 HTTP；它们不是 Auth 方法。

详见 [API 与生命周期](../design/api-lifecycle_zh.md)、[独立能力](../guide/security/advanced-features_zh.md)和[框架集成](../guide/integration/framework-integration_zh.md)。
