// Package builder validates configuration and assembles an engine from component instances and factories. Package builder 校验配置，并使用组件实例与工厂装配引擎。
//
// The root Builder provides the application entry, while internal/defaults selects bundled component factories. 根 Builder 提供应用入口，internal/defaults 选择内置组件工厂。
//
// Keep component selection and capability composition outside this generic assembly package. 具体组件选择和独立能力组合不放在本通用装配包中。
package builder
