// @Author daixk 2026/1/21 11:35:00

// Package jsonv2 reserves the module path for the future standard-library JSON v2 codec. jsonv2 包为未来基于标准库的 JSON v2 编解码器预留模块路径。
//
// The module intentionally exports no Codec while DToken-Go targets Go 1.25; implementation starts after the minimum Go version reaches 1.27. DToken-Go 以 Go 1.25 为最低版本期间，本模块有意不导出 Codec；最低版本升级到 Go 1.27 后再开始实现。
package jsonv2
