# Public API, capability contracts, and lifecycle

The public API has two supported levels: applications use the root Auth facade; custom adapters may use root Manager, ManagerOption, and Strategy. Manager is a supported advanced API, not a transitional compatibility alias. Package paths under internal are not public API.

## Instances and engines

Root and framework builders return Auth. Auth.Manager() exposes the shared engine; New(existingManager) creates another facade over that engine, not an independent copy. Closing any wrapper makes every wrapper unavailable. IsClosed reports this state, auth operations return ErrManagerNotFound, and Config, Storage, Codec, and EventManager return nil.

Obtain Manager from a built Auth. Its zero value is unsupported. Do not mutate runtime configuration or replace components after serving requests; build another instance instead.

## Resource ownership

| Situation | Ownership |
| --- | --- |
| Successful build | Storage, logger, and pool are engine-owned by default, including explicitly supplied components |
| Failed build | Factory-created components are cleaned up; explicitly supplied components remain caller-owned |
| WithComponentOwnership | Sets the complete policy; false fields remain caller-owned |
| Builder.Clone | Copies configuration but shares injected components, providers, and callback state |
| New(existingManager) | Shares the original engine lifetime |
| Capability Service | Borrows components; the event emitter does not transfer ownership |

Closing an engine is idempotent. There is no cross-engine reference counting for shared components. When several Auth instances share storage, a logger, or a pool, set its ownership flag to false and close it from the application after all users finish. Use factories returning fresh components for independent ownership.

Auth.Close stops maintenance, waits for accepted asynchronous work, and releases owned resources. It does not drain HTTP handlers. Stop accepting requests, wait for handlers, close Auth instances, and finally close caller-owned resources.

## Global registration

Applications register through global.SetAuth(auth). SetManager remains available for advanced adapters. Both share the same registry and ownership rules:

- Re-registering the same instance leaves it open.
- Replacement under the same auth type closes the old engine.
- DeleteManager and DeleteAllManager remove entries before closing engines.
- Nil and closed instances cannot be registered.
- Registration does not clone the instance. Auth.Close remains available, and later registry lookups remove closed entries.

WithAuth binds a concrete instance and never follows registry replacement automatically. WithManagerResolver(global.GetManager) explicitly selects through the registry per request. An established request scope keeps its instance; closing it does not silently switch that request to a replacement. Applications requiring uninterrupted in-flight requests must coordinate routing and drain requests before retiring the old instance.

## Capability contracts

contract/nonce, contract/oauth2, contract/ticket, and contract/shortkey define service interfaces and data types. HTTP contexts and framework adapters depend on these contracts, not on concrete capability implementations or global.

The bundled capability packages implement these interfaces with compile-time assertions. Their existing public data names alias the contract types, preserving source compatibility and serialized fields.

Capabilities accepts custom implementations. Missing and typed-nil bindings retain disabled-module errors or false for boolean checks. WithCapabilities(nil) preserves bindings; an empty Capabilities clears them. Rebinding a request adapter preserves services, while switching authentication instances clears capability bindings.

Standalone Service lifetimes are caller-managed. Supplying an Auth as the event emitter establishes event delivery only; callers must keep borrowed storage and codecs available. HTTP middleware additionally validates the bound engine lifecycle.

Contracts and bundled capabilities remain part of the root module release. This separates package dependencies and runtime composition without adding separate versions or a dynamic plugin mechanism.
