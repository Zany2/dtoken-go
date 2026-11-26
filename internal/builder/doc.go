// Package builder supplies bundled factories, validates configuration, and assembles an engine from components. Package builder 提供内置工厂、校验配置，并使用组件装配引擎。
//
// Only the root package owns a fluent Builder; Build here is a stateless assembly function. 仅根包持有链式 Builder，本包 Build 是无状态装配函数。
//
// Bundled implementations are selected in defaults.go; authentication rules remain in the engine and capabilities are composed by applications. 内置实现集中在 defaults.go 中选择，认证规则保留在引擎中，独立能力由应用组合。
package builder
