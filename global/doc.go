// Package global provides optional registry-based access to authentication instances. Package global 提供基于可选注册表的认证实例访问入口。
//
// Applications build Auth through the root dtoken package and register it with SetAuth; instance operations do not require global registration. 应用通过根 dtoken 包构建 Auth，并使用 SetAuth 注册；实例操作不要求全局注册。
//
// Registry entries share their engine with Auth wrappers, and replacement or deletion closes the registered engine. 注册项与 Auth 门面共享引擎，替换或删除注册项会关闭对应引擎。
//
// HTTP integrations opt into registry lookup through WithManagerResolver; the shared HTTP layer does not import this package. HTTP 集成通过 WithManagerResolver 显式选择注册表查找，共享 HTTP 层不导入本包。
package global
