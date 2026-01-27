// @Author daixk 2026/1/22 13:46:00
package utils

import "fmt"

// ToBytes 将任意类型转换为字节切片
func ToBytes(value any) ([]byte, error) {
	switch v := value.(type) {
	case string:
		return []byte(v), nil
	case []byte:
		return v, nil
	case byte:
		return []byte{v}, nil
	case rune:
		return []byte(string(v)), nil
	default:
		return nil, fmt.Errorf("unsupported type: %T", value)
	}
}
