package beego

import (
	"context"
	"errors"
	"net/http"
	"testing"

	derror "github.com/Zany2/dtoken-go/errors"
	manager "github.com/Zany2/dtoken-go/internal/engine"
	"github.com/Zany2/dtoken-go/storage/memory"
	beegocontext "github.com/beego/beego/v2/server/web/context"
)

// TestBeegoSessionFacades verifies request binding, context propagation, and unavailable-instance errors. TestBeegoSessionFacades 验证请求实例绑定、上下文传播及实例不可用时的错误。
func TestBeegoSessionFacades(t *testing.T) {
	for _, check := range []struct {
		name string
		get  func(*beegocontext.Context) (*manager.Session, error)
	}{
		{"account", GetSessionByContext},
		{"token", GetSessionByTokenByContext},
	} {
		t.Run(check.name, func(t *testing.T) {
			if session, err := check.get(nil); session != nil || !errors.Is(err, derror.ErrNotLogin) {
				t.Fatalf("nil context: session=%v err=%v", session, err)
			}
			unbound, _ := newBeegoBehaviorContext(http.MethodGet, "/session", "", "")
			if session, err := check.get(unbound); session != nil || !errors.Is(err, derror.ErrManagerNotFound) {
				t.Fatalf("unbound context: session=%v err=%v", session, err)
			}

			storage := &beegoReviewStorage{Storage: memory.NewStorage()}
			mgr := newBeegoReviewManager(t, "beego-session", storage)
			token, err := mgr.Login(context.Background(), "session-user")
			if err != nil {
				t.Fatal(err)
			}
			c, _ := newBeegoBehaviorContext(http.MethodGet, "/session", mgr.GetConfig().TokenName, token)
			request, cancel := context.WithCancel(context.Background())
			defer cancel()
			c.Request = c.Request.WithContext(request)
			RegisterDTokenContextMiddleware(context.Background(), WithManager(mgr))(c)
			reads := 0
			storage.inspect = func(got context.Context) {
				reads++
				if got != request {
					t.Error("current request context was lost")
				}
			}
			session, err := check.get(c)
			if err != nil {
				t.Fatal(err)
			}
			if session == nil || session.LoginID != "session-user" || session.AuthType != mgr.GetConfig().AuthType || reads == 0 {
				t.Fatalf("request session=%v storage reads=%d", session, reads)
			}

			cancel()
			if _, err := check.get(c); !errors.Is(err, derror.ErrStorageUnavailable) {
				t.Fatalf("canceled request: %v", err)
			}
			mgr.CloseManager()
			if session, err := check.get(c); session != nil || !errors.Is(err, derror.ErrManagerNotFound) {
				t.Fatalf("closed manager: session=%v err=%v", session, err)
			}
		})
	}
}
