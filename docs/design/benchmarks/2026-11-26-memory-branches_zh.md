# Memory 关键分支与遗留方法收尾

2026-11-26，在[方法逻辑验收](2026-11-26-memory-methods_zh.md)之后，继续核对剩余语句覆盖。本轮删除 4 个确认无调用的私有辅助方法，新增 4 个顶层测试、16 个具体场景；未改变公开 API、存储格式或业务行为，未接触 SSO，未继续连接 Redis。

## 检查与改动

此前未覆盖的关键语句主要为刷新过程的读取／解码错误、清理过程的存储错误及关闭异常处理。选取会影响凭据消费、会话索引保留和资源释放的分支补充行为断言，没有逐个制造所有底层故障，也没有为提高覆盖数字添加只调用方法的测试。

新增测试位于 `internal/engine/manager_memory_failure_paths_test.go`，数据使用内置 Memory，故障包装器只拦截指定读取或关闭操作。

| 场景 | 数量 | 断言 |
| --- | ---: | --- |
| 刷新消费前读取失败 | 6 | 原子／普通存储分别在访问主记录、活跃时间、超时会话读取时失败；返回存储错误，无成功结果或事件，原生命周期数据和 TTL 不被延长；恢复后正常刷新可重试，原本不活跃超时的凭据仍按超时处理 |
| 刷新记录异常 | 6 | 不支持的值类型、无效 JSON、跨认证体系记录分别执行刷新／撤销；返回约定错误或幂等结果，不改动访问凭据与会话；正常访问校验仍有效 |
| 登录触发终端清理失败 | 3 | 主记录、活跃时间、刷新归属读取失败时新登录失败，保存的 Session 与凭据不被误删；恢复后重试成功并满足终端数限制 |
| 资源关闭失败 | 1 | 自有存储 Close 返回错误仍释放池、刷新并关闭日志器；诊断可见，重复 Close 不重复释放，拒绝后续登录和异步任务 |

这些用例验证当前顺序处理语义，不承诺多键事务或存储故障后的自动回滚。没有发现需要修改生产业务逻辑的新缺陷。

全仓 Go 源码引用检索确认：`getCheckedTokenSession` 仅调用 `checkLoginAndGetContext`，两者没有其他入口；`ensureTerminalTokenAlive`、`isTerminalTokenAlive` 也没有调用方。已删除这 4 个私有方法，保留实际使用的校验和终端存活实现。删除后引用检索无残留，根模块编译和测试通过。

## 验证结果

- 16 个新增场景通过。
- `go vet ./... ./tests/gin_core_flow/...` 通过。
- 根模块全部测试及 Gin 流程回归以 `-count=1` 重新运行，通过。其他非 SSO 独立模块沿用上一轮 24 模块检查结果，本轮没有重新执行它们。
- `gofmt` 与本轮相关文件的 diff 空白检查通过。
- race 仍缺少所需 CGO/C 编译器；没有安装工具链或将普通测试当作 race 通过。Redis 和部署演练仍未完成。

最新根模块语句覆盖为 **8,673 / 9,714 = 89.3%**，引擎为 **4,107 / 4,544 = 90.4%**。相比前次，执行过的语句增加 14 条，同时删除 12 条没有调用的语句；覆盖率提升包含这两部分，不能解读为性能提高或全部分支正确。

根模块 1,261 个函数清单项中，1,251 个有可执行语句的函数／方法均有命中，另有 10 个 NopLogger 空方法不适用语句覆盖。公开函数／方法仍为 921 个，其中含语句的 911 个均命中。依然存在未执行分支，覆盖统计不证明所有参数组合、时序或错误场景。

## 原始记录

- [完整回归输出](2026-11-26-memory-branches-final.txt)
- [逐函数覆盖输出](2026-11-26-memory-branches-functions.txt)
- [最新方法清单](2026-11-26-memory-branches-methods.csv)
- [最新包语句统计](2026-11-26-memory-branches-packages.csv)

统计沿用前次口径：合并根模块与 Gin 流程各进程的覆盖块，根模块汇总排除独立 Redis 和测试应用模块；包含 `contract/storagetest`。原始工具总计 86.0% 包含上述独立模块，不与根模块的 89.3% 混用。

复现命令沿用前次 Memory 环境设置，将 profile 文件名改为 `dtoken-memory-branches-final.cover`：

```powershell
go vet ./... ./tests/gin_core_flow/...
$taskCoverage = Join-Path $env:TEMP 'dtoken-memory-branches-final.cover'
go test ./... ./tests/gin_core_flow/... -count=1 '-coverpkg=./...' -covermode=atomic "-coverprofile=$taskCoverage" -timeout=5m
go tool cover "-func=$taskCoverage"
```

本轮代码收尾完成。剩余工作以相应环境的 race、Redis 生命周期和实际部署验收为主；当前没有新增缺陷要求继续扩展架构或追求 100% 语句覆盖。
