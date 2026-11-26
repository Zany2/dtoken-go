package acceptance

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

const acceptanceLogoutSuccess = `{"code":0,"data":{"result":"ok","stage":"complete","notifications":{"complete":true,"clients":[{"clientId":"first-app","status":"delivered"},{"clientId":"second-app","status":"delivered"}]}}}`

// TestAcceptanceLogoutReports rejects incomplete or misattributed success reports. TestAcceptanceLogoutReports 拒绝不完整或客户端归属不符的成功报告。
func TestAcceptanceLogoutReports(t *testing.T) {
	for _, tc := range []struct {
		name, from, to string
		valid          bool
	}{
		{name: "complete", valid: true},
		{name: "business failure", from: `"code":0`, to: `"code":1`},
		{name: "missing code", from: `"code":0,`},
		{name: "missing result", from: `"result":"ok",`},
		{name: "partial stage", from: `"stage":"complete"`, to: `"stage":"cleanup"`},
		{name: "partial notifications", from: `"complete":true`, to: `"complete":false`},
		{name: "wrong application", from: `second-app`, to: `private-unrelated-client`},
		{name: "duplicate application", from: `second-app`, to: `first-app`},
		{name: "empty identity", from: `second-app`, to: ``},
		{name: "skipped application", from: `"status":"delivered"`, to: `"status":"skipped"`},
		{name: "failed application", from: `"status":"delivered"`, to: `"status":"failed"`},
		{name: "malformed", from: acceptanceLogoutSuccess, to: `{"private-detail":`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload := acceptanceLogoutSuccess
			if tc.from != "" {
				payload = strings.Replace(payload, tc.from, tc.to, 1)
			}
			err := acceptanceLogoutComplete(&acceptanceResponse{status: http.StatusOK, body: []byte(payload)}, []string{"first-app", "second-app"})
			if (err == nil) != tc.valid {
				t.Fatalf("report validity=%v, want %v", err == nil, tc.valid)
			}
			if err != nil && strings.Contains(err.Error(), "private-") {
				t.Fatal("report diagnostics exposed response data")
			}
		})
	}
	for _, response := range []*acceptanceResponse{nil, {status: http.StatusUnauthorized}, {status: http.StatusInternalServerError, body: []byte(acceptanceLogoutSuccess)}} {
		if acceptanceLogoutComplete(response, nil) == nil {
			t.Fatal("missing or failed HTTP response acknowledged cleanup")
		}
	}
	response := &acceptanceResponse{status: http.StatusOK, body: []byte(acceptanceLogoutSuccess)}
	if err := acceptanceLogoutComplete(response, []string{"first-app"}); err != nil {
		t.Fatal("additional delivered clients should be allowed")
	}
	if acceptanceLogoutComplete(response, []string{""}) == nil {
		t.Fatal("unknown expected identity accepted")
	}
}

// TestAcceptanceLogoutCleanup checks both cleanup hops without contacting a service. TestAcceptanceLogoutCleanup 不访问服务即可检查清理的两次请求。
func TestAcceptanceLogoutCleanup(t *testing.T) {
	for _, scenario := range []string{"complete", "unauthorized", "partial", "wrong redirect", "transport"} {
		t.Run(scenario, func(t *testing.T) {
			c, err := newAcceptanceClient("https://app.example", "https://center.example")
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			c.http.Transport = acceptanceTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					if r.Method != http.MethodPost || r.URL.String() != "https://app.example/logout-all" || r.Header.Get("Origin") != "https://app.example" {
						t.Fatal("cleanup did not start with an application logout request")
					}
					if scenario == "transport" {
						return nil, errors.New("private-transport-detail")
					}
					location := "https://center.example/sso/logout?sign=private-signature"
					if scenario == "wrong redirect" {
						location = "https://outside.example/sso/logout?sign=private-signature"
					}
					return &http.Response{StatusCode: http.StatusSeeOther, Header: http.Header{"Location": {location}}, Body: io.NopCloser(strings.NewReader(""))}, nil
				}
				if calls != 2 || r.Method != http.MethodGet || r.URL.String() != "https://center.example/sso/logout?sign=private-signature" {
					t.Fatal("cleanup followed an unexpected destination or repeated a request")
				}
				status, payload := http.StatusOK, acceptanceLogoutSuccess
				if scenario == "unauthorized" {
					status = http.StatusUnauthorized
				}
				if scenario == "partial" {
					payload = strings.Replace(payload, `"complete":true`, `"complete":false`, 1)
				}
				return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(payload))}, nil
			})
			err = c.cleanupLogout("https://app.example", "https://center.example", []string{"first-app", "second-app"})
			if (err == nil) != (scenario == "complete") {
				t.Fatalf("cleanup confirmation=%v for %s", err == nil, scenario)
			}
			if err != nil && strings.Contains(err.Error(), "private-") {
				t.Fatal("cleanup diagnostics exposed private data")
			}
			wantCalls := 2
			if scenario == "transport" || scenario == "wrong redirect" {
				wantCalls = 1
			}
			if calls != wantCalls {
				t.Fatalf("cleanup requests=%d, want %d", calls, wantCalls)
			}
		})
	}
}
