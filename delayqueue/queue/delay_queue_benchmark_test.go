package queue

import (
	"fmt"
	"sync"
	"testing"
	"time"
	"timewheel/delayqueue/config"
	"timewheel/delayqueue/task"
)

// MockRequest 模拟请求
type MockRequest struct {
	id string
}

func (r *MockRequest) GetTaskID() string                 { return r.id }
func (r *MockRequest) GetTaskType() string               { return "mock" }
func (r *MockRequest) Execute(handler interface{}) error { return nil }

// MockHandler 模拟处理器
type MockHandler struct {
	processTime time.Duration // 模拟处理时间
}

func (h *MockHandler) Handle(req task.Request) error {
	time.Sleep(h.processTime)
	return nil
}

func BenchmarkDelayQueue(b *testing.B) {
	tests := []struct {
		name        string
		concurrent  int // 并发数
		totalTasks  int // 总任务数
		processTime int // 任务处理时间(ms)
		delay       int // 延迟时间(ms)
		slotNum     int // 槽数量
		interval    int // 检查间隔(ms)
		maxRetries  int // 最大重试次数
	}{
		{
			name:        "Heavy Tasks",
			concurrent:  10,
			totalTasks:  5000,
			processTime: 1000,
			delay:       100,
			slotNum:     60,
			interval:    100,
			maxRetries:  3,
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			cfg := &config.DelayQueueConfig{
				SlotNum:       tt.slotNum,
				Interval:      time.Duration(tt.interval) * time.Millisecond,
				MaxRetries:    tt.maxRetries,
				RetryInterval: time.Second,
			}
			dq, err := NewDelayQueue(cfg)
			if err != nil {
				b.Fatalf("Failed to create delay queue: %v", err)
			}
			defer dq.Stop()

			handler := &MockHandler{
				processTime: time.Duration(tt.processTime) * time.Millisecond,
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup
				taskPerGoroutine := tt.totalTasks / tt.concurrent

				// 启动并发 goroutine 添加任务
				for j := 0; j < tt.concurrent; j++ {
					wg.Add(1)
					go func(base int) {
						defer wg.Done()
						for k := 0; k < taskPerGoroutine; k++ {
							req := &MockRequest{
								id: fmt.Sprintf("task_%d_%d", base, k),
							}
							err := dq.AddTask(req, handler, time.Duration(tt.delay)*time.Millisecond)
							if err != nil {
								b.Errorf("Failed to add task: %v", err)
							}
						}
					}(j * taskPerGoroutine)
				}

				wg.Wait()
				// 等待所有任务执行完成
				time.Sleep(time.Duration(tt.delay+tt.processTime) * time.Millisecond)
			}
		})
	}
}

// 测试任务执行延迟
func BenchmarkTaskDelay(b *testing.B) {
	cfg := &config.DelayQueueConfig{
		SlotNum:       60,
		Interval:      100 * time.Millisecond,
		MaxRetries:    3,
		RetryInterval: time.Second,
	}
	dq, err := NewDelayQueue(cfg)
	if err != nil {
		b.Fatalf("Failed to create delay queue: %v", err)
	}
	defer dq.Stop()

	handler := &MockHandler{processTime: time.Millisecond}
	delays := []time.Duration{
		100 * time.Millisecond,
		500 * time.Millisecond,
		1 * time.Second,
	}

	for _, delay := range delays {
		b.Run(fmt.Sprintf("Delay_%v", delay), func(b *testing.B) {
			var totalDelay time.Duration
			for i := 0; i < b.N; i++ {
				start := time.Now()
				req := &MockRequest{id: fmt.Sprintf("task_%d", i)}
				err := dq.AddTask(req, handler, delay)
				if err != nil {
					b.Errorf("Failed to add task: %v", err)
				}
				totalDelay += time.Since(start)
			}
			b.ReportMetric(float64(totalDelay.Nanoseconds()/int64(b.N)), "ns/op")
		})
	}
}
