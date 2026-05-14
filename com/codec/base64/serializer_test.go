package base64

import (
	"encoding/base64"
	"reflect"
	"testing"
)

// TestBase64SerializerName verifies the serializer name 测试序列化器名称
func TestBase64SerializerName(t *testing.T) {
	s := NewBase64Serializer()
	if got := s.Name(); got != "base64" {
		t.Fatalf("Name() = %q, want %q", got, "base64")
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
