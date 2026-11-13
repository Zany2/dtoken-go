# 公开 API、能力契约与生命周期

当前版本明确支持两层 API：应用优先使用根包 Auth；自定义适配器可以使用根包 Manager、ManagerOption 和 Strategy。Manager 是受支持的高级 API，并非待删除的兼容入口。internal 下的包路径不属于公开 API。

## 实例与引擎

NewBuilder().Build() 返回 Auth。框架 NewBuilder() 转发根包构建器，返回类型相同。

Auth.Manager() 返回共享的底层引擎。New(existingManager) 创建共享同一引擎的门面，不复制配置、存储或任务池。任何一个门面关闭引擎后，其他门面的 IsClosed() 也返回 true，认证操作返回 ErrManagerNotFound。已关闭 Auth 的 Config、Storage、Codec、EventManager 返回 nil。

从已构建的 Auth 获取 Manager，不直接创建 Manager 零值。开始处理请求后，不应修改 Manager.GetConfig() 返回的配置或替换组件；需要调整配置时构建新实例。

## 资源所有权

| 场景 | 所有权与关闭行为 |
| --- | --- |
| 构建成功 | 存储、日志和任务池默认归引擎管理，包括显式传入的组件 |
| 构建失败 | 构建过程创建的组件会清理；显式传入的组件仍归调用方 |
| WithComponentOwnership | 完整指定所有权，false 字段归调用方管理 |
| Builder.Clone | 复制配置；显式传入的组件、访问提供器和回调内部状态仍共享 |
| New(existingManager) | 与已有门面共享引擎生命周期；不会获得独立资源 |
| 能力 Service | 借用构造器传入的组件，不关闭它们；事件发布器不会转移所有权 |

单个引擎的资源关闭是幂等的，但不会对多个引擎共享的组件做全局引用计数。多个 Auth 使用同一存储、日志或任务池时，应明确将相应所有权设为 false，由应用在全部使用者结束后关闭。需要独立组件时，使用每次创建新实例的组件工厂。

```go
shared := memory.NewStorage()
builder := dtoken.NewBuilder().
    Storage(shared).
    UseManagerOption(dtoken.WithComponentOwnership(dtoken.ComponentOwnership{
        Storage: false,
        Logger: true,
        Pool: true,
    }))
first, err := builder.AuthType("user").Build()
if err != nil { panic(err) }
defer first.Close()
second, err := builder.Clone().AuthType("admin").Build()
if err != nil { panic(err) }
defer second.Close()
```

Auth.Close() 停止引擎维护任务、等待已接收的异步工作并释放自有资源。它不是 HTTP 服务器的请求排空器。应用应先停止接收新请求，等待正在处理的请求结束，再关闭 Auth，最后关闭调用方持有的共享资源。

## 全局注册表

应用使用 global.SetAuth(auth) 注册实例；自定义底层集成仍可使用 SetManager。两者共享同一注册表及关闭规则：

- 同一实例重复注册不会关闭实例。
- 同认证类型的新实例替换旧实例时，注册表关闭旧引擎。
- DeleteManager 和 DeleteAllManager 先移除注册项，再关闭相应引擎。
- 空实例及已关闭实例不会被重新注册。
- 注册不会创建独立副本，应用仍可主动调用 Auth.Close()；注册表查询会清理已关闭条目。

WithAuth(auth) 绑定具体实例，替换注册项不会让该中间件自动切换。需要每次请求通过注册表选择实例时，显式配置 WithManagerResolver(global.GetManager)。已经建立的请求作用域保持自己的实例，关闭后不会隐式借用新实例。需要不中断正在执行的请求时，应用必须先协调路由切换和请求排空，再退役旧实例。

## 能力契约

contract/nonce、contract/oauth2、contract/ticket、contract/shortkey 定义服务接口和数据类型。HTTP 上下文及框架适配器只依赖这些契约，不导入能力实现或 global。

nonce、oauth2、ticket、shortkey 提供内置实现，通过编译期断言验证满足契约。原能力包中的公开数据类型保留为契约类型的别名，现有结构体字面量、方法签名和序列化字段保持一致。

Capabilities 的字段接受服务接口，自定义服务可直接注入。未绑定服务或带类型的 nil 返回原有模块未启用结果；布尔检查返回 false。WithCapabilities(nil) 保留已有绑定，空 Capabilities 显式清空绑定。请求适配器更换时保留绑定，认证实例更换时清空能力绑定。

独立 Service 的生命周期由调用方管理。向 NewService 传入 Auth 作为事件发布器，只建立事件联系；直接调用 Service 时仍须保证其借用的存储和编解码器可用。HTTP 请求则由绑定引擎的可用性检查控制。

契约包和能力实现仍随根模块一起发布；拆分的是包依赖与运行时装配，没有增加独立版本或动态插件机制。
