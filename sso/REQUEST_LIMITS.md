# SSO HTTP request limits

[中文](REQUEST_LIMITS_zh.md) · [Request consistency](REQUEST_VALIDATION.md) · [Configuration](CONFIGURATION.md)

Center endpoints, single-logout callbacks and browser login callbacks bound inbound parameters before signatures, nonce/state consumption and application hooks. Signed and unsigned requests obey the same limits. Trusted direct `Server` calls do not pass through HTTP limits.

## Configuration and coverage

`HTTPOptions.RequestLimits` controls requests received by the center; `ClientConfig.RequestLimits` controls callbacks received by the application. Configure them independently. `httpserver.Options` exposes the same field, and the facade also offers the `RequestLimits` alias and `DefaultRequestLimits()`.

| Field | Default | Measurement |
| --- | --- | --- |
| `MaxQueryBytes` | 8 KiB (8192 bytes) | Encoded `URL.RawQuery`, excluding the path and `?` |
| `MaxBodyBytes` | 64 KiB (65536 bytes) | Encoded POST form body, including percent escapes and separators |

Nonpositive fields independently use their defaults, rather than disabling limits. Both legacy and checked constructors apply defaults. Exact boundaries are allowed; larger inputs are rejected. Zero, unknown or understated Content-Length and chunked delivery cannot bypass actual body reads. A declared Content-Length above the limit is rejected without reading the body.

```go
limits := sso.RequestLimits{MaxQueryBytes: 16 << 10, MaxBodyBytes: 128 << 10}
options := sso.DefaultHTTPOptions()
options.RequestLimits = limits // Center inbound requests 中心入站请求
config := sso.DefaultClientConfig()
config.RequestLimits = limits // Application inbound callbacks 子应用入站回调
```

This snippet shows limit fields only; see the configuration guide for full assembly. `ClientConfig.MaxResponseBytes` independently limits SDK response reads and still defaults to 1 MiB. It does not control inbound callbacks. Request limits apply at receivers; they neither truncate parameters nor limit SDK sends. Coordinate receivers and proxy limits when larger application extensions are necessary.

| Entry point | Input checked |
| --- | --- |
| Center `HandleAuthorize` | GET query |
| Center `HandleToken`, `HandleIntrospect`, `HandleUserInfo`, `HandleRevoke` | POST query and form |
| Center `HandleLogout` | GET query or POST query and form |
| `ClientApp.VerifyLogoutCallback` / `LogoutCallbackHandler` | POST query and form |
| `BrowserClient.CompleteLogin` / `CallbackHandler` | GET query, before state consumption or center calls |

Directly mounted handlers have the same protection as routes registered by `Handler()`. Custom paths and parameter names do not change byte limits. Standard query/body merging, scalar validation and PKCE restrictions remain in effect.

## Middleware and HTTP boundaries

Unread `application/x-www-form-urlencoded` bodies are bounded with `http.MaxBytesReader`. Unknown-length streams read at most the limit plus one probe byte from the underlying body. Failed parsing does not publish partial Form/PostForm results or continue business processing with the valid subset of parameters.

When upstream middleware has already called `ParseForm` or `ParseMultipartForm`, cached forms remain supported. The consumed raw encoding cannot be measured again. Checks still cover available Content-Length, raw query, decoded PostForm key/value bytes and decoded merged Form key/value bytes. PostForm uses MaxBodyBytes; merged POST Form uses MaxBodyBytes + MaxQueryBytes; merged GET Form uses MaxQueryBytes. Repeated keys count once per value. These decoded checks do not allocate an encoded copy or retroactively bound upstream allocations.

Apply gateway or middleware limits **before** body parsing and handle its parse errors. Do not ignore an upstream parse error and pass partial forms to SSO. A smaller outer `MaxBytesReader` still applies, and its overflow also maps to 413. The deployment example retains its outer 64 KiB boundary for application login forms; raising the library limit does not raise this outer limit.

Standard `ParseForm` content-type behavior is retained; this module adds no JSON, multipart file-upload or compressed-body parsing. It does not drain bodies that do not participate in protocol parsing. Proxies and `http.Server` remain responsible for headers, complete request lines, connection counts and read timeouts. Application-owned login/upload endpoints and custom callbacks need their own limits.

## Errors and effects

- Overflow returns `ErrRequestTooLarge` and **413** at built-in HTTP entry points. Center errors remain JSON `Response` objects; application callbacks retain text error responses.
- Malformed queries/forms and read failures return fixed `ErrInvalidRequest` / **400**, without parser or reader diagnostics. Browser callbacks retain their fixed browser-request error message.
- Rejection does not claim a nonce, consume browser state, consume/revoke credentials, register callbacks, invoke application login/logout hooks or deliver logout notifications. Corrected requests must still satisfy existing lifetime, signature, PKCE and one-time-use rules.
- This change normalizes input parsing errors only; it does not rewrite existing business-hook or storage error responses.

Once SSO form parsing/size checks fail for an `http.Request`, subsequent validation retains that failure instead of accepting the exhausted body as a valid query-only request. Copies preserving Context retain the failure too. Correct inputs through a new request, not mutation/retry of the rejected request; no nonce or credential is consumed. Upstream parse errors ignored before SSO is called still require upstream handling.

[request_reparse_test.go](request_reparse_test.go) adds three unexecuted regression groups for repeated parsing, five center form endpoints and logout callbacks. The `TestRequestLimits` commands below include these cases.

## Upgrade and verification

Inbound defaults are stricter. Large queries, extension forms or middleware-injected maps may now return 413. Check encoded sizes before upgrading and configure receivers/proxies instead of disabling signing. Signature algorithms, scalar validation and all four credential modes retain their semantics.

[request_limits_test.go](request_limits_test.go) covers exact boundaries, encoded bytes, unknown/false lengths, bounded reads, safe parsing errors, cached forms, outer limits, every center entry point, four credential modes, nonce preservation, logout callbacks and all four browser flows. Formatting and static review only have been performed; compilation, tests and race checks have **not run**. From the repository root:

```powershell
go test ./sso/... -run TestRequestLimits -count=1
go test -race ./sso/... -run TestRequestLimits -count=1
```

These cases use memory storage, httptest Recorders and in-process transports without listeners or Redis. See the [testing guide](TESTING.md) for full regression commands.
