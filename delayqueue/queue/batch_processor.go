package queue

import (
	"sync"
	"time"
)

type BatchProcessor struct {
	tasks    []interface{}
	size     int
	interval time.Duration
	process  func([]interface{}) error
	mutex    sync.Mutex
	timer    *time.Timer
}

func NewBatchProcessor(size int, interval time.Duration, process func([]interface{}) error) *BatchProcessor {
	bp := &BatchProcessor{
		tasks:    make([]interface{}, 0, size),
		size:     size,
		interval: interval,
		process:  process,
	}
	bp.timer = time.AfterFunc(interval, bp.flush)
	return bp
}

func (bp *BatchProcessor) Add(item interface{}) {
	bp.mutex.Lock()
	bp.tasks = append(bp.tasks, item)
	if len(bp.tasks) >= bp.size {
		tasks := bp.tasks
		bp.tasks = make([]interface{}, 0, bp.size)
		bp.mutex.Unlock()
		bp.process(tasks)
		return
	}
	bp.mutex.Unlock()
}

func (bp *BatchProcessor) flush() {
	bp.mutex.Lock()
	if len(bp.tasks) > 0 {
		tasks := bp.tasks
		bp.tasks = make([]interface{}, 0, bp.size)
		bp.mutex.Unlock()
		bp.process(tasks)
	} else {
		bp.mutex.Unlock()
	}
	bp.timer.Reset(bp.interval)
}
