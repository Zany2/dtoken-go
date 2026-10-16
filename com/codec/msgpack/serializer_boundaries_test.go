package msgpack

import (
	"reflect"
	"testing"
)

// TestDecodeRejectsTypedNilTargets verifies dependency fast paths cannot panic on invalid targets. TestDecodeRejectsTypedNilTargets 验证依赖快速路径不会因非法目标触发 panic。
func TestDecodeRejectsTypedNilTargets(t *testing.T) {
	s := NewMsgPackSerializer()
	for _, tt := range []struct {
		name   string
		value  any
		target any
	}{
		{"nil", nil, nil},
		{"non-pointer", "value", "target"},
		{"string slice", []string{"role"}, (*[]string)(nil)},
		{"nil string slice", nil, (*[]string)(nil)},
		{"string map", map[string]string{"key": "value"}, (*map[string]string)(nil)},
		{"nil string map", nil, (*map[string]string)(nil)},
		{"interface map", map[string]any{"key": "value"}, (*map[string]any)(nil)},
		{"nil interface map", nil, (*map[string]any)(nil)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			data, err := s.Encode(tt.value)
			if err != nil {
				t.Fatal(err)
			}
			if err := s.Decode(data, tt.target); err == nil {
				t.Fatal("Decode accepted an invalid target")
			}
		})
	}
}

// TestDecodeRequiresOneCompleteValue verifies truncated and concatenated records are rejected. TestDecodeRequiresOneCompleteValue 验证截断或拼接记录被拒绝。
func TestDecodeRequiresOneCompleteValue(t *testing.T) {
	s := NewMsgPackSerializer()
	valid, err := s.Encode("record")
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name string
		data []byte
	}{
		{"empty", nil},
		{"truncated", valid[:len(valid)-1]},
		{"second value", append(append([]byte(nil), valid...), 0xc0)},
		{"invalid suffix", append(append([]byte(nil), valid...), 0xc1)},
		{"whitespace suffix", append(append([]byte(nil), valid...), ' ')},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var got any
			if err := s.Decode(tt.data, &got); err == nil {
				t.Fatal("Decode accepted an incomplete or trailing record")
			}
			// A rejected record must not leave pooled decoder state for the next request. 拒绝记录后，池中解码器状态不能影响下一次请求。
			var next string
			if err := s.Decode(valid, &next); err != nil || next != "record" {
				t.Fatalf("Decode after failure = %q, %v", next, err)
			}
		})
	}
}

// TestDecodePreservesPointerAllocationAndUnknownFields verifies compatible payloads retain their existing behavior. TestDecodePreservesPointerAllocationAndUnknownFields 验证兼容载荷保留现有行为。
func TestDecodePreservesPointerAllocationAndUnknownFields(t *testing.T) {
	type payload struct {
		Name string
		Data []byte
	}
	s := NewMsgPackSerializer()
	data, err := s.Encode(map[string]any{"Name": "alice", "Data": []byte{0, 128, 255}, "FutureField": "ignored"})
	if err != nil {
		t.Fatal(err)
	}
	var got *payload
	if err := s.Decode(data, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, &payload{Name: "alice", Data: []byte{0, 128, 255}}) {
		t.Fatalf("decoded payload = %+v", got)
	}
}
