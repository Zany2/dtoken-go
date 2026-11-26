# SSO 升级与发布指南

[English](MIGRATION.md) · [发布验收清单](RELEASE_CHECKLIST_zh.md)

本文汇总已有 SSO 调用方升级当前源码需要核对的行为，以及工作区之外的交付验证。本文不指定发布版本号，也不表示当前改动已通过构建、回归或真实部署验收。

## 升级前核对

| 范围 | 需要确认的接入方式 |
| --- | --- |
| 构造与默认值 | 原构造函数保留；优先采用 `NewHTTPServerChecked`、`NewClientAppChecked`。从 `DefaultHTTPOptions()`／`DefaultClientConfig()` 开始才能继承布尔默认值；零值结构体不会自动启用签名或 SLO |
| 请求参数 | 协议字段只提交一次，Query 和 Body 不能重复；多个 scope 放在单个值中。凭证操作只携带当前模式的字段，不能把其他凭证字段作为空值一起提交 |
| 入站大小 | 中心和子应用接收方新增默认查询串 8 KiB、表单体 64 KiB 限制，超限返回 413；大扩展参数需协调 RequestLimits 与上游代理上限。提前解析中间件需自行限制原始读取并处理解析错误，见[请求大小限制](REQUEST_LIMITS_zh.md) |
| 模式选择 | SDK 使用请求 Mode 或配置 Mode，不按非空结构体字段自动切换。原始 HTTP `/token` 未提供 mode 时仍按非空 code 推断，其余凭证端点默认 Ticket |
| URL 与 POST | 使用 `ExchangeTicket`／`ExchangeOAuth2Code` 提交兑换请求。旧兑换 URL 辅助方法不适合作为携带客户端密钥的接入方式；PKCE verifier 只通过 POST 提交 |
| 密钥与来源 | 客户端独立签名密钥优先于中心默认密钥，兑换密钥不充当签名密钥。回调地址保持精确匹配；同源比较的规范化不会放宽回调白名单 |
| 防重放与 PKCE | 两端协调启用，统一参数名及时间窗口。先让子应用发送 PKCE 证明，再对中心注册项启用 `RequirePKCE`；已有未携带证明的流程应重新发起 |
| 浏览器流程 | `BrowserClient` 的服务端 state、Cookie 和 PKCE verifier 必须作为同一流程使用。切换旧手写回调时重新发起登录；一次性凭证已消费后的失败不能重放 |
| 注销 | 应用配置可信 `LoginIDResolver` 和 `CenterLogout`。HTTP 全局退出与可信 `LogoutAccount` 的顺序不同，见下文；通知回调必须幂等 |
| 存储与生命周期 | 同一服务副本共享存储及命名空间，应用负责借用存储的关闭。普通 Storage 仍受支持，但不具备原子扩展时不能承诺跨副本一次性竞争保证 |

更多错误分类与细节见[配置](CONFIGURATION_zh.md)、[请求参数一致性](REQUEST_VALIDATION_zh.md)、[浏览器客户端](BROWSER_CLIENT_zh.md)及[防重放](REPLAY_PROTECTION_zh.md)。上述约束适用于本项目协议，不能据此认定与 Java Sa-Token、通用 OAuth2 或 OIDC 实现可以直接互通。

## 注销及滚动升级边界

普通 HTTP 全局退出先通知子应用，再撤销中心会话和 SSO 凭证；严格通知失败会保留后续阶段以便重试。可信后台 `LogoutAccount` 则先撤销中心和旧 SSO 凭证，再通知子应用，失败时保留未完成的绑定供重试。已完成的步骤不回滚。永久禁用账号还需要身份系统阻止新登录和新授权。

账号撤销依赖所有参与副本都执行注销版本检查。混跑不理解该检查的旧版本，不能承诺账号下线在所有副本一致生效。更新全部参与验证／消费凭证的服务，再验收账号下线及跨副本重放拒绝。不要通过删除注销版本记录或更换存储前缀完成回滚。

签名支持可选 VerificationKey 双密钥验签；先升级所有接收方，再按[轮换指南](SIGNING_ROTATION_zh.md)准备、切换和退役密钥。Cookie 密钥或浏览器状态格式变化可能要求重新登录。停止旧副本前排空在途请求，保留仍在有效期内的凭证、nonce 和注销版本记录。具体部署和故障边界见[部署指南](DEPLOYMENT_zh.md)。

## 模块与发布边界

SSO、SSO Redis 适配器和五个示例是独立 Go 模块。根目录 `go test ./...` 不包含这些嵌套模块。工作区声明的 Go 版本与单模块 `go.mod` 的最低版本是不同验证目标；CI 已为 SSO 及其 Redis 适配器增加读取各自 go.mod 的最低 Go 检查；配置了检查入口不表示已经通过运行验证。

发布时先提供满足依赖要求的根模块；SSO 依赖根模块，通用 Redis 存储适配器也依赖根模块，SSO Redis 适配器再依赖 SSO 和通用 Redis 适配器。具体版本以各模块 `go.mod` 为准，发布前更新实际依赖和对应子模块标签。本文不代替已有[模块发布迁移流程](../docs/design/module-migration_zh.md)。

