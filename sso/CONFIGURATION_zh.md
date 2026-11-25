# SSO 配置与启动校验

[English](CONFIGURATION.md)

`ClientConfig.SharedStoreServer` 可显式启用 Ticket／Code 的共享存储兑换；`UseHTTPCheck=true` 强制 HTTP。未配置本地服务时保持历史 HTTP 行为。校验包含本地服务初始化状态、凭证模式及参数名一致性，不探测共享 Redis，详见[部署方式](DEPLOYMENT_MODES_zh.md)。

中心 HTTP 失败统一使用固定公开消息，并通过可选 `OnHTTPError` 观察钩子保留私有诊断。六个端点的状态兼容性及回调规则见 [HTTP 错误与诊断](HTTP_ERRORS_zh.md)。

应用组装时优先使用带校验的构造入口，让配置错误在 HTTP 监听前返回。这些检查只在本地进行，不访问认证中心、不探测存储可用性，也不验证两端密钥是否一致。

## 接入入口

配置后的协议参数名同时定义请求中的单值字段。重复值（含 Query／Body 重复）在 nonce 消费前拒绝，凭证操作仅接受所选模式对应的凭证字段。自定义参数名及兼容要求见[请求参数一致性](REQUEST_VALIDATION_zh.md)。

HTTP(S) URL 校验拒绝显式空端口、零端口及超出范围的端口。同源比较统一主机大小写、IP 写法和有效默认端口；授权重定向地址仍按字符串精确匹配。浏览器状态 Cookie 还会拒绝未知 `SameSite` 值。Origin 请求头约束见[浏览器接入](BROWSER_CLIENT_zh.md)。

| API | 行为 |
| --- | --- |
| `Endpoints.Validate()` / `ParamNames.Validate()` | 补齐省略字段的默认值后检查，不修改传入值 |
| `(*Client).Validate()` | 检查客户端注册元数据，不写入存储 |
| `(*Server).RegisterClientChecked(client)` | 校验元数据后执行已有注册流程 |
| `HTTPOptions.Validate()` | 检查路由、参数、Cookie、签名／防重放及内联客户端，不访问存储 |
| `NewHTTPServerChecked(server, options)` | 先校验全部本地配置，再注册内联客户端，并返回初始化错误 |
| `httpserver.NewChecked(server, options)` | 通过可选门面调用同一构造入口 |
| `ClientConfig.Validate()` / `(*ClientApp).Validate()` | 检查规范化的客户端配置，不发起 HTTP 请求 |
| `NewClientAppChecked(config)` | 校验后创建客户端辅助对象 |
| `(*HTTPServer).InitializationError()` | 获取保留的初始化错误，不是健康检查 |

`NewBrowserClient` 和 `NewRemoteAuth` 也会检查完整的 `ClientApp` 配置，包括当前流程尚未使用的端点。组件自身的模式、状态 Cookie、超时和授权范围约束继续有效。

`HTTPOptions.RequestLimits` 和 `ClientConfig.RequestLimits` 的查询串／表单体默认分别为 8 KiB／64 KiB；每个非正字段独立使用默认值，不会禁用限制。入站限制与出站响应的 `MaxResponseBytes` 独立。413 错误、提前解析的表单和代理配置见[HTTP 请求大小限制](REQUEST_LIMITS_zh.md)。

## 启动示例

<!-- dtoken-compile: sso-configuration -->
```go
package main

import (
	"net/http"

	"github.com/Zany2/dtoken-go/sso"
)

// assembleSSO checks configuration and transfers the server lifetime to the caller on success. assembleSSO 检查配置，成功后由调用方管理 Server 生命周期。
func assembleSSO(resolve sso.LoginIDResolver, exchangeKey, signingKey string) (*sso.Server, http.Handler, *sso.ClientApp, error) {
	server := sso.NewServer()
	options := sso.DefaultHTTPOptions()
	options.RequestLimits = sso.RequestLimits{MaxQueryBytes: 8 << 10, MaxBodyBytes: 64 << 10}
	options.LoginIDResolver = resolve
	options.Clients = map[string]sso.Client{
		"orders": {
			ClientSecret: exchangeKey,
			SecretKey:    signingKey,
			RedirectURIs: []string{"https://orders.example.com/sso/callback"},
			Modes:        []sso.Mode{sso.ModeTicket},
			Scopes:       []string{"profile"},
		},
	}
	handler, err := sso.NewHTTPServerChecked(server, options)
	if err != nil {
		_ = server.Close()
		return nil, nil, nil, err
	}

	config := sso.DefaultClientConfig()
	config.RequestLimits = options.RequestLimits
	config.ClientID = "orders"
	config.ClientSecret = exchangeKey
	config.SecretKey = signingKey
	config.ServerURL = "https://login.example.com"
	app, err := sso.NewClientAppChecked(config)
	if err != nil {
		_ = server.Close()
		return nil, nil, nil, err
	}
	return server, handler.Handler(), app, nil
}

// main leaves authentication, secret injection and HTTP listening to the application. main 将身份核验、密钥注入及 HTTP 监听交给应用。
func main() {}
```

部署时使用持久化共享存储；上例使用默认内存存储只是展示组装方式。`ClientSecret` 用于凭证兑换认证，`SecretKey` 用于协议请求签名。客户端独立签名密钥优先于中心默认密钥；相关客户端均使用独立密钥时，不要求中心配置全局默认签名密钥。

