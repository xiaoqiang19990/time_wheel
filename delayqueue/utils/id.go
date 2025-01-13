package utils

import (
	"fmt"
	"sync/atomic"
	"time"
)

var (
	sequence uint64
)

// GenerateTaskID 生成唯一的任务ID
// 格式: task_timestamp_sequence
func GenerateTaskID() string {
	seq := atomic.AddUint64(&sequence, 1)
	return fmt.Sprintf("task_%d_%d", time.Now().UnixNano(), seq)
}

// GenerateTaskIDWithPrefix 生成带前缀的唯一任务ID
// 格式: prefix_timestamp_sequence
func GenerateTaskIDWithPrefix(prefix string) string {
	seq := atomic.AddUint64(&sequence, 1)
	return fmt.Sprintf("%s_%d_%d", prefix, time.Now().UnixNano(), seq)
}
