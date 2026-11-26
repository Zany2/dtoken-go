package nonce

import "testing"

// TestStorageNamespaceConstruction preserves key encoding and boundaries across independently constructed instances. TestStorageNamespaceConstruction 保持独立构造实例的键编码与命名空间边界。
func TestStorageNamespaceConstruction(t *testing.T) {
	for _, input := range []struct{ prefix, authType, want string }{
		{"app", "user", "app:user:v2:"},
		{"app:", "user:", "app:user:v2:"},
		{"app:", "admin:user:", `app:admin\:user:v2:`},
		{"app:admin:", "user:", `app\:admin:user:v2:`},
		{`app\blue:`, `user\admin:`, `app\\blue:user\\admin:v2:`},
		{"应用:", "用户:", "应用:用户:v2:"},
	} {
		t.Run(input.prefix+"/"+input.authType, func(t *testing.T) {
			manager, err := NewDefaultNonceManager(input.authType, input.prefix, newNonceTestStorage())
			if err != nil {
				t.Fatal(err)
			}
			// Credential text remains a literal suffix, including separators. 凭据文本保持字面后缀，包括其中的分隔符。
			value := `value:with\separator`
			for _, key := range []struct{ got, suffix string }{
				{manager.getNonceKey(value), NonceKeySuffix},
			} {
				if key.got != input.want+key.suffix+value {
					t.Fatalf("key = %q, want %q", key.got, input.want+key.suffix+value)
				}
			}
		})
	}
}
