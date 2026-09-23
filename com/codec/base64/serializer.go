// @Author daixk 2026/1/28 14:44:00
package base64

import (
	"encoding/base64"
	"encoding/json"

	"github.com/Zany2/dtoken-go/core/adapter"
)

// Base64Serializer implements a Base64 JSON serializer Base64 JSON 序列化器实现
type Base64Serializer struct{}

// Interface assertion keeps codec contract checked at compile time 接口断言在编译期检查编解码器契约
var _ adapter.Codec = (*Base64Serializer)(nil)

// Encode serializes JSON and then encodes it with Base64 先编码为 JSON 再做 Base64 编码
func (s *Base64Serializer) Encode(v any) ([]byte, error) {
	// Serialize the value to JSON first 先用 JSON 序列化
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	// Encode directly into the returned byte slice. 直接编码到返回的字节切片中。
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(jsonBytes)))
	base64.StdEncoding.Encode(encoded, jsonBytes)
	return encoded, nil
}

// Decode decodes Base64 data and then deserializes JSON 先做 Base64 解码再从 JSON 反序列化
func (s *Base64Serializer) Decode(data []byte, v any) error {
	// Decode directly from the input bytes. 直接从输入字节解码。
	jsonBytes := make([]byte, base64.StdEncoding.DecodedLen(len(data)))
	n, err := base64.StdEncoding.Decode(jsonBytes, data)
	if err != nil {
		return err
	}

	// Deserialize the decoded bytes from JSON 再用 JSON 反序列化
	return json.Unmarshal(jsonBytes[:n], v)
}

// Name returns the serializer name 返回序列化器名称
func (s *Base64Serializer) Name() string {
	return "base64"
}

// NewBase64Serializer creates a Base64 serializer 创建 Base64 序列化器
func NewBase64Serializer() *Base64Serializer {
	return &Base64Serializer{}
}
