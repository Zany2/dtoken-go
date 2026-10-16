package oauth2

import (
	"context"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/derror"
)

// TestPKCERejectsIncompleteAndMalformedInput verifies proof-key validation never silently downgrades. TestPKCERejectsIncompleteAndMalformedInput 验证错误的证明密钥参数不会静默降级。
func TestPKCERejectsIncompleteAndMalformedInput(t *testing.T) {
	ctx := context.Background()
	s := newOAuth2TestServer()
	client := oauth2TestClient()
	if err := s.RegisterClient(client); err != nil {
		t.Fatal(err)
	}
	verifier := strings.Repeat("v", 43)
	for _, tc := range []struct{ challenge, method string }{
		{"", CodeChallengeMethodS256},
		{"", "unknown"},
		{" ", ""},
		{strings.Repeat("v", 42), CodeChallengeMethodPlain},
		{strings.Repeat("v", 129), CodeChallengeMethodPlain},
		{verifier + "!", CodeChallengeMethodPlain},
		{" " + verifier, CodeChallengeMethodPlain},
		{verifier + "\n", CodeChallengeMethodPlain},
		{verifier, "unknown"},
		{strings.Repeat("~", 43), CodeChallengeMethodS256},
		{strings.Repeat("A", 42) + "B", CodeChallengeMethodS256},
		{oauth2PKCEChallenge(verifier) + "=", CodeChallengeMethodS256},
	} {
		if code, err := s.GenerateAuthorizationCodeWithPKCE(ctx, client.ClientID, "user", client.RedirectURIs[0], nil, tc.challenge, tc.method); code != nil || !errors.Is(err, derror.ErrInvalidParam) {
			t.Fatalf("challenge %q, method %q: code = %v, error = %v", tc.challenge, tc.method, code, err)
		}
	}

	for _, valid := range []string{verifier, strings.Repeat("v", 124) + "-._~"} {
		for _, method := range []string{CodeChallengeMethodPlain, CodeChallengeMethodS256} {
			challenge := valid
			if method == CodeChallengeMethodS256 {
				challenge = oauth2PKCEChallenge(valid)
			}
			code, err := s.GenerateAuthorizationCodeWithPKCE(ctx, client.ClientID, "user", client.RedirectURIs[0], nil, challenge, method)
			if err != nil {
				t.Fatal(err)
			}
			for _, invalid := range []string{"", "short", " " + valid, valid + "\n", strings.Repeat("v", 42) + "!", strings.Repeat("x", 43)} {
				if token, err := s.ExchangeCodeForTokenWithPKCE(ctx, code.Code, client.ClientID, client.ClientSecret, client.RedirectURIs[0], invalid); token != nil || !errors.Is(err, derror.ErrInvalidCodeVerifier) {
					t.Fatalf("verifier %q: token = %v, error = %v", invalid, token, err)
				}
			}
			if token, err := s.ExchangeCodeForTokenWithPKCE(ctx, code.Code, client.ClientID, client.ClientSecret, client.RedirectURIs[0], valid); err != nil || token == nil {
				t.Fatalf("valid verifier after rejection: token = %v, error = %v", token, err)
			}
		}
	}

	code, err := s.GenerateAuthorizationCode(ctx, client.ClientID, "user", client.RedirectURIs[0], nil)
	if err != nil {
		t.Fatal(err)
	}
	if token, err := s.ExchangeCodeForTokenWithPKCE(ctx, code.Code, client.ClientID, client.ClientSecret, client.RedirectURIs[0], verifier); token != nil || !errors.Is(err, derror.ErrInvalidCodeVerifier) {
		t.Fatalf("verifier without original challenge: token = %v, error = %v", token, err)
	}
	if _, err := s.ExchangeCodeForToken(ctx, code.Code, client.ClientID, client.ClientSecret, client.RedirectURIs[0]); err != nil {
		t.Fatal(err)
	}
}

