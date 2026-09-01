// @Author daixk 2026/1/21 13:48:00
package msgpack

import (
	"reflect"
	"testing"
)

// TestMsgPackSerializer_Name tests serializer name behavior 测试序列化器名称行为
func TestMsgPackSerializer_Name(t *testing.T) {
	s := NewMsgPackSerializer()
	if s == nil {
		t.Fatal("NewMsgPackSerializer() returned nil")
	}
	if got := s.Name(); got != "msgpack" {
		t.Errorf("Name() = %q, want %q", got, "msgpack")
	}
}

// TestMsgPackSerializer_Encode tests MsgPack encoding behavior 测试 MsgPack 编码行为
func TestMsgPackSerializer_Encode(t *testing.T) {
	s := NewMsgPackSerializer()

	// Person defines test data for MsgPack serializer tests 定义 MsgPack 序列化测试数据
	type Person struct {
		Name string
		Age  int
	}

	tests := []struct {
		name    string
		input   any
		wantErr bool
	}{
		{
			name:  "basic struct",
			input: Person{Name: "Alice", Age: 30},
		},
		{
			name:  "map",
			input: map[string]int{"score": 95},
		},
		{
			name:  "slice",
			input: []int{1, 2, 3},
		},
		{
			name:  "primitive",
			input: 42,
		},
		{
			name:    "unsupported type (chan)",
			input:   make(chan int),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.Encode(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Encode() derror = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestMsgPackSerializer_Decode tests MsgPack decoding behavior 测试 MsgPack 解码行为
func TestMsgPackSerializer_Decode(t *testing.T) {
	s := NewMsgPackSerializer()

	// Person defines test data for MsgPack serializer tests 定义 MsgPack 序列化测试数据
	type Person struct {
		Name string
		Age  int
	}

	tests := []struct {
		name    string
		prepare func() ([]byte, any) // 返回编码后的数据 + 目标指针
		want    any
		wantErr bool
	}{
		{
			name: "decode to struct",
			prepare: func() ([]byte, any) {
				data, _ := s.Encode(Person{Name: "Bob", Age: 25})
				return data, &Person{}
			},
			want: &Person{Name: "Bob", Age: 25},
		},
		{
			name: "decode to map",
			prepare: func() ([]byte, any) {
				data, _ := s.Encode(map[string]int{"count": 100})
				target := &map[string]int{}
				return data, target
			},
			want: &map[string]int{"count": 100},
		},
		{
			name: "decode to slice",
			prepare: func() ([]byte, any) {
				data, _ := s.Encode([]string{"a", "b"})
				target := &[]string{}
				return data, target
			},
			want: &[]string{"a", "b"},
		},
		{
			name: "malformed data",
			prepare: func() ([]byte, any) {
				return []byte{0xFF, 0xFF, 0xFF}, &struct{}{} // 无效 msgpack 数据
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, targetPtr := tt.prepare()
			err := s.Decode(data, targetPtr)
			if (err != nil) != tt.wantErr {
				t.Errorf("Decode() derror = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(targetPtr, tt.want) {
				t.Errorf("Decode() got = %v, want %v", targetPtr, tt.want)
			}
		})
	}
}

// TestMsgPackSerializerNilAndBinaryRoundTrip verifies nil and binary payloads round trip correctly. TestMsgPackSerializerNilAndBinaryRoundTrip 验证 nil 和二进制载荷可以正确往返。
func TestMsgPackSerializerNilAndBinaryRoundTrip(t *testing.T) {
	s := NewMsgPackSerializer()

	nilData, err := s.Encode(nil)
	if err != nil {
		t.Fatalf("Encode(nil) error = %v", err)
	}
	if len(nilData) == 0 {
		t.Fatal("Encode(nil) returned empty data")
	}
	var nilValue any
	if err := s.Decode(nilData, &nilValue); err != nil {
		t.Fatalf("Decode(nil) error = %v", err)
	}
	if nilValue != nil {
		t.Fatalf("Decode(nil) = %#v, want nil", nilValue)
	}

	want := []byte{0x00, 0x01, 0x7f, 0x80, 0xff}
	data, err := s.Encode(want)
	if err != nil {
		t.Fatalf("Encode(binary) error = %v", err)
	}
	var got []byte
	if err := s.Decode(data, &got); err != nil {
		t.Fatalf("Decode(binary) error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Decode(binary) = %#v, want %#v", got, want)
	}
}

// TestMsgPackSerializerInvalidTargets verifies invalid decode targets return errors. TestMsgPackSerializerInvalidTargets 验证非法解码目标会返回错误。
func TestMsgPackSerializerInvalidTargets(t *testing.T) {
	s := NewMsgPackSerializer()
	data, err := s.Encode(map[string]string{"name": "Alice"})
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	if err := s.Decode(data, map[string]string{}); err == nil {
		t.Fatal("Decode() should reject a non-pointer target")
	}
	var nilTarget *struct{}
	if err := s.Decode(data, nilTarget); err == nil {
		t.Fatal("Decode() should reject a nil pointer target")
	}
}
