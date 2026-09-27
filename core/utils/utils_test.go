// @Author daixk 2026/05/15
package utils

import (
	"errors"
	"math"
	"strconv"
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
		{name: "signed minimum", value: int64(math.MinInt64), want: math.MinInt64},
		{name: "signed maximum", value: int64(math.MaxInt64), want: math.MaxInt64},
		{name: "unsigned maximum valid", value: uint64(math.MaxInt64), want: math.MaxInt64},
		{name: "minimum string", value: "-9223372036854775808", want: math.MinInt64},
		{name: "maximum string", value: "9223372036854775807", want: math.MaxInt64},
		{name: "minimum bytes", value: []byte("-9223372036854775808"), want: math.MinInt64},
		{name: "maximum bytes", value: []byte("9223372036854775807"), want: math.MaxInt64},
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

// TestToInt64UnsignedBoundaries verifies overflow rejection on both 32-bit and 64-bit platforms. TestToInt64UnsignedBoundaries 验证 32 位和 64 位平台的无符号溢出边界。
func TestToInt64UnsignedBoundaries(t *testing.T) {
	for _, value := range []uint64{uint64(math.MaxInt64) + 1, math.MaxUint64} {
		if got, err := ToInt64(value); err == nil || got != 0 {
			t.Fatalf("ToInt64(%d) = %d, %v, want zero and error", value, got, err)
		}
	}

	maxUint := ^uint(0)
	got, err := ToInt64(maxUint)
	if strconv.IntSize == 32 {
		if err != nil || got != math.MaxUint32 {
			t.Fatalf("ToInt64(max uint32) = %d, %v, want %d", got, err, uint64(math.MaxUint32))
		}
	} else {
		if err == nil || got != 0 {
			t.Fatalf("ToInt64(max uint64) = %d, %v, want zero and error", got, err)
		}
		maxValid := uint64(math.MaxInt64)
		if got, err := ToInt64(uint(maxValid)); err != nil || got != math.MaxInt64 {
			t.Fatalf("ToInt64(max valid uint) = %d, %v, want MaxInt64", got, err)
		}
		if got, err := ToInt64(uint(maxValid + 1)); err == nil || got != 0 {
			t.Fatalf("ToInt64(first overflowing uint) = %d, %v, want zero and error", got, err)
		}
	}
}

// TestToInt64RejectsOutOfRangeStrings verifies range errors survive wrapping. TestToInt64RejectsOutOfRangeStrings 验证字符串越界错误在包装后仍可识别。
func TestToInt64RejectsOutOfRangeStrings(t *testing.T) {
	for _, value := range []string{"-9223372036854775809", "9223372036854775808"} {
		for _, input := range []any{value, []byte(value)} {
			if got, err := ToInt64(input); got != 0 || !errors.Is(err, strconv.ErrRange) {
				t.Fatalf("ToInt64(%T(%q)) = %d, %v, want zero and ErrRange", input, input, got, err)
			}
		}
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
		math.Nextafter(float64(math.MinInt64), math.Inf(-1)),
		float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1)),
		float32(math.MaxInt64), math.Nextafter32(float32(math.MinInt64), float32(math.Inf(-1))),
	} {
		if got, err := ToInt64(value); err == nil || got != 0 {
			t.Fatalf("ToInt64(%v) = %d, %v, want zero and error", value, got, err)
		}
	}

	tests := []struct {
		value any
		want  int64
	}{
		{value: float64(math.MinInt64), want: math.MinInt64},
		{value: float32(math.MinInt64), want: math.MinInt64},
		// MaxInt64 rounds to 2^63 as a float64; its predecessor is 2^63 - 1024. MaxInt64 转为 float64 后舍入为 2^63，前一个浮点数为 2^63 - 1024。
		{value: math.Nextafter(float64(math.MaxInt64), 0), want: math.MaxInt64 - 1023},
		{value: float64(12.9), want: 12},
		{value: float64(-12.9), want: -12},
		{value: float32(12.9), want: 12},
		{value: float32(-12.9), want: -12},
		{value: float64(0.9), want: 0},
		{value: float64(-0.9), want: 0},
	}
	for _, tt := range tests {
		if got, err := ToInt64(tt.value); err != nil || got != tt.want {
			t.Fatalf("ToInt64(%T(%v)) = %d, %v, want %d", tt.value, tt.value, got, err, tt.want)
		}
	}
}
