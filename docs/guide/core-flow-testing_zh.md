# 核心流程测试指南

[English](core-flow-testing.md) | 中文文档

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

如果 Windows 环境里曾经设置过非 Windows 目标，可以先重置：

```powershell
$env:GOOS='windows'
$env:GOARCH='amd64'
go clean -cache -testcache
go test ./tests/gin_core_flow -v
```

## 存储配置

当前 `gin_core_flow` 默认使用 Redis：

```text
redis://:root@192.168.19.104:6379/0
```

每个测试 app 会生成独立短前缀：

```text
dt:gcf:1:
dt:gcf:2:
```

测试结束时只清理当前前缀下的 key：

```text
dt:gcf:1:*
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

当前测试只删除 `dt:gcf:*` 本次测试前缀下的 key。其它前缀不会删除，例如：

```text
dtoken:gin-core-flow:oauth2:client:demo-client
```

这类 key 通常来自手动启动示例服务，或旧版本测试遗留。

### 自动续期 TTL 为什么会浮动？

Redis TTL 是秒级返回，和内存存储的时间精度不完全一致。测试中只验证 TTL 落在合理区间，不依赖精确毫秒。

### 什么时候需要启动 gin_core_app？

只有手动调接口、用 Postman 或浏览器联调时才需要启动：

```powershell
go run ./tests/gin_core_app/cmd/server
```

自动化测试不需要单独启动服务。

## 相关文档

- [Redis 存储指南](redis-storage_zh.md)
- [登录认证指南](authentication_zh.md)
- [并发登录策略](concurrency-login_zh.md)
- [Session 与终端管理](session-terminal_zh.md)
