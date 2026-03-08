// Package monitor provides functions and types for monitor
package monitor

import (
	"math"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

// 定义两个全局变量存数据
var (
	currentCPU float64
	currentMem float64
)

// Start 启动监控协程 (只需要在程序启动时调用一次)
func Start() {
	// 开启一个 Goroutine (后台线程)
	go func() {
		for {
			updateStats()
			// 每 2 秒更新一次，避免频繁占用资源
			time.Sleep(2 * time.Second)
		}
	}()
}

// GetStats 提供给外部读取数据的方法
func GetStats() (float64, float64) {
	return currentCPU, currentMem
}

// updateStats 内部逻辑：真正去干活获取数据的函数
func updateStats() {
	// 1. 获取内存
	v, err := mem.VirtualMemory()
	if err == nil {
		currentMem = v.UsedPercent
	}

	// 2. 获取 CPU
	c, err := cpu.Percent(0, false)
	if err == nil && len(c) > 0 {
		currentCPU = c[0]
	}

	// 这里做个简单的小优化：保留 1 位小数即可，看着干净
	currentCPU = math.Round(currentCPU*10) / 10
	currentMem = math.Round(currentMem*10) / 10
}
