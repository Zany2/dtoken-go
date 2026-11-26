# Offline namespace copying

English | [中文文档](namespace-copy_zh.md)

The `migration` package explicitly migrates one AuthType in the same Storage. The current runtime uses `{encoded KeyPrefix}{encoded AuthType}v2:` without falling back to the old unversioned namespace. Deployments should use `PrepareAuthUpgrade` to migrate credentials, restrictions, and colocated capabilities into the actual versioned destination. The tool neither starts applications nor deletes source data. SSO is outside its scope.

| Preparation entry point | Destination and purpose |
| --- | --- |
| `Prepare` | Raw copy at unversioned namespace boundaries; no format conversion |
| `PrepareTokenUpgrade` | Stage token, Extra, and refresh conversion in an unversioned destination; copy restrictions unchanged |
| `PrepareAuthUpgrade` | Migrate an unversioned source into the current `v2:` destination, including restriction normalization; use for runtime upgrades |

## Preconditions

- Storage must implement `contract.ScannerStorage`. Both prefixes must be explicit. Omitting AuthType selects the builder default; names use builder normalization. `PrepareAuthUpgrade` allows equal configured prefixes because the destination adds a fixed `v2:` segment. Raw copying and token staging still require non-overlapping namespaces.
- Stop request writes, auto-renewal, activity updates, scheduled maintenance, and independent capability writers on every instance before `Prepare`; keep them stopped through cutover. The tool does not detect source changes or support online incremental migration.
- Closing Auth may close storage connections it owns. Drain requests and background work, then use a live connection managed by the migration program. The tool borrows Storage and does not close it.
- The implementation enumerates all backend keys, filters literal namespace prefixes, and snapshots selected values in memory. Key-list memory depends on the whole backend; value memory depends on the selected namespace. Assess memory and maintenance-window capacity first. This is not a streaming or paginated large-scale migrator.

## Prepare and apply

Use a maintenance program with writers stopped, an already connected `store`, and a `sourceCodec` matching the source data:

```go
import (
    "context"
    "fmt"

    "github.com/Zany2/dtoken-go/contract"
    "github.com/Zany2/dtoken-go/migration"
)

func upgradeNamespace(ctx context.Context, store contract.Storage, sourceCodec contract.Codec) error {
    plan, err := migration.PrepareAuthUpgrade(ctx, store, migration.Options{
        AuthType:     "user",
        SourcePrefix: "dtoken:",
        TargetPrefix: "dtoken-next:",
    }, sourceCodec)
    if err != nil {
        return err
    }
    fmt.Printf("Prepared: %+v\n", plan.Summary())

    // Execute only after reviewing the summary during the stopped-writer window. 在停写窗口核对预检摘要后执行。
    result, err := plan.Apply(ctx)
    fmt.Printf("Copied: %+v\n", result)
    return err
}
```

All preparation entry points perform no writes. The returned Plan owns a snapshot and must not be copied or serialized. `Summary` reports actual namespaces, planned records, permanent/expiring counts, and records skipped because they disappeared or expired during preparation. Conversion entry points perform the checks described below; raw copying does not validate payload contents.

The example reads `dtoken:user:` and writes `dtoken-next:user:v2:`. Setting `TargetPrefix` to `dtoken:` instead writes `dtoken:user:v2:`, allowing the application to retain its configured KeyPrefix. Do not manually append the version to KeyPrefix or AuthType. Preparation rejects occupied destination data and excludes an existing `v2:` subtree at the source; current-version data is not nested into the destination again.

`Apply` rechecks that the entire destination namespace is empty. It uses `AtomicStorage.SetIfAbsent` where available, otherwise Get/Set. The ordinary fallback has a race, so stopping writers is required. Single-key atomic writes do not make the copy transactional.

## Data and expiration

