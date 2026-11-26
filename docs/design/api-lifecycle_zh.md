# 公开 API、能力契约与生命周期

[English](api-lifecycle.md) | 中文文档

本文维护公开 API、资源共享和关闭约定。整体依赖见[架构设计](architecture_zh.md)，源码位置见[目录指南](source-layout_zh.md)，模块发布边界见[模块化设计](modular_zh.md)。

当前版本明确支持两层 API：应用优先使用根包 Auth；自定义适配器可以使用根包 Manager、ManagerOption 和 Strategy。Manager 是受支持的高级 API，并非待删除的兼容入口。internal 下的包路径不属于公开 API。

根包通过 engine_api.go 暴露的类型别名及其公开方法仍属于兼容性范围；TokenInfo、Session、TerminalInfo 等共享模型在 contract 中声明，根门面和引擎使用别名引用。新增公开方法、字段或修改签名时，应同时检查实例门面、高级 API、能力契约及框架导出的影响。

## 入口与类型声明来源

| 对外入口或类型 | 声明来源 | 使用约定 |
| --- | --- | --- |
| Auth、Builder、实例操作选项 | 根包 auth.go、builder.go、options.go | 应用优先使用的入口 |
| Config | config.Config，根包提供别名 | 核心配置在 config 包维护 |
| TokenInfo、Session、TerminalInfo、封禁模型、AccessProvider | contract，根 types.go 和引擎提供别名 | 修改共享模型时检查所有使用方和序列化兼容性 |
| Manager、Strategy、ManagerLoginOptions、RefreshTokenPair 等高级类型 | internal/engine，根 engine_api.go 提供别名 | 应用从根包命名这些类型，不导入实现路径 |
| 组件工厂类型、Components、ComponentFactories | internal/builder，根 engine_api.go 提供别名 | 供自定义组件装配使用 |
| 能力服务接口和数据类型 | contract/nonce、contract/oauth2、contract/ticket、contract/shortkey | HTTP 依赖契约；能力实现包复用对应数据类型 |
| 框架 DTokenContext、Capabilities | httpauth/context，框架 export.go 提供别名 | 框架保留原生请求入口，公共请求类型保持同源 |

根 LoginOptions 与 ManagerLoginOptions 服务于不同调用层次，不是同一个类型的两种名称；RefreshTokenPair 等高级结果也不是 contract/model.go 中的基础模型。类型整理应以实际别名关系为准，不因名称相近而复制、合并或迁移声明。

## 实例与引擎

`NewBuilder().Build()` 自动注册，`BuildInstance()` 创建不注册的独立实例。两者均返回 `(*Auth, error)`，失败处理由调用方决定。框架 NewBuilder() 转发根包构建器，行为相同。

Auth.Manager() 返回共享的底层引擎，引擎开始关闭后返回 nil。New(existingManager) 创建共享同一引擎的门面，不复制配置、存储或任务池。任何一个门面开始关闭引擎后，其他门面的 IsClosed() 也返回 true，认证操作返回 ErrManagerNotFound。已关闭 Auth 的 Config、Storage、Codec、EventManager 返回 nil。

固定 Auth 在构造后保持 Manager 指针不变；AuthType 选择器只保存体系名称，每次调用查找注册表。关闭状态统一由 Manager 维护，Auth 不维护单独的关闭标记。重复或通过不同固定门面调用 Close，都会等待该引擎的同一次资源释放完成。

提前保存的高级 Manager 同样遵循业务调用的关闭约定：开始关闭后，登录、校验、授权、Session、终端、封禁和刷新令牌方法返回 `ErrManagerNotFound`，布尔方法返回 `false`，关闭检查先于参数校验和存储访问。关闭前已接收的内部维护任务仍会完成封禁复核、续期和事件通知，之后才释放资源。同步请求仍由应用负责排空，入口检查不会等待进行中的调用。配置和策略快照仍可查看，但组件获取方法不会延长资源的使用期限。

从已构建的 Auth 获取 Manager，不直接创建 Manager 零值。Manager.GetConfig() 与 Auth.Config() 均返回独立快照；构建时固定 AuthType、KeyPrefix 及其他配置。需要调整配置时构建新实例，再按生命周期约定替换。

`Manager.GetStrategy()` 同样返回快照。修改返回值不会改变正在运行的 Session 创建、权限或角色匹配逻辑；自定义策略通过构建期 `WithStrategy` 注入。函数闭包捕获的状态不会被复制，其并发安全仍由调用方负责。

