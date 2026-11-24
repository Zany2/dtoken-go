# SSO Client Example

For signing, replay protection, PKCE, multiple applications, and Redis replicas together, see the [complete integration example](../sso_deployment/README.md) and [deployment guide](../../sso/DEPLOYMENT.md).

The examples validate SSO configuration before listening: centers use `RegisterClientChecked` / `NewHTTPServerChecked`, and clients check `ClientApp.Validate()`. Invalid configuration becomes a startup error; see [configuration and startup validation](../../sso/CONFIGURATION.md).

This example demonstrates a business application connected to SSO:

- `/protected`: protected resource, redirects to the SSO Server when not logged in.
- `/sso/callback`: receives Ticket, calls SSO Server `/sso/token`, and gets `loginId`.
- `/sso/logout-callback`: receives unified logout callbacks from the SSO Server and clears local sessions.
- `/logout`: deletes the current local session and clears its Cookie.

The demo client stores local login state in process memory and keeps only a local `sessionId` in the Cookie. This allows the SSO Server logout callback to delete all local sessions for the received `loginId`. `/sso/logout-callback` is handled by `ClientApp.LogoutCallbackHandler`, which also verifies signatures when signing is enabled in real projects.

Local sessions expire on the server within two hours, even if a copied Cookie is submitted manually. Positive `remainTokenTimeout` and `remainSessionTimeout` values from exchange shorten that limit to the smallest value. These fields describe center-session lifetime in seconds, not Ticket lifetime. Zero means unspecified and -1 means unlimited; neither extends the local limit. Values below -1 are rejected. The deadline starts before the exchange request to conservatively account for network time; an exhausted deadline cannot create a session. The Cookie's `Expires` uses the same deadline, and server-side checks enforce it independently of Cookie second-level precision.

Expired sessions are removed on access and swept when a new session is created. A successful new login replaces the current browser's previous session; exchange or lifetime-validation failures preserve it. Restarting the client clears all local sessions. The paired demo center uses `EnrichExchangeResult` to return the original Auth token's remaining lifetime, which bounds this client's local session. Centers omitting these fields retain the local limit. This is a snapshot at login; subsequent center logout still relies on logout callbacks to remove local sessions immediately.

Start login from `/protected`. The client sends a random value through the SSO `back` parameter and binds it to a signed, HttpOnly Cookie valid for five minutes. The callback must match that Cookie before exchanging the Ticket. One browser has one pending login flow; starting another flow replaces the previous state. After an accepted callback, the state Cookie is cleared, including on an exchange failure. If login expires or exchange fails, restart from `/protected`.

This paired demo listens only on `localhost:9001` and uses mock authentication and public demo client credentials. Request and logout-callback signing are disabled to match `examples/sso_server`; the callback's client ID and timestamp checks alone do not authenticate its sender. For deployment, enable signing on both sides with managed secrets, configure HTTPS/Secure Cookies, and use shared session storage as needed.

Local `/logout` does not log out of the center. Revisiting `/protected` can therefore sign in again through the existing center Cookie. For coordinated logout, visit `http://localhost:9000/sso/logout` while both applications are running.

## Run

Select the new browser integration with `go run ./examples/sso_client -browser-flow`. This mode uses `BrowserClient` for one-time state and parallel login tabs while reusing application sessions; its authenticated page provides POST local/global logout buttons. `/logout-all` revokes locally before redirecting to the center, which reports the final outcome. The single-pending-flow behavior and GET logout links above describe the original default mode. Both modes retain the paired development configuration with memory storage and signing disabled. See the [browser integration guide](../../sso/BROWSER_CLIENT.md).

Start the SSO Server first:

```powershell
go run ./examples/sso_server
```

Then start the SSO Client:

```powershell
go run ./examples/sso_client
```

Open:

```text
http://localhost:9001/protected
```

The browser will redirect to the SSO Server login page, return to the Client with a Ticket, and create local client login state.

When `/sso/logout` is executed on the SSO Server, the Server calls this example's `/sso/logout-callback`, and the Client deletes the corresponding local sessions.

### PKCE in browser mode

`-browser-flow` now enables PKCE S256. Authorization sends only the challenge; the verifier remains in server-side flow storage and is submitted in the exchange POST. Upgrade the center first. The ordinary manual login flow retains its existing configuration. If you enable `RequirePKCE: true` on this client's registration, every Ticket/Code flow for it must submit proof, including any manually integrated flow. See the [PKCE guide](../../sso/PKCE.md).
