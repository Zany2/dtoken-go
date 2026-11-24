# SSO 测试说明

本文档说明 SSO 独立模块的推荐验证方式，覆盖内存模式、Redis 模式、Server/Client 联调和统一登出。

## 发布集成检查

现有 CI 已加入 SSO、其 Redis 适配器及全部五个 SSO 示例的独立竞态步骤。仓库外用例 [sso-consumer_test.go](../.github/scripts/testdata/sso-consumer_test.go) 只使用公开 API，覆盖四种模式、带校验的构造入口、HTTP 门面、签名、自定义参数名、防重放配置、PKCE 及消费／撤销后的失效。由 [check-consumer.sh](../.github/scripts/check-consumer.sh) 复制到临时外部模块执行，普通模块测试命令不会自动发现此 `testdata` 用例。

中英文配置指南已提供完整的标记程序，由现有文档编译工具读取。以上新增检查尚未执行，命令、发布版本选择及验证边界见[升级指南](MIGRATION_zh.md)。

新增 SSO 最低 Go CI 矩阵及专用四模块消费者，覆盖发布的 Redis 构造／所有权和浏览器登录至账号下线公开 API。专用消费者关闭工作区、禁止工具链自动升级，并执行竞态检查；不连接 Redis。脚本、模块发现回归及所有新增 CI 均尚未运行，命令见[专用发布检查](MIGRATION_zh.md#sso-专用发布检查)。

## 单元测试

中心错误回归覆盖六个路由、私有诊断、初始化入口、防重放存储恢复、四种凭证模式及兑换前后失败。退出回归另检查严格／尽力结果的观察次数。源码尚未运行，详见 [HTTP 错误验证](HTTP_ERRORS_zh.md#生命周期与验证)。

Cookie 轮换回归源码覆盖 Cookie 信任范围、仅主密钥签发、准备／切换／退役、签名期限及待完成 PKCE 状态。部署用例通过共享内存存储验证新旧副本接续、中心会话复用及轮换后的撤销。这些用例尚未执行；配置、定向命令和上线边界见 [Cookie 轮换](SIGNING_ROTATION_zh.md#浏览器会话与状态-cookie-轮换)。

[logout_callback_errors_test.go](logout_callback_errors_test.go) 覆盖普通／原子防重放存储及业务钩子的回调错误脱敏、直接 API 诊断保留、协议状态分类，以及钩子失败后使用新签名请求重试。这些回归仅完成静态检查，尚未执行；行为说明见[回调重试规则](LOGOUT_RELIABILITY_zh.md#重试规则)。

新增 [request_reparse_test.go](request_reparse_test.go) 检查请求体解析失败后同一请求的重复校验，以及新请求仍可使用未消费 nonce／凭证。部署 HTTP 验收同时要求捕获的旧 Cookie 非空，撤销后的跳转匹配中心授权入口。仅静态复核，新增回归未执行。

[request_limits_test.go](request_limits_test.go) 补充入站大小限制回归，覆盖所有中心入口、四种凭证与浏览器流程、流式读取、缓存表单、错误脱敏及拒绝后 nonce／state／凭证保留。源码尚未执行，专项命令及兼容说明见[HTTP 请求大小限制](REQUEST_LIMITS_zh.md)。

[signing_keys_test.go](signing_keys_test.go) 新增六组轮换回归，覆盖信任范围、六个动作的 nonce 连续性、回调密钥退役、四种凭证模式、重试签名及配置错误。部署模块另有交错密钥下的登录／注销和配置校验用例。尚未执行，命令见[密钥轮换验证](SIGNING_ROTATION_zh.md)。

[request_validation_test.go](request_validation_test.go) 新增 8 组请求一致性回归，覆盖全部协议单值字段、四种凭证模式、签名／未签名请求、Query／Body 重复、自定义参数名、拒绝前不消费 nonce／state／凭证，以及不触发业务与注销副作用。测试尚未执行，兼容说明及专项命令见[请求参数一致性](REQUEST_VALIDATION_zh.md)。

本次整体静态复核补充了以下取消／超时回归矩阵，共 9 个测试函数及其参数化场景。源码已补齐，**未执行**测试：

| 源码 | 验证边界 |
| --- | --- |
| [cancellation_center_review_test.go](cancellation_center_review_test.go) | 四种模式签发前取消、存储写入后的取消；四种模式的 introspect/userinfo 钩子返回；Ticket／Code 已消费后的补充钩子取消 |
| [cancellation_logout_review_test.go](cancellation_logout_review_test.go) | 传输和响应体忽略单次超时、超时重试与恢复、父请求取消分类、中心注销各阶段停止、客户端确认与 nonce 消费 |
| [cancellation_browser_review_test.go](cancellation_browser_review_test.go) | 浏览器状态写入取消、登录钩子取消、本地／全局退出钩子取消、已消费 state 不恢复 |

预期结果是取消后不返回身份、成功跳转或注销确认，不再推进后续有副作用的阶段。已经完成的存储写入或业务钩子不回滚，已消费的一次性凭证与 nonce 不恢复。自定义钩子／传输仍需遵守 Context；检查发生在调用返回之后，不能中断一直不返回的实现。

可先执行本批专项，再按下文及[发布验收清单](RELEASE_CHECKLIST_zh.md)执行完整模块与部署验证：

```powershell
go test ./sso/... -run TestCancellationReview -count=1
go test -race ./sso/... -run TestCancellationReview -count=1
```

这些新增用例使用内存存储和受控传输，不连接真实 Redis 或外部服务。格式化和静态源码复核不能代替 Go 编译、测试或竞态检查结果。

账号下线回归源码（`account_logout_test.go`、`example_account_logout_test.go` 及部署模块的 `account_logout_test.go`）覆盖通知前撤销、四种凭证、普通／共享存储、阶段故障恢复、取消、结果脱敏、禁用 SLO，以及实际 Auth 会话和多副本签名通知闭环。这些源码尚未执行，专项命令和生命周期边界见[账号下线指南](ACCOUNT_LOGOUT_zh.md)。

本轮收尾新增 `origin_review_test.go`、`client_cancellation_test.go` 和 `examples/sso_deployment/lifecycle_review_test.go`，覆盖同源等价与精确回调绑定、非法端口／SameSite、取消后的延迟成功响应、中心绝对期限及资源释放前等待已取消处理器返回。同时删除早期回归源码对无资源 SSO 内存存储的非法 `Close` 调用。以上用例尚未执行，统一执行入口见[发布验收清单](RELEASE_CHECKLIST_zh.md)。

在仓库根目录执行。SSO 模块默认使用内存存储，命令应包含其子包：

```powershell
go test ./sso/... -v
```

Redis 和各示例应用是独立的工作区模块，不包含在上述命令中：

```powershell
go test ./sso/storage/redis/... -v
go test ./examples/sso_server/... ./examples/sso_client/... -v
go test ./examples/sso_gin_server/... ./examples/sso_gin_client/... -v
go test ./examples/sso_deployment/... -v
```

Redis 构造器测试不需要服务；未设置 `DTOKEN_REDIS_URL` 时，Redis 集成流程会跳过。这些命令不会启动示例应用。

重点覆盖：

- 完整部署示例补充实际处理器的多应用／副本登录、签名全局注销、伪造回调拒绝、已消费浏览器状态、部署配置、登录／注销来源校验、就绪故障、Cookie 策略及借用存储所有权。这些示例用例已编写但未执行，Redis 和浏览器验证见[部署验收](DEPLOYMENT_zh.md)。

- 配置与启动回归源码（`config_validation_test.go`）：非法路由及解码冲突不触发 mux panic、存储写入前拒绝、借用存储所有权、启动存储故障、注册元数据、签名／Cookie／防重放组合、安全诊断、Browser／RemoteAuth 构造接入，以及带校验入口的签名授权／换票。本轮新增用例尚未执行，接入方式见[配置说明](CONFIGURATION_zh.md)。

- Ticket 生成、校验、消费、撤销和过期。
- 共享 Token 生成、校验、撤销和过期。
- 远程会话创建、校验、续期、撤销和过期。
- OAuth2 Code 生成、消费、撤销和边界错误。
- HTTP 协议端点：`authorize`、`token`、`introspect`、`userinfo`、`revoke`、`logout`。
- ClientApp：授权 URL、换票、验签、统一登出回调 Handler。
- 客户端协议响应限制：默认及自定义上限、恰好达到边界、未知或不准确的 Content-Length、成功及错误响应、边界处和之前的读取故障、响应体关闭，以及换票、自省、用户信息、撤销入口失败时不返回部分数据。两种换票模式拒绝小于 -1 的有效期，并保留合法特殊值及大整数精度。
- 换票结果扩展保持凭证身份及授权范围，传递中心会话信息；失败不返回部分数据、不新增回调登记，且钩子失败后不能重放已消费凭证。
- 四种模式均通过可信钩子获取授权元数据；非法请求在钩子前被拒绝，钩子失败不签发凭证、不登记回调。
- 账号注销使四种凭证跨客户端、跨共享存储的 Server 实例失效，覆盖命名空间隔离、普通存储回退、注销后重新签发及旧记录兼容。
- 注销版本存储错误正常上报，注销失败保留 Cookie，Ticket/Code 原子消费在取出凭证后再次检查撤销状态。
- ClientSession：登记子系统会话、覆盖更新、查询和清理。
- 独立签名密钥优先级、全局密钥兼容、密钥轮换，以及跨客户端签名冒用拒绝。
- 同一次单点注销向不同客户端使用各自的密钥签名，支持独立密钥与默认密钥混用。
- 注销确认区分 HTTP 成功和业务成功；格式错误、过大或读取失败的响应保留严格模式重试状态，并报告所有失败客户端 ID。
- 四种模式下带签名的 ClientApp/HTTPServer 完整流程，覆盖自定义参数名、路由前缀以及原子存储和普通存储两条路径。
- 客户端或回调地址绑定错误不会消费凭证；已消费或撤销的凭证不能继续获取用户信息。
- 授权重定向及 JSON 响应禁止缓存；损坏的注销表单返回 HTTP 400，且不触发本地注销。
- 两个客户端示例均检查浏览器状态、服务端会话过期和换票成功后的会话替换。
- 两个客户端示例根据中心 Token/Session 的正数有效期限制本地期限，计入换票耗时，处理未指定、无限及超大值，并在非法有效期下保留已有会话；换票扩展测试同时检查 Cookie 与本地存储的截止时间。
- 两个中心示例组合 Auth 登录、可信会话绑定、换票扩展及账号注销；用例覆盖复制 Cookie、多浏览器、账号隔离、已撤销及未绑定 Ticket、不泄露 Token，以及严格和尽力回调策略。
- introspect/userinfo 的应用状态检查覆盖四种模式：会话失效不披露元数据，查询故障保留错误，无效客户端或凭证不调用钩子，未配置钩子保持原行为，撤销不依赖钩子。两个中心均在仅撤销原 Auth Token 后拒绝读取，即使 SSO Ticket 与另一浏览器仍有效。

## 防重放保护

防重放回归测试覆盖时间边界、缺失或重复元数据、操作绑定、无效签名隔离、跨实例原子占用、普通存储串行回退、存储故障与 nonce 保留时间。HTTP 用例使用自定义路径和参数名覆盖四种凭证模式、授权登录往返及回调失败重试。

可在工作区根目录手动执行：

```powershell
go test ./sso/... -run TestReplay -count=1
go test -race ./sso/... -run TestReplay -count=1
```

这些测试使用进程内 HTTP 处理器和内存存储，不代表实际 Redis 连通性或部署时钟同步已经验证。部署验证时应同时启用双方配置，多副本共用存储、前缀和时间窗口，再确认原样重放请求被拒绝、重新生成的请求可以正常处理。配置及重试语义见 [防重放指南](./REPLAY_PROTECTION_zh.md)。

## 浏览器接入回归

`browser_test.go` 与 `browser_protocol_test.go` 覆盖多标签页及跨浏览器绑定、重复回调、状态过期、返回路径、四种凭证模式与防重放组合、存储消费并发与故障、会话期限和退出来源校验。`examples/sso_client/browser_test.go` 检查本地会话替换和全局退出前的本地撤销。可从工作区根目录手动执行：

```powershell
go test ./sso/... -run TestBrowser -count=1
go test -race ./sso/... -run TestBrowser -count=1
go test ./examples/sso_client/... -run TestBrowserDemoLifecycle -count=1
```

这些回归测试不替代真实浏览器对 SameSite、Secure、Cookie 路径及多标签页的验证，也不替代共享 Redis 部署检查。完整接入说明见 [浏览器客户端指南](./BROWSER_CLIENT_zh.md)。

## 单点注销可靠性回归

`logout_notifications_test.go` 和 `logout_notification_lifecycle_test.go` 覆盖失败目标重试、不可重试确认错误、响应资源释放、多请求共享并发上限、单次超时、父请求取消、登记删除与白名单变更、重试期间密钥轮换和新 nonce、严格／尽力阶段结果，以及观察钩子与 HTTP 脱敏。可从工作区根目录手动执行：

```powershell
go test ./sso/... -run 'TestLogoutNotification|TestLogoutAcknowledgement|TestLogoutReportsAllFailedClients|TestCenterLogoutLifecycle' -count=1
go test -race ./sso/... -run TestLogoutNotification -count=1
go test ./examples/sso_server/... ./examples/sso_gin_server/... -count=1
```

用例使用内存存储与可控制的 HTTP Transport，不替代真实网络超时、Redis 故障恢复或多副本容量验证。配置及生命周期边界见 [单点注销可靠性](./LOGOUT_RELIABILITY_zh.md)。

## Gin 示例联调

启动统一登录中心：

```powershell
go run ./examples/sso_gin_server
```

启动子系统：

```powershell
go run ./examples/sso_gin_client
```

访问受保护资源：

```text
http://localhost:9101/protected
```

验证流程：

1. 未登录子系统生成带签名、有效期五分钟的浏览器状态 Cookie，跳转 `/sso/authorize`；中心未登录时再跳转 `http://localhost:9100/login`。
2. 登录中心写入中心 Cookie，并重定向回 `/sso/authorize`。
3. 登录中心生成 Ticket，跳回子系统 `/sso/callback`，并通过 `back` 回传浏览器状态。
4. 子系统先校验签名状态 Cookie 与 `back`，再调用 `/sso/token` 换取 `loginId`；状态缺失、不匹配、篡改或过期均不得触发换票。
5. 子系统创建带两小时服务端截止时间的本地会话，访问 `/protected` 返回登录主体，复制 Cookie 不能延长会话。

每个浏览器只保留一个待完成授权流程。状态校验通过后，即使换票失败也会清除浏览器状态 Cookie；重试时重新访问 `/protected`，不要刷新回调地址。重启任一示例进程都会使该进程的本地状态失效。

## 统一登出验证

登录成功并保留登录中心 Cookie 后访问：

```text
http://localhost:9100/sso/logout
```

服务端会通过可信登录解析器确定注销主体，不能使用 `loginId` 查询参数任意指定账号。

两个中心均通过 Auth 实例校验 Cookie。注销成功后账号的所有中心浏览器会话均失效，重放保存的 Cookie 发起授权应跳转至 `/login`；其他账号仍可授权，重新登录后可签发可用 Ticket。换票结果应返回正数 `remainTokenTimeout` 与 `deviceId`，且不泄露中心 Token。在授权与换票之间撤销原 Auth Token，可检查其他仍在线浏览器不能替代该 Ticket 绑定的会话。

客户端回调成功时：

- SSO Server 清除中心 Cookie。
- SSO Server 向已登记的 Client 回调 `/sso/logout-callback`。
- Client 根据 `loginId` 删除本地会话。
- 再次访问 `http://localhost:9101/protected` 会重新跳转到登录中心。

标准库服务端采用严格单点注销：回调失败时保留中心 Cookie 和客户端会话索引，便于重试，但不回滚已经成功的客户端回调。Gin 服务端采用尽力单点注销：即使回调失败，也清除中心 Cookie 和索引。因此，不可达客户端可能保留本地会话，直到本地注销或会话到期；仅凭中心返回成功，不能确认所有客户端均已注销。

客户端 `/logout` 只删除当前浏览器的本地会话。中心仍登录时，再次访问 `/protected` 可能立即重新登录。

如果开启签名，把 Server 和 Client 的 `SecretKey` 设置一致，并将 `CheckSign` 设为 `true`。Client 侧 `LogoutCallbackHandler` 会自动校验回调签名。

`LogoutCallbackHandler` 要求匹配配置的客户端 ID，且时间戳与客户端时钟相差不超过五分钟。这能拒绝过旧请求，但不会对窗口内的有效回调去重，本地注销操作应保持幂等。关闭签名时，仅检查客户端 ID 和时间戳不能认证发送方。

## Redis 模式验证

真实 Redis 的[多副本生命周期用例](storage/redis/REPLICAS_zh.md)新增四组流程：PKCE 凭证并发消费、授权重放与密钥轮换、浏览器状态跨连接恢复、账号下线部分失败后的跨中心恢复。需显式配置 `DTOKEN_REDIS_URL`；CI 的 Redis 步骤对 SSO Redis 模块启用竞态检测。用例不启动示例进程，本轮未执行。

生产部署建议使用 Redis 存储：

```go
import (
	"github.com/Zany2/dtoken-go/sso"
	ssoredis "github.com/Zany2/dtoken-go/sso/storage/redis"
)

server, err := ssoredis.NewServer(
	"redis://:password@127.0.0.1:6379/0",
	sso.WithKeyPrefix("dtoken:"),
	sso.WithAuthType("sso:"),
	sso.WithConfig(sso.DefaultConfig()),
)
if err != nil {
	return err
}
defer server.Close()
```

使用上述配置时，Redis 下建议重点观察以下键前缀：

- `dtoken:sso:sso:client:`：子系统注册信息。
- `dtoken:sso:sso:ticket:`：一次性 Ticket，消费后应删除。
- `dtoken:sso:sso:oauth2:code:`：OAuth2 Code，消费后应删除。
- `dtoken:sso:sso:client-session:`：统一登出使用的子系统会话记录，按配置的注销策略完成后清理。
- `dtoken:sso:sso:logout-version:`：永久保存的账号撤销标记，关联凭证仍可能存在时不可单独删除。

键由 `keyPrefix + authType + 键后缀 + 标识` 拼接而成。后缀常量本身已带 `sso:`，因此上述配置会产生相邻的两个 `sso:` 段。

验证建议：

1. 登录前确认 Redis 中已存在客户端注册 key。
2. 发起登录后，观察 Ticket key 短暂出现。
3. 子系统换票成功后，Ticket key 应被删除。
4. 登录中心按配置策略完成 `/sso/logout` 后，`dtoken:sso:sso:client-session:` 对应键应被删除。
5. 账号的 `dtoken:sso:sso:logout-version:` 标记应保留。此前签发的带版本凭证应失效，重新签发的凭证应可正常校验。

可选集成测试：

```powershell
$env:DTOKEN_REDIS_URL="redis://:password@127.0.0.1:6379/0"
go test ./sso/storage/redis/... -v
```

没有设置 `DTOKEN_REDIS_URL` 时，该测试会自动跳过。

请使用支持 `GETDEL` 的测试 Redis 实例（Redis 6.2+）。集成测试每次使用独立键前缀，并在关闭连接前清理记录，覆盖 Ticket/Code 消费、共享 Token 撤销、远程会话续期和撤销，以及客户端会话清理。集成测试被跳过不代表已验证真实 Redis 部署。

## 安全边界

- 生产环境建议开启 `CheckSign` 并设置 `SecretKey`。
- Ticket 和 OAuth2 Code 的 `redirect` 必须完整匹配客户端 `RedirectURIs`。
- 统一登出 `callback` 必须属于当前客户端：完整匹配 `RedirectURIs`、与某个 `RedirectURIs` 同源，或匹配 `AllowOrigins`。
- 不建议把过宽的域名加入 `AllowOrigins`，避免恶意回调地址造成 SSRF 风险。
- 注销回调默认 5 分钟有效，超出窗口会被 Client 侧拒绝。

## API 命名稳定性

当前建议保持以下命名：

| 名称 | 定位 |
| --- | --- |
| `ClientApp` | 子系统侧接入辅助对象 |
| `ClientSession` | 登录中心记录的“登录主体 - 子系统”绑定 |
| `LogoutCallback` | 子系统收到的统一登出回调数据 |
| `VerifyLogoutCallback` | 子系统侧手动验签和解析回调 |
| `LogoutCallbackHandler` | 子系统侧标准回调 Handler |
| `LogoutCallbackBestEffort` | 服务端推送失败时是否仍清理中心记录 |

这些命名与当前 SSO 职责一致，暂时不建议继续拆分或改名。

## PKCE 授权绑定回归

以下命令供开发者手动执行，本轮未运行：

```powershell
go test ./sso/... -run 'TestPKCE|TestBrowserProtocolModes' -count=1
go test -race ./sso/... -run 'TestPKCEConsumptionLifecycle|TestPKCEBrowserFlowIsolation' -count=1
go test ./examples/sso_client/... -run TestBrowserDemo -count=1
```

覆盖 RFC 7636 向量、S256 格式、当前客户端策略、旧接口绕过、普通/原子存储、错误证明保留凭证、并发单次消费、注销版本失效、HTTP 身份查询旁路、POST 证明提交、签名与自定义参数，以及多标签页串票。真实 Redis 和真实浏览器仍需部署后联调；本轮仅完成格式化与静态检查。配置及升级步骤见 [PKCE 指南](PKCE_zh.md)。

## 远程会话鉴权回归

以下命令供开发者手动执行，本轮未运行：

```powershell
go test ./sso/... -run 'TestRemoteAuth|TestRemoteBearerCredential' -count=1
```

回归源码覆盖远程会话和共享 Token、签名防重放、自定义协议名、逐请求检查、撤销与账号注销、范围匹配、Bearer 歧义输入、身份绑定、超时取消、业务 Context、响应大小限制及错误脱敏。示例源码为 `example_remote_auth_test.go`。本轮仅完成格式化和静态检查，未运行服务、构建、测试或真实存储联调。接入说明见 [远程鉴权指南](REMOTE_AUTH_zh.md)。

## 动态授权策略回归

以下命令供开发者手动执行，本轮未运行：

```powershell
go test ./sso/... -run 'TestAuthorization|TestExchangeResultEnrichment|TestPKCE|TestRemoteAuth' -count=1
```

新增源码覆盖四种模式的签发与检查、nil／空范围语义、禁止权限扩张、可信策略输入、普通／原子存储下失败保留凭证、PKCE 前置校验、回调登记、无 SLO 兑换、策略错误脱敏、取消后的拒绝、签名防重放浏览器流程及远程鉴权联动。仅完成格式化和静态检查，未运行测试、构建、示例或真实存储联调。配置及范围边界见 [动态授权策略指南](AUTHORIZATION_POLICY_zh.md)。

## 已启动服务验收

`examples/sso_deployment/acceptance` 新增一个显式启用的真实 HTTP 场景及两个无网络客户端边界用例。普通示例模块测试默认跳过真实访问；设置 `SSO_ACCEPTANCE_RUN=true` 会对配置的环境登录和全局退出测试账号，需使用专用账号。变量、预期状态与故障矩阵见[验收指南](../examples/sso_deployment/ACCEPTANCE_zh.md)。本轮未运行任何用例或服务。
