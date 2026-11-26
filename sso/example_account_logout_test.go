package sso_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	dtoken "github.com/Zany2/dtoken-go"
	"github.com/Zany2/dtoken-go/sso"
)

// ExampleHTTPServer_LogoutAccount composes application-owned Auth revocation with trusted SSO account logout. ExampleHTTPServer_LogoutAccount 将应用持有的 Auth 撤销与可信 SSO 账号下线组合。
func ExampleHTTPServer_LogoutAccount() {
	auth, err := dtoken.NewBuilder().IsPrintBanner(false).IsLog(false).Build()
	if err != nil {
		panic(err)
	}
	defer auth.Close()
	server := sso.NewServer()
	defer server.Close()
	token, err := auth.Login(context.Background(), dtoken.LoginOptions{LoginID: "alice"})
	if err != nil {
		panic(err)
	}
	options := sso.DefaultHTTPOptions()
	options.SecretKey = "example-only-replace-with-managed-signing-key"
	options.LoginIDResolver = func(*http.Request) (string, bool) { return "", false }
	options.CenterLogout = func(r *http.Request, loginID string) error {
		return auth.LogoutByLoginID(r.Context(), loginID)
	}
	handler, err := sso.NewHTTPServerChecked(server, options)
	if err != nil {
		panic(err)
	}

	// The trusted application has already authorized this account; disable login/issuance separately for permanent suspension. 可信应用已对目标账号授权；永久封禁还需另行禁用登录和签发。
	result, err := handler.LogoutAccount(httptest.NewRequest("POST", "/trusted-operation", nil), "alice")
	if err != nil {
		panic(err)
	}
	info, err := auth.IntrospectToken(context.Background(), token)
	if err != nil {
		panic(err)
	}
	fmt.Println(result.Complete, info.Active)
	// Output: true false
}
