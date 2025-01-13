package task

import (
	"reflect"
	"strings"
	"time"
)

// Request 任务请求接口
type Request interface {
	GetTaskID() string
	GetTaskType() string
	Execute(handler interface{}) error
}

// GetTypeFromStruct 从结构体名称解析任务类型
func GetTypeFromStruct(req interface{}) string {
	t := reflect.TypeOf(req)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// 去掉 "Request" 后缀，转换为小写
	return strings.ToLower(strings.TrimSuffix(t.Name(), "Request"))
}

// Handler 任务处理器接口
type Handler interface {
	Handle(req Request) error
}

// Task 任务结构体
type Task struct {
	ID        string
	Request   Request
	Handler   Handler
	Delay     time.Duration
	Timestamp time.Time
	Retries   int
}

// NewTask 创建新任务
func NewTask(id string, req Request, handler Handler, delay time.Duration) *Task {
	now := time.Now()
	executeTime := now.Add(delay)

	return &Task{
		ID:        id,
		Request:   req,
		Handler:   handler,
		Delay:     delay,
		Timestamp: executeTime,
		Retries:   0,
	}
}