The raw `Prepare` entry point includes all keys in the selected namespace: Token, Extra, Session, activity/renewal state, refresh credentials and reverse mappings, bans, colocated capabilities, and application suffixes. Strings, bytes, and basic numeric/boolean values are copied without conversion; byte buffers are cloned. Other value types fail preparation. This entry point does not invent AccessID, rewrite embedded paths/prefixes, repair corrupt records, or resolve ambiguous legacy ban keys. Applications must separately transform custom payloads containing old prefixes.

Each finite record captures a time anchor before reading TTL. Applying subtracts elapsed time and skips expired records instead of writing them with an unlimited lifetime. Only `TTLNoExpire` becomes permanent; `TTLNotFound` and zero TTL are skipped; other invalid negative TTLs fail. Relative TTL still depends on backend precision and transport/write latency, so exact absolute expiration across keys is not guaranteed.

## Converting token storage formats

To stage credential conversion in an unversioned destination without performing runtime cutover, use:

```go
plan, err := migration.PrepareTokenUpgrade(ctx, store, migration.Options{
    AuthType:     "user",
    SourcePrefix: "dtoken:",
    TargetPrefix: "dtoken-next:",
}, sourceCodec)
```

`sourceCodec` must match the source data. This entry point does not guess JSON, Base64, or other encodings. Check `err`, review `plan.Summary()`, then explicitly call `Apply`. Its unversioned staging result is not a current-runtime destination; use `PrepareAuthUpgrade` for deployment upgrades. Both conversion entry points share the following credential rules. Target applications must retain the same Codec for Session, Extra, refresh credentials, and capability data.

- Token primary records become `dtk:2:`. Existing AccessID, TerminalIndex, Revoked, and logical logout states are preserved; credentials without AccessID receive new identities.
- Non-nil inline Extra moves to `token-extra:<accessID>` with the access token's deadline. Empty maps remain distinct from nil if the source encoding preserved that distinction; information already omitted by a legacy encoder cannot be recovered. Existing separate Extra is checked for presence and decodability.
- Original bindings are checked using the engine's `refreshMatchesAccessRecord` rule. Matching anonymous refresh records receive the token's new AccessID; anonymous refresh records without a matching token receive independent identities to avoid attaching them to reused token values. Refresh records retain their own TTL, Extra, and TerminalExtra. Natural access expiry still permits refresh under existing rules.
- Preflight rejects foreign AuthType records, duplicate access lifecycle identities, invalid primary records, missing/corrupt required Extra, and reverse mappings that disagree with existing refresh records. Anonymous legacy matching remains limited by historical field precision; migration cannot reconstruct identities never stored by the source.
- Session checks cover decoding, account ownership, and basic fields only. Missing sessions are not rebuilt, stale terminals are not pruned, and terminal indices are unchanged. Bans, capabilities, and custom records are copied unchanged; this is not full semantic validation of the namespace.

The summary adds `UpgradedTokens`, `UpdatedRefreshTokens`, and `AddedTokenExtras`. `Records`, `Permanent`, and `Expiring` count planned destination writes, including new Extra records. Conversion performs no storage writes and returns no executable Plan on error. Conversion time is charged against captured deadlines. Already canonical credentials with complete identities do not have their snapshot payloads rewritten.

## Converting restrictions and ownership

To migrate credentials and restrictions together, use this preparation entry point, then review the summary and explicitly call `Apply` as above:

```go
plan, err := migration.PrepareAuthUpgrade(ctx, store, migration.Options{
    AuthType:     "user",
    SourcePrefix: "dtoken:",
    TargetPrefix: "dtoken-next:",
}, sourceCodec)
```

This entry point handles account, service, device-type, and concrete-device restrictions in the same snapshot without rescanning source storage, placing final keys in the `v2:` destination. Each restriction uses the current escaped key and explicit `loginId` and `kind`. Reasons, timestamps, service levels, and remaining TTL retain their meaning; `Apply` still skips expired records. `Summary.UpgradedRestrictions` counts changed business keys or ownership, excluding the uniformly added version segment. Canonical records are not re-encoded.

