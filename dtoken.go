// Package dtoken provides instance-oriented authentication and component assembly. Package dtoken 提供实例化认证与组件装配。
//
// Build an Auth with NewBuilder, use its typed operations, and bind it to HTTP middleware with WithAuth. 使用 NewBuilder 构建 Auth，通过类型化方法操作，并以 WithAuth 绑定 HTTP 中间件。
//
// Manager and ManagerOption are supported advanced APIs for custom adapters. They share the Auth lifecycle and expose implementation-level operations; applications should prefer Auth methods. Manager 与 ManagerOption 是受支持的高级适配 API，与 Auth 共享生命周期并提供底层操作；应用优先使用 Auth 方法。
//
// A successful build transfers component ownership according to WithComponentOwnership. A failed build leaves explicitly supplied components with the caller. 构建成功后按 WithComponentOwnership 管理组件；构建失败时显式传入的组件仍由调用方持有。
package dtoken
