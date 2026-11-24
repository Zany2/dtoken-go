// Package dtoken provides instance and registry authentication facades with component assembly. Package dtoken 提供实例与注册表认证门面及组件装配。
//
// NewBuilder().Build() registers and returns Auth; call it directly, select AuthType, or use package functions for the default namespace. BuildInstance creates an unregistered instance. NewBuilder().Build() 注册并返回 Auth；可直接调用实例、通过 AuthType 选择体系，或用包级函数调用默认体系；BuildInstance 创建未注册实例。
//
// Manager and ManagerOption are supported advanced APIs for custom adapters. They share the Auth lifecycle and expose implementation-level operations; applications should prefer Auth methods. Manager 与 ManagerOption 是受支持的高级适配 API，与 Auth 共享生命周期并提供底层操作；应用优先使用 Auth 方法。
//
// Assembly transfers component ownership according to WithComponentOwnership. Assembly failure leaves explicit components with the caller; a registration race closes the assembled losing candidate. 装配完成后按 WithComponentOwnership 管理组件；装配失败时显式组件仍由调用方持有，注册竞争失败则关闭已完成装配的候选实例。
package dtoken
