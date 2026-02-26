// @Author daixk 2025/12/12 11:56:00
package adapter

// Pool 工作任务池的抽象接口
type Pool interface {
	// Submit 提交一个任务以异步执行
	Submit(task func()) error
	// Stop 停止池并释放所有资源
	Stop()
	// Stats 返回运行时统计信息：当前运行数、最大容量和使用率
	Stats() (running, capacity int, usage float64)
}