现有隔离检查器支持选择模块。在仓库根目录手动执行：

```powershell
go run ./.github/scripts/check-modules ./sso ./sso/storage/redis ./examples/sso_server ./examples/sso_client ./examples/sso_gin_server ./examples/sso_gin_client ./examples/sso_deployment
go run ./.github/scripts/check-docs
```

第一条命令复制选中模块到临时目录，禁用工作区并添加当前源码替换，然后运行 vet／test。它检查源码模块依赖边界，不能证明发布版本可下载。第二条命令编译标记过的文档示例，已包含中英文 SSO 配置示例；不会运行示例，也不会编译所有未标记代码块。

仓库外使用方检查使用现有 Bash 入口，Windows 可在 Git Bash 中执行：

```bash
# 当前源码：核心、Gin 与 SSO；临时模块关闭工作区，使用源码替换。
bash .github/scripts/check-consumer.sh

# 已发布版本：先将变量设为实际发布标签，不使用源码替换。
bash .github/scripts/check-consumer.sh "$ROOT_VERSION" "$GIN_VERSION" "$SSO_VERSION"
```

保留旧的双版本参数形式，只检查核心及 Gin；只有三版本形式才包含发布的 SSO。依赖解析后会检查实际选中版本是否等于请求版本，防止较高的传递依赖版本悄悄代替待验版本。此使用方用例覆盖四种模式的公开构造、HTTP 门面、签名、自定义参数名、防重放配置、Ticket／Code 的 PKCE 兑换以及消费／撤销后的失效；使用内存存储与进程内 HTTP 传输，不监听端口。它不包含发布的 Redis 适配器，后者仍需独立验证。

## SSO 专用发布检查

SSO 的四模块发布集合为根模块、`sso`、`storage/redis` 和 `sso/storage/redis`。新增脚本不依赖 Gin，也不加载 `go.work`；它在临时消费者模块中检查公开 API。以下命令由使用者运行，Windows 使用 Git Bash：

```bash
# 当前源码：只替换这四个模块。
bash .github/scripts/check-sso-consumer.sh

# 发布版本：先设置四个实际版本号，不添加源码替换。
bash .github/scripts/check-sso-consumer.sh "$ROOT_VERSION" "$SSO_VERSION" "$REDIS_VERSION" "$SSO_REDIS_VERSION"

# 安装与目标 go.mod 匹配的 Go 后，检查独立模块副本。
GOWORK=off GOTOOLCHAIN=local go run ./.github/scripts/check-modules -sso ./sso
GOWORK=off GOTOOLCHAIN=local go run ./.github/scripts/check-modules -sso ./sso/storage/redis
```

`-sso` 不带模块路径时检查 SSO 和 SSO Redis；仅通过四个 go.mod 发现源码替换集合，不执行工作区模块发现。副本检查使用 readonly vet/test，源码的 go.mod/go.sum 不被改写。脚本的模块发现回归也已补充，尚未执行。

专用消费者脚本接受零个或四个参数，发布模式只接受显式版本号（含预发布或伪版本），拒绝空值、分支和 latest。依赖解析后逐一核对最终选中版本，并拒绝任何替换；较高的传递依赖不会被当作指定版本验证成功。脚本固定 `GOWORK=off`、`GOTOOLCHAIN=local`，不会自动升级工具链；当前源码使用方同时验证两个适配器的组合，所用 Go 必须满足全部依赖要求。

验收源码覆盖四种凭证模式、签名、防重放、PKCE，以及浏览器登录、请求上限拒绝后 state 保留、双密钥过渡、账号下线和 Redis 适配器借用／持有连接的关闭语义。消费者通过进程内 HTTP 传输执行；Redis 构造用例阻止网络拨号，不代表真实 Redis 运行验收。浏览器用例只以标记模拟本地会话，不替代完整部署示例中的 Auth 会话联调。脚本运行 vet 和 race test，需要可用的竞态工具链。

新增 [SSO published consumer 工作流](../.github/workflows/sso-release.yml) 仅手动触发，四个版本必填；它只验收已发布模块，不创建标签或执行发布。使用所选源码 ref 中 sso/go.mod 声明的 Go 版本和验收源码，不承诺兼容缺少新 API 的旧版本。常规 CI 的 `SSO minimum Go` 矩阵检查 SSO 和 Redis 适配器，并在 SSO 项下运行当前源码专用消费者。

## CI 与验收记录

现有工作区 CI 已有模块遍历与真实 Redis 场景。本轮增加独立 SSO、Redis 适配器和五个示例的竞态步骤；文档检查增加 SSO 配置示例；常规仓库外使用方增加 SSO。手动 CI 可同时填写根模块、Gin 和 SSO 的实际版本进行发布使用方检查。竞态步骤不设置 Redis 地址，真实 Redis 场景仍由单独步骤负责。

本轮仅静态检查和格式化，未执行上述命令或 CI。发布记录应至少包含代码版本（以及未提交改动）、Go 版本／平台、各命令退出结果、Redis 是否实际运行、跳过项，以及真实浏览器／代理验收结论。完整命令见[发布验收清单](RELEASE_CHECKLIST_zh.md)。
