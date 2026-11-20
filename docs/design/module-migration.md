# Module organization and migration

English | [中文文档](module-migration_zh.md)

This guide owns old-path and API migration, release ordering, and consumer verification. See [architecture](architecture.md) for system layers, [modular design](modular.md) for package and module boundaries, and [source layout](source-layout.md) for code locations.

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

## Access provider migration

SetCustomPermissionListFunc, SetCustomRoleListFunc, and their Ext variants are no longer provided. Inject AccessProviderFunc with root Builder.SetAccessProvider and read namespace, account, device, and token from AccessSubject. Nil results fall back to the session; non-nil empty slices mean no access. See [AccessProvider](../guide/core/access-provider.md).

## Independent capabilities

The builder has no capability switches. Construct capability Managers and use NewService(manager, auth) for instance events. Share the intended AuthType, KeyPrefix, Storage, and Codec. Bind request services with httpauth/context.Capabilities and framework WithCapabilities. Missing services return ErrModuleNotEnabled. Middleware without an override preserves bindings for the same Manager; changing Manager creates a new scope.

## Facade method migration

Framework packages retain `NewBuilder` and request helpers such as `LoginByContext` or `LoginByCtx`. Import `Auth` and authentication option types from the root package. Replace framework-level global calls such as `Default`, `NewByAuthType`, `SetManager`, and `Login` with the corresponding `global` entry, or use the built `Auth` instance directly for authentication operations. Component constructors come from their component packages.

The following table covers the 40 capability methods removed from the old `Auth` facade. Former package-level facade calls use the same service destinations; select a service composed for the intended authentication instance instead of passing an `authType` argument. Import `nonce`, `oauth2`, `ticket`, and `shortkey` from their current root-level package paths. After building `auth`, compose the service variables used in the table:

```go
cfg := auth.Config()
nonceService := nonce.NewService(nonce.NewDefaultNonceManager(cfg.AuthType, cfg.KeyPrefix, auth.Storage()), auth)
oauth2Service := oauth2.NewService(oauth2.NewDefaultOAuth2Server(cfg.AuthType, cfg.KeyPrefix, auth.Storage(), auth.Codec()), auth)
ticketService := ticket.NewService(ticket.NewDefaultManager(cfg.AuthType, cfg.KeyPrefix, auth.Storage(), auth.Codec()), auth)
shortKeyService := shortkey.NewService(shortkey.NewDefaultManager(cfg.AuthType, cfg.KeyPrefix, auth.Storage(), auth.Codec()), auth)
```

| Previous method name | Current service method |
| --- | --- |
| `GenerateNonce` | `nonceService.GenerateNonce` |
| `GenerateNonceWithTimeout` | `nonceService.GenerateNonceWithTimeout` |
| `VerifyNonce` | `nonceService.VerifyNonce` |
| `VerifyAndConsumeNonce` | `nonceService.VerifyAndConsumeNonce` |
| `IsNonceValid` | `nonceService.IsNonceValid` |
| `GetNonceTTL` | `nonceService.GetNonceTTL` |
| `RegisterOAuth2Client` | `oauth2Service.RegisterOAuth2Client` |
| `UnregisterOAuth2Client` | `oauth2Service.UnregisterOAuth2Client` |
| `GetOAuth2Client` | `oauth2Service.GetOAuth2Client` |
| `OAuth2Token` | `oauth2Service.OAuth2Token` |
| `GenerateOAuth2AuthorizationCode` | `oauth2Service.GenerateOAuth2AuthorizationCode` |
| `GenerateOAuth2AuthorizationCodeWithPKCE` | `oauth2Service.GenerateOAuth2AuthorizationCodeWithPKCE` |
| `ExchangeOAuth2CodeForToken` | `oauth2Service.ExchangeOAuth2CodeForToken` |
| `ExchangeOAuth2CodeForTokenWithPKCE` | `oauth2Service.ExchangeOAuth2CodeForTokenWithPKCE` |
| `OAuth2ClientCredentialsToken` | `oauth2Service.OAuth2ClientCredentialsToken` |
| `OAuth2PasswordGrantToken` | `oauth2Service.OAuth2PasswordGrantToken` |
| `RefreshOAuth2AccessToken` | `oauth2Service.RefreshOAuth2AccessToken` |
| `ValidateOAuth2AccessToken` | `oauth2Service.ValidateOAuth2AccessToken` |
| `ValidateOAuth2AccessTokenAndGetInfo` | `oauth2Service.ValidateOAuth2AccessTokenAndGetInfo` |
| `RevokeOAuth2Token` | `oauth2Service.RevokeOAuth2Token` |
| `CreateTicket`, `CreateTicketWithOptions` | `ticketService.CreateTicket` |
| `ValidateTicket`, `ValidateTicketWithOptions` | `ticketService.ValidateTicket` |
| `ConsumeTicket`, `ConsumeTicketWithOptions` | `ticketService.ConsumeTicket` |
| `RevokeTicket` | `ticketService.RevokeTicket` |
| `GetTicketStatus` | `ticketService.GetTicketStatus` |
| `GetTicketTTL` | `ticketService.GetTicketTTL` |
| `CreateShortKey`, `CreateShortKeyWithOptions` | `shortKeyService.CreateShortKey` |
| `ConfirmShortKey`, `ConfirmShortKeyWithOptions` | `shortKeyService.ConfirmShortKey` |
| `ValidateShortKey`, `ValidateShortKeyWithOptions` | `shortKeyService.ValidateShortKey` |
| `ConsumeShortKey`, `ConsumeShortKeyWithOptions` | `shortKeyService.ConsumeShortKey` |
| `RevokeShortKey` | `shortKeyService.RevokeShortKey` |
| `GetShortKeyStatus` | `shortKeyService.GetShortKeyStatus` |
| `GetShortKeyTTL` | `shortKeyService.GetShortKeyTTL` |

Methods without a `WithOptions` merge retain their operation arguments; the receiver changes to the service. The seven `WithOptions` names are consolidated as follows:

- `CreateTicket(ctx, loginID)` becomes `ticketService.CreateTicket(ctx, ticket.CreateOptions{LoginID: loginID})`.
- `CreateShortKey(ctx)` becomes `shortKeyService.CreateShortKey(ctx, shortkey.CreateOptions{})`.
- `ConfirmShortKey(ctx, key, loginID)` becomes `shortKeyService.ConfirmShortKey(ctx, key, shortkey.ConfirmOptions{LoginID: loginID})`.
- Existing creation and confirmation options are passed directly to those methods. Ticket and ShortKey validation and consumption methods accept an optional `ValidateOptions` argument; omit it for the old unconstrained call or pass the original options to retain constraints.

These services borrow the instance components and emit events through `auth`; keep the instance available for their lifetime. For HTTP requests, bind them through `WithCapabilities`. Existing request helpers keep their framework-specific names; Nonce validity is named `IsValidNonceByContext` or `IsValidNonceByCtx`, while the service method is `IsNonceValid`.

## Dependencies and releases

Remove dependencies and replacements for old submodules, then select new root and extension versions. The root uses a vX.Y.Z tag; extensions use current directory prefixes such as integrations/gin/vX.Y.Z and storage/redis/vX.Y.Z. The first release of the refactored architecture targets v0.0.1; the runtime version, internal module requirements, and workspace replacements use this baseline. These source declarations do not establish remote availability of the new structure. Release dependencies before updating and releasing extensions. Runtime version information lives in contract/version.go.

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
