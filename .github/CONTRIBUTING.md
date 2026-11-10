# Contributing / 贡献指南

English and Chinese issues and pull requests are welcome. 欢迎使用中文或英文提交 Issue 和 PR。

## Discussing changes / 讨论变更

Use the bug report form for reproducible defects and the feature request form for proposed improvements. For larger changes or public API additions, describe the use case and scope in an issue first. Follow the security policy when reporting vulnerabilities.

可复现缺陷请使用问题反馈表单，功能改进请使用功能建议表单。较大变更或公开 API 扩展，请先在 Issue 中说明场景与范围。安全漏洞请按安全报告说明反馈。

- [Issue forms / 反馈入口](https://github.com/Zany2/dtoken-go/issues/new/choose)
- [Security policy / 安全报告](SECURITY.md)

## Repository layout / 仓库结构

This repository contains multiple Go modules managed by `go.work`. 使用 `go.work` 管理多个 Go 模块。

| Directory / 目录 | Responsibility / 职责 |
| --- | --- |
| `core` | Authentication, configuration, managers, context, and protocols / 认证、配置、管理器、上下文与协议 |
| `dtoken` | Public global and instance facade APIs / 公开的全局与实例门面 API |
| `defaults` | Default builder and component wiring / 默认构建器与组件装配 |
| `com` | Pluggable codecs, storage, logging, generators, and pools / 可插拔编解码、存储、日志、生成器和协程池 |
| `integrations` | Framework-specific middleware and adapters / Web 框架中间件与适配器 |
| `sso` | SSO server, client, protocols, and storage / 单点登录服务端、客户端、协议与存储 |
| `examples` | Runnable integration examples / 可运行的集成示例 |
| `tests` | Application fixtures and HTTP flow tests / 测试应用与 HTTP 流程测试 |
| `docs` | English and Chinese documentation / 中英文文档 |

## Development environment / 开发环境

Use the Go version declared in `go.work` or a compatible newer version. The workspace currently requires Go 1.27.0, while most individual modules declare Go 1.25.0. Run module commands from the corresponding module directory so workspace dependencies resolve to this checkout.

使用 `go.work` 声明的 Go 版本或兼容的更高版本。当前工作区要求 Go 1.27.0，大多数独立模块声明 Go 1.25.0。在对应模块目录执行 Go 命令，工作区会将内部模块依赖解析到当前源码。

The `com/codec/jsonv2` module requires `GOEXPERIMENT=jsonv2`. Enable it for that module's commands. 运行 `com/codec/jsonv2` 模块相关命令时，需要启用 `GOEXPERIMENT=jsonv2`。

## Change guidelines / 变更规范

- Keep changes focused and respect the module boundaries above. 保持改动聚焦，遵循模块职责边界。
- Reuse existing helpers and preserve public API compatibility unless a change explicitly requires otherwise. 复用现有逻辑，按需求明确处理公开 API 兼容性。
- Add concise bilingual Go comments using `// English 中文` for exported APIs and non-obvious logic. 公开 API 和关键逻辑使用简洁的双语注释。
- Run `gofmt` on changed Go files. 对修改的 Go 文件使用 `gofmt` 格式化。
- Add focused regression coverage for bug fixes and update affected English and Chinese documentation. 缺陷修复补充针对性回归覆盖，同步受影响的中英文文档。
- Explain new dependencies and compatibility or storage-format changes in the PR. 在 PR 中说明新增依赖、兼容性和存储格式变化。

## Verification / 验证

For a changed module, run the following commands from its directory. 在修改的模块目录执行以下命令。

```bash
go vet ./...
go test -count=1 -timeout=5m ./...
```

For JSON v2, run from `com/codec/jsonv2`. JSON v2 模块使用以下命令。

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
