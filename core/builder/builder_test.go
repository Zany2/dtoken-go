package builder

import "testing"

// TestBuildReturnsErrorForInvalidConfig verifies Build returns error instead of panic 测试 Build 在配置无效时返回错误而不是 panic
func TestBuildReturnsErrorForInvalidConfig(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Build should return error instead of panic: %v", r)
		}
	}()

	// Use invalid token name to trigger config validation 使用无效 Token 名称触发配置校验
	mgr, err := NewBuilder().TokenName("").Build()
	if err == nil {
		t.Fatal("Build should return config error")
	}
	if mgr != nil {
		t.Fatal("Build should not return manager when config is invalid")
	}
}
