//go:build goexperiment.jsonv2

// @Author daixk 2025/12/22 15:56:00

package jsonv2

import (
	"reflect"
	"testing"
	"time"
)

// TestJSONV2SerializerName verifies the serializer name. 测试 JSON v2 编解码器名称。
func TestJSONV2SerializerName(t *testing.T) {
	serializer := NewJSONV2Serializer()
	if serializer == nil {
		t.Fatal("NewJSONV2Serializer() returned nil")
	}
	if got := serializer.Name(); got != "jsonv2" {
		t.Fatalf("Name() = %q, want %q", got, "jsonv2")
	}
}

// TestJSONV2SerializerRoundTrip verifies struct, map, and slice round trips. 测试结构体、映射和切片往返编解码。
func TestJSONV2SerializerRoundTrip(t *testing.T) {
	type payload struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}

	serializer := NewJSONV2Serializer()
	tests := []struct {
		name  string
		input any
		check func(t *testing.T, data []byte)
	}{
		{
			name:  "struct",
			input: payload{Name: "alice", Count: 2},
			check: func(t *testing.T, data []byte) {
				var got payload
				if err := serializer.Decode(data, &got); err != nil {
					t.Fatalf("Decode() error = %v", err)
				}
				want := payload{Name: "alice", Count: 2}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("Decode() = %+v, want %+v", got, want)
				}
			},
		},
		{
			name:  "map",
			input: map[string]int{"score": 95},
			check: func(t *testing.T, data []byte) {
				var got map[string]int
				if err := serializer.Decode(data, &got); err != nil {
					t.Fatalf("Decode() error = %v", err)
				}
				want := map[string]int{"score": 95}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("Decode() = %#v, want %#v", got, want)
				}
			},
		},
		{
			name:  "slice",
			input: []int{1, 2, 3},
			check: func(t *testing.T, data []byte) {
				var got []int
				if err := serializer.Decode(data, &got); err != nil {
					t.Fatalf("Decode() error = %v", err)
				}
				want := []int{1, 2, 3}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("Decode() = %#v, want %#v", got, want)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := serializer.Encode(tt.input)
			if err != nil {
				t.Fatalf("Encode() error = %v", err)
			}
			if len(data) == 0 {
				t.Fatal("Encode() returned empty data")
			}
			tt.check(t, data)
		})
	}
}

// TestJSONV2SerializerErrors verifies unsupported values and invalid JSON. 测试不支持的值和非法 JSON。
func TestJSONV2SerializerErrors(t *testing.T) {
	serializer := NewJSONV2Serializer()

	if _, err := serializer.Encode(make(chan int)); err == nil {
		t.Fatal("Encode() should reject channels")
	}

	var target struct{}
	if err := serializer.Decode([]byte(`{"invalid":}`), &target); err == nil {
		t.Fatal("Decode() should reject malformed JSON")
	}
}

// TestJSONV2SerializerNilAndInvalidTargets verifies nil and target validation. 测试 nil 值和目标值校验。
func TestJSONV2SerializerNilAndInvalidTargets(t *testing.T) {
	serializer := NewJSONV2Serializer()

	data, err := serializer.Encode(nil)
	if err != nil {
		t.Fatalf("Encode(nil) error = %v", err)
	}
	if string(data) != "null" {
		t.Fatalf("Encode(nil) = %q, want %q", data, "null")
	}

	var got any
	if err := serializer.Decode(data, &got); err != nil {
		t.Fatalf("Decode(null) error = %v", err)
	}
	if got != nil {
		t.Fatalf("Decode(null) = %#v, want nil", got)
	}

	if err := serializer.Decode([]byte(`{"name":"alice"}`), map[string]string{}); err == nil {
		t.Fatal("Decode() should reject a non-pointer target")
	}
	var nilTarget *struct{}
	if err := serializer.Decode([]byte(`{}`), nilTarget); err == nil {
		t.Fatal("Decode() should reject a nil pointer target")
	}
}

// TestJSONV2SerializerDurationCompatibility verifies durations retain the numeric nanosecond representation. TestJSONV2SerializerDurationCompatibility 验证时长保留纳秒数值表示。
func TestJSONV2SerializerDurationCompatibility(t *testing.T) {
	type payload struct {
		Timeout time.Duration `json:"timeout"`
	}

	serializer := NewJSONV2Serializer()
	data, err := serializer.Encode(payload{Timeout: 1500 * time.Millisecond})
	if err != nil {
		t.Fatalf("Encode() duration error = %v", err)
	}
	if string(data) != `{"timeout":1500000000}` {
		t.Fatalf("Encode() duration = %q, want %q", data, `{"timeout":1500000000}`)
	}

	var got payload
	if err := serializer.Decode(data, &got); err != nil {
		t.Fatalf("Decode() duration error = %v", err)
	}
	if got.Timeout != 1500*time.Millisecond {
		t.Fatalf("Decode() duration = %v, want %v", got.Timeout, 1500*time.Millisecond)
	}
}

// TestJSONV2SerializerDecodeBoundaries verifies exactly one complete JSON value is required. TestJSONV2SerializerDecodeBoundaries 验证输入必须恰好包含一个完整 JSON 值。
func TestJSONV2SerializerDecodeBoundaries(t *testing.T) {
	serializer := NewJSONV2Serializer()
	for _, tt := range []struct {
		name string
		data string
	}{
		{"empty", ""},
		{"truncated", `{"name":`},
		{"second value", `{"name":"alice"}{"name":"bob"}`},
		{"invalid suffix", `{"name":"alice"}!`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var got map[string]string
			if err := serializer.Decode([]byte(tt.data), &got); err == nil {
				t.Fatal("Decode() accepted incomplete or trailing data")
			}
		})
	}

	var got map[string]string
	if err := serializer.Decode([]byte(" \n{\"name\":\"alice\"}\t"), &got); err != nil {
		t.Fatalf("Decode() rejected trailing whitespace: %v", err)
	}
	if got["name"] != "alice" {
		t.Fatalf("Decode() name = %q, want %q", got["name"], "alice")
	}
}
