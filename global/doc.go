// Package global preserves legacy access to the root dtoken registry. Package global 保留访问根 dtoken 注册表的旧版入口。
//
// Root Builder.Build registers automatically; SetAuth explicitly registers or replaces instances built with BuildInstance. 根 Builder.Build 自动注册；SetAuth 用于显式注册或替换通过 BuildInstance 构建的实例。
//
// Registry entries share their engine with Auth wrappers, and replacement or deletion closes the registered engine. 注册项与 Auth 门面共享引擎，替换或删除注册项会关闭对应引擎。
//
// HTTP integrations opt into registry lookup through WithManagerResolver; the shared HTTP layer does not import this package. HTTP 集成通过 WithManagerResolver 显式选择注册表查找，共享 HTTP 层不导入本包。
package global
