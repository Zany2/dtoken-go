# Single logout results, concurrency, and retries

SSO logout follows: notify clients, revoke application-owned center sessions, invalidate account SSO credentials, clear client bindings, and clear the center cookie. Notifications provide per-client results, bounded retries, an instance-wide concurrency limit, and an outcome observer. They do not implement background delivery or a persistent retry queue.

## Configuration

Every attempt rechecks its deadline after registration lookup, transport return and acknowledgement reading. Custom transports/readers returning late success are classified as `timeout` failures and remain subject to the attempt limit; parent cancellation is `canceled` and prevents further retries. Response bodies are still closed. Canceled application logout hooks do not acknowledge success, and consumed nonces are not restored.

Browser center logout checks cancellation after center-session revocation, credential revocation and binding cleanup. Results retain the stage where cancellation occurred without clearing the center cookie or reporting success. Completed effects are not rolled back; authorized recovery can use [LogoutAccount](ACCOUNT_LOGOUT.md).

Extend existing `HTTPOptions`:

```go
options.EnableSLO = true
options.LogoutCallbackBestEffort = false
options.LogoutCallbackConcurrency = 4
options.LogoutCallbackMaxAttempts = 3
options.LogoutCallbackRetryInterval = 200 * time.Millisecond
options.LogoutCallbackTimeout = 3 * time.Second
options.OnLogoutResult = func(r *http.Request, result sso.LogoutResult) {
    if result.Cause != nil {
        logger.Printf("logout stopped at stage=%s", result.Stage)
    }
    if result.Notifications != nil {
        for _, target := range result.Notifications.Clients {
            if target.Status == sso.LogoutCallbackFailed || target.Status == sso.LogoutCallbackCanceled {
                logger.Printf("client=%q attempts=%d reason=%s", target.ClientID, target.Attempts, target.Reason)
            }
        }
    }
}
handler := sso.NewHTTPServer(server, options)
```

| Option | Default | Meaning |
| --- | --- | --- |
| `LogoutCallbackConcurrency` | 8 | Total concurrent delivery attempts across requests on one `HTTPServer` |
| `LogoutCallbackMaxAttempts` | 1 | Maximum attempts per target per logout request, including the first |
| `LogoutCallbackRetryInterval` | 100 ms | Fixed delay before retrying temporary failures |
| `LogoutCallbackTimeout` | 3 s | Per-attempt timeout including registration lookup, HTTP, and acknowledgement reading |

Nonpositive values use defaults. The default remains one delivery, so upgrading does not enable extra attempts. Queue waits are outside the per-attempt timeout; configure a parent request Context deadline for a total budget. A custom `LogoutHTTPClient` timeout also applies. Retry delays release shared delivery slots. Total time can include queueing, multiple timeouts, and retry delays. One worker handles each target sequentially; successful targets are not retried within the same invocation.

The concurrency bound applies to one `HTTPServer` instance. Reuse handlers and account for replica counts when planning overall load. It does not replace ingress rate limiting. Custom storage and transports should honor Context cancellation.

## Retry policy

| Outcome | Retry within this invocation? |
| --- | --- |
| HTTP transport/read failure or attempt timeout | Yes, within the attempt limit |
| Client registration lookup failure | Yes, within the attempt limit |
| HTTP 408, 429, or 500–599 | Yes, within the attempt limit |
| Other non-2xx status | No |
| Disallowed callback, missing signing key, invalid configuration | No |
| Oversized/malformed 2xx acknowledgement, missing integer `code`, or nonzero `code` | No |
| Parent cancellation or deadline | Stop waits and new attempts |

Acknowledgements retain compatibility with empty 2xx responses, plain `ok`, and JSON `{"code":0}`, bounded to 64 KiB. A business rejection inside HTTP 200 is not assumed temporary. Retry scheduling uses the configured fixed interval and does not interpret `Retry-After`.

Every attempt re-reads registration, rechecks callback permissions, selects the current signing key, and generates a fresh signature. Replay protection adds a new nonce per attempt; do not resend old signed bodies. A client may have logged out before its response was lost, so its logout handler must tolerate repeated notifications.

