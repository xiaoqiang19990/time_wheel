package storage

import (
	"time"
)

// TaskInfo 任务存储信息
type TaskInfo struct {
	ID        string        `json:"id"`
	Type      string        `json:"type"`
	Data      []byte        `json:"data"`
	Delay     time.Duration `json:"delay"`
	Timestamp time.Time     `json:"timestamp"`
	Retries   int           `json:"retries"`
}

// Storage 存储接口
type Storage interface {
	// Save 保存任务
	Save(task *TaskInfo) error
	// Remove 删除任务
	Remove(taskID string) error
	// GetAll 获取所有未完成的任务
	GetAll() ([]*TaskInfo, error)
	// Close 关闭存储
	Close() error
}
