// Package httpauth provides shared HTTP authentication checks, manager resolution, and public error mapping. Package httpauth 提供共享 HTTP 鉴权检查、管理器解析和公开错误映射。
//
// Framework integrations adapt their native handlers to these checks, while the context subpackage binds request operations and capability services. 框架集成将原生处理链接入这些检查，context 子包绑定请求操作和能力服务。
//
// Registry lookup is explicit; shared HTTP code does not import global or concrete capability implementations. 注册表查找须显式配置；共享 HTTP 代码不导入 global 或具体能力实现。
package httpauth
