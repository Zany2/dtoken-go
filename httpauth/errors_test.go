package httpauth

import (
	"errors"
	"fmt"
	"testing"

	derror "github.com/Zany2/dtoken-go/errors"
)

// TestPublicErrorsHideInfrastructureDetails covers raw, wrapped and typed failures. TestPublicErrorsHideInfrastructureDetails 覆盖原始、包装及带业务码的基础设施故障。
func TestPublicErrorsHideInfrastructureDetails(t *testing.T) {
	for _, tc := range []struct {
		name    string
		err     error
		code    int
		message string
	}{
		{"unknown", errors.New("private backend address"), derror.CodeServerError, "internal server error"},
		{"storage", fmt.Errorf("%w: private backend address", derror.ErrStorageUnavailable), derror.CodeStorageError, "internal server error"},
		{"codec", fmt.Errorf("%w: private payload", derror.ErrSerializeFailed), derror.CodeStorageError, "internal server error"},
		{"typed server", derror.NewDTokenError(derror.CodeServerError, "private backend address", nil), derror.CodeServerError, "internal server error"},
		{"typed storage", derror.NewDTokenError(derror.CodeStorageError, "private payload", nil), derror.CodeStorageError, "internal server error"},
		{"wrapped business", fmt.Errorf("private detail: %w", derror.ErrInvalidToken), derror.CodeTokenInvalid, derror.ErrInvalidToken.Error()},
		{"custom business", derror.NewDTokenError(derror.CodeBadRequest, "public validation message", errors.New("private detail")), derror.CodeBadRequest, "public validation message"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code, message := GetErrorCodeAndMessage(tc.err)
			if code != tc.code || message != tc.message {
				t.Fatalf("response = %d %q, want %d %q", code, message, tc.code, tc.message)
			}
		})
	}
}
