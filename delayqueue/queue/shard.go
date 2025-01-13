package queue

import (
	"container/list"
	"sync"
	"timewheel/delayqueue/task"
)

// QueueShard 队列分片
type QueueShard struct {
	slots   []*Slot
	current int
	tasks   map[string]int
	mutex   sync.RWMutex
}

func NewQueueShard(slotNum int) *QueueShard {
	shard := &QueueShard{
		slots: make([]*Slot, slotNum),
		tasks: make(map[string]int),
	}

	for i := 0; i < slotNum; i++ {
		shard.slots[i] = &Slot{tasks: list.New()}
	}

	return shard
}

func (s *QueueShard) AddTask(t *task.Task, pos int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.slots[pos].tasks.PushBack(t)
	s.tasks[t.ID] = pos
}

func (s *QueueShard) RemoveTask(taskID string) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	pos, exists := s.tasks[taskID]
	if !exists {
		return false
	}

	slot := s.slots[pos]
	for e := slot.tasks.Front(); e != nil; e = e.Next() {
		t := e.Value.(*task.Task)
		if t.ID == taskID {
			slot.tasks.Remove(e)
			delete(s.tasks, taskID)
			return true
		}
	}
	return false
}
