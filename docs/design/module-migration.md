# Module organization and migration

English | [中文文档](module-migration_zh.md)

## Current boundaries

The root module publishes the instance facade, public contracts, authentication implementation, basic components, shared HTTP authentication, and independent capability packages. Capabilities are composed explicitly; they are not separate Go modules. The workspace contains 30 modules.

Frameworks, codec/jsonv2, codec/msgpack, logging/gf, storage/redis, sso, and sso/storage/redis are released separately. Examples and application tests retain their own modules.

## Package migration

This is a complete package and API migration. Old paths are not compatibility entry points.

| Previous entry | Current entry |
| --- | --- |
| dtoken subdirectory | Root instance API and explicit global package |
| core/adapter | contract |
| core/config, core/derror, core/listener | config, errors, event |
| core/manager, core/builder, defaults | Root NewBuilder and public extension types |
| core/context, integrations/authcheck | httpauth/context, httpauth |
| core/nonce, core/oauth2, core/ticket, core/shortkey | nonce, oauth2, ticket, shortkey |
| com/codec, com/generator, com/log, com/pool, com/storage | codec, generator, logging, pool, storage |

## Application construction

```go
import (
    "github.com/Zany2/dtoken-go"
    "github.com/Zany2/dtoken-go/global"
)

auth, err := dtoken.NewBuilder().AuthType("user").Build()
if err != nil {
    panic(err)
}
defer auth.Close()

// Optional global registration. 可选全局注册。
global.SetAuth(auth)
defer global.DeleteManager("user")
```

Build() returns *dtoken.Auth; auth.Manager() exposes the underlying manager. Prefer framework WithAuth(auth) or Context(auth). Import strategies, assembly options, and factories from the root, never from internal. Every framework NewBuilder() forwards to the root builder and Build() returns Auth. Frameworks no longer re-export global APIs or component constructors. HTTP middleware requires a bound instance; enable global lookup explicitly with WithManagerResolver(global.GetManager).

## Independent capabilities

The builder has no capability switches. Construct capability Managers and use NewService(manager, auth) for instance events. Share the intended AuthType, KeyPrefix, Storage, and Codec. Bind request services with httpauth/context.Capabilities and framework WithCapabilities. Missing services return ErrModuleNotEnabled. Middleware without an override preserves bindings for the same Manager; changing Manager creates a new scope.

## Dependencies and releases

Remove dependencies and replacements for old submodules, then select new root and extension versions. The root uses a vX.Y.Z tag; extensions use current directory prefixes such as integrations/gin/vX.Y.Z and storage/redis/vX.Y.Z. The source dependency baseline v0.0.3 does not establish remote availability of the new structure. Release dependencies before updating and releasing extensions. Runtime version information lives in contract/version.go.

## Verification

Maintainers explicitly run these commands. Standalone module checks copy modules outside the checkout, exclude nested modules, and use GOWORK=off with temporary replacements. The external consumer covers instances, public extension types, global registration, HTTP capability bindings, and logout.

```bash
go run ./.github/scripts/check-modules
go run ./.github/scripts/check-docs
bash .github/scripts/check-consumer.sh
bash .github/scripts/check-consumer.sh "$ROOT_VERSION" "$GIN_VERSION"
```

The last command requires both published versions and verifies selected remote versions. JSON v2 requires GOEXPERIMENT=jsonv2. Redis CI covers the root module, storage/redis, sso/storage/redis, and tests/gin_core_flow.

See [API and lifecycle](api-lifecycle.md) for supported public APIs, component ownership, and registry replacement. HTTP capability bindings depend on service interfaces under contract, not concrete implementations.
