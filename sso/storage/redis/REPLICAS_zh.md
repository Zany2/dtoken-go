# Redis 多副本生命周期验收

[English](REPLICAS.md) · [适配器说明](README_zh.md) · [发布验收](../../RELEASE_CHECKLIST_zh.md)

这组集成用例使用真实 Redis、独立连接池及多个独立 Server／ClientApp／BrowserClient 实例。HTTP 请求通过实际处理器在进程内传输，不监听端口。它验证共享存储语义，不声称启动了多个操作系统进程、真实浏览器或 Redis 集群。

## 验收矩阵

| 用例 | 验收行为 |
| --- | --- |
| `TestRedisReplicasAtomicCredentials` | Ticket 和 Code 绑定 PKCE；错误证明不消费凭证；八个并发请求通过两个连接兑换，恰好一个成功 |
| `TestRedisReplicasReplayAndRotation` | 同一签名授权请求落到两个中心，恰好一次签发；关闭原连接并重新构造中心后 nonce 仍存在；换密钥不能绕过重放，退役签名不占用新 nonce |
| `TestRedisReplicasBrowserCallback` | 四种模式均在一个副本保存浏览器 state，然后关闭其连接；旧对象故障时拒绝返回身份；新副本从 Redis 恢复流程，八个并发回调恰好成功一次。Ticket／Code 包含 PKCE |
| `TestRedisReplicasAccountLogoutRecovery` | 两个应用、两个账号、四种凭证；通知 Beta 暂时返回 503，中心及旧凭证先撤销，回调绑定保留；新中心连接重试完成通知和清理，Bob 不受影响，Alice 可重新签发 |

账号下线用例中的中心／应用会话是 Redis 中的应用自有标记，由实际 `CenterLogout` 和注销回调删除，用于验证组合顺序。完整 Auth 令牌和浏览器会话语义仍由[部署示例](../../../examples/sso_deployment/README_zh.md)的回归负责。故障通过关闭客户端连接和返回 503 模拟，不会主动停止 Redis，也不覆盖网络分区、持久化丢失或故障转移。

## 执行方式

由使用者准备支持 GETDEL 的 Redis 6.2+，在仓库根目录执行：

```powershell
$env:DTOKEN_REDIS_URL = 'redis://127.0.0.1:6379/15'
go test ./sso/storage/redis/... -run TestRedisReplicas -count=1 -v -timeout=5m
go test -race ./sso/storage/redis/... -run TestRedisReplicas -count=1 -v -timeout=5m
```

地址应指向测试实例。未设置变量时集成用例明确跳过；已设置但无法连接时失败，不能视为通过。测试连接错误不打印原始 Redis URL，避免把其中的凭据写入日志。Go 版本按工作区要求准备，竞态检查还需要平台支持的 race 工具链。

现有 CI 的 Redis 服务步骤已覆盖这个独立模块，并对该模块启用 `-race`。无 Redis 环境的普通测试步骤依旧跳过这些用例；只有配置了地址的 Redis 步骤才验证真实后端。本轮未运行测试、构建或 CI，也未连接 Redis。

## 数据与清理

每次用例使用随机 `dtoken:test:replicas:<random>:` 前缀，协议、浏览器状态、nonce 和会话标记分别使用其子前缀。并发请求全部结束后，先关闭副本，再通过独立管理连接 SCAN 该前缀，逐项检查归属并删除；不会执行 FLUSHDB、FLUSHALL 或扫描全部键后无差别删除。

测试使用与实际生命周期一致的 TTL：浏览器状态、nonce、凭证会过期，注册及账号撤销版本可能持久保存。异常中断或 Redis 不可用可能使清理失败并留下本次前缀，因此仍应使用专用测试实例。不得把测试清理方法当成生产注销版本的清理策略。

## 仍需真实部署验证

实际多进程滚动更新、负载均衡、HTTPS／SameSite Cookie、浏览器标签页、Redis 持久化与故障转移、持续负载和网络超时需单独验收。记录 Go／Redis 版本、提交及未提交改动、测试命令和结果，不能以新增用例源码代替通过记录。
