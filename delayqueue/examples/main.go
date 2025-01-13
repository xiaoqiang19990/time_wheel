package main

import (
	"fmt"
	"time"
	"timewheel/delayqueue/handlers"
	"timewheel/delayqueue/service"
)

func main() {
	// 初始化服务
	if err := service.Init("config/config.yaml"); err != nil {
		panic(err)
	}
	defer service.Stop()

	// 创建计算任务
	calcReq := handlers.NewCalculationRequest(5, 5)
	err := service.CommonRegister.AddTask(calcReq, 10*time.Second)
	if err != nil {
		fmt.Printf("Failed to add task %s: %v\n", calcReq.TaskID, err)
		return
	}
	fmt.Printf("Task %s scheduled to run in %v\n", calcReq.TaskID, 10*time.Second)

	// 等待3秒后重启服务，模拟服务重启场景
	time.Sleep(3 * time.Second)
	fmt.Println("\nRestarting service...")
	service.Stop()

	// 重新初始化服务，应该恢复未完成的任务
	if err := service.Init("config/config.yaml"); err != nil {
		panic(err)
	}
	fmt.Println("Service restarted, tasks should be recovered")

	// 等待任务执行完成
	time.Sleep(10 * time.Second)
	fmt.Println("\nAll tasks completed!")
}
