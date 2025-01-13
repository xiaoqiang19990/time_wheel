package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

// DelayQueueConfig 延时队列配置
type DelayQueueConfig struct {
	SlotNum       int           `yaml:"slot_num"`
	Interval      time.Duration `yaml:"interval"`
	MaxRetries    int           `yaml:"max_retries"`
	RetryInterval time.Duration `yaml:"retry_interval"`
	WorkerNum     int           `yaml:"worker_num"`
	RedisAddr     string        `yaml:"redis_addr"`
	RedisPassword string        `yaml:"redis_password"`
	RedisDB       int           `yaml:"redis_db"`
}

// Config 配置
type Config struct {
	DelayQueue DelayQueueConfig `yaml:"delay_queue"`
}

// DefaultConfig 默认配置
var DefaultConfig = Config{
	DelayQueue: DelayQueueConfig{
		SlotNum:       60,
		Interval:      100 * time.Millisecond,
		MaxRetries:    3,
		RetryInterval: time.Second,
		WorkerNum:     10,
	},
}

// LoadConfig 加载配置
func LoadConfig(filename string) (*Config, error) {
	if filename == "" {
		return &DefaultConfig, nil
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return &DefaultConfig, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return &DefaultConfig, err
	}

	return &cfg, nil
}
