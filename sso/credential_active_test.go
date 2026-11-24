package sso

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// TestCredentialActiveHook checks all credential modes and distinguishes inactive application state from lookup errors. TestCredentialActiveHook 检查全部凭证模式，并区分应用状态失效与查询故障。
func TestCredentialActiveHook(t *testing.T) {
	for _, mode := range []Mode{ModeTicket, ModeSharedToken, ModeRemoteSession, ModeOAuth2} {
		t.Run(string(mode), func(t *testing.T) {
			server := NewServer()
			defer server.Close()
			client := newTestClient()
			client.Modes = []Mode{mode}
			if err := server.RegisterClient(client); err != nil {
				t.Fatal(err)
			}
			active, calls := true, 0
			var checkErr error
			options := HTTPOptions{
				LoginIDResolver: func(*http.Request) (string, bool) { return "alice", true },
				AuthorizationExtra: func(*http.Request, string, string) (map[string]any, error) {
					return map[string]any{"centerSessionID": "session-ref"}, nil
				},
				CheckCredentialActive: func(r *http.Request, info CredentialInfo) (bool, error) {
					calls++
					if !info.Active || info.Mode != mode || info.LoginID != "alice" || info.ClientID != client.ClientID || info.Extra["centerSessionID"] != "session-ref" || len(info.Scopes) != 1 || info.Scopes[0] != "profile" {
						t.Fatalf("unvalidated hook metadata: %+v", info)
					}
					if r.URL.Path != "/sso/introspect" && r.URL.Path != "/sso/userinfo" {
						t.Fatalf("unexpected hook path: %s", r.URL.Path)
					}
					return active, checkErr
				},
			}
			h := NewHTTPServer(server, options)
			w := httptest.NewRecorder()
			query := url.Values{"client": {client.ClientID}, "redirect": {client.RedirectURIs[0]}, "mode": {string(mode)}, "scope": {"profile"}}
			h.HandleAuthorize(w, httptest.NewRequest(http.MethodGet, "/sso/authorize?"+query.Encode(), nil))
			if w.Code != http.StatusFound {
				t.Fatalf("authorize: %d %s", w.Code, w.Body.String())
			}
			location, err := url.Parse(w.Header().Get("Location"))
			if err != nil {
				t.Fatal(err)
			}
			values := location.Query()
			values.Set("client", client.ClientID)
			values.Set("clientSecret", client.ClientSecret)
			values.Set("mode", string(mode))
			post := func(path string, v url.Values) *httptest.ResponseRecorder {
				t.Helper()
				out := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(v.Encode()))
				r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				h.Handler().ServeHTTP(out, r)
				return out
			}
			decode := func(w *httptest.ResponseRecorder) CredentialInfo {
				t.Helper()
				var result struct {
					Data CredentialInfo `json:"data"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
					t.Fatal(err)
				}
				return result.Data
			}
			for _, path := range []string{"/sso/introspect", "/sso/userinfo"} {
				active, checkErr = true, nil
				before := calls
				w = post(path, values)
				info := decode(w)
				if w.Code != http.StatusOK || !info.Active || info.LoginID != "alice" || calls != before+1 {
					t.Fatalf("active: %d %+v calls=%d", w.Code, info, calls)
				}

				active = false
				w = post(path, values)
				if path == "/sso/introspect" {
					info = decode(w)
					if w.Code != http.StatusOK || info.Active || info.LoginID != "" || info.ClientID != "" || len(info.Extra) != 0 || len(info.Scopes) != 0 {
						t.Fatalf("inactive disclosed identity: %d %+v", w.Code, info)
					}
				} else if w.Code != http.StatusUnauthorized {
					t.Fatalf("inactive userinfo: %d", w.Code)
				}

				for _, err := range []error{errors.New("session lookup failed"), ErrStorageUnavailable} {
					checkErr = err
					w = post(path, values)
					if w.Code != http.StatusInternalServerError {
						t.Fatalf("lookup failure hidden: %d %s", w.Code, w.Body.String())
					}
					var failure Response
					if err := json.Unmarshal(w.Body.Bytes(), &failure); err != nil || failure.Data != nil {
						t.Fatalf("lookup failure returned partial data: %s", w.Body.String())
					}
				}
			}

			// Invalid callers and credentials never reach the application check. 无效调用方与凭证不能触发应用检查。
			active, checkErr = true, nil
			before := calls
			bad := cloneValues(values)
			bad.Set("clientSecret", "wrong")
			if w = post("/sso/introspect", bad); w.Code != http.StatusUnauthorized || calls != before {
				t.Fatal("unauthenticated request reached hook")
			}
			other := *client
			other.ClientID = "other-client"
			if err := server.RegisterClient(&other); err != nil {
				t.Fatal(err)
			}
			bad = cloneValues(values)
			bad.Set("client", other.ClientID)
			w = post("/sso/introspect", bad)
			if w.Code != http.StatusOK || decode(w).Active || calls != before {
				t.Fatal("foreign credential reached hook")
			}

			// Omitting the optional hook preserves storage-only validation and never consumes credentials. 未配置可选钩子时保留仅校验存储的行为，且不消费凭证。
			options.CheckCredentialActive = nil
			h = NewHTTPServer(server, options)
			active = false
			w = post("/sso/introspect", values)
			if w.Code != http.StatusOK || !decode(w).Active || calls != before {
				t.Fatal("nil hook changed existing behavior")
			}
			options.CheckCredentialActive = func(*http.Request, CredentialInfo) (bool, error) { calls++; return true, nil }
			h = NewHTTPServer(server, options)
			w = post("/sso/revoke", values)
			if w.Code != http.StatusOK || calls != before {
				t.Fatalf("revocation should not depend on hook: %d", w.Code)
			}
			w = post("/sso/introspect", values)
			if w.Code != http.StatusOK || decode(w).Active || calls != before {
				t.Fatal("hook reactivated revoked credential")
			}
		})
	}
}
