// @Author daixk 2025/12/22 15:56:00
package contract

// Pool defines worker task pool interface Pool 定义工作任务池接口
type Pool interface {
	// Submit must be concurrency-safe and nonblocking; returning nil accepts asynchronous execution exactly once, while an error rejects the task without executing it. Submit 必须并发安全且非阻塞；返回 nil 表示恰好异步执行一次，返回错误表示拒绝且不执行任务。
	Submit(task func()) error

	// Stop stops pool and releases resources Stop 停止池并释放所有资源
	Stop()

	// Stats returns runtime statistics Stats 返回运行时统计信息
	Stats() (running, capacity int, usage float64)
}
