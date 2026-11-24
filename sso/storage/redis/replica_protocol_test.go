package redis

import (
	"context"
	"crypto/rand"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/sso"
	redisstorage "github.com/Zany2/dtoken-go/storage/redis"
)

// TestRedisReplicasAtomicCredentials permits exactly one PKCE redemption across independent Redis connections. TestRedisReplicasAtomicCredentials 在独立 Redis 连接间仅允许一次 PKCE 兑换成功。
func TestRedisReplicasAtomicCredentials(t *testing.T) {
	f := newReplicaFixture(t)
	left, _ := f.center(t, f.store(t), nil)
	right, _ := f.center(t, f.store(t), nil)
	client := replicaRegistration("alpha")
	client.RequirePKCE = true
	if err := left.RegisterClientChecked(client); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []sso.Mode{sso.ModeTicket, sso.ModeOAuth2} {
		t.Run(string(mode), func(t *testing.T) {
			ctx := context.Background()
			verifier := sso.NewPKCEVerifier()
			challenge, err := sso.PKCEChallenge(verifier)
			if err != nil {
				t.Fatal(err)
			}
			options := sso.OneTimeCredentialOptions{CodeChallenge: challenge, CodeChallengeMethod: sso.PKCEMethodS256}
			var value string
			if mode == sso.ModeTicket {
				credential, err := left.GenerateTicketWithOptions(ctx, client.ClientID, "alice", client.RedirectURIs[0], nil, nil, options)
				if err != nil {
					t.Fatal(err)
				}
				value = credential.Ticket
			} else {
				credential, err := left.GenerateOAuth2CodeWithOptions(ctx, client.ClientID, "alice", client.RedirectURIs[0], nil, nil, options)
				if err != nil {
					t.Fatal(err)
				}
				value = credential.Code
			}
			consume := func(server *sso.Server, proof string) error {
				if mode == sso.ModeTicket {
					_, err := server.ConsumeTicketWithVerifier(ctx, value, client.ClientID, client.ClientSecret, client.RedirectURIs[0], proof)
					return err
				}
				_, err := server.ConsumeOAuth2CodeWithVerifier(ctx, value, client.ClientID, client.ClientSecret, client.RedirectURIs[0], proof)
				return err
			}
			if err := consume(right, sso.NewPKCEVerifier()); !errors.Is(err, sso.ErrInvalidPKCE) {
				t.Fatalf("incorrect proof: %v", err)
			}
			start := make(chan struct{})
			results := make(chan error, 8)
			for i := 0; i < cap(results); i++ {
				server := []*sso.Server{left, right}[i%2]
				go func() { <-start; results <- consume(server, verifier) }()
			}
			close(start)
			successes := 0
			for i := 0; i < cap(results); i++ {
				err := <-results
				if err == nil {
					successes++
				} else if !errors.Is(err, sso.ErrInvalidTicket) && !errors.Is(err, sso.ErrInvalidOAuth2Code) {
					t.Errorf("unexpected redemption failure: %v", err)
				}
			}
			if successes != 1 {
				t.Fatalf("successful redemptions = %d, want 1", successes)
			}
		})
	}
}

// replicaAuthorize signs one fixed nonce so multiple handlers receive exactly the same operation. replicaAuthorize 对固定随机值签名，使多个处理器收到相同操作。
func replicaAuthorize(client *sso.Client, key, nonce string) string {
	values := url.Values{"client": {client.ClientID}, "redirect": {client.RedirectURIs[0]}, "mode": {string(sso.ModeTicket)}, "action": {sso.ActionAuthorize}, "timestamp": {time.Now().UTC().Format(time.RFC3339Nano)}, "nonce": {nonce}}
	return "https://center.example/sso/authorize?" + sso.NewSigner(key).AttachSign(values).Encode()
}

// TestRedisReplicasReplayAndRotation retains request history and registration changes after reconnecting a center. TestRedisReplicasReplayAndRotation 在中心重连后保留请求历史和注册变更。
func TestRedisReplicasReplayAndRotation(t *testing.T) {
	f := newReplicaFixture(t)
	leftStore := f.store(t)
	left, leftHTTP := f.center(t, leftStore, nil)
	_, rightHTTP := f.center(t, f.store(t), nil)
	client := replicaRegistration("alpha")
	client.VerificationKey = "next-signing-key"
	if err := left.RegisterClientChecked(client); err != nil {
		t.Fatal(err)
	}
	target := replicaAuthorize(client, client.SecretKey, rand.Text())
	start := make(chan struct{})
	results := make(chan *httptest.ResponseRecorder, 8)
	for i := 0; i < cap(results); i++ {
		handler := []*sso.HTTPServer{leftHTTP, rightHTTP}[i%2].Handler()
		go func() {
			<-start
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest("GET", target, nil))
			results <- w
		}()
	}
	close(start)
	successes := 0
	for i := 0; i < cap(results); i++ {
		w := <-results
		if w.Code == http.StatusFound {
			successes++
		} else if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), sso.ErrRequestReplayed.Error()) {
			t.Errorf("unexpected replay result: %d %s", w.Code, w.Body.String())
		}
	}
	if successes != 1 {
		t.Fatalf("successful authorizations = %d, want 1", successes)
	}
	if err := leftStore.Close(); err != nil {
		t.Fatal(err)
	}
	restarted, restartedHTTP := f.center(t, f.store(t), nil)
	oldKey := client.SecretKey
	client.SecretKey, client.VerificationKey = client.VerificationKey, oldKey
	if err := restarted.RegisterClientChecked(client); err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(target)
	if err != nil {
		t.Fatal(err)
	}
	u.RawQuery = sso.NewSigner(client.SecretKey).AttachSign(u.Query()).Encode()
	w := httptest.NewRecorder()
	rightHTTP.Handler().ServeHTTP(w, httptest.NewRequest("GET", u.String(), nil))
	if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), sso.ErrRequestReplayed.Error()) {
		t.Fatal("reconnect or key rotation reset shared nonce history")
	}
	client.VerificationKey = ""
	if err := restarted.RegisterClientChecked(client); err != nil {
		t.Fatal(err)
	}
	u, err = url.Parse(replicaAuthorize(client, oldKey, rand.Text()))
	if err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	rightHTTP.Handler().ServeHTTP(w, httptest.NewRequest("GET", u.String(), nil))
	if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), sso.ErrInvalidSign.Error()) {
		t.Fatal("peer accepted a retired signing key")
	}
	u.RawQuery = sso.NewSigner(client.SecretKey).AttachSign(u.Query()).Encode()
	w = httptest.NewRecorder()
	restartedHTTP.Handler().ServeHTTP(w, httptest.NewRequest("GET", u.String(), nil))
	if w.Code != http.StatusFound {
		t.Fatalf("invalid old signature consumed nonce: %d %s", w.Code, w.Body.String())
	}
}

