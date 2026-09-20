package oauth2

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/derror"
)

// TestRemainingAuthCodeDurationPreservesDeadline verifies precise deadlines and large durations. TestRemainingAuthCodeDurationPreservesDeadline 验证精确到期时间与极大时长。
func TestRemainingAuthCodeDurationPreservesDeadline(t *testing.T) {
	code := &AuthorizationCode{CreateTime: time.Now().Unix(), ExpiresIn: 60}
	upper := time.Until(time.Unix(code.CreateTime+code.ExpiresIn, 0))
	if got := remainingAuthCodeDuration(code); got <= 0 || got > upper {
		t.Fatalf("remaining TTL = %v, want positive and <= %v", got, upper)
	}
	code.ExpiresIn = durationSeconds(time.Duration(math.MaxInt64))
	if got := remainingAuthCodeDuration(code); got <= 0 {
		t.Fatalf("large duration overflowed to %v", got)
	}
	code.CreateTime = time.Now().Unix() - 1
	code.ExpiresIn = 1
	if got := remainingAuthCodeDuration(code); got != 0 {
		t.Fatalf("expired TTL = %v, want 0", got)
	}
}

// TestAuthorizationCodeClaimRejectsStaleExchange verifies two servers cannot exchange one code from stale reads. TestAuthorizationCodeClaimRejectsStaleExchange 验证两个服务端不能通过过时读取重复兑换同一授权码。
func TestAuthorizationCodeClaimRejectsStaleExchange(t *testing.T) {
	ctx := context.Background()
	storage := &oauth2ClaimStorage{oauth2TestStorage: newOAuth2TestStorage()}
	first := NewOAuth2Server("auth:", "dt:", storage, oauth2TestCodec{})
	second := NewOAuth2Server("auth:", "dt:", storage, oauth2TestCodec{})
	client := oauth2TestClient()
	if err := first.RegisterClient(client); err != nil {
		t.Fatal(err)
	}
	code, err := first.GenerateAuthorizationCode(ctx, client.ClientID, "user", client.RedirectURIs[0], []string{"read"})
	if err != nil {
		t.Fatal(err)
	}

	// Exchange on the second server after the first has read an unused snapshot. 首个服务端读到未使用快照后，在第二个服务端完成兑换。
	storage.afterReadKey = first.getCodeKey(code.Code)
	storage.afterRead = func() {
		token, exchangeErr := second.ExchangeCodeForToken(ctx, code.Code, client.ClientID, client.ClientSecret, client.RedirectURIs[0])
		if exchangeErr != nil || token == nil {
			t.Fatalf("winning exchange = %v, %v", token, exchangeErr)
		}
	}
	token, err := first.ExchangeCodeForToken(ctx, code.Code, client.ClientID, client.ClientSecret, client.RedirectURIs[0])
	if token != nil || !errors.Is(err, derror.ErrAuthCodeUsed) {
		t.Fatalf("stale exchange = %v, %v; want ErrAuthCodeUsed", token, err)
	}
	if storage.afterRead != nil {
		t.Fatal("interleaved exchange was not triggered")
	}
	accessTokens := 0
	for key := range storage.values {
		if strings.HasPrefix(key, first.getTokenKey("")) {
			accessTokens++
		}
	}
	if accessTokens != 1 {
		t.Fatalf("issued access tokens = %d, want 1", accessTokens)
	}
	if _, err = first.ExchangeCodeForToken(ctx, code.Code, client.ClientID, client.ClientSecret, client.RedirectURIs[0]); !errors.Is(err, derror.ErrAuthCodeUsed) {
		t.Fatalf("replay error = %v, want preserved ErrAuthCodeUsed", err)
	}
}

