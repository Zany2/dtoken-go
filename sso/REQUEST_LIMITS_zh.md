# SSO HTTP 请求大小限制

[English](REQUEST_LIMITS.md) · [请求参数一致性](REQUEST_VALIDATION_zh.md) · [配置与启动校验](CONFIGURATION_zh.md)

中心协议入口、单点注销回调和浏览器登录回调在验签、消费 nonce／state 及调用业务钩子之前限制入站参数大小。签名正确或关闭签名都不会绕过限制。直接调用可信 `Server` 方法不经过 HTTP 限制。

## 配置和覆盖范围

`HTTPOptions.RequestLimits` 控制中心接收请求，`ClientConfig.RequestLimits` 控制子应用接收回调；两端可以独立配置。`httpserver.Options` 使用相同字段，该门面也提供 `RequestLimits` 类型别名和 `DefaultRequestLimits()`。

| 配置 | 默认值 | 计量方式 |
| --- | --- | --- |
| `MaxQueryBytes` | 8 KiB（8192 字节） | `URL.RawQuery` 的编码字节数，不包含路径及 `?` |
| `MaxBodyBytes` | 64 KiB（65536 字节） | POST 表单体的编码字节数；包含百分号编码及字段分隔符 |

非正值分别使用默认值，不表示禁用限制。原构造函数和 Checked 构造函数都应用默认值。恰好等于上限允许继续处理，超过上限拒绝；不能用 `Content-Length: 0`、未知长度或分块请求绕过实际读取上限。若声明的 Content-Length 已超过上限，直接拒绝且不读请求体。

```go
limits := sso.RequestLimits{MaxQueryBytes: 16 << 10, MaxBodyBytes: 128 << 10}
options := sso.DefaultHTTPOptions()
options.RequestLimits = limits // Center inbound requests 中心入站请求
config := sso.DefaultClientConfig()
config.RequestLimits = limits // Application inbound callbacks 子应用入站回调
```

此代码仅展示限制字段；完整组装见配置指南。`ClientConfig.MaxResponseBytes` 仍独立控制 SDK 读取中心响应体，默认 1 MiB；它不控制入站回调。限制只约束接收方，不会裁剪 URL 或请求参数，也不限制 SDK 发送大小。若扩展参数确需更大空间，应协调相应接收方与代理上限。

| 入口 | 检查范围 |
| --- | --- |
| 中心 `HandleAuthorize` | GET 查询串 |
| 中心 `HandleToken`、`HandleIntrospect`、`HandleUserInfo`、`HandleRevoke` | POST 查询串和表单 |
| 中心 `HandleLogout` | GET 查询串或 POST 查询串和表单 |
| `ClientApp.VerifyLogoutCallback` / `LogoutCallbackHandler` | POST 查询串和表单 |
| `BrowserClient.CompleteLogin` / `CallbackHandler` | GET 查询串，检查在消费 state 与联系中心之前 |

直接挂载各处理器同样生效，不要求通过中心 `Handler()` 的 ServeMux。自定义路由与协议参数名不影响字节限制。Query／Body 仍按标准库规则合并，后续单值参数及 PKCE 检查继续生效。

## 表单中间件及 HTTP 边界

未解析的 `application/x-www-form-urlencoded` 请求体通过 `http.MaxBytesReader` 限制读取；未知长度请求最多向底层读取上限加 1 字节，以确认是否超限。解析失败时不发布标准库返回的部分 Form/PostForm，不使用已解析出的合法字段继续业务流程。

若上游已调用 `ParseForm` 或 `ParseMultipartForm`，组件保留缓存表单语义。此时不能重新测量已被读取的原始编码请求体；仍检查可用的 Content-Length、原始查询串、缓存 PostForm 的解码键值总字节数，以及合并 Form 的解码键值总字节数。PostForm 上限为 MaxBodyBytes，POST 合并 Form 上限为 MaxBodyBytes + MaxQueryBytes，GET 合并 Form 上限为 MaxQueryBytes。重复键按每个值重复计数；这些解码检查不分配新的编码副本，也不等同于对上游已完成的内存分配施加限制。

因此，在解析请求体之前配置网关或中间件读取上限，并处理其解析错误；不要忽略解析错误后将部分表单交给 SSO。外层已有的更小 `MaxBytesReader` 上限仍有效，其超限错误同样映射为 413。完整部署示例保留外层 64 KiB 限制，覆盖应用自身登录表单；提高库内限制不会提高外层限制。

本模块沿用标准库 `ParseForm` 的媒体类型语义，不新增 JSON、multipart 文件上传或压缩请求体解析。不会为了检查一个不参与协议解析的请求体而主动读取它。请求头、完整请求行、连接数及读取超时由代理和 `http.Server` 控制；应用自定义登录接口、上传接口及自行编写的回调需要各自设置限制。

## 错误与副作用

- 超限返回 `ErrRequestTooLarge`，内置 HTTP 入口返回 **413**。中心保持 JSON `Response` 格式；子应用回调沿用文本错误格式。
- 畸形查询串、表单及读取错误返回固定 `ErrInvalidRequest`／**400**，不回显解析器或读取器诊断。浏览器回调仍使用固定的浏览器请求错误文本。
- 被拒绝的输入不会占用 nonce、消费 state、消费或撤销凭证、登记回调、触发业务登录／注销钩子或注销通知。合法修正请求仍需满足原有时效、签名、PKCE 及一次性使用规则。
- 本次只统一输入解析错误；业务钩子、存储等其他错误的原有返回策略未在本模块统一改写。

同一个 `http.Request` 一旦在 SSO 表单解析／大小检查中失败，后续校验保留该失败结果，避免请求体耗尽后被误当作合法的纯查询串请求。保留 Context 的请求副本也保留失败结果。修正输入时应创建新请求，不要修改原请求重试；这不会占用 nonce 或消费凭证。上游在调用 SSO 前忽略的解析错误仍需由上游自行处理。

新增 [request_reparse_test.go](request_reparse_test.go) 包含三组回归：重复解析、五个中心表单入口和注销回调，均未执行。下列 `TestRequestLimits` 专项命令也包含这些用例。

## 升级与验证

这是入站默认限制的收紧。已有大查询串、大扩展表单或上游注入的大参数表可能开始返回 413；升级前核对实际编码大小，并配置接收方及代理，而不是关闭签名。原签名算法、单值参数规则和四种凭证模式保持原语义。

[request_limits_test.go](request_limits_test.go) 覆盖边界值、编码字节、未知／虚报长度、读取上限、解析错误脱敏、缓存表单、外层限制、全部中心入口、四种凭证、nonce 保留、注销回调和四种浏览器流程。仅完成格式化与静态检查，未执行编译、测试或竞态检查。由使用者在仓库根目录运行：

```powershell
go test ./sso/... -run TestRequestLimits -count=1
go test -race ./sso/... -run TestRequestLimits -count=1
```

这些用例使用内存存储、httptest Recorder 和进程内传输，不监听端口或连接 Redis。完整回归见[测试指南](TESTING_zh.md)。
