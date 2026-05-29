# SSO 测试说明

本文档说明 SSO 独立模块的推荐验证方式，覆盖内存模式、Redis 模式、Server/Client 联调和统一登出。

## 单元测试

SSO 根模块默认使用内存存储，可以直接运行：

```powershell
go test ./sso -v
```

重点覆盖：

- Ticket 生成、校验、消费、撤销和过期。
- 共享 Token 生成、校验、撤销和过期。
- 远程会话创建、校验、续期、撤销和过期。
- OAuth2 Code 生成、消费、撤销和边界错误。
- HTTP 协议端点：`authorize`、`token`、`introspect`、`userinfo`、`revoke`、`logout`。
- ClientApp：授权 URL、换票、验签、统一登出回调 Handler。
- ClientSession：登记子系统会话、覆盖更新、查询和清理。

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

1. 子系统未登录，跳转到 `http://localhost:9100/login`。
2. 登录中心写入中心 Cookie，并重定向回 `/sso/authorize`。
3. 登录中心生成 Ticket，并跳回子系统 `/sso/callback`。
4. 子系统调用 `/sso/token` 换取 `loginId`。
5. 子系统创建本地会话，再访问 `/protected` 应返回登录主体。

## 统一登出验证

登录成功后访问：

```text
http://localhost:9100/sso/logout?loginId=user-1001
```

预期结果：

- SSO Server 清除中心 Cookie。
- SSO Server 向已登记的 Client 回调 `/sso/logout-callback`。
- Client 根据 `loginId` 删除本地会话。
- 再次访问 `http://localhost:9101/protected` 会重新跳转到登录中心。

如果开启签名，把 Server 和 Client 的 `SecretKey` 设置一致，并将 `CheckSign` 设为 `true`。Client 侧 `LogoutCallbackHandler` 会自动校验回调签名。

## Redis 模式验证

生产部署建议使用 Redis 存储：

```go
import ssoredis "github.com/Zany2/dtoken-go/sso/storage/redis"

server, err := ssoredis.NewServer(
	"redis://:password@127.0.0.1:6379/0",
	sso.WithKeyPrefix("dtoken:"),
	sso.WithAuthType("sso:"),
	sso.WithConfig(sso.DefaultConfig()),
)
if err != nil {
	return err
}
```

Redis 下建议重点观察以下 key 类型：

- `sso:client:`：子系统注册信息。
- `sso:ticket:`：一次性 Ticket，消费后应删除。
- `sso:oauth2:code:`：OAuth2 Code，消费后应删除。
- `sso:client-session:`：统一登出使用的子系统会话记录，中心登出成功后应清理。

验证建议：

1. 登录前确认 Redis 中已存在客户端注册 key。
2. 发起登录后，观察 Ticket key 短暂出现。
3. 子系统换票成功后，Ticket key 应被删除。
4. 登录中心执行 `/sso/logout` 后，`sso:client-session:` 对应 key 应被删除。

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
