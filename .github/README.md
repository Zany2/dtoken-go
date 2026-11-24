# GitHub repository configuration / GitHub 仓库配置

| File / 文件 | Purpose / 用途 |
| --- | --- |
| `ISSUE_TEMPLATE/bug_report.yml` | Reproducible bug reports / 可复现问题反馈 |
| `ISSUE_TEMPLATE/feature_request.yml` | Feature proposals / 功能建议 |
| `ISSUE_TEMPLATE/config.yml` | Template chooser and documentation/security links / 模板选择与文档、安全入口 |
| `PULL_REQUEST_TEMPLATE.md` | Change, compatibility, and verification details / 变更、兼容性与验证说明 |
| `CONTRIBUTING.md` | Module boundaries and contribution guidance / 模块边界与贡献指南 |
| `SECURITY.md` | Private vulnerability reporting guidance / 私密漏洞报告说明 |
| `workflows/ci.yml` | Workspace vet/tests and Redis scenarios / 工作区检查、测试与 Redis 场景 |
| `dependabot.yml` | Weekly Go dependency and GitHub Actions updates / 每周检查 Go 依赖与 GitHub Actions 更新 |
| `scripts/check-consumer.sh` | Local-source or published-version facade, Gin and SSO consumer checks / 当前源码或发布版本的门面、Gin 及 SSO 使用方检查 |

## Activation / 生效方式

Merge the templates and Dependabot configuration into the repository's default branch. CI runs on pushes to `main`, `develop`, and `architecture-refactor`, on pull requests, and through manual dispatch. It reads the Go version and module list from `go.work`, enabling `jsonv2` for its codec module.

将模板与 Dependabot 配置合并到仓库默认分支后生效。CI 在推送到 `main`、`develop`、`architecture-refactor`，提交 PR 或手动触发时执行。工作流从 `go.work` 读取 Go 版本和模块列表，为 JSON v2 编解码模块启用实验特性。

Dependabot checks all module directory groups weekly, limits Go update PRs to five and Actions update PRs to three, and groups minor/patch Go updates. Repository-internal module versions remain coordinated by releases. Review update PRs using the same compatibility checks as other changes.

Dependabot 每周检查各组模块目录，Go 依赖更新最多保留五个 PR，Actions 更新最多保留三个 PR，并合并 Go 次版本和补丁版本更新。仓库内部模块版本由发布流程统一维护。依赖更新 PR 按通常的兼容性要求审核。

Repository administrators can enable private vulnerability reporting in the repository settings. Once CI has passed on GitHub, administrators can select `Main module minimum Go`, `Workspace checks`, and `Module checks without workspace` as required status checks in a branch ruleset.

仓库管理员可在设置中启用私密漏洞报告。CI 在 GitHub 成功运行后，可在分支规则中将 `Main module minimum Go`、`Workspace checks` 和 `Module checks without workspace` 设为合并前必须通过的检查。

## Official references / 官方参考

