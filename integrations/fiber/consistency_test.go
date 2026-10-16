package fiber

import (
	"context"
	"net/http"
	"testing"

	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/dtoken"
	gofiber "github.com/gofiber/fiber/v2"
)

// TestFiberRepeatedRegistration preserves request ownership unless explicitly overridden. TestFiberRepeatedRegistration 验证重复注册保留请求归属，除非显式覆盖。
func TestFiberRepeatedRegistration(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	global := newFiberReviewManager(t, "", nil)
	local := newFiberReviewManager(t, "fiber-registration", nil)
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
			app := gofiber.New()
			app.Use(RegisterDTokenContextMiddleware(context.Background(), WithManager(local)))
			app.Use(RegisterDTokenContextMiddleware(context.Background(), tc.opts...))
			called := false
			app.Get("/", func(c *gofiber.Ctx) error {
				called = true
				if got, err := GetManagerByContext(c); err != nil || got != tc.want {
					t.Errorf("manager=%p error=%v want=%p", got, err, tc.want)
				}
				return c.SendStatus(http.StatusNoContent)
			})
			raw := fiberReviewRequest(http.MethodGet, "/")
			app.Handler()(raw)
			if !called || raw.Response.StatusCode() != http.StatusNoContent {
				t.Fatalf("called=%v status=%d", called, raw.Response.StatusCode())
			}
		})
	}
}
