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

	err := service.CommonRegister.AddTask(calcReq, 2*time.Second)
	if err != nil {
		fmt.Printf("Failed to add task %s: %v\n", calcReq.TaskID, err)
		return
	}
	fmt.Printf("Task %s scheduled to run in %v\n", calcReq.TaskID, 2*time.Second)
	time.Sleep(1 * time.Second)

	// 创建计算任务
	calcReq1 := handlers.NewCalculationRequest(5, 6)

	err1 := service.CommonRegister.AddTask(calcReq1, 20*time.Second)
	if err1 != nil {
		fmt.Printf("Failed to add task %s: %v\n", calcReq1.TaskID, err1)
		return
	}
	fmt.Printf("Task %s scheduled to run in %v\n", calcReq1.TaskID, 20*time.Second)

	// 等待所有任务执行完成
	time.Sleep(25 * time.Second)
	fmt.Println("\nAll tasks completed!")
}