// TestAuthorizationCodeChecksBeforeClaim verifies rejected requests leave the authorization code usable. TestAuthorizationCodeChecksBeforeClaim 验证请求校验失败后授权码仍可使用。
func TestAuthorizationCodeChecksBeforeClaim(t *testing.T) {
	ctx := context.Background()
	storage := &oauth2ClaimStorage{oauth2TestStorage: newOAuth2TestStorage()}
	server := NewOAuth2Server("auth:", "dt:", storage, oauth2TestCodec{})
	client := oauth2TestClient()
	if err := server.RegisterClient(client); err != nil {
		t.Fatal(err)
	}
	verifier := strings.Repeat("v", 43)
	code, err := server.GenerateAuthorizationCodeWithPKCE(ctx, client.ClientID, "user", client.RedirectURIs[0], nil, oauth2PKCEChallenge(verifier), CodeChallengeMethodS256)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		secret, redirect, verifier string
		want                       error
	}{
		{"wrong", client.RedirectURIs[0], verifier, derror.ErrInvalidClientCredentials},
		{client.ClientSecret, "https://other.example/callback", verifier, derror.ErrRedirectURIMismatch},
		{client.ClientSecret, client.RedirectURIs[0], "wrong", derror.ErrInvalidCodeVerifier},
	} {
		if token, err := server.ExchangeCodeForTokenWithPKCE(ctx, code.Code, client.ClientID, tt.secret, tt.redirect, tt.verifier); token != nil || !errors.Is(err, tt.want) {
			t.Fatalf("rejected exchange = %v, %v; want %v", token, err, tt.want)
		}
	}
	if token, err := server.ExchangeCodeForTokenWithPKCE(ctx, code.Code, client.ClientID, client.ClientSecret, client.RedirectURIs[0], verifier); err != nil || token == nil {
		t.Fatalf("valid exchange after rejected requests = %v, %v", token, err)
	}
}

// TestAuthorizationCodeClaimFailureDoesNotIssueToken verifies missing records and claim failures stop issuance. TestAuthorizationCodeClaimFailureDoesNotIssueToken 验证记录消失或领取失败会停止签发。
func TestAuthorizationCodeClaimFailureDoesNotIssueToken(t *testing.T) {
	for _, missing := range []bool{false, true} {
		name := "storage failure"
		if missing {
			name = "record disappeared"
		}
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			storage := &oauth2ClaimStorage{oauth2TestStorage: newOAuth2TestStorage()}
			server := NewOAuth2Server("auth:", "dt:", storage, oauth2TestCodec{})
			client := oauth2TestClient()
			if err := server.RegisterClient(client); err != nil {
				t.Fatal(err)
			}
			code, err := server.GenerateAuthorizationCode(ctx, client.ClientID, "user", client.RedirectURIs[0], nil)
			if err != nil {
				t.Fatal(err)
			}
			want := derror.ErrStorageUnavailable
			if missing {
				want = derror.ErrInvalidAuthCode
				storage.afterReadKey = server.getCodeKey(code.Code)
				storage.afterRead = func() {
					if err := storage.Delete(ctx, server.getCodeKey(code.Code)); err != nil {
						t.Fatal(err)
					}
				}
			} else {
				storage.claimError = errors.New("claim unavailable")
			}
			token, err := server.ExchangeCodeForToken(ctx, code.Code, client.ClientID, client.ClientSecret, client.RedirectURIs[0])
			if token != nil || !errors.Is(err, want) {
				t.Fatalf("exchange = %v, %v; want %v", token, err, want)
			}
			for key := range storage.values {
				if strings.HasPrefix(key, server.getTokenKey("")) || strings.HasPrefix(key, server.getRefreshKey("")) {
					t.Fatal("claim failure issued a token mapping")
				}
			}
		})
	}
}

// oauth2ClaimStorage models atomic operations in a single-goroutine, deterministically interleaved test. oauth2ClaimStorage 在单 goroutine 的确定性交错测试中模拟原子操作。
type oauth2ClaimStorage struct {
	*oauth2TestStorage
	afterReadKey string
	afterRead    func()
	claimError   error
}

func (s *oauth2ClaimStorage) Get(ctx context.Context, key string) (any, error) {
	value, err := s.oauth2TestStorage.Get(ctx, key)
	if key == s.afterReadKey && s.afterRead != nil {
		hook := s.afterRead
		s.afterRead = nil
		hook()
	}
	return value, err
}

func (s *oauth2ClaimStorage) GetAndDelete(ctx context.Context, key string) (any, error) {
	if s.claimError != nil {
		return nil, s.claimError
	}
	value, err := s.oauth2TestStorage.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	return value, s.Delete(ctx, key)
}

func (s *oauth2ClaimStorage) SetIfAbsent(ctx context.Context, key string, value any, ttl time.Duration) (bool, error) {
	if s.Exists(ctx, key) {
		return false, nil
	}
	return true, s.Set(ctx, key, value, ttl)
}
