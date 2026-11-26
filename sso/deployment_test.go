package sso

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type deploymentTestBackend struct{} // Deployment validation must never invoke session operations. 部署校验不得调用会话操作。

// Resolve rejects unexpected backend access during static validation. Resolve 拒绝静态校验期间意外访问后端。
func (deploymentTestBackend) Resolve(context.Context, string) (*SessionInfo, error) {
	panic("unexpected session lookup")
}

// Create rejects unexpected session creation during static validation. Create 拒绝静态校验期间意外创建会话。
func (deploymentTestBackend) Create(context.Context, SessionCreate) (*SessionInfo, error) {
	panic("unexpected session creation")
}

// Revoke rejects unexpected session changes during static validation. Revoke 拒绝静态校验期间意外变更会话。
func (deploymentTestBackend) Revoke(context.Context, SessionRevoke) error {
	panic("unexpected session revocation")
}

// TestDeploymentConfig validates topologies independently of low-level credential modes. TestDeploymentConfig 独立于底层凭证模式校验部署结构。
func TestDeploymentConfig(t *testing.T) {
	_, app, _ := directExchangeTestClient(t, ModeTicket, false)
	for _, mode := range []Deployment{DeploymentSharedCookie, DeploymentSharedStore, DeploymentHTTP} {
		t.Run(string(mode), func(t *testing.T) {
			cfg := DeploymentConfig{Deployment: mode, Client: app.config, Sessions: deploymentTestBackend{}}
			if mode != DeploymentSharedStore {
				cfg.Client.ExchangeService = nil
			}
			if err := cfg.Validate(); err != nil {
				t.Fatal(err)
			}
			missing := cfg
			missing.Sessions = nil
			if err := missing.Validate(); !errors.Is(err, ErrSSOConfig) {
				t.Fatalf("missing backend: %v", err)
			}
			wrong := cfg
			wrong.Client.Mode = ModeOAuth2
			if err := wrong.Validate(); !errors.Is(err, ErrModeUnsupported) {
				t.Fatalf("credential mode: %v", err)
			}
			if mode == DeploymentSharedStore {
				cfg.Client.UseHTTPCheck = true
			} else {
				cfg.Client.ExchangeService = app.config.ExchangeService
			}
			if err := cfg.Validate(); !errors.Is(err, ErrSSOConfig) {
				t.Fatalf("conflict: %v", err)
			}
		})
	}
	missingLocal := DeploymentConfig{Deployment: DeploymentSharedStore, Client: app.config, Sessions: deploymentTestBackend{}}
	missingLocal.Client.ExchangeService = nil
	if err := missingLocal.Validate(); !errors.Is(err, ErrSSOConfig) {
		t.Fatalf("missing local exchange: %v", err)
	}
	if err := (DeploymentConfig{}).Validate(); !errors.Is(err, ErrModeUnsupported) {
		t.Fatalf("zero deployment: %v", err)
	}
}

// TestSessionPrivateTokensAreNotJSON prevents backend credentials entering protocol serialization. TestSessionPrivateTokensAreNotJSON 防止后端凭据进入协议序列化。
func TestSessionPrivateTokensAreNotJSON(t *testing.T) {
	for _, value := range []any{
		SessionInfo{LoginID: "alice", Token: "private-bearer-token"},
		SessionRevoke{Scope: RevokeSessionToken, Token: "private-bearer-token"},
	} {
		payload, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(payload), "private-bearer-token") {
			t.Fatal("private credential serialized")
		}
	}
}
