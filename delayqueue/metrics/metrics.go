package metrics

import (
	"sync/atomic"
	"time"
)

type Metrics struct {
	PendingTasks    int64        // 待处理任务数
	ExecutedTasks   int64        // 已执行任务数
	RetryTasks      int64        // 重试任务数
	FailedTasks     int64        // 失败任务数
	ExecutionTime   int64        // 任务执行时间
	QueueLatency    int64        // 队列延迟
	ProcessingTime  int64        // 处理时间
	StorageLatency  int64        // 存储延迟
	ConcurrentTasks int64        // 并发任务数
	LastUpdateTime  atomic.Value // 最后更新时间
}

func (m *Metrics) RecordTaskExecution(start time.Time) {
	atomic.AddInt64(&m.ExecutedTasks, 1)
	atomic.AddInt64(&m.ExecutionTime, time.Since(start).Milliseconds())
}

func (m *Metrics) RecordStorageLatency(start time.Time) {
	atomic.AddInt64(&m.StorageLatency, time.Since(start).Milliseconds())
}

func (m *Metrics) IncrementConcurrentTasks() {
	atomic.AddInt64(&m.ConcurrentTasks, 1)
}

func (m *Metrics) DecrementConcurrentTasks() {
	atomic.AddInt64(&m.ConcurrentTasks, -1)
}
