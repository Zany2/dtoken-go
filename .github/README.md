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

## Activation / 生效方式

Merge the templates and Dependabot configuration into the repository's default branch. CI runs on pushes to `main`, `develop`, and `architecture-refactor`, on pull requests, and through manual dispatch. It reads the Go version and module list from `go.work`, enabling `jsonv2` for its codec module.

将模板与 Dependabot 配置合并到仓库默认分支后生效。CI 在推送到 `main`、`develop`、`architecture-refactor`，提交 PR 或手动触发时执行。工作流从 `go.work` 读取 Go 版本和模块列表，为 JSON v2 编解码模块启用实验特性。

Dependabot checks all module directory groups weekly, limits Go update PRs to five and Actions update PRs to three, and groups minor/patch Go updates. Repository-internal module versions remain coordinated by releases. Review update PRs using the same compatibility checks as other changes.

Dependabot 每周检查各组模块目录，Go 依赖更新最多保留五个 PR，Actions 更新最多保留三个 PR，并合并 Go 次版本和补丁版本更新。仓库内部模块版本由发布流程统一维护。依赖更新 PR 按通常的兼容性要求审核。

Repository administrators can enable private vulnerability reporting in the repository settings. Once CI has passed on GitHub, administrators can select `Workspace checks` as a required status check in a branch ruleset.

仓库管理员可在设置中启用私密漏洞报告。CI 在 GitHub 成功运行后，可在分支规则中将 `Workspace checks` 设为合并前必须通过的检查。

## Official references / 官方参考

- [Go starter workflow / Go 工作流模板](https://github.com/actions/starter-workflows/blob/main/ci/go.yml)
- [Issue form syntax / Issue 表单语法](https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/syntax-for-issue-forms)
- [Pull request templates / PR 模板](https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/creating-a-pull-request-template-for-your-repository)
- [Dependabot options / Dependabot 配置](https://docs.github.com/en/code-security/reference/supply-chain-security/dependabot-options-reference)
- [Enable private vulnerability reporting / 启用私密漏洞报告](https://docs.github.com/en/code-security/how-tos/report-and-fix-vulnerabilities/configure-vulnerability-reporting/configure-for-a-repository)

CI also runs `.github/scripts/check-modules.sh` with workspace resolution disabled. Temporary module files map local dependencies to the current checkout. This checks the standalone package graph against local sources. Published-version resolution requires the release verification described in the [migration guide](../docs/design/module-migration.md).

CI 还会通过 `.github/scripts/check-modules.sh` 关闭工作区解析，使用临时模块文件将依赖映射到当前源码，检查独立模块的包依赖关系。发布版本的解析需要按[迁移指南](../docs/design/module-migration_zh.md)完成发布验证。