## 资源所有权

| 场景 | 所有权与关闭行为 |
| --- | --- |
| 构建成功 | 存储、日志和任务池默认归引擎管理，包括显式传入的组件 |
| 装配失败 | 工厂创建的组件会清理；显式传入的组件仍归调用方 |
| 注册竞争失败 | 已完成装配的候选实例按 ComponentOwnership 关闭，已注册的实例保持不变 |
| WithComponentOwnership | 完整指定所有权，false 字段归调用方管理 |
| Builder.Clone | 复制配置；显式传入的组件、访问提供器和回调内部状态仍共享 |
| New(existingManager) | 与已有门面共享引擎生命周期；不会获得独立资源 |
| 能力 Service | 借用构造器传入的组件，不关闭它们；事件发布器不会转移所有权 |

单个引擎的资源关闭是幂等的，但不会对多个引擎共享的组件做全局引用计数。多个 Auth 使用同一存储、日志或任务池时，应明确将相应所有权设为 false，由应用在全部使用者结束后关闭。需要独立组件时，使用每次创建新实例的组件工厂。

下面的片段中，`dtoken` 指根包，`memory` 指 `storage/memory`：

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

根包持有 Manager 注册表，Build 自动注册，无需额外调用 `SetAuth`：

```go
userAuth, err := dtoken.NewBuilder().AuthType("user").Build()
if err != nil { panic(err) }
defer userAuth.Close()

// Both paths use the same engine. 两种调用路径使用同一引擎。
token, err := userAuth.LoginID(ctx, "10001")
if err != nil { panic(err) }
err = dtoken.AuthType("user").CheckLogin(ctx, token)
```

- 不设置 AuthType 的 Builder 注册默认体系 `auth:`。`dtoken.Login`、`dtoken.IsLogin` 等包级方法始终使用默认体系，签名与 Auth 方法一致。
- `dtoken.AuthType("user")` 返回动态选择器，每次业务操作查注册表；缓存选择器也会跟随显式替换。缺失体系返回 `ErrManagerNotFound`，布尔校验返回 false，不回退到默认体系。
- 业务选项只包含操作参数，不提供 AuthType 字段；通过根包默认调用、AuthType(name) 或固定 Auth 实例选择引擎。
- `Build()` 同名注册返回 `errors.ErrManagerAlreadyRegistered`，保留原实例。并发构建仅一个成功；竞争失败时关闭已装配候选的自有资源。共享组件须显式声明为调用方所有。
- 需要显式替换时，先 `BuildInstance()`，再 `dtoken.SetAuth(auth)`；`dtoken.SetAuth`、`SetManager` 显式替换成功后关闭旧引擎，注册失败返回 error。同一实例重复 Set 不会关闭它。
- 实例 `auth.Close()` 将共享引擎标记为关闭中并注销仍指向自身的注册项，然后等待资源释放；旧实例关闭不会移除替换后的新实例。
- `dtoken.Close()` 关闭默认体系，`dtoken.AuthType("user").Close()` 关闭当前 user 体系，`dtoken.CloseAll()` 关闭所有当前注册实例；重复调用安全。CloseAll 不关闭未注册的独立实例，也不禁止应用随后重新构建。

注册表保留写入互斥锁来协调注册、替换和删除，普通查找不获取此锁。替换或删除时，在释放注册表锁前让旧 Manager 进入关闭状态并拒绝新异步任务；在锁外等待已接收任务和监听器结束并释放资源。注册表不再单独维护退役集合，正在关闭的引擎由自身状态阻止重新注册。

`WithAuth(auth)` 绑定具体引擎，不跟随注册表替换。需要每次请求从注册表选择时，显式传入 `WithManagerResolver(dtoken.GetManager)`。已经建立的请求作用域保持原实例。应用应先协调路由切换并排空请求，再替换或关闭旧引擎；注册表锁和关闭标记不会等待正在执行的同步业务请求。

迁移已有应用时：删除 Build 后冗余的 SetAuth；将依赖“同名但独立”行为的构建改为 BuildInstance；需要直接使用底层 Manager 时调用返回实例的 Manager()。关闭入口共享资源所有权，不需要逐个关闭后再额外 CloseAll。

框架不安装 signal 监听器，也不调用 os.Exit。应用负责监听退出信号，停止接收请求并等待处理完成，再调用 CloseAll。强制终止进程无法保证执行关闭回调。

