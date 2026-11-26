package utils

import "strings"

// StorageVersionSegment isolates the current storage layout from unversioned records. StorageVersionSegment 将当前存储布局与无版本记录隔离。
const StorageVersionSegment = "v2:"

// StorageNamespace encodes independent namespace components and appends the current storage version. StorageNamespace 编码独立命名空间组件并追加当前存储版本。
func StorageNamespace(prefix, authType string) string {
	return LegacyStorageNamespace(prefix, authType) + StorageVersionSegment
}

// LegacyStorageNamespace addresses unversioned data for explicit offline migration only. LegacyStorageNamespace 仅为显式离线迁移定位无版本数据。
func LegacyStorageNamespace(prefix, authType string) string {
	encode := func(value string) string {
		value = strings.TrimSuffix(value, ":")
		value = strings.ReplaceAll(value, "\\", "\\\\")
		return strings.ReplaceAll(value, ":", "\\:") + ":"
	}
	return encode(prefix) + encode(authType)
}