- Kind is determined by service/device payload fields and checked against any declared `kind`, not inferred solely from key prefixes. An account named `service:account`, for example, remains an account restriction.
- With an explicit `loginId`, the full identity must reconstruct either the current or legacy source key. Ownership can be recovered from payload fields and the complete key only when both `loginId` and `kind` are absent and the key contains no backslash.
- Ownerless backslash keys, inconsistent kinds/fields, empty owners, invalid fields, and corrupt payloads fail preflight. The tool does not guess whether a backslash is literal or escaped, or infer restriction owners from Session lists.
- Multiple old/current records resolving to one canonical key fail migration even when payloads look similar. They can have different remaining TTLs, levels, or read precedence, so no record or deadline is automatically selected or merged. Overlapping source keys for different identities are supported when their final destinations differ.
- Resolve ambiguity during the stopped-writer window using trusted business information to supply `loginId` and `kind` (`account`, `service`, or `device`), preserving actual service/device fields, the source Codec, and remaining TTL. For duplicate destinations, explicitly choose the intended business rule and resolve the source records before preparing again. Do not discard unknown restrictions merely to bypass errors; source repair is not performed automatically.

Any restriction-preflight error returns no Plan. Earlier credential conversion remains in memory only and causes no destination writes. `Prepare` and `PrepareTokenUpgrade` retain their raw restriction-copy behavior; use `PrepareAuthUpgrade` when restriction normalization is required.

## Failure and cutover

1. Any occupied destination key causes `ErrDestinationNotEmpty`; existing data is not intentionally overwritten. Preparation never writes, but application failures can leave partial destination data.
2. `Result.Written` counts acknowledged writes only; `Expired` counts snapshot records that expired before writing. A failed write may have reached storage before a network error, so inspect the backend.
3. A Plan is single-use after the first application attempt, including cancellation or failure. Reuse returns `ErrPlanUsed`. There is no automatic rollback or cleanup; explicitly handle partial destination data before preparing again.
4. After `PrepareAuthUpgrade` migration and validation, configure every application with the target prefix, for example `NewBuilder().KeyPrefix("dtoken-next:").AuthType("user")`. Keep existing settings when migrating under the same prefix. Manager, Nonce, OAuth2, Ticket, and ShortKey automatically append `v2:`; upgrade their versions together and keep matching prefixes and auth types. Retain the other settings and Codec. Credentials expired during copying remain expired. Requiring fresh login does not justify discarding restrictions or persistent account data.
5. Verify login checks, refresh, logout, bans, Session, and capability behavior before separately deleting the old namespace. Once the destination accepts writes, source and destination diverge; switching back is not a lossless rollback.

With the same configured prefix, the new namespace is a `v2:` subtree beneath the old boundary. Later source cleanup must exclude that subtree; do not clear the entire old prefix. Preparation also treats the source's `v2:` subtree as reserved and excludes it. If an old application used that custom suffix, review its data separately.

The configured default KeyPrefix remains `dtoken:`, but runtime storage now includes the fixed `v2:` segment. Old namespaces are not automatically read or deleted. Runtime access credentials reject legacy payloads and compact records missing AuthType or AccessID with ErrInvalidToken, without attempting Codec decoding. PrepareAuthUpgrade supplies the required identity and compact format. Runtime restrictions no longer fall back to legacy suffixes or infer missing ownership. PrepareAuthUpgrade converts their keys and supplies loginId and kind. Copying old restriction bytes into the new namespace is insufficient, even when the old and canonical suffixes happen to be identical. No actual migration has been executed. Same-prefix cutover, namespace isolation, capability consumption, and post-migration lifecycle unit tests passed in the non-Redis root-module run on 2026-11-26. This does not establish a successful deployment migration or live Redis rehearsal; see the [acceptance record (Chinese)](performance-acceptance_zh.md).
