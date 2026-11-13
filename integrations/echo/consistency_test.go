package echo

import (
	"context"
	dtglobal "github.com/Zany2/dtoken-go/global"
	manager "github.com/Zany2/dtoken-go/internal/engine"
	echo4 "github.com/labstack/echo/v4"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestEchoRepeatedRegistration preserves request ownership unless explicitly overridden. TestEchoRepeatedRegistration 验证重复注册保留请求归属，除非显式覆盖。
func TestEchoRepeatedRegistration(t *testing.T) {
	dtglobal.DeleteAllManager()
	t.Cleanup(dtglobal.DeleteAllManager)
	global := newEchoReviewManager(t, "", nil)
	local := newEchoReviewManager(t, "echo-registration", nil)
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
			e := echo4.New()
			e.Use(RegisterDTokenContextMiddleware(context.Background(), WithManager(local)))
			e.Use(RegisterDTokenContextMiddleware(context.Background(), tc.opts...))
			called := false
			e.GET("/", func(c echo4.Context) error {
				called = true
				if got, err := GetManagerByContext(c); err != nil || got != tc.want {
					t.Errorf("manager=%p error=%v want=%p", got, err, tc.want)
				}
				return c.NoContent(http.StatusNoContent)
			})
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
			if !called || rec.Code != http.StatusNoContent {
				t.Fatalf("called=%v status=%d", called, rec.Code)
			}
		})
	}
}
