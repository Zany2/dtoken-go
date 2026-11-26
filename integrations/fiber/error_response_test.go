package fiber

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	derror "github.com/Zany2/dtoken-go/errors"
	gofiber "github.com/gofiber/fiber/v2"
)

// TestDefaultResponseHidesCause verifies the public response keeps codes but omits backend details. TestDefaultResponseHidesCause 验证公开响应保留错误码且不泄露后端详情。
func TestDefaultResponseHidesCause(t *testing.T) {
	failure := fmt.Errorf("%w: private backend address", derror.ErrStorageUnavailable)
	app := gofiber.New()
	app.Get("/", func(c *gofiber.Ctx) error { return writeErrorResponse(c, failure) })
	res, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	statusCode := res.StatusCode
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
