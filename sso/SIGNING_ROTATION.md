# SSO signing-key rotation

[中文](SIGNING_ROTATION_zh.md) · [Configuration](CONFIGURATION.md) · [Release checklist](RELEASE_CHECKLIST.md)

`SecretKey` remains the only outbound signing key. Optional `VerificationKey` accepts one additional incoming key, for at most two keys. Empty retains existing single-key behavior. HMAC-SHA256, canonicalization, parameters and routes remain unchanged; there is no key ID. The companion can hold the next key during preparation or the previous key after switching.

## Configuration and trust scopes

| Location | Verification scope | Outbound behavior |
| --- | --- | --- |
| `Client.SecretKey` / `Client.VerificationKey` | Center verifies that client's requests using current registration on each request | Center signs logout notifications only with that registration's `SecretKey` |
| `ServerOptions.SecretKey` / `ServerOptions.VerificationKey` | Clients without a dedicated group and applicable requests without a client identity | Fallback signing uses only its `SecretKey` |
| `ClientConfig.SecretKey` / `ClientConfig.VerificationKey` | Application verifies center logout callbacks | Application signs requests only with its `SecretKey` |

A dedicated client group entirely replaces the fallback group. Verification failure never tries global primary/companion keys or another client's keys. Anonymous selection retains its existing rules. Registration lookup failures or removal never reuse cached keys or bypass errors through fallback.

A companion requires a nonempty `SecretKey` in the same configuration. A client cannot declare only `VerificationKey` and inherit the global primary. Checked HTTP/ClientApp constructors also reject companions when `CheckSign` is false; legacy constructors still disable verification when `CheckSign` is false. Diagnostics do not echo keys. Identical primary and companion values are permitted but add no effect.

The variables below must come from application secret management:

```go
registration.SecretKey = oldKey
registration.VerificationKey = nextKey
if err := server.RegisterClientChecked(registration); err != nil {
    return err
}

config := sso.DefaultClientConfig()
config.ClientID = "orders"
config.ServerURL = "https://login.example.com"
config.SecretKey = oldKey
config.VerificationKey = nextKey
// Set the exchange secret, replay protection and other application settings before NewClientAppChecked.
```

## Three-stage rollout

| Stage | Target configuration on both sides | Before proceeding |
| --- | --- | --- |
| Prepare | `SecretKey=old`, `VerificationKey=new` | Every receiving replica understands and has received the new field; senders still use old |
| Switch | `SecretKey=new`, `VerificationKey=old` | All senders have switched; either receiver configuration accepts both |
| Retire | `SecretKey=new`, `VerificationKey=""` | Old senders are drained, the overlap window has ended, and old-request rejection/relogin is acceptable |

Deploy code that understands the field before preparation. Old code may ignore stored companion fields but cannot verify new signatures. Shared-storage registration overwrites the complete record; old replicas starting up can overwrite the new key group. Keep registration configuration consistent or coordinate one registration writer in the application.

ClientApp retains construction-time configuration. Do not mutate live objects concurrently; create and switch application references to replacements. Replace HTTPServer instances to switch fallback configuration. Dedicated registrations can be updated with `RegisterClientChecked`; requests and every notification retry read the current record.

## Window and replay behavior

Companion keys **do not expire automatically**. The key count is bounded; deployment controls retirement time. Preserve replay storage, namespace and window across rotation. Signing the same client/action/nonce with another key is still replay, and rotation must not clear nonce history. Invalid or retired signatures are rejected before nonce consumption.

To drain previously signed requests, account for request lifetimes and allowed clock skew after the last old sender stops. Current future-timestamp tolerance may require up to `2 × MaxAge`. Also account for pending browser flows. Without replay protection, ordinary signed requests have no uniform expiry, so choose an explicit cutoff instead of retaining old keys indefinitely. If compromise is suspected, remove the old key immediately and accept failures of old requests.

Notification retries always use the current registration's primary key and a fresh nonce. Rotation does not change retry counts, timeouts or strict/best-effort policy. Protocol key rotation does not rotate `ClientSecret`, cookie keys or PKCE proofs. Cookie rotation is configured separately as described below.

## Browser session and state cookie rotation

