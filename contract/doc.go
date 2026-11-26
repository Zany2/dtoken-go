// Package contract defines shared domain models and component and request adapter interfaces. Package contract 定义共享领域模型、组件接口和请求适配接口。
//
// Component contracts live in storage.go, codec.go, generator.go, log.go, and pool.go; model.go and access.go define shared authentication data, and context.go defines HTTP adapter contracts. 组件契约位于 storage.go、codec.go、generator.go、log.go 和 pool.go；model.go 与 access.go 定义共享认证数据，context.go 定义 HTTP 适配契约。
//
// Capability subpackages define service interfaces and data types; their implementations live in the corresponding root capability packages. 能力子包定义服务接口和数据类型，对应实现位于根模块下的能力包。
//
// Keep application assembly and dependencies on concrete storage, frameworks, and capability implementations outside this package; storagetest supplies reusable checks for storage implementers. 应用装配及对具体存储、框架和能力实现的依赖不放在本包中；storagetest 为存储实现者提供可复用检查。
package contract
