// @Author daixk 2026/1/22 13:46:00
package utils

import (
	"fmt"
	"math"
)

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

// ToInt64 将任意类型转换为 int64
func ToInt64(value any) (int64, error) {
	switch v := value.(type) {
	case int:
		return int64(v), nil
	case int8:
		return int64(v), nil
	case int16:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case uint:
		return int64(v), nil
	case uint8:
		return int64(v), nil
	case uint16:
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint64:
		if v > math.MaxInt64 {
			return 0, fmt.Errorf("uint64 value %d overflows int64", v)
		}
		return int64(v), nil
	case float32:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case string:
		// 简单尝试：仅支持纯数字字符串（不处理空格、进制等）
		var i int
		if _, err := fmt.Sscanf(v, "%d", &i); err == nil {
			return int64(i), nil
		}
		return 0, fmt.Errorf("cannot parse string %q as int64", v)
	case bool:
		if v {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("unsupported type: %T", value)
	}
}
