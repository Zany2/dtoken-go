# Source layout and responsibilities

English | [中文文档](source-layout_zh.md)

This guide explains where code lives and where changes belong. See [architecture](architecture.md) for system layers, [modular design](modular.md) for dependencies and release boundaries, and [API and lifecycle](api-lifecycle.md) for public APIs and ownership. This map describes organization, not verification of individual feature behavior.

## Directory map

Paths are relative to the repository root. Each `go.mod` defines a module; `go.work` combines modules for local development. A directory with an independent responsibility does not necessarily have an independent release.

```text
dtoken-go/
├── auth*.go              Instance facade and grouped operations
├── builder.go            Application construction entry
├── options.go            Instance operation options
├── types.go              Public domain type aliases
├── engine_api.go         Advanced engine APIs and assembly types
├── config.go             Root configuration entry
├── registry.go           Shared registry, AuthType selection and shutdown
├── default_*.go          Default namespace forwarding
├── internal/
│   ├── engine/           Core authentication implementation
│   ├── builder/          Component assembly and bundled default factories
│   ├── utils/            Shared internal primitives
│   └── banner/           Startup display
├── contract/             Component, domain, and request contracts
│   ├── nonce/            Nonce service contract
│   ├── oauth2/           OAuth2 service contract
│   ├── ticket/           Ticket service contract
│   ├── shortkey/         ShortKey service contract
│   └── storagetest/      Reusable storage contract checks
├── config/               Core configuration and constants
├── errors/               Public errors and codes
├── event/                Event contracts and listener dispatch
├── nonce/                Nonce implementation and service
├── oauth2/               OAuth2 implementation and service
├── ticket/               Ticket implementation and service
├── shortkey/             ShortKey implementation and service
├── httpauth/             Shared HTTP checks and error mapping
│   └── context/          Request scope, bindings, and auth operations
├── integrations/         Framework adapters
├── codec/                Codec implementations
├── generator/            Token generator implementations
├── logging/              Logger implementations
├── pool/                 Async execution implementations
├── storage/              Storage implementations
├── sso/                  Independent SSO subsystem
├── examples/             Application integration examples
├── tests/                Application-level composition checks
├── docs/                 Tutorials, guides, references, and design
└── .github/              CI, repository checks, and collaboration
```

## Application entry and assembly

| Location | Responsibility | Placement boundary |
| --- | --- | --- |
| Root `Auth` and `auth_*.go` | Translate application calls into engine operations and expose instance lifecycle | Authentication rules and persistence belong in the engine |
| Root `builder.go` | Application configuration, bundled logger/pool options, and construction of `Auth` | Own configuration, components, and factories directly; call stateless assembly |
| `options.go`, `types.go`, `config.go` | Public options, shared model aliases, and configuration entry points | Keep one declaration source for shared models |
| `engine_api.go` | Supported advanced engine types and assembly options | Aliases remain public APIs even when their implementation lives under internal |
| Root `registry.go` and `default_*.go` | Shared registry, default calls and AuthType selectors | Engine and shared HTTP code must not depend on the registry |
| `internal/builder/builder.go` | Stateless Build function: validate configuration, resolve components/factories, and build the engine | Resolve supplied contracts and factories |
| `internal/builder/defaults.go` | Supply bundled component factories in the same assembly package | Keep concrete default selection here; applications may override components |

Only the root package has a fluent Builder; internal/builder groups assembly and default factory functions without duplicate setters or Builder state. Add component implementations in their component packages; extend the root Builder only when an application configuration entry is needed.

## Public foundations and capabilities

| Location | Contents |
| --- | --- |
| `contract/storage.go`, `codec.go`, `generator.go`, `log.go`, `pool.go` | Replaceable component contracts |
| `contract/model.go`, `access.go` | Authentication domain models and access provider contracts |
| `contract/context.go` | Framework request, response, and Cookie adapter contracts |
| `contract/<capability>/service.go` | Capability service interfaces and shared data types |
| `contract/storagetest` | Reusable storage contract checks; runtime code does not depend on them |
| `config`, `errors` | Core configuration and public errors |
| `event` | Event payloads, emitter contracts, and listener runtime implementation |
| `nonce`, `oauth2`, `ticket`, `shortkey` | Capability-specific configuration, implementation, and event-aware service wrappers |

Each capability's `service.go` implements its request-facing service contract and connects an event emitter. Its Manager or Server implements the underlying operations. Capability configuration stays with its capability, and component configuration stays with its component, instead of accumulating in core `config.Config`.

