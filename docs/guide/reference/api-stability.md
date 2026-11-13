# API boundaries and migration

This branch deliberately changes package paths and capability composition. Old directories, old global entry points, and builder capability switches are not compatibility APIs. See the [migration guide](../../design/module-migration.md).

Public areas include the root instance facade, global, contract, config, errors, event, capability packages, components, frameworks, and SSO. Applications do not import internal implementation packages; nameable engine extension types are exposed by the root.

Consumers must select matching root and extension releases. Gin canonical sources maintain framework convenience exports. Applications prefer the root builder and explicit instance injection. Run standalone module and external consumer checks before release.
