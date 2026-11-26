package main

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	dtoken "github.com/Zany2/dtoken-go"
	"github.com/Zany2/dtoken-go/sso"
)

// TestDeploymentCanonicalOrigins accepts browser serialization and rejects equivalent center/client origins. TestDeploymentCanonicalOrigins 接受浏览器来源序列化并拒绝等价中心和子应用来源。
func TestDeploymentCanonicalOrigins(t *testing.T) {
	for input, expected := range map[string]string{
		"https://[0:0:0:0:0:0:0:1]:443":  "https://[::1]",
		"https://[::ffff:127.0.0.1]:443": "https://[::ffff:7f00:1]",
		"https://127.0.0.1:443":          "https://127.0.0.1",
	} {
		if got := canonicalOrigin(input); got != expected {
			t.Fatalf("canonical origin %q = %q, want %q", input, got, expected)
		}
	}
	env := deploymentEnvironment(t, "center", "")
	env["SSO_PUBLIC_URL"] = "http://LOCALHOST:080"
	cfg, err := loadConfig("center", func(key string) string { return env[key] })
	if err != nil || cfg.publicURL != "http://localhost" {
		t.Fatalf("normalized center = %q, %v", cfg.publicURL, err)
	}
	a, err := newApplication(cfg, sso.NewMemoryStorage(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer a.close()
	form := url.Values{"username": {cfg.demoUser}, "password": {cfg.demoPassword}}
	if response := requestHandler(a.handler, http.MethodPost, cfg.publicURL+"/login", form, "http://localhost"); response.Code != http.StatusSeeOther {
		t.Fatalf("canonical origin login = %d", response.Code)
	}
	env = deploymentEnvironment(t, "client", "alpha")
	env["SSO_PUBLIC_URL"], env["SSO_CENTER_URL"] = "http://LOCALHOST:80", "http://localhost"
	if _, err := loadConfig("client", func(key string) string { return env[key] }); err == nil {
		t.Fatal("equivalent public origins accepted as different services")
	}
	env = deploymentEnvironment(t, "center", "")
	env["SSO_CLIENTS_JSON"] = strings.ReplaceAll(env["SSO_CLIENTS_JSON"], "http://localhost:9201", "http://LOCALHOST:09201")
	cfg, err = loadConfig("center", func(key string) string { return env[key] })
	if err != nil || cfg.clients["alpha"].RedirectURIs[0] != "http://localhost:9201/sso/callback" {
		t.Fatalf("canonical callback registration: %v", err)
	}
	env["SSO_CLIENTS_JSON"] = strings.ReplaceAll(env["SSO_CLIENTS_JSON"], "http://LOCALHOST:09201", "http://user@localhost:9201")
	if _, err := loadConfig("center", func(key string) string { return env[key] }); err == nil {
		t.Fatal("callback normalization discarded forbidden userinfo")
	}
}

// TestCenterAbsoluteDeadlineConstrainsExchange verifies server Auth lifetime cannot extend the original center deadline. TestCenterAbsoluteDeadlineConstrainsExchange 验证服务端 Auth 有效期不能延长原中心截止时间。
func TestCenterAbsoluteDeadlineConstrainsExchange(t *testing.T) {
	a, err := newApplication(testConfiguration(t, "center", ""), sso.NewMemoryStorage(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer a.close()
	for _, expired := range []bool{false, true} {
		deadline := time.Now().Add(30 * time.Second)
		if expired {
			deadline = time.Now().Add(-time.Second)
		}
		token, err := a.auth.Login(context.Background(), dtoken.LoginOptions{LoginID: "alice", DeviceID: deadline.String(), Timeout: time.Hour, Extra: map[string]any{"ssoDeadline": deadline.Format(time.RFC3339Nano)}})
		if err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		sso.SetLoginIDCookie(w, a.cookie, token)
		r := cookieRequest(a.cfg.publicURL, responseCookie(t, w, a.cookie.Name))
		info, err := a.session(r)
		if err != nil || (info == nil) != expired {
			t.Fatalf("center expired=%v session=%v error=%v", expired, info != nil, err)
		}
		extra := map[string]any{"centerSessionID": sessionRef(token)}
		result := &sso.TicketExchangeResult{LoginID: "alice", Extra: extra}
		err = a.enrichExchange(r, "alpha", result)
		if expired {
			if !errors.Is(err, sso.ErrInvalidTicket) {
				t.Fatalf("expired center exchange = %v", err)
			}
			if _, err := a.authorizationExtra(r, "alpha", "alice"); !errors.Is(err, sso.ErrInvalidTicket) {
				t.Fatalf("expired center issue = %v", err)
			}
		} else if err != nil || result.RemainTokenTimeout <= 0 || result.RemainTokenTimeout > 30 {
			t.Fatalf("remaining center lifetime = %d, %v", result.RemainTokenTimeout, err)
		}
	}
}

// TestShutdownWaitsForHandlersAfterConnectionClose protects borrowed storage while canceled handlers unwind. TestShutdownWaitsForHandlersAfterConnectionClose 在已取消处理器退出期间保护借用存储。
func TestShutdownWaitsForHandlersAfterConnectionClose(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	address := make(chan string, 1)
	entered, canceled, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	shutdown := make(chan struct{})
	server := &http.Server{Addr: "127.0.0.1:0", Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-r.Context().Done()
		close(canceled)
		<-release
	}), BaseContext: func(listener net.Listener) context.Context {
		address <- listener.Addr().String()
		return context.Background()
	}}
	server.RegisterOnShutdown(func() { close(shutdown) })
	finished := make(chan error, 1)
	go func() { finished <- serve(ctx, server) }()
	t.Cleanup(func() {
		cancel()
		_ = server.Close()
		select {
		case <-release:
		default:
			close(release)
		}
	})
	var target string
	select {
	case target = <-address:
	case err := <-finished:
		t.Fatalf("server startup: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("server did not start")
	}
	clientDone := make(chan struct{})
	go func() {
		defer close(clientDone)
		client := &http.Client{Timeout: 5 * time.Second}
		if response, err := client.Get("http://" + target); err == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
		}
	}()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("handler did not start")
	}
	cancel()
	select {
	case <-shutdown:
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown did not start")
	}
	_ = server.Close()
	select {
	case <-canceled:
	case <-time.After(3 * time.Second):
		t.Fatal("handler context not canceled")
	}
	select {
	case err := <-finished:
		t.Fatalf("serve returned while handler still owned storage: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	close(release)
	select {
	case err := <-finished:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("handler completion did not release server")
	}
	select {
	case <-clientDone:
	case <-time.After(3 * time.Second):
		t.Fatal("client connection not released")
	}
}
