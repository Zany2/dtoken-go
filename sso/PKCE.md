# SSO credential binding with PKCE S256

Ticket and OAuth2 code flows can bind redemption to the login attempt that initiated authorization. The client generates a private `code_verifier`, sends only its SHA-256 `code_challenge` during authorization, and submits the original verifier when redeeming the credential. The center checks the persisted challenge before consumption.

This uses RFC 7636 S256, also applied to this module's Ticket protocol. It does not make `ModeOAuth2` a full OAuth2/OIDC server. Client authentication, signing, HTTPS, and browser state binding remain necessary for their respective responsibilities.

## Browser integration

Enable the policy on the center's registered client:

```go
err := server.RegisterClient(&sso.Client{
    ClientID: "app-a",
    ClientSecret: "secret-a",
    RedirectURIs: []string{"https://app.example.com/sso/callback"},
    Modes: []sso.Mode{sso.ModeTicket},
    RequirePKCE: true,
})
if err != nil {
    return err
}
```

Enable automatic proof management in the application:

```go
browser, err := sso.NewBrowserClient(app, sso.BrowserClientOptions{
    Storage: stateStorage,
    CallbackURL: "https://app.example.com/sso/callback",
    StateCookie: sso.CookieOptions{SecretKey: "replace-with-a-strong-key"},
    PKCE: true,
})
if err != nil {
    return err
}
```

Continue using `BeginLogin` and `CallbackHandler`. Each flow gets an independent 256-bit random verifier stored with its return path and browser binding in server-side storage. The browser receives the challenge, state, and binding cookie. The verifier is absent from authorization URLs, cookies, and login results. Completion consumes the bound browser state, then submits the verifier in a POST form.

Swapping a credential between tabs fails without consuming the other tab's credential. That tab can still complete its legitimate flow. The failed flow has already consumed its browser state and must restart. Network or storage errors during exchange also require restarting, not replaying the browser callback.

`BeginLogin` owns all three PKCE parameters and rejects them in `extra`. `PKCE: true` supports only Ticket and OAuth2 code modes; reusable modes return `ErrModeUnsupported`. Changing this option requires restarting pending flows created under the old configuration. Replicas should share consistent options, state storage, and cookie keys.

## Manual client integration

```go
verifier := sso.NewPKCEVerifier()
authURL, err := app.AuthURLWithPKCE(callbackURL, verifier, nil)
if err != nil {
    return err
}
// Retain verifier in this flow's server-side state, then redirect to authURL.

// After validating callback browser state, retrieve the same verifier.
result, err := app.ExchangeCredential(ctx, sso.CredentialRequest{
    Mode: sso.ModeTicket,
    Ticket: ticket,
    RedirectURI: callbackURL,
    CodeVerifier: verifier,
})
```

For authorization codes, use `ModeOAuth2` and `Code`. `AuthURLWithPKCE` derives the challenge and preserves signing, replay protection, logout registration, and custom parameter names. It does not persist the verifier or validate browser state for the caller.

`ClientApp.Introspect` and `UserInfo` also require `CredentialRequest.CodeVerifier` for proof-bound one-time credentials. They remain read-only. The HTTP endpoints do not disclose identity or extension metadata on proof failure, preventing an identity lookup from bypassing redemption checks.

## Direct center APIs

```go
challenge, err := sso.PKCEChallenge(verifier)
if err != nil {
    return err
}
ticket, err := server.GenerateTicketWithOptions(ctx, clientID, loginID, callbackURL, scopes, extra,
    sso.OneTimeCredentialOptions{
        Timeout: time.Minute,
        CodeChallenge: challenge,
        CodeChallengeMethod: sso.PKCEMethodS256,
    })
if err != nil {
    return err
}
identity, err := server.ConsumeTicketWithVerifier(ctx, ticket.Ticket, clientID, clientSecret, callbackURL, verifier)
```

Codes use `GenerateOAuth2CodeWithOptions` and `ConsumeOAuth2CodeWithVerifier`. Nonpositive `Timeout` uses the mode's configured default. Challenges are independent credential fields, not business `Extra` data; custom codecs must preserve them.

Atomic storage uses atomic consumption and revalidates the consumed payload. Ordinary storage retains serialized Get/Delete within one `Server` instance. There are no new storage interfaces, distributed locks, or dependencies. Expiry, client authentication, redirect binding, account logout generations, and single consumption still apply.

`ValidateTicket` remains a trusted center-side inspection API without proof checks; do not expose it as a client identity exchange endpoint. `Revoke` retains client authentication and ownership checks without requiring a verifier, allowing credential cleanup.

## Protocol and errors

| Configuration | Default parameter | Location |
| --- | --- | --- |
| `ParamNames.CodeChallenge` | `code_challenge` | Authorization GET |
| `ParamNames.CodeChallengeMethod` | `code_challenge_method` | Authorization GET; must be `S256` |
| `ParamNames.CodeVerifier` | `code_verifier` | Exchange, userinfo, or introspect POST body |

Verifiers contain 43–128 ASCII unreserved characters: letters, digits, and `-._~`. Challenges are canonical unpadded base64url SHA-256 digests, exactly 43 characters. Plain challenges, omitted methods, duplicates, explicit empty values, verifier parameters during authorization, and challenge parameters during exchange are rejected. Verifiers in POST query strings are rejected. All protocol parameter names must be distinct.

`ErrPKCERequired` means the current registration requires binding but the authorization or stored credential lacks it. `ErrInvalidPKCE` means an invalid challenge or proof. Both map to HTTP 400 without echoing proofs. In normal consumption, invalid proof leaves the credential intact and does not register a logout callback at exchange time. Replay protection may already have consumed that request's nonce: a retry must use a new nonce and signature.

An optional policy still enforces any challenge stored on a credential. Supplying a verifier for an unbound credential fails, preventing silent challenge-stripping downgrade. Relaxing client policy does not remove existing bindings.

## Compatibility and rollout

1. Upgrade every center replica and custom codec before enabling clients. An old center may ignore challenges and cannot be considered protected.
2. Enable browser PKCE or integrate it manually. `examples/sso_client -browser-flow` enables automatic proof management; the ordinary manual example retains its existing configuration.
3. Set `RequirePKCE: true` after updating the application. Existing unbound tickets/codes then require fresh authorization before redemption or identity lookup.

Both settings default to false. Existing issuance and consumption signatures remain: unbound credentials retain their behavior when PKCE is optional; legacy issuance returns `ErrPKCERequired` when required, and legacy consumption cannot bypass stored bindings. The policy covers only Ticket/Code. Restrict the registered `Modes` to those modes if all allowed flows must use proof binding.

Treat short-lived verifiers in flow storage as login secrets. Do not log them or place them in business `Extra` or public results. PKCE does not extend sessions, change logout scope, or automatically retry one-time exchanges.

## Verification

Regression sources cover the RFC vector, validation, policy changes, legacy bypass, both storage capabilities, concurrent consumption, logout invalidation, HTTP identity bypass, signatures, custom parameters, swapped tab credentials, and pending-state migration. Formatting and static review were performed; tests, builds, examples, and real Redis integration were not run. See [testing](TESTING.md) for manual commands.
