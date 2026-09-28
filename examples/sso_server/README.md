# SSO Server Example

This example demonstrates a minimal centralized login center:

- `/login`: mock SSO login page.
- `/sso/authorize`: issues a Ticket and redirects back to the client app.
- `/sso/token`: exchanges a Ticket for login subject information.
- `/sso/logout`: clears the SSO-center cookie and pushes client logout callbacks.

The client app sends a `callback` parameter during login redirect. After successful authorization, the SSO Server records that callback URL. When the user logs out from the login center, the Server notifies registered client apps so they can clear local login state.

This is a local flow demonstration: `/login` accepts an arbitrary login ID without checking a password, and an empty ID defaults to `user-1001`. The server listens only on `localhost:9000`. Login submissions read the POST form body and use Go's cross-origin request protection. The protocol routes remain available for client-to-server requests; request signing is disabled in this paired demo, while Ticket exchange still verifies the registered client secret and redirect URI.

The center uses in-memory storage and a randomly generated Cookie signing key per process. Restarting it clears registered session records and invalidates existing center Cookies. Cookies expire after two hours. Logout clears the browser's Cookie, but a previously copied Cookie remains usable until its signed expiry; immediate revocation requires a server-side session resolver and corresponding session invalidation.

Logout uses strict callback handling: if a registered client is offline or rejects the callback, center logout returns an error and retains its Cookie and callback records for retry. A successful logout clears both. Keep the client running when trying `/sso/logout`.

## Run

```powershell
go run ./examples/sso_server
```

Run this command from the repository root with Go 1.25 or later. For deployment, replace the mock login with actual authentication, configure shared storage and managed secrets, and use HTTPS with Secure Cookies.

Default address:

```text
http://localhost:9000
```

Start `examples/sso_client` at the same time, then open:

```text
http://localhost:9001/protected
```
