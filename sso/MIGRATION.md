# SSO upgrade and release guide

[中文](MIGRATION_zh.md) · [Release checklist](RELEASE_CHECKLIST.md)

This guide collects compatibility checks for existing SSO consumers and verification outside the workspace. It assigns no release version and does not establish passing builds, regression tests or deployment acceptance for the current changes.

## Before upgrading

| Area | Integration requirement |
| --- | --- |
| Construction and defaults | Existing constructors remain available; prefer `NewHTTPServerChecked` and `NewClientAppChecked`. Start with `DefaultHTTPOptions()` / `DefaultClientConfig()` for boolean defaults. Zero-valued structs do not enable signing or SLO |
| Parameters | Send each protocol field once, including across query/body. Put multiple scopes in one value. Credential operations must omit fields for other modes, including empty foreign fields |
| Inbound size | Center and application receivers default to 8 KiB queries and 64 KiB form bodies; overflow returns 413. Coordinate RequestLimits and proxies for larger extensions. Middleware parsing first must bound raw reads and handle errors; see [request limits](REQUEST_LIMITS.md) |
| Mode selection | SDK requests use the request mode or configured mode, without inferring from nonempty struct fields. Raw HTTP `/token` still infers from nonempty code when mode is absent; other credential endpoints default to Ticket |
| URLs and POST | Exchange through `ExchangeTicket` / `ExchangeOAuth2Code`. Legacy exchange URL helpers are unsuitable for sending client secrets. Submit PKCE verifiers only through POST |
| Keys and origins | Client signing keys override the center fallback; exchange secrets are not signing keys. Redirect URIs remain exact matches; origin normalization does not broaden redirect allow-lists |
| Replay and PKCE | Coordinate both ends, parameter names and clock windows. Have clients send PKCE proofs before enforcing `RequirePKCE` at the center; restart pending flows lacking proofs |
| Browser flows | Server-side state, cookies and PKCE verifiers belong to the same BrowserClient flow. Restart login when migrating handwritten callbacks. Failure after one-time consumption cannot be retried with the same credential |
| Logout | Supply a trusted `LoginIDResolver` and `CenterLogout`. HTTP logout and trusted `LogoutAccount` use different ordering, described below. Callback handlers must be idempotent |
| Storage and ownership | Replicas of a service share storage and namespaces; applications close borrowed storage. Ordinary Storage remains supported, but cross-replica one-time competition cannot be guaranteed without atomic extensions |

See [configuration](CONFIGURATION.md), [request consistency](REQUEST_VALIDATION.md), [browser integration](BROWSER_CLIENT.md) and [replay protection](REPLAY_PROTECTION.md) for details and error classifications. These are this project's protocol rules, not evidence of direct interoperability with Java Sa-Token, general OAuth2 or OIDC implementations.

## Logout and rolling upgrades

Ordinary HTTP global logout notifies applications before revoking center sessions and SSO credentials. Strict notification failures preserve later stages for retry. Trusted administrative `LogoutAccount` revokes center sessions and old credentials before notification, retaining unfinished bindings for retry. Completed stages are not rolled back. Permanent account suspension also requires the identity system to deny new login and issuance.

Account revocation depends on every participating replica checking logout versions. Mixing in old versions that do not understand these checks cannot guarantee consistent revocation. Update all services validating or consuming credentials, then verify account logout and cross-replica replay rejection. Do not roll back by deleting logout-version records or changing storage prefixes.

Optional VerificationKey supports overlapping verification. Upgrade all receivers first, then prepare, switch and retire keys using the [rotation guide](SIGNING_ROTATION.md). Cookie keys or browser-state format changes may require users to log in again. Drain in-flight requests before stopping old replicas and retain live credentials, nonces and logout-version records. See [deployment](DEPLOYMENT.md) for operational boundaries.

## Modules and publishing

SSO, its Redis adapter and the five examples are separate Go modules. Root `go test ./...` does not include them. The workspace Go version and each module's minimum Go version are separate verification targets; CI now includes minimum-Go jobs reading the SSO and SSO Redis go.mod files independently; configured checks are not evidence of passing execution.

Publish a compatible root module first. Both SSO and the general Redis adapter depend on the root; the SSO Redis adapter depends on SSO and the general Redis adapter. Use each module's `go.mod` for exact requirements and update actual dependency versions and module tags before release. Follow the existing [module migration process](../docs/design/module-migration.md).

