// Package context binds authentication operations and capability contracts to a request adapter. Package context 将认证操作和能力契约绑定到请求适配器。
//
// Framework packages supply request adapters and select an engine; applications compose services and bind them through WithCapabilities. 框架包提供请求适配器并选择引擎；应用组合服务并通过 WithCapabilities 绑定。
//
// This package borrows engines and services without constructing or owning their runtime components. 本包借用引擎与服务，不创建或持有其运行时组件的所有权。
//
// Capability operations honor shutdown when an engine is bound; standalone services can be used without an engine, while token extraction and cookie operations also require a request adapter. 绑定引擎时能力操作遵循其关闭状态；独立服务可不绑定引擎，Token 提取和 Cookie 操作还要求请求适配器。
package context
