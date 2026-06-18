# API Stability

This project is modular, but its public surface is intended to stay predictable across patch and minor releases.

## Stable Public Areas

- `core` contains the core auth and manager logic.
- `dtoken` exposes the main global facade.
- `defaults` wires the default builder and bundled components.
- `com/*` contains reusable component implementations.
- `integrations/*` exposes framework-specific facades and middleware.
- `sso/*` contains optional SSO-related modules.

## What Should Stay Compatible

- Exported function names and signatures.
- Exported types used by framework adapters and examples.
- Error values returned by public APIs.
- Module paths and package layout for published tags.

## What May Change

- Internal helpers under `internal` paths.
- Private fields and unexported functions.
- Example code and docs that are not part of the API contract.

## Versioning Rule

- Patch releases should not break existing callers.
- Minor releases may add new APIs, but should avoid breaking existing ones.
- Breaking changes should be introduced only with a deliberate version bump.

For package-level usage details, see the [Core API Cheatsheet](core-api-cheatsheet.md).
