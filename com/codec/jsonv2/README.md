# JSON v2 Codec (Reserved)

This module reserves `github.com/Zany2/dtoken-go/com/codec/jsonv2` for a future Codec based on the standard-library JSON v2 implementation.

The DToken-Go workspace currently targets Go 1.25, so this module intentionally exposes no Codec implementation. It will be implemented after the project's minimum Go version reaches Go 1.27. Until then, use one of these options:

- `com/codec/json`
- `com/codec/msgpack`
- `com/codec/base64`
- a custom implementation of `adapter.Codec`

Keeping the placeholder preserves the planned module path and workspace structure without adding build tags, experimental dependencies, or compatibility shims.

[中文文档](README_zh.md)
