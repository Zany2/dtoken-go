# Modular design

English | [中文文档](modular_zh.md)

This document defines package dependencies and Go module release boundaries. See [architecture](architecture.md) for layers, [source layout](source-layout.md) for directories and file responsibilities, and the [migration guide](module-migration.md) for upgrade steps.

| Layer | Packages |
| --- | --- |
| Application entry | Root instance/default facades and shared registry; legacy global facades |
| Public contracts and configuration | contract, config, errors |
| Event services | event: emitter contracts and listener dispatch implementation |
| Implementation | internal/engine, internal/builder, internal/defaults |
| Independent capabilities | nonce, oauth2, ticket, shortkey |
| Shared HTTP | httpauth, httpauth/context |
| Framework adapters | integrations/<framework> |
| Components | codec, generator, logging, pool, storage |
| Single sign-on | sso, sso/storage/redis |

## Dependency rules

- The engine depends on contract, config, errors, event, and internal primitives, not frameworks, capabilities, or concrete component implementations.
- Capabilities depend on public contracts and shared primitives. They receive components and optional event emitters through constructors, without depending on the root facade, global, or engine.
- contract contains shared interfaces and data types, without importing capabilities, frameworks, or default wiring. event includes runtime implementation.
- config may depend on contract and errors; contract does not depend back on config. errors and internal primitives do not depend on upper runtime layers, and event does not depend on the engine or assembly.
- The root assembles basic components through the internal builder and default factories. Build automatically registers; BuildInstance opts out. The root owns the shared registry and global delegates to it.
- internal/builder depends on the engine and component contracts without selecting implementations. internal/defaults explicitly selects bundled JSON, generator, logger, pool, and memory storage implementations; update its import allow-list when changing bundled defaults.
- Shared HTTP code calls the engine and capability service contracts. It imports neither concrete capabilities nor global. Registry lookup requires an explicit ManagerResolver.
- Framework adapters use shared HTTP code and handle native requests, responses, and handler chains. Their builders forward to the root and return Auth; they do not re-export global calls or optional component constructors.
- Framework adapters may use helper subpackages within their own directory. They do not directly import concrete components, internal builders, peer adapters, SSO, or example/test applications. The root Builder's default components remain an intentional indirect assembly dependency.
- Components may depend on their own component directory, contracts, configuration, errors, and internal primitives, not the engine, HTTP, or assembly layers. contract/storagetest is for tests rather than runtime dependencies.
- Add components through their packages. Gin canonical sources and the generator maintain export.go and error_export.go; other adapter code is maintained per framework.

These rules distinguish implementation dependencies from assembly: the root Builder deliberately selects bundled components, while the engine stays independent of concrete components.

## Go module boundaries

| Release unit | Directory scope | Reason for separation |
| --- | --- | --- |
| Root module | Root facade, global, internal, contract, config, errors, event, four capability packages, httpauth, and basic components outside the submodules below | Version the authentication core and basic capabilities together |
| Optional components | codec/jsonv2, codec/msgpack, logging/gf, storage/redis | Isolate optional dependencies or toolchain requirements |
| Framework integrations | Eight framework directories under integrations | Isolate framework dependencies and release independently |
| SSO | sso, sso/storage/redis | Independent subsystem and storage extension |
| Examples and application tests | Applications under examples, tests/gin_core_app, tests/gin_core_flow | Isolate application dependencies from the core runtime |

Each go.mod defines an actual module boundary; go.work only combines modules for local development. nonce, oauth2, ticket, and shortkey are independently composed packages released with the root module. Their service contracts also share that release.

Prefer an existing module for new directories. Add go.mod only to isolate dependencies, toolchains, or release cadence, and update the workspace, CI, examples, and release documentation together. Moving a public package changes consumer imports and is not merely directory housekeeping.

## Releases and ownership

See the [migration guide](module-migration.md) for tag naming, dependency release order, and consumer verification. Source version declarations do not establish that a release is available remotely.

Runtime ownership is independent of module releases: engines close their owned components, capability services borrow components, and the registry shares engines with Auth. See [API and lifecycle](api-lifecycle.md) for the complete contract.

## Existing structural checks

- Root architecture_test.go uses direct repository import allow-lists for contracts, configuration, errors, events, internal primitives, the engine, capabilities, shared HTTP, components, and both internal assembly layers. HTTP and frameworks also have forbidden-dependency rules.
- Checks scan non-_test.go Go files, including separate component and framework submodules present in the checkout. Published root modules skip framework scanning when integrations is absent; component parent directories still cover their bundled implementations.
- architecture_integrations_test.go checks framework request operation name coverage; it does not replace signature or behavior verification.
- The Gin export generator's check mode verifies the two shared export files.
- Standalone module and external consumer CI checks cover dependency resolution and public use outside the workspace.

Import checks do not analyze transitive third-party dependencies or replace public API, compilation, or behavior verification. Functional verification is performed separately for each module.
