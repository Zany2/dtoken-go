package kratos

import (
	"errors"
	"fmt"
	"testing"

	derror "github.com/Zany2/dtoken-go/errors"
	kerrors "github.com/go-kratos/kratos/v2/errors"
	"google.golang.org/grpc/status"
)

// TestDefaultResponseHidesCause verifies transport messages are public while server-side error identity is preserved. TestDefaultResponseHidesCause 验证传输消息不泄露详情且服务端保留原始错误身份。
func TestDefaultResponseHidesCause(t *testing.T) {
	failure := fmt.Errorf("%w: private backend address", derror.ErrStorageUnavailable)
	result := writeErrorResponse(failure)
	if !errors.Is(result, failure) {
		t.Fatal("original error was lost")
	}
	if kerrors.FromError(result).Message != "internal server error" || status.Convert(result).Message() != "internal server error" {
		t.Fatalf("unsafe transport message: %v", result)
	}
}
