# SSO Server Example

This example demonstrates a minimal centralized login center:

- `/login`: mock SSO login page.
- `/sso/authorize`: issues a Ticket and redirects back to the client app.
- `/sso/token`: exchanges a Ticket for login subject information.
- `/sso/logout`: clears the SSO-center cookie and pushes client logout callbacks.

The client app sends a `callback` parameter during login redirect. After successful authorization, the SSO Server records that callback URL. When the user logs out from the login center, the Server notifies registered client apps so they can clear local login state.

## Run

```powershell
go run ./examples/sso_server
```

Default address:

```text
http://localhost:9000
```

Start `examples/sso_client` at the same time, then open:

```text
http://localhost:9001/protected
```
