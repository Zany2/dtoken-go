package chi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/dtoken"
)

// TestChiRepeatedRegistration preserves request ownership unless explicitly overridden. TestChiRepeatedRegistration 验证重复注册保留请求归属，除非显式覆盖。
func TestChiRepeatedRegistration(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	global := newChiReviewManager(t, "")
	local := newChiReviewManager(t, "chi-registration")
	dtoken.SetManager(global)
	for _, tc := range []struct {
		name string
		opts []AuthOption
		want *manager.Manager
	}{
		{"inherit", nil, local},
		{"auth_type", []AuthOption{WithAuthType(global.GetConfig().AuthType)}, global},
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
