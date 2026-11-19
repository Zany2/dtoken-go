// Package engine implements authentication state and lifecycle behind the public dtoken facade. Package engine 实现公开 dtoken 门面背后的认证状态与生命周期。
//
// Operation files use shared account coordination, token maintenance, and runtime shutdown state within this package. 操作文件在同一包内共享账号协调、Token 维护和运行时关闭状态。
//
// Components enter through contract interfaces; default wiring, framework adapters, global registration, and independent capabilities belong outside the engine. 组件通过 contract 接口接入；默认装配、框架适配、全局注册和独立能力不属于引擎职责。
//
// Applications obtain supported advanced engine types through the root dtoken package; internal implementation placement does not remove those public API obligations. 应用通过根 dtoken 包获取受支持的高级引擎类型；实现位于 internal 不会取消这些类型的公开 API 约定。
package engine
