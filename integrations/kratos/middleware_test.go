// @Author daixk 2026/06/07
package kratos

import (
	"errors"
	"testing"

	"github.com/Zany2/dtoken-go/core/derror"
)

// TestAuthMiddlewareUsesTokenExpiredLoginError verifies auth failure semantics TestAuthMiddlewareUsesTokenExpiredLoginError 验证认证失败语义
func TestAuthMiddlewareUsesTokenExpiredLoginError(t *testing.T) {
	if err := authMiddlewareLoginError(); !errors.Is(err, derror.ErrTokenExpired) {
		t.Fatalf("authMiddlewareLoginError() = %v, want ErrTokenExpired", err)
	}
}