## 能力契约

contract/nonce、contract/oauth2、contract/ticket、contract/shortkey 定义服务接口和数据类型。HTTP 上下文及框架适配器只依赖这些契约，不导入能力实现。

nonce、oauth2、ticket、shortkey 提供内置实现，通过编译期断言验证满足契约。原能力包中的公开数据类型保留为契约类型的别名，现有结构体字面量、方法签名和序列化字段保持一致。

Capabilities 的字段接受服务接口，自定义服务可直接注入。未绑定服务或带类型的 nil 返回原有模块未启用结果；布尔检查返回 false。请求上下文方法 `dCtx.WithCapabilities(nil)` 保留已有绑定，`dCtx.WithCapabilities(&requestcontext.Capabilities{})` 显式清空绑定。框架选项则接收值，例如 `gindt.WithCapabilities(requestcontext.Capabilities{Nonce: service})`。请求适配器更换时保留绑定，认证实例更换时清空能力绑定。

独立 Service 的生命周期由调用方管理。向 NewService 传入 Auth 作为事件发布器，只建立事件联系；直接调用 Service 时仍须保证其借用的存储和编解码器可用。DTokenContext 绑定了 Manager 时，直接调用其 Nonce、OAuth2、Ticket、ShortKey 分组也会在引擎开始关闭后返回 ErrManagerNotFound，布尔检查返回 false；未绑定 Manager 的上下文仍可调用显式绑定的独立能力服务。需要当前登录身份的能力操作仍要求有效认证绑定。

读取当前请求 Token 或操作 Cookie 的方法要求未关闭的 Manager 和有效请求适配器：Manager 缺失或关闭时返回 ErrManagerNotFound，请求适配器缺失（包括带类型的 nil）时返回 ErrInvalidParam。无错误返回值的 Token 提取返回空字符串，布尔认证检查返回 false，直接 Cookie 写入不执行操作。Cookie 登录在写入认证状态之前检查绑定。Auth.Login、Refresh 等使用显式参数的操作只要求未关闭的 Manager，不要求请求适配器。

Ticket 和 ShortKey 的 AuthType 身份比较与存储键一致：`user` 与 `user:` 等价，已有载荷无需迁移；创建结果保留调用方传入的 AuthType 形式。比较只忽略一个末尾冒号，不合并空白或其他不同名称。组合能力时建议使用 Auth.Config() 中的 AuthType 和 KeyPrefix。

契约包和能力实现仍随根模块一起发布；拆分的是包依赖与运行时装配，没有增加独立版本或动态插件机制。

## 公开生命周期示例

<!-- dtoken-compile: lifecycle -->
```go
package main

import (
    "log"
	"fmt"
	"github.com/Zany2/dtoken-go"
)

func main() {
	auth, err := dtoken.NewBuilder().IsPrintBanner(false).Build()
	if err != nil {
		log.Print(err)
		return
	}
	defer auth.Close()
	shared := dtoken.New(auth.Manager())
	shared.Close()
	fmt.Println(auth.IsClosed()) // true
}
```

## 应用管理退出信号

此示例等待 HTTP 请求处理完毕后，才执行 defer CloseAll。使用 WebSocket 等长连接时，应用还须等待其处理循环退出。关闭回调不能从引擎自己的异步任务或事件监听器内同步执行。

<!-- dtoken-compile: graceful-shutdown -->
```go
package main

import (
    "log"
    "context"
    "errors"
    "net/http"
    "os"
    "os/signal"
    "syscall"

    "github.com/Zany2/dtoken-go"
)

func main() {
    if _, err := dtoken.NewBuilder().IsPrintBanner(false).Build(); err != nil {
        log.Print(err)
        return
    }
    defer dtoken.CloseAll()

    stopped, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()
    server := &http.Server{Addr: ":8080", Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if err := dtoken.CheckLogin(r.Context(), r.Header.Get("Authorization")); err != nil {
            w.WriteHeader(http.StatusUnauthorized)
            return
        }
        w.WriteHeader(http.StatusNoContent)
    })}
    done := make(chan error, 1)
    go func() { done <- server.ListenAndServe() }()
    var serveErr error
    select {
    case serveErr = <-done:
    case <-stopped.Done():
    }
    if err := server.Shutdown(context.Background()); err != nil {
        log.Print(err)
        return
    }
    if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
        log.Print(serveErr)
        return
    }
}
```
