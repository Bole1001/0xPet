package entity

type Pet struct {
	OriginalContent string // 存档：永远保存最干净的那份 ASCII
	Content         string // 显示用：可能会被故障特效修改成乱码
	Width           int    // 窗口宽度 (像素)
	Height          int    // 窗口高度 (像素)
}
