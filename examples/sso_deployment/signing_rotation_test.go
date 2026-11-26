package main

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Zany2/dtoken-go/sso"
)

// TestDeploymentSigningRotation verifies login and logout while center and application use different active keys. TestDeploymentSigningRotation 在中心和子应用使用不同主密钥时验证登录与注销。
func TestDeploymentSigningRotation(t *testing.T) {
	newKey := strings.Repeat("n", 32)
	clientEnv := deploymentEnvironment(t, "client", "alpha")
	clientEnv["SSO_VERIFICATION_KEY"] = newKey
	clientConfig, err := loadConfig("client", func(key string) string { return clientEnv[key] })
	if err != nil {
		t.Fatal(err)
	}
	centerEnv := deploymentEnvironment(t, "center", "")
	var clients []sso.Client
	if err := json.Unmarshal([]byte(centerEnv["SSO_CLIENTS_JSON"]), &clients); err != nil {
		t.Fatal(err)
	}
	clients[0].VerificationKey, clients[0].SecretKey = clients[0].SecretKey, newKey
	payload, err := json.Marshal(clients)
	if err != nil {
		t.Fatal(err)
	}
	centerEnv["SSO_CLIENTS_JSON"] = string(payload)
	centerConfig, err := loadConfig("center", func(key string) string { return centerEnv[key] })
	if err != nil {
		t.Fatal(err)
	}
	storage := sso.NewMemoryStorage()
	transport := deploymentTransport{}
	center, err := newApplication(centerConfig, storage, transport)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(center.close)
	alpha, err := newApplication(clientConfig, storage, transport)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(alpha.close)
	transport["localhost:9200"], transport["localhost:9201"] = center.handler, alpha.handler
	cookie, _ := loginThroughReplicas(t, alpha, alpha, center, nil)
	result, err := center.centerHTTP.LogoutAccount(httptest.NewRequest("POST", "/trusted-logout", nil), "alice")
	if err != nil || result == nil || !result.Complete {
		t.Fatalf("rotated logout: %+v %v", result, err)
	}
	r := httptest.NewRequest("GET", clientConfig.publicURL+"/protected", nil)
	r.AddCookie(cookie)
	if info, err := alpha.session(r); err != nil || (info != nil && info.Active) {
		t.Fatal("rotated callback did not revoke local session")
	}
}

// TestDeploymentSigningRotationConfiguration validates optional keys without disclosing their values. TestDeploymentSigningRotationConfiguration 校验可选密钥且不披露其值。
func TestDeploymentSigningRotationConfiguration(t *testing.T) {
	for _, role := range []string{"center", "client"} {
		env := deploymentEnvironment(t, role, "alpha")
		if role == "client" {
			env["SSO_VERIFICATION_KEY"] = "private-short-key"
		} else {
			var clients []sso.Client
			if err := json.Unmarshal([]byte(env["SSO_CLIENTS_JSON"]), &clients); err != nil {
				t.Fatal(err)
			}
			clients[0].VerificationKey = "private-short-key"
			payload, err := json.Marshal(clients)
			if err != nil {
				t.Fatal(err)
			}
			env["SSO_CLIENTS_JSON"] = string(payload)
		}
		if _, err := loadConfig(role, func(key string) string { return env[key] }); err == nil || strings.Contains(err.Error(), "private-short-key") {
			t.Fatalf("invalid key configuration: %v", err)
		}
	}
}
