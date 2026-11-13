# Contributing / 贡献指南

English and Chinese issues and pull requests are welcome. 欢迎使用中文或英文提交 Issue 和 PR。

## Discussing changes / 讨论变更

Use the bug report form for reproducible defects and the feature request form for proposed improvements. For larger changes or public API additions, describe the use case and scope in an issue first. Follow the security policy when reporting vulnerabilities.

可复现缺陷请使用问题反馈表单，功能改进请使用功能建议表单。较大变更或公开 API 扩展，请先在 Issue 中说明场景与范围。安全漏洞请按安全报告说明反馈。

- [Issue forms / 反馈入口](https://github.com/Zany2/dtoken-go/issues/new/choose)
- [Security policy / 安全报告](SECURITY.md)

## Repository layout / 仓库结构

The root module contains core, facade, default wiring, basic components, and shared auth checks. `go.work` also includes independent extensions, examples, and test modules. 根模块包含核心、门面、默认装配、基础组件与共享鉴权，`go.work` 同时管理独立扩展、示例与测试模块。

| Directory / 目录 | Responsibility / 职责 |
| --- | --- |
| Root package / 根包 | Instance facade, builder, extension types / 实例门面、构建器和扩展类型 |
| global | Global facade and registry / 全局门面与注册表 |
| internal/engine, internal/builder, internal/defaults | Authentication and default wiring / 认证实现与默认装配 |
| contract, config, errors, event | Public contracts, configuration, events / 公共契约、配置与事件 |
| nonce, oauth2, ticket, shortkey | Explicit capability services / 显式组合的能力服务 |
| httpauth, httpauth/context | Shared checks and service bindings / 共享鉴权与服务绑定 |
| codec, generator, logging, pool, storage | Pluggable components / 可插拔组件 |
| `integrations` | Framework-specific middleware and adapters / Web 框架中间件与适配器 |
| `sso` | SSO server, client, protocols, and storage / 单点登录服务端、客户端、协议与存储 |
| `examples` | Runnable integration examples / 可运行的集成示例 |
| `tests` | Application fixtures and HTTP flow tests / 测试应用与 HTTP 流程测试 |
| `docs` | English and Chinese documentation / 中英文文档 |

## Development environment / 开发环境

Use the Go version declared in `go.work` or a compatible newer version. The workspace currently requires Go 1.27.0, while most individual modules declare Go 1.25.0. Run module commands from the corresponding module directory so workspace dependencies resolve to this checkout.

使用 `go.work` 声明的 Go 版本或兼容的更高版本。当前工作区要求 Go 1.27.0，大多数独立模块声明 Go 1.25.0。在对应模块目录执行 Go 命令，工作区会将内部模块依赖解析到当前源码。

The `codec/jsonv2` module requires `GOEXPERIMENT=jsonv2`. Enable it for that module's commands. 运行 `codec/jsonv2` 模块相关命令时，需要启用 `GOEXPERIMENT=jsonv2`。

## Change guidelines / 变更规范

- Keep changes focused and respect the module boundaries above. 保持改动聚焦，遵循模块职责边界。
- Reuse existing helpers and preserve public API compatibility unless a change explicitly requires otherwise. 复用现有逻辑，按需求明确处理公开 API 兼容性。
- Keep instance APIs in the root and registry APIs in global; compose capabilities outside the engine. 实例 API 放在根包，注册表 API 放在 global，能力在引擎之外组合。
- Use root `dtoken.NewBuilder().Build()` and explicit `WithAuth` injection in new application examples. 新应用示例使用根包构建器和显式实例注入。
- Add optional components in their own packages. Frameworks expose HTTP integration APIs; shared export updates use the generator in `integrations/gin`. 可选组件在各自包中扩展，框架只提供 HTTP 集成 API，共用导出更新通过 `integrations/gin` 中的生成器同步。
- Add concise bilingual Go comments using `// English 中文` for exported APIs and non-obvious logic. 公开 API 和关键逻辑使用简洁的双语注释。
- Run `gofmt` on changed Go files. 对修改的 Go 文件使用 `gofmt` 格式化。
- Add focused regression coverage for bug fixes and update affected English and Chinese documentation. 缺陷修复补充针对性回归覆盖，同步受影响的中英文文档。
- Explain new dependencies and compatibility or storage-format changes in the PR. 在 PR 中说明新增依赖、兼容性和存储格式变化。

## Verification / 验证

For changes to core, facade, defaults, basic components, or shared auth checks, run from the repository root. For independent extensions, run from the module directory. 核心、门面、默认装配、基础组件或共享鉴权的变更请从仓库根目录验证，独立扩展从对应模块目录验证。

```bash
go vet ./...
go test -count=1 -timeout=5m ./...
```

For JSON v2, run from `codec/jsonv2`. JSON v2 模块使用以下命令。

```bash
GOEXPERIMENT=jsonv2 go vet ./...
GOEXPERIMENT=jsonv2 go test -count=1 -timeout=5m ./...
```

The commands above use Bash syntax. In PowerShell, set `$env:GOEXPERIMENT='jsonv2'` before the JSON v2 commands and remove it afterward with `Remove-Item Env:GOEXPERIMENT`.

以上环境变量命令使用 Bash 语法。PowerShell 请先设置 `$env:GOEXPERIMENT='jsonv2'`，执行完成后使用 `Remove-Item Env:GOEXPERIMENT` 清除该设置。

Redis integration tests use `DTOKEN_REDIS_URL`. Use a dedicated test database because these tests create and delete data. Manager Redis scenarios additionally use `DTOKEN_MANAGER_TEST_STORAGE=redis`. See the flow-test guide for setup and isolation details.

Redis 集成测试通过 `DTOKEN_REDIS_URL` 指定地址，测试会创建和删除数据，请使用专用测试库。Manager 的 Redis 场景还需设置 `DTOKEN_MANAGER_TEST_STORAGE=redis`。环境与隔离说明见流程测试指南。

- [Flow-test guide / 流程测试指南](../tests/gin_core_flow/README.md)
- [CI workflow / CI 工作流](workflows/ci.yml)

CI enumerates workspace modules, runs vet and tests, then reruns Redis-backed scenarios with a temporary Redis service. Report the commands actually run and any skipped checks in the PR.

CI 会遍历工作区模块执行检查和测试，再使用临时 Redis 服务运行相关场景。PR 请如实填写执行的命令、结果与跳过的检查。

The regular pool suite checks a bounded scale-up and scale-down cycle. The 30-minute observation test uses the `stress` build tag. 常规协程池测试验证有界的扩容与缩容周期，30 分钟观察用例通过 `stress` 构建标签显式启用。

```bash
go test -tags=stress -run '^TestRenewPoolManagerAutoScale$' -count=1 -timeout=35m ./pool/ants
```

Module ownership, builder selection, and release migration are described in the [module migration guide](../docs/design/module-migration.md). 模块归属、构建入口与发布迁移见[模块迁移指南](../docs/design/module-migration_zh.md)。

Independent module checks copy each module outside the checkout, exclude nested modules, and disable workspace resolution. Local replacements still select the checkout dependency versions. Cross-framework generated exports are checked separately at repository level. 独立模块检查会将各模块复制到仓库外，排除嵌套模块并关闭工作区解析；依赖版本仍通过本地替换指向当前源码。跨框架生成文件在仓库级单独检查。

```bash
go run ./.github/scripts/check-modules
go run ./.github/scripts/check-modules ./integrations/gin
go run ./.github/scripts/check-docs
```

The documentation check compiles marked Go blocks directly from both READMEs, quick-start, authentication, permission and integration guides; repeated markers in one document form one program. It does not execute the examples or start servers. 文档检查直接编译中英文 README、快速开始、认证、权限和集成指南中带标记的 Go 代码块；同一文档中的相同标记会合并为一个程序，不运行示例或启动服务器。

From `integrations/gin`, run `go run ./internal/integrationexportgen -check` to check all framework exports without rewriting them. 在 `integrations/gin` 中运行该命令可只检查所有框架导出，不改写文件。