// TestRedisReplicasBrowserCallback moves a pending flow to new connections and races callback consumption in all modes. TestRedisReplicasBrowserCallback 将待完成流程转移到新连接，并在四种模式下竞争消费回调。
func TestRedisReplicasBrowserCallback(t *testing.T) {
	for _, mode := range []sso.Mode{sso.ModeTicket, sso.ModeOAuth2, sso.ModeSharedToken, sso.ModeRemoteSession} {
		t.Run(string(mode), func(t *testing.T) {
			f := newReplicaFixture(t)
			server, firstCenter := f.center(t, f.store(t), nil)
			_, secondCenter := f.center(t, f.store(t), nil)
			registration := replicaRegistration("alpha")
			if err := server.RegisterClientChecked(registration); err != nil {
				t.Fatal(err)
			}
			newBrowser := func(store *redisstorage.Storage) *sso.BrowserClient {
				app := f.client(t, store, registration, mode, secondCenter.Handler())
				browser, err := sso.NewBrowserClient(app, sso.BrowserClientOptions{Storage: store, KeyPrefix: f.prefix + "flows:", CallbackURL: registration.RedirectURIs[0], PKCE: mode == sso.ModeTicket || mode == sso.ModeOAuth2, StateCookie: sso.CookieOptions{SecretKey: "test-browser-cookie-key"}})
				if err != nil {
					t.Fatal(err)
				}
				return browser
			}
			firstStore := f.store(t)
			first := newBrowser(firstStore)
			begin := httptest.NewRecorder()
			if err := first.BeginLogin(begin, httptest.NewRequest("GET", "https://alpha.example/protected", nil), "/protected", nil); err != nil {
				t.Fatal(err)
			}
			cookies := begin.Result().Cookies()
			if len(cookies) != 1 {
				t.Fatalf("state cookies = %d", len(cookies))
			}
			if err := firstStore.Close(); err != nil {
				t.Fatal(err)
			}
			authorize := httptest.NewRecorder()
			firstCenter.Handler().ServeHTTP(authorize, httptest.NewRequest("GET", begin.Header().Get("Location"), nil))
			if authorize.Code != http.StatusFound {
				t.Fatalf("authorize: %d %s", authorize.Code, authorize.Body.String())
			}
			failedRequest := httptest.NewRequest("GET", authorize.Header().Get("Location"), nil)
			failedRequest.AddCookie(cookies[0])
			if result, err := first.CompleteLogin(httptest.NewRecorder(), failedRequest); result != nil || !errors.Is(err, sso.ErrStorageUnavailable) {
				t.Fatalf("closed storage did not fail closed: %v", err)
			}
			second := newBrowser(f.store(t))
			third := newBrowser(f.store(t))
			type outcome struct {
				result *sso.BrowserLoginResult
				err    error
			}
			results := make(chan outcome, 8)
			start := make(chan struct{})
			for i := 0; i < cap(results); i++ {
				browser := []*sso.BrowserClient{second, third}[i%2]
				go func() {
					<-start
					r := httptest.NewRequest("GET", authorize.Header().Get("Location"), nil)
					r.AddCookie(cookies[0])
					result, err := browser.CompleteLogin(httptest.NewRecorder(), r)
					results <- outcome{result, err}
				}()
			}
			close(start)
			successes := 0
			for i := 0; i < cap(results); i++ {
				got := <-results
				if got.err == nil {
					successes++
					if got.result == nil || got.result.Identity.LoginID != "alice" || got.result.ReturnTo != "/protected" || !got.result.ExpiresAt.After(time.Now()) {
						t.Error("invalid completed browser identity or lifetime")
					}
				} else if !errors.Is(got.err, sso.ErrInvalidLoginState) {
					t.Errorf("unexpected callback failure: %v", got.err)
				}
			}
			if successes != 1 {
				t.Fatalf("successful callbacks = %d, want 1", successes)
			}
		})
	}
}
