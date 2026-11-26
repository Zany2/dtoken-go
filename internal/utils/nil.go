package utils

import "reflect"

// IsNil detects absent interface values, including typed nils, without rejecting value implementations. IsNil 识别接口中的空值与带类型空值，兼容值类型实现。
func IsNil(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}
