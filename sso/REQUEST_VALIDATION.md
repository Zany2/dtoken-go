# SSO request parameter consistency

[中文](REQUEST_VALIDATION_zh.md) · [Testing](TESTING.md)

The center, ClientApp and browser integration treat configured protocol fields as scalar values, preventing duplicate parameters and mixed credentials from being interpreted differently across integration layers. Format validation does not replace signatures, client secrets, PKCE, credential binding or application authorization.

## Scalar fields and extensions

Every configured `ParamNames` field may appear at most once, including client identity/secret, mode, redirect, callback, back, scope, login identity, credentials and signing/replay metadata. Identical duplicates are rejected too. Custom names are checked as configured; their default names do not become aliases.

POST query and form-body values retain standard-library merging. A protocol field present in both is a duplicate. This does not make every POST parameter body-only or change the existing prohibition on exchange/inspection PKCE verifiers in URLs. Client exchange secrets should be sent through POST; signing keys are never sent as protocol parameters.

A single empty string remains subject to existing field rules. Missing credentials, empty callbacks and empty modes retain endpoint-specific handling. For direct `url.Values` input, present protocol fields with zero-length value slices are rejected.

Send multiple scopes as one value, such as `scope=profile%20email`, rather than repeating `scope`. Application extensions outside configured protocol names can remain multi-valued, such as `tag=a&tag=b`. The low-level Signer retains its multi-value ordering, encoding and verification algorithm: a valid signature does not prove that the protocol input has a valid shape.

## Credentials and modes

| Mode | Permitted credential field | Operations |
| --- | --- | --- |
| `ticket` | `ticket` | Exchange, inspect, userinfo, revoke |
| `oauth2` | `code` | Exchange, inspect, userinfo, revoke |
| `shared_token` | `tokenValue` | Inspect, userinfo, revoke |
| `remote_session` | `sessionId` | Inspect, userinfo, revoke |

These operations reject credential fields belonging to another mode, including empty foreign fields. For example, `mode=ticket&ticket=...&tokenValue=` returns `ErrInvalidRequest`. With no mode, `/token` retains code inference when a nonempty code is supplied, otherwise Ticket; the other three endpoints default to Ticket. Missing selected credentials retain their existing downstream error handling.

An empty `CredentialRequest.Mode` uses ClientApp's configured mode; it does not infer mode from another struct field. Populate only the selected credential. The SDK rejects mixed credentials before transport. Standard helpers overwrite fields owned by their configuration before validating the final request. Replay-enabled calls still generate fresh action/timestamp/nonce metadata before validation and signing.

## Ordering and effects

- All six center HTTP endpoints validate format before signatures and nonce consumption. Duplicate protocol fields and mixed credentials return HTTP 400 / `ErrInvalidRequest`. PKCE fields retain their existing `ErrInvalidPKCE` classification and validation order; unknown or unsupported modes return `ErrModeUnsupported`.
- Requests rejected by these duplicate-field or mixed-credential checks cannot invoke identity resolvers, authorization/state hooks, credential consumption/revocation or logout delivery, and create no callback bindings.
- Client `VerifyLogoutCallback` rejects duplicate protocol fields before signature verification and nonce consumption; its handler does not invoke the application logout hook.
- Browser completion preserves existing state/credential error classifications and validates other protocol fields before consuming state. The original valid callback remains eligible unless expired or consumed by another request.
- Format rejection does not claim a nonce. Corrected and re-signed input may use that unconsumed nonce; ordinary SDK calls generate a fresh one. Accepted requests keep existing replay protection.

Trusted core Server calls use structured arguments and do not pass through HTTP validation. Custom HTTP adapters should apply equivalent scalar and mode checks rather than relying on signature verification alone.

## Compatibility and verification

This deliberately tightens ambiguous input. Existing single-value protocol requests are unchanged. Callers relying on first-value selection, submitting every credential field, or repeating scope must adjust. Check proxies and form components for query parameters duplicating body fields. Turning off signing does not disable format validation.

[request_validation_test.go](request_validation_test.go) adds eight parameterized test groups: all protocol fields, four modes, nonce ordering across six actions, center lifecycle, SDK/callback behavior, browser state preservation, authorize/logout side effects, and signed query/body duplicates with custom names and replay protection.

Only formatting and static inspection were performed; builds, tests and race detection were not run. Run manually from the repository root:

```powershell
go test ./sso/... -run TestRequestValidation -count=1
go test -race ./sso/... -run TestRequestValidation -count=1
```

Full module and real deployment acceptance remain in the [release checklist](RELEASE_CHECKLIST.md).
