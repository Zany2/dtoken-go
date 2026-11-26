package echo

import (
	"errors"
	"testing"

	"github.com/Zany2/dtoken-go"
	derror "github.com/Zany2/dtoken-go/errors"
)

// TestInstanceBindingReturnsErrors verifies nil and closed instances return errors before middleware registration. TestInstanceBindingReturnsErrors 验证空实例和已关闭实例在中间件注册前返回错误。
func TestInstanceBindingReturnsErrors(t *testing.T) {
	auth, err := dtoken.NewBuilder().IsPrintBanner(false).AutoRenew(false).BuildInstance()
	if err != nil {
		t.Fatal(err)
	}
	defer auth.Close()
	if option, err := WithAuth(auth); err != nil || option == nil {
		t.Fatalf("WithAuth(open) = (%v, %v)", option == nil, err)
	}
	if handler, err := Context(auth); err != nil || handler == nil {
		t.Fatalf("Context(open) = (%v, %v)", handler == nil, err)
	}
	auth.Close()
	for _, invalid := range []*dtoken.Auth{nil, auth} {
		if option, err := WithAuth(invalid); option != nil || !errors.Is(err, derror.ErrManagerNotFound) {
			t.Fatalf("WithAuth(invalid) returned option=%v, error=%v", option != nil, err)
		}
		if handler, err := Context(invalid); handler != nil || !errors.Is(err, derror.ErrManagerNotFound) {
			t.Fatalf("Context(invalid) returned handler=%v, error=%v", handler != nil, err)
		}
	}
}
