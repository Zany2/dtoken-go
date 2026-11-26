package sso

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/contract"
)

// TestBrowserProtocolModes exercises center issuance through local session establishment in every mode. TestBrowserProtocolModes 验证所有模式从中心签发到本地会话建立的流程。
func TestBrowserProtocolModes(t *testing.T) {
	for _, mode := range []Mode{ModeTicket, ModeOAuth2, ModeSharedToken, ModeRemoteSession} {
		t.Run(string(mode), func(t *testing.T) {
			s := NewServer()
			defer s.Close()
			client := newTestClient()
			client.Modes, client.SecretKey = []Mode{mode}, "app-signing-key"
			client.RequirePKCE = mode == ModeTicket || mode == ModeOAuth2
			if err := s.RegisterClient(client); err != nil {
				t.Fatal(err)
			}
			params := normalizeParamNames(ParamNames{Back: "state", Ticket: "credential", Code: "authCode", Client: "application", CodeChallenge: "proof_hash", CodeChallengeMethod: "proof_method", CodeVerifier: "proof"})
			paths := DefaultEndpoints().ReplacePrefix("/identity")
			h := NewHTTPServer(s, HTTPOptions{
				ServerOptions:    ServerOptions{CheckSign: true, Params: params, Endpoints: paths},
				ReplayProtection: newReplayTestProtection(t, nil),
				LoginIDResolver:  func(*http.Request) (string, bool) { return "alice", true },
				EnrichExchangeResult: func(_ *http.Request, _ string, result *TicketExchangeResult) error {
					result.RemainSessionTimeout = 45
					return nil
				},
			})
			app := NewClientApp(ClientConfig{
				Mode: mode, ClientID: client.ClientID, ClientSecret: client.ClientSecret, SecretKey: client.SecretKey, CheckSign: true,
				ServerURL: "https://center.example", Params: params, Endpoints: paths, ReplayProtection: newReplayTestProtection(t, nil),
				HTTPClient: &http.Client{Transport: logoutReviewTransport(func(r *http.Request) (*http.Response, error) {
					if err := r.ParseForm(); err != nil {
						t.Fatal(err)
					}
					if client.RequirePKCE && (!validPKCEVerifier(r.PostForm.Get(params.CodeVerifier)) || r.URL.RawQuery != "") {
						t.Fatal("exchange proof must be carried in the POST body")
					}
					w := httptest.NewRecorder()
					h.Handler().ServeHTTP(w, r)
					return w.Result(), nil
				})},
			})
			memory := NewMemoryStorage()
			b, err := NewBrowserClient(app, BrowserClientOptions{PKCE: client.RequirePKCE, Storage: memory, CallbackURL: client.RedirectURIs[0], StateCookie: CookieOptions{SecretKey: "cookie-key"}, SessionMaxAge: time.Minute})
			if err != nil {
				t.Fatal(err)
			}
			start := httptest.NewRecorder()
			if err := b.BeginLogin(start, httptest.NewRequest("GET", "https://app.example.com/login", nil), "/orders/42", url.Values{params.Scope: {"profile"}}); err != nil {
				t.Fatal(err)
			}
			centerResponse := httptest.NewRecorder()
			h.Handler().ServeHTTP(centerResponse, httptest.NewRequest("GET", start.Header().Get("Location"), nil))
			if centerResponse.Code != http.StatusFound {
				t.Fatalf("center: %d %s", centerResponse.Code, centerResponse.Body.String())
			}
			calls := 0
			callback := b.CallbackHandler(func(w http.ResponseWriter, r *http.Request, result *BrowserLoginResult) error {
				calls++
				if result.Identity.LoginID != "alice" || result.ReturnTo != "/orders/42" || len(result.Identity.Scopes) != 1 || result.Identity.Scopes[0] != "profile" || time.Until(result.ExpiresAt) > time.Minute {
					t.Fatalf("login result: %+v", result)
				}
				if (mode == ModeTicket || mode == ModeOAuth2) && time.Until(result.ExpiresAt) > 45*time.Second {
					t.Fatal("center deadline not applied")
				}
				http.SetCookie(w, &http.Cookie{Name: "application-session", Value: "new-session", Path: "/", Expires: result.ExpiresAt, HttpOnly: true, Secure: true})
				return nil
			})
			request := httptest.NewRequest("GET", centerResponse.Header().Get("Location"), nil)
			request.AddCookie(start.Result().Cookies()[0])
			response := httptest.NewRecorder()
			callback(response, request)
			if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/orders/42" || calls != 1 {
				t.Fatalf("callback: %d %s calls=%d", response.Code, response.Body.String(), calls)
			}
			response = httptest.NewRecorder()
			callback(response, request)
			if response.Code != http.StatusBadRequest || calls != 1 {
				t.Fatalf("repeated callback: %d calls=%d", response.Code, calls)
			}
		})
	}
}

