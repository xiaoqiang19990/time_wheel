package handlers

import (
	"timewheel/delayqueue/task"
)

// HandlerRegistry 处理器注册表
type handlerRegistry struct {
	handlers map[string]struct {
		reqType task.Request // 请求类型示例
		handler task.Handler // 处理器实例
	}
}

var defaultRegistry = &handlerRegistry{
	handlers: make(map[string]struct {
		reqType task.Request
		handler task.Handler
	}),
}

// register 注册处理器
func register(reqType task.Request, handler task.Handler) {
	defaultRegistry.handlers[reqType.GetTaskType()] = struct {
		reqType task.Request
		handler task.Handler
	}{
		reqType: reqType,
		handler: handler,
	}
}

// GetAllHandlers 获取所有已注册的处理器
func GetAllHandlers() map[string]struct {
	ReqType task.Request
	Handler task.Handler
} {
	result := make(map[string]struct {
		ReqType task.Request
		Handler task.Handler
	})
	for k, v := range defaultRegistry.handlers {
		result[k] = struct {
			ReqType task.Request
			Handler task.Handler
		}{
			ReqType: v.reqType,
			Handler: v.handler,
		}
	}
	return result
}

func init() {
	// 注册计算任务处理器
	register(&CalculationRequest{}, NewCalculatorHandler())
}
