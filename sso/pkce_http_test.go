package sso

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// TestPKCEHTTPIdentityEndpoints blocks proof bypasses through token, userinfo, and introspect before allowing valid consumption. TestPKCEHTTPIdentityEndpoints 阻止从 token、userinfo 和 introspect 绕过证明，并允许合法消费。
func TestPKCEHTTPIdentityEndpoints(t *testing.T) {
	for _, mode := range []Mode{ModeTicket, ModeOAuth2} {
		t.Run(string(mode), func(t *testing.T) {
			s := NewServer()
			defer s.Close()
			client := newTestClient()
			client.Modes, client.RequirePKCE = []Mode{mode}, true
			if err := s.RegisterClient(client); err != nil {
				t.Fatal(err)
			}
			verifier := NewPKCEVerifier()
			challenge, _ := PKCEChallenge(verifier)
			value, err := issuePKCETestCredential(s, client, mode, OneTimeCredentialOptions{CodeChallenge: challenge, CodeChallengeMethod: PKCEMethodS256})
			if err != nil {
				t.Fatal(err)
			}
			h := NewHTTPServer(s, HTTPOptions{ServerOptions: ServerOptions{EnableSLO: true}})
			key := map[Mode]string{ModeTicket: "ticket", ModeOAuth2: "code"}[mode]
			values := url.Values{"client": {client.ClientID}, "clientSecret": {client.ClientSecret}, "redirect": {client.RedirectURIs[0]}, "mode": {string(mode)}, key: {value}, "callback": {"https://app.example.com/logout"}}
			for _, handler := range []http.HandlerFunc{h.HandleToken, h.HandleUserInfo, h.HandleIntrospect} {
				for _, proof := range [][]string{nil, {""}, {NewPKCEVerifier()}, {verifier, verifier}} {
					invalid := cloneValues(values)
					if proof != nil {
						invalid["code_verifier"] = proof
					}
					w := protocolReviewPost(handler, invalid)
					if w.Code != http.StatusBadRequest || strings.Contains(w.Body.String(), "alice") || strings.Contains(w.Body.String(), verifier) {
						t.Fatalf("proof rejection: %d %s", w.Code, w.Body.String())
					}
				}
				queryRequest := httptest.NewRequest("POST", "/?code_verifier="+verifier, strings.NewReader(values.Encode()))
				queryRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				out := httptest.NewRecorder()
				handler(out, queryRequest)
				if out.Code != http.StatusBadRequest {
					t.Fatalf("accepted proof in query: %d", out.Code)
				}
			}
			sessions, err := s.GetClientSessions(context.Background(), "alice")
			if err != nil || len(sessions) != 0 {
				t.Fatalf("failed proof registered logout: %v %v", sessions, err)
			}
			values.Set("code_verifier", verifier)
			for _, handler := range []http.HandlerFunc{h.HandleUserInfo, h.HandleIntrospect, h.HandleToken} {
				out := protocolReviewPost(handler, values)
				if out.Code != http.StatusOK || !strings.Contains(out.Body.String(), "alice") || strings.Contains(out.Body.String(), verifier) || strings.Contains(out.Body.String(), challenge) {
					t.Fatalf("valid proof: %d %s", out.Code, out.Body.String())
				}
			}
			sessions, err = s.GetClientSessions(context.Background(), "alice")
			if err != nil || len(sessions) != 1 {
				t.Fatalf("successful exchange did not bind logout: %v %v", sessions, err)
			}
			if out := protocolReviewPost(h.HandleToken, values); out.Code == http.StatusOK {
				t.Fatal("repeated proof reused a consumed credential")
			}
		})
	}
}

