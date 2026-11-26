# 模块组织与迁移

[English](module-migration.md) | 中文文档

本文维护旧路径与 API 的升级步骤、发布顺序和消费者验证方式。当前系统分层见[架构设计](architecture_zh.md)，包与模块边界见[模块化设计](modular_zh.md)，代码位置见[源码目录](source-layout_zh.md)。

当前 Manager 及 Nonce/OAuth2/Ticket/ShortKey 均在编码后的 KeyPrefix、AuthType 之后增加固定 `v2:` 段，不回退读取无版本空间。部署前参见[离线迁移](namespace-copy_zh.md)，使用 `PrepareAuthUpgrade` 将旧数据转换到实际版本化目标；两个前缀可以相同。`Prepare` 和 `PrepareTokenUpgrade` 保留无版本复制或暂存语义，不能代替完整升级。归属歧义和目标冲突会使预检失败；工具不会启动应用或清理旧数据，新空间内部的兼容解析清理仍待完成。

## 登录校验与封禁拆分

本轮性能重构不再让 `CheckLogin`、`IsLogin`、身份查询、按 Token 访问 Session 和角色权限方法隐式读取封禁。临时设备封禁下凭据仍可有效；账号封禁已经废弃的凭据则返回 `ErrInvalidToken`。需要具体封禁原因时使用显式检查，自省仍执行限制检查。

升级受保护路由时，将原来依赖隐式封禁的 `AuthMiddleware` 改为 `AccessMiddleware`，通过 `WithRouteAccessHandler` 设置 `req.CheckDisable = true`；使用注解时设置 `CheckDisable: true`。直接使用公共 HTTP 检查时设置 `httpauth.Request.CheckDisable`。这些入口复用一次 Token 身份读取，覆盖账号、设备类型和具体设备。只要求登录态的路由可以继续使用 `AuthMiddleware`。

封禁检查仍同步决定本次访问结果，并在提交登录维护任务前执行。启用活跃超时会额外读取活跃记录；权限回退、自省和异步续期的开销不计入“基础校验一次 GET”。示例路由已显式声明限制；应用升级时也需要同步声明，不能仅替换依赖版本。

## 续期与活跃维护

续期维护也已按实际需要访问 Session：自动续期检查 key 的 TTL，不校验 Session 内容；缺失时不续期，也不重建。仅活跃更新不再依赖 Session 存在。手动续期、共享登录及依赖 Session 内容的业务操作保留前置校验。自定义存储可选实现 `contract.PersistentStorage` 来避免永久化时读取和写回完整值；原有 `Expire(<=0)` 删除语义不变。

Session 内容及未续期的活跃值更新优先使用可选 `contract.UpdateStorage`。Redis 使用 `SET XX KEEPTTL`，Memory 和普通存储保留 TTL + Set 非原子回退。基础 Storage 和 FullStorage 接口不强制增加方法。

池满或无池时不再创建额外 goroutine：自动续期跳过本次任务，活跃更新、显式 `LoginByToken` 和事件投递同步回退。独立事件管理器需要通过 `SetAsyncPool` 配置池才能异步执行；`TriggerSync` 按优先级内联执行直接监听器。应用不能依赖这些回调一定在另一个 goroutine 运行，也不应在回调内等待同一事件的其他回调或调用等待自身完成的 Close/Wait。

## Provider 按需回退

按 Token 查询权限、角色和 HTTP 组合授权不再预先加载 Session。有效凭据由 Provider 返回非 `nil` 数据时，Session 缺失或损坏不再阻止授权；空列表仍表示明确无权限，不会回退。仅 Provider 未配置或返回 `nil` 时读取 Session，组合检查最多复用一次正常路径读取。依赖 Session 存在性控制访问的应用应通过注销、踢出等 API 撤销凭据，或在业务策略中显式检查会话；不能继续用删除 Session 代替撤销。完整自省的 Session 校验约定保持不变。

## 错误处理接口迁移

框架不再提供主动 panic 的构建和上下文读取接口，调用方负责处理返回错误：

