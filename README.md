# Delay Queue

一个基于时间轮算法的延时队列实现。

## 特性

- 基于时间轮算法，高效处理大量定时任务
- 支持任务延时执行
- 支持任务重试机制
- 支持自定义任务处理器
- 支持优雅关闭
- 支持并发任务处理
- 支持任务执行指标统计

## 快速开始

### 1. 配置

在 `config/config.yaml` 中配置延时队列参数：

```yaml
delay_queue:
  slot_num: 60        # 时间轮槽数
  interval: 100ms     # 时间轮转动间隔
  max_retries: 3      # 最大重试次数
  retry_interval: 1s  # 重试间隔
  worker_num: 20      # 工作线程数
```

### 2. 定义任务

```go
// 定义任务请求
type CalculationRequest struct {
    TaskID string `json:"task_id"` // 自动生成的唯一ID
    A      int    `json:"a"`
    B      int    `json:"b"`
}

// 创建任务请求
func NewCalculationRequest(a, b int) *CalculationRequest {
    return &CalculationRequest{
        TaskID: utils.GenerateTaskIDWithPrefix("calc"),
        A:      a,
        B:      b,
    }
}

// 实现任务处理器
type CalculatorHandler struct{}

func (h *CalculatorHandler) Handle(req task.Request) error {
    // 处理任务逻辑
    return nil
}
```

### 3. 注册任务处理器

```go
func init() {
    // 注册任务处理器
    register(&CalculationRequest{}, NewCalculatorHandler())
}
```

### 4. 使用示例

```go
func main() {
    // 初始化服务
    if err := service.Init("config/config.yaml"); err != nil {
        panic(err)
    }
    defer service.Stop()

    // 创建任务
    calcReq := handlers.NewCalculationRequest(5, 5)

    // 添加延时任务
    err := service.CommonRegister.AddTask(calcReq, 2*time.Second)
    if err != nil {
        fmt.Printf("Failed to add task: %v\n", err)
        return
    }
}
```

## 架构设计

### 核心组件

- **DelayQueue**: 延时队列核心实现，基于时间轮算法
- **CommonRegister**: 任务注册器，管理任务处理器
- **HandlerRegistry**: 处理器注册表，存储任务类型与处理器的映射
- **Task**: 任务定义，包含任务ID、请求参数、处理器等信息

### 接口定义

```go
// Request 任务请求接口
type Request interface {
    GetTaskID() string
    GetTaskType() string
    Execute(handler interface{}) error
}

// Handler 任务处理器接口
type Handler interface {
    Handle(req Request) error
}
```

## 注意事项

1. 任务ID必须唯一
2. 任务处理器需要实现 Handler 接口
3. 任务请求需要实现 Request 接口
4. 建议根据实际需求调整配置参数

## License

MIT License 