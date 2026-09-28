package gin_core_flow_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/derror"
	gincoreapp "github.com/Zany2/dtoken-go/tests/gin_core_app"
)

// TestDecodeFlowData rejects invalid payloads without preserving stale response fields. TestDecodeFlowData 拒绝非法数据，并避免残留旧响应字段。
func TestDecodeFlowData(t *testing.T) {
	type payload struct {
		Active bool   `json:"active"`
		UserID string `json:"userId"`
	}
	for _, raw := range []string{"", "  ", "null", " null ", "{", `{"active":false,"userId":123}`} {
		t.Run(raw, func(t *testing.T) {
			before := payload{Active: true, UserID: "previous-user"}
			got := before
			if err := decodeFlowData(json.RawMessage(raw), &got); err == nil {
				t.Fatalf("decodeFlowData(%q) succeeded, want error", raw)
			}
			if got != before {
				t.Fatalf("failed decode changed destination to %+v, want %+v", got, before)
			}
		})
	}

	// Omitted fields must reset on a later response. 后续响应缺省的字段必须清零。
	got := payload{Active: true, UserID: "previous-user"}
	if err := decodeFlowData(json.RawMessage(`{"userId":"current-user"}`), &got); err != nil {
		t.Fatal(err)
	}
	if got != (payload{UserID: "current-user"}) {
		t.Fatalf("decoded payload = %+v, want only current-user", got)
	}
	values := map[string]string{"stale": "previous"}
	if err := decodeFlowData(json.RawMessage(`{"current":"value"}`), &values); err != nil {
		t.Fatal(err)
	}
	if len(values) != 1 || values["current"] != "value" {
		t.Fatalf("decoded map retained stale entries: %v", values)
	}

	// Invalid destinations must return an error instead of panicking. 非法目标必须返回错误而非 panic。
	for _, dst := range []any{nil, payload{}, (*payload)(nil)} {
		if err := decodeFlowData(json.RawMessage(`{}`), dst); err == nil {
			t.Fatalf("decodeFlowData destination %T succeeded, want error", dst)
		}
	}
}

// TestFlowClientDefaultsToMemory keeps the default command useful without Redis. TestFlowClientDefaultsToMemory 验证未配置 Redis 时仍执行真实 HTTP 流程。
func TestFlowClientDefaultsToMemory(t *testing.T) {
	t.Setenv("DTOKEN_REDIS_URL", "")
	c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})
	token := c.login("memory-flow-user")
	var me struct {
		LoginID string `json:"loginId"`
	}
	c.expect("GET", "/api/me", nil, token, http.StatusOK, derror.CodeSuccess, &me)
	if me.LoginID != "memory-flow-user" {
		t.Fatalf("memory login ID = %q, want memory-flow-user", me.LoginID)
	}
}

// TestFlowStorageCleanup verifies real namespaced keys are removed without touching other prefixes. TestFlowStorageCleanup 验证清理真实命名空间中的键，且不影响其他前缀。
func TestFlowStorageCleanup(t *testing.T) {
	c := newFlowClient(t, gincoreapp.Config{TokenTimeout: 30 * time.Second, ActiveTimeout: -1})
	token := c.login("cleanup-user")
	userToken := c.multiAuthLogin("/multi-auth/user/login", "cleanup-user", "web", "browser-1")
	var generated struct {
		Nonce string `json:"nonce"`
	}
	c.expect("GET", "/nonce", nil, "", http.StatusOK, derror.CodeSuccess, &generated)
	if generated.Nonce == "" {
		t.Fatal("nonce is empty")
	}

	storage := c.app.Manager().GetStorage()
	scanner, ok := storage.(adapter.ScannerStorage)
	if !ok {
		t.Fatal("flow storage must support scoped cleanup")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	prefix := c.app.Manager().GetConfig().KeyPrefix
	keys, err := scanner.Keys(ctx, prefix+"*")
	if err != nil || len(keys) == 0 {
		t.Fatalf("scan populated flow prefix %q: keys=%v, error=%v", prefix, keys, err)
	}

	// Keep an unrelated key in the same backend to detect overbroad cleanup. 在同一后端保留无关键，检测清理范围过大。
	otherPrefix := flowKeyPrefix(t)
	if prefix == otherPrefix {
		t.Fatal("independent flow prefixes collided")
	}
	otherKey := otherPrefix + "sentinel"
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if err := storage.Delete(cleanupCtx, otherKey); err != nil {
			t.Errorf("remove unrelated sentinel: %v", err)
		}
	})
	if err := storage.Set(ctx, otherKey, "preserve", time.Minute); err != nil {
		t.Fatal(err)
	}
	clearFlowStorage(t, storage, prefix)
	keys, err = scanner.Keys(ctx, prefix+"*")
	if err != nil || len(keys) != 0 {
		t.Fatalf("keys after flow cleanup = %v, error=%v, want empty", keys, err)
	}
	if value, err := storage.Get(ctx, otherKey); err != nil || value == nil {
		t.Fatalf("unrelated key after cleanup = %v, error=%v, want preserved", value, err)
	}

	// Verify cleanup against ordinary auth, multi-auth, and nonce consumers. 通过普通认证、多认证体系和 nonce 接口验证清理结果。
	c.expect("GET", "/api/me", nil, token, http.StatusUnauthorized, derror.CodeNotLogin, nil)
	c.expect("GET", "/multi-auth/user/me", nil, userToken, http.StatusUnauthorized, derror.CodeNotLogin, nil)
	c.expect("POST", "/nonce/verify", map[string]any{"nonce": generated.Nonce}, "", http.StatusBadRequest, derror.CodeBadRequest, nil)
}
