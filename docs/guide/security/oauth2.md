English | [中文文档](../security/oauth2_zh.md)

# OAuth2 Guide

## Overview

The current project includes a lightweight OAuth2 authorization server implementation under:

- `core/oauth2`
- `core/manager/feature_oauth2.go`
- the OAuth2 wrapper functions in `dtoken`

## Supported Grant Types

The current implementation supports 4 grant types:

- `authorization_code`
- `refresh_token`
- `client_credentials`
- `password`

Matching constants:

```go
oauth2.GrantTypeAuthorizationCode
oauth2.GrantTypeRefreshToken
oauth2.GrantTypeClientCredentials
oauth2.GrantTypePassword
```

## Default Expiration

According to the current code:

- authorization code: `10` minutes
- access token: `2` hours
- refresh token: `30` days

Enable the optional module with `dtoken.NewBuilder().EnableOAuth2()` before using the global helpers. Authorization codes persist a precise `ExpiresAt` deadline; older records without it retain `CreateTime + ExpiresIn` validation.

## Client Model

```go
client := &oauth2.Client{
    ClientID:     "web-app",
    ClientSecret: "secret",
    RedirectURIs: []string{"http://localhost:3000/callback"},
    GrantTypes: []oauth2.GrantType{
        oauth2.GrantTypeAuthorizationCode,
        oauth2.GrantTypeRefreshToken,
    },
    Scopes: []string{"read", "write", "profile"},
}
```

## Register Client

```go
err := dtoken.RegisterOAuth2Client(&oauth2.Client{
    ClientID:     "web-app",
    ClientSecret: "secret",
    RedirectURIs: []string{"http://localhost:3000/callback"},
    GrantTypes: []oauth2.GrantType{
        oauth2.GrantTypeAuthorizationCode,
        oauth2.GrantTypeRefreshToken,
    },
    Scopes: []string{"read", "write", "profile"},
})
```

## Unified Token Entry

`dtoken` provides a unified token entry:

```go
token, err := dtoken.OAuth2Token(ctx, &oauth2.TokenRequest{
    GrantType:    oauth2.GrantTypeClientCredentials,
    ClientID:     "server-app",
    ClientSecret: "secret",
    Scopes:       []string{"read"},
}, nil)
```

## Authorization Code Flow

### Generate Authorization Code

```go
authCode, err := dtoken.GenerateOAuth2AuthorizationCode(
    ctx,
    "web-app",
    "10001",
    "http://localhost:3000/callback",
    []string{"read", "profile"},
)
```

This validates:

1. whether the `clientID` exists
2. whether the `redirectURI` is whitelisted
3. whether scopes are allowed
4. whether `userID` is empty

### Exchange For Token

```go
token, err := dtoken.ExchangeOAuth2CodeForToken(
    ctx,
    authCode.Code,
    "web-app",
    "secret",
    "http://localhost:3000/callback",
)
```

### PKCE

Clients can bind the authorization code to a proof key. The current implementation still requires a non-empty client secret; PKCE does not replace client authentication.

```go
authCode, err := dtoken.GenerateOAuth2AuthorizationCodeWithPKCE(
    ctx,
    "mobile-app",
    "10001",
    "myapp://oauth/callback",
    []string{"read", "profile"},
    codeChallenge,
    oauth2.CodeChallengeMethodS256,
)

token, err := dtoken.ExchangeOAuth2CodeForTokenWithPKCE(
    ctx,
    authCode.Code,
    "mobile-app",
    "secret",
    "myapp://oauth/callback",
    codeVerifier,
)
```

`codeChallengeMethod` supports `oauth2.CodeChallengeMethodPlain` and `oauth2.CodeChallengeMethodS256`. If a challenge is provided with an empty method, it defaults to `plain`. The unified token entry also accepts `TokenRequest.CodeVerifier` for `authorization_code` requests.