`contract` currently groups component, domain, and HTTP contracts. Extend the existing categorized files; create subpackages only for independent dependency or maintenance needs. Contracts neither wire concrete components nor import capability implementations. `event` includes runtime behavior and is not solely a type declaration package.

## Engine file index

These files share the `internal/engine` package and divide state ownership and common operations. Filename prefixes aid navigation; they do not create Go visibility boundaries.

| File or group | Responsibility |
| --- | --- |
| `manager.go` | Manager state composition and component access |
| `constructor.go` | Engine assembly options and component ownership |
| `model.go` | Aliases for shared contract models |
| `strategy.go`, `auth_options.go` | Strategy hooks and engine operation options |
| `auth_login.go`, `auth_logout.go`, `auth_terminate.go` | Login and terminal termination operations |
| `auth_refresh.go`, `auth_introspection.go` | Refresh credentials and token status queries |
| `auth_session*.go` | Session queries and extension data operations |
| `auth_access.go`, `auth_permission.go`, `auth_role.go` | Access providers, permission and role operations |
| `auth_disable.go` | Disable operation entry points |
| `internal_account_lock.go` | Account write lock registry and lifetime |
| `internal_concurrency.go` | Internal login concurrency policy operations |
| `internal_maintenance.go` | Token maintenance generations and merged activity timestamps |
| `internal_lifecycle.go` | Background task admission, shutdown, and resource release |
| `internal_session.go`, `internal_token.go`, `internal_terminal.go` | Shared Session, Token, and terminal operations |
| `internal_storage.go`, `internal_keys.go` | Persistence, expiration conversion, and storage keys |
| `internal_access_mutation.go`, `internal_access_values.go` | Access mutation flows and permission/role list operations |
| `internal_disable.go`, `internal_events.go` | Disable primitives and engine event publishing |

Manager owns account coordination, Token maintenance, and runtime shutdown state separately. The files share this state and its primitives, so they remain in one package. Place new operations in their corresponding files; consider splitting packages only when state ownership and dependencies can be independent.

## Shared HTTP and framework directories

`httpauth` owns common checks, Manager resolution, and error mapping. `httpauth/context` owns request token extraction, Cookie operations, authentication operations, and capability bindings. Frameworks supply request adapters rather than reimplementing domain rules.

The eight framework directories are `beego`, `chi`, `echo`, `fiber`, `gf`, `gin`, `hertz`, and `kratos`. They share this file layout:

| File | Responsibility |
| --- | --- |
| `context.go` | Adapt native requests and responses |
| `middleware.go` | Integrate the handler chain, configuration, and auth failure handling |
| `instance.go` | Supply Auth instance injection helpers |
| `annotation.go` | Translate annotations into request checks |
| `context_facade.go`, `context_*_facade.go` | Expose operations accepting native framework contexts |
| `export.go`, `error_export.go` | Consistent convenience exports of public types, constants, and errors |

The two Gin export files are canonical sources. `integrations/gin/internal/integrationexportgen` maintains their generated counterparts in other frameworks. It does not generate middleware, request adapters, or all facade methods. New shared operations require corresponding framework entry points and maintenance of existing coverage checks.

## Examples, verification, and documentation

Package-local `*_test.go` files follow their implementation. Root `architecture*_test.go` files constrain cross-package structure. `tests/gin_core_app` provides a composed application, and `tests/gin_core_flow` checks its flows. Examples prioritize integration guidance and must not become runtime dependencies.

`.github/workflows` orchestrates CI. `.github/scripts` contains module, documentation-example, and external-consumer checks. Keep these tools and the Gin export generator at their existing locations; place new tools near the code they maintain.

`docs/tutorial` covers onboarding, `docs/guide` covers usage, `docs/api` covers API references, and `docs/design` covers design decisions. Additions and moves must update English and Chinese navigation and relative links. Link feature walkthroughs to their corresponding guides.

## Where changes belong

| Change | Preferred location |
| --- | --- |
| Core authentication rule | `internal/engine`, with a root facade operation if needed |
| Dynamic access or component extension interface | Check existing `contract` interfaces first |
| Storage, codec, or logger implementation | Its component directory; split modules according to dependency and release needs |
| Independent capability | Its capability package and service contract when HTTP injection is needed |
| Shared request authentication | `httpauth` or `httpauth/context` |
| Native framework behavior | The corresponding `integrations/<framework>` |
| Global registration and lookup | Root `registry.go` |
| Cross-component usage scenario | `examples` for guidance or `tests` for verification |

SSO remains an independent module. This guide identifies its repository location; its internal layout and protocols are reviewed separately.
