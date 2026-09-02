// @Author daixk 2026/05/15
package utils

import (
	"math"
	"testing"
)

// TestToBytesConvertsSupportedTypes verifies every supported byte conversion. TestToBytesConvertsSupportedTypes 验证所有受支持的字节转换类型。
func TestToBytesConvertsSupportedTypes(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{name: "string", value: "hello", want: "hello"},
		{name: "bytes", value: []byte("hello"), want: "hello"},
		{name: "byte", value: byte('x'), want: "x"},
		{name: "rune", value: rune('Z'), want: "Z"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToBytes(tt.value)
			if err != nil {
				t.Fatalf("ToBytes() error = %v", err)
			}
			if string(got) != tt.want {
				t.Fatalf("ToBytes() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestToBytesRejectsUnsupportedTypes verifies unsupported values return an error. TestToBytesRejectsUnsupportedTypes 验证不支持的值会返回错误。
func TestToBytesRejectsUnsupportedTypes(t *testing.T) {
	for _, value := range []any{nil, 123, true, struct{}{}} {
		if got, err := ToBytes(value); err == nil || got != nil {
			t.Fatalf("ToBytes(%T) = %q, %v, want nil and error", value, got, err)
		}
	}
}

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

// TestToInt64RejectsNonFiniteAndOverflowingFloats verifies float safety boundaries. TestToInt64RejectsNonFiniteAndOverflowingFloats 验证浮点数非有限和溢出边界。
func TestToInt64RejectsNonFiniteAndOverflowingFloats(t *testing.T) {
	for _, value := range []any{
		math.NaN(), math.Inf(1), math.Inf(-1),
		float64(math.MaxInt64), float64(math.MinInt64) * 2,
	} {
		if got, err := ToInt64(value); err == nil || got != 0 {
			t.Fatalf("ToInt64(%v) = %d, %v, want zero and error", value, got, err)
		}
	}

	if got, err := ToInt64(float64(math.MinInt64)); err != nil || got != math.MinInt64 {
		t.Fatalf("ToInt64(MinInt64 float) = %d, %v, want MinInt64", got, err)
	}
	if got, err := ToInt64(float64(12.9)); err != nil || got != 12 {
		t.Fatalf("ToInt64(12.9) = %d, %v, want 12", got, err)
	}
}
