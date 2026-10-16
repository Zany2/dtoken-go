package context

import (
	stdctx "context"
	"errors"
	"testing"

	"github.com/Zany2/dtoken-go/core/derror"
)

// TestMalformedHeaderFallsBackToEnabledSources verifies malformed configured headers cannot mask subsequent token sources. TestMalformedHeaderFallsBackToEnabledSources 验证格式错误的配置头不会遮蔽后续 Token 来源。
func TestMalformedHeaderFallsBackToEnabledSources(t *testing.T) {
	for _, malformed := range []string{"Bearer", "Bearer   ", "Bearer abc def", "Basic abc", "raw token"} {
		for _, source := range []string{"bearer", "cookie", "query", "body", "disabled", "configured authorization"} {
			t.Run(malformed+"/"+source, func(t *testing.T) {
				dctx, req, mgr := newTestDTokenContext(t)
				cfg := mgr.GetConfig()
				cfg.IsReadCookie = false
				if source == "configured authorization" {
					cfg.TokenName = "Authorization"
				}
				req.headers[cfg.TokenName] = malformed
				want := "next-token"
				switch source {
				case "bearer":
					req.headers[authHeader] = "Bearer " + want
				case "cookie", "configured authorization":
					cfg.IsReadCookie = true
					req.cookies[cfg.TokenName] = want
				case "query":
					cfg.IsReadQuery = true
					req.queries[cfg.TokenName] = want
				case "body":
					cfg.IsReadBody = true
					req.forms[cfg.TokenName] = want
				case "disabled":
					req.cookies[cfg.TokenName] = want
					req.queries[cfg.TokenName] = want
					req.forms[cfg.TokenName] = want
					want = ""
				}
				if got := dctx.GetTokenValue(); got != want {
					t.Fatalf("GetTokenValue() = %q, want %q", got, want)
				}
			})
		}
	}
}

// TestTokenSourceFallbackDoesNotRetryInvalidCredentials verifies extraction fallback does not switch identities after failed authentication. TestTokenSourceFallbackDoesNotRetryInvalidCredentials 验证来源回退不会在认证失败后切换身份。
func TestTokenSourceFallbackDoesNotRetryInvalidCredentials(t *testing.T) {
	ctx := stdctx.Background()
	dctx, req, mgr := newTestDTokenContext(t)
	token, err := mgr.Login(ctx, "cookie-user")
	if err != nil {
		t.Fatal(err)
	}
	cfg := mgr.GetConfig()
	req.cookies[cfg.TokenName] = token
	req.headers[cfg.TokenName] = "Bearer"
	if loginID, err := dctx.Auth().GetLoginID(ctx); err != nil || loginID != "cookie-user" {
		t.Fatalf("GetLoginID(malformed header) = %q, %v, want cookie-user", loginID, err)
	}

	for _, header := range []string{"unknown-token", "Bearer unknown-token"} {
		req.headers[cfg.TokenName] = header
		if got := dctx.GetTokenValue(); got != "unknown-token" {
			t.Fatalf("GetTokenValue() = %q, want higher-priority unknown-token", got)
		}
		if _, err := dctx.Auth().GetLoginID(ctx); !errors.Is(err, derror.ErrInvalidToken) {
			t.Fatalf("GetLoginID(invalid credential) = %v, want ErrInvalidToken", err)
		}
	}
}
