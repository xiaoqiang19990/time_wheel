package task

import (
	"reflect"
	"strings"
	"time"
)

// Request 定义任务请求参数的接口
type Request interface {
	// GetTaskID 获取任务ID
	GetTaskID() string
	// GetTaskType 获取任务类型
	GetTaskType() string
	// Execute 执行处理函数
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

// Handler 定义任务处理器的接口
type Handler interface {
	// Handle 处理任务
	Handle(req Request) error
}

// Task 定义延时任务
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
