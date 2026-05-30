# SSO Client Example

This example demonstrates a business application connected to SSO:

- `/protected`: protected resource, redirects to the SSO Server when not logged in.
- `/sso/callback`: receives Ticket, calls SSO Server `/sso/token`, and gets `loginId`.
- `/sso/logout-callback`: receives unified logout callbacks from the SSO Server and clears local sessions.
- `/logout`: clears the local client login cookie.

The demo client stores local login state in process memory and keeps only a local `sessionId` in the Cookie. This allows the SSO Server logout callback to delete all local sessions for the received `loginId`. `/sso/logout-callback` is handled by `ClientApp.LogoutCallbackHandler`, which also verifies signatures when signing is enabled in real projects.

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
