# SSO Client Example

This example demonstrates a business application connected to SSO:

- `/protected`: protected resource, redirects to the SSO Server when not logged in.
- `/sso/callback`: receives Ticket, calls SSO Server `/sso/token`, and gets `loginId`.
- `/sso/logout-callback`: receives unified logout callbacks from the SSO Server and clears local sessions.
- `/logout`: deletes the current local session and clears its Cookie.

The demo client stores local login state in process memory and keeps only a local `sessionId` in the Cookie. This allows the SSO Server logout callback to delete all local sessions for the received `loginId`. `/sso/logout-callback` is handled by `ClientApp.LogoutCallbackHandler`, which also verifies signatures when signing is enabled in real projects.

Local sessions expire on the server after two hours, even if a copied Cookie is submitted manually. Expired sessions are removed on access and swept when a new session is created. A successful new login replaces the current browser's previous session; an exchange failure preserves it. Restarting the client clears all local sessions.

Start login from `/protected`. The client sends a random value through the SSO `back` parameter and binds it to a signed, HttpOnly Cookie valid for five minutes. The callback must match that Cookie before exchanging the Ticket. One browser has one pending login flow; starting another flow replaces the previous state. After an accepted callback, the state Cookie is cleared, including on an exchange failure. If login expires or exchange fails, restart from `/protected`.

This paired demo listens only on `localhost:9001` and uses mock authentication and public demo client credentials. Request and logout-callback signing are disabled to match `examples/sso_server`; the callback's client ID and timestamp checks alone do not authenticate its sender. For deployment, enable signing on both sides with managed secrets, configure HTTPS/Secure Cookies, and use shared session storage as needed.

Local `/logout` does not log out of the center. Revisiting `/protected` can therefore sign in again through the existing center Cookie. For coordinated logout, visit `http://localhost:9000/sso/logout` while both applications are running.

## Run

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
