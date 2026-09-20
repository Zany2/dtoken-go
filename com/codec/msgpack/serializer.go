// @Author daixk 2025/11/27 20:58:00
package msgpack

import (
	"bytes"
	"fmt"
	"reflect"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/vmihailenco/msgpack/v5"
)

// MsgPackSerializer implements a MsgPack serializer MsgPack 序列化器实现
type MsgPackSerializer struct{}

// Interface assertion keeps codec contract checked at compile time 接口断言在编译期检查编解码器契约
var _ adapter.Codec = (*MsgPackSerializer)(nil)

// Encode serializes a value into MsgPack 编码为 MsgPack
func (s *MsgPackSerializer) Encode(v any) ([]byte, error) {
	return msgpack.Marshal(v)
}

// Decode deserializes exactly one MsgPack value into a non-nil pointer. Decode 将单个完整 MsgPack 值解码到非 nil 指针。
func (s *MsgPackSerializer) Decode(data []byte, v any) error {
	// Validate before dependency fast paths dereference typed-nil slice or map pointers. 在依赖快速路径解引用 typed-nil 切片或映射指针前校验。
	target := reflect.ValueOf(v)
	if target.Kind() != reflect.Ptr || target.IsNil() {
		return fmt.Errorf("msgpack: decode target must be a non-nil pointer, got %T", v)
	}

	// A byte reader avoids read-ahead so trailing record data remains observable. 字节读取器避免预读，确保能识别记录的尾随数据。
	reader := bytes.NewReader(data)
	decoder := msgpack.GetDecoder()
	decoder.Reset(reader)
	defer msgpack.PutDecoder(decoder)
	if err := decoder.Decode(v); err != nil {
		return err
	}
	if reader.Len() != 0 {
		return fmt.Errorf("msgpack: unexpected trailing data: %d bytes", reader.Len())
	}
	return nil
}

// Name returns the serializer name 返回序列化器名称
func (s *MsgPackSerializer) Name() string { return "msgpack" }

// NewMsgPackSerializer creates a MsgPack serializer 创建 MsgPack 序列化器
func NewMsgPackSerializer() *MsgPackSerializer {
	return &MsgPackSerializer{}
}
