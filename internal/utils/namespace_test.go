package utils

import "testing"

// TestStorageNamespaceBoundaries verifies component boundaries and storage-version isolation. TestStorageNamespaceBoundaries 验证组件边界及存储版本隔离。
func TestStorageNamespaceBoundaries(t *testing.T) {
	if got := StorageNamespace("dtoken:", "auth:"); got != "dtoken:auth:v2:" {
		t.Fatalf("ordinary namespace = %q", got)
	}
	if got := LegacyStorageNamespace("dtoken:", "auth:"); got != "dtoken:auth:" {
		t.Fatalf("legacy namespace = %q", got)
	}
	seen := make(map[string]bool)
	for _, prefix := range []string{"app:", "app:admin:", `app\admin:`, `app\:admin:`} {
		for _, authType := range []string{"user:", "user:token:", `user\token:`, `user\:token:`} {
			key := StorageNamespace(prefix, authType)
			if seen[key] {
				t.Fatalf("namespace collision: %q", key)
			}
			seen[key] = true
		}
	}
	if StorageNamespace("app:", "admin:user:") == StorageNamespace("app:admin:", "user:") {
		t.Fatal("moving a component between prefix and auth type preserved the key")
	}
}