// TestAuthorizationCodePreciseDeadline verifies exact deadlines take precedence over rounded legacy fields. TestAuthorizationCodePreciseDeadline 验证精确截止时间优先于旧秒级字段。
func TestAuthorizationCodePreciseDeadline(t *testing.T) {
	ctx := context.Background()
	s := newOAuth2TestServer()
	client := oauth2TestClient()
	if err := s.RegisterClient(client); err != nil {
		t.Fatal(err)
	}
	s.codeExpiration = 1500 * time.Millisecond
	before := time.Now()
	code, err := s.GenerateAuthorizationCode(ctx, client.ClientID, "user", client.RedirectURIs[0], nil)
	if err != nil {
		t.Fatal(err)
	}
	after := time.Now()
	if code.ExpiresAt.Before(before.Add(s.codeExpiration)) || code.ExpiresAt.After(after.Add(s.codeExpiration)) || code.ExpiresIn != 2 {
		t.Fatalf("incorrect precise deadline or public lifetime: %+v", code)
	}
	stored, err := s.getAuthorizationCode(ctx, code.Code)
	if err != nil || stored == nil || !stored.ExpiresAt.Equal(code.ExpiresAt) {
		t.Fatalf("deadline did not survive encoding: %v, %v", stored, err)
	}

	for _, expired := range []bool{false, true} {
		// Deliberately disagree with the legacy fields to exercise deadline precedence. 特意让旧字段与精确截止时间不一致，以验证优先级。
		code.Used = false
		code.CreateTime = time.Now().Unix() - 60
		code.ExpiresIn = 1
		code.ExpiresAt = time.Now().Add(time.Minute)
		if expired {
			code.CreateTime = time.Now().Unix()
			code.ExpiresIn = 60
			code.ExpiresAt = time.Now().Add(-time.Second)
		}
		data, err := s.serializer.Encode(code)
		if err != nil {
			t.Fatal(err)
		}
		if err := s.storage.Set(ctx, s.getCodeKey(code.Code), data, time.Minute); err != nil {
			t.Fatal(err)
		}
		token, err := s.ExchangeCodeForToken(ctx, code.Code, client.ClientID, client.ClientSecret, client.RedirectURIs[0])
		if expired {
			if token != nil || !errors.Is(err, derror.ErrAuthCodeExpired) {
				t.Fatalf("expired exact deadline: %v, %v", token, err)
			}
		} else if err != nil || token == nil {
			t.Fatalf("valid exact deadline: %v, %v", token, err)
		}
	}
	if got := remainingAuthCodeDuration(&AuthorizationCode{CreateTime: math.MaxInt64, ExpiresIn: 1}); got != 0 {
		t.Fatalf("overflowed legacy deadline: %v", got)
	}
}

// TestAuthorizationCodeExpiryDuringEncoding verifies expired writes never get a non-positive storage TTL. TestAuthorizationCodeExpiryDuringEncoding 验证编码期间过期不会以非正 TTL 写入存储。
func TestAuthorizationCodeExpiryDuringEncoding(t *testing.T) {
	for _, used := range []bool{false, true} {
		ctx := context.Background()
		s := newOAuth2TestServer()
		client := oauth2TestClient()
		if err := s.RegisterClient(client); err != nil {
			t.Fatal(err)
		}
		var code *AuthorizationCode
		if used {
			var err error
			code, err = s.GenerateAuthorizationCode(ctx, client.ClientID, "user", client.RedirectURIs[0], nil)
			if err != nil {
				t.Fatal(err)
			}
		}
		s.serializer = oauth2ExpiryCodec{}
		if used {
			if token, err := s.ExchangeCodeForToken(ctx, code.Code, client.ClientID, client.ClientSecret, client.RedirectURIs[0]); token != nil || !errors.Is(err, derror.ErrAuthCodeExpired) {
				t.Fatalf("expired used-marker write: %v, %v", token, err)
			}
		} else if code, err := s.GenerateAuthorizationCode(ctx, client.ClientID, "user", client.RedirectURIs[0], nil); code != nil || !errors.Is(err, derror.ErrAuthCodeExpired) {
			t.Fatalf("expired initial write: %v, %v", code, err)
		}
		for key := range s.storage.(*oauth2TestStorage).values {
			if strings.HasPrefix(key, s.getTokenKey("")) || strings.HasPrefix(key, s.getRefreshKey("")) {
				t.Fatal("expired authorization code issued a token")
			}
		}
	}
}

