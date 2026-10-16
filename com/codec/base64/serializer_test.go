// @Author daixk 2025/12/22 15:56:00
package base64

import (
	"encoding/base64"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// TestBase64SerializerName verifies the serializer name 测试序列化器名称
func TestBase64SerializerName(t *testing.T) {
	s := NewBase64Serializer()
	if s == nil {
		t.Fatal("NewBase64Serializer() returned nil")
	}
	if got := s.Name(); got != "base64" {
		t.Fatalf("Name() = %q, want %q", got, "base64")
	}
}

// TestBase64SerializerDecodeBufferLifetime checks parallel decoding, capacity boundaries and retained JSON values. TestBase64SerializerDecodeBufferLifetime 验证并发解码、容量边界和 JSON 结果的独立生命周期。
func TestBase64SerializerDecodeBufferLifetime(t *testing.T) {
	s := NewBase64Serializer()
	for _, tc := range []struct {
		name string
		size int
	}{
		{"empty_string", 0},
		{"below_initial_capacity", 4096 - 3},
		{"above_initial_capacity", 4096},
		{"below_retained_capacity", 65536 - 3},
		{"oversized", 128 * 1024},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			want := strings.Repeat("x", tc.size)
			data, err := s.Encode(want)
			if err != nil {
				t.Fatal(err)
			}
			var retained json.RawMessage
			if err := s.Decode(data, &retained); err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 8; i++ {
				var got string
				if err := s.Decode(data, &got); err != nil || got != want {
					t.Fatalf("Decode() error=%v, decoded length=%d, want length=%d", err, len(got), len(want))
				}

				// A smaller value must not include bytes left by an earlier payload. 较小载荷不得混入先前载荷留下的字节。
				if err := s.Decode([]byte("e30="), &map[string]any{}); err != nil {
					t.Fatal(err)
				}
			}
			if string(retained) != `"`+want+`"` {
				t.Fatal("later decodes changed the retained JSON value")
			}
		})
	}
}

// BenchmarkBase64SerializerDecode compares pooled scratch space with the previous allocation path. BenchmarkBase64SerializerDecode 比较临时缓冲池与原有逐次分配路径。
func BenchmarkBase64SerializerDecode(b *testing.B) {
	s := NewBase64Serializer()
	for _, payload := range []struct {
		name string
		size int
	}{
		{"1KB", 1024},
		{"4KB", 4096},
		{"16KB", 16 * 1024},
		{"128KB", 128 * 1024},
	} {
		data, err := s.Encode(strings.Repeat("x", payload.size))
		if err != nil {
			b.Fatal(err)
		}
		for _, variant := range []struct {
			name   string
			decode func([]byte, any) error
		}{
			{"pooled", s.Decode},
			{"unpooled", func(data []byte, v any) error {
				buffer := make([]byte, base64.StdEncoding.DecodedLen(len(data)))
				n, err := base64.StdEncoding.Decode(buffer, data)
				if err != nil {
					return err
				}
				return json.Unmarshal(buffer[:n], v)
			}},
		} {
			b.Run(payload.name+"/"+variant.name, func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(data)))
				for b.Loop() {
					var got string
					if err := variant.decode(data, &got); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

// TestBase64SerializerRoundTrip verifies JSON plus Base64 round trip 测试 JSON 与 Base64 的往返编解码
func TestBase64SerializerRoundTrip(t *testing.T) {
	type payload struct {
		Name string
		Age  int
	}

	s := NewBase64Serializer()
	input := payload{Name: "alice", Age: 18}

	data, err := s.Encode(input)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	if _, err = base64.StdEncoding.DecodeString(string(data)); err != nil {
		t.Fatalf("Encode() returned invalid base64: %v", err)
	}

	var got payload
	if err = s.Decode(data, &got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if !reflect.DeepEqual(got, input) {
		t.Fatalf("Decode() = %+v, want %+v", got, input)
	}
}

// TestBase64SerializerErrors verifies unsupported input and malformed payload errors 测试不支持输入与非法载荷错误
func TestBase64SerializerErrors(t *testing.T) {
	s := NewBase64Serializer()

	if _, err := s.Encode(make(chan int)); err == nil {
		t.Fatal("Encode() should fail for unsupported JSON type")
	}

	var out map[string]any
	if err := s.Decode([]byte("not-base64"), &out); err == nil {
		t.Fatal("Decode() should fail for malformed base64")
	}

	badJSON := []byte(base64.StdEncoding.EncodeToString([]byte("{bad-json}")))
	if err := s.Decode(badJSON, &out); err == nil {
		t.Fatal("Decode() should fail for malformed JSON")
	}
}

// TestBase64SerializerNilAndInvalidTargets verifies null values and invalid decode targets. TestBase64SerializerNilAndInvalidTargets 验证 null 值和非法解码目标。
func TestBase64SerializerNilAndInvalidTargets(t *testing.T) {
	s := NewBase64Serializer()

	data, err := s.Encode(nil)
	if err != nil {
		t.Fatalf("Encode(nil) error = %v", err)
	}
	if string(data) != base64.StdEncoding.EncodeToString([]byte("null")) {
		t.Fatalf("Encode(nil) = %q, want base64-encoded null", data)
	}

	var got any
	if err := s.Decode(data, &got); err != nil {
		t.Fatalf("Decode(null) error = %v", err)
	}
	if got != nil {
		t.Fatalf("Decode(null) = %#v, want nil", got)
	}

	if err := s.Decode(data, map[string]string{}); err == nil {
		t.Fatal("Decode() should reject a non-pointer target")
	}
	var nilTarget *struct{}
	if err := s.Decode(data, nilTarget); err == nil {
		t.Fatal("Decode() should reject a nil pointer target")
	}
}

// TestBase64SerializerDecodeBoundaries verifies the decoded payload contains exactly one complete JSON value. TestBase64SerializerDecodeBoundaries 验证解码后的载荷必须恰好包含一个完整 JSON 值。
func TestBase64SerializerDecodeBoundaries(t *testing.T) {
	s := NewBase64Serializer()
	for _, tt := range []struct {
		name string
		json string
	}{
		{"empty", ""},
		{"truncated", `{"name":`},
		{"second value", `{"name":"alice"}{"name":"bob"}`},
		{"invalid suffix", `{"name":"alice"}!`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			data := []byte(base64.StdEncoding.EncodeToString([]byte(tt.json)))
			var got map[string]string
			if err := s.Decode(data, &got); err == nil {
				t.Fatal("Decode() accepted incomplete or trailing JSON data")
			}
		})
	}

	data := []byte(base64.StdEncoding.EncodeToString([]byte(" \n{\"name\":\"alice\"}\t")))
	var got map[string]string
	if err := s.Decode(data, &got); err != nil {
		t.Fatalf("Decode() rejected trailing JSON whitespace: %v", err)
	}
	if got["name"] != "alice" {
		t.Fatalf("Decode() name = %q, want %q", got["name"], "alice")
	}
}
