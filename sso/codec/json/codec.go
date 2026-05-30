// @Author daixk 2026/05/29
package jsoncodec

import "encoding/json"

// Codec is the built-in JSON codec for SSO. Codec 是 SSO 内置 JSON 编解码器。
type Codec struct{}

// Name returns codec name. Name 返回编解码器名称。
func (Codec) Name() string { return "json" }

// Encode encodes value to JSON. Encode 将值编码为 JSON。
func (Codec) Encode(v any) ([]byte, error) { return json.Marshal(v) }

// Decode decodes JSON bytes into value. Decode 将 JSON 字节解码到目标值。
func (Codec) Decode(data []byte, v any) error { return json.Unmarshal(data, v) }
