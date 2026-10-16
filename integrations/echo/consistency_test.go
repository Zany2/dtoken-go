package echo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/dtoken"
	echo4 "github.com/labstack/echo/v4"
)

// TestEchoRepeatedRegistration preserves request ownership unless explicitly overridden. TestEchoRepeatedRegistration 验证重复注册保留请求归属，除非显式覆盖。
func TestEchoRepeatedRegistration(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	global := newEchoReviewManager(t, "", nil)
	local := newEchoReviewManager(t, "echo-registration", nil)
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
