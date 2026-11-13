# Modular design

English | [中文文档](modular_zh.md)

Package boundaries follow responsibilities; Go module boundaries isolate optional dependencies and releases. See the [migration guide](module-migration.md).

| Layer | Packages |
| --- | --- |
| Application entry | Root instance facade, global registry |
| Public contracts | contract, config, errors, event |
| Implementation | internal/engine, internal/builder, internal/defaults |
| Independent capabilities | nonce, oauth2, ticket, shortkey |
| Shared HTTP | httpauth, httpauth/context |
| Framework adapters | integrations/<framework> |
| Components | codec, generator, logging, pool, storage |
| Single sign-on | sso, sso/storage/redis |

## Dependency rules

- The engine depends on public contracts, not frameworks or capabilities.
- Capabilities receive storage, codecs, and optional event emitters through constructors.
- The root wires basic components; instances require no global registration. global depends on the root facade and owns its registry.
- HTTP contexts receive explicit instances and services; frameworks adapt requests and middleware. HTTP packages do not import global. Registry lookup requires an explicit ManagerResolver.
- Framework builders return the root Auth type. Frameworks do not re-export global calls or optional component constructors.
- Add components through their packages. Shared framework exports use Gin canonical sources and the generator.

## Releases and ownership

Basic packages and capabilities share the root version. Optional components, frameworks, and SSO release independently. Workspace, standalone module, and external consumer checks verify dependency boundaries.

The engine closes its owned Storage, Logger, Pool, and background tasks. Declare shared components caller-owned with root WithComponentOwnership. global closes a replaced Manager with the same AuthType; applications coordinate request shutdown and resource release. Capability services borrow components and do not close them independently.

See [API and lifecycle](api-lifecycle.md) for supported public APIs, component ownership, and registry replacement. HTTP capability bindings depend on service interfaces under contract, not concrete implementations.