Verifiers and `plain` challenges must contain 43–128 ASCII characters from `A-Z`, `a-z`, `0-9`, `-`, `.`, `_`, and `~`. An S256 challenge must be the 43-character unpadded base64url encoding of the SHA-256 digest. Whitespace in challenges and verifiers is rejected rather than trimmed. Supplying a method without a challenge, or a verifier for a code issued without PKCE, is rejected.

Returned `AccessToken`:

```go
type AccessToken struct {
    Token        string
    TokenType    string
    ExpiresIn    int64
    RefreshToken string
    Scopes       []string
    UserID       string
    ClientID     string
}
```

## Client Credentials Flow

```go
token, err := dtoken.OAuth2ClientCredentialsToken(
    ctx,
    "server-app",
    "secret",
    []string{"read"},
)
```

This is meant for service-to-service calls without a user context.

## Password Flow

```go
validateUser := func(username, password string) (string, error) {
    if username == "admin" && password == "123456" {
        return "10001", nil
    }
    return "", fmt.Errorf("invalid credentials")
}

token, err := dtoken.OAuth2PasswordGrantToken(
    ctx,
    "native-app",
    "secret",
    "admin",
    "123456",
    []string{"read", "write"},
    validateUser,
)
```

The current implementation requires `validateUser`; otherwise it returns an error.

## Refresh Access Token

```go
newToken, err := dtoken.RefreshOAuth2AccessToken(
    ctx,
    "web-app",
    token.RefreshToken,
    "secret",
)
```

The current implementation:

1. validates the client credentials, grant type, and refresh-token ownership
2. checks the requested scopes against the original grant and current client allowlist
3. creates a new token pair
4. consumes the old refresh token, using an atomic operation when available
5. removes the old access token and returns the new pair

`RefreshOAuth2AccessToken` retains the original scopes. To narrow them, pass `Scopes` in an `OAuth2Token` request with `GrantTypeRefreshToken`. Omitted or empty scopes retain the original grant; supplied scopes must be a subset. Rejected scope requests do not consume the old refresh token.

## Validate Access Token

```go
valid := dtoken.ValidateOAuth2AccessToken(ctx, token.Token)

info, err := dtoken.ValidateOAuth2AccessTokenAndGetInfo(ctx, token.Token)
```

## Revoke Token

```go
err := dtoken.RevokeOAuth2Token(ctx, token.Token)
```

Revocation clears both:

- the access token
- its matching refresh token

## Scope Validation

The current implementation checks whether requested scopes belong to the client allowlist before issuing a token.  
If the client `Scopes` field is empty, scope restriction is treated as open.

Authorization-code exchange and refresh also recheck the current client scope allowlist. Code exchange additionally requires the original redirect URI to remain registered. These checks apply to new issuance; they do not retroactively revoke existing access tokens.

## Recommended Integration Model

### Authorization Endpoint

1. the user logs into your own system first
2. your system generates an authorization code
3. your system redirects back to the client callback URL

### Token Endpoint

1. the client sends `grant_type`
2. the server calls `OAuth2Token` or one of the grant-specific helpers
3. the server returns an `AccessToken` payload

### Resource Endpoint

1. extract the token from `Authorization: Bearer xxx`
2. call `ValidateOAuth2AccessToken` or `ValidateOAuth2AccessTokenAndGetInfo`
3. apply resource authorization based on `Scopes`

## Current Boundaries

The OAuth2 implementation is already usable for common authorization flows, with a few important boundaries:

1. there is no `GetOAuth2Server()` public entry in the current API
2. no ready-made HTTP introspection endpoint is provided; you write the route yourself
3. PKCE is built into the authorization-code helpers and unified token entry, but you still generate and store the client-side verifier in your application
4. ordinary storage supports sequential code exchange and refresh; concurrent one-time consumption across server instances requires truly atomic `adapter.AtomicStorage` operations

## Related Documentation

- [Nonce Anti-Replay](../security/nonce.md)
- [Refresh Token Guide](../security/refresh-token.md)
- [Authentication Guide](../core/authentication.md)
