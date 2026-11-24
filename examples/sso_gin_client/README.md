# Gin SSO Client Example

For signing, replay protection, PKCE, multiple applications, and Redis replicas together, see the [complete integration example](../sso_deployment/README.md) and [deployment guide](../../sso/DEPLOYMENT.md).

The examples validate SSO configuration before listening: centers use `RegisterClientChecked` / `NewHTTPServerChecked`, and clients check `ClientApp.Validate()`. Invalid configuration becomes a startup error; see [configuration and startup validation](../../sso/CONFIGURATION.md).

This example shows a Gin business application connected to the centralized SSO login center.

- `/protected`: protected resource, redirects to the SSO Server when not logged in.
- `/sso/callback`: verifies browser state, then exchanges the Ticket at SSO Server `/sso/token` for `loginId`.
- `/sso/logout-callback`: receives unified logout callbacks and clears local sessions.
- `/logout`: clears only the local client login state.

The demo client listens only on `localhost:9101`. It stores local sessions in process memory and keeps only a random `sessionId` in an HttpOnly, SameSite=Lax Cookie. The local lifetime is capped at two hours and shortened by the smallest positive `remainTokenTimeout` or `remainSessionTimeout` returned by exchange. These fields are center-session seconds, not Ticket lifetime. Zero means unspecified and -1 means unlimited; neither extends the local limit. Values below -1 are rejected. The deadline starts before exchange to conservatively include network time; exhausted lifetimes cannot create sessions. Cookie `Expires` uses the same deadline, with server-side checks enforcing expiry independently of Cookie second-level precision. Replaying a copied Cookie cannot extend the session. Expired entries are removed on access and during new logins. Restarting the client clears its sessions.

The paired demo center uses `EnrichExchangeResult` to return the original Auth token's remaining lifetime, which bounds this client's local session. Centers omitting these fields retain the local limit. Lifetime-validation failures preserve the previous local session. The deadline is a snapshot at login; later center logout still relies on callbacks for immediate local session removal.

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

`/logout` clears only this browser's local session. The center remains logged in, so another visit to `/protected` may immediately sign in again. The Gin center enables best-effort single logout with a three-second callback timeout: it clears the center login even if the Client callback fails. In that case, existing Client sessions remain valid until local logout or their calculated deadline; a successful center logout response does not guarantee that every Client session was removed.
