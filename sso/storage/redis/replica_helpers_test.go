package redis

import (
	"context"
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/sso"
	redisstorage "github.com/Zany2/dtoken-go/storage/redis"
)

// replicaFixture isolates live Redis records and owns independent connections used by simulated replicas. replicaFixture 隔离真实 Redis 记录并持有模拟副本的独立连接。
type replicaFixture struct {
	endpoint string
	prefix   string
	admin    *redisstorage.Storage
	stores   []*redisstorage.Storage
}

// newReplicaFixture skips only absent opt-in configuration and never clears the Redis database. newReplicaFixture 仅在未配置集成环境时跳过，绝不清空 Redis 数据库。
func newReplicaFixture(t *testing.T) *replicaFixture {
	t.Helper()
	endpoint := strings.TrimSpace(os.Getenv("DTOKEN_REDIS_URL"))
	if endpoint == "" {
		t.Skip("set DTOKEN_REDIS_URL to run Redis replica integration tests")
	}
	admin, err := redisstorage.NewStorage(endpoint)
	if err != nil {
		t.Fatal("cannot connect to configured test Redis")
	}
	f := &replicaFixture{endpoint: endpoint, prefix: "dtoken:test:replicas:" + rand.Text() + ":", admin: admin}
	t.Cleanup(func() {
		defer admin.Close()
		for _, store := range f.stores {
			_ = store.Close()
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		keys, err := admin.Keys(ctx, f.prefix+"*")
		if err != nil {
			t.Error("cannot enumerate this test's Redis records for cleanup")
			return
		}
		for _, key := range keys {
			if !strings.HasPrefix(key, f.prefix) {
				t.Error("refusing cleanup outside test prefix")
				return
			}
		}
		if len(keys) > 0 {
			if err := admin.Delete(ctx, keys...); err != nil {
				t.Error("cannot clean this test's Redis records")
			}
		}
	})
	return f
}

// store creates a separate Redis connection without changing the fixture namespace. store 新建独立 Redis 连接，保持测试命名空间不变。
func (f *replicaFixture) store(t *testing.T) *redisstorage.Storage {
	t.Helper()
	store, err := redisstorage.NewStorage(f.endpoint)
	if err != nil {
		t.Fatal("cannot create replica connection to configured Redis")
	}
	f.stores = append(f.stores, store)
	return store
}

// replay constructs an independent protection instance backed by a shared Redis namespace. replay 构造由共享 Redis 命名空间支撑的独立防重放实例。
func (f *replicaFixture) replay(t *testing.T, store *redisstorage.Storage, scope string) *sso.ReplayProtection {
	t.Helper()
	protection, err := sso.NewReplayProtection(sso.ReplayProtectionOptions{Storage: store, KeyPrefix: f.prefix + scope, MaxAge: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	return protection
}

// center wires an independent center without overwriting persisted client registrations. center 组装独立中心，不覆盖已持久化的客户端注册。
func (f *replicaFixture) center(t *testing.T, store *redisstorage.Storage, transport http.RoundTripper) (*sso.Server, *sso.HTTPServer) {
	t.Helper()
	server := NewServerFromStorage(store, sso.WithKeyPrefix(f.prefix+"protocol:"))
	t.Cleanup(func() { _ = server.Close() })
	options := sso.DefaultHTTPOptions()
	options.LoginIDResolver = func(*http.Request) (string, bool) { return "alice", true }
	options.CenterLogout = func(r *http.Request, loginID string) error {
		return store.Delete(r.Context(), f.prefix+"center-session:"+loginID)
	}
	options.ReplayProtection = f.replay(t, store, "center-nonces:")
	options.LogoutHTTPClient = &http.Client{Transport: transport, Timeout: 5 * time.Second}
	center, err := sso.NewHTTPServerChecked(server, options)
	if err != nil {
		t.Fatal(err)
	}
	return server, center
}

// replicaRegistration provides isolated signing identities for all four protocol modes. replicaRegistration 为四种协议模式提供独立的签名身份。
func replicaRegistration(id string) *sso.Client {
	return &sso.Client{ClientID: id, ClientSecret: id + "-exchange", SecretKey: id + "-signing", RedirectURIs: []string{"https://" + id + ".example/callback"}, Modes: []sso.Mode{sso.ModeTicket, sso.ModeOAuth2, sso.ModeSharedToken, sso.ModeRemoteSession}}
}

// client composes the public SDK using independent callback replay state and an in-process center transport. client 使用独立回调防重放实例及进程内中心传输组装公开 SDK。
func (f *replicaFixture) client(t *testing.T, store *redisstorage.Storage, registration *sso.Client, mode sso.Mode, center http.Handler) *sso.ClientApp {
	t.Helper()
	config := sso.DefaultClientConfig()
	config.ClientID, config.ClientSecret = registration.ClientID, registration.ClientSecret
	config.SecretKey, config.VerificationKey = registration.SecretKey, registration.VerificationKey
	config.Mode, config.ServerURL = mode, "https://center.example"
	config.HTTPClient = &http.Client{Transport: replicaHandlerTransport{handler: center}, Timeout: 5 * time.Second}
	config.ReplayProtection = f.replay(t, store, "callback-nonces:"+registration.ClientID+":")
	app, err := sso.NewClientAppChecked(config)
	if err != nil {
		t.Fatal(err)
	}
	return app
}

// replicaHandlerTransport uses real HTTP handlers without creating listening sockets. replicaHandlerTransport 使用实际 HTTP 处理器但不创建监听端口。
type replicaHandlerTransport struct {
	handler http.Handler
}

// RoundTrip dispatches one request to its selected handler. RoundTrip 将请求分发到所选处理器。
func (transport replicaHandlerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()
	transport.handler.ServeHTTP(w, r)
	return w.Result(), nil
}
