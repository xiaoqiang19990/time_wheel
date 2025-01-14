package handlers

import (
	"encoding/json"
	"fmt"
	"timewheel/delayqueue/task"
	"timewheel/delayqueue/utils"
)

// CalculationRequest 计算请求参数
type CalculationRequest struct {
	TaskID string `json:"task_id"`
	A      int    `json:"a"`
	B      int    `json:"b"`
}

// NewCalculationRequest 创建计算请求
func NewCalculationRequest(a, b int) *CalculationRequest {
	return &CalculationRequest{
		TaskID: utils.GenerateTaskIDWithPrefix(task.GetTypeFromStruct(CalculationRequest{})),
		A:      a,
		B:      b,
	}
}

// GetTaskID 实现 Request 接口
func (r *CalculationRequest) GetTaskID() string {
	return r.TaskID
}

// GetTaskType 获取任务类型
func (r *CalculationRequest) GetTaskType() string {
	// 返回结构体名称，去掉 "Request" 后缀
	return task.GetTypeFromStruct(r)
}

// Execute 实现 Request 接口
func (r *CalculationRequest) Execute(handler interface{}) error {
	switch h := handler.(type) {
	case func(*CalculationRequest) error:
		return h(r)
	case task.Handler:
		return h.Handle(r)
	default:
		return fmt.Errorf("invalid handler type for CalculationRequest: %T", handler)
	}
}

// CalculatorHandler 计算处理器
type CalculatorHandler struct{}

// NewCalculatorHandler 创建计算任务处理器
func NewCalculatorHandler() *CalculatorHandler {
	return &CalculatorHandler{}
}

// Handle 实现 Handler 接口
func (h *CalculatorHandler) Handle(req task.Request) error {
	calcReq, ok := req.(*CalculationRequest)
	if !ok {
		fmt.Println("invalid request type for calculator handler")
		return fmt.Errorf("invalid request type for calculator handler")
	}

	result := calcReq.A + calcReq.B
	fmt.Printf("Calculation result for task %s: %d + %d = %d\n",
		calcReq.TaskID, calcReq.A, calcReq.B, result)
	return nil
}

func (h *CalculationRequest) UnmarshalRequest(data []byte) (task.Request, error) {
	var req CalculationRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

// 修改 CalculationRequest
func (r *CalculationRequest) Marshal() []byte {
	data, _ := json.Marshal(r)
	return data
}
