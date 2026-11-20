# 多认证体系

代码片段的导入别名：dt = github.com/Zany2/dtoken-go, global = github.com/Zany2/dtoken-go/global.

[English](../core/multi-auth.md) | 中文文档

多认证体系用于在同一个项目里维护多套互相隔离的认证逻辑，例如普通用户、后台管理员、开放平台客户端。DToken-Go 通过 `AuthType` 区分不同认证体系。

多实例本身不要求全局注册：分别构建 userAuth、adminAuth，并通过 WithAuth 注入即可。下文演示额外启用注册表的方式，使用 global 表示全局包、dt 表示根包。

## 核心概念

`AuthType` 会参与 Manager 注册、Token 查询、Session 查询、权限角色和存储 key 生成。同一个 Token 只能在所属的认证体系里被识别。

```text
dtoken:user:token:xxx
dtoken:admin:token:yyy
dtoken:app:session:client-001
```

`AuthType("user")` 会自动规范化为 `user:`，调用全局 API 时传入 `"user"` 或 `"user:"` 都可以。

## 注册多套 Manager

<!-- dtoken-compile: multi-auth -->
```go
package main

import (
	"context"

	dt "github.com/Zany2/dtoken-go"
	global "github.com/Zany2/dtoken-go/global"
	gindt "github.com/Zany2/dtoken-go/integrations/gin"
	"github.com/Zany2/dtoken-go/storage/memory"
	"github.com/gin-gonic/gin"
)

func main() {
	defer global.DeleteAllManager()
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
	global.SetAuth(userAuth)

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
	global.SetAuth(adminAuth)

	userToken, _ := global.Login(ctx, "10001", "", "", "user")
	adminToken, _ := global.Login(ctx, "admin-1", "", "", "admin")
	_, _ = userToken, adminToken
	registerRoutes(ctx, gin.New(), userAuth, adminAuth)
}
```

Manager 默认持有并关闭通过 Builder 配置的 Storage、Logger 和 Pool。`ComponentOwnership` 是完整的所有权策略，所有字段都应显式设置，因为未填写的布尔字段值为 `false`。共享组件时，将对应字段设为 `false`；共享组件随后由调用方在应用退出时统一关闭。由调用方持有的 Pool 必须保持运行，直到使用它的全部 Manager 都已关闭，因为每个 Manager 在关闭时都会等待自己已接收的异步任务。

应先停止接收新业务请求、等待执行中的同步调用结束，再由应用外部关闭流程调用 `Auth.Close()`；高级 Manager 入口为 `CloseManager()`。不要在 Manager 任务或事件监听器内部同步调用它：关闭过程会等待已接收的任务和监听器，可能包含调用者自身。回调需要发起关闭时，应通知应用生命周期控制方后返回。关闭后不再支持业务调用；`CloseManager` 不负责排空业务请求。

也可以使用 `BuildAndSetManager` 在构建时覆盖并注册 `AuthType`：

```go
_, err := global.BuildAndSetManager(
    dt.NewBuilder().
        KeyPrefix("dtoken").
        Storage(storage).
        UseManagerOption(sharedOwnership),
    "admin",
)
```

## 调用指定认证体系

全局 API 大多支持最后一个可选 `authType` 参数：

```go
token, err := global.Login(ctx, "10001", "web", "chrome", "user")

loginID, err := global.GetLoginID(ctx, token, "user")
err = global.CheckPermission(ctx, "10001", "user:read", "user")
roles, err := global.GetRoles(ctx, "10001", "user")
```

如果不传 `authType`，会使用默认认证体系 `auth:`。

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

各路由组通过 `WithAuth` 显式绑定实例，与全局注册无关。若需要按认证类型动态查询注册表，应显式配置 `WithManagerResolver(global.GetManager)`，再使用 `WithAuthType` 选择认证类型。

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
eventManager, _ := global.GetEventManager("admin")
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
