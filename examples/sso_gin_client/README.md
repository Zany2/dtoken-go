# Gin SSO Client Example

This example shows a Gin business application connected to the centralized SSO login center.

- `/protected`: protected resource, redirects to the SSO Server when not logged in.
- `/sso/callback`: verifies browser state, then exchanges the Ticket at SSO Server `/sso/token` for `loginId`.
- `/sso/logout-callback`: receives unified logout callbacks and clears local sessions.
- `/logout`: clears only the local client login state.

The demo client listens only on `localhost:9101`. It stores local sessions in process memory and keeps only a random `sessionId` in an HttpOnly, SameSite=Lax Cookie. Both the Cookie and the server-side session have a two-hour lifetime; replaying a copied Cookie does not extend that deadline. Expired entries are removed on access and during new logins. Restarting the client clears its sessions.

Unauthenticated visits to `/protected` create random browser state, stored in a signed five-minute Cookie and echoed by the center in the `back` parameter. The callback verifies both before exchanging the Ticket. A browser has one pending flow: a new authorization start replaces its previous state Cookie. Accepted state is cleared from the browser even if exchange fails; retry from `/protected` instead of refreshing the callback URL. The signing key is generated per process, so a restart also invalidates pending flows. A successful exchange replaces the browser's previous local session; a failed exchange preserves it.

Authentication responses use `Cache-Control: no-store`, and callback responses also use `Referrer-Policy: no-referrer`. The Gin access logger skips `/sso/callback` so its Ticket and state are not recorded in that access log.

This pair uses public demo credentials and `CheckSign: false` for local demonstration. `/sso/logout-callback` uses `ClientApp.LogoutCallbackHandler` to require the configured client ID and a timestamp within five minutes, but those checks alone do not authenticate the sender. For deployment, configure private credentials, request signing on both ends, HTTPS, and Secure Cookies; this example is not production authentication.

## Run

Start the Gin SSO Server first:

```powershell
go run ./examples/sso_gin_server
```

Then start the Gin SSO Client:

```powershell
go run ./examples/sso_gin_client
```

Open:

```text
http://localhost:9101/protected
```

The browser redirects to the SSO Server login page, returns to the Client with a Ticket, and creates local client login state.

## Verify Single Logout

After login, open:

```text
http://localhost:9100/sso/logout
```

The SSO Server pushes `/sso/logout-callback`, and the Client deletes all local sessions for the logged-in subject without affecting other users.

`/logout` clears only this browser's local session. The center remains logged in, so another visit to `/protected` may immediately sign in again. The Gin center enables best-effort single logout with a three-second callback timeout: it clears the center login even if the Client callback fails. In that case, existing Client sessions remain valid until local logout or their two-hour deadline; a successful center logout response does not guarantee that every Client session was removed.
