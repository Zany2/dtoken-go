# 多认证体系

代码片段的导入别名：dt = github.com/Zany2/dtoken-go.

[English](../core/multi-auth.md) | 中文文档

多认证体系用于在同一个项目里维护多套互相隔离的认证逻辑，例如普通用户、后台管理员、开放平台客户端。DToken-Go 通过 `AuthType` 区分不同认证体系。

Build 自动注册并返回 Auth，可直接调用返回实例、使用 dt.AuthType(name) 选择体系，或通过 dt 包级方法调用默认体系。BuildInstance 用于创建不注册的独立实例。

## 核心概念

`AuthType` 会参与 Manager 注册、Token 查询、Session 查询、权限角色和存储 key 生成。同一个 Token 只能在所属的认证体系里被识别。

```text
dtoken:user:token:xxx
dtoken:admin:token:yyy
dtoken:app:session:client-001
```

`AuthType("user")` 会自动规范化为 `user:`，构建、`dt.AuthType(name)` 和 `dt.GetManager(name)` 都接受 `"user"` 或 `"user:"`。`TokenName` 决定 HTTP 读取 Token 的字段名，存储命名空间由 `KeyPrefix` 与 `AuthType` 决定。

## 构建并注册多个体系

<!-- dtoken-compile: multi-auth -->
```go
package main

import (
	"context"

	dt "github.com/Zany2/dtoken-go"
	gindt "github.com/Zany2/dtoken-go/integrations/gin"
	"github.com/Zany2/dtoken-go/storage/memory"
	"github.com/gin-gonic/gin"
)

func main() {
	defer dt.CloseAll()
	ctx := context.Background()
	storage := memory.NewStorage()
	sharedOwnership := dt.WithComponentOwnership(dt.ComponentOwnership{
		Storage: false,
		Logger:  true,
		Pool:    true,
	})

	userAuth, err := dt.NewBuilder().
		AuthType("user").
		KeyPrefix("dtoken").
		TokenName("user-token").
		Storage(storage).
		UseManagerOption(sharedOwnership).
		Build()
	if err != nil {
		panic(err)
	}

	adminAuth, err := dt.NewBuilder().
		AuthType("admin").
		KeyPrefix("dtoken").
		TokenName("admin-token").
		Storage(storage).
		UseManagerOption(sharedOwnership).
		Build()
	if err != nil {
		panic(err)
	}

	userToken, _ := dt.AuthType("user").LoginID(ctx, "10001")
	adminToken, _ := adminAuth.LoginID(ctx, "admin-1")
	_, _ = userToken, adminToken
	registerRoutes(ctx, gin.New(), userAuth, adminAuth)
}
```

Manager 默认持有并关闭通过 Builder 配置的 Storage、Logger 和 Pool。`ComponentOwnership` 是完整的所有权策略，所有字段都应显式设置，因为未填写的布尔字段值为 `false`。共享组件时，将对应字段设为 `false`；共享组件随后由调用方在应用退出时统一关闭。由调用方持有的 Pool 必须保持运行，直到使用它的全部 Manager 都已关闭，因为每个 Manager 在关闭时都会等待自己已接收的异步任务。

应先停止接收新业务请求、等待执行中的同步调用结束，再由应用外部关闭流程调用 `Auth.Close()`；高级 Manager 入口为 `CloseManager()`。不要在 Manager 任务或事件监听器内部同步调用它：关闭过程会等待已接收的任务和监听器，可能包含调用者自身。回调需要发起关闭时，应通知应用生命周期控制方后返回。关闭后不再支持业务调用；`CloseManager` 不负责排空业务请求。

同名 Build 返回 `errors.ErrManagerAlreadyRegistered`，不会替换已有实例。需要主动替换时使用 `BuildInstance()` 构建，再调用 `dt.SetAuth(auth)`。

## 调用指定认证体系

先选择体系，再调用与实例一致的方法：

```go
user := dt.AuthType("user")
token, err := user.Login(ctx, dt.LoginOptions{LoginID: "10001", Device: "web", DeviceID: "chrome"})

loginID, err := user.GetLoginID(ctx, token)
err = user.CheckPermission(ctx, dt.PermissionOptions{LoginID: "10001", Permission: "user:read"})
roles, err := user.GetRoles(ctx, "10001")
```

