// @Author daixk 2026/1/28 14:44:00
package base64

import (
	"encoding/base64"
	"encoding/json"
)

type Base64Serializer struct{}

func (s *Base64Serializer) Encode(v any) ([]byte, error) {
	// 1. 先用 JSON 序列化
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	// 2. 再用 Base64 编码（得到字符串）
	b64Str := base64.StdEncoding.EncodeToString(jsonBytes)
	// 3. 返回 []byte(b64Str)
	return []byte(b64Str), nil
}

func (s *Base64Serializer) Decode(data []byte, v any) error {
	// 1. 将 data 视为 Base64 字符串，先解码
	jsonBytes, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return err
	}
	// 2. 再用 JSON 反序列化
	return json.Unmarshal(jsonBytes, v)
}

func (s *Base64Serializer) Name() string {
	return "base64"
}

func NewBase64Serializer() *Base64Serializer {
	return &Base64Serializer{}
}
