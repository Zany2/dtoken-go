package chi

import (
	"context"
	dtglobal "github.com/Zany2/dtoken-go/global"
	manager "github.com/Zany2/dtoken-go/internal/engine"
	"github.com/Zany2/dtoken-go/nonce"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestChiRepeatedRegistration preserves request ownership unless explicitly overridden. TestChiRepeatedRegistration 验证重复注册保留请求归属，除非显式覆盖。
func TestChiRepeatedRegistration(t *testing.T) {
	dtglobal.DeleteAllManager()
	t.Cleanup(dtglobal.DeleteAllManager)
	global := newChiReviewManager(t, "")
	local := newChiReviewManager(t, "chi-registration")
	dtglobal.SetManager(global)
	for _, tc := range []struct {
		name string
		opts []AuthOption
		want *manager.Manager
	}{
		{"inherit", nil, local},
		{"auth_type", []AuthOption{WithManagerResolver(dtglobal.GetManager), WithAuthType(global.GetConfig().AuthType)}, global},
		{"explicit", []AuthOption{WithManager(local), WithAuthType("missing")}, local},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				if got, err := GetManagerByCtx(r.Context()); err != nil || got != tc.want {
					t.Errorf("manager=%p error=%v want=%p", got, err, tc.want)
				}
				w.WriteHeader(http.StatusNoContent)
			})
			chain := RegisterDTokenContextMiddleware(WithManager(local))(
				RegisterDTokenContextMiddleware(tc.opts...)(handler),
			)
			rec := httptest.NewRecorder()
			chain.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
			if !called || rec.Code != http.StatusNoContent {
				t.Fatalf("called=%v status=%d", called, rec.Code)
			}
		})
	}
}

// TestChiRequestRebindingPreservesCapabilities keeps service bindings across adapter replacement. TestChiRequestRebindingPreservesCapabilities 在请求适配器替换后保留服务绑定。
func TestChiRequestRebindingPreservesCapabilities(t *testing.T) {
	mgr := newChiReviewManager(t, "chi-services")
	service := nonce.NewService(nonce.NewDefaultNonceManager(mgr.GetConfig().AuthType, mgr.GetConfig().KeyPrefix, mgr.GetStorage()), nil)
	token, err := mgr.Login(context.Background(), "user")
	if err != nil {
		t.Fatal(err)
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scope, ok := GetDTokenContext(r)
		if !ok {
			t.Fatal("request scope missing")
		}
		value, err := scope.Nonce().Generate(r.Context())
		if err != nil || !service.VerifyNonce(r.Context(), value) {
			t.Errorf("nonce binding lost: %q, %v", value, err)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	chain := RegisterDTokenContextMiddleware(WithManager(mgr), WithCapabilities(Capabilities{Nonce: service}))(
		AuthMiddleware()(handler),
	)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set(mgr.GetConfig().TokenName, token)
	response := httptest.NewRecorder()
	chain.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d", response.Code)
	}
}
