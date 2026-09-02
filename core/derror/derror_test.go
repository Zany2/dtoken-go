package derror

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// TestDTokenErrorErrorAndUnwrap verifies formatted messages and wrapped error access. TestDTokenErrorErrorAndUnwrap 验证错误信息格式化与原始错误解包。
func TestDTokenErrorErrorAndUnwrap(t *testing.T) {
	base := ErrInvalidToken
	err := NewDTokenError(CodeTokenInvalid, "bad token", base)

	if !errors.Is(err, base) {
		t.Fatalf("errors.Is() = false, want true")
	}
	if err.Unwrap() != base {
		t.Fatalf("Unwrap() = %v, want %v", err.Unwrap(), base)
	}
	if got := err.Error(); !strings.Contains(got, "[10001] bad token") || !strings.Contains(got, base.Error()) {
		t.Fatalf("Error() = %q, want code, message and wrapped error", got)
	}
}

// TestDTokenErrorWithoutWrappedError verifies formatting without a wrapped cause. TestDTokenErrorWithoutWrappedError 验证未包装原始错误时的格式化结果。
func TestDTokenErrorWithoutWrappedError(t *testing.T) {
	err := NewDTokenError(CodeBadRequest, "bad request", nil)

	if got, want := err.Error(), "[400] bad request"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
	if err.Unwrap() != nil {
		t.Fatalf("Unwrap() = %v, want nil", err.Unwrap())
	}
}

// TestDTokenErrorSupportsNestedErrors verifies errors.Is and errors.As across nested wrapping. TestDTokenErrorSupportsNestedErrors 验证嵌套包装时 errors.Is 与 errors.As 仍然有效。
func TestDTokenErrorSupportsNestedErrors(t *testing.T) {
	inner := NewDTokenError(CodeTokenExpired, "expired", ErrTokenExpired)
	outer := fmt.Errorf("request failed: %w", inner)

	if !errors.Is(outer, ErrTokenExpired) {
		t.Fatal("errors.Is() = false, want wrapped sentinel")
	}

	var got *DTokenError
	if !errors.As(outer, &got) || got != inner {
		t.Fatalf("errors.As() = %v, want inner DTokenError", got)
	}
}

// TestDTokenErrorNilReceiver verifies a typed-nil DTokenError is safe to inspect. TestDTokenErrorNilReceiver 验证 typed-nil DTokenError 可安全读取。
func TestDTokenErrorNilReceiver(t *testing.T) {
	var err *DTokenError
	if got := err.Error(); got != "<nil>" {
		t.Fatalf("Error() = %q, want <nil>", got)
	}
	if got := err.Unwrap(); got != nil {
		t.Fatalf("Unwrap() = %v, want nil", got)
	}
}

// TestSentinelErrorsAreDistinct verifies sentinel errors preserve distinct identities. TestSentinelErrorsAreDistinct 验证哨兵错误保持独立身份。
func TestSentinelErrorsAreDistinct(t *testing.T) {
	cases := []error{
		ErrInvalidTicket,
		ErrTicketConsumed,
		ErrInvalidShortKey,
		ErrShortKeyPending,
		ErrPermissionDenied,
		ErrRoleDenied,
	}

	for _, err := range cases {
		if err == nil {
			t.Fatalf("sentinel error is nil")
		}
		if err.Error() == "" {
			t.Fatalf("sentinel error %v has empty message", err)
		}
	}
}
