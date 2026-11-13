package httpauth

import (
	"errors"
	"testing"

	derror "github.com/Zany2/dtoken-go/errors"
	"github.com/Zany2/dtoken-go/global"
	corecontext "github.com/Zany2/dtoken-go/httpauth/context"
	manager "github.com/Zany2/dtoken-go/internal/engine"
)

// TestResolutionRequiresExplicitRegistry prevents global state from authorizing an unbound request. TestResolutionRequiresExplicitRegistry 防止全局状态为未绑定的请求提供认证实例。
func TestResolutionRequiresExplicitRegistry(t *testing.T) {
	global.DeleteAllManager()
	t.Cleanup(global.DeleteAllManager)
	mgr := newAuthcheckTestManager(t)
	global.SetManager(mgr)
	var typedNil *corecontext.DTokenContext
	for _, cached := range []any{nil, typedNil} {
		if got, err := ResolveManagerFromContext("", cached); got != nil || !errors.Is(err, derror.ErrManagerNotFound) {
			t.Fatalf("implicit lookup = %p, %v", got, err)
		}
		if got, err := ResolveManagerFromContext("", cached, global.GetManager); err != nil || got != mgr {
			t.Fatalf("explicit lookup = %p, %v", got, err)
		}
	}
	if got, err := ResolveManager(nil, ""); got != nil || !errors.Is(err, derror.ErrManagerNotFound) {
		t.Fatalf("nil explicit manager = %p, %v", got, err)
	}
}

// TestScopedRegistrySelection preserves local ownership and explicit tenant routing. TestScopedRegistrySelection 保留本地实例归属并支持显式租户路由。
func TestScopedRegistrySelection(t *testing.T) {
	local := newAuthcheckTestManager(t)
	selected := newAuthcheckTestManager(t)
	calls := 0
	resolver := func(authType ...string) (*manager.Manager, error) {
		calls++
		if len(authType) != 1 || authType[0] != "selected" {
			return nil, derror.ErrManagerNotFound
		}
		return selected, nil
	}
	scope := corecontext.NewContext(nil, local).WithManagerResolver(resolver)
	if got, err := ResolveManagerFromContext("", scope, resolver); err != nil || got != local || calls != 0 {
		t.Fatalf("local scope = %p, %v, resolver calls = %d", got, err, calls)
	}
	if got, err := ResolveManagerFromContext("selected", scope); err != nil || got != selected || calls != 1 {
		t.Fatalf("selected scope = %p, %v, resolver calls = %d", got, err, calls)
	}
	local.CloseManager()
	if got, err := ResolveManagerFromContext("", scope, resolver); got != nil || !errors.Is(err, derror.ErrManagerNotFound) || calls != 1 {
		t.Fatalf("closed scope = %p, %v, resolver calls = %d", got, err, calls)
	}
	selected.CloseManager()
	if got, err := ResolveManagerFromContext("selected", nil, resolver); got != nil || !errors.Is(err, derror.ErrManagerNotFound) {
		t.Fatalf("closed resolver result = %p, %v", got, err)
	}
}