The isolation checker accepts a module selection. Run manually from the repository root:

```powershell
go run ./.github/scripts/check-modules ./sso ./sso/storage/redis ./examples/sso_server ./examples/sso_client ./examples/sso_gin_server ./examples/sso_gin_client ./examples/sso_deployment
go run ./.github/scripts/check-docs
```

The first command copies selected modules to temporary directories, disables workspace resolution, adds checkout replacements and runs vet/test. It checks source dependency boundaries, not published availability. The second compiles marked documentation programs, now including both SSO configuration examples. It does not run examples or compile all unmarked snippets.

Use the existing Bash consumer entry point, through Git Bash on Windows if needed:

```bash
# Current root, Gin and SSO sources; a temporary module uses checkout replacements.
bash .github/scripts/check-consumer.sh

# Published modules: set variables to actual release tags first; no checkout replacements.
bash .github/scripts/check-consumer.sh "$ROOT_VERSION" "$GIN_VERSION" "$SSO_VERSION"
```

The existing two-version form still checks only root and Gin. Three versions include published SSO. After dependency resolution, selected versions must equal the requested versions so transitive requirements cannot silently substitute a different release. The SSO consumer covers four modes through public construction, the HTTP facade, signing, custom names, replay configuration, PKCE exchange for Ticket/Code and invalidation after consumption/revocation. It uses memory storage and in-process HTTP transport without listening. It does not include the published Redis adapter, which still needs separate verification.

## Dedicated SSO release checks

The SSO release set consists of root, `sso`, `storage/redis` and `sso/storage/redis`. The dedicated script does not depend on Gin or load `go.work`; it checks public APIs in a temporary consumer module. Run manually, using Git Bash on Windows:

```bash
# Current sources: replace only these four modules.
bash .github/scripts/check-sso-consumer.sh

# Published modules: set four actual versions first; no source replacements.
bash .github/scripts/check-sso-consumer.sh "$ROOT_VERSION" "$SSO_VERSION" "$REDIS_VERSION" "$SSO_REDIS_VERSION"

# Install the Go version matching the target go.mod before isolated checks.
GOWORK=off GOTOOLCHAIN=local go run ./.github/scripts/check-modules -sso ./sso
GOWORK=off GOTOOLCHAIN=local go run ./.github/scripts/check-modules -sso ./sso/storage/redis
```

With no paths, `-sso` checks SSO and SSO Redis. It discovers replacements from four go.mod files without workspace discovery. Disposable module copies use readonly vet/test; checkout manifests/checksums are not rewritten. Discovery regression sources are added but have not run.

The consumer accepts zero or four arguments. Published mode requires explicit versions (including prereleases and pseudo-versions), rejecting empty values, branches and latest. After resolution it verifies every selected version and rejects all replacements, so transitive version promotion cannot silently pass. It fixes `GOWORK=off` and `GOTOOLCHAIN=local`; no automatic toolchain upgrade occurs. The source consumer composes both Redis adapters, so the installed Go must satisfy the entire dependency set.

Fixtures cover four credential modes, signatures, replay, PKCE, browser login, state preservation after size rejection, overlapping keys, account logout and borrowed/owned Redis connection closure. HTTP transport is in-process; Redis constructor tests block network dialing and do not establish real Redis acceptance. Browser tests use a local-session marker, not the full deployment example's Auth sessions. The script runs vet and race tests and requires a supported race toolchain.

The new [SSO published consumer workflow](../.github/workflows/sso-release.yml) is manual-only with four required versions. It verifies published modules without creating tags or publishing. It uses the chosen source ref's sso/go.mod Go version and fixtures; old releases lacking new APIs are not expected to pass. Regular CI adds the `SSO minimum Go` matrix for SSO and its Redis adapter, with the source consumer running in the SSO entry.

## CI and acceptance records

Workspace CI already iterates modules and runs live Redis scenarios. This batch adds race checks for SSO, its Redis adapter and all five examples; adds SSO configuration snippets to documentation compilation; and includes SSO in the regular external consumer check. Manual CI can accept actual root, Gin and SSO versions for published-consumer verification. The race step does not set a Redis URL; a separate step handles live Redis.

Only static inspection and formatting were performed in this batch; neither these commands nor CI were run. Release records should include the source revision and uncommitted changes, Go version/platform, command results, whether Redis actually ran, skipped checks, and browser/proxy acceptance. See the [release checklist](RELEASE_CHECKLIST.md) for the full verification commands.
