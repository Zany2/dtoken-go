// @Author daixk 2025/12/26 15:17:00
package dlog

// Package dlog provides async logging for dtoken-go Package dlog 为 dtoken-go 提供异步日志实现
//
// -------------------------------------------------- Features - 特性 --------------------------------------------------
// Feature async write with a buffered queue 非阻塞缓冲队列异步写入
// Feature log rotation by size and time 按大小和时间滚动日志
// Feature automatic cleanup of expired backup files 自动清理过期备份文件
// Feature runtime config modification for level prefix and stdout 支持运行时修改级别前缀和控制台输出
// Feature thread safe design with proper locking 具备正确加锁的线程安全设计
//
// -------------------------------------------------- Future Enhancements - 未来增强计划 --------------------------------------------------
// TODO structured logging with JSON format output 结构化日志（JSON 格式输出）
// TODO sampling and rate limiting 日志采样与限流机制
// TODO trace span ID support for distributed tracing 分布式链路追踪 trace span ID 支持
// TODO log aggregation hooks such as ELK or Loki 日志聚合钩子（如发送到 ELK、Loki）
// TODO context aware logging with context.Context 支持 context.Context 的上下文日志
