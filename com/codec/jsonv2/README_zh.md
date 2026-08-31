# JSON v2 Codec（预留）

本模块为未来基于标准库 JSON v2 实现的 Codec 预留 `github.com/Zany2/dtoken-go/com/codec/jsonv2` 模块路径。

DToken-Go 工作区当前以 Go 1.25 为最低版本，因此本模块有意不导出 Codec 实现。等项目最低 Go 版本升级到 Go 1.27 后再正式实现。在此之前，请使用以下方案之一：

- `com/codec/json`
- `com/codec/msgpack`
- `com/codec/base64`
- 自行实现 `adapter.Codec`

保留占位模块可以稳定规划中的模块路径和工作区结构，同时避免引入构建标签、实验性依赖或兼容垫片。

[English](README.md)