// browserFaultStorage injects ordinary state-store operation failures. browserFaultStorage 注入普通状态存储操作故障。
type browserFaultStorage struct {
	contract.Storage
	fail string
}

// Get injects a lookup failure. Get 注入查询故障。
func (s *browserFaultStorage) Get(ctx context.Context, key string) (any, error) {
	if s.fail == "get" {
		return nil, errors.New("state lookup failed")
	}
	return s.Storage.Get(ctx, key)
}

// Set injects a pending-flow write failure. Set 注入待完成流程写入故障。
func (s *browserFaultStorage) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if s.fail == "set" {
		return errors.New("state write failed")
	}
	return s.Storage.Set(ctx, key, value, ttl)
}

// Delete injects a state-consumption failure. Delete 注入状态消费故障。
func (s *browserFaultStorage) Delete(ctx context.Context, keys ...string) error {
	if s.fail == "delete" {
		return errors.New("state deletion failed")
	}
	return s.Storage.Delete(ctx, keys...)
}

// browserAtomicFaultStorage injects failure of atomic state consumption. browserAtomicFaultStorage 注入原子状态消费故障。
type browserAtomicFaultStorage struct{ *MemoryStorage }

// GetAndDelete fails without an ordinary-operation fallback. GetAndDelete 返回故障且不得回退为普通操作。
func (s *browserAtomicFaultStorage) GetAndDelete(context.Context, string) (any, error) {
	return nil, errors.New("atomic state store unavailable")
}

// TestBrowserStorageFailures prevents redirects or exchanges when flow storage fails. TestBrowserStorageFailures 确保流程存储失败时不跳转或换票。
func TestBrowserStorageFailures(t *testing.T) {
	memory := NewMemoryStorage()
	storage := &browserFaultStorage{Storage: memory}
	exchanges := 0
	b := newBrowserTestClient(t, storage, logoutReviewTransport(func(*http.Request) (*http.Response, error) {
		exchanges++
		return nil, errors.New("unexpected exchange")
	}))
	storage.fail = "set"
	w := httptest.NewRecorder()
	if err := b.BeginLogin(w, httptest.NewRequest("GET", "/login", nil), "/", nil); !errors.Is(err, ErrStorageUnavailable) || w.Header().Get("Location") != "" || len(w.Result().Cookies()) != 0 {
		t.Fatalf("begin write failure: %v", err)
	}
	storage.fail = ""
	auth, cookie := beginBrowserTestLogin(t, b, "/")
	for _, fault := range []string{"get", "delete"} {
		storage.fail = fault
		if _, err := b.CompleteLogin(httptest.NewRecorder(), browserTestCallback(b, auth, cookie)); !errors.Is(err, ErrStorageUnavailable) {
			t.Fatalf("%s: %v", fault, err)
		}
	}
	atomicClient := newBrowserTestClient(t, &browserAtomicFaultStorage{MemoryStorage: memory}, b.app.config.HTTPClient.Transport)
	if _, err := atomicClient.CompleteLogin(httptest.NewRecorder(), browserTestCallback(atomicClient, auth, cookie)); !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("atomic failure: %v", err)
	}
	if exchanges != 0 {
		t.Fatalf("storage failure allowed %d exchanges", exchanges)
	}
}

// TestBrowserReusableIdentityRejectsInvalidResponses checks active status, binding, mode, and lifetime before local login. TestBrowserReusableIdentityRejectsInvalidResponses 在本地登录前检查状态、归属、模式和期限。
func TestBrowserReusableIdentityRejectsInvalidResponses(t *testing.T) {
	for _, info := range []CredentialInfo{
		{LoginID: "alice", ClientID: "app", Mode: ModeSharedToken, ExpiresIn: 60},
		{Active: true, LoginID: "alice", ClientID: "other", Mode: ModeSharedToken, ExpiresIn: 60},
		{Active: true, LoginID: "alice", ClientID: "app", Mode: ModeRemoteSession, ExpiresIn: 60},
		{Active: true, LoginID: "alice", ClientID: "app", Mode: ModeSharedToken, ExpiresIn: 0},
	} {
		b := newBrowserTestClient(t, nil, logoutReviewTransport(func(*http.Request) (*http.Response, error) {
			w := httptest.NewRecorder()
			writeJSON(w, 200, OKResponse(info))
			return w.Result(), nil
		}))
		cfg := b.app.Config()
		cfg.Mode = ModeSharedToken
		var err error
		b, err = NewBrowserClient(NewClientApp(cfg), b.options)
		if err != nil {
			t.Fatal(err)
		}
		auth, cookie := beginBrowserTestLogin(t, b, "/")
		if _, err := b.CompleteLogin(httptest.NewRecorder(), browserTestCallback(b, auth, cookie)); !errors.Is(err, ErrBrowserExchange) {
			t.Fatalf("accepted invalid response %+v: %v", info, err)
		}
	}
}
