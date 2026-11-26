package main

import (
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/sso"
)

// TestLocalSessionExpiryPolicy checks lifetime precedence, sentinel values, overflow, and exchange latency. TestLocalSessionExpiryPolicy 检查有效期优先级、特殊值、溢出和换票耗时。
func TestLocalSessionExpiryPolicy(t *testing.T) {
	for _, tc := range []struct {
		name           string
		token, session int64
		elapsed, want  time.Duration
		invalid        bool
	}{
		{name: "unspecified", want: localCookieTTL},
		{name: "unlimited", token: -1, session: -1, want: localCookieTTL},
		{name: "token bound", token: 60, session: 120, want: time.Minute},
		{name: "session bound", token: 120, session: 60, want: time.Minute},
		{name: "unlimited token", token: -1, session: 60, want: time.Minute},
		{name: "unspecified session", token: 60, want: time.Minute},
		{name: "local bound", token: 86400, session: 86400, want: localCookieTTL},
		{name: "large values", token: math.MaxInt64, session: math.MaxInt64, want: localCookieTTL},
		{name: "invalid token", token: -2, invalid: true},
		{name: "invalid session", token: 60, session: -2, invalid: true},
		{name: "latency counted", token: 60, elapsed: 10 * time.Second, want: time.Minute},
		{name: "latency exhausted", token: 1, elapsed: 2 * time.Second, invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			started := time.Now().Add(-tc.elapsed)
			result := &sso.TicketExchangeResult{RemainTokenTimeout: tc.token, RemainSessionTimeout: tc.session}
			expiry, err := localSessionExpiry(result, started)
			if tc.invalid {
				if err == nil || !expiry.IsZero() {
					t.Fatalf("accepted invalid lifetime: %v, %v", expiry, err)
				}
				return
			}
			if err != nil || !expiry.Equal(started.Add(tc.want)) {
				t.Fatalf("expiry=%v error=%v, want=%v", expiry, err, started.Add(tc.want))
			}
		})
	}
}

// TestCallbackUsesCenterLifetime checks a real enriched exchange reaches both the local store and Cookie. TestCallbackUsesCenterLifetime 检查真实换票扩展的有效期同时传递到本地存储和 Cookie。
func TestCallbackUsesCenterLifetime(t *testing.T) {
	resetClientSessions(t)
	center := sso.NewServer()
	defer center.Close()
	if err := center.RegisterClient(&sso.Client{ClientID: clientID, ClientSecret: clientSecret, RedirectURIs: []string{callbackURL}, Modes: []sso.Mode{sso.ModeTicket}}); err != nil {
		t.Fatal(err)
	}
	centerHTTP := sso.NewHTTPServer(center, sso.HTTPOptions{
		EnrichExchangeResult: func(_ *http.Request, _ string, result *sso.TicketExchangeResult) error {
			result.RemainTokenTimeout = 120
			result.RemainSessionTimeout = 60
			return nil
		},
	})
	previous := clientApp
	t.Cleanup(func() { clientApp = previous })
	cfg := clientApp.Config()
	cfg.HTTPClient = &http.Client{Transport: clientReviewTransport(func(r *http.Request) (*http.Response, error) {
		w := httptest.NewRecorder()
		centerHTTP.Handler().ServeHTTP(w, r)
		return w.Result(), nil
	})}
	clientApp = sso.NewClientApp(cfg)
	request := newCallbackRequest(t, "placeholder")
	ticket, err := center.GenerateTicket(request.Context(), clientID, "alice", callbackURL, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	query := request.URL.Query()
	query.Set("ticket", ticket.Ticket)
	request.URL.RawQuery = query.Encode()
	before := time.Now()
	w := httptest.NewRecorder()
	newDemoHandler().ServeHTTP(w, request)
	after := time.Now()
	if w.Code != http.StatusFound {
		t.Fatalf("callback: %d %s", w.Code, w.Body.String())
	}
	var issued *http.Cookie
	for _, item := range w.Result().Cookies() {
		if item.Name == localCookie {
			issued = item
		}
	}
	if issued == nil || issued.MaxAge <= 0 || issued.MaxAge > 60 {
		t.Fatalf("unbounded cookie: %+v", issued)
	}
	localSessions.mu.RLock()
	session, ok := localSessions.values[issued.Value]
	localSessions.mu.RUnlock()
	if !ok || session.loginID != "alice" || session.expiresAt.Before(before.Add(time.Minute)) || session.expiresAt.After(after.Add(time.Minute)) {
		t.Fatalf("unbounded local session: %+v", session)
	}
	if !issued.Expires.Equal(session.expiresAt.UTC().Truncate(time.Second)) {
		t.Fatalf("cookie and store disagree: %v, %v", issued.Expires, session.expiresAt)
	}
	check := httptest.NewRequest(http.MethodGet, "/protected", nil)
	check.AddCookie(issued)
	if id, ok := localLoginID(check); !ok || id != "alice" {
		t.Fatalf("new session: %q %v", id, ok)
	}

	// Replay the same Cookie after the server-side deadline without waiting for wall-clock expiry. 无需等待真实时间，推进服务端截止时间后重放同一 Cookie。
	localSessions.mu.Lock()
	session.expiresAt = time.Now().Add(-time.Second)
	localSessions.values[issued.Value] = session
	localSessions.mu.Unlock()
	if _, ok := localLoginID(check); ok {
		t.Fatal("copied Cookie extended the expired session")
	}
}

// TestInvalidCenterLifetimePreservesSession checks invalid upstream TTLs cannot replace an existing login. TestInvalidCenterLifetimePreservesSession 检查非法中心有效期不能替换已有登录。
func TestInvalidCenterLifetimePreservesSession(t *testing.T) {
	resetClientSessions(t)
	previous := clientApp
	t.Cleanup(func() { clientApp = previous })
	oldID, err := newLocalSession("existing-user")
	if err != nil {
		t.Fatal(err)
	}
	for _, result := range []sso.TicketExchangeResult{
		{LoginID: "new-user", RemainTokenTimeout: -2},
		{LoginID: "new-user", RemainSessionTimeout: -2},
	} {
		payload, err := json.Marshal(sso.OKResponse(result))
		if err != nil {
			t.Fatal(err)
		}
		cfg := clientApp.Config()
		cfg.HTTPClient = &http.Client{Transport: clientReviewTransport(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(string(payload)))}, nil
		})}
		clientApp = sso.NewClientApp(cfg)
		request := newCallbackRequest(t, "ticket")
		request.AddCookie(&http.Cookie{Name: localCookie, Value: oldID})
		w := httptest.NewRecorder()
		newDemoHandler().ServeHTTP(w, request)
		if w.Code != http.StatusBadGateway {
			t.Fatalf("invalid TTL status: %d", w.Code)
		}
		for _, item := range w.Result().Cookies() {
			if item.Name == localCookie {
				t.Fatal("invalid TTL issued a local Cookie")
			}
		}
		if id, ok := localLoginID(request); !ok || id != "existing-user" {
			t.Fatalf("old identity lost: %q %v", id, ok)
		}
	}
	if _, err := newLocalSessionUntil("expired", time.Now().Add(-time.Second)); err == nil {
		t.Fatal("created an expired session")
	}
	localSessions.mu.RLock()
	defer localSessions.mu.RUnlock()
	if len(localSessions.values) != 1 {
		t.Fatalf("failed logins changed sessions: %+v", localSessions.values)
	}
}
