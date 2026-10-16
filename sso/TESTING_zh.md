# SSO 测试说明

本文档说明 SSO 独立模块的推荐验证方式，覆盖内存模式、Redis 模式、Server/Client 联调和统一登出。

## 单元测试

在仓库根目录执行。SSO 模块默认使用内存存储，命令应包含其子包：

```powershell
go test ./sso/... -v
```

Redis 和四个示例是独立的工作区模块，不包含在上述命令中：

```powershell
go test ./sso/storage/redis/... -v
go test ./examples/sso_server/... ./examples/sso_client/... -v
go test ./examples/sso_gin_server/... ./examples/sso_gin_client/... -v
```

Redis 构造器测试不需要服务；未设置 `DTOKEN_REDIS_URL` 时，Redis 集成流程会跳过。这些命令不会启动示例应用。

重点覆盖：

- Ticket 生成、校验、消费、撤销和过期。
- 共享 Token 生成、校验、撤销和过期。
- 远程会话创建、校验、续期、撤销和过期。
- OAuth2 Code 生成、消费、撤销和边界错误。
- HTTP 协议端点：`authorize`、`token`、`introspect`、`userinfo`、`revoke`、`logout`。
- ClientApp：授权 URL、换票、验签、统一登出回调 Handler。
- ClientSession：登记子系统会话、覆盖更新、查询和清理。
- 四种模式下带签名的 ClientApp/HTTPServer 完整流程，覆盖自定义参数名、路由前缀以及原子存储和普通存储两条路径。
- 客户端或回调地址绑定错误不会消费凭证；已消费或撤销的凭证不能继续获取用户信息。
- 授权重定向及 JSON 响应禁止缓存；损坏的注销表单返回 HTTP 400，且不触发本地注销。
- 两个客户端示例均检查浏览器状态、服务端会话过期和换票成功后的会话替换。

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

键由 `keyPrefix + authType + 键后缀 + 标识` 拼接而成。后缀常量本身已带 `sso:`，因此上述配置会产生相邻的两个 `sso:` 段。

验证建议：

1. 登录前确认 Redis 中已存在客户端注册 key。
2. 发起登录后，观察 Ticket key 短暂出现。
3. 子系统换票成功后，Ticket key 应被删除。
4. 登录中心按配置策略完成 `/sso/logout` 后，`dtoken:sso:sso:client-session:` 对应键应被删除。

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
