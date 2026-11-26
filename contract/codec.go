// @Author daixk 2025/12/22 15:56:00
package contract

// Codec defines serialization behavior interface Codec 定义序列化行为接口
// Session, token extensions and capability objects use Codec; new token primary records use an engine-private fixed format. Session、Token 扩展及能力对象使用 Codec；新 Token 主记录使用引擎私有固定格式。
type Codec interface {
	// Name returns codec name Name 返回序列化器名称
	Name() string

	// Encode encodes object to bytes Encode 将对象编码为字节数组
	Encode(v any) ([]byte, error)

	// Decode decodes bytes into target object Decode 将字节数组解码到目标对象
	Decode(data []byte, v any) error
}
