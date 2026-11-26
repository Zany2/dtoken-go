package context_test

import (
	"context"
	"errors"
	"testing"
	"time"

	noncecontract "github.com/Zany2/dtoken-go/contract/nonce"
	derror "github.com/Zany2/dtoken-go/errors"
	requestcontext "github.com/Zany2/dtoken-go/httpauth/context"
)

// externalNonce demonstrates a service supplied without importing the bundled implementation. externalNonce 演示无需导入内置实现即可提供服务。
type externalNonce struct{ calls int }

var _ noncecontract.Service = (*externalNonce)(nil)

func (s *externalNonce) GenerateNonce(context.Context) (string, error) {
	s.calls++
	return "external", nil
}
func (s *externalNonce) GenerateNonceWithTimeout(ctx context.Context, _ time.Duration) (string, error) {
	return s.GenerateNonce(ctx)
}
func (*externalNonce) VerifyNonce(_ context.Context, value string) bool { return value == "external" }
func (*externalNonce) VerifyAndConsumeNonce(_ context.Context, value string) error {
	if value != "external" {
		return derror.ErrInvalidNonce
	}
	return nil
}
func (*externalNonce) IsNonceValid(_ context.Context, value string) bool  { return value == "external" }
func (*externalNonce) GetNonceTTL(context.Context, string) (int64, error) { return 60, nil }

// TestExternalCapabilityBinding verifies custom implementations and typed nil bindings. TestExternalCapabilityBinding 验证自定义实现与带类型空值绑定。
func TestExternalCapabilityBinding(t *testing.T) {
	service := &externalNonce{}
	scope := requestcontext.NewContext(nil, nil).WithCapabilities(&requestcontext.Capabilities{Nonce: service})
	value, err := scope.Nonce().Generate(context.Background())
	if err != nil || value != "external" || service.calls != 1 {
		t.Fatalf("custom service = %q, %v; calls = %d", value, err, service.calls)
	}
	var absent *externalNonce
	scope.WithCapabilities(&requestcontext.Capabilities{Nonce: absent})
	if _, err := scope.Nonce().Generate(context.Background()); !errors.Is(err, derror.ErrModuleNotEnabled) {
		t.Fatalf("typed nil service = %v", err)
	}
}
