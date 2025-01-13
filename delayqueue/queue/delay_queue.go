package queue

import (
	"container/list"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"timewheel/delayqueue/config"
	"timewheel/delayqueue/task"
)

// Options 队列配置选项
type Options struct {
	SlotNum       int
	Interval      time.Duration
	MaxRetries    int
	RetryInterval time.Duration
}

// DelayQueue 时间轮延时队列
type DelayQueue struct {
	slots      []*Slot
	ticker     *time.Ticker
	options    *config.DelayQueueConfig
	current    int
	tasks      map[string]int
	workerPool chan struct{}
	metrics    *Metrics
	wg         sync.WaitGroup // 添加 WaitGroup 来管理 goroutine
	stopCh     chan struct{}  // 添加停止信号通道
}

type Slot struct {
	tasks *list.List
	mutex sync.RWMutex // 每个槽位独立的锁
}

type Metrics struct {
	pendingTasks  int64 // 待处理任务数
	executedTasks int64 // 已执行任务数
	retryTasks    int64 // 重试任务数
	failedTasks   int64 // 失败任务数
	executionTime int64 // 任务执行时间
}

// NewDelayQueue 创建新的延时队列
func NewDelayQueue(cfg *config.DelayQueueConfig) (*DelayQueue, error) {
	dq := &DelayQueue{
		slots:      make([]*Slot, cfg.SlotNum),
		workerPool: make(chan struct{}, cfg.WorkerNum),
		options:    cfg,
		current:    0,
		tasks:      make(map[string]int),
		metrics:    &Metrics{},
		stopCh:     make(chan struct{}),
	}

	for i := 0; i < cfg.SlotNum; i++ {
		dq.slots[i] = &Slot{tasks: list.New()}
	}

	dq.ticker = time.NewTicker(cfg.Interval)
	go dq.run()

	return dq, nil
}

// AddTask 添加延时任务
func (dq *DelayQueue) AddTask(req task.Request, handler task.Handler, delay time.Duration) error {
	t := task.NewTask(req.GetTaskID(), req, handler, delay)
	return dq.addTask(t)
}

// addTask 内部添加任务方法
func (dq *DelayQueue) addTask(t *task.Task) error {
	// 计算延时槽位
	delay := int(t.Delay.Milliseconds() / dq.options.Interval.Milliseconds())
	pos := (dq.current + delay) % dq.options.SlotNum

	fmt.Printf("Adding task %s: slot %d, current slot: %d, delay: %v, execute at: %v\n",
		t.ID, pos, dq.current, t.Delay, t.Timestamp)

	slot := dq.slots[pos]
	slot.mutex.Lock()
	defer slot.mutex.Unlock()

	slot.tasks.PushBack(t)
	dq.tasks[t.ID] = pos
	atomic.AddInt64(&dq.metrics.pendingTasks, 1)
	return nil
}

// RemoveTask 移除任务
func (dq *DelayQueue) RemoveTask(taskID string) bool {
	pos, exists := dq.tasks[taskID]
	if !exists {
		return false
	}

	slot := dq.slots[pos]
	slot.mutex.Lock()
	defer slot.mutex.Unlock()

	for e := slot.tasks.Front(); e != nil; e = e.Next() {
		t := e.Value.(*task.Task)
		if t.ID == taskID {
			slot.tasks.Remove(e)
			delete(dq.tasks, taskID)
			return true
		}
	}
	return false
}

// run 运行时间轮
func (dq *DelayQueue) run() {
	for {
		select {
		case <-dq.stopCh:
			fmt.Println("Delay queue stopping...")
			return
		case now := <-dq.ticker.C:
			currentSlot := dq.slots[dq.current]
			currentSlot.mutex.Lock()

			fmt.Printf("\nChecking slot %d at %v\n", dq.current, now)

			var next *list.Element
			for e := currentSlot.tasks.Front(); e != nil; e = next {
				next = e.Next()
				t := e.Value.(*task.Task)
				fmt.Printf("Task %s: Execute at %v, Current time: %v, Should execute: %v\n",
					t.ID, t.Timestamp, now, now.After(t.Timestamp))

				if now.After(t.Timestamp) {
					currentSlot.tasks.Remove(e)
					delete(dq.tasks, t.ID)
					dq.wg.Add(1)
					go func(task *task.Task) {
						defer dq.wg.Done()
						dq.executeTask(task)
					}(t)
				} else {
					fmt.Printf("Task %s not ready yet, waiting for %v\n",
						t.ID, t.Timestamp.Sub(now))
				}
			}

			currentSlot.mutex.Unlock()
			dq.current = (dq.current + 1) % dq.options.SlotNum
		}
	}
}

// executeTask 执行任务
func (dq *DelayQueue) executeTask(t *task.Task) {
	// 获取工作线程
	dq.workerPool <- struct{}{}
	defer func() {
		<-dq.workerPool
	}()

	atomic.AddInt64(&dq.metrics.pendingTasks, -1)
	start := time.Now()

	fmt.Printf("Executing task %s at %v\n", t.ID, time.Now())
	err := t.Handler.Handle(t.Request)
	fmt.Println("err", err)
	atomic.AddInt64(&dq.metrics.executionTime, time.Since(start).Milliseconds())
	if err != nil {
		if t.Retries < dq.options.MaxRetries {
			atomic.AddInt64(&dq.metrics.retryTasks, 1)
			// 重试逻辑
			t.Retries++
			t.Delay = dq.options.RetryInterval
			t.Timestamp = time.Now().Add(t.Delay)
			if err := dq.addTask(t); err != nil {
				fmt.Printf("Failed to retry task %s: %v\n", t.ID, err)
			}
		} else {
			atomic.AddInt64(&dq.metrics.failedTasks, 1)
			fmt.Printf("Task %s failed after %d retries\n", t.ID, t.Retries)
		}
	} else {
		atomic.AddInt64(&dq.metrics.executedTasks, 1)
		fmt.Printf("Task %s completed successfully\n", t.ID)
	}
}

// Stop 停止时间轮
func (dq *DelayQueue) Stop() {
	close(dq.stopCh) // 发送停止信号
	dq.ticker.Stop()
	// 等待所有任务完成
	dq.wg.Wait()
	fmt.Println("All tasks completed, delay queue stopped")
}
