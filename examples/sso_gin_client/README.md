# Gin SSO Client Example

This example shows a Gin business application connected to the centralized SSO login center.

- `/protected`: protected resource, redirects to the SSO Server when not logged in.
- `/sso/callback`: receives Ticket and calls SSO Server `/sso/token` for `loginId`.
- `/sso/logout-callback`: receives unified logout callbacks and clears local sessions.
- `/logout`: clears only the local client login state.

The demo client stores local login state in process memory and keeps only a local `sessionId` in the Cookie. `/sso/logout-callback` is handled by `ClientApp.LogoutCallbackHandler`, which verifies signatures when signing is enabled in real projects.

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
http://localhost:9100/sso/logout?loginId=user-1001
```

The SSO Server pushes `/sso/logout-callback`, and the Client deletes local sessions for `user-1001`.