type oauth2ExpiryCodec struct{ oauth2TestCodec }

// Encode models expiry during encoding without waiting for wall-clock time. Encode 无需等待时钟即可模拟编码期间过期。
func (c oauth2ExpiryCodec) Encode(value any) ([]byte, error) {
	if code, ok := value.(*AuthorizationCode); ok {
		code.ExpiresAt = time.Now().Add(-time.Second)
	}
	return c.oauth2TestCodec.Encode(value)
}

// TestRefreshScopeNarrowing verifies refresh scope restrictions and their persistence. TestRefreshScopeNarrowing 验证刷新权限缩减及持久化。
func TestRefreshScopeNarrowing(t *testing.T) {
	ctx := context.Background()
	s := newOAuth2TestServer()
	client := oauth2TestClient()
	if err := s.RegisterClient(client); err != nil {
		t.Fatal(err)
	}
	original, err := s.ClientCredentialsToken(ctx, client.ClientID, client.ClientSecret, []string{"read", "write"})
	if err != nil {
		t.Fatal(err)
	}
	req := &TokenRequest{GrantType: GrantTypeRefreshToken, ClientID: client.ClientID, ClientSecret: client.ClientSecret, RefreshToken: original.RefreshToken, Scopes: []string{"read"}}
	narrowed, err := s.Token(ctx, req, nil)
	if err != nil || narrowed == nil || !reflect.DeepEqual(narrowed.Scopes, []string{"read"}) {
		t.Fatalf("scope narrowing: %v, %v", narrowed, err)
	}
	if s.ValidateAccessToken(ctx, original.Token) {
		t.Fatal("old access token survived rotation")
	}
	if _, err := s.Token(ctx, req, nil); !errors.Is(err, derror.ErrInvalidRefreshToken) {
		t.Fatalf("old refresh token survived rotation: %v", err)
	}
	req.RefreshToken = narrowed.RefreshToken
	req.Scopes = []string{"write"}
	if token, err := s.Token(ctx, req, nil); token != nil || !errors.Is(err, derror.ErrInvalidScope) {
		t.Fatalf("scope expansion: %v, %v", token, err)
	}
	retained, err := s.RefreshAccessToken(ctx, client.ClientID, narrowed.RefreshToken, client.ClientSecret)
	if err != nil || retained == nil || !reflect.DeepEqual(retained.Scopes, []string{"read"}) {
		t.Fatalf("omitted scopes after rejected expansion: %v, %v", retained, err)
	}
	if err := s.RevokeToken(ctx, retained.Token); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RefreshAccessToken(ctx, client.ClientID, retained.RefreshToken, client.ClientSecret); !errors.Is(err, derror.ErrInvalidRefreshToken) {
		t.Fatalf("revoked refresh token still usable: %v", err)
	}

	empty, err := s.ClientCredentialsToken(ctx, client.ClientID, client.ClientSecret, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.RefreshToken = empty.RefreshToken
	req.Scopes = []string{"read"}
	if token, err := s.Token(ctx, req, nil); token != nil || !errors.Is(err, derror.ErrInvalidScope) {
		t.Fatalf("empty original grant expanded: %v, %v", token, err)
	}
}

// TestAtomicRefreshRejectsStaleRead verifies two servers cannot rotate the same refresh token twice. TestAtomicRefreshRejectsStaleRead 验证两个服务端不能通过过时读取重复轮换刷新令牌。
func TestAtomicRefreshRejectsStaleRead(t *testing.T) {
	ctx := context.Background()
	storage := &oauth2ClaimStorage{oauth2TestStorage: newOAuth2TestStorage()}
	first := NewOAuth2Server("auth:", "dt:", storage, oauth2TestCodec{})
	second := NewOAuth2Server("auth:", "dt:", storage, oauth2TestCodec{})
	client := oauth2TestClient()
	if err := first.RegisterClient(client); err != nil {
		t.Fatal(err)
	}
	original, err := first.ClientCredentialsToken(ctx, client.ClientID, client.ClientSecret, []string{"read"})
	if err != nil {
		t.Fatal(err)
	}
	var winner *AccessToken
	storage.afterReadKey = first.getRefreshKey(original.RefreshToken)
	storage.afterRead = func() {
		var err error
		winner, err = second.RefreshAccessToken(ctx, client.ClientID, original.RefreshToken, client.ClientSecret)
		if err != nil {
			t.Fatal(err)
		}
	}
	if token, err := first.RefreshAccessToken(ctx, client.ClientID, original.RefreshToken, client.ClientSecret); token != nil || !errors.Is(err, derror.ErrInvalidRefreshToken) {
		t.Fatalf("stale refresh = %v, %v", token, err)
	}
	if winner == nil || !first.ValidateAccessToken(ctx, winner.Token) {
		t.Fatal("winning refresh token pair is missing")
	}
	for key := range storage.values {
		if strings.HasPrefix(key, first.getTokenKey("")) && key != first.getTokenKey(winner.Token) {
			t.Fatalf("unexpected access token remains: %s", key)
		}
		if strings.HasPrefix(key, first.getRefreshKey("")) && key != first.getRefreshKey(winner.RefreshToken) {
			t.Fatalf("unexpected refresh token remains: %s", key)
		}
	}
}

// TestIssuanceRechecksClientPolicy verifies updated scopes and redirect lists apply before consuming credentials. TestIssuanceRechecksClientPolicy 验证消费凭证前重新检查更新后的权限及回调名单。
func TestIssuanceRechecksClientPolicy(t *testing.T) {
	ctx := context.Background()
	s := newOAuth2TestServer()
	client := oauth2TestClient()
	if err := s.RegisterClient(client); err != nil {
		t.Fatal(err)
	}
	code, err := s.GenerateAuthorizationCode(ctx, client.ClientID, "user", client.RedirectURIs[0], []string{"read"})
	if err != nil {
		t.Fatal(err)
	}
	token, err := s.ClientCredentialsToken(ctx, client.ClientID, client.ClientSecret, []string{"read"})
	if err != nil {
		t.Fatal(err)
	}
	client.Scopes = []string{"write"}
	if err := s.RegisterClient(client); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ExchangeCodeForToken(ctx, code.Code, client.ClientID, client.ClientSecret, code.RedirectURI); !errors.Is(err, derror.ErrInvalidScope) {
		t.Fatalf("code issued removed scope: %v", err)
	}
	if _, err := s.RefreshAccessToken(ctx, client.ClientID, token.RefreshToken, client.ClientSecret); !errors.Is(err, derror.ErrInvalidScope) {
		t.Fatalf("refresh issued removed scope: %v", err)
	}
	client.Scopes = []string{"read", "write"}
	client.RedirectURIs = []string{"https://example.com/new-callback"}
	if err := s.RegisterClient(client); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ExchangeCodeForToken(ctx, code.Code, client.ClientID, client.ClientSecret, code.RedirectURI); !errors.Is(err, derror.ErrInvalidRedirectURI) {
		t.Fatalf("code issued against removed redirect: %v", err)
	}
	client.RedirectURIs = []string{code.RedirectURI}
	if err := s.RegisterClient(client); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ExchangeCodeForToken(ctx, code.Code, client.ClientID, client.ClientSecret, code.RedirectURI); err != nil {
		t.Fatalf("rejected code was consumed: %v", err)
	}
	if _, err := s.RefreshAccessToken(ctx, client.ClientID, token.RefreshToken, client.ClientSecret); err != nil {
		t.Fatalf("rejected refresh token was consumed: %v", err)
	}
}
