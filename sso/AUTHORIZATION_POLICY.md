# Dynamic SSO authorization policy

`HTTPOptions.AuthorizationPolicy` checks whether the current user may access a client application and which scopes may be granted at issuance, exchange, and inspection. It connects to application-owned membership, tenant access, account state, or consent records without adding a permission database to SSO.

This is an optional HTTP-center policy. Nil preserves existing behavior. Trusted low-level `Server.Generate*`, `Consume*`, and `Validate*` APIs do not automatically run it; custom public flows using those APIs must enforce equivalent authorization themselves.

## Integration

```go
options := sso.DefaultHTTPOptions()
// Configure clients, signing keys, and a trusted center session resolver.
options.LoginIDResolver = resolveCenterLogin
options.AuthorizationPolicy = func(ctx context.Context, request sso.AuthorizationRequest) (sso.AuthorizationDecision, error) {
    // Application function: resolve current membership using trusted identities.
    membership, err := loadMembership(ctx, request.ClientID, request.LoginID)
    if err != nil {
        return sso.AuthorizationDecision{}, err
    }
    if !membership.Enabled {
        return sso.AuthorizationDecision{}, nil
    }
    granted := make([]string, 0, len(request.Scopes))
    for _, scope := range request.Scopes {
        if slices.Contains(membership.Scopes, scope) {
            granted = append(granted, scope)
        }
    }
    return sso.AuthorizationDecision{Allowed: true, Scopes: granted}, nil
}
handler := sso.NewHTTPServer(server, options)
```

`resolveCenterLogin` and `loadMembership` are application functions. Determine tenant membership from trusted client/user mappings, not browser-supplied tenant identifiers. The policy receives Context and validated identity, mode, phase, and scopes, without the raw HTTP request, secrets, credentials, PKCE verifier, or business Extra.

See [example_authorization_policy_test.go](example_authorization_policy_test.go) for a standalone integration example. The `sso/httpserver` facade exposes the same option.

## Stages

| Phase | HTTP entry | Trusted input | Effect of allowing access |
| --- | --- | --- | --- |
| `AuthorizationIssue` | authorize, all four modes | Center identity, registered client, requested scopes within the static allow-list | Persist the reduced grant in the new credential |
| `AuthorizationExchange` | token, Ticket/Code | Stored credential after client authentication, expiry, ownership, redirect, and PKCE checks | Consume once and return the reduced grant |
| `AuthorizationInspect` | introspect/userinfo, all modes | Authenticated client and active credential after the optional application-state hook | Return the current grant without rewriting storage |

Client mode/scope restrictions, callback validation, signatures, and PKCE remain in force. Policy cannot repair an invalid request or reactivate expired, revoked, or consumed credentials. Forged `loginId` or extra exchange `scope` parameters do not determine policy identity or grant limits.

Replay protection claims the request nonce before policy evaluation. A denied or failed operation needs a freshly signed request; `ClientApp` generates new nonces as usual. Browser callbacks still consume their flow state once, so failures require restarting login.

## Decisions

- The zero decision denies access. Scopes cannot override `Allowed: false`.
- `Allowed: true, Scopes: nil` preserves the complete input grant.
- `Allowed: true, Scopes: []string{}` permits identity access with no scopes.
- A non-nil subset reduces the grant. Matching is exact with no wildcard expansion. Additional scopes produce an error rather than silently expanding access.

Explicit selections preserve input order and remove duplicates. Policy inputs and resulting scopes are copied; changing the input slice cannot enlarge the original grant. Policy implementations must support concurrent calls and must not asynchronously mutate input or returned slices.

Issuance reduction is permanent for that credential: a later stage cannot restore scopes omitted at issuance. Inspection reduction affects only the current response, so a restored policy may return scopes that remain in the original stored grant. A consumed ticket/code cannot be re-exchanged after a policy change.

An allowed empty grant still represents a valid identity. Applications must enforce required scopes with `RemoteAuth.RequiredScopes` or their own checks. Scopes do not replace resource ownership checks. Extra metadata is not filtered by scope automatically; applications must manage its contents and must not use it as an alternative authorization channel.

## Failures and lifecycle

| Decision | Authorize/token | Introspect | Userinfo |
| --- | --- | --- | --- |
| Explicit denial | 403, `ErrAuthorizationDenied` | 200, inactive without identity | 401 without identity |
| Evaluation error or canceled Context | 503, `ErrAuthorizationPolicyUnavailable` | 503 | 503 |
| Expanded grant or consumed payload inconsistent with evaluated input | 500, `ErrAuthorizationDecision` | 500 for invalid grant | 500 for invalid grant |

Errors take precedence over Allowed. Returning `ErrAuthorizationDenied` as an error is still an evaluation failure: return `Allowed: false, nil` for a business denial. Public policy errors contain fixed classifications, not underlying diagnostic text. Record redacted diagnostics inside the policy implementation when needed; never convert a lookup outage into an allow decision.

Issuance evaluation precedes `AuthorizationExtra` and credential generation. Denied issuance creates neither credentials nor callback bindings. Exchange evaluation precedes consumption: normal denial, evaluation failure, and invalid-scope failures leave the credential intact. Consumption still revalidates state, and the consumed identity and original scopes must match the evaluated input before applying the grant.

`EnrichExchangeResult` runs after policy and consumption, sees the reduced grant, and still cannot change identity or scopes. Its failure leaves the credential consumed. Exchange-stage callback registration occurs only on success; authorization-stage bindings are not removed on later policy failure.

`CheckCredentialActive` runs before inspection policy. A false result returns inactive without policy evaluation; lookup errors remain errors. Policies should only evaluate permissions, not recursively invoke the same HTTP exchange flow or mutate credentials/callback bindings. Exchanges registering logout callbacks retain the existing registration mutex during policy evaluation.

## Remote authentication and logout

`RemoteAuth` introspects every protected request and therefore observes current policy:

| Policy outcome | Default application response |
| --- | --- |
| Application access denied | 401 |
| Allowed but missing a required scope after reduction | 403 |
| Policy outage, invalid result, or timeout | 503 |
| Allowed with required scopes | Business handler runs |

Denial does not delete credentials, revoke center login, or clear existing local application sessions. When permissions return, an otherwise active credential may pass again. Use account revocation, individual credential revocation, or existing center logout for permanent invalidation. Revoked credentials never reach the policy again.

Local sessions established through one-time exchange retain a login-time authorization snapshot. Later policy changes do not automatically delete them; use single logout, local checks, or an appropriate remote session design. Changes do not undo requests already executing business logic.

## Deployment boundaries

Configure the policy on every center replica: a replica with no policy retains previous default behavior. Policy data, caches, and consistency belong to the application. This module adds no evaluation cache, storage interface, dependency, permission-management UI, or new HTTP endpoint.

Honor Context and apply suitable database/network timeouts. The center rejects an allow result arriving after cancellation but does not start goroutines to interrupt uncooperative policies. Evaluation and later actions are not a distributed transaction; a subsequent permission change does not retroactively undo completed issuance or business operations.

## Verification

Regression sources cover all four modes, reduced and expanded scopes, input isolation, ordinary/atomic consumption, preserved credentials on failure, PKCE/client authentication order, operation without SLO, remote authentication, account revocation, signed replay-protected browser login, and canceled policy rejection. Formatting and static checks were performed; tests, builds, examples, and real storage integration were not run. See [testing](TESTING.md) for manual commands.
