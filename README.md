# Delay Queue

一个基于时间轮算法的延时队列实现，支持 Redis 持久化和故障恢复。

## 特性

- 基于时间轮算法，高效处理大量定时任务
- Redis 持久化存储，支持任务恢复
- 支持任务延时执行和重试机制
- 支持自定义任务处理器
- 支持优雅关闭和并发处理
- 支持任务执行指标统计

## 快速开始

### 1. 配置

在 `config/config.yaml` 中配置延时队列参数：

```yaml
delay_queue:
  slot_num: 60          # 时间轮槽数
  interval: 100ms       # 时间轮转动间隔
  max_retries: 3        # 最大重试次数
  retry_interval: 1s    # 重试间隔
  worker_num: 20        # 工作线程数
  redis_addr: "127.0.0.1:6379"  # Redis 地址
  redis_password: ""    # Redis 密码
  redis_db: 1          # Redis 数据库
```

### 2. 定义任务

```go
// 定义任务请求
type CalculationRequest struct {
    TaskID string `json:"task_id"` // 自动生成的唯一ID
    A      int    `json:"a"`
    B      int    `json:"b"`
}

// 实现 Request 接口
func (r *CalculationRequest) GetTaskID() string {
    return r.TaskID
}

func (r *CalculationRequest) GetTaskType() string {
    return "calculation"
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
    calcReq := req.(*CalculationRequest)
    result := calcReq.A + calcReq.B
    fmt.Printf("Calculation result: %d + %d = %d\n", 
        calcReq.A, calcReq.B, result)
    return nil
}
```

### 3. 使用示例

```go
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
        fmt.Printf("Failed to add task: %v\n", err)
        return
    }

    // 模拟服务重启，测试任务恢复
    time.Sleep(3 * time.Second)
    fmt.Println("Restarting service...")
    service.Stop()

    // 重新初始化服务，将恢复未完成的任务
    if err := service.Init("config/config.yaml"); err != nil {
        panic(err)
    }
}
```

## 架构设计

### 整体架构

```
┌─────────────────────────────────────────────────────────┐
│                        Service Layer                     │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐ │
│  │   Service   │    │   Register  │    │   Config    │ │
│  └──────┬──────┘    └──────┬──────┘    └──────┬──────┘ │
└─────────┼─────────────────┼─────────────────┼──────────┘
          │                 │                 │
┌─────────┼─────────────────┼─────────────────┼──────────┐
│         │                 │                 │          │
│  ┌──────▼──────┐   ┌─────▼─────┐    ┌─────▼─────┐    │
│  │  DelayQueue │   │  Handler  │    │  Storage  │    │
│  └──────┬──────┘   └─────┬─────┘    └─────┬─────┘    │
│         │                │                 │          │
│  ┌──────▼──────┐   ┌─────▼─────┐    ┌─────▼─────┐    │
│  │  TimeWheel  │   │   Task    │    │   Redis   │    │
│  └──────┬──────┘   └───────────┘    └───────────┘    │
│         │                                             │
│  ┌──────▼──────┐   ┌───────────┐                     │
│  │  Executor   │   │  Metrics  │                     │
│  └─────────────┘   └───────────┘                     │
│                                                      │
│                    Core Layer                        │
└──────────────────────────────────────────────────────┘
```

### 分层说明

1. **Service Layer（服务层）**
   - Service: 提供对外接口，管理服务生命周期
   - Register: 管理任务注册和处理器映射
   - Config: 配置管理和加载

2. **Core Layer（核心层）**
   - DelayQueue: 延时队列核心实现
   - TimeWheel: 时间轮调度器
   - Handler: 任务处理器
   - Storage: 存储接口及实现
   - Metrics: 性能指标收集

### 核心组件

1. **DelayQueue**
```go
type DelayQueue struct {
    slots      []*Slot          // 时间轮槽位
    ticker     *time.Ticker     // 定时器
    options    *Config          // 配置选项
    current    int              // 当前槽位
    tasks      map[string]int   // 任务索引
    workerPool chan struct{}    // 工作线程池
    metrics    *Metrics         // 性能指标
    storage    Storage          // 存储接口
}
```

2. **Task**
```go
type Task struct {
    ID        string           // 任务ID
    Request   Request          // 任务请求
    Handler   Handler          // 任务处理器
    Delay     time.Duration    // 延迟时间
    Timestamp time.Time        // 执行时间
    Retries   int             // 重试次数
}
```

3. **Storage**
```go
type Storage interface {
    Save(task *TaskInfo) error
    Remove(taskID string) error
    GetAll() ([]*TaskInfo, error)
    Close() error
}
```

### 工作流程

1. **任务提交流程**
```
Client Request
     │
     ▼
Service Layer ──► Register ──► DelayQueue
     │                            │
     │                            ▼
     │                        Calculate Slot
     │                            │
     └────► Storage ◄────────────┘
```

2. **任务执行流程**
```
TimeWheel Tick
     │
     ▼
Check Slot ──► Get Tasks ──► Worker Pool
                                │
                                ▼
                          Execute Task ──► Update Metrics
                                │
                                ▼
                          Remove from Storage
```

3. **任务恢复流程**
```
Service Start
     │
     ▼
Load from Storage ──► For Each Task
     │                    │
     │                    ▼
     │              Check Timestamp
     │                    │
     ▼                   ▼
Add to TimeWheel   Execute Immediately
```

### 关键特性实现

1. **并发控制**
   - 工作线程池限制并发执行数量
   - 槽位级别的互斥锁保护
   - 原子操作保证计数准确性

2. **持久化存储**
   - Redis Hash 结构存储任务
   - 任务执行完成自动清理
   - 支持批量操作和恢复

3. **故障恢复**
   - 服务启动时自动加载任务
   - 重新计算执行时间
   - 过期任务立即执行

4. **性能优化**
   - 时间轮算法降低扫描开销
   - 批量操作减少存储开销
   - 并发执行提高吞吐量

### 监控指标

```go
type Metrics struct {
    PendingTasks    int64    // 待处理任务数
    ExecutedTasks   int64    // 已执行任务数
    RetryTasks      int64    // 重试任务数
    FailedTasks     int64    // 失败任务数
    ExecutionTime   int64    // 执行时间统计
    StorageLatency  int64    // 存储延迟统计
}
```

## 存储设计

```go
// Storage 存储接口
type Storage interface {
    Save(task *TaskInfo) error
    Remove(taskID string) error
    GetAll() ([]*TaskInfo, error)
    Close() error
}

// TaskInfo 任务存储信息
type TaskInfo struct {
    ID        string        `json:"id"`
    Type      string        `json:"type"`
    Data      []byte        `json:"data"`
    Delay     time.Duration `json:"delay"`
    Timestamp time.Time     `json:"timestamp"`
    Retries   int          `json:"retries"`
}
```

## 注意事项

1. Redis 配置
   - 确保 Redis 服务可用
   - 合理配置连接参数
   - 监控 Redis 性能

2. 任务设计
   - 任务 ID 必须唯一
   - 实现幂等性
   - 处理好并发情况

3. 性能优化
   - 调整槽位数量和检查间隔
   - 配置工作线程数
   - 监控系统资源

## License

MIT License 