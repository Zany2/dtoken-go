package derror

import (
	"errors"
	"strings"
	"testing"
)

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

func TestDTokenErrorWithoutWrappedError(t *testing.T) {
	err := NewDTokenError(CodeBadRequest, "bad request", nil)

	if got, want := err.Error(), "[400] bad request"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
	if err.Unwrap() != nil {
		t.Fatalf("Unwrap() = %v, want nil", err.Unwrap())
	}
}

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