`ClientApp.LogoutCallbackHandler` returns fixed public error text: verification failures use the HTTP status phrase while preserving the existing status classification; application hook failures return HTTP 500 with `logout callback failed`. Storage diagnostics and hook errors are not echoed in the response. `VerifyLogoutCallback` still returns diagnostic errors for local handling, including `errors.Is` matching of storage causes; calling it consumes the nonce on success, so do not call it before invoking the built-in handler on the same callback. Record application hook diagnostics inside the hook before returning its error. A failed hook keeps the accepted nonce consumed and requires a freshly signed notification to retry. Success is acknowledged only after the hook succeeds and the request remains active.

Removed registrations become `skipped / unregistered`; legacy bindings without a callback become `skipped / no_callback`. Skipping is not confirmation of local session revocation.

## HTTP results and observation

Success retains `code:0` and `data.result:"ok"`; `data` now contains `LogoutResult`. Failures after signature and trusted-subject validation also include the stage and available notification report:

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "result": "ok",
    "stage": "complete",
    "notifications": {
      "complete": false,
      "clients": [
        {"clientId": "app-a", "status": "delivered", "attempts": 1, "httpStatus": 200},
        {"clientId": "app-b", "status": "failed", "attempts": 3, "httpStatus": 503, "reason": "http_status"}
      ]
    }
  }
}
```

This is best-effort center completion with `app-b` still unconfirmed. Consumers should decode into a struct or an object supporting nested data; clients decoding all of `data` into `map[string]string` must adjust.

- `stage` is `notifications`, `center_logout`, `credentials`, `cleanup`, or `complete`. Reaching a later stage means preceding calls returned, not that a cross-storage transaction committed.
- Per-target `status` is `delivered`, `failed`, `skipped`, or `canceled`, ordered by client ID.
- `attempts` includes registration/configuration validation but excludes queue waits. `httpStatus` describes the final attempt's response and is omitted when there was none.
- `notifications.complete` means no failed/canceled targets, permitting deliberate skips. It does not prove every local session has been revoked.
- SLO-disabled responses omit `notifications`. Authentication failures disclose no account notification details and do not invoke the observer.
- `OnLogoutResult` runs once synchronously before the final response. Trusted applications can read `LoginID`, overall `Cause`, and per-target `Cause`; these are not serialized. Public failure messages identify the stage without exposing callback URLs, response bodies, or transport errors.
- Observers must not write the HTTP response, block, or mutate shared configuration. Result slices are copied before observation to isolate response data. Best-effort completion can have a nil overall cause while retaining individual failure causes.

## Lifecycle boundaries

Strict mode stops before center logout if a target ultimately fails, retaining bindings and the center cookie for another request. Already successful client callbacks are not rolled back.

Best-effort mode continues despite individual delivery failures and retains those failures in its report. Failure to load bindings or parent cancellation stops the remaining lifecycle. Application center-logout, credential invalidation, and cleanup errors are not ignored.

A new logout request reads current bindings and starts a new notification round, including targets successful in earlier rounds. There is no cross-request progress cache. A storage failure after center-session revocation can leave the original browser unable to authenticate another attempt. Use the observer to diagnose the stage and perform authorized application recovery; refreshing the URL cannot recover every possible failure. Completed logout effects are not rolled back.

Callbacks remain account-scoped rather than versioned to a specific login. Applications coordinate concurrent fresh logins and logout as needed. Do not persist old reports as long-lived retry jobs: old notifications could affect newer sessions.

## Trusted direct composition

`HTTPServer.NotifyLogoutClients(r, loginID)` returns `(*LogoutNotificationReport, error)` for application-owned flows. It reports real notification failures regardless of `LogoutCallbackBestEffort` and preserves `errors.Is` matching. It only sends notifications; it does not revoke center sessions, clear bindings/cookies, or invoke `OnLogoutResult`.

It does not authenticate an HTTP subject. Callers must authorize `loginID` and must not forward arbitrary submitted account identifiers. Use the existing `HandleLogout` route for the full standard lifecycle.

For administrative forced logout, [LogoutAccount](ACCOUNT_LOGOUT.md) composes center-session and credential revocation before notifications and binding cleanup. Delivery failure returns partial progress and retains bindings for authorized retry; it writes no browser response and invokes no `OnLogoutResult`. `HandleLogout` retains its notification-first strict/best-effort behavior.

The standard-library and Gin center examples explicitly select concurrency 4, one attempt, and a safe summary observer. Increase the attempt limit to enable in-request retries. See the [testing guide](./TESTING.md) for verification scope and commands.
