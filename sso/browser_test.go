package sso

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/contract"
)

// newBrowserTestClient builds a browser integration with optional custom storage and transport. newBrowserTestClient 使用可选自定义存储与传输构建浏览器接入。
func newBrowserTestClient(t *testing.T, storage contract.Storage, transport http.RoundTripper) *BrowserClient {
	t.Helper()
	if storage == nil {
		memory := NewMemoryStorage()
		storage = memory
	}
	if transport == nil {
		transport = logoutReviewTransport(func(*http.Request) (*http.Response, error) {
			w := httptest.NewRecorder()
			writeJSON(w, http.StatusOK, OKResponse(TicketExchangeResult{LoginID: "alice", RemainTokenTimeout: 60}))
			return w.Result(), nil
		})
	}
	app := NewClientApp(ClientConfig{ClientID: "app", ServerURL: "https://center.example", HTTPClient: &http.Client{Transport: transport}})
	b, err := NewBrowserClient(app, BrowserClientOptions{
		Storage: storage, CallbackURL: "https://app.example/sso/callback",
		StateCookie: CookieOptions{SecretKey: "browser-cookie-key"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// beginBrowserTestLogin captures the authorization URL and browser binding cookie. beginBrowserTestLogin 捕获授权地址与浏览器绑定 Cookie。
func beginBrowserTestLogin(t *testing.T, b *BrowserClient, returnTo string) (*url.URL, *http.Cookie) {
	t.Helper()
	w := httptest.NewRecorder()
	if err := b.BeginLogin(w, httptest.NewRequest(http.MethodGet, "https://app.example/login", nil), returnTo, nil); err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(w.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	cookies := w.Result().Cookies()
	if w.Code != http.StatusFound || len(cookies) != 1 || !cookies[0].HttpOnly || !cookies[0].Secure || cookies[0].SameSite != http.SameSiteLaxMode || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("begin login: status=%d cookies=%+v", w.Code, cookies)
	}
	return u, cookies[0]
}

// browserTestCallback constructs a callback for the configured credential mode. browserTestCallback 构造与配置凭证模式匹配的回调。
func browserTestCallback(b *BrowserClient, auth *url.URL, cookie *http.Cookie) *http.Request {
	p := b.app.config.Params
	key := map[Mode]string{ModeTicket: p.Ticket, ModeOAuth2: p.Code, ModeSharedToken: p.TokenValue, ModeRemoteSession: p.SessionID}[b.app.config.Mode]
	values := url.Values{p.Back: {auth.Query().Get(p.Back)}, key: {"credential"}}
	r := httptest.NewRequest(http.MethodGet, b.options.CallbackURL+"?"+values.Encode(), nil)
	if cookie != nil {
		r.AddCookie(cookie)
	}
	return r
}

// TestBrowserBindingAndParallelTabs verifies isolated tab state, browser binding, and one-time completion. TestBrowserBindingAndParallelTabs 验证标签页状态隔离、浏览器绑定与一次性完成。
func TestBrowserBindingAndParallelTabs(t *testing.T) {
	b := newBrowserTestClient(t, nil, nil)
	first, firstCookie := beginBrowserTestLogin(t, b, "/orders?page=2")
	second, secondCookie := beginBrowserTestLogin(t, b, "/profile")
	if firstCookie.Name == secondCookie.Name {
		t.Fatal("parallel flows share a cookie")
	}
	for _, cookie := range []*http.Cookie{nil, secondCookie} {
		if _, err := b.CompleteLogin(httptest.NewRecorder(), browserTestCallback(b, first, cookie)); !errors.Is(err, ErrInvalidLoginState) {
			t.Fatalf("cross-browser binding: %v", err)
		}
	}
	otherBrowser := *secondCookie
	otherBrowser.Name = firstCookie.Name
	if _, err := b.CompleteLogin(httptest.NewRecorder(), browserTestCallback(b, first, &otherBrowser)); !errors.Is(err, ErrInvalidLoginState) {
		t.Fatalf("different binding with matching cookie name: %v", err)
	}
	duplicate := browserTestCallback(b, first, firstCookie)
	duplicate.AddCookie(firstCookie)
	if _, err := b.CompleteLogin(httptest.NewRecorder(), duplicate); !errors.Is(err, ErrInvalidLoginState) {
		t.Fatalf("duplicate cookie: %v", err)
	}
	for i, auth := range []*url.URL{second, first} {
		cookie := []*http.Cookie{secondCookie, firstCookie}[i]
		w := httptest.NewRecorder()
		result, err := b.CompleteLogin(w, browserTestCallback(b, auth, cookie))
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"/profile", "/orders?page=2"}[i]
		if result.Identity.LoginID != "alice" || result.ReturnTo != want || time.Until(result.ExpiresAt) > time.Minute || !result.ExpiresAt.After(time.Now()) {
			t.Fatalf("result: %+v", result)
		}
		cleared := w.Result().Cookies()
		if len(cleared) != 1 || cleared[0].Name != cookie.Name || cleared[0].MaxAge != -1 {
			t.Fatalf("wrong flow cleared: %+v", cleared)
		}
		if _, err := b.CompleteLogin(httptest.NewRecorder(), browserTestCallback(b, auth, cookie)); !errors.Is(err, ErrInvalidLoginState) {
			t.Fatalf("callback replay: %v", err)
		}
	}
}

// TestBrowserConcurrentCallback checks atomic shared instances and the ordinary-storage fallback. TestBrowserConcurrentCallback 检查原子存储多实例与普通存储回退。
func TestBrowserConcurrentCallback(t *testing.T) {
	for _, basic := range []bool{false, true} {
		t.Run(map[bool]string{false: "atomic replicas", true: "ordinary instance"}[basic], func(t *testing.T) {
			memory := NewMemoryStorage()
			var storage contract.Storage = memory
			if basic {
				storage = &basicSSOStorage{inner: memory}
			}
			var exchanges atomic.Int32
			transport := logoutReviewTransport(func(*http.Request) (*http.Response, error) {
				exchanges.Add(1)
				w := httptest.NewRecorder()
				writeJSON(w, http.StatusOK, OKResponse(TicketExchangeResult{LoginID: "alice"}))
				return w.Result(), nil
			})
			first := newBrowserTestClient(t, storage, transport)
			second := first
			if !basic {
				second = newBrowserTestClient(t, storage, transport)
			}
			auth, cookie := beginBrowserTestLogin(t, first, "/")
			var wg sync.WaitGroup
			var successes atomic.Int32
			for i := 0; i < 16; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					b := []*BrowserClient{first, second}[i%2]
					_, err := b.CompleteLogin(httptest.NewRecorder(), browserTestCallback(b, auth, cookie))
					if err == nil {
						successes.Add(1)
					} else if !errors.Is(err, ErrInvalidLoginState) {
						t.Errorf("complete: %v", err)
					}
				}(i)
			}
			wg.Wait()
			if successes.Load() != 1 || exchanges.Load() != 1 {
				t.Fatalf("successes=%d exchanges=%d", successes.Load(), exchanges.Load())
			}
		})
	}
}

// TestBrowserFailureLifecycle checks state consumption, sanitized errors, and failed local-session hooks. TestBrowserFailureLifecycle 检查状态消费、错误脱敏及本地会话钩子失败。
func TestBrowserFailureLifecycle(t *testing.T) {
	for _, upstreamFailure := range []bool{false, true} {
		calls := 0
		b := newBrowserTestClient(t, nil, logoutReviewTransport(func(*http.Request) (*http.Response, error) {
			w := httptest.NewRecorder()
			if upstreamFailure {
				http.Error(w, "secret-token-and-upstream-details", 500)
			} else {
				writeJSON(w, 200, OKResponse(TicketExchangeResult{LoginID: "alice"}))
			}
			return w.Result(), nil
		}))
		auth, cookie := beginBrowserTestLogin(t, b, "/")
		handler := b.CallbackHandler(func(http.ResponseWriter, *http.Request, *BrowserLoginResult) error {
			calls++
			return errors.New("local-store-sensitive-details")
		})
		w := httptest.NewRecorder()
		handler(w, browserTestCallback(b, auth, cookie))
		want, wantCalls := http.StatusInternalServerError, 1
		if upstreamFailure {
			want, wantCalls = http.StatusBadGateway, 0
		}
		if w.Code != want || calls != wantCalls || strings.Contains(w.Body.String(), "details") || len(w.Result().Cookies()) != 1 {
			t.Fatalf("failure: %d calls=%d body=%s", w.Code, calls, w.Body.String())
		}
		if _, err := b.CompleteLogin(httptest.NewRecorder(), browserTestCallback(b, auth, cookie)); !errors.Is(err, ErrInvalidLoginState) {
			t.Fatalf("failed request reused state: %v", err)
		}
		fresh, freshCookie := beginBrowserTestLogin(t, b, "/")
		if fresh.Query().Get("back") == auth.Query().Get("back") || freshCookie.Name == cookie.Name {
			t.Fatal("retry reused browser state")
		}
	}
}

// TestBrowserInputAndDeadlines checks configuration, safe navigation, malformed callbacks, and session caps. TestBrowserInputAndDeadlines 检查配置、安全跳转、错误回调及会话上限。
func TestBrowserInputAndDeadlines(t *testing.T) {
	b := newBrowserTestClient(t, nil, nil)
	for _, target := range []string{"https://evil.example", "//evil.example", "/%2fevil.example", "/\\evil.example", "/%5cevil.example", "/ok%0d%0aLocation:evil", "javascript:alert(1)"} {
		w := httptest.NewRecorder()
		if err := b.BeginLogin(w, httptest.NewRequest("GET", "/login", nil), target, nil); !errors.Is(err, ErrInvalidReturnTo) || len(w.Result().Cookies()) != 0 {
			t.Fatalf("return path %q: %v", target, err)
		}
	}
	for _, edit := range []func(*BrowserClientOptions){
		func(o *BrowserClientOptions) { o.Storage = nil },
		func(o *BrowserClientOptions) { o.StateCookie.SecretKey = "" },
		func(o *BrowserClientOptions) { o.StateCookie.Domain = ".example" },
		func(o *BrowserClientOptions) { o.StateCookie.Path = "/other" },
		func(o *BrowserClientOptions) { o.StateCookie.Name = "bad name" },
		func(o *BrowserClientOptions) { o.StateCookie.SameSite = http.SameSiteStrictMode },
		func(o *BrowserClientOptions) { o.StateCookie.MaxAge = -time.Second },
		func(o *BrowserClientOptions) { o.SessionMaxAge = -time.Second },
		func(o *BrowserClientOptions) { o.CallbackURL = "//app.example/callback" },
		func(o *BrowserClientOptions) { o.CallbackURL += "?ticket=override" },
	} {
		opts := b.options
		edit(&opts)
		if _, err := NewBrowserClient(b.app, opts); !errors.Is(err, ErrBrowserConfig) {
			t.Fatalf("invalid config: %v", err)
		}
	}
	for _, query := range []string{"&ticket=duplicate", "&code=other-mode", "&back=duplicate", "&bad=%ZZ"} {
		auth, cookie := beginBrowserTestLogin(t, b, "/")
		r := browserTestCallback(b, auth, cookie)
		r.URL.RawQuery += query
		if _, err := b.CompleteLogin(httptest.NewRecorder(), r); err == nil {
			t.Fatalf("accepted query %q", query)
		}
		if _, err := b.CompleteLogin(httptest.NewRecorder(), browserTestCallback(b, auth, cookie)); err != nil {
			t.Fatalf("malformed callback consumed valid state: %v", err)
		}
	}
	started := time.Now()
	for _, tc := range []struct {
		token, session int64
		maxAge, want   time.Duration
		fail           bool
	}{
		{0, 0, time.Hour, time.Hour, false}, {-1, -1, time.Hour, time.Hour, false},
		{60, 30, time.Hour, 30 * time.Second, false}, {1<<63 - 1, 0, time.Hour, time.Hour, false},
		{1, 0, 1500 * time.Millisecond, time.Second, false}, {-2, 0, time.Hour, 0, true},
	} {
		deadline, err := browserSessionDeadline(&TicketExchangeResult{LoginID: "alice", RemainTokenTimeout: tc.token, RemainSessionTimeout: tc.session}, started, tc.maxAge)
		if tc.fail {
			if err == nil {
				t.Fatal("accepted invalid TTL")
			}
			continue
		}
		if err != nil || !deadline.Equal(started.Add(tc.want)) {
			t.Fatalf("deadline: %s %v", deadline, err)
		}
	}
	if _, err := browserSessionDeadline(&TicketExchangeResult{LoginID: "alice", RemainTokenTimeout: 1}, time.Now().Add(-time.Minute), time.Hour); !errors.Is(err, ErrRequestExpired) {
		t.Fatalf("exchange latency: %v", err)
	}
}

// TestBrowserStateExpiry checks server deadlines even when storage retains a record. TestBrowserStateExpiry 检查存储仍保留记录时的服务端截止时间。
func TestBrowserStateExpiry(t *testing.T) {
	b := newBrowserTestClient(t, nil, nil)
	auth, cookie := beginBrowserTestLogin(t, b, "/")
	key := b.options.KeyPrefix + auth.Query().Get("back")
	raw, err := b.options.Storage.Get(context.Background(), key)
	if err != nil {
		t.Fatal(err)
	}
	var record browserLoginState
	if err := json.Unmarshal([]byte(raw.(string)), &record); err != nil {
		t.Fatal(err)
	}
	record.ExpiresAt = time.Now().Add(-time.Second)
	payload, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := b.options.Storage.Set(context.Background(), key, string(payload), time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, err := b.CompleteLogin(httptest.NewRecorder(), browserTestCallback(b, auth, cookie)); !errors.Is(err, ErrInvalidLoginState) {
		t.Fatalf("expired state: %v", err)
	}
}

// TestBrowserLogout checks POST and origin enforcement plus local/global routing and hook failure. TestBrowserLogout 检查 POST 与来源限制、本地与全局跳转及钩子失败。
func TestBrowserLogout(t *testing.T) {
	b := newBrowserTestClient(t, nil, nil)
	for _, tc := range []struct {
		method, origin string
		global, fail   bool
		status         int
	}{
		{"GET", "https://app.example", false, false, 405}, {"POST", "", false, false, 403},
		{"POST", "https://evil.example", false, false, 403}, {"POST", "null", false, false, 403},
		{"POST", "https://app.example", false, false, 303}, {"POST", "https://app.example", true, false, 303},
		{"POST", "https://app.example", true, true, 500},
	} {
		calls := 0
		h := b.LogoutHandler(func(http.ResponseWriter, *http.Request) error {
			calls++
			if tc.fail {
				return errors.New("failed")
			}
			return nil
		}, tc.global, "/signed-out")
		r := httptest.NewRequest(tc.method, "https://app.example/logout", nil)
		if tc.origin != "" {
			r.Header.Set("Origin", tc.origin)
		}
		w := httptest.NewRecorder()
		h(w, r)
		if w.Code != tc.status {
			t.Fatalf("logout %+v: %d", tc, w.Code)
		}
		if tc.status == 403 || tc.status == 405 {
			if calls != 0 {
				t.Fatal("invalid request invoked logout")
			}
			continue
		}
		if calls != 1 {
			t.Fatalf("logout calls=%d", calls)
		}
		if tc.fail {
			if w.Header().Get("Location") != "" {
				t.Fatal("redirected after failure")
			}
			continue
		}
		u, err := url.Parse(w.Header().Get("Location"))
		if err != nil {
			t.Fatal(err)
		}
		if tc.global {
			if u.Host != "center.example" || u.Path != "/sso/logout" {
				t.Fatalf("center redirect: %s", u)
			}
		} else if u.String() != "/signed-out" {
			t.Fatalf("local redirect: %s", u)
		}
	}
}
