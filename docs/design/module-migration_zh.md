# 模块组织与迁移

[English](module-migration.md) | 中文文档

## 当前边界

根模块发布实例门面、公共契约、内部认证实现、基础组件、共享 HTTP 鉴权，以及 Nonce、OAuth2、Ticket、ShortKey 独立能力包。能力包显式组合，不是单独的 Go 模块。工作区共 30 个模块。

框架集成、codec/jsonv2、codec/msgpack、logging/gf、storage/redis、sso 和 sso/storage/redis 独立发布。示例与应用测试保留独立模块。

## 包路径迁移

这是完整的包边界和 API 迁移，旧路径不再作为兼容入口。

| Previous entry | Current entry |
| --- | --- |
| dtoken subdirectory | Root instance API and explicit global package |
| core/adapter | contract |
| core/config, core/derror, core/listener | config, errors, event |
| core/manager, core/builder, defaults | Root NewBuilder and public extension types |
| core/context, integrations/authcheck | httpauth/context, httpauth |
| core/nonce, core/oauth2, core/ticket, core/shortkey | nonce, oauth2, ticket, shortkey |
| com/codec, com/generator, com/log, com/pool, com/storage | codec, generator, logging, pool, storage |

## 应用构建

```go
import (
    "github.com/Zany2/dtoken-go"
    "github.com/Zany2/dtoken-go/global"
)

auth, err := dtoken.NewBuilder().AuthType("user").Build()
if err != nil {
    panic(err)
}
defer auth.Close()

// Optional global registration. 可选全局注册。
global.SetAuth(auth)
defer global.DeleteManager("user")
```

Build() 返回 *dtoken.Auth；auth.Manager() 提供底层管理器。框架优先使用 WithAuth(auth) 或 Context(auth)。策略、装配选项和工厂类型从根包导入，不要导入 internal。所有框架 NewBuilder() 均转发根包构建器，Build() 统一返回 Auth。框架不再转导出全局 API 或组件构造器。HTTP 中间件默认要求绑定实例；全局使用须显式配置 WithManagerResolver(global.GetManager)。

## 独立能力

Builder 不提供能力开关。通过各自的构造器创建能力 Manager，再通过 NewService(manager, auth) 接入实例事件。使用所属认证实例的 AuthType、KeyPrefix、Storage 和 Codec。httpauth/context.Capabilities 和框架 WithCapabilities 将服务绑定到请求。未绑定的能力返回 ErrModuleNotEnabled。同一 Manager 的后续中间件未指定覆盖配置时保留请求绑定；切换 Manager 时创建新作用域。

## 依赖与发布

清理旧子模块依赖和替换规则，再添加根模块及所需扩展的新版本。根模块使用 vX.Y.Z tag，扩展使用当前目录前缀，例如 integrations/gin/vX.Y.Z 和 storage/redis/vX.Y.Z。源码中的 v0.0.3 依赖基线不能证明新结构已远程发布。先发布被依赖模块，再更新并发布扩展。运行时版本位于 contract/version.go。

## 验证

维护者显式执行以下命令。独立模块检查将模块复制到仓库外并排除嵌套模块，通过 GOWORK=off 和临时替换验证当前源码；消费者检查在仓库外创建临时模块，覆盖实例、公共扩展类型、全局注册、HTTP 能力绑定与退出。

```bash
go run ./.github/scripts/check-modules
go run ./.github/scripts/check-docs
bash .github/scripts/check-consumer.sh
bash .github/scripts/check-consumer.sh "$ROOT_VERSION" "$GIN_VERSION"
```

最后一个命令需同时传入两个已发布版本，并核对远程解析后的选中版本。JSON v2 需要 GOEXPERIMENT=jsonv2。Redis CI 覆盖根模块、storage/redis、sso/storage/redis 和 tests/gin_core_flow。

公开 API 定位、组件所有权及注册表替换规则见 [API 与生命周期](api-lifecycle_zh.md)。HTTP 能力绑定依赖 contract 下的服务接口，不依赖具体实现。
