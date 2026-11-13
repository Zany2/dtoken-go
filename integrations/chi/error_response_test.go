package chi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	derror "github.com/Zany2/dtoken-go/errors"
)

// TestDefaultResponseHidesCause verifies the public response keeps codes but omits backend details. TestDefaultResponseHidesCause 验证公开响应保留错误码且不泄露后端详情。
func TestDefaultResponseHidesCause(t *testing.T) {
	failure := fmt.Errorf("%w: private backend address", derror.ErrStorageUnavailable)
	rec := httptest.NewRecorder()
	writeErrorResponse(rec, failure)
	statusCode, body := rec.Code, rec.Body.Bytes()
	var response struct {
		Code    int
		Message string
		Data    string
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatal(err)
	}
	if statusCode != http.StatusInternalServerError || response.Code != derror.CodeStorageError || response.Message != "internal server error" || response.Data != "internal server error" {
		t.Fatalf("unsafe response: status=%d body=%s", statusCode, body)
	}
}
