# Module organization and migration

English | [中文文档](module-migration_zh.md)

This guide owns old-path and API migration, release ordering, and consumer verification. See [architecture](architecture.md) for system layers, [modular design](modular.md) for package and module boundaries, and [source layout](source-layout.md) for code locations.

## Error-handling API migration

Construction and required context reads now return errors for the caller to handle:

| Previous call | Current call |
| --- | --- |
| `MustBuild()` | `auth, err := builder.Build()` |
| `MustBuildInstance()` | `auth, err := builder.BuildInstance()` |
| `WithAuth(auth)` used directly as an option | Obtain `option, err := WithAuth(auth)` before configuring middleware |
| `Context(auth)` used directly as middleware | Obtain `middleware, err := Context(auth)` before registering it |
| `RequestContext.MustGet(key)` | `value, err := requestContext.GetRequired(key)` |

`WithAuth` and `Context` return `ErrManagerNotFound` for nil or closed instances. `GetRequired` returns an error matching `ErrKeyNotFound` through `errors.Is` when a key is absent. Keep using `Get(key) (any, bool)` when an existence check is sufficient, and update custom RequestContext implementations to the new method signature. GoFrame body-read panics are returned through `GetBody() ([]byte, error)` rather than rethrown.

## Registry and configuration changes

- Operation options no longer contain AuthType. Use dtoken.XXX for the default system and dtoken.AuthType(name).XXX for a named system.
- BuildAndSetManager has been removed. Use Builder.AuthType(name).Build() for initialization, or BuildInstance() followed by SetAuth with error handling for explicit replacement.
- SetAuth and SetManager now return error. Nil or closed instances return ErrManagerNotFound; a Manager without configuration returns ErrManagerInvalidType. Failed registration preserves the existing engine; registering the same open instance again returns nil.
- Manager.GetConfig() and Auth.Config() return independent snapshots. Configure a new instance to change authentication settings; changing AuthType or KeyPrefix does not migrate stored data.
- HTTP request paths borrow internal/engine configuration read-only, avoiding a copy on each check. integration_facade_architecture_test.go checks grouped facade destinations and arguments across all eight adapters.

## Current boundaries

The root module publishes the instance facade, public contracts, authentication implementation, basic components, shared HTTP authentication, and independent capability packages. Capabilities are composed explicitly; they are not separate Go modules. The workspace contains 30 modules.

Frameworks, codec/jsonv2, codec/msgpack, logging/gf, storage/redis, sso, and sso/storage/redis are released separately. Examples and application tests retain their own modules.

## Package migration

This is a complete package and API migration. Old paths are not compatibility entry points.

| Previous entry | Current entry |
| --- | --- |
| dtoken subdirectory | Root instance/default APIs and AuthType selectors |
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
)

auth, err := dtoken.NewBuilder().AuthType("user").Build()
if err != nil {
    panic(err)
}
defer auth.Close()

