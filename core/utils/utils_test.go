// @Author daixk 2026/05/15
package utils

import "testing"

// TestToInt64ParsesStoredNumberTypes verifies storage-friendly int parsing TestToInt64ParsesStoredNumberTypes 验证存储返回值可解析为 int64
func TestToInt64ParsesStoredNumberTypes(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  int64
	}{
		{name: "int64", value: int64(123), want: 123},
		{name: "string", value: "123", want: 123},
		{name: "trimmed string", value: " 123 ", want: 123},
		{name: "bytes", value: []byte("123"), want: 123},
		{name: "bool", value: true, want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToInt64(tt.value)
			if err != nil {
				t.Fatalf("ToInt64() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("ToInt64() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestToInt64RejectsInvalidStrings verifies strict decimal parsing TestToInt64RejectsInvalidStrings 验证严格十进制解析
func TestToInt64RejectsInvalidStrings(t *testing.T) {
	tests := []any{"", "123abc", []byte("123abc")}
	for _, tt := range tests {
		if _, err := ToInt64(tt); err == nil {
			t.Fatalf("ToInt64(%v) error = nil, want parse error", tt)
		}
	}
}
