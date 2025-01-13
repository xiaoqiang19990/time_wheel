package register

import (
	"fmt"
	"sync"
	"time"
	"timewheel/delayqueue/config"
	"timewheel/delayqueue/queue"
	"timewheel/delayqueue/task"
)

// CommonRegister 通用任务注册器
type CommonRegister struct {
	delayQueue *queue.DelayQueue
	handlers   map[string]task.Handler
	mutex      sync.RWMutex
}

type Common struct {
	Req         task.Request
	HandlerFunc interface{}
}

// NewCommonRegister 创建通用注册器
func NewCommonRegister(path string) (*CommonRegister, error) {
	cfg, err := config.LoadConfig(path)
	if err != nil {
		// 使用默认配置
		cfg = &config.DefaultConfig
	}

	dq, err := queue.NewDelayQueue(&cfg.DelayQueue)
	if err != nil {
		return nil, err
	}

	return &CommonRegister{
		delayQueue: dq,
		handlers:   make(map[string]task.Handler),
	}, nil
}

// RegisterHandler 注册任务处理器
func (r *CommonRegister) RegisterHandler(param []Common) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	for _, v := range param {
		// 创建处理器适配器
		adapter, err := newHandlerAdapter(v.Req, v.HandlerFunc)
		if err != nil {
			return err
		}

		r.handlers[v.Req.GetTaskType()] = adapter
	}
	return nil
}

// handlerAdapter 处理器适配器
type handlerAdapter struct {
	handlerFunc interface{}
	reqType     string
	execFunc    func(task.Request) error
}

// Handle 实现 Handler 接口
func (h *handlerAdapter) Handle(req task.Request) error {
	return h.execFunc(req)
}

// newHandlerAdapter 创建处理器适配器
func newHandlerAdapter(req task.Request, handlerFunc interface{}) (*handlerAdapter, error) {
	execFunc := func(r task.Request) error {
		return r.Execute(handlerFunc)
	}

	return &handlerAdapter{
		handlerFunc: handlerFunc,
		reqType:     fmt.Sprintf("%T", req),
		execFunc:    execFunc,
	}, nil
}

// AddTask 添加延时任务
func (r *CommonRegister) AddTask(req task.Request, delay time.Duration) error {
	r.mutex.RLock()
	h, exists := r.handlers[req.GetTaskType()]
	r.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("no handler registered for task type: %s", req.GetTaskType())
	}

	return r.delayQueue.AddTask(req, h, delay)
}

// Stop 停止注册器
func (r *CommonRegister) Stop() {
	r.delayQueue.Stop()
}

// HandlerBuilder 处理器构建器
type HandlerBuilder struct {
	handlers []Common
}

// NewHandlerBuilder 创建处理器构建器
func NewHandlerBuilder() *HandlerBuilder {
	return &HandlerBuilder{
		handlers: make([]Common, 0),
	}
}

// Add 添加处理器
func (b *HandlerBuilder) Add(req task.Request, handlerFunc interface{}) *HandlerBuilder {
	b.handlers = append(b.handlers, Common{
		Req:         req,
		HandlerFunc: handlerFunc,
	})
	return b
}

// Build 构建处理器列表
func (b *HandlerBuilder) Build() []Common {
	return b.handlers
}
