package sso_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/Zany2/dtoken-go/sso"
	"github.com/Zany2/dtoken-go/sso/httpserver"
)

// ExampleHTTPErrorFunc records a bounded classification through the public HTTP facade without logging a request or raw error. ExampleHTTPErrorFunc 通过公开 HTTP 门面记录固定分类，不记录请求或原始错误。
func ExampleHTTPErrorFunc() {
	server := sso.NewServer()
	defer server.Close()
	options := httpserver.Options{
		OnHTTPError: httpserver.HTTPErrorFunc(func(_ *http.Request, status int, err error) {
			fmt.Println(status, errors.Is(err, sso.ErrMethodNotAllowed))
		}),
	}
	handler := httpserver.New(server, options)
	w := httptest.NewRecorder()
	handler.HandleAuthorize(w, httptest.NewRequest(http.MethodPost, "/sso/authorize", nil))
	fmt.Println(w.Code, w.Header().Get("Cache-Control"))
	// Output:
	// 405 true
	// 405 no-store
}
