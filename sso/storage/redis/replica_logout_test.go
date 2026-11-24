package redis

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/sso"
)

// replicaLogoutTransport makes one target temporarily unavailable while retaining real callback handlers. replicaLogoutTransport 暂时令一个目标不可用，其他请求仍由实际回调处理器处理。
type replicaLogoutTransport struct {
	handlers map[string]http.Handler
	failBeta atomic.Bool
}

// RoundTrip simulates an HTTP failure without stopping Redis or opening a network listener. RoundTrip 模拟 HTTP 故障，不停止 Redis 或创建网络监听。
func (transport *replicaLogoutTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()
	if r.URL.Host == "beta.example" && transport.failBeta.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else if handler := transport.handlers[r.URL.Host]; handler != nil {
		handler.ServeHTTP(w, r)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
	return w.Result(), nil
}

// issueReplicaCredentials prepares all modes without consuming the one-time values. issueReplicaCredentials 准备四种模式的凭证，不消费一次性值。
func issueReplicaCredentials(t *testing.T, server *sso.Server, client *sso.Client, loginID string) []sso.CredentialRequest {
	t.Helper()
	ctx := context.Background()
	ticket, err := server.GenerateTicket(ctx, client.ClientID, loginID, client.RedirectURIs[0], nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	code, err := server.GenerateOAuth2Code(ctx, client.ClientID, loginID, client.RedirectURIs[0], nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	token, err := server.GenerateSharedToken(ctx, client.ClientID, loginID, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	session, err := server.CreateRemoteSession(ctx, client.ClientID, loginID, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return []sso.CredentialRequest{
		{Mode: sso.ModeTicket, Ticket: ticket.Ticket, RedirectURI: client.RedirectURIs[0]},
		{Mode: sso.ModeOAuth2, Code: code.Code, RedirectURI: client.RedirectURIs[0]},
		{Mode: sso.ModeSharedToken, TokenValue: token.Token},
		{Mode: sso.ModeRemoteSession, SessionID: session.SessionID},
	}
}

// assertReplicaCredentials checks all modes through signed HTTP introspection on another center. assertReplicaCredentials 通过另一中心的签名 HTTP 自省检查四种模式。
func assertReplicaCredentials(t *testing.T, app *sso.ClientApp, credentials []sso.CredentialRequest, active bool) {
	t.Helper()
	for _, credential := range credentials {
		info, err := app.Introspect(context.Background(), credential)
		if err != nil || info == nil || info.Active != active {
			t.Fatalf("mode %s: expected active=%v, error=%v", credential.Mode, active, err)
		}
	}
}

// TestRedisReplicasAccountLogoutRecovery resumes failed delivery on a new center without reviving credentials or affecting another account. TestRedisReplicasAccountLogoutRecovery 在新中心恢复失败投递，不复活旧凭证或影响其他账号。
func TestRedisReplicasAccountLogoutRecovery(t *testing.T) {
	f := newReplicaFixture(t)
	transport := &replicaLogoutTransport{handlers: make(map[string]http.Handler)}
	firstStore := f.store(t)
	first, firstHTTP := f.center(t, firstStore, transport)
	_, secondHTTP := f.center(t, f.store(t), transport)
	alpha, beta := replicaRegistration("alpha"), replicaRegistration("beta")
	for _, client := range []*sso.Client{alpha, beta} {
		if err := first.RegisterClientChecked(client); err != nil {
			t.Fatal(err)
		}
		if _, err := first.RegisterClientSession(context.Background(), "alice", client.ClientID, "https://"+client.ClientID+".example/logout-callback"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := first.RegisterClientSession(context.Background(), "bob", alpha.ClientID, "https://alpha.example/logout-callback"); err != nil {
		t.Fatal(err)
	}
	alphaStore, betaStore := f.store(t), f.store(t)
	alphaApp := f.client(t, alphaStore, alpha, sso.ModeTicket, secondHTTP.Handler())
	betaApp := f.client(t, betaStore, beta, sso.ModeTicket, secondHTTP.Handler())
	var alphaCalls, betaCalls atomic.Int32
	transport.handlers["alpha.example"] = alphaApp.LogoutCallbackHandler(func(r *http.Request, callback sso.LogoutCallback) error {
		alphaCalls.Add(1)
		return alphaStore.Delete(r.Context(), f.prefix+"alpha-session:"+callback.LoginID)
	})
	transport.handlers["beta.example"] = betaApp.LogoutCallbackHandler(func(r *http.Request, callback sso.LogoutCallback) error {
		betaCalls.Add(1)
		return betaStore.Delete(r.Context(), f.prefix+"beta-session:"+callback.LoginID)
	})
	for _, key := range []string{"center-session:alice", "alpha-session:alice", "beta-session:alice", "center-session:bob", "alpha-session:bob"} {
		if err := f.admin.Set(context.Background(), f.prefix+key, "active", time.Minute); err != nil {
			t.Fatal(err)
		}
	}
	aliceAlpha := issueReplicaCredentials(t, first, alpha, "alice")
	aliceBeta := issueReplicaCredentials(t, first, beta, "alice")
	bob := issueReplicaCredentials(t, first, alpha, "bob")
	assertReplicaCredentials(t, alphaApp, aliceAlpha, true)
	assertReplicaCredentials(t, betaApp, aliceBeta, true)
	assertReplicaCredentials(t, alphaApp, bob, true)
	transport.failBeta.Store(true)
	result, err := firstHTTP.LogoutAccount(httptest.NewRequest("POST", "/trusted-logout", nil), "alice")
	if err == nil || result == nil || result.Complete || !result.CenterRevoked || !result.CredentialsRevoked || result.Stage != "notifications" {
		t.Fatalf("partial account logout: %+v %v", result, err)
	}
	assertReplicaCredentials(t, alphaApp, aliceAlpha, false)
	assertReplicaCredentials(t, betaApp, aliceBeta, false)
	assertReplicaCredentials(t, alphaApp, bob, true)
	if bindings, err := first.GetClientSessions(context.Background(), "alice"); err != nil || len(bindings) != 2 {
		t.Fatal("partial logout did not retain retry bindings")
	}
	if value, err := f.admin.Get(context.Background(), f.prefix+"beta-session:alice"); err != nil || value == nil {
		t.Fatal("unavailable application unexpectedly acknowledged logout")
	}
	if err := firstStore.Close(); err != nil {
		t.Fatal(err)
	}
	restarted, restartedHTTP := f.center(t, f.store(t), transport)
	transport.failBeta.Store(false)
	result, err = restartedHTTP.LogoutAccount(httptest.NewRequest("POST", "/trusted-logout", nil), "alice")
	if err != nil || result == nil || !result.Complete || !result.BindingsCleared || alphaCalls.Load() != 2 || betaCalls.Load() != 1 {
		t.Fatalf("recovered logout: %+v %v alpha=%d beta=%d", result, err, alphaCalls.Load(), betaCalls.Load())
	}
	for _, key := range []string{"center-session:alice", "alpha-session:alice", "beta-session:alice"} {
		if value, err := f.admin.Get(context.Background(), f.prefix+key); err != nil || value != nil {
			t.Errorf("revoked session still exists: %s", key)
		}
	}
	for _, key := range []string{"center-session:bob", "alpha-session:bob"} {
		if value, err := f.admin.Get(context.Background(), f.prefix+key); err != nil || value == nil {
			t.Errorf("unrelated session lost: %s", key)
		}
	}
	if bindings, err := restarted.GetClientSessions(context.Background(), "alice"); err != nil || len(bindings) != 0 {
		t.Fatal("successful retry retained bindings")
	}
	if bindings, err := restarted.GetClientSessions(context.Background(), "bob"); err != nil || len(bindings) != 1 {
		t.Fatal("unrelated account binding changed")
	}
	assertReplicaCredentials(t, alphaApp, aliceAlpha, false)
	assertReplicaCredentials(t, betaApp, aliceBeta, false)
	assertReplicaCredentials(t, alphaApp, bob, true)
	assertReplicaCredentials(t, alphaApp, issueReplicaCredentials(t, restarted, alpha, "alice"), true)
}