| 原调用 | 当前调用 |
| --- | --- |
| `MustBuild()` | `auth, err := builder.Build()` |
| `MustBuildInstance()` | `auth, err := builder.BuildInstance()` |
| `WithAuth(auth)` 直接作为选项使用 | 先取得 `option, err := WithAuth(auth)`，成功后传入中间件 |
| `Context(auth)` 直接作为中间件使用 | 先取得 `middleware, err := Context(auth)`，成功后注册 |
| `RequestContext.MustGet(key)` | `value, err := requestContext.GetRequired(key)` |

`WithAuth` 和 `Context` 对空实例、已关闭实例返回 `ErrManagerNotFound`。`GetRequired` 在键不存在时返回可由 `errors.Is` 判断的 `ErrKeyNotFound`；只需要判断是否存在时，继续使用 `Get(key) (any, bool)`。自定义 RequestContext 实现也需要同步方法签名。GoFrame 的请求体读取异常通过 `GetBody() ([]byte, error)` 返回，不重新抛出。

## 注册与配置收敛

- 业务选项中的 AuthType 字段已删除；默认调用使用 dtoken.XXX，指定体系使用 dtoken.AuthType(name).XXX。
- BuildAndSetManager 已删除。普通初始化使用 Builder.AuthType(name).Build()；显式替换使用 BuildInstance() 后调用 SetAuth，并检查错误。
- SetAuth、SetManager 现在返回 error。空实例、已关闭实例返回 ErrManagerNotFound，无配置的 Manager 返回 ErrManagerInvalidType；失败不会替换原注册实例。同一有效实例重复注册返回 nil。
- Manager.GetConfig() 与 Auth.Config() 都返回独立配置快照，构建后的认证配置不再通过指针修改。需要更改配置时创建新实例；AuthType 或 KeyPrefix 变更不会迁移旧存储数据。
- HTTP 请求路径借用 internal/engine 的只读配置，避免每次校验复制配置。八个适配器的分组快捷方法由 integration_facade_architecture_test.go 检查转发目标和参数一致性。

## 当前边界

根模块发布实例门面、公共契约、内部认证实现、基础组件、共享 HTTP 鉴权，以及 Nonce、OAuth2、Ticket、ShortKey 独立能力包。能力包显式组合，不是单独的 Go 模块。工作区共 30 个模块。

框架集成、codec/jsonv2、codec/msgpack、logging/gf、storage/redis、sso 和 sso/storage/redis 独立发布。示例与应用测试保留独立模块。

## 包路径迁移

这是完整的包边界和 API 迁移，旧路径不再作为兼容入口。

| Previous entry | Current entry |
| --- | --- |
| dtoken subdirectory | Root instance/default APIs and AuthType selectors |
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

同名 Build 返回 ErrManagerAlreadyRegistered；同名独立实例改用 BuildInstance。显式替换使用 BuildInstance 后 SetAuth。默认包级调用与 AuthType 选择器共用根注册表，退出时先排空请求，再 CloseAll。详见 [API 与生命周期](api-lifecycle_zh.md)。

## 权限提供器迁移

旧 SetCustomPermissionListFunc、SetCustomRoleListFunc 及 Ext 变体不再提供。使用根 Builder.SetAccessProvider 注入 AccessProviderFunc，通过 AccessSubject 取得认证体系、账号、设备与 Token；nil 结果回退到 Session，非 nil 空切片表示无权限。见 [AccessProvider](../guide/core/access-provider_zh.md)。

## 独立能力

Nonce、OAuth2、Ticket、ShortKey 的 Manager/Server 构造器（包括默认构造器）现在统一返回 `(实例, error)`，先检查错误，再调用 `NewService`。`WithConfig` 的 `nil` 配置使用默认值；显式配置必须通过 `Validate`，建议从 `DefaultConfig()` 开始修改。无效配置、缺失或带类型的 nil 存储/编解码器返回 `errors.ErrInvalidParam`。Nonce 的直接 TTL 参数必须为正，OAuth2 的刷新有效期必须大于访问有效期。单次业务操作的超时回退规则保持原样。

