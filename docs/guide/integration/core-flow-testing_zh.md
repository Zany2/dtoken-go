# 核心流程测试指南

[English](../integration/core-flow-testing.md) | 中文文档

## 概览

`tests/gin_core_flow` 是一套面向框架核心能力的真实 HTTP 流程测试。它使用 Gin 编写模拟业务接口，再通过 `httptest.NewServer` 在测试进程内启动临时服务。

这套测试不会启动：

```text
tests/gin_core_app/cmd/server/main.go
```

它会直接调用：

```go
gincoreapp.NewApp(cfg)
httptest.NewServer(app.Router())
```

## 运行方式

在项目根目录执行：

```powershell
go test ./tests/gin_core_flow -v
```

如需启用 Redis 流程测试：

```powershell
$env:DTOKEN_REDIS_URL='redis://:password@localhost:6379/0'
go test ./tests/gin_core_flow -v
```

如果 Windows 环境里曾经设置过非 Windows 目标，可以先重置：

```powershell
$env:GOOS='windows'
$env:GOARCH='amd64'
go clean -cache -testcache
go test ./tests/gin_core_flow -v
```

## 存储配置

`gin_core_flow` 仅在显式设置 `DTOKEN_REDIS_URL` 时使用 Redis：

```text
redis://localhost:6379/0
```

未设置该环境变量时，流程测试使用内存存储实际运行。显式配置 Redis 后，初始化或应用创建失败会导致测试失败，不再跳过。

每个测试 app 会生成随机独立前缀，不同测试进程之间也保持隔离：

```text
dt-gcf-<32 位十六进制字符串>:
```

测试结束时只清理当前前缀下的 key：

```text
dt-gcf-<当前实例的 32 位十六进制字符串>:*
```

不会清空整个 Redis DB。

## 覆盖范围

这套流程测试覆盖：

- 登录、登出、Token 状态和元信息
- 权限、角色、通配、AND/OR、AccessProvider
- 手动续期、自动续期、过期、活跃超时
- Session、终端、多端、终端查询、搜索
- logout、kickout、replace
- 并发登录、Token 复用、最大登录数、账号级和设备级作用域
- 账号、服务、设备、具体设备封禁和解封
- Nonce 生成、校验、消费、过期
- OAuth2 授权码、密码、客户端凭证、刷新、撤销、客户端管理
- 多认证体系隔离
- 核心事件触发
- 所有内置 TokenStyle

## 常见问题

### Redis 里为什么还有 key？

当前测试只删除自身随机前缀下的 key。其它前缀不会删除，例如：

```text
dtoken:gin-core-flow:oauth2:client:demo-client
```

这类 key 通常来自手动启动示例服务，或旧版本测试遗留。

### 自动续期 TTL 为什么会浮动？

Manager 对外返回整数秒 TTL，因此读数可能随秒级边界变化。续期测试通过 Manager 直接读取 TTL，避免测量时触发鉴权和续期；使用有超时限制的轮询观察实际 TTL 增长，并在相应用例中检查续期事件，不再依赖固定的任务等待时间。

### 什么时候需要启动 gin_core_app？

只有手动调接口、用 Postman 或浏览器联调时才需要启动：

```powershell
go run ./tests/gin_core_app/cmd/server
```

自动化测试不需要单独启动服务。

## 相关文档

- [Redis 存储指南](../integration/redis-storage_zh.md)
- [登录认证指南](../core/authentication_zh.md)
- [并发登录策略](../core/concurrency-login_zh.md)
- [Session 与终端管理](../core/session-terminal_zh.md)
