package core

import (
	"strconv"
	"strings"
	"testing"
)

// TestVersionIsSemanticVersion verifies the exported framework version uses three numeric components. TestVersionIsSemanticVersion 验证导出的框架版本由三个数字段组成。
func TestVersionIsSemanticVersion(t *testing.T) {
	parts := strings.Split(Version, ".")
	if len(parts) != 3 {
		t.Fatalf("Version = %q, want major.minor.patch", Version)
	}
	for index, part := range parts {
		if part == "" {
			t.Fatalf("Version = %q, component %d is empty", Version, index)
		}
		value, err := strconv.Atoi(part)
		if err != nil || value < 0 {
			t.Fatalf("Version = %q, component %d = %q is not a non-negative integer", Version, index, part)
		}
	}
}