Builder 不提供能力开关。通过各自的构造器创建能力 Manager，再通过 NewService(manager, auth) 接入实例事件。使用所属认证实例的 AuthType、KeyPrefix、Storage 和 Codec。httpauth/context.Capabilities 和框架 WithCapabilities 将服务绑定到请求。未绑定的能力返回 ErrModuleNotEnabled。同一 Manager 的后续中间件未指定覆盖配置时保留请求绑定；切换 Manager 时创建新作用域。

## 门面方法迁移

框架包保留 `NewBuilder` 及请求快捷方法。Auth、业务选项、默认和指定体系调用统一从根包导入。原 global 包已移除；默认调用改为 dt.XXX，指定体系改为 dt.AuthType(name).XXX，并按根包签名使用类型化选项。原 Default().XXX / NewByAuthType(name).XXX 改为 dt.XXX / dt.AuthType(name).XXX，后续操作返回查找错误；需要立即检查时调用 GetManager。组件构造器从所属组件包导入。

旧调用不能只替换包名前缀，登录、权限变更和封禁等方法需要改为根包的类型化选项。下表仅用来说明升级前后的对应关系，左列 API 已删除，右列 dt 表示根包：

| 升级前 | 当前调用 |
| --- | --- |
| `global.Login(ctx, id, device, deviceID, "user")` | `dt.AuthType("user").Login(ctx, dt.LoginOptions{LoginID: id, Device: device, DeviceID: deviceID})` |
| `global.Logout(ctx, token, "user")` | `dt.AuthType("user").LogoutByToken(ctx, token)` |
| `global.AddPermissions(ctx, id, values, "user")` | `dt.AuthType("user").AddPermissions(ctx, dt.PermissionOptions{LoginID: id, Permissions: values})` |
| `global.Disable(ctx, id, duration, reason, "user")` | `dt.AuthType("user").Disable(ctx, dt.DisableOptions{LoginID: id, Duration: duration, Reason: reason})` |
| `global.Default().XXX(...)` / `global.NewByAuthType("user").XXX(...)` | `dt.XXX(...)` / `dt.AuthType("user").XXX(...)`；立即校验使用 `dt.GetManager(...)`，需要固定门面时包装为 `dt.New(mgr)` |

选择器的创建不返回查找错误，后续业务操作才解析注册表。原来依赖构造时立即报错的代码，应先调用 GetManager。已注册和返回实例共享引擎，关闭任一固定门面会影响其他共享者。更多方法见[根包 API 参考](../api/dtoken_zh.md)。

下表覆盖旧 `Auth` 门面移出的 40 个能力方法。原包级能力调用也迁移到相同服务入口；根据目标认证实例选择已组合的服务，不再通过末尾的 `authType` 参数选择实例。从当前根路径下的 `nonce`、`oauth2`、`ticket`、`shortkey` 包导入能力类型。构建 `auth` 后，表中的服务变量按以下方式组合：

以下片段放在可返回 `error` 的初始化函数中：

```go
cfg := auth.Config()
nonceManager, err := nonce.NewDefaultNonceManager(cfg.AuthType, cfg.KeyPrefix, auth.Storage())
if err != nil {
	return err
}
nonceService := nonce.NewService(nonceManager, auth)
oauth2Server, err := oauth2.NewDefaultOAuth2Server(cfg.AuthType, cfg.KeyPrefix, auth.Storage(), auth.Codec())
if err != nil {
	return err
}
oauth2Service := oauth2.NewService(oauth2Server, auth)
ticketManager, err := ticket.NewDefaultManager(cfg.AuthType, cfg.KeyPrefix, auth.Storage(), auth.Codec())
if err != nil {
	return err
}
ticketService := ticket.NewService(ticketManager, auth)
shortKeyManager, err := shortkey.NewDefaultManager(cfg.AuthType, cfg.KeyPrefix, auth.Storage(), auth.Codec())
if err != nil {
	return err
}
shortKeyService := shortkey.NewService(shortKeyManager, auth)
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
