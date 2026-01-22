package entity

import "image/color"

// 【新增】单个字符的数据单元
type CharData struct {
	OriginalChar string      // 【新增】存档：原本这个位置是什么字（比如 "@"）
	Char         string      // 字符，比如 "@"
	Color        color.Color // 颜色，比如 RGB(255, 0, 0)
}

type Pet struct {
	OriginalContent string // 存档：永远保存最干净的那份 ASCII
	Content         string // 显示用：可能会被故障特效修改成乱码
	Width           int    // 窗口宽度 (像素)
	Height          int    // 窗口高度 (像素)

	// 【新增】彩色模式的数据网格
	// 这是一个二维数组：[行][列] -> 字符数据
	Grid [][]CharData

	CPUUsage   float64 // CPU 使用率 (0-100)
	MemUsage   float64 // 内存 使用率 (0-100)
	IsStressed bool    // 是否处于高压状态 (CPU > 80)
}
