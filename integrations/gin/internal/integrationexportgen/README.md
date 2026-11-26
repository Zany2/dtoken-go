# Integration source generation / 框架源码生成

Run from `integrations/gin`. 从 `integrations/gin` 目录执行。

```sh
go generate
go run ./internal/integrationexportgen -check
```

The generator synchronizes common exports and grouped request forwarding to Beego, Chi, Echo, Fiber, GoFrame, Hertz and Kratos. It adds no runtime dependency between integrations. 生成器向七个框架同步公共导出和分组请求转发，不增加框架之间的运行时依赖。

## Canonical source / 标准源

Maintain shared forwarding in Gin's `context*facade.go` files. Public functions ending in `ByContext` that call a `dCtx` operation group are selected. Generated functions retain argument and return types, variadic forwarding and error handling; native context types and the `ByCtx` suffix are mapped per framework. 在 Gin 的 `context*facade.go` 中维护共享转发。生成器选取以 `ByContext` 结尾、调用 `dCtx` 操作分组的公开函数，保留参数、返回值、可变参数转发和错误处理，按框架映射原生上下文类型及 `ByCtx` 后缀。

`GetLoginID`, `GetTokenInfo` and `IntrospectToken` are excluded because GoFrame and Kratos support additional namespace arguments. Raw context access, manager lookup, native context resolution and middleware remain handwritten. `GetLoginID`、`GetTokenInfo` 和 `IntrospectToken` 因 GoFrame、Kratos 支持额外命名空间参数而保留原实现；原始上下文访问、管理器查询、原生上下文解析和中间件也继续手动维护。

## First generation / 首次生成

Generation writes `context_forward_generated.go` in each target package and removes the corresponding old function declarations and unused imports. Files containing only migrated declarations are removed; files with native helpers or other declarations are preserved. Review these changes before committing. 生成器在目标包写入 `context_forward_generated.go`，移除对应旧函数声明及不再使用的导入。仅包含已迁移声明的文件会删除，含有原生辅助逻辑或其他声明的文件会保留。提交前应审查这些变更。

The first generation is required before the expanded `-check` can pass. `-check` never migrates or writes files. After migration, run generation again whenever canonical forwarding changes. 扩展后的 `-check` 通过前须先完成首次生成；`-check` 不迁移、不写文件。迁移后，每次修改标准转发都应重新生成。

Generator tests cover native context mapping, variadic calls, preservation of native functions, migration idempotence and read-only checks. The root `integration_facade_architecture_test.go` independently compares grouped call destinations across all adapters. 生成器用例覆盖上下文映射、可变参数、原生函数保留、迁移幂等性和只读检查；根目录的 `integration_facade_architecture_test.go` 独立比较所有适配器的分组调用目标。