- [Go starter workflow / Go 工作流模板](https://github.com/actions/starter-workflows/blob/main/ci/go.yml)
- [Issue form syntax / Issue 表单语法](https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/syntax-for-issue-forms)
- [Pull request templates / PR 模板](https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/creating-a-pull-request-template-for-your-repository)
- [Dependabot options / Dependabot 配置](https://docs.github.com/en/code-security/reference/supply-chain-security/dependabot-options-reference)
- [Enable private vulnerability reporting / 启用私密漏洞报告](https://docs.github.com/en/code-security/how-tos/report-and-fix-vulnerabilities/configure-vulnerability-reporting/configure-for-a-repository)

CI also runs `.github/scripts/check-modules.sh` with workspace resolution disabled. Temporary module files map local dependencies to the current checkout. This checks the standalone package graph against local sources. Published-version resolution requires the release verification described in the [migration guide](../docs/design/module-migration.md).

CI 还会通过 `.github/scripts/check-modules.sh` 关闭工作区解析，使用临时模块文件将依赖映射到当前源码，检查独立模块的包依赖关系。发布版本的解析需要按[迁移指南](../docs/design/module-migration_zh.md)完成发布验证。

The minimum-Go job reads go.mod and checks the main module with GOWORK=off, including race detection. The external consumer checks root instance APIs, public extension types, explicit global registration, middleware capability bindings, and logout.

最低 Go 版本任务从 go.mod 读取版本，在 GOWORK=off 下检查主模块并启用竞态检测。仓库外消费者检查覆盖根实例 API、公共扩展类型、自动注册、根包体系选择与旧 global 兼容、中间件能力绑定和退出。

Manual CI accepts `root_version` and `gin_version` together to check published modules. Supplying `sso_version` as well includes the SSO consumer; omitting it preserves the two-module check. Every requested version must be nonempty and remain exactly selected after dependency resolution. The regular consumer step uses the current checkout and includes SSO. See the [SSO upgrade guide](../sso/MIGRATION.md) for scope and commands.

手动 CI 可同时填写 `root_version` 与 `gin_version` 检查发布模块；再填写 `sso_version` 时加入 SSO 使用方，省略则保持原来的双模块检查。请求的版本均须非空，依赖解析后仍须精确选中。常规使用方步骤使用当前源码并包含 SSO。范围和命令见 [SSO 升级指南](../sso/MIGRATION_zh.md)。

The workspace job also race-checks SSO, its Redis adapter and all five SSO examples. Redis integration runs separately with the service URL configured and now enables race detection for the SSO Redis adapter, including replica lifecycle scenarios. The general race step still runs without live Redis. Documentation compilation now includes both SSO configuration guides. None of these additions establishes a passing result until CI or the documented commands actually run.

工作区任务还对 SSO、其 Redis 适配器及全部五个 SSO 示例执行竞态检查。真实 Redis 集成在单独配置服务地址的步骤执行，并为 SSO Redis 适配器启用竞态检测，覆盖多副本生命周期；通用竞态步骤仍不连接真实 Redis。文档编译已纳入中英文 SSO 配置指南。这些入口需要实际运行后才能记录通过结果。

## SSO release verification / SSO 发布验证

`ci.yml` adds `SSO minimum Go (sso)` and `SSO minimum Go (sso/storage/redis)`, reading each module's go.mod with workspace and automatic toolchain upgrades disabled. The `check-modules -sso` discovery path reads only the four release manifests, avoiding the newer go.work requirement. The SSO matrix entry also runs the dedicated source consumer.

`ci.yml` 新增两个 SSO 最低 Go 检查，分别读取 SSO 和 SSO Redis 的 go.mod，关闭工作区及自动工具链升级。`check-modules -sso` 仅读取四个发布模块的清单，避免加载版本更高的 go.work；SSO 矩阵项还运行专用源码消费者。

[scripts/check-sso-consumer.sh](scripts/check-sso-consumer.sh) accepts zero arguments for local sources, or four exact versions in root / SSO / storage Redis / SSO Redis order. It verifies selected versions without replacements in published mode and runs public consumer fixtures with vet and race detection. It does not connect to Redis or start a listener; dependencies may be downloaded.

专用消费者脚本接受零参数（当前源码）或按根模块／SSO／通用 Redis／SSO Redis 顺序传入四个精确版本。发布模式不加源码替换并检查实际选中版本，随后运行公开使用方的 vet 和竞态回归。它不连接 Redis、不监听端口，但可能下载依赖。

[workflows/sso-release.yml](workflows/sso-release.yml) provides a manual-only published check with four required versions, independent of Gin. It never publishes or tags. Fixtures come from the selected workflow ref and require its current public API; use compatible release versions. Existing general consumer invocations remain unchanged. All new execution results are pending; see [SSO release acceptance](../sso/RELEASE_CHECKLIST.md).

独立手动工作流要求填写四个版本，不依赖 Gin，不创建标签或发布。验收源码来自所选工作流 ref，并要求对应的当前公开 API，应选择兼容版本。原通用消费者调用方式保持不变。新增运行结果均待确认，见 [SSO 发布验收](../sso/RELEASE_CHECKLIST_zh.md)。
