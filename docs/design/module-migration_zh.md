# 模块组织与迁移

[English](module-migration.md) | 中文文档

本文维护旧路径与 API 的升级步骤、发布顺序和消费者验证方式。当前系统分层见[架构设计](architecture_zh.md)，包与模块边界见[模块化设计](modular_zh.md)，代码位置见[源码目录](source-layout_zh.md)。

## 当前边界

根模块发布实例门面、公共契约、内部认证实现、基础组件、共享 HTTP 鉴权，以及 Nonce、OAuth2、Ticket、ShortKey 独立能力包。能力包显式组合，不是单独的 Go 模块。工作区共 30 个模块。

框架集成、codec/jsonv2、codec/msgpack、logging/gf、storage/redis、sso 和 sso/storage/redis 独立发布。示例与应用测试保留独立模块。

## 包路径迁移

这是完整的包边界和 API 迁移，旧路径不再作为兼容入口。

| Previous entry | Current entry |
| --- | --- |
| dtoken subdirectory | Root instance/default APIs and AuthType selectors; legacy global compatibility |
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
)

auth, err := dtoken.NewBuilder().AuthType("user").Build()
if err != nil {
    panic(err)
}
defer auth.Close()

// Build registers automatically. Build 自动注册。
selected := dtoken.AuthType("user")
_ = selected
```

Build() 自动注册并返回 *dtoken.Auth；auth.Manager() 提供底层管理器。框架优先使用 WithAuth(auth) 或 Context(auth)。策略、装配选项和工厂类型从根包导入，不要导入 internal。所有框架 NewBuilder() 均转发根包构建器，Build() 统一返回 Auth。框架不再转导出全局 API 或组件构造器。HTTP 中间件默认要求绑定实例；全局使用须显式配置 WithManagerResolver(dtoken.GetManager)。

同名 Build 返回 ErrManagerAlreadyRegistered；同名独立实例改用 BuildInstance。显式替换使用 BuildInstance 后 SetAuth。默认包级调用与 AuthType 选择器共用根注册表，旧 global 调用仍可用。退出时先排空请求，再 CloseAll。详见 [API 与生命周期](api-lifecycle_zh.md)。

## 权限提供器迁移

旧 SetCustomPermissionListFunc、SetCustomRoleListFunc 及 Ext 变体不再提供。使用根 Builder.SetAccessProvider 注入 AccessProviderFunc，通过 AccessSubject 取得认证体系、账号、设备与 Token；nil 结果回退到 Session，非 nil 空切片表示无权限。见 [AccessProvider](../guide/core/access-provider_zh.md)。

## 独立能力

Builder 不提供能力开关。通过各自的构造器创建能力 Manager，再通过 NewService(manager, auth) 接入实例事件。使用所属认证实例的 AuthType、KeyPrefix、Storage 和 Codec。httpauth/context.Capabilities 和框架 WithCapabilities 将服务绑定到请求。未绑定的能力返回 ErrModuleNotEnabled。同一 Manager 的后续中间件未指定覆盖配置时保留请求绑定；切换 Manager 时创建新作用域。

## 门面方法迁移

框架包保留 `NewBuilder` 及 `LoginByContext`、`LoginByCtx` 等请求快捷方法。`Auth` 和认证业务选项类型从根包导入。原框架包级的 `Default`、`NewByAuthType`、`SetManager`、`Login` 等全局调用迁移到 `global` 对应入口；认证操作也可直接使用构建完成的 `Auth` 实例。组件构造器从所属组件包导入。

下表覆盖旧 `Auth` 门面移出的 40 个能力方法。原包级能力调用也迁移到相同服务入口；根据目标认证实例选择已组合的服务，不再通过末尾的 `authType` 参数选择实例。从当前根路径下的 `nonce`、`oauth2`、`ticket`、`shortkey` 包导入能力类型。构建 `auth` 后，表中的服务变量按以下方式组合：

```go
cfg := auth.Config()
nonceService := nonce.NewService(nonce.NewDefaultNonceManager(cfg.AuthType, cfg.KeyPrefix, auth.Storage()), auth)
oauth2Service := oauth2.NewService(oauth2.NewDefaultOAuth2Server(cfg.AuthType, cfg.KeyPrefix, auth.Storage(), auth.Codec()), auth)
ticketService := ticket.NewService(ticket.NewDefaultManager(cfg.AuthType, cfg.KeyPrefix, auth.Storage(), auth.Codec()), auth)
shortKeyService := shortkey.NewService(shortkey.NewDefaultManager(cfg.AuthType, cfg.KeyPrefix, auth.Storage(), auth.Codec()), auth)
```

| 旧方法名 | 当前服务方法 |
| --- | --- |
| `GenerateNonce` | `nonceService.GenerateNonce` |
| `GenerateNonceWithTimeout` | `nonceService.GenerateNonceWithTimeout` |
| `VerifyNonce` | `nonceService.VerifyNonce` |
| `VerifyAndConsumeNonce` | `nonceService.VerifyAndConsumeNonce` |
| `IsNonceValid` | `nonceService.IsNonceValid` |
| `GetNonceTTL` | `nonceService.GetNonceTTL` |
| `RegisterOAuth2Client` | `oauth2Service.RegisterOAuth2Client` |
| `UnregisterOAuth2Client` | `oauth2Service.UnregisterOAuth2Client` |
| `GetOAuth2Client` | `oauth2Service.GetOAuth2Client` |
| `OAuth2Token` | `oauth2Service.OAuth2Token` |
| `GenerateOAuth2AuthorizationCode` | `oauth2Service.GenerateOAuth2AuthorizationCode` |
| `GenerateOAuth2AuthorizationCodeWithPKCE` | `oauth2Service.GenerateOAuth2AuthorizationCodeWithPKCE` |
| `ExchangeOAuth2CodeForToken` | `oauth2Service.ExchangeOAuth2CodeForToken` |
| `ExchangeOAuth2CodeForTokenWithPKCE` | `oauth2Service.ExchangeOAuth2CodeForTokenWithPKCE` |
| `OAuth2ClientCredentialsToken` | `oauth2Service.OAuth2ClientCredentialsToken` |
| `OAuth2PasswordGrantToken` | `oauth2Service.OAuth2PasswordGrantToken` |
| `RefreshOAuth2AccessToken` | `oauth2Service.RefreshOAuth2AccessToken` |
| `ValidateOAuth2AccessToken` | `oauth2Service.ValidateOAuth2AccessToken` |
| `ValidateOAuth2AccessTokenAndGetInfo` | `oauth2Service.ValidateOAuth2AccessTokenAndGetInfo` |
| `RevokeOAuth2Token` | `oauth2Service.RevokeOAuth2Token` |
| `CreateTicket`, `CreateTicketWithOptions` | `ticketService.CreateTicket` |
| `ValidateTicket`, `ValidateTicketWithOptions` | `ticketService.ValidateTicket` |
| `ConsumeTicket`, `ConsumeTicketWithOptions` | `ticketService.ConsumeTicket` |
| `RevokeTicket` | `ticketService.RevokeTicket` |
| `GetTicketStatus` | `ticketService.GetTicketStatus` |
| `GetTicketTTL` | `ticketService.GetTicketTTL` |
| `CreateShortKey`, `CreateShortKeyWithOptions` | `shortKeyService.CreateShortKey` |
| `ConfirmShortKey`, `ConfirmShortKeyWithOptions` | `shortKeyService.ConfirmShortKey` |
| `ValidateShortKey`, `ValidateShortKeyWithOptions` | `shortKeyService.ValidateShortKey` |
| `ConsumeShortKey`, `ConsumeShortKeyWithOptions` | `shortKeyService.ConsumeShortKey` |
| `RevokeShortKey` | `shortKeyService.RevokeShortKey` |
| `GetShortKeyStatus` | `shortKeyService.GetShortKeyStatus` |
| `GetShortKeyTTL` | `shortKeyService.GetShortKeyTTL` |

未合并 `WithOptions` 的方法保留业务参数，仅将接收者改为对应服务。七个 `WithOptions` 名称合并后的参数变化如下：

- `CreateTicket(ctx, loginID)` 改为 `ticketService.CreateTicket(ctx, ticket.CreateOptions{LoginID: loginID})`。
- `CreateShortKey(ctx)` 改为 `shortKeyService.CreateShortKey(ctx, shortkey.CreateOptions{})`。
- `ConfirmShortKey(ctx, key, loginID)` 改为 `shortKeyService.ConfirmShortKey(ctx, key, shortkey.ConfirmOptions{LoginID: loginID})`。
- 原创建和确认选项直接传给上述方法；Ticket 和 ShortKey 的校验、消费方法接收可选的 `ValidateOptions`，原无约束调用省略该参数，原带约束调用继续传入选项。

这些服务借用实例组件，并通过 `auth` 发布事件，使用期间须保证实例可用。HTTP 请求通过 `WithCapabilities` 绑定服务。已有请求快捷方法保留各框架命名；Nonce 有效性检查在请求层名为 `IsValidNonceByContext` 或 `IsValidNonceByCtx`，服务方法名则为 `IsNonceValid`。

## 依赖与发布

清理旧子模块依赖和替换规则，再添加根模块及所需扩展的新版本。根模块使用 vX.Y.Z tag，扩展使用当前目录前缀，例如 integrations/gin/vX.Y.Z 和 storage/redis/vX.Y.Z。本次重构首版以 v0.0.1 为目标，运行时版本、内部模块依赖和工作区替换规则统一使用该版本基线。这些源码声明不能证明新结构已远程发布。先发布被依赖模块，再更新并发布扩展。运行时版本位于 contract/version.go。

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
