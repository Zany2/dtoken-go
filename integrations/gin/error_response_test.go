package gin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	dtoken "github.com/Zany2/dtoken-go"
	derror "github.com/Zany2/dtoken-go/errors"
	"github.com/gin-gonic/gin"
)

// TestDefaultResponseHidesCause verifies the public response keeps codes but omits backend details. TestDefaultResponseHidesCause 验证公开响应保留错误码且不泄露后端详情。
func TestDefaultResponseHidesCause(t *testing.T) {
	failure := fmt.Errorf("%w: private backend address", derror.ErrStorageUnavailable)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	writeErrorResponse(c, failure)
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

// TestFailureCallbackRetainsCause verifies applications can log the original error without reaching handlers. TestFailureCallbackRetainsCause 验证应用能记录原始错误且请求不会进入处理器。
func TestFailureCallbackRetainsCause(t *testing.T) {
	failure := errors.New("private backend address")
	called := false
	handled := false
	router := gin.New()
	router.Use(RegisterDTokenContextMiddleware(context.Background(), WithManagerResolver(func(...string) (*dtoken.Manager, error) { return nil, failure }), WithFailFunc(func(c *gin.Context, err error) {
		called = true
		if !errors.Is(err, failure) {
			t.Errorf("callback error = %v", err)
		}
		c.Status(http.StatusInternalServerError)
	})))
	router.GET("/", func(*gin.Context) { handled = true })
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	if !called || handled {
		t.Fatalf("callback=%t handler=%t", called, handled)
	}
}
