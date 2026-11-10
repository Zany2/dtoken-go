# Module organization and migration

English | [中文文档](module-migration_zh.md)

## Module ownership

The repository has 30 modules, including examples and test applications. The root module `github.com/Zany2/dtoken-go` contains these package trees, which share a release version.

- `core`
- `dtoken`
- `defaults`
- `com/codec/base64` and `com/codec/json`
- `com/generator/dgenerator`
- `com/log/dlog` and `com/log/nop`
- `com/pool/ants`
- `com/storage/memory`
- `integrations/authcheck`

The independent library modules are `com/codec/jsonv2`, `com/codec/msgpack`, `com/log/gf`, `com/storage/redis`, the eight framework integrations, `sso`, and `sso/storage/redis`. Examples and application-level flow tests retain their own modules to isolate their framework dependencies.

Package import paths are preserved. For example, `github.com/Zany2/dtoken-go/core/manager` and `github.com/Zany2/dtoken-go/dtoken` are now packages of the root module. Their directories have a shared `go.mod` at the repository root.

## Builder entry points

| Entry | Intended use |
| --- | --- |
| `dtoken.NewBuilder()` | Preferred application entry, default components and high-level optional feature configuration |
| `defaults.NewBuilder()` | Default components with the low-level core builder |
| `builder.NewBuilder()` in `core/builder` | Explicit component and factory assembly |
| Framework `NewBuilder()` | Existing framework convenience entry returning the default core builder |

Application code can build a Manager with `dtoken.NewBuilder().Build()`, wrap it with `dtoken.New(mgr)` for instance calls, or register it with `dtoken.SetManager(mgr)` for global calls. Builder return types and existing APIs retain their current contracts.

The quick-start example demonstrates the preferred application builder. Framework integrations continue to delegate shared authentication rules to `integrations/authcheck`. Components implement `core/adapter` contracts, and `defaults` assembles the default implementations.

## Consumer migration

An application upgrading from separately versioned core packages must remove its direct and indirect requirements and local replacements for the eleven merged modules listed above. Add a requirement for the root module at the intended release version. Keep independent extension requirements, and select extension releases that depend on the root module.

Old nested module requirements can overlap the packages in the new root module and cause ambiguous imports. Inspect the application's `go.mod`, including transitive dependencies, before completing the upgrade. Package imports can continue using their existing paths.

After selecting compatible releases, run `go mod tidy` and the application's verification commands. For development against this branch, use the repository workspace. The standalone CI check uses temporary local replacements for all checkout modules.

## Release coordination

The `v0.0.3` root requirements in this branch carry forward the previous source baseline. A release containing the consolidated root module must be published before consumers can resolve these dependencies remotely. The branch configuration alone does not establish remote availability.

For a release, choose an available root version, update extension requirements to that version, and publish the root module with a repository tag such as `vX.Y.Z`. Publish independent modules using their directory-prefixed tags, such as `integrations/gin/vX.Y.Z` and `com/storage/redis/vX.Y.Z`. Publish dependencies before dependent extensions and examples.

Before announcing a release, create a clean consumer module outside this checkout with `GOWORK=off`. Install the intended facade and integration release versions, and verify login, protected access, and logout. This validates remotely published artifacts and their versions.

## Verification

Run the following from the repository root to check the consolidated module.

```bash
go vet ./...
go test -count=1 -timeout=5m ./...
```

Run independent extension checks from their directories, with `GOEXPERIMENT=jsonv2` enabled for `com/codec/jsonv2`. CI enumerates all workspace modules and also runs Redis-backed core, storage, SSO, and HTTP flow scenarios.

The following Bash command checks every module with `GOWORK=off` and temporary local replacements. It runs vet and tests while keeping module files in the checkout intact.

```bash
bash .github/scripts/check-modules.sh
```

This check verifies the current checkout's standalone dependency graph. The clean consumer release check above verifies remote distribution.