`CookieOptions.VerificationKey` adds one explicit verification companion to `CookieOptions.SecretKey`. `SetLoginIDCookie` signs only with the primary; `LoginIDFromCookie` accepts either key while enforcing the original signed expiry. The existing `v1` format and single-key defaults remain unchanged. Verification does not refresh a cookie, extend its expiry, write storage or recreate a session. A verification-only configuration cannot issue or authenticate cookies; the checked HTTP constructor rejects a companion without an effective primary. Browser state still requires its own explicit primary.

Use the field in `HTTPOptions.Cookie` for the default center resolver, in application-owned session `CookieOptions`, and in `BrowserClientOptions.StateCookie`. A custom center resolver must explicitly use the cookie options when calling `LoginIDFromCookie`. The protocol's `ServerOptions.VerificationKey` is never inherited by cookies. Existing primary fallback from `HTTPOptions.Cookie.SecretKey` to `ServerOptions.SecretKey` remains supported, but explicit independent cookie keys avoid coupling two rotation schedules. Keep independent keys for different applications and environments.

| Phase | All replicas of the same service | Required outcome |
| --- | --- | --- |
| Prepare | Cookie primary=old, companion=new | Deploy companion-aware readers everywhere before any writer switches |
| Switch | Cookie primary=new, companion=old | Both generations accept old/new session and pending-state cookies; new cookies use new |
| Retire | Cookie primary=new, companion empty | Old cookies fail verification; any remaining old sessions/flows require login again |

Recreate the resolver, BrowserClient or application with the new configuration; do not mutate live shared instances. Keep cookie names, paths, domains, storage namespaces and backing storage stable. Rotation does not transfer in-memory state across processes. The deployment example uses the same per-service cookie pair for local session and state cookies; center and each application rotate independently. Pending PKCE proofs and state records stay in shared storage, and consumed state remains consumed across key changes. Server-side session revocation and absolute deadlines still apply after cookie verification.

Companions do not expire automatically. For a planned retirement, wait from the last old-key issuance for the longest possible lifetime of an old session or state cookie, including clock-skew allowance and any longer historic configuration. Switching configuration does not renew existing browser cookies, so removing the old key earlier deliberately forces affected users to log in again. If a key may be compromised, remove it immediately instead of retaining it for continuity, and revoke the affected server-side sessions through application policy. Never treat cookie-key retirement as a replacement for server-side logout.

For the deployment example, set `SSO_COOKIE_KEY` to the primary and optional `SSO_COOKIE_VERIFICATION_KEY` to the companion in each service's secret configuration. Both nonempty values must be at least 32 bytes. This is independent of `SSO_SIGNING_KEY` / `SSO_VERIFICATION_KEY`. The local launcher clears the optional cookie companion to avoid inheriting unrelated settings.

Cookie regression sources in [cookie_rotation_test.go](cookie_rotation_test.go) cover preparation/cutover/retirement, primary-only issuance, expiry/tampering, configuration and PKCE state continuity. The [deployment regression](../examples/sso_deployment/cookie_rotation_test.go) covers actual center-session reuse, old-replica initiation/new-replica completion, session retirement and account logout. These sources have not been executed. Focused command: `go test ./sso/... ./examples/sso_deployment/... -run 'TestCookieRotation|TestDeploymentCookieRotation' -count=1`. Real Redis rolling deployment remains a separate acceptance step.

## Deployment example

Centers use `secretKey` and optional `verificationKey` in each `SSO_CLIENTS_JSON` registration. Applications use `SSO_SIGNING_KEY` and optional `SSO_VERIFICATION_KEY`. The example requires at least 32 bytes for nonempty keys. The development script clears the companion to avoid inheriting rotation settings from the environment. Never place keys in URLs, logs or error responses.

## Verification

Six groups in `signing_keys_test.go` cover trust scopes, switching/retirement and nonce continuity for six actions, callback replay, all four credential modes, notification retries and configuration diagnostics. Deployment tests add actual login/logout with differing active keys and configuration checks. Only static inspection and formatting were performed; these commands have not run:

```powershell
go test ./sso/... ./examples/sso_deployment/... -run 'TestSigningRotation|TestDeploymentSigningRotation' -count=1
go test -race ./sso/... ./examples/sso_deployment/... -run 'TestSigningRotation|TestDeploymentSigningRotation' -count=1
```

Memory storage and in-process transport do not replace real Redis, rolling-replica or browser acceptance.
