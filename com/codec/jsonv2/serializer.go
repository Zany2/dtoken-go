//go:build goexperiment.jsonv2

// @Author daixk 2026/1/21 11:35:00
package jsonv2

import (
	"encoding/json"
	jsonv2 "encoding/json/v2"

	"github.com/Zany2/dtoken-go/core/adapter"
)

// JSONV2Serializer implements the JSON v2 codec. JSONV2Serializer 实现 JSON v2 编解码器。
type JSONV2Serializer struct{}

// Interface assertion keeps the codec contract checked at compile time. 接口断言在编译期检查编解码器契约。
var _ adapter.Codec = (*JSONV2Serializer)(nil)

// Encode serializes a value into JSON v2. 使用 JSON v2 将值编码为 JSON。
func (s *JSONV2Serializer) Encode(v any) ([]byte, error) {
	// Preserve numeric nanoseconds for duration fields in stored records. 存储记录中的时长字段保留纳秒数值表示。
	return jsonv2.Marshal(v, json.FormatDurationAsNano(true))
}

// Decode deserializes JSON v2 into a value. 使用 JSON v2 将数据解码到值中。
func (s *JSONV2Serializer) Decode(data []byte, v any) error {
	return jsonv2.Unmarshal(data, v, json.FormatDurationAsNano(true))
}

// Name returns the serializer name. 返回编解码器名称。
func (s *JSONV2Serializer) Name() string { return "jsonv2" }

// NewJSONV2Serializer creates a JSON v2 serializer. 创建 JSON v2 编解码器。
func NewJSONV2Serializer() *JSONV2Serializer {
	return &JSONV2Serializer{}
}
