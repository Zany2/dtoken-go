package jsoncodec

import "testing"

// TestCodecRoundTrip verifies JSON codec name, encode, and decode behavior. TestCodecRoundTrip 验证 JSON 编解码器名称、编码和解码行为。
func TestCodecRoundTrip(t *testing.T) {
	codec := Codec{}
	if codec.Name() != "json" {
		t.Fatalf("Name() = %q, want json", codec.Name())
	}

	raw, err := codec.Encode(map[string]any{"loginId": "user-1", "active": true})
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	var decoded struct {
		LoginID string `json:"loginId"`
		Active  bool   `json:"active"`
	}
	if err = codec.Decode(raw, &decoded); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if decoded.LoginID != "user-1" || !decoded.Active {
		t.Fatalf("decoded = %+v, want user-1 active", decoded)
	}
}

// TestCodecDecodeInvalidJSON verifies invalid JSON returns an error. TestCodecDecodeInvalidJSON 验证非法 JSON 会返回错误。
func TestCodecDecodeInvalidJSON(t *testing.T) {
	var decoded map[string]any
	if err := (Codec{}).Decode([]byte("{"), &decoded); err == nil {
		t.Fatal("Decode(invalid) error = nil, want error")
	}
}
