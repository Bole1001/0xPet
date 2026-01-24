package config

import (
	"encoding/json"
	"os"
)

// Config 结构体：对应 config.json 的内容
type Config struct {
	ImagePath     string `json:"image_path"`     // 上次用的图片路径
	ShowColor     bool   `json:"show_color"`     // 是否开启彩色
	ShowGlitch    bool   `json:"show_glitch"`    // 是否开启乱码
	ShowAnimation bool   `json:"show_animation"` // 是否开启浮动
	ShowMonitor   bool   `json:"show_monitor"`   // 是否开启监控文字
}

// NewDefault 生成一份默认配置
// 当找不到配置文件，或者读取失败时，用这个“保底”
func NewDefault() *Config {
	return &Config{
		ImagePath:     "assets/idle.png", // 默认图
		ShowColor:     true,
		ShowGlitch:    true,
		ShowAnimation: true,
		ShowMonitor:   false,
	}
}

// Load 从硬盘读取配置
func Load(filename string) (*Config, error) {
	// 1. 尝试打开文件
	file, err := os.Open(filename)
	if err != nil {
		// 如果文件不存在，直接返回默认配置，不算报错
		if os.IsNotExist(err) {
			return NewDefault(), nil
		}
		return nil, err
	}
	defer file.Close()

	// 2. 解析 JSON
	cfg := &Config{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(cfg); err != nil {
		// 如果 JSON 格式坏了，也返回默认配置
		return NewDefault(), nil
	}

	return cfg, nil
}

// Save 把当前配置写入硬盘
func Save(cfg *Config, filename string) error {
	// 1. 创建/覆盖文件
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// 2. 写入 JSON (SetIndent 让生成的 JSON 带缩进，方便人类阅读)
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(cfg)
}