// TestPKCEAuthorizationPolicy rejects malformed challenges before trusted hooks or credential issuance. TestPKCEAuthorizationPolicy 在可信钩子及凭证签发前拒绝非法挑战。
func TestPKCEAuthorizationPolicy(t *testing.T) {
	s := NewServer()
	defer s.Close()
	client := newTestClient()
	client.Modes = []Mode{ModeTicket, ModeOAuth2, ModeSharedToken, ModeRemoteSession}
	client.RequirePKCE = true
	if err := s.RegisterClient(client); err != nil {
		t.Fatal(err)
	}
	hookCalls := 0
	h := NewHTTPServer(s, HTTPOptions{
		LoginIDResolver: func(*http.Request) (string, bool) { return "alice", true },
		AuthorizationExtra: func(*http.Request, string, string) (map[string]any, error) {
			hookCalls++
			return nil, nil
		},
	})
	verifier := NewPKCEVerifier()
	challenge, _ := PKCEChallenge(verifier)
	for _, mode := range []Mode{ModeTicket, ModeOAuth2} {
		for _, extra := range []url.Values{
			{}, {"code_challenge": {challenge}}, {"code_challenge_method": {PKCEMethodS256}},
			{"code_challenge": {challenge}, "code_challenge_method": {"plain"}},
			{"code_challenge": {challenge, challenge}, "code_challenge_method": {PKCEMethodS256}},
			{"code_challenge": {challenge}, "code_challenge_method": {PKCEMethodS256}, "code_verifier": {verifier}},
		} {
			extra.Set("client", client.ClientID)
			extra.Set("redirect", client.RedirectURIs[0])
			extra.Set("mode", string(mode))
			out := httptest.NewRecorder()
			h.HandleAuthorize(out, httptest.NewRequest("GET", "/sso/authorize?"+extra.Encode(), nil))
			if out.Code != http.StatusBadRequest || hookCalls != 0 || out.Header().Get("Location") != "" {
				t.Fatalf("invalid authorization: %d hook calls %d", out.Code, hookCalls)
			}
		}
	}
	for _, mode := range []Mode{ModeSharedToken, ModeRemoteSession} {
		values := url.Values{"client": {client.ClientID}, "redirect": {client.RedirectURIs[0]}, "mode": {string(mode)}, "code_challenge": {challenge}, "code_challenge_method": {PKCEMethodS256}}
		out := httptest.NewRecorder()
		h.HandleAuthorize(out, httptest.NewRequest("GET", "/sso/authorize?"+values.Encode(), nil))
		if out.Code != http.StatusBadRequest || hookCalls != 0 {
			t.Fatal("PKCE silently ignored in reusable mode")
		}
		values.Del("code_challenge")
		values.Del("code_challenge_method")
		out = httptest.NewRecorder()
		h.HandleAuthorize(out, httptest.NewRequest("GET", "/sso/authorize?"+values.Encode(), nil))
		if out.Code != http.StatusFound {
			t.Fatalf("reusable mode blocked by one-time policy: %d", out.Code)
		}
		hookCalls = 0
	}
}

// TestPKCEClientURLSafety verifies proof privacy, parameter customization, signing, and collision rejection. TestPKCEClientURLSafety 验证证明隐私、参数定制、签名与冲突拒绝。
func TestPKCEClientURLSafety(t *testing.T) {
	params := normalizeParamNames(ParamNames{CodeChallenge: "proof_hash", CodeChallengeMethod: "proof_method", CodeVerifier: "proof"})
	app := NewClientApp(ClientConfig{ClientID: "app", ServerURL: "https://center.example", CheckSign: true, SecretKey: "key", Params: params})
	verifier := NewPKCEVerifier()
	challenge, _ := PKCEChallenge(verifier)
	raw, err := app.AuthURLWithPKCE("https://app.example/callback", verifier, nil)
	if err != nil {
		t.Fatal(err)
	}
	target, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if target.Query().Get(params.CodeChallenge) != challenge || strings.Contains(raw, verifier) || !NewSignerWithParams("key", params).Verify(target.Query()) {
		t.Fatal("challenge or signature mismatch; verifier must remain private")
	}
	if _, err := app.AuthURL("https://app.example/callback", url.Values{params.CodeVerifier: {verifier}}); err == nil {
		t.Fatal("verifier exposed in authorize URL")
	}
	if _, err := app.ExchangeTicketURL("ticket", url.Values{params.CodeVerifier: {verifier}}); err == nil {
		t.Fatal("verifier exposed in exchange URL")
	}
	params.CodeVerifier = params.Ticket
	app = NewClientApp(ClientConfig{ClientID: "app", ServerURL: "https://center.example", Params: params})
	if _, err := app.AuthURL("https://app.example/callback", nil); err == nil {
		t.Fatal("colliding proof parameter accepted")
	}
	h := NewHTTPServer(NewServer(), HTTPOptions{ServerOptions: ServerOptions{Params: params}})
	defer h.server.Close()
	if h.initializationError() == nil {
		t.Fatal("server accepted colliding proof parameter")
	}
}
