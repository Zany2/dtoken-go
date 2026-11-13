# AGENTS.md

Project instructions for AI coding agents working in this repository.

These rules apply to the whole repo unless a more specific instruction file is added in a subdirectory.

## Core Principles

### Think Before Coding

- State assumptions before making non-trivial changes.
- If a request has multiple reasonable interpretations, ask or clearly name the chosen interpretation.
- Prefer the simplest implementation that satisfies the request.
- Push back when a requested change would add unnecessary complexity or conflict with existing project design.

### Simplicity First

- Do not add features, abstractions, configuration, or error handling that was not requested.
- Avoid single-use abstractions.
- Keep diffs small and easy to review.
- If a solution becomes much larger than the problem suggests, simplify it before finishing.

### Surgical Changes

- Touch only files needed for the task.
- Do not reformat, rename, refactor, or clean up adjacent code unless required by the task.
- Preserve existing public APIs unless the request explicitly changes them.
- Remove only imports, variables, functions, or comments made obsolete by your own changes.
- Mention unrelated dead code or suspicious behavior in the final note; do not remove it without approval.

### Goal-Driven Execution

For multi-step work, define the success criteria before editing:

```text
1. [Step] -> verify: [check]
2. [Step] -> verify: [check]
3. [Step] -> verify: [check]
```

Examples:

- Add validation -> add or update focused tests for invalid inputs, then make them pass.
- Fix a bug -> reproduce the bug with a test or clear local check, then fix it.
- Refactor code -> preserve behavior and verify the touched surface.

## Repository Context

- This is a Go workspace managed by `go.work`.
- The root module publishes core, the public facade, default wiring, bundled basic components, and shared auth checks together. Optional components, framework integrations, SSO, examples, and application tests use independent modules.
- Packages are split by responsibility:
  - Root package `dtoken`: instance facade APIs, the high-level builder, and public engine extension types.
  - `global`: global facade APIs and the shared manager registry.
  - `internal/engine`, `internal/builder`, `internal/defaults`: auth implementation and default component wiring.
  - `contract`, `config`, `errors`, `event`: public extension contracts and shared domain types. Capability service interfaces and data types live in contract/nonce, contract/oauth2, contract/ticket, and contract/shortkey.
  - `nonce`, `oauth2`, `ticket`, `shortkey`: independently composed capabilities and event-aware services.
  - `httpauth`, `httpauth/context`: shared HTTP checks and request-scoped service bindings.
  - `codec`, `generator`, `logging`, `pool`, `storage`: pluggable component implementations.
  - `integrations/<framework>`: framework-specific middleware, request adapters, and request helpers.
  - `sso`: independent SSO server, client, protocols, and storage.
  - `examples`, `tests`, `docs`: runnable applications, application checks, and documentation.
- Keep capability implementations out of the internal auth engine and HTTP packages. HTTP adapters depend on capability contracts; bundled services implement those contracts. Compose services explicitly and bind HTTP capabilities through `WithCapabilities`.
- Use the root `NewBuilder().Build()` in applications and inject instances with `WithAuth` or managers with `WithManager`.
- Manager is a supported advanced API, obtained from a built Auth; avoid runtime mutation of its configuration. Use global.SetAuth for application registration and document shared engine ownership.
- Root instance APIs do not require global registration. Global calls use the `global` package explicitly. HTTP registry lookup requires WithManagerResolver(global.GetManager); otherwise inject an instance with WithAuth.
- Add optional components in their component packages. Frameworks do not re-export component constructors or global functions. Maintain shared HTTP exports through the Gin integration export generator.

## Go Coding Rules

- Match existing package style and naming.
- Prefer existing helpers and internal APIs over new abstractions.
- Keep exported API additions intentional and documented.
- Use `gofmt` for edited Go files when Go code changes are made.
- Do not introduce new dependencies unless the user asks or the need is unavoidable.
- Avoid changing generated files such as `*.pb.go` unless the task is specifically about generated output.

## Storage Consistency Policy

- This project is an open framework with pluggable storage, not a database-backed transaction coordinator.
- Multi-step storage operations do not have to be fully CAS-based or transactionally atomic. Based on the current framework design, prefer atomic storage capabilities such as `AtomicStorage` when they are available and naturally fit the operation.
- Multi-step storage operations do not need rollback logic by default. The framework assumes the configured storage backend remains continuously available during normal operation.
- If an operation spans multiple storage keys, keep the normal sequential lifecycle correct first, then apply best-effort atomic hardening where the storage implementation supports it.
- Do not force custom storage implementations to support CAS, Lua scripts, distributed locks, or database-style transactions unless a public API explicitly declares that requirement.
- For storage implementations without atomic extensions, provide a reasonable fallback with ordinary `Get` / `Set` / `Delete` operations.
- Avoid adding complex compensation logic for rare storage outage windows unless the lack of compensation causes a normal-flow lifecycle bug.
- During code review, classify issues as either required correctness fixes or optional concurrency hardening. Required fixes should address ordinary execution defects; optional hardening should not complicate the framework unnecessarily.

## Comment Style

- Use the unified bilingual comment style already used by the project:

```go
// English 中文
```

- Keep function and method documentation on one bilingual line; preserve Go directives and separate `Deprecated:` paragraphs. 函数与方法说明使用单行中英文注释，保留 Go 指令及独立的 `Deprecated:` 段落。
- Keep type, interface, and declaration-group summaries on one bilingual line above the declaration. 类型、接口及声明分组说明放在声明上方，使用单行中英文注释。
- Preserve meaningful package documentation paragraphs and examples; keep each prose paragraph on one bilingual line. 包文档保留必要的语义分段及示例，每段文字说明使用单行中英文注释。
- Place comments above exported variables, constants, and struct fields. For private declarations, put short comments at the end of the line; keep long comments and comments for multiline declarations above. 公开变量、常量及结构体字段的注释放在声明上方；私有声明的简短注释放行尾，较长说明及多行声明的注释保留在上方。
- Comment on local variables only when intent is not obvious, placing the explanation above the relevant logic. 局部变量仅在意图不直观时添加注释，说明放在相关逻辑上方。
- Add concise comments for exported methods, fields, variables, constants, and key logic steps.
- Comments should explain intent, constraints, or non-obvious behavior.
- Place control-flow and operation comments above the relevant statement; combine repeated descriptions of the same step. 流程及操作注释放在相关语句上方，同一步骤的重复说明合并为一条。
- Separate each key logic step with a blank line after the previous block, so step comments visually divide the function flow.
- Avoid empty narration such as comments that merely repeat the code.
- Preserve existing comments where possible.

## Frontend And Styles

- If frontend code is added later, reuse existing shared/common components first.
- Do not create new components when an existing one satisfies the requirement.
- Before writing styles, check installed styling packages and use the matching syntax.
- If `sass` is installed, write styles using SCSS syntax.

## Verification

- The user will run the project themselves.
- Do not run build, test, server, example, or project execution commands unless the user explicitly asks.
- After edits, static checks such as reviewing diffs, checking syntax mentally, or inspecting affected files are acceptable.
- If Go code is changed and formatting is needed, `gofmt` may be used on the touched files only.
- Clearly report what was and was not verified.

## Collaboration Notes

- Assume the worktree may contain user changes. Do not revert changes you did not make.
- If unrelated modifications are present, ignore them unless they block the task.
- If a requested change conflicts with these instructions, explain the conflict before proceeding.