直接调用 dt.CheckLogin 等包级方法使用默认体系 `auth:`。上面的示例只构建 user 和 admin，没有注册默认体系；如需默认调用，另用未设置 AuthType 的 Builder 构建。

| 入口 | 选择行为 |
| --- | --- |
| `dt.CheckLogin(ctx, token)` | 查找默认体系 `auth:` |
| `dt.AuthType("user").CheckLogin(ctx, token)` | 每次操作查找当前 user 引擎 |
| `userAuth.CheckLogin(ctx, token)` | 使用构建返回的固定引擎 |

选择不存在的体系时，CheckLogin 返回 `ErrManagerNotFound`，IsLogin 返回 false，不会回退默认体系。选择器可以缓存，并会跟随显式替换；构建返回的固定实例不会。业务选项中的 `AuthType` 字段不参与路由，体系由上述调用入口确定。

## 独立实例与显式替换

`BuildInstance()` / `MustBuildInstance()` 不访问注册表，适用于测试、依赖注入或需要同名独立引擎的场景；持有者单独调用 `auth.Close()`。同名未注册实例不会自动隔离共享后端中的数据，仍须正确配置 AuthType、KeyPrefix 或独立存储。

需要替换已注册体系时，在停止旧引擎的新请求并排空已有请求后执行：

```go
replacement, err := dt.NewBuilder().AuthType("user").BuildInstance()
if err != nil {
    panic(err)
}
dt.SetAuth(replacement)
```

SetAuth 注册新引擎并关闭旧引擎；已有固定 Auth 和 WithAuth 绑定仍引用旧引擎，不能继续使用。`dt.CloseAll()` 关闭全部当前注册实例，不包含未注册实例，也不禁止随后重新构建。完整约定见 [API 与生命周期](../../design/api-lifecycle_zh.md)。

## 框架中间件中使用

不同路由组可以绑定不同认证体系：

<!-- dtoken-compile: multi-auth -->
```go
func registerRoutes(ctx context.Context, r *gin.Engine, userAuth, adminAuth *dt.Auth) {
	userGroup := r.Group("/api")
	userGroup.Use(gindt.AuthMiddleware(ctx, gindt.WithAuth(userAuth)))

	adminGroup := r.Group("/admin")
	adminGroup.Use(gindt.AuthMiddleware(ctx, gindt.WithAuth(adminAuth)))
	adminGroup.Use(gindt.RoleMiddleware(ctx, []string{"admin"}, gindt.WithAuth(adminAuth)))
}
```

这样 `/api` 只能识别 `user` 体系的 Token，`/admin` 只能识别 `admin` 体系的 Token。

各路由组通过 `WithAuth` 显式绑定实例，与全局注册无关。若需要按认证类型动态查询注册表，应显式配置 `WithManagerResolver(dt.GetManager)`，再使用 `WithAuthType` 选择认证类型。

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
dt.NewBuilder().
    AuthType("admin").
    KeyPrefix("dtoken:prod")
```

## 事件与权限

事件数据会携带 `AuthType`，便于审计时区分来源：

```go
eventManager, _ := dt.GetEventManager("admin")
```

`AccessProvider` 的 `AccessSubject` 也会携带 `AuthType`，可以在同一个权限服务里按认证体系返回不同权限。

## 使用建议

- 普通用户和后台管理员建议使用不同 `AuthType`。
- 多租户不一定要使用 `AuthType`，如果租户共享一套认证规则，租户 ID 更适合放在业务权限或账号模型里。
- 多套认证体系可以共享同一个 Redis，但生产环境建议设置清晰的 `KeyPrefix`。
- 不要让不同认证体系复用同一个 `TokenName`，除非路由隔离非常明确。

## 相关文档

- [配置指南](../reference/configuration_zh.md)
- [Redis 存储](../integration/redis-storage_zh.md)
- [框架集成](../integration/framework-integration_zh.md)
- [AccessProvider](../core/access-provider_zh.md)
