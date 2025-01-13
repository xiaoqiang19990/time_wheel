package queue

import (
	"container/list"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"timewheel/delayqueue/config"
	"timewheel/delayqueue/handlers"
	"timewheel/delayqueue/storage"
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
	wg         sync.WaitGroup
	stopCh     chan struct{}
	storage    storage.Storage
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
		ticker:     time.NewTicker(cfg.Interval),
		options:    cfg,
		metrics:    &Metrics{},
		stopCh:     make(chan struct{}),
		storage:    storage.NewRedisStorage(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB),
		tasks:      make(map[string]int),
		workerPool: make(chan struct{}, cfg.WorkerNum),
	}

	// 初始化分片
	for i := 0; i < cfg.SlotNum; i++ {
		dq.slots[i] = &Slot{
			tasks: list.New(),
			mutex: sync.RWMutex{},
		}
	}

	// 先恢复任务，再启动时间轮
	if err := dq.recoverTasks(); err != nil {
		return nil, fmt.Errorf("failed to recover tasks: %v", err)
	}

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

	// 先保存到Redis
	taskData, err := json.Marshal(t.Request)
	if err != nil {
		return fmt.Errorf("marshal request error: %v", err)
	}

	taskInfo := &storage.TaskInfo{
		ID:        t.ID,
		Type:      t.Request.GetTaskType(),
		Data:      taskData,
		Delay:     t.Delay,
		Timestamp: t.Timestamp,
		Retries:   t.Retries,
	}

	if err := dq.storage.Save(taskInfo); err != nil {
		fmt.Printf("Failed to save task to storage: %v\n", err)
		return err
	}

	// 保存成功后添加到时间轮
	slot := dq.slots[pos]
	slot.mutex.Lock()
	defer slot.mutex.Unlock()

	slot.tasks.PushBack(t)
	dq.tasks[t.ID] = pos
	atomic.AddInt64(&dq.metrics.pendingTasks, 1)

	fmt.Printf("Task %s successfully saved to storage and added to time wheel\n", t.ID)
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

			// fmt.Printf("\nChecking slot %d at %v\n", dq.current, now)

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
	dq.workerPool <- struct{}{}
	defer func() {
		<-dq.workerPool
	}()

	atomic.AddInt64(&dq.metrics.pendingTasks, -1)
	start := time.Now()

	fmt.Printf("Executing task %s at %v\n", t.ID, time.Now())
	err := t.Handler.Handle(t.Request)

	// 先从存储中删除任务
	if err := dq.storage.Remove(t.ID); err != nil {
		fmt.Printf("Failed to remove task from storage: %v\n", err)
	} else {
		fmt.Printf("Task %s removed from storage\n", t.ID)
	}

	atomic.AddInt64(&dq.metrics.executionTime, time.Since(start).Milliseconds())
	if err != nil {
		if t.Retries < dq.options.MaxRetries {
			atomic.AddInt64(&dq.metrics.retryTasks, 1)
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

// recoverTasks 恢复任务
func (dq *DelayQueue) recoverTasks() error {
	tasks, err := dq.storage.GetAll()
	if err != nil {
		return err
	}

	if len(tasks) > 0 {
		fmt.Printf("Recovering %d tasks from storage\n", len(tasks))
	}

	now := time.Now()
	for _, taskInfo := range tasks {
		// 创建请求和处理器
		calcReq := &handlers.CalculationRequest{}
		if err := json.Unmarshal(taskInfo.Data, calcReq); err != nil {
			fmt.Printf("unmarshal request error: %v\n", err)
			continue
		}

		// 使用通用处理器
		handler := handlers.NewCalculatorHandler()

		// 创建新任务
		t := task.NewTask(taskInfo.ID, calcReq, handler, taskInfo.Delay)
		t.Retries = taskInfo.Retries
		t.Timestamp = taskInfo.Timestamp

		if now.After(t.Timestamp) {
			fmt.Printf("Recovering expired task %s for immediate execution\n", t.ID)
			// 对于过期任务，立即执行
			dq.wg.Add(1)
			go func(task *task.Task) {
				defer dq.wg.Done()
				dq.executeTask(task)
			}(t)
		} else {
			// 对于未过期任务，重新计算延迟并添加
			t.Delay = t.Timestamp.Sub(now)
			fmt.Printf("Recovering task %s with delay %v\n", t.ID, t.Delay)
			if err := dq.addTask(t); err != nil {
				fmt.Printf("recover task error: %v\n", err)
			}
		}
	}

	return nil
}