// Build registers automatically. Build 自动注册。
selected := dtoken.AuthType("user")
_ = selected
```

Build() automatically registers and returns *dtoken.Auth; auth.Manager() exposes the underlying manager. Prefer framework WithAuth(auth) or Context(auth). Import strategies, assembly options, and factories from the root, never from internal. Every framework NewBuilder() forwards to the root builder and Build() returns Auth. Frameworks no longer re-export global APIs or component constructors. HTTP middleware requires a bound instance; enable global lookup explicitly with WithManagerResolver(dtoken.GetManager).

Duplicate Build calls return ErrManagerAlreadyRegistered. Use BuildInstance for independent instances with the same namespace, or BuildInstance followed by SetAuth for explicit replacement. Root default calls and AuthType selectors share one registry. Drain requests before CloseAll. See [API and lifecycle](api-lifecycle.md).

## Access provider migration

SetCustomPermissionListFunc, SetCustomRoleListFunc, and their Ext variants are no longer provided. Inject AccessProviderFunc with root Builder.SetAccessProvider and read namespace, account, device, and token from AccessSubject. Nil results fall back to the session; non-nil empty slices mean no access. See [AccessProvider](../guide/core/access-provider.md).

## Independent capabilities

Manager/Server constructors for Nonce, OAuth2, Ticket, and ShortKey, including default constructors, now return `(instance, error)`. Handle the error before calling `NewService`. A nil configuration in `WithConfig` selects defaults; explicit configurations must pass `Validate`, so start custom settings from `DefaultConfig()`. Invalid settings, missing components, and typed-nil storage/codecs return `errors.ErrInvalidParam`. The direct Nonce TTL must be positive, and OAuth2 refresh expiration must exceed access expiration. Per-operation timeout fallback rules are unchanged.

The builder has no capability switches. Construct capability Managers and use NewService(manager, auth) for instance events. Share the intended AuthType, KeyPrefix, Storage, and Codec. Bind request services with httpauth/context.Capabilities and framework WithCapabilities. Missing services return ErrModuleNotEnabled. Middleware without an override preserves bindings for the same Manager; changing Manager creates a new scope.

## Facade method migration

Frameworks retain NewBuilder and request helpers such as LoginByContext or LoginByCtx. Import Auth and operation options from the root. The global package has been removed: migrate default calls to dt.XXX and named calls to dt.AuthType(name).XXX, using typed options where required. Replace Default().XXX / NewByAuthType(name).XXX with dt.XXX / dt.AuthType(name).XXX; operations report lookup errors, or call GetManager for an immediate check. Import components from their own packages.

Changing only the import prefix is insufficient: login, permission mutations, and disable operations require root typed options. This table documents upgrade mappings; APIs in the left column have been removed, and dt in the right column names the root:

| Before upgrade | Current call |
| --- | --- |
| `global.Login(ctx, id, device, deviceID, "user")` | `dt.AuthType("user").Login(ctx, dt.LoginOptions{LoginID: id, Device: device, DeviceID: deviceID})` |
| `global.Logout(ctx, token, "user")` | `dt.AuthType("user").LogoutByToken(ctx, token)` |
| `global.AddPermissions(ctx, id, values, "user")` | `dt.AuthType("user").AddPermissions(ctx, dt.PermissionOptions{LoginID: id, Permissions: values})` |
| `global.Disable(ctx, id, duration, reason, "user")` | `dt.AuthType("user").Disable(ctx, dt.DisableOptions{LoginID: id, Duration: duration, Reason: reason})` |
| `global.Default().XXX(...)` / `global.NewByAuthType("user").XXX(...)` | `dt.XXX(...)` / `dt.AuthType("user").XXX(...)`; check immediately with `dt.GetManager(...)`, and use `dt.New(mgr)` for a fixed facade |

Creating a selector does not report lookup errors; later operations resolve the registry. Code that relied on immediate constructor errors should call GetManager first. Registered and returned instances share an engine, so closing a fixed facade affects its other sharers. See the [root API reference](../api/dtoken.md) for further methods.

The following table covers the 40 capability methods removed from the old `Auth` facade. Former package-level facade calls use the same service destinations; select a service composed for the intended authentication instance instead of passing an `authType` argument. Import `nonce`, `oauth2`, `ticket`, and `shortkey` from their current root-level package paths. After building `auth`, compose the service variables used in the table:

Place the following fragment inside an initialization function that returns an error:

```go
cfg := auth.Config()
nonceManager, err := nonce.NewDefaultNonceManager(cfg.AuthType, cfg.KeyPrefix, auth.Storage())
if err != nil {
	return err
}
nonceService := nonce.NewService(nonceManager, auth)
oauth2Server, err := oauth2.NewDefaultOAuth2Server(cfg.AuthType, cfg.KeyPrefix, auth.Storage(), auth.Codec())
if err != nil {
	return err
}
oauth2Service := oauth2.NewService(oauth2Server, auth)
ticketManager, err := ticket.NewDefaultManager(cfg.AuthType, cfg.KeyPrefix, auth.Storage(), auth.Codec())
if err != nil {
	return err
}
ticketService := ticket.NewService(ticketManager, auth)
shortKeyManager, err := shortkey.NewDefaultManager(cfg.AuthType, cfg.KeyPrefix, auth.Storage(), auth.Codec())
if err != nil {
	return err
}
shortKeyService := shortkey.NewService(shortKeyManager, auth)
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

Maintainers explicitly run these commands. Standalone module checks copy modules outside the checkout, exclude nested modules, and use GOWORK=off with temporary replacements. The external consumer covers instances, public extension types, registry registration, HTTP capability bindings, and logout.

```bash
go run ./.github/scripts/check-modules
go run ./.github/scripts/check-docs
bash .github/scripts/check-consumer.sh
bash .github/scripts/check-consumer.sh "$ROOT_VERSION" "$GIN_VERSION"
```

The last command requires both published versions and verifies selected remote versions. JSON v2 requires GOEXPERIMENT=jsonv2. Redis CI covers the root module, storage/redis, sso/storage/redis, and tests/gin_core_flow.

See [API and lifecycle](api-lifecycle.md) for supported public APIs, component ownership, and registry replacement. HTTP capability bindings depend on service interfaces under contract, not concrete implementations.
