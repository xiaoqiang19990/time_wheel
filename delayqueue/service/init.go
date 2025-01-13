package service

import (
	"timewheel/delayqueue/handlers"
	"timewheel/delayqueue/register"
)

var (
	CommonRegister *register.CommonRegister
)

// Init 初始化服务
func Init(path string) error {
	var err error
	CommonRegister, err = register.NewCommonRegister(path)
	if err != nil {
		return err
	}

	// 获取所有已注册的处理器
	allHandlers := handlers.GetAllHandlers()

	// 构建处理器列表
	builder := register.NewHandlerBuilder()
	for _, h := range allHandlers {
		builder.Add(h.ReqType, h.Handler)
	}

	// 注册处理器
	if err := CommonRegister.RegisterHandler(builder.Build()); err != nil {
		return err
	}

	return nil
}

// Stop 停止服务
func Stop() {
	if CommonRegister != nil {
		CommonRegister.Stop()
	}
}
