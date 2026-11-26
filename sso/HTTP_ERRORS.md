# Center HTTP errors and diagnostics

[中文](HTTP_ERRORS_zh.md) · [Configuration](CONFIGURATION.md) · [Testing](TESTING.md)

The six center endpoints (`authorize`, `token`, `introspect`, `userinfo`, `revoke`, `logout`) use one error boundary. Known protocol errors use fixed classification messages; unknown errors use `SSO operation failed`. Wrapped storage, serialization and application-hook diagnostics are not copied into public responses. JSON retains `code`, `message` and optional `data`; errors retain `no-store`, `no-cache` and `nosniff` headers.

## Status and compatibility

| Failure | HTTP status |
| --- | --- |
| Invalid parameters, bindings/proofs, one-time credentials | 400 |
| Invalid signature/client credentials, expired request, used nonce | 401 |
| Explicit policy denial at issue/exchange | 403 |
| Missing client registration | 404 |
| Unsupported method | 405 |
| Request exceeds configured limits | 413 |
| Policy evaluation unavailable | 503 |
| Initialization/configuration, storage, cancellation, unexpected hook error | 500 |

Initialization failures return 500 through direct handlers, registered mux routes and `Handler()`. Configuration/storage classifications take precedence over credential-like causes they wrap. Policy infrastructure failures remain 503 even when the cause resembles a denial. Endpoint-specific overrides remain, such as 401 for a logout subject mismatch. Diagnostic text and some fixed authentication messages change: use HTTP status/business code, not raw message matching.

Ordinary inactive introspection still returns HTTP 200 with `active:false`; inactive userinfo returns 401. Storage/activity-hook failures return errors instead of inactivity. A resolver returning only `(string, bool)` cannot carry an error: `false` still means unauthenticated, and application-owned resolver diagnostics remain the application's responsibility.

## Local observation

Set `HTTPOptions.OnHTTPError` to `func(r *http.Request, status int, err error)`. The `httpserver` facade exports the same `HTTPErrorFunc`. The callback runs synchronously once before each center-generated failure response, with the operation's error and existing error chain. It receives no ResponseWriter and cannot replace the response.

```go
options.OnHTTPError = func(_ *http.Request, status int, err error) {
    category := "protocol"
    if errors.Is(err, sso.ErrStorageUnavailable) {
        category = "storage"
    }
    log.Printf("SSO rejection status=%d category=%s", status, category)
}
```

The snippet needs `errors`, `net/http`, `log` and `sso` imports. A complete public-API example is in [example_http_error_test.go](example_http_error_test.go). The deployment center logs only 5xx status values and retains its logout-stage observer.

Treat requests/errors as read-only. Raw errors, URLs, forms, headers and cookies may contain secrets; never log them verbatim. Use fixed categories and trusted application correlation. Observers must return promptly, handle concurrent requests and manage their own failures; the library neither recovers observer panics nor starts background tasks. Error chains are passed through unchanged, without reconstructing causes already converted to strings by an adapter or caller.

Success responses, login redirects, inactive introspection, mux-generated 404s, direct core API calls and client-side BrowserClient/RemoteAuth/logout-callback handlers do not invoke this center observer. `LogoutAccount` keeps returning its own result/cause. Checked-constructor failures return directly before a handler exists; use the returned error for startup diagnostics.

Authenticated HTTP logout failures invoke `OnLogoutResult` first, then `OnHTTPError` once. Public stage messages and reports remain unchanged. Best-effort HTTP 200 with failed notifications invokes only `OnLogoutResult`; absence of `OnHTTPError` is not proof of delivery success.

## Lifecycle and verification

### Response encoding

SSO JSON writers encode the complete envelope before committing the HTTP status. Unsupported extension values (for example NaN, channels, cyclic maps or failing custom JSON marshalers) produce HTTP 500 with `{"code":500,"message":"SSO response encoding failed"}`. No partial identity or extension data is sent. Successful JSON retains HTML escaping, integer precision and its trailing newline; it is encoded only once.

Center success responses use `OnHTTPError` to report an encoding failure once, before headers are committed. The error matches `ErrSerializeFailed` and retains the JSON error chain for local diagnostics. Client-side writers use the same fixed fallback without invoking the center observer. Authenticated logout errors keep their existing lifecycle observer and stage response, without duplicate observation.

Encoding happens after the underlying operation: failure does not restore a consumed Ticket/Code/nonce, undo a revoke/logout, or remove already-completed callback registrations. Inspection encoding failure does not itself revoke credentials. A fresh signed retry is required after nonce consumption; consumed one-time credentials require a new login. JSON must be buffered in memory, so applications should keep extension data reasonably small. This change adds no response-size configuration or streaming support.

The guarantee applies before HTTP commitment. If the connection breaks during the body write, the server cannot replace the status or guarantee delivery; it does not append a second error response or repeat the operation. Such transport failures are not reported as encoding failures through `OnHTTPError`. Previously written headers/body from application hooks or middleware also cannot be undone. Custom marshalers must terminate normally: their panics and unbounded work are not recovered by this encoding boundary.

[http_response_test.go](http_response_test.go) covers invalid JSON values, custom errors, full-response fallback, four credential modes, exchange and nonce consumption, logout report privacy, successful encoding fidelity and partial-write failure. These cases have not run. Focused command: `go test ./sso/... -run 'TestHTTPResponse' -count=1`.

Reporting adds no retry, rollback or restoration. Rejection before a successful nonce claim leaves it available; a claimed nonce stays consumed after a later hook failure, requiring a fresh signed retry. Policy failure before Ticket/Code exchange preserves the credential; enrichment failure after exchange keeps it consumed. A failed deletion can leave a credential valid. Drive recovery from operation results, not compensation in the observer.

[http_errors_test.go](http_errors_test.go) and [http_errors_lifecycle_test.go](http_errors_lifecycle_test.go) cover six endpoints, classification, three initialization paths, replay-storage recovery, four credential modes, exchange stages and observation counts. Existing logout-stage tests cover strict/best-effort observations. Sources are formatted and statically reviewed, **not executed**. Run from the repository root when ready:

```powershell
go test ./sso/... -run 'TestHTTPError|TestLogoutNotificationHTTPStages|ExampleHTTPErrorFunc' -count=1
go test ./sso/... ./examples/sso_deployment/... -count=1
```

Release, race and Redis checks remain in [the release checklist](RELEASE_CHECKLIST.md).
