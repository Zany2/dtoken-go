package utils

import "strings"

// StorageNamespace encodes two independent namespace components, preserving ordinary colon-terminated names. StorageNamespace 编码两个独立的命名空间组件，保留普通冒号结尾名称的键格式。
func StorageNamespace(prefix, authType string) string {
	encode := func(value string) string {
		value = strings.TrimSuffix(value, ":")
		value = strings.ReplaceAll(value, "\\", "\\\\")
		return strings.ReplaceAll(value, ":", "\\:") + ":"
	}
	return encode(prefix) + encode(authType)
}