应用传入可信身份解析器和托管密钥，构造成功后再挂载返回的 Handler，并在请求排空后关闭返回的 Server。此示例已纳入仓库文档编译检查；该检查不执行 `main`，也不代表身份认证及部署已验证。

## 校验规则

- 可选 `VerificationKey` 要求同一配置中存在 `SecretKey`。带校验的 HTTP／ClientApp 构造器还要求配置额外密钥时开启签名。客户端独立组不能与中心默认组混用，详见[密钥轮换](SIGNING_ROTATION_zh.md)。额外验签字段不影响 Cookie 签名。

- 六个中心端点（`Authorize`、`Token`、`Introspect`、`UserInfo`、`Revoke`、`Logout`）必须是互异且规范的绝对字面路径。拒绝根路径／尾斜杠、通配符、方法／主机模式、查询串、片段、控制字符、反斜杠、编码后的分隔符及解码后重复路径；允许 `/sso/%61uth` 这类普通转义。预留消息和客户端路由字段不会作为中心路由挂载或检查冲突。
- 参数名在补齐默认字段后必须互异、非空且不含空白或控制字符。客户端与中心的自定义路由、参数必须一致；`ServerURL` 中的网关前缀会保留。
- 客户端 ID 非空且不含空白／控制字符；模式必须已实现；每个 scope 是不含空白、逗号的独立非空名称。回调地址必须是无用户信息、无片段的 HTTP(S) 绝对 URL；允许来源只能没有路径或为 `/`，且不能有查询串。`RequirePKCE` 至少需要 Ticket／Code 模式之一；省略模式时仍采用 Ticket。直接使用核心 API 时允许空回调列表，但这不代表允许任意 HTTP 回调。
- 中心必须提供 `LoginIDResolver` 或已签名中心 Cookie 的密钥（`Cookie.SecretKey`，可回退到 `ServerOptions.SecretKey`）。检查 Cookie 名称、域、路径、安全前缀及 `SameSite=None` 与 `Secure` 的组合。自定义身份解析器不会跳过已有 Cookie 属性的检查。
- 启用签名时，每个内联客户端必须有独立密钥或全局默认密钥。同时启用匿名访问及签名时，需要全局默认密钥，或者显式配置携带独立密钥的内联匿名客户端。匿名白名单在启用匿名访问时检查。静态校验无法枚举外部存储中的既有客户端，请求期间仍执行客户端／密钥检查；只有存储中的匿名密钥不能满足上述静态规则。
- 客户端 `ServerURL` 必须是 HTTP(S) 绝对地址，不含用户信息、查询串、片段、路径空白或反斜杠。`RegisterCallback` 启用时必须提供有效的绝对 `LogoutCallbackURL`，未使用的回调字段不强制填写。防重放必须配合签名及有效的防重放组件。

保留已有默认值规则：路由与参数逐项补齐；客户端响应大小、回调时效、中心通知超时／并发数／重试配置、Cookie 时长继续按已有非正值规则回退。零值配置结构体不会自动启用布尔默认值；需要默认行为时从 `DefaultHTTPOptions()`／`DefaultClientConfig()` 开始配置。Browser 与 RemoteAuth 继续遵循各自文档中的负值规则。

## 错误、兼容与资源所有权

用 `errors.Is(err, sso.ErrSSOConfig)` 识别静态配置错误，用 `errors.As` 读取 `*sso.ConfigurationError` 的 `Field` 和固定 `Reason`。诊断不回显 URL、密钥或注册配置值；适用时保留 `ErrInvalidRedirectURI`、`ErrSignSecretRequired`、`ErrModeUnsupported` 等原有领域分类。每次返回首个错误，内联客户端按键排序检查，保证报告稳定。空核心对象与存储失败仍使用各自错误分类。自定义存储错误可能包含后端详情，公开 HTTP 响应不应直接输出原始初始化错误。

全部内联配置通过检查后才开始写注册记录，校验不会向调用方 map 回填客户端 ID。实际注册仍是非事务性的顺序操作：存储故障可能留下部分注册结果，包含对既有记录的覆盖，不执行回滚。构造失败不会关闭传入的 Server、HTTP 客户端或借用存储，应用继续负责资源清理。

旧 `NewClientApp`、`NewHTTPServer`、`RegisterClient` 签名保留。旧 HTTP 构造入口保留路由、参数、防重放及注册错误，自身端点配置无效时不会在组装 mux 时触发 panic。初始化失败时，`Handler()` 返回固定 HTTP 500 响应；端点配置无效或缺少核心对象时，`Register(mux)` 不注册路由。挂载独立处理器前应使用带校验的入口或检查 `InitializationError()`。应用 mux 已有路由造成的外部冲突仍遵循 `http.ServeMux` 的 panic 行为，由应用负责处理。旧构造器不会执行本页列出的全部严格校验。

四个 HTTP／Gin 示例均在监听前检查配置。`config_validation_test.go` 补充了写入前拒绝、资源所有权、旧路由入口、注册检查、诊断、组件接入和签名授权／换票流程的回归源码。执行方式见[测试说明](TESTING_zh.md)。
