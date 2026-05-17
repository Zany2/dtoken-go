// @Author daixk 2025/11/27 20:58:00
package msgpack

import (
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

// Decode deserializes MsgPack into a value 从 MsgPack 解码
func (s *MsgPackSerializer) Decode(data []byte, v any) error {
	return msgpack.Unmarshal(data, v)
}

// Name returns the serializer name 返回序列化器名称
func (s *MsgPackSerializer) Name() string { return "msgpack" }

// NewMsgPackSerializer creates a MsgPack serializer 创建 MsgPack 序列化器
func NewMsgPackSerializer() *MsgPackSerializer {
	return &MsgPackSerializer{}
}
