// @Author daixk 2025/12/12 10:46:00
package adapter

// Codec 序列化行为抽象接口
type Codec interface {
	// Name 序列化器名称
	Name() string
	// Encode 将对象编码为字节数组
	Encode(v any) ([]byte, error)
	// Decode 将字节数组解码到目标对象
	Decode(data []byte, v any) error
}
