package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStorage struct {
	client *redis.Client
}

// TaskInfo Redis存储的任务信息
type RedisTaskInfo struct {
	ID        string        `json:"id"`
	Type      string        `json:"type"` // 任务类型，用于恢复时找到对应的处理器
	Data      []byte        `json:"data"` // 序列化后的请求数据
	Delay     time.Duration `json:"delay"`
	Timestamp time.Time     `json:"timestamp"`
	Retries   int           `json:"retries"`
}

func NewRedisStorage(addr string, password string, db int) *RedisStorage {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	return &RedisStorage{client: client}
}

// Save 保存任务到Redis
func (r *RedisStorage) Save(task *TaskInfo) error {
	ctx := context.Background()
	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("marshal task error: %v", err)
	}

	// 使用任务ID作为field，将任务信息存储在固定的Hash key中
	result := r.client.HSet(ctx, "delay_queue:tasks", task.ID, data)
	if err := result.Err(); err != nil {
		return fmt.Errorf("save task error: %v", err)
	}

	fmt.Printf("Task saved to Redis - ID: %s, Type: %s\n", task.ID, task.Type)
	return nil
}

// Remove 从Redis删除任务
func (r *RedisStorage) Remove(taskID string) error {
	ctx := context.Background()
	if err := r.client.HDel(ctx, "delay_queue:tasks", taskID).Err(); err != nil {
		return fmt.Errorf("remove task error: %v", err)
	}
	return nil
}

// GetAll 获取所有未完成的任务
func (r *RedisStorage) GetAll() ([]*TaskInfo, error) {
	ctx := context.Background()
	// 获取所有任务
	result, err := r.client.HGetAll(ctx, "delay_queue:tasks").Result()
	if err != nil {
		return nil, fmt.Errorf("get all tasks error: %v", err)
	}

	tasks := make([]*TaskInfo, 0, len(result))
	for _, data := range result {
		var task TaskInfo
		if err := json.Unmarshal([]byte(data), &task); err != nil {
			fmt.Printf("unmarshal task error: %v\n", err)
			continue
		}
		tasks = append(tasks, &task)
	}

	return tasks, nil
}

// Close 关闭Redis连接
func (r *RedisStorage) Close() error {
	return r.client.Close()
}
